package service

import (
	"bytes"
	"fmt"
	"local-finance/internal/db"
	"local-finance/internal/models"
	"os"
	"path/filepath"
	"testing"
)

func safetyDatabase(t *testing.T) *db.DB {
	t.Helper()
	d, e := db.NewDB(filepath.Join(t.TempDir(), "ledger.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestSafetyImportRollsBackAllWrites(t *testing.T) {
	for _, table := range []string{"statement_imports", "credit_card_bills", "transactions"} {
		t.Run(table, func(t *testing.T) {
			d := safetyDatabase(t)
			// Let the first transaction through, then fail a later row to exercise rollback.
			when := ""
			if table == "transactions" {
				when = " WHEN (SELECT COUNT(*) FROM transactions) > 0"
			}
			if _, e := d.Exec("CREATE TRIGGER audit_block BEFORE INSERT ON " + table + when + " BEGIN SELECT RAISE(ABORT, 'simulated write failure'); END"); e != nil {
				t.Fatal(e)
			}
			raw, e := os.ReadFile("../../samples/credit_cards/HDFC_Swiggy_Credit_Card.pdf")
			if e != nil {
				t.Fatal(e)
			}
			if _, e = NewTransactionService(d).ImportStatement("sample.pdf", bytes.NewReader(raw), "", "", ""); e == nil {
				t.Fatal("reported success despite write failure")
			}
			for _, name := range []string{"accounts", "statement_imports", "credit_card_bills", "transactions", "card_reward_rules"} {
				rows, e := d.Query("SELECT COUNT(*) FROM " + name)
				if e != nil {
					t.Fatal(e)
				}
				var count int
				rows.Next()
				rows.Scan(&count)
				rows.Close()
				if count != 0 {
					t.Errorf("partial writes to %s: %d", name, count)
				}
			}
		})
	}
}

func TestSafetyAutoReconciliationRequiresPaymentEvidence(t *testing.T) {
	for _, tc := range []struct {
		debit, credit string
		want          int
	}{
		{"GROCERY PURCHASE", "MERCHANT REFUND", 0},
		{"GROCERY PURCHASE", "CREDIT ADJUSTMENT", 0},
		{"CRED CLUB BILL PAYMENT", "MERCHANT REFUND", 0},
		{"CRED CLUB BILL PAYMENT", "PAYMENT RECEIVED VIA IMPS", 1},
	} {
		t.Run(tc.debit+tc.credit, func(t *testing.T) {
			d := safetyDatabase(t)
			savings, _ := d.GetOrCreateAccount("Bank", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
			card, _ := d.GetOrCreateAccount("Card", models.AccountTypeCreditCard, "2", "2", "", "", "", "", "", "", nil)
			for _, tx := range []*models.Transaction{{AccountID: savings.ID, TxHash: "debit", TxDate: "2026-10-01", TxType: models.TxTypeDebit, Amount: 500, RawNarration: tc.debit}, {AccountID: card.ID, TxHash: "credit", TxDate: "2026-10-01", TxType: models.TxTypeCredit, Amount: 500, RawNarration: tc.credit}} {
				if _, e := d.UpsertTransaction(tx); e != nil {
					t.Fatal(e)
				}
			}
			count, _, e := NewReconciliationService(d).ScanAndAutoReconcile()
			if e != nil {
				t.Fatal(e)
			}
			if count != tc.want {
				t.Errorf("linked=%d want %d", count, tc.want)
			}
		})
	}
}

type oversizedStatement struct{}

func TestSafetyReconciliationSearchesPastRefundAndAvoidsAmbiguity(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprintf("ambiguous=%v", ambiguous), func(t *testing.T) {
			d := safetyDatabase(t)
			savings, _ := d.GetOrCreateAccount("Bank", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
			card, _ := d.GetOrCreateAccount("Card", models.AccountTypeCreditCard, "2", "2", "", "", "", "", "", "", nil)
			debit := &models.Transaction{AccountID: savings.ID, TxHash: "debit", TxDate: "2026-10-01", TxType: models.TxTypeDebit, Amount: 500, RawNarration: "CRED BILL PAYMENT"}
			payment := &models.Transaction{AccountID: card.ID, TxHash: "payment", TxDate: "2026-10-01", TxType: models.TxTypeCredit, Amount: 500, RawNarration: "PAYMENT RECEIVED"}
			refund := &models.Transaction{AccountID: card.ID, TxHash: "refund", TxDate: "2026-10-02", TxType: models.TxTypeCredit, Amount: 500, RawNarration: "MERCHANT REFUND"}
			for _, tx := range []*models.Transaction{debit, payment, refund} {
				if _, e := d.UpsertTransaction(tx); e != nil {
					t.Fatal(e)
				}
			}
			if ambiguous {
				other := &models.Transaction{AccountID: card.ID, TxHash: "other-payment", TxDate: "2026-09-30", TxType: models.TxTypeCredit, Amount: 500, RawNarration: "PAYMENT RECEIVED"}
				if _, e := d.UpsertTransaction(other); e != nil {
					t.Fatal(e)
				}
				summary, e := NewReconciliationService(d).GetSummary()
				if e != nil {
					t.Fatal(e)
				}
				seen := map[string]bool{}
				for _, pair := range summary.Candidates {
					if pair.DebitTx.ID == debit.ID {
						seen[pair.CreditTx.ID] = true
					}
				}
				if !seen[payment.ID] || !seen[other.ID] {
					t.Fatal("competing payment suggestions hidden from manual selection")
				}
			}
			count, _, e := NewReconciliationService(d).ScanAndAutoReconcile()
			if e != nil {
				t.Fatal(e)
			}
			want := 1
			if ambiguous {
				want = 0
			}
			if count != want {
				t.Fatalf("linked %d, want %d", count, want)
			}
			after, e := d.GetTransaction(debit.ID)
			if e != nil {
				t.Fatal(e)
			}
			if !ambiguous && (after.TransferPeerID == nil || *after.TransferPeerID != payment.ID) {
				t.Fatal("did not select genuine payment after refund")
			}
		})
	}
}

func (oversizedStatement) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

func TestSafetyStatementReadersBoundInput(t *testing.T) {
	s := NewTransactionService(safetyDatabase(t))
	if _, e := s.ImportStatement("oversized.csv", oversizedStatement{}, "", "", ""); e == nil {
		t.Fatal("import accepted unbounded input")
	}
	if _, e := s.PreviewStatement("oversized.csv", oversizedStatement{}, "", "", ""); e == nil {
		t.Fatal("preview accepted unbounded input")
	}
}

func TestSafetyAtomicReimportPreservesIdentityAndUserEdits(t *testing.T) {
	d := safetyDatabase(t)
	s := NewTransactionService(d)
	raw, e := os.ReadFile("../../samples/credit_cards/HDFC_Swiggy_Credit_Card.pdf")
	if e != nil {
		t.Fatal(e)
	}
	first, e := s.ImportStatement("sample.pdf", bytes.NewReader(raw), "", "", "")
	if e != nil {
		t.Fatal(e)
	}
	txs, _, e := d.ListTransactions(db.TransactionFilter{AccountID: first.AccountID, Limit: 100})
	if e != nil || len(txs) == 0 {
		t.Fatalf("no imported transactions: %v", e)
	}
	original := txs[0]
	category, notes, tags := "cat_health", "Keep this note", "manual-tag"
	manual := true
	if _, e = d.UpdateTransaction(original.ID, models.UpdateTransactionRequest{CategoryID: &category, Notes: &notes, Tags: &tags, IsManualCategory: &manual}); e != nil {
		t.Fatal(e)
	}
	second, e := s.ImportStatement("renamed.pdf", bytes.NewReader(raw), "", "", "")
	if e != nil {
		t.Fatal(e)
	}
	if second.AccountID != first.AccountID || second.InsertedCount != 0 || second.DuplicateCount != first.TotalParsed {
		t.Fatalf("reimport lost idempotence: %+v", second)
	}
	after, e := d.GetTransaction(original.ID)
	if e != nil {
		t.Fatal(e)
	}
	if after.TxHash != original.TxHash || after.CategoryID == nil || *after.CategoryID != category || after.Notes != notes || after.Tags != tags || !after.IsManualCategory {
		t.Fatalf("reimport overwrote user fields: %+v", after)
	}
}
