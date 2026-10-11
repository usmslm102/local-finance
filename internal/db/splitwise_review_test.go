package db

import (
	"bytes"
	"fmt"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"path/filepath"
	"testing"
)

func TestSplitwiseNearMatchNeedsExplicitApproval(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Fictional Bank", models.AccountTypeSavings, "", "XX9876", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, date string, amount float64, kind models.TxType, excluded, transfer bool) string {
		t.Helper()
		tx := &models.Transaction{AccountID: account.ID, TxHash: hash, TxDate: date, RawNarration: hash, CleanedPayee: hash, Amount: amount, TxType: kind, IsExcluded: excluded, IsTransfer: transfer, Notes: "Keep notes", Tags: "keep-tag"}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	closest := add("Fictional airline", "2026-08-12", 12710, models.TxTypeDebit, false, false) // inclusive15-day boundary
	second := add("Other fictional shop", "2026-08-27", 11250, models.TxTypeDebit, false, false)
	invalidIDs := []string{}
	for _, row := range []struct {
		name, date         string
		kind               models.TxType
		excluded, transfer bool
	}{
		{"too old", "2026-08-11", models.TxTypeDebit, false, false},
		{"too new", "2026-08-28", models.TxTypeDebit, false, false},
		{"credit", "2026-08-27", models.TxTypeCredit, false, false},
		{"excluded", "2026-08-27", models.TxTypeDebit, true, false},
		{"transfer", "2026-08-27", models.TxTypeDebit, false, true},
	} {
		amount := 12000.0
		if row.transfer {
			amount = 12700
		}
		invalidIDs = append(invalidIDs, add(row.name, row.date, amount, row.kind, row.excluded, row.transfer))
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Fictional flight,Plane,12000,INR,6000,-6000\n"), "Fictional travel", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Review != 1 || result.Matched != 0 || result.Skipped != 0 {
		t.Fatalf("import: %+v %v", result, err)
	}
	e := entries[0]
	before, err := d.GetAnalyticsOverview()
	if err != nil {
		t.Fatal(err)
	}
	strict, err := d.SplitwiseCandidates(e.ID, models.SplitwiseMatchOptions{Share: e.Share})
	if err != nil || len(strict) != 0 {
		t.Fatalf("strict candidates: %+v %v", strict, err)
	}
	near, err := d.SplitwiseCandidates(e.ID, models.SplitwiseMatchOptions{Share: e.Share, IncludeNearMatches: true})
	if err != nil || len(near) != 2 || near[0].ID != second || near[1].ID != closest {
		t.Fatalf("near ordering/window: %+v %v", near, err)
	}
	req := models.SplitwiseConfirmation{Share: e.Share, TransactionID: closest, CategoryID: "cat_travel"}
	if err := d.ConfirmSplitwise(e.ID, req); err == nil {
		t.Fatal("large difference auto-approved")
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	still, _ := d.GetAnalyticsOverview()
	if before.TotalExpense != still.TotalExpense {
		t.Fatal("pending review changed expenses")
	}
	req.AllowAmountDifference = true
	for _, id := range invalidIDs {
		invalid := req
		invalid.TransactionID = id
		if err := d.ConfirmSplitwise(e.ID, invalid); err == nil {
			t.Fatalf("approved ineligible bank %s", id)
		}
	}
	altered := req
	altered.Share++
	if err := d.ConfirmSplitwise(e.ID, altered); err == nil {
		t.Fatal("approved changed share")
	}
	if err := d.ConfirmSplitwise(e.ID, req); err != nil {
		t.Fatal(err)
	}
	after, _ := d.GetAnalyticsOverview()
	if after.TotalExpense != before.TotalExpense-12710+6000 {
		t.Fatalf("personal expense: before=%v after=%v", before.TotalExpense, after.TotalExpense)
	}
	bank, err := d.GetTransaction(closest)
	if err != nil || bank.Amount != 12710 || bank.TxHash != "Fictional airline" || bank.Notes != "Keep notes" || bank.Tags != "keep-tag" || bank.PersonalExpenseAmount == nil || *bank.PersonalExpenseAmount != 6000 {
		t.Fatalf("original bank changed: %+v %v", bank, err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(e.ID, models.SplitwiseConfirmation{Ignore: true, Share: e.Share}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	reverted, _ := d.GetAnalyticsOverview()
	if reverted.TotalExpense != before.TotalExpense {
		t.Fatal("dismissed mapping returned")
	}
}

func TestSplitwisePendingUpgradeReconciliation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	d, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	account, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "", "XX4567", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Pending paid expense,General,100,INR,60,-60\n2026-08-27,Pending uneven amount,General,200,INR,200,-200\n"), "Upgrade example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	listed, _ := d.ListSplitwise()
	if len(listed) != 2 {
		t.Fatal("unmatched payer entries disappeared")
	}
	for i, amount := range []float64{100, 260} {
		bank := &models.Transaction{AccountID: account.ID, TxHash: fmt.Sprint("upgrade", i), TxDate: "2026-08-27", Amount: amount, TxType: models.TxTypeDebit}
		if _, err := d.UpsertTransaction(bank); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	d, err = NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	listed, _ = d.ListSplitwise()
	for _, entry := range listed {
		if entry.Description == "Pending paid expense" && entry.Status != "CONFIRMED" {
			t.Fatal("upgrade did not reconcile stored pending exact match")
		}
		if entry.Description == "Pending uneven amount" && (entry.Status != "REVIEW" || entry.TransactionID != nil) {
			t.Fatal("upgrade auto-approved large difference")
		}
	}
	got, _ := d.GetAnalyticsOverview()
	if got.TotalExpense != 300 {
		t.Fatalf("upgrade expenses=%v", got.TotalExpense)
	}
	if err := d.ReconcileSplitwise(); err != nil {
		t.Fatal(err)
	}
	again, _ := d.GetAnalyticsOverview()
	if again.TotalExpense != got.TotalExpense {
		t.Fatal("startup reconciliation not idempotent")
	}
}

func TestSplitwisePaymentCannotApproveAmountDifference(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "", "XX9012", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "Fictional repayment", TxDate: "2026-08-27", RawNarration: "Asha Example", Amount: 140, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Sanjay Example paid Asha Example,Payment,100,INR,100,-100\n"), "Payment example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	candidates, err := d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{Share: 0, IncludeNearMatches: true})
	if err != nil || len(candidates) != 0 {
		t.Fatalf("loosened settlement matching: %+v %v", candidates, err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{TransactionID: bank.ID, AllowAmountDifference: true}); err == nil {
		t.Fatal("settlement bypassed tolerance")
	}
}

func TestSplitwiseReviewSearchAndApprovalDoNotDependOnShortlist(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "", "XX4321", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	var actualID string
	for i := 0; i < 56; i++ {
		amount := float64(106 + i)
		narration := fmt.Sprintf("Unrelated shop %d", i)
		if i == 55 {
			amount = 500
			narration = "Correct fictional vendor"
		}
		tx := &models.Transaction{AccountID: account.ID, TxHash: fmt.Sprint("search", i), TxDate: "2026-08-27", RawNarration: narration, CleanedPayee: narration, Amount: amount, TxType: models.TxTypeDebit}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		if i == 55 {
			actualID = tx.ID
		}
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Fictional shared purchase,General,100,INR,60,-60\n"), "Search example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Review != 1 {
		t.Fatalf("review: %+v %v", result, err)
	}
	near, err := d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{Share: entries[0].Share, IncludeNearMatches: true})
	if err != nil || len(near) != 50 {
		t.Fatalf("shortlist: %d %v", len(near), err)
	}
	for _, candidate := range near {
		if candidate.ID == actualID {
			t.Fatal("test must put correct debit beyond shortlist")
		}
	}
	searched, err := d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{Share: entries[0].Share, IncludeNearMatches: true, Search: "Correct fictional"})
	if err != nil || len(searched) != 1 || searched[0].ID != actualID {
		t.Fatalf("search: %+v %v", searched, err)
	}
	req := models.SplitwiseConfirmation{Share: entries[0].Share, TransactionID: actualID, AllowAmountDifference: true}
	if err := d.ConfirmSplitwise(entries[0].ID, req); err != nil {
		t.Fatal(err)
	}
	// A second expense cannot consume the already-approved debit.
	second := entries[0]
	second.ID = "second-entry"
	second.Description = "Second expense"
	if _, err := d.ImportSplitwise([]models.SplitwiseEntry{second}); err != nil {
		t.Fatal(err)
	}
	req.TransactionID = actualID
	if err := d.ConfirmSplitwise(second.ID, req); err == nil {
		t.Fatal("approved already-linked bank debit")
	}
	// Removal suppression cannot be bypassed via the approval endpoint.
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Ignore: true, Share: entries[0].Share}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(second.ID, req); err == nil {
		t.Fatal("approved removed bank debit")
	}
	if err := d.ResetSplitwise(entries[0].ID); err != nil {
		t.Fatal(err)
	}
	restored, _ := d.ListSplitwise()
	found := false
	for _, entry := range restored {
		if entry.ID == entries[0].ID && entry.Status == "REVIEW" {
			found = true
		}
	}
	if !found {
		t.Fatal("undo did not restore review")
	}
}

func TestSplitwiseStaleReviewCannotReplaceOrDismissConfirmedEntry(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Fictional bank", models.AccountTypeSavings, "", "XX5500", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Stale review example,General,200,INR,120,-120\n"), "Stale example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	exact := &models.Transaction{AccountID: account.ID, TxHash: "stale-exact", TxDate: "2026-08-27", Amount: 200, TxType: models.TxTypeDebit}
	near := &models.Transaction{AccountID: account.ID, TxHash: "stale-near", TxDate: "2026-08-27", Amount: 260, TxType: models.TxTypeDebit}
	for _, bank := range []*models.Transaction{exact, near} {
		if _, err := d.UpsertTransaction(bank); err != nil {
			t.Fatal(err)
		}
	}
	if err := d.ReconcileSplitwise(); err != nil {
		t.Fatal(err)
	}
	for _, req := range []models.SplitwiseConfirmation{
		{Share: entries[0].Share, TransactionID: near.ID, AllowAmountDifference: true, ExpectedStatus: "REVIEW"},
		{Share: entries[0].Share, Ignore: true, ExpectedStatus: "REVIEW"},
		{Share: entries[0].Share, TransactionID: exact.ID, ExpectedStatus: "REVIEW"},
	} {
		if err := d.ConfirmSplitwise(entries[0].ID, req); err == nil {
			t.Fatal("stale review changed confirmed entry")
		}
	}
	mappings, _ := d.ListSplitwiseMappings()
	if len(mappings) != 1 || mappings[0].Bank == nil || mappings[0].Bank.ID != exact.ID {
		t.Fatalf("lost original mapping: %+v", mappings)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Ignore: true, Share: entries[0].Share}); err != nil {
		t.Fatal("explicit mapping removal must work", err)
	}
}

func TestSplitwiseStartupPreservesRestoredManualCategory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manual-restore.db")
	d, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-08-27,Restored expense,General,100,INR,-40,40\n"), "Manual example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	category, note, tags := "cat_health", "Keep manual note", "keep-tag"
	if _, err := d.UpdateTransaction(entries[0].ID, models.UpdateTransactionRequest{CategoryID: &category, Notes: &note, Tags: &tags}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Share: entries[0].Share, Ignore: true}); err != nil {
		t.Fatal(err)
	}
	if err := d.ResetSplitwise(entries[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	d, err = NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	restored, err := d.GetTransaction(entries[0].ID)
	if err != nil || restored.CategoryID == nil || *restored.CategoryID != category || restored.Notes != note || restored.Tags != tags || !restored.IsManualCategory {
		t.Fatalf("manual metadata changed on startup: %+v %v", restored, err)
	}
}
