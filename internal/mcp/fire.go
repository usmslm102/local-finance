package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
	"local-finance/internal/retirement"
)

func registerFIRETools(server *sdk.Server, database *db.DB) {
	registerTool(server, toolSpec{name: "get_fire_baseline", description: "Get six complete months of recorded personal living costs, salary evidence and inferred investment funding plus latest distinct INR/USD portfolio values for retirement planning. Funding is not a valuation or CAGR. Missing months and valuation dates remain explicit. Age, RSUs, ESPP, PF/NPS and returns need user-supplied assumptions. Reuses the app FIRE baseline; does not save a scenario or alter the ledger.", query: func(arguments) (any, error) { return database.GetFIREBaseline(time.Now()) }})
	inferred, err := jsonschema.For[retirement.Scenario](nil)
	if err != nil {
		panic(err)
	}
	encoded, err := json.Marshal(inferred)
	if err != nil {
		panic(err)
	}
	var schema map[string]any
	if err = json.Unmarshal(encoded, &schema); err != nil {
		panic(err)
	}
	// The engine owns numeric/business validation; bound the record collection at
	// the protocol boundary as well. All scenario fields are explicit, including zero.
	properties := schema["properties"].(map[string]any)
	assets := properties["assets"].(map[string]any)
	assets["minItems"], assets["maxItems"] = 1, 12
	registerPreviewToolSchema(server, "calculate_fire", "Calculate the app's deterministic retirement scenario without saving inputs or changing financial data. First get_fire_baseline; confirm all assumptions with the user. All scenario fields are required, including zero-valued optional benefits. Asset balances use currency INR/USD; monthly and monthly_increase are always INR. Returns, inflation and growth are annual percentages. Kinds: MARKET, OTHER, EMPLOYER, PF, NPS. PF/NPS access_age and accessible_percent are planning assumptions, not legal eligibility. USD assets or grants require explicit usd_inr. Monthly RSU vesting goes to one USD EMPLOYER bucket after rsu_net_percent retention; unvested awards do not count as starting wealth. All contributions and employment vesting stop at the calculated retirement month. Each candidate must meet the initial withdrawal target with accessible assets and fund spending to horizon. No probability-of-success or live-price claim. Points are paginated; retirement_age and at_retirement describe the complete calculation. Use the same confirmed inputs for every page.", schema, func(ctx context.Context, raw json.RawMessage) (any, error) {
		var input retirement.Scenario
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, errors.New("invalid scenario argument types; ages and periods require integer encodings")
		}
		return retirement.ProjectContext(ctx, input)
	})
}
