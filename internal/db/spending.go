package db

import (
	"fmt"
	"local-finance/internal/models"
)

// SQL aliases here are internal identifiers, never user input.
func sqlColumnPrefix(alias string) string {
	if alias != "" {
		return alias + "."
	}
	return ""
}

func excludeTransferCategoryFilter(alias string) string {
	return fmt.Sprintf("COALESCE(%scategory_id, '') != '%s'", sqlColumnPrefix(alias), models.CategoryTransfersID)
}

// Transfers belong to neither income nor expenses, including activity counts.
func financialActivityFilter(alias string) string {
	prefix := sqlColumnPrefix(alias)
	return prefix + "is_transfer = 0 AND " + prefix + "is_excluded = 0 AND " + excludeTransferCategoryFilter(alias)
}

func spendingFilter(alias string) string {
	return sqlColumnPrefix(alias) + "tx_type = 'DEBIT' AND " + sqlColumnPrefix(alias) + "amount > 0 AND " + financialActivityFilter(alias)
}

func incomeFilter(alias string) string {
	return sqlColumnPrefix(alias) + "tx_type = 'CREDIT' AND " + financialActivityFilter(alias)
}

func financialFlowFilter(alias string) string {
	prefix := sqlColumnPrefix(alias)
	return "(" + prefix + "tx_type = 'DEBIT' OR " + prefix + "tx_type = 'CREDIT') AND " + financialActivityFilter(alias)
}
