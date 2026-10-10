package db

import (
	"database/sql"
	"fmt"
	"strings"

	"local-finance/internal/models"
)

// GetRule includes inactive rules so callers can edit or reactivate them.
func (d *DB) GetRule(id string) (*models.CategorizationRule, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getRule(id)
}

func (d *DB) getRule(id string) (*models.CategorizationRule, error) {
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

// PatchRule writes only supplied fields, so concurrent disjoint patches preserve
// each other. Rule/category existence and the returned value share the writer lock.
func (d *DB) PatchRule(r *models.CategorizationRule, fields []string) (*models.CategorizationRule, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	current, err := d.getRule(r.ID)
	if err != nil {
		return nil, err
	}
	values := map[string]any{
		"priority": r.Priority, "match_field": r.MatchField, "match_type": r.MatchType,
		"match_pattern": r.MatchPattern, "exclude_pattern": r.ExcludePattern, "tx_type": r.TxType,
		"target_category_id": r.TargetCategoryID, "assign_tags": r.AssignTags, "is_active": r.IsActive,
	}
	var sets []string
	var args []any
	for _, field := range fields {
		if field == "id" {
			continue
		}
		value, ok := values[field]
		if !ok {
			return nil, fmt.Errorf("unsupported rule field")
		}
		// These fields form the matching invariant; merge them with the current
		// saved rule before validation, rather than validating a stale snapshot.
		switch field {
		case "match_type":
			current.MatchType = r.MatchType
		case "match_pattern":
			current.MatchPattern = r.MatchPattern
		case "exclude_pattern":
			current.ExcludePattern = r.ExcludePattern
		case "target_category_id":
			current.TargetCategoryID = r.TargetCategoryID
		}
		sets = append(sets, field+" = ?")
		args = append(args, value)
	}
	if len(sets) == 0 {
		return nil, fmt.Errorf("no rule fields supplied")
	}
	if err := current.ValidatePatterns(); err != nil {
		return nil, err
	}
	args = append(args, r.ID, current.TargetCategoryID)
	result, err := d.conn.Exec("UPDATE categorization_rules SET "+strings.Join(sets, ", ")+" WHERE id = ? AND EXISTS (SELECT 1 FROM categories WHERE id = ?)", args...)
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
	return d.getRule(r.ID)
}
