package db

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwiseMemberSettlementDoesNotDoubleCountExpense(t *testing.T) {
	d := safetyDB(t)
	data, err := os.ReadFile("../../samples/splitwise/fictional-group.csv")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewReader(data), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.ImportSplitwise(entries); err != nil {
		t.Fatal(err)
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || len(members) != 2 {
		t.Fatalf("members: %+v %v", members, err)
	}
	if err = d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Aliases: []string{"asha@fictional", "  ASHA@FICTIONAL "}}); err != nil {
		t.Fatal(err)
	}
	members, err = d.ListSplitwiseMembers()
	if err != nil || len(members[0].Aliases) != 1 {
		t.Fatalf("aliases: %+v %v", members, err)
	}
	if err = d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Unknown", Aliases: []string{"unknown@fictional"}}); err == nil {
		t.Fatal("accepted unknown member")
	}
	account, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, payee string, amount float64, kind models.TxType) string {
		t.Helper()
		category := "cat_food"
		tx := &models.Transaction{AccountID: account.ID, TxHash: hash, TxDate: "2026-10-01", RawNarration: "UPI/" + payee + "/123", CleanedPayee: payee, Amount: amount, TxType: kind, CategoryID: &category, Notes: "keep", Tags: "keep", IsManualCategory: true}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	payment := add("settlement", "asha@fictional", 50, models.TxTypeDebit)
	receipt := add("received", "Dev Example", 80, models.TxTypeCredit)
	add("unrelated", "Joanne Example", 10, models.TxTypeDebit)
	if err = d.RecalculateAccountBalance(account.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := d.ListAccounts()
	// Taxi was paid by another member a month earlier. Settlement is not bound
	// by the expense's 15-day matching window and must not add a second expense.
	if err = d.ConfirmSplitwise(entries[1].ID, models.SplitwiseConfirmation{Share: 5000}); err != nil {
		t.Fatal(err)
	}
	totals, err := d.GetAnalyticsOverview()
	if err != nil || totals.TotalExpense != 110 || totals.TotalIncome != 80 {
		t.Fatalf("before confirmation: %+v %v", totals, err)
	}
	suggestions, err := d.SplitwiseSettlementCandidates()
	if err != nil || len(suggestions) != 2 {
		t.Fatalf("suggestions: %+v %v", suggestions, err)
	}
	for _, req := range []models.SplitwiseSettlementConfirmation{{TransactionID: payment, Group: "Example", Member: "Asha Example"}, {TransactionID: receipt, Group: "Example", Member: "Dev Example"}} {
		if err = d.ConfirmSplitwiseSettlement(req); err != nil {
			t.Fatal(err)
		}
	}
	totals, err = d.GetAnalyticsOverview()
	if err != nil || totals.TotalExpense != 60 || totals.TotalIncome != 0 {
		t.Fatalf("double counted settlement: %+v %v", totals, err)
	}
	after, _ := d.ListAccounts()
	if before[0].CurrentBalance != after[0].CurrentBalance {
		t.Fatal("changed bank balance")
	}
	statement, err := d.GetTransaction(payment)
	if err != nil || statement.Amount != 50 || statement.Notes != "keep" || statement.Tags != "keep" || statement.TxHash != "settlement" || statement.SplitwiseKind == nil || *statement.SplitwiseKind != "PAYMENT" {
		t.Fatalf("bank metadata: %+v %v", statement, err)
	}
	if statement.PersonalExpenseAmount == nil || *statement.PersonalExpenseAmount != 0 {
		t.Fatal("missing zero settlement share")
	}
	suggestions, err = d.SplitwiseSettlementCandidates()
	if err != nil || len(suggestions) != 0 {
		t.Fatalf("confirmed still suggested: %+v %v", suggestions, err)
	}
	if err = d.ConfirmSplitwiseSettlement(models.SplitwiseSettlementConfirmation{TransactionID: payment, Group: "Example", Member: "Asha Example"}); err == nil {
		t.Fatal("double linked settlement")
	}
	if err = d.ResetSplitwise("sw-settlement-" + payment); err != nil {
		t.Fatal(err)
	}
	totals, err = d.GetAnalyticsOverview()
	if err != nil || totals.TotalExpense != 110 {
		t.Fatalf("undo: %+v %v", totals, err)
	}
	if err = d.ConfirmSplitwiseSettlement(models.SplitwiseSettlementConfirmation{TransactionID: payment, Group: "Example", Member: "Asha Example"}); err != nil {
		t.Fatal(err)
	}
	// Re-import updates names but preserves classifications and aliases.
	for i := range entries {
		entries[i].Members = append(entries[i].Members, "New Example")
	}
	if count, err := d.ImportSplitwise(entries); err != nil || count != 6 {
		t.Fatalf("membership refresh: %d %v", count, err)
	}
	exported, err := d.ExportAllDataJSON()
	if err != nil || len(exported.SplitwiseMembers) != 3 || len(exported.SplitwiseMembers[0].Aliases) != 1 {
		t.Fatalf("export: %+v %v", exported, err)
	}
	backup := filepath.Join(t.TempDir(), "members.db")
	if err := d.BackupTo(backup); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(backup)
	if err != nil {
		t.Fatal(err)
	}
	if err = d.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	var aliases int
	if err = d.conn.QueryRow(`SELECT COUNT(*) FROM splitwise_member_aliases`).Scan(&aliases); err != nil || aliases != 0 {
		t.Fatalf("reset aliases: %d %v", aliases, err)
	}
	if err := d.RestoreFrom(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	members, err = d.ListSplitwiseMembers()
	if err != nil || len(members) != 3 || len(members[0].Aliases) != 1 {
		t.Fatalf("restore aliases: %+v %v", members, err)
	}
	totals, err = d.GetAnalyticsOverview()
	if err != nil || totals.TotalExpense != 60 || totals.TotalIncome != 0 {
		t.Fatalf("restore settlements: %+v %v", totals, err)
	}
}

func TestSplitwiseMemberNameBoundaries(t *testing.T) {
	for _, tc := range []struct {
		payee string
		want  bool
	}{{"UPI/Ann Example/123", true}, {"Joann Example", false}, {"ANN EXAMPLE", true}, {"Ann Exampleton", false}, {"Other merchant", false}} {
		if got := matchesSplitwiseMember(models.Transaction{RawNarration: tc.payee}, models.SplitwiseMember{Name: "Ann Example"}); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.payee, got, tc.want)
		}
	}
}

