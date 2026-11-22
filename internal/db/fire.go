package db

import (
	"encoding/json"
	"strings"
	"time"

	"local-finance/internal/models"
)

type FIREMonth struct {
	Month    string  `json:"month"`
	Observed bool    `json:"observed"`
	Expense  float64 `json:"expense"`
	Salary   float64 `json:"salary"`
	India    float64 `json:"india"`
	US       float64 `json:"us"`
	Deposits float64 `json:"deposits"`
	Other    float64 `json:"other"`
}
type FIREPortfolio struct {
	Provider string   `json:"provider"`
	AsOf     string   `json:"as_of"`
	Currency string   `json:"currency"`
	Value    *float64 `json:"value"`
}
type FIREBaseline struct {
	Months          []FIREMonth     `json:"months"`
	ObservedMonths  int             `json:"observed_months"`
	MonthlyExpense  float64         `json:"monthly_expense"`
	MonthlySalary   float64         `json:"monthly_salary"`
	MonthlyIndia    float64         `json:"monthly_india"`
	MonthlyUSINR    float64         `json:"monthly_us_inr"`
	MonthlyDeposits float64         `json:"monthly_deposits"`
	MonthlyOther    float64         `json:"monthly_other"`
	IndiaBalance    float64         `json:"india_balance"`
	USDBalance      float64         `json:"usd_balance"`
	Portfolios      []FIREPortfolio `json:"portfolios"`
	Warnings        []string        `json:"warnings"`
}

// Six complete calendar months; never divide an incomplete month by a guessed
// fraction. Missing months stay explicit and are not fabricated as zero spending.
func (d *DB) GetFIREBaseline(now time.Time) (FIREBaseline, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	end := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	start := end.AddDate(0, -6, 0)
	out := FIREBaseline{Months: []FIREMonth{}, Portfolios: []FIREPortfolio{}, Warnings: []string{"Estimates use six complete calendar months of recorded INR personal activity. Observed months are not proof of complete statement coverage; missing activity can understate spending. Review Data health.", "Broker/deposit funding is inferred from destination labels or the investment category. It is gross outgoing funding, not net of withdrawals, a valuation, historical CAGR, or proof that money was invested. Unclassified investment funding is separate.", "Only the latest imported snapshot per provider/account/currency supplies a balance. Accounts are never exposed in this baseline. Snapshot dates can differ; reconcile manual balances and avoid adding the same holdings twice.", "Age, PF/NPS, ESPP, employer stock and unvested grants cannot reliably be inferred from bank statements. Enter them explicitly; bank salary is normally after payroll deductions."}}
	indices := map[string]int{}
	for i := 0; i < 6; i++ {
		month := start.AddDate(0, i, 0).Format("2006-01")
		indices[month] = i
		out.Months = append(out.Months, FIREMonth{Month: month})
	}
	rows, err := d.conn.Query(`WITH RECURSIVE investment_categories(id) AS (SELECT id FROM categories WHERE id='cat_invest' UNION SELECT c.id FROM categories c JOIN investment_categories p ON c.parent_id=p.id)
 SELECT strftime('%Y-%m',t.tx_date),t.tx_type,t.amount,COALESCE(t.category_id,''),UPPER(COALESCE(t.cleaned_payee,'')||' '||COALESCE(t.raw_narration,'')||' '||COALESCE(c.name,'')),COALESCE(t.category_id IN (SELECT id FROM investment_categories),0),COALESCE(`+salaryEvidenceFilter("t")+`,0)
 FROM personal_transactions t LEFT JOIN accounts a ON a.id=t.account_id LEFT JOIN categories c ON c.id=t.category_id
 WHERE t.tx_date>=? AND t.tx_date<? AND COALESCE(a.currency,'INR')='INR' AND `+financialActivityFilter("t"), start.Format(time.DateOnly), end.Format(time.DateOnly))
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var month, direction, category, label string
		var amount float64
		var investment, salary bool
		if err = rows.Scan(&month, &direction, &amount, &category, &label, &investment, &salary); err != nil {
			rows.Close()
			return out, err
		}
		idx, ok := indices[month]
		if !ok {
			continue
		}
		m := &out.Months[idx]
		m.Observed = true
		if direction == "CREDIT" && salary {
			m.Salary += amount
			continue
		}
		if direction != "DEBIT" || amount <= 0 {
			continue
		}
		switch {
		case strings.Contains(label, "ZERODHA") || strings.Contains(label, "GROWW"):
			m.India += amount
		case strings.Contains(label, "INDMONEY"):
			m.US += amount
		case strings.Contains(label, "FIXED DEPOSIT"):
			m.Deposits += amount
		case investment:
			m.Other += amount
		default:
			m.Expense += amount
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	for _, m := range out.Months {
		if m.Observed {
			out.ObservedMonths++
			out.MonthlyExpense += m.Expense
			out.MonthlySalary += m.Salary
			out.MonthlyIndia += m.India
			out.MonthlyUSINR += m.US
			out.MonthlyDeposits += m.Deposits
			out.MonthlyOther += m.Other
		}
	}
	if out.ObservedMonths > 0 {
		n := float64(out.ObservedMonths)
		out.MonthlyExpense /= n
		out.MonthlySalary /= n
		out.MonthlyIndia /= n
		out.MonthlyUSINR /= n
		out.MonthlyDeposits /= n
		out.MonthlyOther /= n
	}
	snapshots, err := d.conn.Query("SELECT data_json, portfolio_key FROM (" + investmentSnapshotsWithPortfolio + ") ORDER BY as_of DESC, imported_at DESC, id DESC")
	if err != nil {
		return out, err
	}
	defer snapshots.Close()
	seen := map[string]bool{}
	for snapshots.Next() {
		var data, key string
		if err = snapshots.Scan(&data, &key); err != nil {
			return out, err
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		var snapshot models.InvestmentSnapshot
		if err = json.Unmarshal([]byte(data), &snapshot); err != nil {
			return out, err
		}
		out.Portfolios = append(out.Portfolios, FIREPortfolio{snapshot.Provider, snapshot.AsOf, snapshot.Currency, snapshot.CurrentValue})
		if snapshot.CurrentValue == nil {
			continue
		}
		switch snapshot.Currency {
		case "INR":
			out.IndiaBalance += *snapshot.CurrentValue
		case "USD":
			out.USDBalance += *snapshot.CurrentValue
		default:
			out.Warnings = append(out.Warnings, "A non-INR/non-USD portfolio was omitted; add its converted value explicitly.")
		}
	}
	return out, snapshots.Err()
}
