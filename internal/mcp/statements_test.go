package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
	"local-finance/internal/models"
)

const fictionalStatementCSV = "bank_name,account_type,account_number_mask,date,narration,amount,tx_type,reference_number\nFictional Bank,SAVINGS,XXXX1234,2026-01-10,Corner Shop,125.50,DEBIT,ref-1\n"

func TestStatementWritesImportReimportDeleteAndPermissions(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	allowed, denied := true, false
	path := filepath.Join(t.TempDir(), "statement.csv")
	if err := os.WriteFile(path, []byte(fictionalStatementCSV), 0600); err != nil {
		t.Fatal(err)
	}
	// Independent access modes expose only their allowed tools.
	for _, permissions := range []struct {
		categories, statements bool
		count                  int
	}{{false, false, 23}, {true, false, 25}, {false, true, 25}, {true, true, 27}} {
		if _, err := m.ConfigureAccess(true, m.Status().Port, &permissions.categories, &permissions.statements); err != nil {
			t.Fatal(err)
		}
		session := connect(t, m, token)
		listed, err := session.ListTools(context.Background(), nil)
		if err != nil || len(listed.Tools) != permissions.count {
			t.Fatalf("discovery %+v: %+v %v", permissions, listed, err)
		}
		if !permissions.statements {
			if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_statement_csv", Arguments: map[string]any{"path": path}}); err == nil {
				t.Fatal("unpermitted import")
			}
		}
	}
	if _, err := m.ConfigureAccess(true, m.Status().Port, &denied, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	first := decodeWrite[models.ImportResult](t, callWrite(t, session, "import_statement_csv", map[string]any{"path": path}, false))
	if first.InsertedCount != 1 || first.TotalParsed != 1 || first.StatementImportID == "" {
		t.Fatalf("import: %+v", first)
	}
	rows, _, err := database.ListTransactions(db.TransactionFilter{AccountID: first.AccountID, Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows: %+v %v", rows, err)
	}
	txID := rows[0].ID
	if _, err := database.Exec(`UPDATE transactions SET category_id = 'cat_groceries', is_manual_category = 1, notes = 'keep note', tags = 'manual' WHERE id = ?`, txID); err != nil {
		t.Fatal(err)
	}
	second := decodeWrite[models.ImportResult](t, callWrite(t, session, "import_statement_csv", map[string]any{"path": path, "account_id": first.AccountID}, false))
	if second.InsertedCount != 0 || second.DuplicateCount != 1 {
		t.Fatalf("reimport: %+v", second)
	}
	saved, err := database.GetTransaction(txID)
	if err != nil || !saved.IsManualCategory || saved.CategoryID == nil || *saved.CategoryID != "cat_groceries" || saved.Tags != "manual" || saved.Notes != "keep note" {
		t.Fatalf("manual edits: %+v %v", saved, err)
	}
	older := decodeWrite[db.StatementDeletion](t, callWrite(t, session, "delete_statement_import", map[string]any{"statement_import_id": first.StatementImportID}, false))
	if older.DeletedTransactions != 0 {
		t.Fatalf("older deletion: %+v", older)
	}
	// Delete a linked bill and clear a surviving transfer counterpart.
	if _, err := database.Exec(`INSERT INTO transactions (id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,payment_mode,reference_number,tx_type,amount,is_transfer,transfer_peer_id) VALUES ('peer',?,'peer-hash','2026-01-11','Peer','Peer','OTHER','','CREDIT',10,1,?)`, first.AccountID, txID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO credit_card_bills (id,account_id,statement_import_id,statement_date,payment_due_date,total_due_amount) VALUES ('bill',?,?,'2026-01-10','2026-01-20',125.50)`, first.AccountID, second.StatementImportID); err != nil {
		t.Fatal(err)
	}
	deleted := decodeWrite[db.StatementDeletion](t, callWrite(t, session, "delete_statement_import", map[string]any{"statement_import_id": second.StatementImportID}, false))
	if deleted.DeletedTransactions != 1 || deleted.DeletedBills != 1 {
		t.Fatalf("deletion: %+v", deleted)
	}
	peer, err := database.GetTransaction("peer")
	if err != nil || peer.IsTransfer || peer.TransferPeerID != nil {
		t.Fatalf("peer link: %+v %v", peer, err)
	}
	accounts, err := database.ListAccounts()
	if err != nil || len(accounts) != 1 || accounts[0].CurrentBalance != 10 {
		t.Fatalf("balance: %+v %v", accounts, err)
	}
	callWrite(t, session, "delete_statement_import", map[string]any{"statement_import_id": second.StatementImportID}, true)
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &denied); err != nil {
		t.Fatal(err)
	}
	if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "delete_statement_import", Arguments: map[string]any{"statement_import_id": "missing"}}); err == nil {
		t.Fatal("old session retained statement permission")
	}
}

func TestStatementWriteFailuresLeaveLedgerUntouched(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	directory := t.TempDir()
	path := filepath.Join(directory, "bad.csv")
	invalid := fictionalStatementCSV + strings.Replace(fictionalStatementCSV[strings.Index(fictionalStatementCSV, "\n")+1:], "125.50", "NaN", 1)
	if err := os.WriteFile(path, []byte(invalid), 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{path, "relative.csv", filepath.Join(directory, "missing.csv"), directory, `\\server\share\statement.csv`} {
		callWrite(t, session, "import_statement_csv", map[string]any{"path": p}, true)
	}
	if err := os.WriteFile(path, []byte(fictionalStatementCSV), 0600); err != nil {
		t.Fatal(err)
	}
	callWrite(t, session, "import_statement_csv", map[string]any{"path": path, "account_id": "missing"}, true)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(21 << 20); err != nil {
		t.Fatal(err)
	}
	file.Close()
	callWrite(t, session, "import_statement_csv", map[string]any{"path": path}, true)
	imports, err := database.ListStatementImports()
	if err != nil || len(imports) != 0 {
		t.Fatalf("partial imports: %+v %v", imports, err)
	}
	accounts, err := database.ListAccounts()
	if err != nil || len(accounts) != 0 {
		t.Fatalf("partial accounts: %+v %v", accounts, err)
	}
}
