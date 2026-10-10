package models

// SplitwiseEntry stores net balances separately from statement cash movements.
// Amounts are integer paise; a CSV net balance does not prove who paid.
type SplitwiseEntry struct {
	ID            string  `json:"id"`
	Group         string  `json:"group"`
	Person        string  `json:"person"`
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Category      string  `json:"category"`
	Cost          int64   `json:"cost_cents"`
	Net           int64   `json:"net_cents"`
	Share         int64   `json:"share_cents"`
	Kind          string  `json:"kind"`
	Status        string  `json:"status"`
	TransactionID *string `json:"transaction_id,omitempty"`
	CategoryID    *string `json:"category_id,omitempty"`
}

type SplitwiseConfirmation struct {
	Share         int64  `json:"share_cents"`
	TransactionID string `json:"transaction_id"`
	Ignore        bool   `json:"ignore"`
	CategoryID    string `json:"category_id"`
}
