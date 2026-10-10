package db

import (
	"bytes"
	"fmt"
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"regexp"
	"strings"
	"testing"
)

func TestSplitwiseAutomaticImportRulesMappingsAndRemoval(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo bank", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, payee, date string, amount float64, kind models.TxType) string {
		t.Helper()
		category := "cat_others"
		tx := &models.Transaction{AccountID: account.ID, TxHash: hash, RawNarration: payee, CleanedPayee: payee, TxDate: date, Amount: amount, TxType: kind, CategoryID: &category}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	exact := add("exact", "Lunch vendor", "2026-08-26", 300, models.TxTypeDebit) // 15-day boundary.
	nearer := add("near", "Other vendor", "2026-09-10", 299, models.TxTypeDebit)
	zero := add("zero", "Gift vendor", "2026-09-10", 250, models.TxTypeDebit)
	debit := add("repayment", "UPI/ASHA@FICTIONAL/123", "2026-10-10", 40, models.TxTypeDebit)
	credit := add("credit", "UPI/ASHA@FICTIONAL/456", "2026-10-10", 40, models.TxTypeCredit)
	csv := "Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-10,Lunch,Food,300,INR,200,-200\n2026-09-10,Taxi,Travel,100,INR,-40,40\n2026-09-10,Gift for Asha,General,250,INR,250,-250\n2026-09-10,Missing purchase,General,999,INR,900,-900\n2026-09-10,Others-only dinner,Food,200,INR,0,0\n"
	entries, err := parser.ParseSplitwise(bytes.NewBufferString(csv), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Matched != 2 || result.PaidByOthers != 1 || result.Skipped != 1 || result.Uninvolved != 1 {
		t.Fatalf("auto import: %+v %v", result, err)
	}
	// A saved regex only identifies the bank participant of a CSV Payment.
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)asha@fictional"}); err != nil {
		t.Fatal(err)
	}
	paymentEntries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-10-10,Sanjay Example paid Asha Example,Payment,40,INR,40,-40\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(paymentEntries); err != nil {
		t.Fatal(err)
	}
	before, err := d.GetAnalyticsOverview()
	if err != nil || before.TotalExpense != 439 || before.TotalIncome != 40 {
		t.Fatalf("totals: %+v %v", before, err)
	}
	bank, err := d.GetTransaction(exact)
	if err != nil || bank.Amount != 300 || bank.PersonalExpenseAmount == nil || *bank.PersonalExpenseAmount != 100 {
		t.Fatalf("exact ranking: %+v %v", bank, err)
	}
	bank, err = d.GetTransaction(nearer)
	if err != nil || bank.SplitwiseEntryID != nil {
		t.Fatalf("used approximate before exact: %+v %v", bank, err)
	}
	bank, err = d.GetTransaction(zero)
	if err != nil || bank.PersonalExpenseAmount == nil || *bank.PersonalExpenseAmount != 0 {
		t.Fatalf("zero share payer skipped: %+v %v", bank, err)
	}
	bank, err = d.GetTransaction(debit)
	if err != nil || !bank.IsTransfer || bank.CategoryID == nil || *bank.CategoryID != models.CategoryTransfersID {
		t.Fatalf("outgoing not transfer: %+v %v", bank, err)
	}
	bank, err = d.GetTransaction(credit)
	if err != nil || bank.SplitwiseEntryID != nil || bank.IsTransfer {
		t.Fatalf("incoming auto excluded: %+v %v", bank, err)
	}
	mappings, err := d.ListSplitwiseMappings()
	if err != nil || len(mappings) != 4 {
		t.Fatalf("mappings: %+v %v", mappings, err)
	}
	var taxi models.SplitwiseEntry
	for _, m := range mappings {
		if m.Entry.Description == "Taxi" {
			taxi = m.Entry
			if m.Bank != nil {
				t.Fatal("invented bank movement")
			}
		}
	}
	listed, count, err := d.ListTransactions(TransactionFilter{Search: "Taxi", Limit: 25})
	if err != nil || count != 1 || len(listed) != 1 || listed[0].Amount != 40 || listed[0].AccountID != "" {
		t.Fatalf("synthetic transaction UI: %+v %d %v", listed, count, err)
	}
	if err := d.CreateRule(&models.CategorizationRule{MatchType: "REGEX", MatchField: "raw_narration", MatchPattern: "(?i)taxi", TargetCategoryID: "cat_travel", TxType: "DEBIT", IsActive: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReapplyRules(); err != nil {
		t.Fatal(err)
	}
	synthetic, err := d.GetTransaction(taxi.ID)
	if err != nil || synthetic.CategoryID == nil || *synthetic.CategoryID != "cat_travel" {
		t.Fatalf("normal rules: %+v %v", synthetic, err)
	}
	cat, note, tags := "cat_health", "My note", "my-tag"
	if _, err := d.UpdateTransaction(taxi.ID, models.UpdateTransactionRequest{CategoryID: &cat, Notes: &note, Tags: &tags}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ReapplyRules(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	synthetic, err = d.GetTransaction(taxi.ID)
	if err != nil || synthetic.CategoryID == nil || *synthetic.CategoryID != cat || synthetic.Notes != note || synthetic.Tags != tags {
		t.Fatalf("manual edit lost: %+v %v", synthetic, err)
	}
	if err := d.ConfirmSplitwise(entries[0].ID, models.SplitwiseConfirmation{Ignore: true, Share: entries[0].Share}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(paymentEntries[0].ID, models.SplitwiseConfirmation{Ignore: true}); err != nil {
		t.Fatal(err)
	}
	if err := d.ConfirmSplitwise(paymentEntries[0].ID, models.SplitwiseConfirmation{Ignore: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	bank, err = d.GetTransaction(debit)
	if err != nil || bank.IsTransfer || bank.SplitwiseEntryID != nil {
		t.Fatalf("removed transfer returned: %+v %v", bank, err)
	}
	bank, err = d.GetTransaction(exact)
	if err != nil || bank.PersonalExpenseAmount != nil {
		t.Fatalf("removed split returned: %+v %v", bank, err)
	}
	// A later bank payment without CSV evidence must remain ordinary spending.
	var nextID string
	err = d.WithStatementImport(func(w *StatementWriter) error {
		tx := &models.Transaction{AccountID: account.ID, TxHash: "future", RawNarration: "UPI/asha@fictional/789", CleanedPayee: "asha@fictional", TxDate: "2026-11-01", Amount: 20, TxType: models.TxTypeDebit}
		if _, err := w.UpsertTransaction(tx); err != nil {
			return err
		}
		nextID = tx.ID
		return w.ReconcileSplitwise()
	})
	if err != nil {
		t.Fatal(err)
	}
	bank, err = d.GetTransaction(nextID)
	if err != nil || bank.IsTransfer || bank.SplitwiseEntryID != nil {
		t.Fatalf("future bank payment classified without CSV: %+v %v", bank, err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "["}); err == nil {
		t.Fatal("accepted invalid regex")
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || members[0].Pattern != "(?i)asha@fictional" {
		t.Fatalf("invalid regex overwrote saved rule: %+v %v", members, err)
	}
}

func TestSplitwiseUninvolvedGroupRetainsOnlyMemberSettings(t *testing.T) {
	d := safetyDB(t)
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example,Dev Example\n2026-09-01,Other people lunch,Food,100,INR,0,50,-50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Uninvolved != 1 {
		t.Fatalf("uninvolved: %+v %v", result, err)
	}
	all, err := d.ListSplitwise()
	if err != nil || len(all) != 0 {
		t.Fatalf("added uninvolved transaction: %+v %v", all, err)
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || len(members) != 2 {
		t.Fatalf("lost group member settings: %+v %v", members, err)
	}
}

func TestSplitwiseExplicitExpensePrecedesGenericMemberTransfer(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "purchase", RawNarration: "UPI Asha Example", CleanedPayee: "Asha Example", TxDate: "2026-09-01", Amount: 100, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-02,Shared purchase,General,100,INR,60,-60\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Matched != 1 || result.Transfers != 0 {
		t.Fatalf("priority: %+v %v", result, err)
	}
	spending, err := d.GetAnalyticsOverview()
	if err != nil || spending.TotalExpense != 40 {
		t.Fatalf("dropped own share: %+v %v", spending, err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)asha"}); err != nil {
		t.Fatal(err)
	}
	spending, err = d.GetAnalyticsOverview()
	if err != nil || spending.TotalExpense != 40 {
		t.Fatalf("regex overwrote purchase: %+v %v", spending, err)
	}
}

func TestSplitwiseExpenseRecoversAfterCsvThenBankThenCsv(t *testing.T) {
	d := safetyDB(t)
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-02,Shared purchase,General,100,INR,60,-60\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Skipped != 1 {
		t.Fatalf("missing bank: %+v %v", result, err)
	}
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "purchase", RawNarration: "UPI Asha Example", CleanedPayee: "Asha Example", TxDate: "2026-09-01", Amount: 100, TxType: models.TxTypeDebit}
	err = d.WithStatementImport(func(w *StatementWriter) error {
		if _, err := w.UpsertTransaction(bank); err != nil {
			return err
		}
		return w.ReconcileSplitwise()
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err = d.ImportSplitwiseAutomatically(entries)
	if err != nil || result.Matched != 1 {
		t.Fatalf("reimport recovery: %+v %v", result, err)
	}
	spending, err := d.GetAnalyticsOverview()
	if err != nil || spending.TotalExpense != 40 {
		t.Fatalf("own share lost: %+v %v", spending, err)
	}
	mappings, err := d.ListSplitwiseMappings()
	if err != nil || len(mappings) != 1 || mappings[0].Entry.Kind != "EXPENSE" || mappings[0].Bank == nil || mappings[0].Bank.ID != bank.ID {
		t.Fatalf("generic transfer not replaced: %+v %v", mappings, err)
	}
}

func TestSplitwisePaymentRequiresCsvAmountDirectionAndMember(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	add := func(hash, narration, date string, amount float64, direction models.TxType) string {
		t.Helper()
		tx := &models.Transaction{AccountID: account.ID, TxHash: hash, RawNarration: narration, CleanedPayee: narration, TxDate: date, Amount: amount, TxType: direction}
		if _, err := d.UpsertTransaction(tx); err != nil {
			t.Fatal(err)
		}
		return tx.ID
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Ravi M,Dev Example\n2026-09-02,Taxi,Travel,100,INR,-50,50,0\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	unrelated := add("unrelated", "UPI/ravi@fictional", "2026-09-01", 70, models.TxTypeDebit)
	wrongPerson := add("wrong-person", "UPI/dev@fictional", "2026-09-02", 50, models.TxTypeCredit)
	wrongAmount := add("wrong-amount", "UPI/ravi@fictional", "2026-09-02", 80, models.TxTypeCredit)
	wrongDirection := add("wrong-direction", "UPI/ravi@fictional", "2026-09-02", 50, models.TxTypeDebit)
	right := add("right", "UPI/ravi@fictional", "2026-09-01", 50, models.TxTypeCredit)
	for i := 0; i < 55; i++ {
		add(fmt.Sprintf("noise-%d", i), "Other person", "2026-09-02", 50, models.TxTypeCredit)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Ravi M", Pattern: "(?i)ravi@fictional"}); err != nil {
		t.Fatal(err)
	}
	tx, err := d.GetTransaction(unrelated)
	if err != nil || tx.IsTransfer {
		t.Fatalf("regex alone created transfer: %+v %v", tx, err)
	}
	payments, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Ravi M,Dev Example\n2026-09-02,Ravi M paid Sanjay Example,Payment,50,INR,-50,50,0\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err := d.ImportSplitwiseAutomatically(payments)
	if err != nil || result.Transfers != 1 {
		t.Fatalf("payment import: %+v %v", result, err)
	}
	tx, err = d.GetTransaction(right)
	if err != nil || !tx.IsTransfer || tx.SplitwiseEntryID == nil {
		t.Fatalf("payment missing: %+v %v", tx, err)
	}
	for _, id := range []string{unrelated, wrongPerson, wrongAmount, wrongDirection} {
		tx, err = d.GetTransaction(id)
		if err != nil || tx.IsTransfer || tx.SplitwiseEntryID != nil {
			t.Fatalf("unrelated bank transaction changed: %+v %v", tx, err)
		}
	}
	incoming, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Ravi M,Dev Example\n2026-09-03,Unmatched payment,Payment,999,INR,-999,999,0\n2026-09-03,Uninvolved payment,Payment,10,INR,0,10,-10\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	result, err = d.ImportSplitwiseAutomatically(incoming)
	if err != nil || result.Skipped != 1 || result.Uninvolved != 1 {
		t.Fatalf("unmatched payment: %+v %v", result, err)
	}
}

func TestSplitwiseMatchedPaymentSuggestsReusableRule(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	vpa := "ravi.m+test@fictional"
	bank := &models.Transaction{AccountID: account.ID, TxHash: "first-payment", TxDate: "2026-09-01", Amount: 50, TxType: models.TxTypeCredit, RawNarration: "UPI unknown alias 123", CleanedPayee: "Unknown alias", UPIVPA: &vpa}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	payments, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Ravi M\n2026-09-02,Ravi M paid Sanjay Example,Payment,50,INR,-50,50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if payments[0].Counterparty != "Ravi M" {
		t.Fatalf("lost balance participant: %+v", payments)
	}
	result, err := d.ImportSplitwiseAutomatically(payments)
	if err != nil || result.Matched != 1 {
		t.Fatalf("first payment: %+v %v", result, err)
	}
	members, err := d.ListSplitwiseMembers()
	if err != nil || len(members) != 1 || members[0].MatchedPayments != 1 || members[0].SuggestedPattern == "" {
		t.Fatalf("suggestion: %+v %v", members, err)
	}
	suggested := members[0]
	re, err := regexp.Compile(suggested.SuggestedPattern)
	if err != nil || !re.MatchString(vpa) || re.MatchString("ravixmttest@fictional") {
		t.Fatalf("unsafe suggestion: %+v %v", suggested, err)
	}
	suggested.Pattern = suggested.SuggestedPattern
	if err := d.SaveSplitwiseMemberAliases(suggested); err != nil {
		t.Fatal(err)
	}
	unrelated := &models.Transaction{AccountID: account.ID, TxHash: "unrelated", TxDate: "2026-09-02", Amount: 500, TxType: models.TxTypeDebit, RawNarration: vpa, UPIVPA: &vpa}
	if err := d.WithStatementImport(func(w *StatementWriter) error {
		if _, err := w.UpsertTransaction(unrelated); err != nil {
			return err
		}
		return w.ReconcileSplitwise()
	}); err != nil {
		t.Fatal(err)
	}
	tx, err := d.GetTransaction(unrelated.ID)
	if err != nil || tx.IsTransfer {
		t.Fatalf("suggested rule invented payment: %+v %v", tx, err)
	}
	members, err = d.ListSplitwiseMembers()
	if err != nil || members[0].Pattern != suggested.Pattern {
		t.Fatalf("one-step save lost rule: %+v %v", members, err)
	}
}

func TestSplitwiseLegacyNameOnlyPaymentCannotReapply(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "legacy", TxDate: "2026-09-01", RawNarration: "Asha Example", CleanedPayee: "Asha Example", Amount: 50, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entry := models.SplitwiseEntry{ID: "sw-settlement-" + bank.ID, Group: "Example", Person: "Sanjay Example", Members: []string{"Sanjay Example", "Asha Example"}, Date: bank.TxDate, Description: "Settlement with Asha Example", Cost: 5000, Net: 5000, Kind: "PAYMENT", Status: "REVIEW"}
	if _, err := d.ImportSplitwise([]models.SplitwiseEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: entry.Group, Name: "Asha Example", Pattern: "(?i)asha"}); err != nil {
		t.Fatal(err)
	}
	tx, err := d.GetTransaction(bank.ID)
	if err != nil || tx.IsTransfer || tx.SplitwiseEntryID != nil {
		t.Fatalf("legacy payment recreated: %+v %v", tx, err)
	}
}

func TestSplitwisePaymentEvidenceUpgradePreservesCsvAndRemoval(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "csv", TxDate: "2026-09-01", RawNarration: "Ravi Makwana", CleanedPayee: "Ravi Makwana", Amount: 50, TxType: models.TxTypeCredit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(bytes.NewBufferString("Date,Description,Category,Cost,Currency,Sanjay Example,Ravi Makwana\n2026-09-02,Ravi m. paid Sanjay N.,Payment,50,INR,-50,50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"CONFIRMED", "REVIEW", "IGNORED"} {
		if _, err := d.conn.Exec(`INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,member_names) VALUES(?,'Example','Sanjay Example','2026-09-02','Settlement with Ravi Makwana','Payment',5000,-5000,0,'PAYMENT',?,'[]')`, "sw-settlement-"+status, status); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := d.conn.Exec(`ALTER TABLE splitwise_entries DROP COLUMN counterparty; DELETE FROM goose_db_version WHERE version_id>=19`); err != nil {
		t.Fatal(err)
	}
	path := d.path
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	members, err := upgraded.ListSplitwiseMembers()
	if err != nil || len(members) != 1 || members[0].MatchedPayments != 1 || members[0].SuggestedPattern == "" {
		t.Fatalf("retained CSV mapping hidden: %+v %v", members, err)
	}
	all, err := upgraded.ListSplitwise()
	if err != nil || len(all) != 2 {
		t.Fatalf("legacy cleanup: %+v %v", all, err)
	}
	for _, entry := range all {
		if strings.HasPrefix(entry.ID, "sw-settlement-") && entry.Status != "IGNORED" {
			t.Fatalf("legacy evidence retained: %+v", entry)
		}
	}
	if err := upgraded.WithStatementImport(func(w *StatementWriter) error { return w.ReconcileSplitwise() }); err != nil {
		t.Fatal(err)
	}
	if _, err := upgraded.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	tx, err := upgraded.GetTransaction(bank.ID)
	if err != nil || tx.SplitwiseEntryID == nil || *tx.SplitwiseEntryID != entries[0].ID {
		t.Fatalf("CSV match lost: %+v %v", tx, err)
	}
}
