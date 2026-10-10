package models

import (
	"errors"
	"regexp"
	"strings"
)

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
