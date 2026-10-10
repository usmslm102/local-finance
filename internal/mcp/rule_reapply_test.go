package mcp

import (
	"testing"

	"local-finance/internal/models"
)

func TestMCPRuleSavesReapplyWholeLedgerAndRollback(t *testing.T) {
	m, database := testManager(t)
	for _, query := range []string{
		`INSERT INTO accounts (id,bank_name,account_type) VALUES ('a','Fictional A','SAVINGS'),('b','Fictional B','SAVINGS')`,
		`INSERT INTO transactions (id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,payment_mode,reference_number,tx_type,amount,category_id,notes,tags,is_manual_category,is_transfer) VALUES
  ('auto-a','a','ha','2026-01-10','QuasarShop','QuasarShop','OTHER','','DEBIT',10,'cat_others','keep note','keep tag',0,0),
  ('auto-b','b','hb','2026-01-10','QuasarShop','QuasarShop','OTHER',NULL,'CREDIT',10,'cat_others','','',0,0),
  ('manual','b','hm','2026-01-10','QuasarShop','QuasarShop','OTHER','','DEBIT',10,'cat_groceries','manual note','manual tag',1,0),
  ('transfer','a','ht','2026-01-10','QuasarShop','QuasarShop','OTHER','','DEBIT',10,'cat_transfers','','',0,1)`,
	} {
		if _, err := database.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	token := enable(t, m)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, &allowed, nil); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	args := map[string]any{"match_pattern": "QuasarShop", "target_category_id": "cat_entertainment", "priority": 10000, "tx_type": "DEBIT"}
	saved := decodeWrite[struct {
		models.CategorizationRule
		Count int `json:"ledger_updated_count"`
	}](t, callWrite(t, session, "save_categorization_rule", args, false))
	if saved.Count != 1 {
		t.Fatalf("create updated count: %+v", saved)
	}
	check := func(id, category string) {
		t.Helper()
		tx, err := database.GetTransaction(id)
		if err != nil || tx.CategoryID == nil || *tx.CategoryID != category {
			t.Fatalf("%s category: %+v %v", id, tx, err)
		}
	}
	if _, err := database.Exec(`UPDATE transactions SET reference_number = '' WHERE id = 'auto-b'`); err != nil {
		t.Fatal(err)
	}
	check("auto-a", "cat_entertainment")
	check("auto-b", "cat_others")
	check("manual", "cat_groceries")
	check("transfer", "cat_transfers")
	args["id"] = saved.ID
	args["tx_type"] = "ALL"
	callWrite(t, session, "save_categorization_rule", args, false)
	check("auto-b", "cat_entertainment")
	// An error partway through recategorization must also roll back the rule patch.
	if _, err := database.Exec(`CREATE TRIGGER reject_reapply BEFORE UPDATE OF category_id ON transactions WHEN NEW.category_id = 'cat_food' BEGIN SELECT RAISE(ABORT,'fictional failure'); END`); err != nil {
		t.Fatal(err)
	}
	args["target_category_id"] = "cat_food"
	callWrite(t, session, "save_categorization_rule", args, true)
	rule, err := database.GetRule(saved.ID)
	if err != nil || rule.TargetCategoryID != "cat_entertainment" {
		t.Fatalf("rule partially saved: %+v %v", rule, err)
	}
	check("auto-a", "cat_entertainment")
	check("auto-b", "cat_entertainment")
	args["target_category_id"] = "cat_entertainment"
	args["is_active"] = false
	disabled := decodeWrite[struct {
		models.CategorizationRule
		Count int `json:"ledger_updated_count"`
	}](t, callWrite(t, session, "save_categorization_rule", args, false))
	if disabled.Count != 2 {
		t.Fatalf("disable updated count: %+v", disabled)
	}
	check("auto-a", "cat_others")
	check("auto-b", "cat_others")
	check("manual", "cat_groceries")
	check("transfer", "cat_transfers")
	tx, err := database.GetTransaction("auto-a")
	if err != nil || tx.Notes != "keep note" || tx.Tags != "keep tag" {
		t.Fatalf("edits lost: %+v %v", tx, err)
	}
}
