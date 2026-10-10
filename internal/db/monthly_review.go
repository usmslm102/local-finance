package db

import (
	"database/sql"
	"errors"
	"math"
	"sort"
	"time"

	"local-finance/internal/models"
)

var ErrInvalidReviewPeriod = errors.New("choose a month in YYYY-MM format that is not in the future")

// Keep review totals and their evidence on exactly the same definition of spending.
// Credits (including refunds) remain separate until explicit refund linking exists.
var reviewSpending = spendingFilter("t")

func reviewPeriods(month string, now time.Time) (models.ReviewPeriod, models.ReviewPeriod, bool, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil || start.Year() < 2 || month > now.Format("2006-01") {
		return models.ReviewPeriod{}, models.ReviewPeriod{}, false, ErrInvalidReviewPeriod
	}
	end := start.AddDate(0, 1, -1)
	previous := start.AddDate(0, -1, 0)
	previousEnd := start.AddDate(0, 0, -1)
	partial := month == now.Format("2006-01")
	if partial {
		end = start.AddDate(0, 0, now.Day()-1)
		if now.Day() < previousEnd.Day() {
			previousEnd = previous.AddDate(0, 0, now.Day()-1)
		}
	}
	return models.ReviewPeriod{Start: start.Format(time.DateOnly), End: end.Format(time.DateOnly)},
		models.ReviewPeriod{Start: previous.Format(time.DateOnly), End: previousEnd.Format(time.DateOnly)}, partial, nil
}

type reviewAccumulator struct {
	cents int64
	count int
}

func (a reviewAccumulator) spend() models.ReviewSpend {
	return models.ReviewSpend{Amount: float64(a.cents) / 100, Count: a.count}
}

type reviewPair struct {
	current, previous reviewAccumulator
}

func (p reviewPair) delta() float64 { return float64(p.current.cents-p.previous.cents) / 100 }

type reviewCategoryAccumulator struct {
	name string
	reviewPair
	merchants map[string]*reviewPair
}

