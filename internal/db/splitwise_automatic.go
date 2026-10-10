package db

import (
	"database/sql"
	"fmt"
	"local-finance/internal/models"
	"strings"
	"unicode/utf8"
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
			options := models.SplitwiseMatchOptions{Share: e.Share}
			if e.Kind == "PAYMENT" {
				member, err := paymentMember(e, members)
				if err != nil {
					if _, err := conn.Exec(`DELETE FROM splitwise_entries WHERE id=? AND status='REVIEW'`, e.ID); err != nil {
						return result, err
					}
					result.Skipped++
					continue
				}
				options.PaymentMember = &member
			}
			matches, err := splitwiseCandidates(conn, e.ID, options)
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
			result.Matched++
			if e.Kind == "PAYMENT" {
				result.Transfers++
			}
		} else if e.Kind == "EXPENSE" && e.Share > 0 {
			result.PaidByOthers++
		}
		if _, err := conn.Exec(`UPDATE splitwise_entries SET status='CONFIRMED',transaction_id=?,category_id=? WHERE id=?`, link, category, e.ID); err != nil {
			return result, fmt.Errorf("apply Splitwise entry: %w", err)
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

// CSV balances identify the other participant; older exports can be recovered
// from an unambiguous complete member name in their Payment description.
func paymentMember(e models.SplitwiseEntry, members []models.SplitwiseMember) (models.SplitwiseMember, error) {
	if strings.HasPrefix(e.ID, "sw-settlement-") {
		return models.SplitwiseMember{}, fmt.Errorf("legacy name-only settlement is not CSV payment evidence")
	}
	var found *models.SplitwiseMember
	for _, m := range members {
		if m.Group != e.Group || m.Person != e.Person {
			continue
		}
		matches := m.Name == e.Counterparty
		if e.Counterparty == "" {
			matches = matchesSplitwiseMember(models.Transaction{RawNarration: e.Description}, models.SplitwiseMember{Name: m.Name})
			// Older exports abbreviate participant names ("Ravi m." for
			// "Ravi Makwana"). Require both the first name and last initial;
			// the ambiguity check below still rejects multiple possible members.
			parts := strings.Fields(settlementName(m.Name))
			if !matches && len(parts) >= 2 {
				initial, _ := utf8.DecodeRuneInString(parts[1])
				matches = matchesSplitwiseMember(models.Transaction{RawNarration: e.Description}, models.SplitwiseMember{Name: parts[0] + " " + string(initial)})
			}
		}
		if matches {
			if found != nil {
				return models.SplitwiseMember{}, fmt.Errorf("ambiguous payment member")
			}
			copy := m
			found = &copy
		}
	}
	if found == nil {
		return models.SplitwiseMember{}, fmt.Errorf("payment member not found")
	}
	return *found, nil
}
