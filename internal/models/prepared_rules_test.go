package models

import "testing"

func TestPreparedRulesPreserveMatchAndExceptionSemantics(t *testing.T) {
	for _, pattern := range []string{"(?i)salary", "salary", "[", ""} {
		for _, exclude := range []string{"maid, staff", "cook|maid", "[.*", ""} {
			rules := []CategorizationRule{{MatchField: "raw_narration", MatchType: "REGEX", MatchPattern: pattern, ExcludePattern: exclude, TxType: "CREDIT", TargetCategoryID: "cat_salary"}}
			prepared := PrepareRules(rules)
			for _, narration := range []string{"SALARY", "salary", "salary maid", "salary cook", "other"} {
				for _, kind := range []TxType{TxTypeCredit, TxTypeDebit} {
					tx := Transaction{RawNarration: narration, CleanedPayee: narration, TxType: kind}
					old, new := MatchCategoryRule(tx, rules), MatchCategoryRule(tx, prepared)
					if (old == nil) != (new == nil) || (old != nil && *old != *new) {
						t.Fatalf("prepared matcher changed %q %q %+v", pattern, exclude, tx)
					}
				}
			}
			if rules[0].prepared {
				t.Fatal("mutated caller's rules")
			}
		}
	}
}
