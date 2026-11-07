package parser

// LocalFinanceCSVRowSchema describes CSV cells, so every value is a string.
// Required columns and the money pattern come from the actual parser contract.
func LocalFinanceCSVRowSchema() map[string]any {
	text := func(description string) map[string]any {
		return map[string]any{"type": "string", "description": description}
	}
	nonempty := func(description string) map[string]any {
		return map[string]any{"type": "string", "minLength": 1, "pattern": `\S`, "description": description}
	}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": append([]string{}, localCSVRequired...),
		"properties": map[string]any{
			"bank_name":           nonempty("Bank name; identical on every row. Reuse the existing account's name when matching it."),
			"account_type":        map[string]any{"type": "string", "enum": []string{"SAVINGS", "CURRENT", "CREDIT_CARD", "WALLET"}},
			"account_number_mask": nonempty("Masked account/card identifier; identical on every row. Match the existing account when applicable."),
			"date":                map[string]any{"type": "string", "format": "date", "pattern": `^[0-9]{4}-[0-9]{2}-[0-9]{2}$`, "description": "Valid transaction date, YYYY-MM-DD."},
			"narration":           nonempty("Source narration, preserving references and meaningful text; keep consistent on repeat conversions."),
			"amount":              map[string]any{"type": "string", "pattern": `^[0-9]+(?:\.[0-9]{1,2})?$`, "not": map[string]any{"pattern": `^0+(?:\.0{1,2})?$`}, "description": "Strictly positive INR decimal, at most two fractional digits; no currency symbols, grouping, sign or exponent."},
			"tx_type":             map[string]any{"type": "string", "enum": []string{"DEBIT", "CREDIT"}, "description": "DEBIT for money out/card charges; CREDIT for money in/refunds/card payments."},
			"account_number":      text("Optional full identifier, identical on every row; preserve leading zeros. Prefer the mask when sufficient."),
			"reference_number":    text("Exact source reference, preserving leading zeros; keep stable for deduplication. Empty uses narration extraction."),
			"value_date":          map[string]any{"type": "string", "anyOf": []any{map[string]any{"const": ""}, map[string]any{"format": "date", "pattern": `^[0-9]{4}-[0-9]{2}-[0-9]{2}$`}}, "description": "Source value date YYYY-MM-DD, or empty when unavailable."},
			"running_balance":     map[string]any{"type": "string", "anyOf": []any{map[string]any{"const": ""}, map[string]any{"pattern": csvMoney.String()}}, "description": "Source signed balance with at most two fractional digits, or empty; zero and negative balances are valid."},
			"cleaned_payee":       text("Optional payee; empty lets LocalFinance clean the source narration."),
		},
	}
}
