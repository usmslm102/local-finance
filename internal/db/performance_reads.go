package db

import (
	"fmt"
	"local-finance/internal/models"
	"strings"
	"time"
)

func budgetMonth(value string, now time.Time) time.Time {
	month, err := time.Parse("2006-01", value)
	if err != nil {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	return month
}

// total_changes covers this connection's committed writes (and harmless extra
// invalidations after rollbacks); data_version covers writes on other connections.
func (d *DB) DataRevision() (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var changes, version int64
	err := d.conn.QueryRow(`SELECT total_changes(),data_version FROM pragma_data_version`).Scan(&changes, &version)
	return fmt.Sprintf("%s:%d:%d", d.revisionID, changes, version), err
}

// readCategoryDetails performs a fixed two queries for cash-flow comparisons.
// Historical windows remain the latest recorded months, independent of the
// chosen reporting period; blank payees still occupy a top-three rank.
func (d *DB) readCategoryDetails(dateClause string, args []interface{}) (map[string][]models.MonthlyCategoryDataPoint, map[string][]models.PayeeSpend, error) {
	history := map[string][]models.MonthlyCategoryDataPoint{}
	payees := map[string][]models.PayeeSpend{}
	rows, err := d.conn.Query(`WITH totals AS (
 SELECT COALESCE(c.id,'uncat') category,substr(t.tx_date,1,7) month,SUM(t.amount) amount
 FROM personal_transactions t LEFT JOIN categories c ON c.id=t.category_id
 WHERE ` + spendingFilter("t") + ` GROUP BY COALESCE(c.id,'uncat'),substr(t.tx_date,1,7)
 ), ranked AS (SELECT *,ROW_NUMBER() OVER(PARTITION BY category ORDER BY month DESC) rank FROM totals)
 SELECT category,month,amount FROM ranked WHERE rank<=6 ORDER BY category,month`)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var point models.MonthlyCategoryDataPoint
		if err := rows.Scan(&id, &point.Month, &point.Amount); err != nil {
			rows.Close()
			return nil, nil, err
		}
		history[id] = append(history[id], point)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	rows, err = d.conn.Query(fmt.Sprintf(`WITH totals AS (
 SELECT COALESCE(c.id,'uncat') category,cleaned_payee,payment_mode,SUM(amount) amount,COUNT(t.id) count
 FROM personal_transactions t LEFT JOIN categories c ON c.id=t.category_id
 WHERE `+spendingFilter("t")+` AND %s GROUP BY COALESCE(c.id,'uncat'),cleaned_payee,payment_mode
 ), ranked AS (SELECT *,ROW_NUMBER() OVER(PARTITION BY category ORDER BY amount DESC,cleaned_payee,payment_mode) rank FROM totals)
 SELECT category,COALESCE(cleaned_payee,''),COALESCE(payment_mode,''),amount,count FROM ranked WHERE rank<=3 ORDER BY category,rank`, dateClause), args...)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var id string
		var item models.PayeeSpend
		if err := rows.Scan(&id, &item.Payee, &item.PaymentMode, &item.TotalSpent, &item.TxCount); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if item.Payee != "" {
			payees[id] = append(payees[id], item)
		}
	}
	err = rows.Err()
	rows.Close()
	return history, payees, err
}

// ExistingTxHashes bounds SQLite binds and memory by the incoming statement,
// rather than loading every fingerprint ever stored for an account.
func (d *DB) ExistingTxHashes(hashes []string) (map[string]bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	found := make(map[string]bool)
	for start := 0; start < len(hashes); start += 500 {
		end := start + 500
		if end > len(hashes) {
			end = len(hashes)
		}
		marks := make([]string, end-start)
		args := make([]any, end-start)
		for i, h := range hashes[start:end] {
			marks[i] = "?"
			args[i] = h
		}
		rows, err := d.conn.Query("SELECT tx_hash FROM transactions WHERE tx_hash IN ("+strings.Join(marks, ",")+")", args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var h string
			if err := rows.Scan(&h); err != nil {
				rows.Close()
				return nil, err
			}
			found[h] = true
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return found, nil
}

// ListSubscriptionCandidates uses the same ledger, order and pre-filter cap as
// ListTransactions, without its count query, display joins or rich row decoding.
func (d *DB) ListSubscriptionCandidates() ([]models.Transaction, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	rows, err := d.conn.Query(`SELECT id,account_id,tx_date,raw_narration,cleaned_payee,amount,category_id,is_transfer,is_excluded
 FROM ledger_transactions WHERE tx_type='DEBIT' ORDER BY tx_date DESC,created_at DESC,id DESC LIMIT 100000`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount, &t.CategoryID, &t.IsTransfer, &t.IsExcluded); err != nil {
			return nil, err
		}
		t.TxType = models.TxTypeDebit
		result = append(result, t)
	}
	return result, rows.Err()
}