func TestSplitwiseSettlementRejectsLinkedExpenseAndInternalTransfer(t *testing.T) {
	d := safetyDB(t)
	entry := models.SplitwiseEntry{ID: "expense", Group: "Example", Person: "Sanjay", Members: []string{"Sanjay", "Asha"}, Date: "2026-10-01", Description: "Meal", Category: "Food", Cost: 10000, Net: 5000, Share: 5000, Kind: "EXPENSE", Status: "REVIEW"}
	if _, err := d.ImportSplitwise([]models.SplitwiseEntry{entry}); err != nil {
		t.Fatal(err)
	}
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	tx := &models.Transaction{AccountID: account.ID, TxHash: "expense", TxDate: entry.Date, RawNarration: "Asha", CleanedPayee: "Asha", Amount: 100, TxType: models.TxTypeDebit}
	if _, err = d.UpsertTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if err = d.ConfirmSplitwise(entry.ID, models.SplitwiseConfirmation{Share: 5000, TransactionID: tx.ID}); err != nil {
		t.Fatal(err)
	}
	if err = d.ConfirmSplitwiseSettlement(models.SplitwiseSettlementConfirmation{TransactionID: tx.ID, Group: entry.Group, Member: "Asha"}); err == nil {
		t.Fatal("overwrote confirmed expense")
	}
	if err = d.ResetSplitwise(entry.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = d.conn.Exec(`UPDATE transactions SET transfer_peer_id='other' WHERE id=?`, tx.ID); err != nil {
		t.Fatal(err)
	}
	if err = d.ConfirmSplitwiseSettlement(models.SplitwiseSettlementConfirmation{TransactionID: tx.ID, Group: entry.Group, Member: "Asha"}); err == nil {
		t.Fatal("accepted paired internal transfer")
	}
}
