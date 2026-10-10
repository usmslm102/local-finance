package db

import (
	"errors"

	"local-finance/internal/models"
)

// UpdateCategory merges presentation fields into the current custom category
// under the writer lock. Identity, parent and ledger/rule references are preserved.
func (d *DB) UpdateCategory(id string, patch models.CategoryPatch) (*models.Category, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var saved models.Category
	err := d.conn.QueryRow("SELECT id, name, parent_id, color_hex, icon, is_system FROM categories WHERE id = ?", id).Scan(&saved.ID, &saved.Name, &saved.ParentID, &saved.ColorHex, &saved.Icon, &saved.IsSystem)
	if err != nil {
		return nil, err
	}
	if saved.IsSystem {
		return nil, errors.New("system categories cannot be edited")
	}
	patch.ApplyTo(&saved)
	if err := saved.ValidateDefinition(); err != nil {
		return nil, err
	}
	_, err = d.conn.Exec("UPDATE categories SET name = ?, color_hex = ?, icon = ? WHERE id = ? AND is_system = 0", saved.Name, saved.ColorHex, saved.Icon, saved.ID)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}
