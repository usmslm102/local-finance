package models

// SplitwiseEntry stores net balances separately from statement cash movements.
// Amounts are integer paise; a CSV net balance does not prove who paid.
type SplitwiseEntry struct {
	ID                   string   `json:"id"`
	Group                string   `json:"group"`
	Person               string   `json:"person"`
	Date                 string   `json:"date"`
	Description          string   `json:"description"`
	Category             string   `json:"category"`
	Cost                 int64    `json:"cost_cents"`
	Net                  int64    `json:"net_cents"`
	Share                int64    `json:"share_cents"`
	Kind                 string   `json:"kind"`
	Status               string   `json:"status"`
	TransactionID        *string  `json:"transaction_id,omitempty"`
	CategoryID           *string  `json:"category_id,omitempty"`
	Members              []string `json:"members"`
	Notes                string   `json:"notes"`
	Tags                 string   `json:"tags"`
	IsManualCategory     bool     `json:"is_manual_category"`
	RemovedTransactionID *string  `json:"removed_transaction_id,omitempty"`
}

// Member aliases are explicit bank payee names or UPI addresses, never automatic rules.
type SplitwiseMember struct {
	Group   string   `json:"group"`
	Name    string   `json:"name"`
	Person  string   `json:"person"`
	Aliases []string `json:"aliases"`
	Pattern string   `json:"pattern"`
}

type SplitwiseImportResult struct {
	Inserted     int `json:"inserted"`
	Duplicates   int `json:"duplicates"`
	Matched      int `json:"matched"`
	PaidByOthers int `json:"paid_by_others"`
	Skipped      int `json:"skipped"`
	Uninvolved   int `json:"uninvolved"`
	Transfers    int `json:"transfers"`
}

type SplitwiseSettlementCandidate struct {
	Transaction Transaction `json:"transaction"`
	Group       string      `json:"group"`
	Member      string      `json:"member"`
}

type SplitwiseSettlementConfirmation struct {
	TransactionID string `json:"transaction_id"`
	Group         string `json:"group"`
	Member        string `json:"member"`
}

type SplitwiseMapping struct {
	Entry SplitwiseEntry `json:"entry"`
	Bank  *Transaction   `json:"bank,omitempty"`
}

type SplitwiseConfirmation struct {
	Share         int64  `json:"share_cents"`
	TransactionID string `json:"transaction_id"`
	Ignore        bool   `json:"ignore"`
	CategoryID    string `json:"category_id"`
}

type SplitwiseMatchOptions struct {
	Share                 int64
	Search                string
	AllowMemberSettlement bool
}
