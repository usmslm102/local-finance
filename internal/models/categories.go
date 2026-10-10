package models

import (
	"errors"
	"regexp"
	"strings"
)

// CategoryTransfersID is the stable ID of the user-renamable transfer category.
const CategoryTransfersID string = "cat_transfers"

// CategoryPatch edits presentation without changing identity or parent.
type CategoryPatch struct {
	Name     *string `json:"name"`
	ColorHex *string `json:"color_hex"`
	Icon     *string `json:"icon"`
}

func (p CategoryPatch) ApplyTo(c *Category) {
	if p.Name != nil {
		c.Name = strings.TrimSpace(*p.Name)
	}
	if p.ColorHex != nil {
		c.ColorHex = *p.ColorHex
	}
	if p.Icon != nil {
		c.Icon = *p.Icon
	}
}

var categoryColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (c *Category) ValidateDefinition() error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Icon) == "" {
		return errors.New("name and icon cannot be empty")
	}
	if !categoryColorPattern.MatchString(c.ColorHex) {
		return errors.New("color_hex must be #RRGGBB")
	}
	return nil
}
