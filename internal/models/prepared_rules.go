package models

import (
	"regexp"
	"strings"
)

// PrepareRules returns an operation-owned copy. Invalid persisted patterns keep
// their previous non-matching behaviour; case sensitivity is unchanged.
func PrepareRules(rules []CategorizationRule) []CategorizationRule {
	result := append([]CategorizationRule(nil), rules...)
	for i := range result {
		r := &result[i]
		r.prepared = true
		if r.MatchType == "REGEX" {
			r.matchExpression, _ = regexp.Compile(r.MatchPattern)
		}
		pattern := strings.TrimSpace(r.ExcludePattern)
		if strings.ContainsAny(pattern, "|.*+?^$") {
			r.excludeExpression, _ = regexp.Compile("(?i)" + pattern)
		}
	}
	return result
}
