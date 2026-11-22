package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"local-finance/internal/retirement"
)

func TestFIREMCPReadOnlyParityAndPaging(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	session := connect(t, m, token)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tool := range tools.Tools {
		if tool.Name == "get_fire_baseline" || tool.Name == "calculate_fire" {
			seen[tool.Name] = tool.Annotations.ReadOnlyHint
		}
	}
	if !seen["get_fire_baseline"] || !seen["calculate_fire"] {
		t.Fatal("FIRE tools missing read-only hints")
	}
	baseline := callWrite(t, session, "get_fire_baseline", map[string]any{}, false)
	if baseline.StructuredContent == nil {
		t.Fatal("baseline unavailable")
	}
	scenario := retirement.Scenario{Age: 30, Horizon: 90, MonthlyExpense: 100000, RetirementExpense: 100000, Inflation: 6, WithdrawalRate: 3, RSUVestYears: 4, RSUNetPercent: 70, Assets: []retirement.Asset{{Kind: "MARKET", Name: "Fixture", Currency: "INR", Balance: 50000000, Return: 8, AccessiblePercent: 100}}}
	before, _ := database.DataRevision()
	raw, _ := json.Marshal(scenario)
	args := map[string]any{}
	_ = json.Unmarshal(raw, &args)
	args["page_size"] = 1
	result := callWrite(t, session, "calculate_fire", args, false)
	var response struct {
		Data       retirement.Result     `json:"data"`
		Pagination map[string]pagination `json:"pagination"`
	}
	data, _ := json.Marshal(result.StructuredContent)
	_ = json.Unmarshal(data, &response)
	expected, err := retirement.Project(scenario)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Data.RetirementAge, expected.RetirementAge) || !reflect.DeepEqual(response.Data.AtRetirement, expected.AtRetirement) || len(response.Data.Points) != 1 || response.Pagination["/points"].Total != len(expected.Points) || !response.Pagination["/points"].HasMore {
		t.Fatalf("calculator parity/paging: %+v", response)
	}
	args["page"] = 2
	next := decodeWrite[retirement.Result](t, callWrite(t, session, "calculate_fire", args, false))
	if len(next.Points) != 1 || next.Points[0].Age != expected.Points[1].Age {
		t.Fatal("next page changed or duplicated trajectory")
	}
	args["age"] = json.Number("30.0")
	callWrite(t, session, "calculate_fire", args, true)
	after, _ := database.DataRevision()
	if before != after {
		t.Fatal("MCP calculation mutated data")
	}
}
