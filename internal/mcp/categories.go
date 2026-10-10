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
			ID *string `json:"id"`
			models.CategoryPatch
		}
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, errors.New("invalid category arguments")
		}
		if input.ID != nil {
			if strings.TrimSpace(*input.ID) == "" {
				return nil, errors.New("id cannot be empty")
			}
			saved, err := database.UpdateCategory(*input.ID, input.CategoryPatch)
			if err != nil {
				return nil, errors.New("unable to update custom category; check id and fields")
			}
			return saved, nil
		}
		category := models.Category{ColorHex: "#64748B", Icon: "tag"}
		input.CategoryPatch.ApplyTo(&category)
		if err := category.ValidateDefinition(); err != nil {
			return nil, err
		}
		if err := database.CreateCategory(&category); err != nil {
			return nil, errors.New("unable to save category")
		}
		return category, nil
	})
}
