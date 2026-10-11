package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"local-finance/internal/service"
)

func TestStatementRecordsPreserveOverlappingImportMembership(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	svc := service.NewTransactionService(database)
	header := "bank_name,account_type,account_number_mask,date,narration,amount,tx_type,reference_number\n"
	rows := []string{
		"Demo Bank,SAVINGS,XXXX4321,2026-09-03,Corner Shop,125.50,DEBIT,one\n",
		"Demo Bank,SAVINGS,XXXX4321,2026-09-02,Book Shop,200.00,DEBIT,two\n",
		"Demo Bank,SAVINGS,XXXX4321,2026-09-01,Salary,1000.00,CREDIT,three\n",
	}
	first, err := svc.ImportStatement("first.csv", strings.NewReader(header+strings.Join(rows, "")), "", parser.LocalFinanceCSVParserID, "")
	if err != nil {
		t.Fatal(err)
	}
	initial, _, err := database.ListTransactions(db.TransactionFilter{StatementImportID: first.StatementImportID})
	if err != nil || len(initial) != 3 {
		t.Fatalf("first records: %v %v", initial, err)
	}
	if _, err := database.Exec(`UPDATE transactions SET notes='Keep note', tags='manual', category_id='cat_food', is_manual_category=1 WHERE id=?`, initial[0].ID); err != nil {
		t.Fatal(err)
	}
	second, err := svc.ImportStatement("overlap.csv", strings.NewReader(header+strings.Join(rows[:2], "")), first.AccountID, parser.LocalFinanceCSVParserID, "")
	if err != nil {
		t.Fatal(err)
	}
	if second.InsertedCount != 0 || second.DuplicateCount != 2 {
		t.Fatalf("unexpected re-import: %+v", second)
	}
	if _, err := database.UpsertTransaction(&models.Transaction{AccountID: first.AccountID, TxHash: "unrelated", TxDate: "2026-09-04", RawNarration: "Unrelated", TxType: models.TxTypeDebit, Amount: 50}); err != nil {
		t.Fatal(err)
	}

	check := func(query string, total, pages int, amounts ...float64) {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, localTestRequest(http.MethodGet, "/api/transactions?"+query, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
		var result struct {
			Items      []models.Transaction `json:"items"`
			Total      int                  `json:"total"`
			TotalPages int                  `json:"total_pages"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.Total != total || result.TotalPages != pages || len(result.Items) != len(amounts) {
			t.Fatalf("unexpected records for %s: %+v", query, result)
		}
		for i, amount := range amounts {
			if result.Items[i].Amount != amount {
				t.Fatalf("amount %v, want %v", result.Items[i].Amount, amount)
			}
		}
	}
	firstScope := "statement_import_id=" + first.StatementImportID
	secondScope := "statement_import_id=" + second.StatementImportID
	check(firstScope+"&page_size=2", 3, 2, 125.5, 200)
	check(firstScope+"&page_size=2&page=2", 3, 2, 1000)
	check(firstScope+"&category_id=cat_food&search=Corner&max_amount=130", 1, 1, 125.5)
	check(secondScope, 2, 1, 125.5, 200)
	check("statement_import_id=missing", 0, 0)
	saved, err := database.GetTransaction(initial[0].ID)
	if err != nil || saved.Notes != "Keep note" || saved.Tags != "manual" || !saved.IsManualCategory {
		t.Fatalf("edits lost: %+v %v", saved, err)
	}

	// Existing deletion ownership remains the latest import, while membership is many-to-many.
	deleted, err := database.DeleteStatementImport(first.StatementImportID)
	if err != nil || deleted.DeletedTransactions != 1 {
		t.Fatalf("older deletion: %+v %v", deleted, err)
	}
	check(secondScope, 2, 1, 125.5, 200)
	check(firstScope, 0, 0)
	if err := database.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	check(secondScope, 0, 0)
}

func TestStatementRecordsLatestImportDeletion(t *testing.T) {
	_, _, database, cleanup := setupTestRouter(t)
	defer cleanup()
	svc := service.NewTransactionService(database)
	header := "bank_name,account_type,account_number_mask,date,narration,amount,tx_type,reference_number\n"
	shared := "Demo Bank,SAVINGS,XXXX4321,2026-09-03,Shared expense,125.50,DEBIT,shared\n"
	olderOnly := "Demo Bank,SAVINGS,XXXX4321,2026-09-02,Older expense,200.00,DEBIT,older\n"
	newerOnly := "Demo Bank,SAVINGS,XXXX4321,2026-09-04,Newer expense,300.00,DEBIT,newer\n"
	first, err := svc.ImportStatement("first.csv", strings.NewReader(header+shared+olderOnly), "", parser.LocalFinanceCSVParserID, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.ImportStatement("second.csv", strings.NewReader(header+shared+newerOnly), first.AccountID, parser.LocalFinanceCSVParserID, "")
	if err != nil {
		t.Fatal(err)
	}
	// The existing deletion contract removes rows owned by the latest upload,
	// including shared rows, and must clear their links from every older import.
	deleted, err := database.DeleteStatementImport(second.StatementImportID)
	if err != nil || deleted.DeletedTransactions != 2 {
		t.Fatalf("latest deletion: %+v %v", deleted, err)
	}
	remaining, total, err := database.ListTransactions(db.TransactionFilter{StatementImportID: first.StatementImportID})
	if err != nil || total != 1 || len(remaining) != 1 || remaining[0].Amount != 200 {
		t.Fatalf("older records after latest deletion: %+v total=%d err=%v", remaining, total, err)
	}
	links, err := database.Query(`SELECT COUNT(*) FROM statement_import_transactions sit
 LEFT JOIN statement_imports s ON s.id=sit.statement_import_id
 LEFT JOIN transactions t ON t.id=sit.transaction_id
 WHERE s.id IS NULL OR t.id IS NULL`)
	if err != nil {
		t.Fatal(err)
	}
	defer links.Close()
	var orphaned int
	if !links.Next() {
		t.Fatal("missing orphan count")
	}
	if err := links.Scan(&orphaned); err != nil || orphaned != 0 {
		t.Fatalf("orphaned record links: %d %v", orphaned, err)
	}
}
