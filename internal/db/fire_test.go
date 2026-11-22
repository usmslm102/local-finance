package db

import (
	"testing"
	"time"

	"local-finance/internal/models"
)

func TestFIREBaselineUsesPersonalSharesAndExcludesInvestmentFunding(t *testing.T) {
	d := safetyDB(t)
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := d.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO accounts(id,bank_name,account_type,currency) VALUES('inr','Fixture','SAVINGS','INR'),('usd','Fixture USD','SAVINGS','USD')`)
	exec(`INSERT INTO categories(id,name,parent_id,color_hex,icon,is_system) VALUES('invest-child','Other investing','cat_invest','#123456','Bank',0)`)
	insert := func(id, date, kind, category, label string, amount float64) {
		exec(`INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,tx_type,amount,category_id) VALUES(?,'inr',?,?,?,?,?,?,?)`, id, id, date, label, label, kind, amount, category)
	}
	insert("shared", "2026-10-10", "DEBIT", "cat_food", "Shop", 100)
	exec(`INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,transaction_id) VALUES('sw-fixture','Fixture','Alex','2026-10-10','Shop','Food',10000,4000,6000,'EXPENSE','CONFIRMED','shared')`)
	insert("salary", "2026-10-10", "CREDIT", "cat_salary", "Salary", 1000)
	insert("payroll", "2026-10-11", "CREDIT", "cat_others", "PAYROLL", 200)
	insert("broker", "2026-10-10", "DEBIT", "cat_others", "ZERODHA", 200)
	insert("us-broker", "2026-09-10", "DEBIT", "cat_invest", "INDMONEY", 100)
	insert("fd", "2026-09-10", "DEBIT", "cat_invest", "FIXED DEPOSIT", 75)
	insert("other", "2026-10-10", "DEBIT", "invest-child", "Unclassified investing", 40)
	insert("excluded", "2026-10-10", "DEBIT", "cat_food", "Excluded", 9999)
	exec(`UPDATE transactions SET is_excluded=1 WHERE id='excluded'`)
	insert("transfer", "2026-10-10", "DEBIT", "cat_transfers", "Self transfer", 888)
	insert("current", "2026-11-10", "DEBIT", "cat_food", "Current partial month", 900)
	exec(`INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,tx_type,amount) VALUES('foreign','usd','foreign','2026-08-01','Non-INR spend','DEBIT',999)`)
	before, _ := d.DataRevision()
	b, err := d.GetFIREBaseline(time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if b.ObservedMonths != 2 || len(b.Months) != 6 || b.MonthlyExpense != 30 || b.MonthlySalary != 600 || b.MonthlyIndia != 100 || b.MonthlyUSINR != 50 || b.MonthlyDeposits != 37.5 || b.MonthlyOther != 20 {
		t.Fatalf("baseline: %+v", b)
	}
	if b.Months[0].Month != "2026-05" || b.Months[0].Observed || b.Months[5].Expense != 60 {
		t.Fatalf("calendar or personal share wrong: %+v", b.Months)
	}
	after, _ := d.DataRevision()
	if before != after {
		t.Fatal("baseline modified ledger")
	}
}

func TestFIREBaselineLatestSnapshotsKeepCurrenciesSeparate(t *testing.T) {
	d := safetyDB(t)
	save := func(id, ref, date, currency string, value *float64) {
		t.Helper()
		_, _, err := d.SaveInvestmentSnapshot(&models.InvestmentSnapshot{ID: id, Provider: "Fixture", AccountRef: ref, AsOf: date, Currency: currency, CurrentValue: value, ImportedAt: date}, id)
		if err != nil {
			t.Fatal(err)
		}
	}
	old, newer, second, usd := 500.0, 1000.0, 200.0, 10.0
	save("old", "A", "2026-01-01", "INR", &old)
	save("new", "A", "2026-10-01", "INR", &newer)
	save("other", "B", "2026-10-01", "INR", &second)
	save("us", "A", "2026-10-01", "USD", &usd)
	save("missing", "C", "2026-10-01", "INR", nil)
	b, err := d.GetFIREBaseline(time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if b.IndiaBalance != 1200 || b.USDBalance != 10 || len(b.Portfolios) != 4 || b.ObservedMonths != 0 || b.MonthlyExpense != 0 {
		t.Fatalf("snapshot grouping: %+v", b)
	}
}
