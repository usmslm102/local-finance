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
	disabled, priority := false, 0
	category := models.Category{Name: "Fictional", ColorHex: "#123456", Icon: "tag"}
	if err := database.CreateCategory(&category); err != nil {
		t.Fatal(err)
	}
	color, icon := "#abcdef", "shopping-cart"
	start := make(chan struct{})
	errors := make(chan error, 4)
	var group sync.WaitGroup
	for _, save := range []func() error{
		func() error {
			_, err := database.PatchRule(rule.ID, models.CategorizationRulePatch{IsActive: &disabled})
			return err
		},
		func() error {
			_, err := database.PatchRule(rule.ID, models.CategorizationRulePatch{Priority: &priority})
			return err
		},
		func() error {
			_, err := database.UpdateCategory(category.ID, models.CategoryPatch{ColorHex: &color})
			return err
		},
		func() error {
			_, err := database.UpdateCategory(category.ID, models.CategoryPatch{Icon: &icon})
			return err
		},
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
	literal := "[" // Valid literal for the matcher this client saw.
	matcher, pattern := "REGEX", "^Shop$"
	if _, err := database.PatchRule(rule.ID, models.CategorizationRulePatch{MatchType: &matcher, MatchPattern: &pattern}); err != nil {
		t.Fatal(err)
	}
	if _, err := database.PatchRule(rule.ID, models.CategorizationRulePatch{MatchPattern: &literal}); err == nil {
		t.Fatal("stale client saved invalid regex")
	}
	saved, err := database.GetRule(rule.ID)
	if err != nil || saved.MatchType != "REGEX" || saved.MatchPattern != "^Shop$" {
		t.Fatalf("rejected patch changed rule: %+v %v", saved, err)
	}
}
