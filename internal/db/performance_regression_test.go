package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func improvementDB(t *testing.T) *DB {
	t.Helper()
	d, err := NewDB(filepath.Join(t.TempDir(), "improvements.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
func improvementExec(t *testing.T, d *DB, query string, args ...any) {
	t.Helper()
	if _, err := d.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func improvementAccount(t *testing.T, d *DB, id string) {
	t.Helper()
	improvementExec(t, d, `INSERT INTO accounts(id,bank_name,account_type,account_number_mask,currency,credit_limit) VALUES(?,'Fictional Bank','CREDIT_CARD',?,'INR',10000)`, id, id)
}
func improvementTx(t *testing.T, d *DB, id, payee, date, category, kind string, amount float64) {
	t.Helper()
	improvementExec(t, d, `INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,category_id,tx_type,amount,payment_mode,reference_number) VALUES(?,'a',?,?,?, ?,NULLIF(?,''),?,?,'UPI','')`, id, id, date, payee, payee, category, kind, amount)
}

func TestLatestCardVersionsAgreeAcrossOverviewAndPortfolio(t *testing.T) {
	d := improvementDB(t)
	improvementAccount(t, d, "a")
	improvementAccount(t, d, "b")
	bill := func(id, account, date, created string, amount, rewards, cb float64) {
		improvementExec(t, d, `INSERT INTO credit_card_bills(id,account_id,statement_date,payment_due_date,total_due_amount,reward_points_balance,cashback_earned,created_at) VALUES(?,?,?,'2026-12-01',?,?,?,?)`, id, account, date, amount, rewards, cb, created)
	}
	bill("old", "a", "2026-10-01", "2026-10-02", 60, 5, 2)
	bill("first", "a", "2026-11-01", "2026-11-02", 100, 10, 3)
	bill("repeat", "a", "2026-11-01", "2026-11-03", 100, 10, 3)
	bill("other", "b", "2026-11-01", "2026-11-03", 30, 20, 4)
	check := func(want float64) {
		t.Helper()
		overview, err := d.GetAnalyticsOverview()
		if err != nil {
			t.Fatal(err)
		}
		portfolio, err := d.GetCardPortfolioOverview()
		if err != nil {
			t.Fatal(err)
		}
		latest, err := d.ListLatestCreditCardBills("")
		if err != nil {
			t.Fatal(err)
		}
		var total float64
		for _, b := range latest {
			total += b.TotalDueAmount
		}
		if overview.TotalCreditDue != want || total != want || len(latest) != 2 || len(overview.UpcomingBills) != 2 {
			t.Fatalf("latest mismatch: %+v latest=%+v", overview, latest)
		}
		raw, err := d.ListCreditCardBills("")
		if err != nil || len(raw) < 4 {
			t.Fatal("source history lost", err)
		}
		// Card portfolio must select the same amended version, including same-date ties.
		var cardSum float64
		for _, c := range portfolio.Cards {
			cardSum += c.TotalDueAmount
		}
		if cardSum != want {
			t.Fatalf("portfolio=%v overview=%v", cardSum, want)
		}
		if overview.TotalRewardPoints != 30 || overview.TotalCashbackEarned != 9 {
			t.Fatalf("rewards/cashback counted versions: %+v", overview)
		}
	}
	check(130)
	bill("amend", "a", "2026-11-01", "2026-11-04", 110, 10, 3)
	check(140)
	bill("repeat-amend", "a", "2026-11-01", "2026-11-05", 110, 10, 3)
	check(140)
}

func TestMerchantSummaryAndExactHistory(t *testing.T) {
	d := improvementDB(t)
	improvementAccount(t, d, "a")
	improvementTx(t, d, "one", "Alpha", "2026-02-01", "cat_food", "DEBIT", 10)
	improvementTx(t, d, "two", "Zed", "2026-02-02", "cat_groceries", "DEBIT", 200)
	improvementTx(t, d, "credit", "Zed", "2026-02-03", "cat_food", "CREDIT", 50)
	for _, sortBy := range []string{"total_spend", "name", "aov", "recent", "tx_count"} {
		full, err := d.ListMerchants("", "", sortBy)
		if err != nil {
			t.Fatal(err)
		}
		lean, err := d.ListMerchantsWithSource("", "", sortBy, false)
		if err != nil {
			t.Fatal(err)
		}
		if full.AverageOrderValue != 105 || full.TotalSpend != 210 || full.TopCategory != "Groceries & Essentials" {
			t.Fatalf("bad summary %+v", full)
		}
		for i := range full.Merchants {
			if full.Merchants[i].PrimarySource != "Fictional Bank" {
				t.Fatal("default source lost")
			}
			full.Merchants[i].PrimarySource = ""
		}
		if !reflect.DeepEqual(full, lean) {
			t.Fatal("source opt-out changed financial fields")
		}
	}
	for i := 0; i < 87; i++ {
		improvementTx(t, d, fmt.Sprintf("p%03d", i), "Frequent", "2026-03-01", "cat_food", "DEBIT", 1)
	}
	improvementTx(t, d, "unrelated", "Unrelated", "2026-03-01", "cat_food", "DEBIT", 99)
	improvementExec(t, d, `UPDATE transactions SET raw_narration='Frequent but another payee' WHERE id='unrelated'`)
	seen := map[string]bool{}
	for page := 1; page <= 2; page++ {
		profile, err := d.GetMerchantProfilePage("Frequent", page)
		if err != nil {
			t.Fatal(err)
		}
		if profile.TotalTxCount != 87 || profile.HistoryPage != page {
			t.Fatalf("bad profile %+v", profile)
		}
		for _, tx := range profile.RecentTransactions {
			if tx.CleanedPayee != "Frequent" || seen[tx.ID] {
				t.Fatal("wrong/repeated history", tx.ID)
			}
			seen[tx.ID] = true
		}
	}
	if len(seen) != 87 {
		t.Fatal("missing old purchases", len(seen))
	}
}

func TestGroupedReportsPreserveSharesDatesAndExclusions(t *testing.T) {
	d := improvementDB(t)
	improvementAccount(t, d, "a")
	improvementTx(t, d, "feb", "Shared", "2024-02-29", "cat_food", "DEBIT", 200)
	improvementTx(t, d, "next", "Later", "2024-03-01", "cat_food", "DEBIT", 50)
	improvementTx(t, d, "refund", "Refund", "2024-02-29", "cat_food", "CREDIT", 100)
	improvementTx(t, d, "excluded", "Excluded", "2024-02-29", "cat_food", "DEBIT", 900)
	improvementExec(t, d, `UPDATE transactions SET is_excluded=1 WHERE id='excluded'`)
	improvementExec(t, d, `INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,transaction_id,category_id) VALUES('share','G','Self','2024-02-29','Shared','Food',20000,0,8000,'EXPENSE','CONFIRMED','feb','cat_food')`)
	budget, err := d.GetCategoryBudgetSummary("2024-02")
	if err != nil {
		t.Fatal(err)
	}
	overview, err := d.GetAnalyticsOverviewRange("2024-02-01", "2024-02-29")
	if err != nil {
		t.Fatal(err)
	}
	flow, err := d.GetCashFlowIntelligence("2024-02")
	if err != nil {
		t.Fatal(err)
	}
	if overview.TotalExpense != 80 || flow.Summary.TotalOutflow != 80 {
		t.Fatal("share or range changed", overview.TotalExpense, flow.Summary.TotalOutflow)
	}
	found := false
	for _, b := range budget.Budgets {
		if b.CategoryID == "cat_food" {
			found = true
			if b.ActualSpend != 80 {
				t.Fatalf("budget %+v", b)
			}
		}
	}
	if !found {
		t.Fatal("missing food budget")
	}
	for _, c := range flow.CategoryComparisons {
		if c.CategoryID == "cat_food" {
			if len(c.History) != 2 || c.History[1].Amount != 50 || c.ThreeMonthAvg != 65 || len(c.TopPayees) != 1 || c.TopPayees[0].TotalSpent != 80 {
				t.Fatalf("history/payees %+v", c)
			}
		}
	}
}

func TestBoundedReadsAndRevision(t *testing.T) {
	d := improvementDB(t)
	improvementAccount(t, d, "a")
	before, err := d.DataRevision()
	if err != nil {
		t.Fatal(err)
	}
	again, _ := d.DataRevision()
	if again != before {
		t.Fatal("read changed revision")
	}
	hashes := []string{}
	ids := []string{}
	for i := 0; i < 510; i++ {
		id := fmt.Sprintf("hash-%03d", i)
		hashes = append(hashes, id)
		ids = append(ids, id)
		improvementTx(t, d, id, "Shop", "2026-01-01", "cat_food", "DEBIT", 1)
	}
	hashes = append(hashes, "missing", hashes[0])
	found, err := d.ExistingTxHashes(hashes)
	if err != nil || len(found) != 510 || found["missing"] {
		t.Fatal("hash lookup", len(found), err)
	}
	if err := d.MarkTransactionsRecurring(ids); err != nil {
		t.Fatal(err)
	}
	lean, err := d.ListSubscriptionCandidates()
	if err != nil {
		t.Fatal(err)
	}
	rich, _, err := d.ListTransactions(TransactionFilter{TxType: "DEBIT", Limit: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if len(lean) != len(rich) {
		t.Fatal("candidate count")
	}
	for i, tx := range lean {
		if tx.ID != rich[i].ID || tx.Amount != rich[i].Amount || tx.CleanedPayee != rich[i].CleanedPayee {
			t.Fatal("candidate semantics changed")
		}
	}
	changed, _ := d.DataRevision()
	if changed == before {
		t.Fatal("same-connection write invisible")
	}
	other, err := sql.Open("sqlite", d.GetPath())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if _, err := other.Exec(`UPDATE transactions SET notes='external edit' WHERE id=?`, ids[0]); err != nil {
		t.Fatal(err)
	}
	external, _ := d.DataRevision()
	if external == changed {
		t.Fatal("other-connection write invisible")
	}
}

func TestReconcileSkipsFinishedHistoryAndUnneededMembers(t *testing.T) {
	d := improvementDB(t)
	improvementExec(t, d, `INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,member_names) VALUES('finished','G','Self','2026-01-01','Food','Food',100,0,100,'EXPENSE','CONFIRMED','legacy-invalid-json')`)
	if err := d.ReconcileSplitwise(); err != nil {
		t.Fatalf("decoded finished history: %v", err)
	}
}

func TestBudgetFallbackAlwaysStartsAtFirstOfMonth(t *testing.T) {
	for _, now := range []time.Time{time.Date(2025, 1, 31, 12, 0, 0, 0, time.UTC), time.Date(2024, 3, 31, 12, 0, 0, 0, time.UTC)} {
		month := budgetMonth("invalid", now)
		if month.Day() != 1 || month.Month() != now.Month() || month.AddDate(0, 1, 0).Month() != now.Month()+1 {
			t.Fatal("month fallback overflow", month)
		}
	}
}

func TestNullPayeeDoesNotFailCashflowOrInventTopPayee(t *testing.T) {
	d := improvementDB(t)
	improvementAccount(t, d, "a")
	improvementTx(t, d, "null", "Known", "2026-01-01", "cat_food", "DEBIT", 10)
	improvementExec(t, d, `UPDATE transactions SET cleaned_payee=NULL WHERE id='null'`)
	flow, err := d.GetCashFlowIntelligence("2026-01")
	if err != nil {
		t.Fatal(err)
	}
	if flow.Summary.TotalOutflow != 10 {
		t.Fatal("lost anonymous purchase")
	}
	for _, c := range flow.CategoryComparisons {
		if c.CategoryID == "cat_food" && len(c.TopPayees) != 0 {
			t.Fatal("invented payee", c.TopPayees)
		}
	}
}

func TestRestoreChangesFreshnessToken(t *testing.T) {
	source := improvementDB(t)
	destination := improvementDB(t)
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := source.BackupTo(backup); err != nil {
		t.Fatal(err)
	}
	before, err := destination.DataRevision()
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := destination.RestoreFrom(file); err != nil {
		t.Fatal(err)
	}
	after, err := destination.DataRevision()
	if err != nil || before == after {
		t.Fatal("restore invisible", before, after, err)
	}
}

func TestReuploadOfOldSourceDoesNotReplaceAmendment(t *testing.T) {
	for _, tied := range []bool{false, true} {
		t.Run(fmt.Sprint("tied=", tied), func(t *testing.T) {
			d := improvementDB(t)
			improvementAccount(t, d, "a")
			add := func(id, hash, created string, amount float64) {
				improvementExec(t, d, `INSERT INTO statement_imports(id,account_id,filename,file_hash,statement_format,parser_used) VALUES(?,'a',?,?,'PDF','fixture')`, id, id, hash)
				improvementExec(t, d, `INSERT INTO credit_card_bills(id,account_id,statement_import_id,statement_date,payment_due_date,total_due_amount,created_at) VALUES(?,'a',?,'2026-01-01','2026-01-20',?,?)`, id, id, amount, created)
			}
			createdB := "2026-01-03"
			if tied {
				createdB = "2026-01-02"
			}
			add("original", "hash-a", "2026-01-02", 100)
			add("amendment", "hash-b", createdB, 110)
			add("repeat-old", "hash-a", "2026-01-04", 100)
			overview, err := d.GetAnalyticsOverview()
			if err != nil || overview.TotalCreditDue != 110 {
				t.Fatal("old source displaced amendment", overview, err)
			}
			raw, _ := d.ListCreditCardBills("")
			if len(raw) != 3 {
				t.Fatal("source ownership/history removed")
			}
		})
	}
}
