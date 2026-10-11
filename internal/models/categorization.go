package models

// MatchCategoryRule is shared by bank imports, rule reapplication and Splitwise.
// Rules arrive in priority order; a nil result means no rule matched.
func MatchCategoryRule(t Transaction, rules []CategorizationRule) *string {
	for _, r := range rules {
		if r.MatchesTransaction(t) {
			category := r.TargetCategoryID
			return &category
		}
	}
	return nil
}
