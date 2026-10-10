package db

import (
	"database/sql"

	"local-finance/internal/models"
)

// GetRule includes inactive rules so callers can edit or reactivate them.
func (d *DB) GetRule(id string) (*models.CategorizationRule, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return (&statementStore{conn: d.conn}).getRule(id)
}

func (d *statementStore) getRule(id string) (*models.CategorizationRule, error) {
	var r models.CategorizationRule
	err := d.conn.QueryRow(`
		SELECT r.id, r.priority, r.match_field, r.match_type, r.match_pattern,
		       COALESCE(r.exclude_pattern, ''), COALESCE(r.tx_type, 'ALL'),
		       r.target_category_id, COALESCE(c.name, ''), COALESCE(r.assign_tags, ''), r.is_active
		FROM categorization_rules r LEFT JOIN categories c ON r.target_category_id = c.id
		WHERE r.id = ?
	`, id).Scan(&r.ID, &r.Priority, &r.MatchField, &r.MatchType, &r.MatchPattern,
		&r.ExcludePattern, &r.TxType, &r.TargetCategoryID, &r.TargetCategory, &r.AssignTags, &r.IsActive)
	return &r, err
}

// PatchRule merges supplied fields into the current rule under the writer lock.
// Validation, rule/category existence and the result share the same lock.
func (d *DB) PatchRule(id string, patch models.CategorizationRulePatch) (*models.CategorizationRule, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return (&statementStore{conn: d.conn}).PatchRule(id, patch)
}

func (d *statementStore) PatchRule(id string, patch models.CategorizationRulePatch) (*models.CategorizationRule, error) {
	current, err := d.getRule(id)
	if err != nil {
		return nil, err
	}
	patch.ApplyTo(current)
	if err := current.ValidatePatterns(); err != nil {
		return nil, err
	}
	result, err := d.conn.Exec(`
		UPDATE categorization_rules SET priority = ?, match_field = ?, match_type = ?,
		match_pattern = ?, exclude_pattern = ?, tx_type = ?, target_category_id = ?, assign_tags = ?, is_active = ?
		WHERE id = ? AND EXISTS (SELECT 1 FROM categories WHERE id = ?)
	`, current.Priority, current.MatchField, current.MatchType, current.MatchPattern,
		current.ExcludePattern, current.TxType, current.TargetCategoryID, current.AssignTags, current.IsActive, id, current.TargetCategoryID)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, sql.ErrNoRows
	}
	return d.getRule(id)
}
