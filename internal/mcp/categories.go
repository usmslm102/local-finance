package mcp

import (
	"encoding/json"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
	"local-finance/internal/models"
)

func registerCategoryWriteTool(server *sdk.Server, database *db.DB) {
	properties := map[string]any{
		"id":        map[string]any{"type": "string", "maxLength": 2000},
		"name":      map[string]any{"type": "string", "maxLength": 200},
		"color_hex": map[string]any{"type": "string", "pattern": "^#[0-9a-fA-F]{6}$"},
		"icon":      map[string]any{"type": "string", "minLength": 1, "maxLength": 100},
	}
	registerWriteTool(server, "save_category", "Create a custom category (omit id) or update a custom category (provide id from list_categories). Name is required; omitted color_hex and icon preserve values on updates, or default to #64748B and tag on creation. System categories cannot be edited. Category identity and parent are preserved. No transactions are recategorized. Creation retries can create duplicates; use the returned id to update.", properties, []string{"name"}, func(raw json.RawMessage) (any, error) {
		var input struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &input)
		var values map[string]json.RawMessage
		_ = json.Unmarshal(raw, &values)
		category := models.Category{ColorHex: "#64748B", Icon: "tag"}
		if _, updating := values["id"]; updating {
			if strings.TrimSpace(input.ID) == "" {
				return nil, errors.New("id cannot be empty")
			}
			categories, err := database.ListCategories()
			if err != nil {
				return nil, errors.New("unable to load categories")
			}
			found := false
			for _, existing := range categories {
				if existing.ID == input.ID {
					if existing.IsSystem {
						return nil, errors.New("system categories cannot be edited")
					}
					category, found = existing, true
					break
				}
			}
			if !found {
				return nil, errors.New("unable to load category; check id")
			}
		}
		if err := json.Unmarshal(raw, &category); err != nil {
			return nil, errors.New("invalid category arguments")
		}
		category.Name = strings.TrimSpace(category.Name)
		if category.Name == "" || strings.TrimSpace(category.Icon) == "" {
			return nil, errors.New("name and icon cannot be empty")
		}
		var err error
		if category.ID == "" {
			err = database.CreateCategory(&category)
		} else {
			err = database.UpdateCategory(&category)
		}
		if err != nil {
			return nil, errors.New("unable to save category")
		}
		return category, nil
	})
}
