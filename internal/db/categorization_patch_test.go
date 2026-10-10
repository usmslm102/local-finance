package db

import (
	"path/filepath"
	"sync"
	"testing"

	"local-finance/internal/models"
)

func TestConcurrentCategorizationPatchesPreserveOmittedFields(t *testing.T) {
	database, err := NewDB(filepath.Join(t.TempDir(), "finance.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	rule := models.CategorizationRule{Priority: 50, MatchField: "cleaned_payee", MatchType: "CONTAINS", MatchPattern: "Fictional Shop", TargetCategoryID: "cat_groceries", IsActive: true}
	if err := database.CreateRule(&rule); err != nil {
		t.Fatal(err)
	}
	// Both clients loaded the same rule before either submitted a disjoint patch.
	disable, priority := rule, rule
	disable.IsActive = false
	priority.Priority = 0
	category := models.Category{Name: "Fictional", ColorHex: "#123456", Icon: "tag"}
	if err := database.CreateCategory(&category); err != nil {
		t.Fatal(err)
	}
	color, icon := category, category
	color.ColorHex = "#abcdef"
	icon.Icon = "shopping-cart"
	start := make(chan struct{})
	errors := make(chan error, 4)
	var group sync.WaitGroup
	for _, save := range []func() error{
		func() error { _, err := database.PatchRule(&disable, []string{"is_active"}); return err },
		func() error { _, err := database.PatchRule(&priority, []string{"priority"}); return err },
		func() error { _, err := database.UpdateCategory(&color, []string{"color_hex"}); return err },
		func() error { _, err := database.UpdateCategory(&icon, []string{"icon"}); return err },
	} {
		group.Add(1)
		go func() { defer group.Done(); <-start; errors <- save() }()
	}
	close(start)
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	saved, err := database.GetRule(rule.ID)
	if err != nil || saved.IsActive || saved.Priority != 0 {
		t.Fatalf("lost concurrent rule patch: %+v %v", saved, err)
	}
	categories, err := database.ListCategories()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, saved := range categories {
		if saved.ID == category.ID {
			found = true
			if saved.ColorHex != "#abcdef" || saved.Icon != "shopping-cart" {
				t.Fatalf("lost concurrent category patch: %+v", saved)
			}
		}
	}
	if !found {
		t.Fatal("category missing")
	}
}

func TestRulePatchValidatesCurrentMatcher(t *testing.T) {
	database, err := NewDB(filepath.Join(t.TempDir(), "finance.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	rule := models.CategorizationRule{MatchField: "cleaned_payee", MatchType: "CONTAINS", MatchPattern: "Shop", TargetCategoryID: "cat_groceries", IsActive: true}
	if err := database.CreateRule(&rule); err != nil {
		t.Fatal(err)
	}
	stale := rule
	stale.MatchPattern = "[" // Valid literal for the matcher this client loaded.
	rule.MatchType = "REGEX"
	rule.MatchPattern = "^Shop$"
	if _, err := database.PatchRule(&rule, []string{"match_type", "match_pattern"}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.PatchRule(&stale, []string{"match_pattern"}); err == nil {
		t.Fatal("stale client saved invalid regex")
	}
	saved, err := database.GetRule(rule.ID)
	if err != nil || saved.MatchType != "REGEX" || saved.MatchPattern != "^Shop$" {
		t.Fatalf("rejected patch changed rule: %+v %v", saved, err)
	}
}
