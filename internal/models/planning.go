package models

// Goal balances are user-entered plans, never inferred ledger transactions.
type SavingsGoal struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	TargetAmount        float64 `json:"target_amount"`
	SavedAmount         float64 `json:"saved_amount"`
	MonthlyContribution float64 `json:"monthly_contribution"`
	TargetDate          string  `json:"target_date"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type DataHealthAccount struct {
	AccountID         string `json:"account_id"`
	LatestTransaction string `json:"latest_transaction"`
	TransactionCount  int    `json:"transaction_count"`
	ReviewCount       int    `json:"review_count"`
}
