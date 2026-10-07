package db

import (
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"local-finance/internal/models"
)

func TestMigrationMarksExistingSelfTransfers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	database, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { database.Close() }()
	if err := goose.DownTo(database.conn, "migrations", 12); err != nil {
		t.Fatal(err)
	}
	account, err := database.GetOrCreateAccount("Example Bank", models.AccountTypeSavings, "123456789012", "XX9012", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	category, notes, tags := "cat_others", "Keep this note", "personal"
	originalIDs := make(map[string]string)
	for _, item := range []struct {
		id, narration string
		txType        models.TxType
	}{
		{"self-debit", "UPI-ALEX-alex@okhdfcbank-HDFC-123456789012-SELF TRANSFER", models.TxTypeDebit},
		{"self-credit", "NEFT CR-ABC123-ALEX-self transfer", models.TxTypeCredit},
		{"purchase", "UPI-SHOP-shop@okhdfcbank-HDFC-123456789014-PAYMENT", models.TxTypeDebit},
	} {
		transaction := &models.Transaction{
			AccountID: account.ID, TxHash: item.id,
			TxDate: "2026-04-01", RawNarration: item.narration,
			TxType: item.txType, Amount: 100, PaymentMode: models.PaymentModeOther,
			CategoryID: &category, IsManualCategory: true, Notes: notes, Tags: tags,
		}
		_, err := database.UpsertTransaction(transaction)
		if err != nil {
			t.Fatal(err)
		}
		originalIDs[item.id] = transaction.ID
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	transactions, count, err := database.ListTransactions(TransactionFilter{Limit: 10})
	if err != nil || count != 3 {
		t.Fatalf("transaction count=%d, err=%v", count, err)
	}
	for _, tx := range transactions {
		if tx.IsTransfer != (tx.TxHash != "purchase") {
			t.Errorf("incorrect transfer flag for %s: %v", tx.ID, tx.IsTransfer)
		}
		if originalIDs[tx.TxHash] != tx.ID || !tx.IsManualCategory || tx.CategoryID == nil || *tx.CategoryID != category || tx.Notes != notes || tx.Tags != tags {
			t.Errorf("migration changed transaction identity or user edits: %+v", tx)
		}
	}
	overview, err := database.GetAnalyticsOverview()
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalIncome != 0 || overview.TotalExpense != 100 {
		t.Fatalf("existing self transfers still affect totals: %+v", overview)
	}
}
