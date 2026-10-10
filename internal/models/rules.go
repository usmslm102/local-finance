package models

import (
	"errors"
	"regexp"
	"strings"
)

// CategorizationRulePatch distinguishes omitted fields from explicit zero,
// false, or empty values. ApplyTo merges it into the current saved rule.
type CategorizationRulePatch struct {
	Priority         *int    `json:"priority"`
	MatchField       *string `json:"match_field"`
	MatchType        *string `json:"match_type"`
	MatchPattern     *string `json:"match_pattern"`
	ExcludePattern   *string `json:"exclude_pattern"`
	TxType           *string `json:"tx_type"`
	TargetCategoryID *string `json:"target_category_id"`
	AssignTags       *string `json:"assign_tags"`
	IsActive         *bool   `json:"is_active"`
}

func (p CategorizationRulePatch) ApplyTo(r *CategorizationRule) {
	if p.Priority != nil {
		r.Priority = *p.Priority
	}
	if p.MatchField != nil {
		r.MatchField = *p.MatchField
	}
	if p.MatchType != nil {
		r.MatchType = *p.MatchType
	}
	if p.MatchPattern != nil {
		r.MatchPattern = *p.MatchPattern
	}
	if p.ExcludePattern != nil {
		r.ExcludePattern = *p.ExcludePattern
	}
	if p.TxType != nil {
		r.TxType = *p.TxType
	}
	if p.TargetCategoryID != nil {
		r.TargetCategoryID = *p.TargetCategoryID
	}
	if p.AssignTags != nil {
		r.AssignTags = *p.AssignTags
	}
	if p.IsActive != nil {
		r.IsActive = *p.IsActive
	}
}

// ValidatePatterns checks patterns using the same regex mode and exclusion
// detection as categorization. Matcher and pattern must be validated together.
func (r *CategorizationRule) ValidatePatterns() error {
	if strings.TrimSpace(r.MatchPattern) == "" {
		return errors.New("match_pattern cannot be empty")
	}
	if r.MatchType == "REGEX" {
		if _, err := regexp.Compile("(?i)" + r.MatchPattern); err != nil {
			return errors.New("invalid match_pattern regex")
		}
	}
	if strings.ContainsAny(r.ExcludePattern, "|.*+?^$") {
		if _, err := regexp.Compile("(?i)" + strings.TrimSpace(r.ExcludePattern)); err != nil {
			return errors.New("invalid exclude_pattern regex")
		}
	}
	return nil
}

// MatchesTransaction is shared by statement categorization and ledger reapplication.
// Transfer precedence and unmatched-category policy belong to the caller.
func (r *CategorizationRule) MatchesTransaction(tx Transaction) bool {
	if !r.MatchesTxType(string(tx.TxType)) || r.MatchesException(tx.RawNarration, tx.CleanedPayee) {
		return false
	}
	target := tx.RawNarration
	switch r.MatchField {
	case "cleaned_payee":
		target = tx.CleanedPayee
	case "reference_number":
		target = tx.ReferenceNumber
	case "upi_vpa":
		target = ""
		if tx.UPIVPA != nil {
			target = *tx.UPIVPA
		}
	}
	switch r.MatchType {
	case "CONTAINS":
		return strings.Contains(strings.ToUpper(target), strings.ToUpper(r.MatchPattern))
	case "EXACT":
		return strings.EqualFold(target, r.MatchPattern)
	case "STARTS_WITH":
		return strings.HasPrefix(strings.ToUpper(target), strings.ToUpper(r.MatchPattern))
	case "REGEX":
		re, err := regexp.Compile(r.MatchPattern)
		return err == nil && re.MatchString(target)
	}
	return false
}
