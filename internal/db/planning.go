package db

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"local-finance/internal/models"
)

var ErrInvalidGoal = errors.New("enter a name of 1–100 characters, valid amounts up to ₹1 trillion with at most two decimals, and a target date between 1900 and 2200")

func validateGoal(goal *models.SavingsGoal) error {
	goal.Name = strings.TrimSpace(goal.Name)
	date, err := time.Parse(time.DateOnly, goal.TargetDate)
	if err != nil || date.Year() < 1900 || date.Year() > 2200 || goal.Name == "" || utf8.RuneCountInString(goal.Name) > 100 {
		return ErrInvalidGoal
	}
	for _, amount := range []float64{goal.TargetAmount, goal.SavedAmount, goal.MonthlyContribution} {
		if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < 0 || amount > 1e12 {
			return ErrInvalidGoal
		}
		if _, fraction, hasFraction := strings.Cut(strconv.FormatFloat(amount, 'f', -1, 64), "."); hasFraction && len(fraction) > 2 {
			return ErrInvalidGoal
		}
	}
	if goal.TargetAmount <= 0 {
		return ErrInvalidGoal
	}
	return nil
}

func (d *DB) ListSavingsGoals() ([]models.SavingsGoal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listSavingsGoals()
}
func (d *DB) listSavingsGoals() ([]models.SavingsGoal, error) {
	rows, err := d.conn.Query(`SELECT id,name,target_amount,saved_amount,monthly_contribution,target_date,created_at,updated_at FROM savings_goals ORDER BY target_date,created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	goals := []models.SavingsGoal{}
	for rows.Next() {
		var g models.SavingsGoal
		if err := rows.Scan(&g.ID, &g.Name, &g.TargetAmount, &g.SavedAmount, &g.MonthlyContribution, &g.TargetDate, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

func (d *DB) SaveSavingsGoal(goal *models.SavingsGoal, id string) (bool, error) {
	if err := validateGoal(goal); err != nil {
		return false, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	goal.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if id == "" {
		goal.ID, goal.CreatedAt = uuid.NewString(), goal.UpdatedAt
		_, err := d.conn.Exec(`INSERT INTO savings_goals(id,name,target_amount,saved_amount,monthly_contribution,target_date,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, goal.ID, goal.Name, goal.TargetAmount, goal.SavedAmount, goal.MonthlyContribution, goal.TargetDate, goal.CreatedAt, goal.UpdatedAt)
		return err == nil, err
	}
	result, err := d.conn.Exec(`UPDATE savings_goals SET name=?,target_amount=?,saved_amount=?,monthly_contribution=?,target_date=?,updated_at=? WHERE id=?`, goal.Name, goal.TargetAmount, goal.SavedAmount, goal.MonthlyContribution, goal.TargetDate, goal.UpdatedAt, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}
func (d *DB) DeleteSavingsGoal(id string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	result, err := d.conn.Exec(`DELETE FROM savings_goals WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// A small aggregate keeps data-health scans independent of ledger size.
func (d *DB) GetDataHealthAccounts() ([]models.DataHealthAccount, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.conn.Query(`SELECT a.id,COALESCE(MAX(t.tx_date),''),COUNT(t.id),
 COALESCE(SUM(CASE WHEN t.id IS NOT NULL AND (COALESCE(t.category_id,'') IN ('','cat_others')) AND ` + financialActivityFilter("t") + ` THEN 1 ELSE 0 END),0)
 FROM accounts a LEFT JOIN ledger_transactions t ON t.account_id=a.id GROUP BY a.id ORDER BY a.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.DataHealthAccount{}
	for rows.Next() {
		var a models.DataHealthAccount
		if err := rows.Scan(&a.AccountID, &a.LatestTransaction, &a.TransactionCount, &a.ReviewCount); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
