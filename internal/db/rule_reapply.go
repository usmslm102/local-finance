package db

import (
	"database/sql"

	"github.com/google/uuid"
	"local-finance/internal/models"
)

func (d *statementStore) InsertRule(r *models.CategorizationRule) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.TxType == "" {
		r.TxType = "ALL"
	}

	result, err := d.conn.Exec(`
		INSERT INTO categorization_rules (id, priority, match_field, match_type, match_pattern, exclude_pattern, tx_type, target_category_id, assign_tags, is_active)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ? WHERE EXISTS (SELECT 1 FROM categories WHERE id = ?)
	`, r.ID, r.Priority, r.MatchField, r.MatchType, r.MatchPattern, r.ExcludePattern, r.TxType, r.TargetCategoryID, r.AssignTags, r.IsActive, r.TargetCategoryID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	saved, err := d.getRule(r.ID)
	if err == nil {
		*r = *saved
	}
	return err
}

// SaveRuleAndReapply commits the rule and ledger categorization together.
func (d *DB) SaveRuleAndReapply(id *string, patch models.CategorizationRulePatch) (*models.CategorizationRule, int, error) {
	var saved *models.CategorizationRule
	var count int
	err := d.WithStatementImport(func(w *StatementWriter) error {
		var err error
		if id != nil {
			saved, err = w.PatchRule(*id, patch)
		} else {
			rule := models.CategorizationRule{MatchField: "cleaned_payee", MatchType: "CONTAINS", TxType: "ALL", Priority: 50, IsActive: true}
			patch.ApplyTo(&rule)
			if err = rule.ValidatePatterns(); err != nil {
				return err
			}
			err = w.InsertRule(&rule)
			saved = &rule
		}
		if err != nil {
			return err
		}
		count, err = w.reapplyRules(true)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return saved, count, nil
}
func (d *statementStore) reapplyRules(resetUnmatched bool) (int, error) {
	return d.reapplyRulesMatching(resetUnmatched, "")
}

// predicate is internal SQL; callers supply values separately.
func (d *statementStore) reapplyRulesMatching(resetUnmatched bool, predicate string, args ...any) (int, error) {
	rules, err := d.ListRules()
	if err != nil {
		return 0, err
	}

	rows, err := d.conn.Query(`
		SELECT id, raw_narration, COALESCE(cleaned_payee, ''), COALESCE(reference_number, ''), upi_vpa, category_id, tx_type, COALESCE(is_transfer, 0)
		FROM transactions
		WHERE COALESCE(is_manual_category, 0) = 0
	`+predicate, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type txItem struct {
		id         string
		narration  string
		payee      string
		ref        string
		vpa        string
		currentCat *string
		txType     string
		isTransfer bool
	}
	var txs []txItem
	for rows.Next() {
		var item txItem
		var vpa, cat sql.NullString
		if err := rows.Scan(&item.id, &item.narration, &item.payee, &item.ref, &vpa, &cat, &item.txType, &item.isTransfer); err != nil {
			return 0, err
		}
		if vpa.Valid {
			item.vpa = vpa.String
		}
		if cat.Valid {
			item.currentCat = &cat.String
		}
		txs = append(txs, item)
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	updatedCount := 0
	txStmt, err := d.conn.Prepare(`UPDATE transactions SET category_id = ? WHERE id = ?`)
	if err != nil {
		return 0, err
	}
	defer txStmt.Close()

	for _, item := range txs {
		var matchedCatID *string
		if resetUnmatched && item.isTransfer {
			cid := models.CategoryTransfersID
			matchedCatID = &cid
		}
		tx := models.Transaction{RawNarration: item.narration, CleanedPayee: item.payee, ReferenceNumber: item.ref, UPIVPA: &item.vpa, TxType: models.TxType(item.txType)}
		for _, r := range rules {
			if matchedCatID != nil {
				break
			}
			if r.MatchesTransaction(tx) {
				cid := r.TargetCategoryID
				matchedCatID = &cid
				break
			}
		}

		if matchedCatID != nil {
			if item.currentCat == nil || *item.currentCat != *matchedCatID {
				if _, err := txStmt.Exec(*matchedCatID, item.id); err != nil {
					return 0, err
				}
				updatedCount++
			}
		} else {
			// If no rule matches, but transaction was previously categorized as "cat_salary":
			// Salary & Income is strictly an income category (CREDIT).
			// If it's a DEBIT, or if it matched an exclusion keyword (e.g. cook, maid, staff),
			// or if no active salary rule matches it, reset it to "cat_others".
			if resetUnmatched || (item.currentCat != nil && *item.currentCat == "cat_salary") {
				catOthers := "cat_others"
				if item.currentCat == nil || *item.currentCat != catOthers {
					if _, err := txStmt.Exec(catOthers, item.id); err != nil {
						return 0, err
					}
					updatedCount++
				}
			}
		}
	}

	return updatedCount, nil
}
