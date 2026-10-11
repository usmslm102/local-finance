package db

import (
	"strings"
	"testing"

	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwiseGenericPaymentEndpointsRequireMemberEvidence(t *testing.T) {
	d := safetyDB(t)
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-02,Asha Example paid Sanjay Example,Payment,50,INR,-50,50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)^asha@fictional$"}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	a, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "evidence", "evidence", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, payee string) string {
		t.Helper()
		tx := &models.Transaction{AccountID: a.ID, TxHash: hash, TxDate: "2026-09-01", RawNarration: payee, CleanedPayee: payee, Amount: 50, TxType: models.TxTypeCredit}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	wrong := add("salary-evidence", "Salary")
	got, err := d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{})
	if err != nil || len(got) != 0 {
		t.Fatalf("wrong member suggested: %+v %v", got, err)
	}
	got, err = d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{PaymentMember: &models.SplitwiseMember{Name: "Salary"}})
	if err != nil || len(got) != 0 {
		t.Fatalf("caller substituted payment evidence: %+v %v", got, err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{TransactionID: wrong}); err == nil {
		t.Fatal("salary confirmed as member repayment")
	}
	right := add("asha-evidence", "asha@fictional")
	got, err = d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{})
	if err != nil || len(got) != 1 || got[0].ID != right {
		t.Fatalf("valid member candidates: %+v %v", got, err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{TransactionID: right}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Ignore: true}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Ignore: true}); err != nil {
		t.Fatal(err)
	}
	bank, err := d.GetTransaction(right)
	if err != nil || bank.SplitwiseEntryID != nil {
		t.Fatalf("removal failed: %+v %v", bank, err)
	}
	legacy := entries[0]
	legacy.ID = "sw-settlement-" + wrong
	if _, err := d.ImportSplitwise([]models.SplitwiseEntry{legacy}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.SplitwiseCandidates(legacy.ID, models.SplitwiseMatchOptions{}); err == nil {
		t.Fatal("synthetic settlement offered as CSV evidence")
	}
	if err := d.ConfirmSplitwise(legacy.ID, models.SplitwiseConfirmation{TransactionID: wrong}); err == nil {
		t.Fatal("synthetic settlement confirmed as CSV evidence")
	}
}

func TestSplitwisePaymentSearchCannotManufactureSoleCandidate(t *testing.T) {
	d := safetyDB(t)
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-02,Asha Example paid Sanjay Example,Payment,50,INR,-50,50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	a, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "search", "search", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, payee := range []string{"Salary", "Other receipt"} {
		tx := &models.Transaction{AccountID: a.ID, TxHash: payee, TxDate: "2026-09-01", RawNarration: payee, CleanedPayee: payee, Amount: 50, TxType: models.TxTypeCredit}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
	}
	got, err := d.SplitwiseCandidates(entries[0].ID, models.SplitwiseMatchOptions{Search: "Salary"})
	if err != nil || len(got) != 0 {
		t.Fatalf("search bypassed ambiguous evidence: %+v %v", got, err)
	}
}
