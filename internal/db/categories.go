package db

import (
	"database/sql"

	"local-finance/internal/models"
)

// UpdateCategory edits custom category presentation without changing its identity,
// parent, or the transactions and rules that reference it. System defaults are protected.
func (d *DB) UpdateCategory(c *models.Category) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	result, err := d.conn.Exec(`UPDATE categories SET name = ?, color_hex = ?, icon = ? WHERE id = ? AND is_system = 0`, c.Name, c.ColorHex, c.Icon, c.ID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	return err
}
