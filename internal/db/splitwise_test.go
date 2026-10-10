package db

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwisePersonalExpensesAndCashLedger(t *testing.T) {
	d := safetyDB(t)
	a, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "example", "example", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, date string, amount float64, kind models.TxType) string {
		t.Helper()
		cat := "cat_food"
		tx := &models.Transaction{AccountID: a.ID, TxHash: hash, TxDate: date, RawNarration: hash, CleanedPayee: hash, Amount: amount, TxType: kind, CategoryID: &cat, Notes: "Keep note", Tags: "keep", IsManualCategory: true}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	dinner := add("dinner", "2026-09-01", 300, models.TxTypeDebit)
	debit := add("settlement out", "2026-09-03", 50, models.TxTypeDebit)
	credit := add("settlement in", "2026-09-04", 80, models.TxTypeCredit)
	add("salary", "2026-09-01", 1000, models.TxTypeCredit)
	if err := d.RecalculateAccountBalance(a.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := d.ListAccounts()
	data, err := os.ReadFile("../../samples/splitwise/fictional-group.csv")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewReader(data), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if n, err := d.ImportSplitwise(entries); err != nil || n != 5 {
		t.Fatalf("import: %d %v", n, err)
	}
	assertTotals := func(expense, income float64) {
		t.Helper()
		result, err := d.GetAnalyticsOverview()
		if err != nil || result.TotalExpense != expense || result.TotalIncome != income {
			t.Fatalf("totals: %+v %v; want expense %.2f income %.2f", result, err, expense, income)
		}
	}
	assertTotals(350, 1080) // Pending rows never change expenses.
	candidates, err := d.SplitwiseCandidates(entries[0].ID, 10000)
	if err != nil || len(candidates) != 1 || candidates[0].ID != dinner {
		t.Fatalf("candidates: %+v %v", candidates, err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Share: 10000, TransactionID: dinner}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[1].ID, models.SplitwiseConfirmation{Share: 5000}); err != nil {
		t.Fatal(err)
	}
	for i, id := range map[int]string{2: debit, 3: credit} {
		if err := d.ConfirmSplitwise(entries[i].ID, models.SplitwiseConfirmation{TransactionID: id}); err != nil {
			t.Fatal(err)
		}
	}
	assertTotals(150, 1000)
	if n, err := d.ImportSplitwise(entries); err != nil || n != 0 {
		t.Fatalf("duplicate import: %d %v", n, err)
	}
	assertTotals(150, 1000)
	after, _ := d.ListAccounts()
	if before[0].CurrentBalance != after[0].CurrentBalance {
		t.Fatal("Splitwise changed bank balance")
	}
	statement, err := d.GetTransaction(dinner)
	if err != nil || statement.Amount != 300 || statement.TxHash != "dinner" || statement.Notes != "Keep note" || statement.Tags != "keep" || !statement.IsManualCategory {
		t.Fatalf("statement changed: %+v %v", statement, err)
	}
	if _, err := d.UpsertTransaction(statement); err != nil {
		t.Fatal(err)
	}
	assertTotals(150, 1000)
	review, err := d.GetMonthlyReview("2026-09", time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || review.Current.Amount != 150 {
		t.Fatalf("review: %+v %v", review, err)
	}
	evidence, err := d.GetMonthlyReviewEvidence("2026-09", "", false, 1, time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || evidence.Total != 1 || len(evidence.Items) != 1 || evidence.Items[0].Account != "Splitwise · " {
		t.Fatalf("evidence: %+v %v", evidence, err)
	}
	cash, err := d.GetCashFlowIntelligence("2026-09")
	if err != nil {
		t.Fatal(err)
	}
	if cash.Summary.TotalOutflow != 150 {
		t.Fatalf("cash flow omitted shared expenses: %+v", cash)
	}
	export, err := d.ExportAllDataJSON()
	if err != nil || len(export.Splitwise) != 5 {
		t.Fatalf("JSON export: %v", err)
	}
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := d.BackupTo(backup); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	list, _ := d.ListSplitwise()
	if len(list) != 0 {
		t.Fatal("reset retained Splitwise")
	}
	if err := d.RestoreFrom(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	assertTotals(150, 1000)
	if err := d.ResetSplitwise(entries[0].ID); err != nil {
		t.Fatal(err)
	}
	assertTotals(350, 1000)
	if err := d.ConfirmSplitwise(entries[1].ID, models.SplitwiseConfirmation{Ignore: true}); err != nil {
		t.Fatal(err)
	}
	assertTotals(300, 1000)
}

func TestSplitwiseConfirmationRejectsUnsafeLinks(t *testing.T) {
	d := safetyDB(t)
	a, err := d.GetOrCreateAccount("Example", models.AccountTypeSavings, "x", "x", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Dinner,Food,100.00,INR,50.00,-50.00\n2026-09-01,Another,Food,100.00,INR,50.00,-50.00\n")
	entries, err := parser.ParseSplitwise(bytes.NewReader(data), "Group", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, date string
		amount     float64
		kind       models.TxType
		excluded   bool
	}{
		{"wrong amount", "2026-09-01", 99, models.TxTypeDebit, false}, {"wrong date", "2026-10-01", 100, models.TxTypeDebit, false}, {"wrong direction", "2026-09-01", 100, models.TxTypeCredit, false}, {"excluded", "2026-09-01", 100, models.TxTypeDebit, true},
	} {
		tx := &models.Transaction{AccountID: a.ID, TxHash: tc.name, RawNarration: tc.name, TxDate: tc.date, Amount: tc.amount, TxType: tc.kind, IsExcluded: tc.excluded}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Share: 5000, TransactionID: tx.ID}); err == nil {
			t.Fatalf("accepted %s", tc.name)
		}
	}
	for _, req := range []models.SplitwiseConfirmation{{Share: 5000}, {Share: -1}, {Share: 10001}, {Share: 5000, TransactionID: "missing"}} {
		if err := d.ConfirmSplitwise(entries[0].ID, req); err == nil {
			t.Fatal("accepted invalid confirmation")
		}
	}
	tx := &models.Transaction{AccountID: a.ID, TxHash: "valid", RawNarration: "valid", TxDate: "2026-09-01", Amount: 100, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Share: 5000, TransactionID: tx.ID}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[1].ID, models.SplitwiseConfirmation{Share: 5000, TransactionID: tx.ID}); err == nil {
		t.Fatal("linked one bank movement twice")
	}
	// Multiple payers: net + share is paid, rather than always the full group cost.
	partial := &models.Transaction{AccountID: a.ID, TxHash: "partial", RawNarration: "partial", TxDate: "2026-09-01", Amount: 75, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(partial); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[1].ID, models.SplitwiseConfirmation{Share: 2500, TransactionID: partial.ID}); err != nil {
		t.Fatal(err)
	}
}