// GetMonthlyReview reads a consistent SQLite snapshot without changing ledger data.
// Coverage describes the union of reported statement ranges, not a balance audit.
func (d *DB) GetMonthlyReview(month string, now time.Time) (*models.MonthlyReview, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	months := []string{}
	rows, err := tx.Query(`SELECT DISTINCT month FROM (
		SELECT substr(tx_date, 1, 7) AS month FROM personal_transactions
		UNION SELECT substr(end_date, 1, 7) FROM statement_imports
	) WHERE month <= ? ORDER BY month DESC`, now.Format("2006-01"))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			rows.Close()
			return nil, err
		}
		if _, _, _, err := reviewPeriods(m, now); err == nil {
			months = append(months, m)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if month == "" {
		month = now.Format("2006-01")
		if len(months) > 0 {
			month = months[0]
		}
	}
	current, previous, partial, err := reviewPeriods(month, now)
	if err != nil {
		return nil, err
	}
	start, _ := time.Parse(time.DateOnly, current.Start)
	result := &models.MonthlyReview{
		Month: month, NextMonth: start.AddDate(0, 1, 0).Format("2006-01"), AvailableMonths: months,
		CurrentPeriod: current, PreviousPeriod: previous, IsPartialMonth: partial,
		Coverage: []models.ReviewCoverage{}, Categories: []models.ReviewCategory{},
	}
	result.Coverage, err = readReviewCoverage(tx, current, previous)
	if err != nil {
		return nil, err
	}
	result.CoverageComplete = len(result.Coverage) > 0
	for _, c := range result.Coverage {
		if c.CurrentDays != c.CurrentTotal || c.PreviousDays != c.PreviousTotal {
			result.CoverageComplete = false
		}
	}

	rows, err = tx.Query(`SELECT t.tx_date, COALESCE(t.category_id, ''), COALESCE(c.name, 'Uncategorized'),
		COALESCE(NULLIF(TRIM(t.cleaned_payee), ''), 'Unknown payee'), t.amount
		FROM personal_transactions t LEFT JOIN categories c ON c.id = t.category_id
		WHERE `+reviewSpending+` AND ((t.tx_date BETWEEN ? AND ?) OR (t.tx_date BETWEEN ? AND ?))`,
		current.Start, current.End, previous.Start, previous.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := map[string]*reviewCategoryAccumulator{}
	var total reviewPair
	for rows.Next() {
		var date, categoryID, name, payee string
		var amount float64
		if err := rows.Scan(&date, &categoryID, &name, &payee, &amount); err != nil {
			return nil, err
		}
		category := categories[categoryID]
		if category == nil {
			category = &reviewCategoryAccumulator{name: name, merchants: map[string]*reviewPair{}}
			categories[categoryID] = category
		}
		if category.merchants[payee] == nil {
			category.merchants[payee] = &reviewPair{}
		}
		for _, pair := range []*reviewPair{&total, &category.reviewPair, category.merchants[payee]} {
			bucket := &pair.previous
			if date >= current.Start {
				bucket = &pair.current
			}
			bucket.cents += int64(math.Round(amount * 100))
			bucket.count++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.Current, result.Previous, result.Delta = total.current.spend(), total.previous.spend(), total.delta()
	for id, category := range categories {
		item := models.ReviewCategory{ID: id, Name: category.name, Current: category.current.spend(),
			Previous: category.previous.spend(), Delta: category.delta(), Merchants: []models.ReviewMerchant{}}
		for name, merchant := range category.merchants {
			item.Merchants = append(item.Merchants, models.ReviewMerchant{Name: name,
				Current: merchant.current.spend(), Previous: merchant.previous.spend(), Delta: merchant.delta()})
		}
		sort.Slice(item.Merchants, func(i, j int) bool {
			if math.Abs(item.Merchants[i].Delta) == math.Abs(item.Merchants[j].Delta) {
				return item.Merchants[i].Name < item.Merchants[j].Name
			}
			return math.Abs(item.Merchants[i].Delta) > math.Abs(item.Merchants[j].Delta)
		})
		if len(item.Merchants) > 3 {
			item.Merchants = item.Merchants[:3]
		}
		result.Categories = append(result.Categories, item)
	}
	sort.Slice(result.Categories, func(i, j int) bool {
		a, b := result.Categories[i], result.Categories[j]
		if math.Abs(a.Delta) == math.Abs(b.Delta) {
			if a.Name == b.Name {
				return a.ID < b.ID
			}
			return a.Name < b.Name
		}
		return math.Abs(a.Delta) > math.Abs(b.Delta)
	})
	return result, nil
}

func readReviewCoverage(tx *sql.Tx, current, previous models.ReviewPeriod) ([]models.ReviewCoverage, error) {
	rows, err := tx.Query(`SELECT a.id, COALESCE(NULLIF(a.nickname, ''), a.bank_name, 'Splitwise') ||
		CASE WHEN COALESCE(a.account_number_mask, '') = '' THEN '' ELSE ' · ' || a.account_number_mask END,
		COALESCE(s.start_date, ''), COALESCE(s.end_date, '')
		FROM accounts a LEFT JOIN statement_imports s ON s.account_id = a.id ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	coverage := []models.ReviewCoverage{}
	var currentDates, previousDates map[string]bool
	for rows.Next() {
		var id, name, from, to string
		if err := rows.Scan(&id, &name, &from, &to); err != nil {
			return nil, err
		}
		if len(coverage) == 0 || coverage[len(coverage)-1].AccountID != id {
			coverage = append(coverage, models.ReviewCoverage{AccountID: id, Name: name,
				CurrentTotal: reviewDayCount(current), PreviousTotal: reviewDayCount(previous)})
			currentDates, previousDates = map[string]bool{}, map[string]bool{}
		}
		item := &coverage[len(coverage)-1]
		_, fromErr := time.Parse(time.DateOnly, from)
		_, toErr := time.Parse(time.DateOnly, to)
		if fromErr != nil || toErr != nil || from > to {
			continue
		}
		if to > item.LatestEnd {
			item.LatestEnd = to
		}
		markReviewDays(currentDates, current, from, to)
		markReviewDays(previousDates, previous, from, to)
		item.CurrentDays, item.PreviousDays = len(currentDates), len(previousDates)
	}
	return coverage, rows.Err()
}

func reviewDayCount(period models.ReviewPeriod) int {
	start, _ := time.Parse(time.DateOnly, period.Start)
	end, _ := time.Parse(time.DateOnly, period.End)
	return int(end.Sub(start).Hours()/24) + 1
}

func markReviewDays(days map[string]bool, period models.ReviewPeriod, from, to string) {
	start, _ := time.Parse(time.DateOnly, period.Start)
	for day := start; day.Format(time.DateOnly) <= period.End; day = day.AddDate(0, 0, 1) {
		date := day.Format(time.DateOnly)
		if date >= from && date <= to {
			days[date] = true
		}
	}
}

// GetMonthlyReviewEvidence paginates the exact spending rows behind one category.
// An empty category ID means uncategorized, rather than all categories.
func (d *DB) GetMonthlyReviewEvidence(month, category string, previous bool, page int, now time.Time) (*models.ReviewEvidence, error) {
	currentPeriod, previousPeriod, _, err := reviewPeriods(month, now)
	if err != nil {
		return nil, err
	}
	period := currentPeriod
	if previous {
		period = previousPeriod
	}
	if page < 1 || page > 1000000 {
		return nil, errors.New("page must be between 1 and 1000000")
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result := &models.ReviewEvidence{Items: []models.ReviewTransaction{}, Page: page, PageSize: 50, Period: period}
	where := reviewSpending + ` AND t.tx_date BETWEEN ? AND ? AND COALESCE(t.category_id, '') = ?`
	if err := tx.QueryRow(`SELECT COUNT(*) FROM personal_transactions t WHERE `+where,
		period.Start, period.End, category).Scan(&result.Total); err != nil {
		return nil, err
	}
	rows, err := tx.Query(`SELECT t.id, t.tx_date, COALESCE(NULLIF(TRIM(t.cleaned_payee), ''), 'Unknown payee'),
		COALESCE(NULLIF(a.nickname, ''), a.bank_name, 'Splitwise') || ' · ' || COALESCE(a.account_number_mask, ''), t.amount
		FROM personal_transactions t LEFT JOIN accounts a ON a.id = t.account_id WHERE `+where+`
		ORDER BY t.tx_date DESC, t.id LIMIT ? OFFSET ?`, period.Start, period.End, category, result.PageSize, (page-1)*result.PageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.ReviewTransaction
		if err := rows.Scan(&item.ID, &item.Date, &item.Payee, &item.Account, &item.Amount); err != nil {
			return nil, err
		}
		result.Items = append(result.Items, item)
	}
	return result, rows.Err()
}
