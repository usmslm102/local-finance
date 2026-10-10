package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/models"
)

func callWrite(t *testing.T, session *sdk.ClientSession, name string, args map[string]any, wantError bool) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil || result.IsError != wantError {
		t.Fatalf("%s: err=%v result=%+v", name, err, result)
	}
	if len(result.Content) == 0 {
		t.Fatal("missing text fallback")
	}
	return result
}

func decodeWrite[T any](t *testing.T, result *sdk.CallToolResult) T {
	t.Helper()
	var out struct {
		Data T `json:"data"`
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out.Data
}

func TestCategorizationWritesAndPermissionChanges(t *testing.T) {
	m, database := testManager(t)
	if _, err := database.Exec(`INSERT INTO accounts (id,bank_name,account_type,account_number_mask) VALUES ('a','Fictional Bank','SAVINGS','XXXX1234')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO transactions (id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,tx_type,amount,category_id,notes,tags,is_manual_category,payment_mode,reference_number) VALUES ('tx','a','hash','2026-01-10','SHOP','Shop','DEBIT',100,'cat_groceries','manual note','manual',1,'UPI','fictional-ref')`); err != nil {
		t.Fatal(err)
	}
	before, err := database.GetTransaction("tx")
	if err != nil {
		t.Fatal(err)
	}
	token := enable(t, m)
	read := connect(t, m, token)
	for _, name := range []string{"save_category", "save_categorization_rule"} {
		if _, err := read.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: map[string]any{}}); err == nil {
			t.Fatalf("read-only connection exposed %s", name)
		}
	}
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 25 {
		t.Fatalf("tool discovery: %+v %v", tools, err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "save_category" || tool.Name == "save_categorization_rule" {
			if tool.Annotations.ReadOnlyHint || tool.Annotations.IdempotentHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint != (tool.Name == "save_categorization_rule") || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
				t.Fatalf("write annotations: %+v", tool.Annotations)
			}
		} else if !tool.Annotations.ReadOnlyHint {
			t.Fatalf("read annotation changed: %s", tool.Name)
		}
	}
	info := decodeWrite[map[string]any](t, callWrite(t, session, "get_app_info", map[string]any{}, false))
	if info["access"] != "read-and-categorization-write" {
		t.Fatalf("app info: %+v", info)
	}
	category := decodeWrite[models.Category](t, callWrite(t, session, "save_category", map[string]any{"name": " Fictional Shopping "}, false))
	if category.ID == "" || category.Name != "Fictional Shopping" || category.ColorHex != "#64748B" || category.Icon != "tag" || category.IsSystem {
		t.Fatalf("category defaults: %+v", category)
	}
	category = decodeWrite[models.Category](t, callWrite(t, session, "save_category", map[string]any{"id": category.ID, "name": "Shopping", "color_hex": "#123abc"}, false))
	if category.ColorHex != "#123abc" || category.Icon != "tag" {
		t.Fatalf("category update: %+v", category)
	}
	rule := decodeWrite[models.CategorizationRule](t, callWrite(t, session, "save_categorization_rule", map[string]any{"match_pattern": "Shop", "target_category_id": category.ID, "tx_type": "DEBIT", "assign_tags": "shopping", "exclude_pattern": "REFUND"}, false))
	if rule.ID == "" || rule.MatchField != "cleaned_payee" || rule.MatchType != "CONTAINS" || rule.Priority != 50 || !rule.IsActive || rule.TargetCategory != "Shopping" {
		t.Fatalf("rule defaults: %+v", rule)
	}
	zeroPriority := decodeWrite[models.CategorizationRule](t, callWrite(t, session, "save_categorization_rule", map[string]any{"match_pattern": "Fallback", "target_category_id": category.ID, "priority": 0}, false))
	if zeroPriority.Priority != 0 {
		t.Fatalf("explicit creation priority lost: %+v", zeroPriority)
	}
	storedZero, err := database.GetRule(zeroPriority.ID)
	if err != nil || storedZero.Priority != 0 {
		t.Fatalf("persisted creation priority: %+v %v", storedZero, err)
	}
	rule = decodeWrite[models.CategorizationRule](t, callWrite(t, session, "save_categorization_rule", map[string]any{"id": rule.ID, "match_pattern": "Shop", "target_category_id": category.ID, "is_active": false, "priority": 0}, false))
	if rule.IsActive || rule.Priority != 0 || rule.TxType != "DEBIT" || rule.AssignTags != "shopping" || rule.ExcludePattern != "REFUND" {
		t.Fatalf("rule patch: %+v", rule)
	}
	stored, err := database.GetRule(rule.ID)
	if err != nil || !reflect.DeepEqual(*stored, rule) {
		t.Fatalf("persisted rule: %+v %v", stored, err)
	}
	rule = decodeWrite[models.CategorizationRule](t, callWrite(t, session, "save_categorization_rule", map[string]any{"id": rule.ID, "match_pattern": "Shop", "target_category_id": category.ID, "is_active": true}, false))
	if !rule.IsActive || rule.Priority != 0 {
		t.Fatalf("reactivation: %+v", rule)
	}
	// Renaming a category keeps the rule's reference intact.
	callWrite(t, session, "save_category", map[string]any{"id": category.ID, "name": "Shopping renamed"}, false)
	stored, err = database.GetRule(rule.ID)
	if err != nil || stored.TargetCategoryID != category.ID || stored.TargetCategory != "Shopping renamed" {
		t.Fatalf("category references: %+v %v", stored, err)
	}
	for _, name := range []string{"reapply_rules", "delete_category", "delete_rule", "update_transaction"} {
		if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: map[string]any{}}); err == nil {
			t.Fatalf("unexpected tool: %s", name)
		}
	}
	after, err := database.GetTransaction("tx")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("ledger changed: before=%+v after=%+v err=%v", before, after, err)
	}
	m.Close()
	if err := m.Start(); err != nil || !m.Status().AllowCategorizationWrites {
		t.Fatalf("lost persisted permission: %v", err)
	}
	allowed = false
	if _, err := m.ConfigureAccess(true, m.Status().Port, &allowed); err != nil {
		t.Fatal(err)
	}
	// The same authenticated session must lose writes, even if its tool list is cached.
	for _, name := range []string{"save_category", "save_categorization_rule"} {
		if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: map[string]any{"name": "Unauthorized"}}); err == nil {
			t.Fatalf("revoked permission accepted %s", name)
		}
	}
	read = connect(t, m, token)
	tools, err = read.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 23 {
		t.Fatalf("read-only discovery: %+v %v", tools, err)
	}
	callWrite(t, read, "list_categories", map[string]any{}, false)
}

