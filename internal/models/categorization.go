package models

import (
	"regexp"
	"strings"
)

// MatchCategoryRule is shared by bank imports, rule reapplication and Splitwise.
// Rules arrive in priority order; a nil result means no rule matched.
func MatchCategoryRule(t Transaction, rules []CategorizationRule) *string {
	for _, r := range rules {
		if !r.MatchesTxType(string(t.TxType)) || r.MatchesException(t.RawNarration, t.CleanedPayee) {
			continue
		}
		value := t.RawNarration
		switch r.MatchField {
		case "cleaned_payee":
			value = t.CleanedPayee
		case "reference_number":
			value = t.ReferenceNumber
		case "upi_vpa":
			value = ""
			if t.UPIVPA != nil {
				value = *t.UPIVPA
			}
		}
		matched := false
		switch r.MatchType {
		case "CONTAINS":
			matched = strings.Contains(strings.ToUpper(value), strings.ToUpper(r.MatchPattern))
		case "EXACT":
			matched = strings.EqualFold(value, r.MatchPattern)
		case "STARTS_WITH":
			matched = strings.HasPrefix(strings.ToUpper(value), strings.ToUpper(r.MatchPattern))
		case "REGEX":
			if re, err := regexp.Compile(r.MatchPattern); err == nil {
				matched = re.MatchString(value)
			}
		}
		if matched {
			category := r.TargetCategoryID
			return &category
		}
	}
	return nil
}
