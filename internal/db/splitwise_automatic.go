package db

import (
	"database/sql"
	"fmt"
	"local-finance/internal/models"
)

// ImportSplitwiseAutomatically performs matching and accounting in one commit.
// Unmatched cash-backed entries are dropped; paid-by-other shares need no cash link.
func (d *DB) ImportSplitwiseAutomatically(entries []models.SplitwiseEntry) (models.SplitwiseImportResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return models.SplitwiseImportResult{}, err
	}
	defer tx.Rollback()
	inserted, err := importSplitwise(tx, entries)
	if err != nil {
		return models.SplitwiseImportResult{}, err
	}
	result, err := reconcileSplitwise(tx)
	if err != nil {
		return result, err
	}
	result.Inserted = inserted
	result.Duplicates = len(entries) - inserted
	return result, tx.Commit()
}

func (w *StatementWriter) ReconcileSplitwise() error {
	_, err := reconcileSplitwise(w.conn)
	return err
}

func reconcileSplitwise(conn statementConnection) (models.SplitwiseImportResult, error) {
	result := models.SplitwiseImportResult{}
	members, err := listSplitwiseMembers(conn)
	if err != nil {
		return result, err
	}
	entries, err := listSplitwise(conn)
	if err != nil {
		return result, err
	}
	rules, err := (&statementStore{conn: conn}).ListRules()
	if err != nil {
		return result, err
	}
	for _, e := range entries {
		if e.Status != "REVIEW" {
			continue
		}
		paid, _, err := splitwiseMovement(e.Cost, e.Net, e.Share, e.Kind)
		if err != nil {
			return result, err
		}
		if paid == 0 && e.Share == 0 {
			if _, err := conn.Exec(`DELETE FROM splitwise_entries WHERE id=? AND status='REVIEW'`, e.ID); err != nil {
				return result, err
			}
			result.Uninvolved++
			continue
		}
		var link any
		category := "cat_others"
		if matched := models.MatchCategoryRule(models.Transaction{RawNarration: e.Description, CleanedPayee: e.Description, TxType: models.TxTypeDebit}, rules); matched != nil {
			category = *matched
		}
		if paid > 0 {
			matches, err := splitwiseCandidates(conn, e.ID, models.SplitwiseMatchOptions{Share: e.Share, AllowMemberSettlement: e.Kind == "EXPENSE"})
			if err != nil {
				return result, err
			}
			if len(matches) == 0 {
				if _, err := conn.Exec(`DELETE FROM splitwise_entries WHERE id=? AND status='REVIEW'`, e.ID); err != nil {
					return result, err
				}
				result.Skipped++
				continue
			}
			link = matches[0].ID
			if e.Kind == "EXPENSE" {
				// Explicit CSV evidence can replace a generic member classification
				// created by an earlier bank import, while ignored mappings stay protected.
				if _, err := conn.Exec(`DELETE FROM splitwise_entries WHERE id=? AND status='CONFIRMED' AND kind='PAYMENT'`, "sw-settlement-"+matches[0].ID); err != nil {
					return result, err
				}
			}
			result.Matched++
		} else if e.Kind == "EXPENSE" && e.Share > 0 {
			result.PaidByOthers++
		}
		if _, err := conn.Exec(`UPDATE splitwise_entries SET status='CONFIRMED',transaction_id=?,category_id=? WHERE id=?`, link, category, e.ID); err != nil {
			return result, fmt.Errorf("apply Splitwise entry: %w", err)
		}
	}
	// Only outgoing member payments are automatically self transfers. Incoming
	// payments require an explicit Payment export row, so salaries aren't hidden.
	rows, err := conn.Query(`SELECT t.id,t.raw_narration,COALESCE(t.cleaned_payee,''),t.upi_vpa FROM transactions t JOIN accounts a ON a.id=t.account_id
 WHERE t.tx_type='DEBIT' AND t.amount>0 AND a.currency='INR' AND t.is_excluded=0 AND COALESCE(t.transfer_peer_id,'')=''
 AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=t.id OR (s.status='IGNORED' AND s.removed_transaction_id=t.id)) ORDER BY t.tx_date,t.id`)
	if err != nil {
		return result, err
	}
	payments := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.RawNarration, &t.CleanedPayee, &t.UPIVPA); err != nil {
			rows.Close()
			return result, err
		}
		payments = append(payments, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return result, err
	}
	for _, t := range payments {
		for _, m := range members {
			if matchesSplitwiseMember(t, m) {
				if err := recordSplitwiseSettlement(conn, models.SplitwiseSettlementConfirmation{TransactionID: t.ID, Group: m.Group, Member: m.Name}, m); err != nil {
					return result, err
				}
				result.Transfers++
				break
			}
		}
	}
	return result, nil
}

func reapplySplitwiseRules(conn statementConnection, rules []models.CategorizationRule) (int, error) {
	rows, err := conn.Query(`SELECT id,description,category_id FROM splitwise_entries WHERE kind='EXPENSE' AND status='CONFIRMED' AND is_manual_category=0`)
	if err != nil {
		return 0, err
	}
	type item struct {
		id, description string
		category        sql.NullString
	}
	entries := []item{}
	for rows.Next() {
		var e item
		if err := rows.Scan(&e.id, &e.description, &e.category); err != nil {
			rows.Close()
			return 0, err
		}
		entries = append(entries, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, e := range entries {
		category := "cat_others"
		if matched := models.MatchCategoryRule(models.Transaction{RawNarration: e.description, CleanedPayee: e.description, TxType: models.TxTypeDebit}, rules); matched != nil {
			category = *matched
		}
		if e.category.Valid && e.category.String == category {
			continue
		}
		if _, err := conn.Exec(`UPDATE splitwise_entries SET category_id=? WHERE id=?`, category, e.id); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
