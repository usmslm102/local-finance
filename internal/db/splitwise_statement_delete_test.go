package db

import (
	"strings"
	"testing"
	"time"

	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwiseStatementDeletionRemovesDependentMappings(t *testing.T) {
	d := safetyDB(t)
	a, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	importBank := func(id string) {
		t.Helper()
		err := d.WithStatementImport(func(w *StatementWriter) error {
			if err := w.CreateStatementImport(&models.StatementImport{ID: id, AccountID: a.ID, Filename: "fictional.csv", FileHash: id, StatementFormat: "CSV", ParserUsed: "demo", ImportedAt: time.Now()}); err != nil {
				return err
			}
			for _, item := range []struct {
				hash, narration string
				amount          float64
			}{{"purchase", "Taxi vendor", 100}, {"repayment", "Asha Example", 25}, {"removed", "Demo shop", 200}} {
				category := "cat_others"
				tx := &models.Transaction{AccountID: a.ID, StatementImportID: &id, TxHash: item.hash, TxDate: "2026-09-01", RawNarration: item.narration, CleanedPayee: item.narration, Amount: item.amount, TxType: models.TxTypeDebit, CategoryID: &category}
				if _, err := w.UpsertTransaction(tx); err != nil {
					return err
				}
			}
			return w.ReconcileSplitwise()
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	assertExpense := func(want float64) {
		t.Helper()
		got, err := d.GetAnalyticsOverview()
		if err != nil || got.TotalExpense != want {
			t.Fatalf("expense: got %+v want %v err %v", got, want, err)
		}
	}
	importBank("original")
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Demo taxi,Travel,100,INR,50,-50\n2026-09-01,Independent meal,General,100,INR,-30,30\n2026-09-01,Sanjay paid Asha,Payment,25,INR,25,-25\n2026-09-01,Removed purchase,General,200,INR,100,-100\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)^Asha Example$"}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[3].ID, models.SplitwiseConfirmation{Ignore: true, Share: entries[3].Share}); err != nil {
		t.Fatal(err)
	}
	assertExpense(280)
	deleted, err := d.DeleteStatementImport("original")
	if err != nil || deleted.DeletedTransactions != 3 {
		t.Fatalf("statement deletion: %+v %v", deleted, err)
	}
	assertExpense(30)
	mappings, err := d.ListSplitwiseMappings()
	if err != nil || len(mappings) != 1 || mappings[0].Entry.ID != entries[1].ID || mappings[0].Bank != nil {
		t.Fatalf("independent expense preservation: %+v %v", mappings, err)
	}
	all, err := d.ListSplitwise()
	if err != nil || len(all) != 2 {
		t.Fatalf("dependent mappings survived: %+v %v", all, err)
	}
	foundRemoval := false
	for _, entry := range all {
		if entry.ID == entries[3].ID {
			foundRemoval = entry.Status == "IGNORED" && entry.RemovedTransactionID != nil
		}
	}
	if !foundRemoval {
		t.Fatal("ignored removal suppression lost")
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || len(members) != 1 || members[0].Pattern != "(?i)^Asha Example$" {
		t.Fatalf("group/rule preservation: %+v %v", members, err)
	}
	importBank("reimport")
	assertExpense(355) // Original bank cash plus the independent meal, without an orphaned taxi share.
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	assertExpense(280)
	all, err = d.ListSplitwise()
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range all {
		if entry.ID == entries[3].ID && entry.Status != "IGNORED" {
			t.Fatal("CSV reimport recreated removed mapping")
		}
	}
}

func TestSplitwiseDataHealthReviewMatchesTransactionFilter(t *testing.T) {
	d := safetyDB(t)
	a, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		hash, narration string
		amount          float64
	}{{"taxi", "Demo vendor", 100}, {"payment", "Asha Example", 25}} {
		category := "cat_others"
		tx := &models.Transaction{AccountID: a.ID, TxHash: item.hash, TxDate: "2026-09-01", RawNarration: item.narration, CleanedPayee: item.narration, Amount: item.amount, TxType: models.TxTypeDebit, CategoryID: &category}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.CreateRule(&models.CategorizationRule{MatchField: "raw_narration", MatchType: "EXACT", MatchPattern: "Fictional taxi", TargetCategoryID: "cat_travel", TxType: "DEBIT", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Fictional taxi,Travel,100,INR,50,-50\n2026-09-01,Sanjay paid Asha,Payment,25,INR,25,-25\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	health, err := d.GetDataHealthAccounts()
	if err != nil || len(health) != 1 {
		t.Fatalf("data health: %+v %v", health, err)
	}
	_, count, err := d.ListTransactions(TransactionFilter{AccountID: a.ID, ReviewNeeded: true})
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || health[0].ReviewCount != count {
		t.Fatalf("data health review count %d differs from visible rows %d", health[0].ReviewCount, count)
	}
	if health[0].TransactionCount != 2 {
		t.Fatalf("lost original bank activity count: %+v", health[0])
	}
}
