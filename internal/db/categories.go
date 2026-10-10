package db

import (
	"database/sql"
	"fmt"
	"strings"

	"local-finance/internal/models"
)

// UpdateCategory edits custom category presentation without changing its identity,
// parent, or the transactions and rules that reference it. System defaults are protected.
func (d *DB) UpdateCategory(c *models.Category, fields []string) (*models.Category, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	values := map[string]any{"name": c.Name, "color_hex": c.ColorHex, "icon": c.Icon}
	var sets []string
	var args []any
	for _, field := range fields {
		if field == "id" {
			continue
		}
		value, ok := values[field]
		if !ok {
			return nil, fmt.Errorf("unsupported category field")
		}
		sets = append(sets, field+" = ?")
		args = append(args, value)
	}
	if len(sets) == 0 {
		return nil, fmt.Errorf("no category fields supplied")
	}
	args = append(args, c.ID)
	result, err := d.conn.Exec("UPDATE categories SET "+strings.Join(sets, ", ")+" WHERE id = ? AND is_system = 0", args...)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}
	var saved models.Category
	err = d.conn.QueryRow("SELECT id, name, parent_id, color_hex, icon, is_system FROM categories WHERE id = ?", c.ID).Scan(&saved.ID, &saved.Name, &saved.ParentID, &saved.ColorHex, &saved.Icon, &saved.IsSystem)
	return &saved, err
}