func TestInvalidCategorizationWritesDoNotMutateData(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	before, err := database.ListCategories()
	if err != nil {
		t.Fatal(err)
	}
	rulesBefore, err := database.ListRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range []map[string]any{
		{}, {"name": " "}, {"name": nil}, {"name": "X", "is_system": true},
		{"name": "X", "id": ""}, {"name": "X", "id": "missing"}, {"name": "X", "id": "cat_groceries"},
		{"name": "X", "color_hex": "red"}, {"name": "X", "icon": " "}, {"name": strings.Repeat("x", 201)},
		{"name": "X", "parent_id": "missing"},
	} {
		callWrite(t, session, "save_category", args, true)
	}
	for _, patch := range []map[string]any{
		{"match_pattern": " "}, {"target_category_id": "missing"}, {"target_category_id": " "},
		{"id": "missing"}, {"id": ""}, {"match_type": "INVALID"}, {"match_field": "INVALID"},
		{"tx_type": "INVALID"}, {"priority": 1.5}, {"priority": 2147483648}, {"is_active": nil},
		{"match_type": "REGEX", "match_pattern": "["}, {"exclude_pattern": "(*"},
		{"match_pattern": strings.Repeat("x", 2001)}, {"sql": "DELETE FROM transactions"},
	} {
		args := map[string]any{"match_pattern": "Shop", "target_category_id": "cat_groceries"}
		for key, value := range patch {
			args[key] = value
		}
		callWrite(t, session, "save_categorization_rule", args, true)
	}
	callWrite(t, session, "save_categorization_rule", map[string]any{}, true)
	after, err := database.ListCategories()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("invalid writes changed categories")
	}
	rulesAfter, err := database.ListRules()
	if err != nil || !reflect.DeepEqual(rulesBefore, rulesAfter) {
		t.Fatal("invalid writes changed rules")
	}
}
