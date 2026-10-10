package db

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"local-finance/internal/models"
)

var ErrSplitwiseNotFound = errors.New("Splitwise entry not found")

const splitwiseAmountTolerance int64 = 500 // Paise: the user permits discrepancies up to INR 5.

func splitwiseMovement(cost, net, share int64, kind string) (int64, string, error) {
	if share < 0 || share > cost {
		return 0, "", fmt.Errorf("share must be between zero and total cost")
	}
	paid, direction := net+share, "DEBIT"
	if kind == "PAYMENT" {
		if share != 0 {
			return 0, "", fmt.Errorf("settlements have no expense share")
		}
		paid = net
		if paid < 0 {
			paid = -paid
			direction = "CREDIT"
		}
	}
	if paid < 0 || paid > cost {
		return 0, "", fmt.Errorf("paid amount must be between zero and total cost")
	}
	return paid, direction, nil
}

func (d *DB) ImportSplitwise(entries []models.SplitwiseEntry) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	inserted := 0
	for _, e := range entries {
		result, err := tx.Exec(`INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, e.ID, e.Group, e.Person, e.Date, e.Description, e.Category, e.Cost, e.Net, e.Share, e.Kind, e.Status)
		if err != nil {
			return 0, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		inserted += int(count)
	}
	return inserted, tx.Commit()
}

func (d *DB) ListSplitwise() ([]models.SplitwiseEntry, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listSplitwise()
}

func (d *DB) listSplitwise() ([]models.SplitwiseEntry, error) {
	rows, err := d.conn.Query(`SELECT id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,transaction_id,category_id FROM splitwise_entries ORDER BY tx_date DESC,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []models.SplitwiseEntry{}
	for rows.Next() {
		var e models.SplitwiseEntry
		if err := rows.Scan(&e.ID, &e.Group, &e.Person, &e.Date, &e.Description, &e.Category, &e.Cost, &e.Net, &e.Share, &e.Kind, &e.Status, &e.TransactionID, &e.CategoryID); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ConfirmSplitwise validates and links in one SQLite transaction. A statement
// movement can be consumed once. No statement fields or account balances change.
func (d *DB) ConfirmSplitwise(id string, req models.SplitwiseConfirmation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var cost, net int64
	var kind, date string
	err = tx.QueryRow(`SELECT cost_cents,net_cents,kind,tx_date FROM splitwise_entries WHERE id=?`, id).Scan(&cost, &net, &kind, &date)
	if err == sql.ErrNoRows {
		return ErrSplitwiseNotFound
	}
	if err != nil {
		return err
	}
	status := "CONFIRMED"
	var link *string
	if req.Ignore {
		status = "IGNORED"
	} else {
		paid, direction, err := splitwiseMovement(cost, net, req.Share, kind)
		if err != nil {
			return err
		}
		if paid > 0 && req.TransactionID == "" {
			return fmt.Errorf("select the matching statement transaction for the amount you paid or received")
		}
		if paid == 0 && req.TransactionID != "" {
			return fmt.Errorf("no statement link is needed when you paid nothing")
		}
		if req.TransactionID != "" {
			var amount float64
			var txType string
			var eligible bool
			err = tx.QueryRow(`SELECT t.amount,t.tx_type,(COALESCE(t.transfer_peer_id,'')='' AND t.is_excluded=0 AND a.currency='INR' AND t.tx_date BETWEEN date(?,'-15 days') AND ?) FROM transactions t JOIN accounts a ON a.id=t.account_id WHERE t.id=?`, date, date, req.TransactionID).Scan(&amount, &txType, &eligible)
			if err == sql.ErrNoRows {
				return fmt.Errorf("statement transaction not found")
			}
			if err != nil {
				return err
			}
			if !eligible || txType != direction || amount <= 0 || math.Abs(math.Round(amount*100)-float64(paid)) > float64(splitwiseAmountTolerance) {
				return fmt.Errorf("statement must match INR direction and amount within ₹5, date from 15 days before through the entry date, and cannot already be excluded or paired as an internal transfer")
			}
			var count int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM splitwise_entries WHERE transaction_id=? AND id!=?`, req.TransactionID, id).Scan(&count); err != nil {
				return err
			}
			if count > 0 {
				return fmt.Errorf("statement transaction is already linked to a Splitwise entry")
			}
			link = &req.TransactionID
		}
	}
	if req.CategoryID != "" {
		var exists bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM categories WHERE id=? AND id!=?)`, req.CategoryID, models.CategoryTransfersID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("choose an expense category")
		}
	}
	_, err = tx.Exec(`UPDATE splitwise_entries SET status=?,share_cents=?,transaction_id=?,category_id=NULLIF(?,'') WHERE id=?`, status, req.Share, link, req.CategoryID, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) ResetSplitwise(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	result, err := d.conn.Exec(`UPDATE splitwise_entries SET status='REVIEW',transaction_id=NULL WHERE id=?`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrSplitwiseNotFound
	}
	return nil
}

// Candidate IDs are suggestions only: equal amounts and nearby dates never
// authorize an automatic link. Confirmation rechecks all constraints atomically.
func (d *DB) SplitwiseCandidates(id string, options models.SplitwiseMatchOptions) ([]models.Transaction, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var cost, net int64
	var kind, date string
	err := d.conn.QueryRow(`SELECT cost_cents,net_cents,kind,tx_date FROM splitwise_entries WHERE id=?`, id).Scan(&cost, &net, &kind, &date)
	if err == sql.ErrNoRows {
		return nil, ErrSplitwiseNotFound
	}
	if err != nil {
		return nil, err
	}
	paid, direction, err := splitwiseMovement(cost, net, options.Share, kind)
	if err != nil {
		return nil, err
	}
	items := []models.Transaction{}
	if paid == 0 {
		return items, nil
	}
	search := "%" + strings.TrimSpace(options.Search) + "%"
	rows, err := d.conn.Query(`SELECT t.id,t.tx_date,t.raw_narration,t.amount,t.tx_type,COALESCE(NULLIF(a.nickname,''),a.bank_name),t.is_transfer FROM transactions t JOIN accounts a ON a.id=t.account_id WHERE a.currency='INR' AND t.tx_type=? AND t.amount>0 AND ABS(ROUND(t.amount*100)-?)<=? AND t.tx_date BETWEEN date(?,'-15 days') AND ? AND COALESCE(t.transfer_peer_id,'')='' AND t.is_excluded=0 AND (t.raw_narration LIKE ? OR t.cleaned_payee LIKE ?) AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=t.id AND s.id!=?) ORDER BY CASE WHEN ROUND(t.amount*100)=? THEN 0 ELSE 1 END,ABS(julianday(t.tx_date)-julianday(?)),ABS(ROUND(t.amount*100)-?),t.id LIMIT 50`, direction, paid, splitwiseAmountTolerance, date, date, search, search, id, paid, date, paid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.TxDate, &t.RawNarration, &t.Amount, &t.TxType, &t.AccountName, &t.IsTransfer); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
