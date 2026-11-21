package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"testing"
)

func TestOverviewAndMerchantEvidenceRemainFreshThroughMCP(t *testing.T) {
	manager, database := testManager(t)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := database.Exec(query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO accounts(id,bank_name,account_type,account_number_mask,currency) VALUES('fixture','Fictional','CREDIT_CARD','XX1234','INR')`)
	for i := 0; i < 87; i++ {
		id := fmt.Sprintf("evidence-%03d", i)
		exec(`INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,tx_type,amount,category_id,payment_mode,reference_number) VALUES(?,'fixture',?,'2026-01-10','Merchant','Merchant','DEBIT',1,'cat_food','UPI','')`, id, id)
	}
	for _, id := range []string{"version-a", "version-b"} {
		exec(`INSERT INTO credit_card_bills(id,account_id,statement_date,payment_due_date,total_due_amount,created_at) VALUES(?,'fixture','2026-01-01','2026-01-20',100,'2026-01-02')`, id)
	}
	session := connect(t, manager, enable(t, manager))
	call := func(name string, args map[string]any) map[string]any {
		t.Helper()
		result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
		if err != nil || result.IsError {
			t.Fatalf("%s: %v %+v", name, err, result)
		}
		bytes, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err := json.Unmarshal(bytes, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	check := func(amount float64) {
		t.Helper()
		out := call("get_overview", map[string]any{"start_date": "2026-01-01", "end_date": "2026-01-31"})
		data := out["data"].(map[string]any)
		if data["total_expense"] != amount || data["total_credit_due"] != float64(100) {
			t.Fatalf("stale/wrong MCP overview %+v", data)
		}
		app, err := database.GetAnalyticsOverviewRange("2026-01-01", "2026-01-31")
		if err != nil || app.TotalExpense != amount || app.TotalCreditDue != 100 {
			t.Fatal("app/MCP disagreement", err)
		}
	}
	check(87)
	exec(`INSERT INTO transactions(id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,tx_type,amount,category_id) VALUES('new','fixture','new','2026-01-11','Other','Other','DEBIT',3,'cat_food')`)
	check(90)
	for page, want := range map[int]int{1: 50, 2: 37} {
		out := call("get_merchant_profile", map[string]any{"name": "Merchant", "page": page, "page_size": 50})
		data := out["data"].(map[string]any)
		if len(data["recent_transactions"].([]any)) != want {
			t.Fatalf("history lost to double pagination: %+v", out)
		}
		meta := out["pagination"].(map[string]any)["/recent_transactions"].(map[string]any)
		if meta["total"] != float64(87) || meta["has_more"] != (page == 1) {
			t.Fatal("incorrect history metadata", meta)
		}
	}
	for _, input := range []struct{ page, size, want int }{{1, 1, 1}, {2, 1, 1}, {1, 200, 87}, {100, 1, 0}} {
		out := call("get_merchant_profile", map[string]any{"name": "Merchant", "page": input.page, "page_size": input.size})
		data := out["data"].(map[string]any)
		if len(data["recent_transactions"].([]any)) != input.want {
			t.Fatalf("page compatibility %+v: %+v", input, out)
		}
		meta := out["pagination"].(map[string]any)["/recent_transactions"].(map[string]any)
		if meta["page"] != float64(input.page) || meta["page_size"] != float64(input.size) {
			t.Fatal("wrong page metadata", meta)
		}
	}
}
