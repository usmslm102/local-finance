package db

import (
	"local-finance/internal/models"
	"local-finance/internal/parser"
	"strings"
	"testing"
)

func TestSplitwiseRuleSaveReappliesAtomically(t *testing.T) {
	d := safetyDB(t)
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Fictional taxi,Travel,100,INR,-50,50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	pattern, category := "Fictional taxi", "cat_travel"
	patch := models.CategorizationRulePatch{MatchPattern: &pattern, TargetCategoryID: &category}
	if _, err := d.conn.Exec(`CREATE TRIGGER reject_splitwise_rule_update BEFORE UPDATE OF category_id ON splitwise_entries BEGIN SELECT RAISE(ABORT,'fictional rule failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.SaveRuleAndReapply(nil, patch); err == nil {
		t.Fatal("expected Splitwise update to reject the atomic rule save")
	}
	var count int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM categorization_rules WHERE match_pattern=?`, pattern).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rule save survived rollback: %d %v", count, err)
	}
	if _, err := d.conn.Exec(`DROP TRIGGER reject_splitwise_rule_update`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := d.SaveRuleAndReapply(nil, patch); err != nil {
		t.Fatal(err)
	}
	tx, err := d.GetTransaction(entries[0].ID)
	if err != nil || tx.CategoryID == nil || *tx.CategoryID != category {
		t.Fatalf("Splitwise rule not applied: %+v %v", tx, err)
	}
}
