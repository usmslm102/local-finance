package db

import "local-finance/internal/models"

// GetRule includes inactive rules so callers can edit or reactivate them.
func (d *DB) GetRule(id string) (*models.CategorizationRule, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
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
