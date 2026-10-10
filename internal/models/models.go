package models

import (
	"regexp"
	"strings"
	"time"
)

type AccountType string

const (
	AccountTypeSavings    AccountType = "SAVINGS"
	AccountTypeCurrent    AccountType = "CURRENT"
	AccountTypeCreditCard AccountType = "CREDIT_CARD"
	AccountTypeWallet     AccountType = "WALLET"
)

type TxType string

const (
	TxTypeDebit  TxType = "DEBIT"
	TxTypeCredit TxType = "CREDIT"
)

type PaymentMode string

const (
	PaymentModeUPI        PaymentMode = "UPI"
	PaymentModeCardPOS    PaymentMode = "CARD_POS"
	PaymentModeCardOnline PaymentMode = "CARD_ONLINE"
	PaymentModeNetBanking PaymentMode = "NET_BANKING"
	PaymentModeNEFT       PaymentMode = "NEFT"
	PaymentModeIMPS       PaymentMode = "IMPS"
	PaymentModeRTGS       PaymentMode = "RTGS"
	PaymentModeATM        PaymentMode = "ATM"
	PaymentModeCharges    PaymentMode = "CHARGES"
	PaymentModeInterest   PaymentMode = "INTEREST"
	PaymentModeSalary     PaymentMode = "SALARY"
	PaymentModeOther      PaymentMode = "OTHER"
)

type Account struct {
	ID                string      `json:"id"`
	BankName          string      `json:"bank_name"`
	AccountType       AccountType `json:"account_type"`
	AccountNumber     *string     `json:"account_number,omitempty"`
	AccountNumberMask string      `json:"account_number_mask"`
	Currency          string      `json:"currency"`
	OpeningBalance    float64     `json:"opening_balance"`
	CurrentBalance    float64     `json:"current_balance"`
	CreditLimit       *float64    `json:"credit_limit,omitempty"`
	BillingCycleDay   *int        `json:"billing_cycle_day,omitempty"`
	Nickname          *string     `json:"nickname,omitempty"`
	CustomerID        *string     `json:"customer_id,omitempty"`
	IFSCCode          *string     `json:"ifsc_code,omitempty"`
	BranchName        *string     `json:"branch_name,omitempty"`
	AccountHolderName *string     `json:"account_holder_name,omitempty"`
	CardNetwork       *string     `json:"card_network,omitempty"` // VISA, MASTERCARD, RUPAY, AMEX
	CardVariant       *string     `json:"card_variant,omitempty"` // Regalia, Swiggy, Amazon Pay, Flipkart
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

type UpdateAccountRequest struct {
	BankName          *string      `json:"bank_name,omitempty"`
	AccountType       *AccountType `json:"account_type,omitempty"`
	AccountNumber     *string      `json:"account_number,omitempty"`
	AccountNumberMask *string      `json:"account_number_mask,omitempty"`
	Nickname          *string      `json:"nickname,omitempty"`
	AccountHolderName *string      `json:"account_holder_name,omitempty"`
}

type StatementImport struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	BankName          string    `json:"bank_name,omitempty"`
	Filename          string    `json:"filename"`
	FileHash          string    `json:"file_hash"`
	StatementFormat   string    `json:"statement_format"`
	ParserUsed        string    `json:"parser_used"`
	StartDate         *string   `json:"start_date,omitempty"`
	EndDate           *string   `json:"end_date,omitempty"`
	TotalTransactions int       `json:"total_transactions"`
	OpeningBalance    *float64  `json:"opening_balance,omitempty"`
	ClosingBalance    *float64  `json:"closing_balance,omitempty"`
	TotalDebits       *float64  `json:"total_debits,omitempty"`
	TotalCredits      *float64  `json:"total_credits,omitempty"`
	ImportedAt        time.Time `json:"imported_at"`
}

type Category struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id,omitempty"`
	ColorHex string  `json:"color_hex"`
	Icon     string  `json:"icon"`
	IsSystem bool    `json:"is_system"`
}

type CategorizationRule struct {
	ID               string `json:"id"`
	Priority         int    `json:"priority"`
	MatchField       string `json:"match_field"` // raw_narration, cleaned_payee, reference_number, upi_vpa
	MatchType        string `json:"match_type"`  // CONTAINS, REGEX, EXACT, STARTS_WITH
	MatchPattern     string `json:"match_pattern"`
	ExcludePattern   string `json:"exclude_pattern,omitempty"` // Comma-separated negative keywords or regex to skip rule
	TxType           string `json:"tx_type,omitempty"`         // ALL, DEBIT, CREDIT
	TargetCategoryID string `json:"target_category_id"`
	TargetCategory   string `json:"target_category,omitempty"`
	AssignTags       string `json:"assign_tags,omitempty"`
	IsActive         bool   `json:"is_active"`
}

// MatchesTxType checks if this rule applies to the given transaction type (DEBIT or CREDIT).
func (r *CategorizationRule) MatchesTxType(txType string) bool {
	if r.TxType == "" || strings.EqualFold(r.TxType, "ALL") {
		return true
	}
	return strings.EqualFold(r.TxType, txType)
}

// MatchesException checks if the transaction narration or payee matches any negative keywords or regex to skip.
func (r *CategorizationRule) MatchesException(narration, payee string) bool {
	pattern := strings.TrimSpace(r.ExcludePattern)
	if pattern == "" {
		return false
	}
	upperNarr := strings.ToUpper(narration)
	upperPayee := strings.ToUpper(payee)

	// Check comma-separated tokens first
	tokens := strings.Split(pattern, ",")
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		upperTok := strings.ToUpper(tok)
		if strings.Contains(upperNarr, upperTok) || strings.Contains(upperPayee, upperTok) {
			return true
		}
	}

	// Also check regex pattern if meta-characters are present
	if strings.ContainsAny(pattern, "|.*+?^$") {
		if re, err := regexp.Compile("(?i)" + pattern); err == nil {
			if re.MatchString(narration) || re.MatchString(payee) {
				return true
			}
		}
	}

	return false
}

type Transaction struct {
	ID                 string      `json:"id"`
	AccountID          string      `json:"account_id"`
	AccountName        string      `json:"account_name,omitempty"`
	StatementImportID  *string     `json:"statement_import_id,omitempty"`
	TxHash             string      `json:"tx_hash"`
	TxDate             string      `json:"tx_date"` // YYYY-MM-DD
	ValueDate          *string     `json:"value_date,omitempty"`
	RawNarration       string      `json:"raw_narration"`
	CleanedPayee       string      `json:"cleaned_payee"`
	PaymentMode        PaymentMode `json:"payment_mode"`
	ReferenceNumber    string      `json:"reference_number"`
	TxType             TxType      `json:"tx_type"` // DEBIT or CREDIT
	Amount             float64     `json:"amount"`
	RunningBalance     *float64    `json:"running_balance,omitempty"`
	CategoryID         *string     `json:"category_id,omitempty"`
	CategoryName       *string     `json:"category_name,omitempty"`
	CategoryColor      *string     `json:"category_color,omitempty"`
	CategoryIcon       *string     `json:"category_icon,omitempty"`
	UPIVPA             *string     `json:"upi_vpa,omitempty"`
	CardLast4          *string     `json:"card_last4,omitempty"`
	MerchantCategory   *string     `json:"merchant_category,omitempty"`
	CashbackAmount     float64     `json:"cashback_amount"`
	RewardPointsEarned float64     `json:"reward_points_earned"`
	IsTransfer          bool        `json:"is_transfer"`
	IsExcluded          bool        `json:"is_excluded"`
	TransferPeerID      *string     `json:"transfer_peer_id,omitempty"`
	TransferMatchReason *string     `json:"transfer_match_reason,omitempty"`
	NetAmount           *float64    `json:"net_amount,omitempty"`
	OriginalCurrency    *string     `json:"original_currency,omitempty"`
	OriginalAmount      *float64    `json:"original_amount,omitempty"`
	IsRecurring         bool        `json:"is_recurring"`
	IsManualCategory    bool        `json:"is_manual_category"`
	Notes               string      `json:"notes"`
	Tags                string      `json:"tags"`
	CreatedAt           time.Time   `json:"created_at"`
}

type UpdateTransactionRequest struct {
	CategoryID       *string `json:"category_id"`
	Notes            *string `json:"notes,omitempty"`
	Tags             *string `json:"tags,omitempty"`
	IsManualCategory *bool   `json:"is_manual_category,omitempty"`
}

type CreditCardBill struct {
	ID                   string    `json:"id"`
	AccountID            string    `json:"account_id"`
	BankName             string    `json:"bank_name,omitempty"`
	StatementImportID    *string   `json:"statement_import_id,omitempty"`
	StatementDate        string    `json:"statement_date"`     // YYYY-MM-DD
	PaymentDueDate       string    `json:"payment_due_date"`    // YYYY-MM-DD
	TotalDueAmount       float64   `json:"total_due_amount"`
	MinimumDueAmount     *float64  `json:"minimum_due_amount,omitempty"`
	RewardPointsEarned   float64   `json:"reward_points_earned"`
	RewardPointsBalance  float64   `json:"reward_points_balance"`
	CashbackEarned       float64   `json:"cashback_earned"`
	CashbackCredited     float64   `json:"cashback_credited"`
	FinanceCharges       float64   `json:"finance_charges"`
	CreditLimit          *float64  `json:"credit_limit,omitempty"`
	AvailableCreditLimit *float64  `json:"available_credit_limit,omitempty"`
	PaymentStatus        string    `json:"payment_status"` // UNPAID, PAID, PARTIALLY_PAID
	CreatedAt            time.Time `json:"created_at"`
}

type ImportResult struct {
	StatementImportID string   `json:"statement_import_id"`
	AccountID         string   `json:"account_id"`
	BankName          string   `json:"bank_name"`
	AccountType       string   `json:"account_type"`
	ParserUsed        string   `json:"parser_used"`
	Confidence        float64  `json:"confidence"`
	TotalParsed       int      `json:"total_parsed"`
	InsertedCount     int      `json:"inserted_count"`
	DuplicateCount    int      `json:"duplicate_count"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	Warnings          []string `json:"warnings,omitempty"`
}

type AnalyticsOverview struct {
	TotalIncome           float64           `json:"total_income"`
	TotalExpense          float64           `json:"total_expense"`
	NetSavings            float64           `json:"net_savings"`
	SavingsRate           float64           `json:"savings_rate"`
	TotalBankLiquidity    float64           `json:"total_bank_liquidity"`
	TotalCreditDue        float64           `json:"total_credit_due"`
	TotalCreditLimit      float64           `json:"total_credit_limit"`
	CreditUtilizationRate float64           `json:"credit_utilization_rate"`
	TotalCashbackEarned   float64           `json:"total_cashback_earned"`
	TotalRewardPoints     float64           `json:"total_reward_points"`
	TotalAccounts         int               `json:"total_accounts"`
	TotalTransactions     int               `json:"total_transactions"`
	CategoryBreakdown     []CategorySpend   `json:"category_breakdown"`
	MonthlyTrends         []MonthlyCashFlow `json:"monthly_trends"`
	TopPayees             []PayeeSpend      `json:"top_payees"`
	UpcomingBills         []CreditCardBill  `json:"upcoming_bills,omitempty"`
}

type CategorySpend struct {
	CategoryID   string  `json:"category_id"`
	CategoryName string  `json:"category_name"`
	ColorHex     string  `json:"color_hex"`
	Icon         string  `json:"icon"`
	TotalAmount  float64 `json:"total_amount"`
	TxCount      int     `json:"tx_count"`
	Percentage   float64 `json:"percentage"`
}

type MonthlyCashFlow struct {
	Month   string  `json:"month"` // YYYY-MM
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
}

type PayeeSpend struct {
	Payee       string  `json:"payee"`
	PaymentMode string  `json:"payment_mode"`
	TotalSpent  float64 `json:"total_spent"`
	TxCount     int     `json:"tx_count"`
}

type PreviewTransactionItem struct {
	Date            string   `json:"date"`
	ValueDate       *string  `json:"value_date,omitempty"`
	RawNarration    string   `json:"raw_narration"`
	CleanedPayee    string   `json:"cleaned_payee"`
	PaymentMode     string   `json:"payment_mode"`
	ReferenceNumber string   `json:"reference_number"`
	TxType          string   `json:"tx_type"`
	Amount          float64  `json:"amount"`
	RunningBalance  *float64 `json:"running_balance,omitempty"`
	CategoryID      *string  `json:"category_id,omitempty"`
	CategoryName    string   `json:"category_name,omitempty"`
	CategoryColor   string   `json:"category_color,omitempty"`
	CategoryIcon    string   `json:"category_icon,omitempty"`
	UPIVPA          *string  `json:"upi_vpa,omitempty"`
	IsTransfer      bool     `json:"is_transfer"`
	IsDuplicate     bool     `json:"is_duplicate"`
}

type StatementPreviewResult struct {
	FileID               string                   `json:"file_id"`
	Filename             string                   `json:"filename"`
	FileSize             int64                    `json:"file_size"`
	FileHash             string                   `json:"file_hash"`
	ParserID             string                   `json:"parser_id"`
	ParserName           string                   `json:"parser_name"`
	Confidence           float64                  `json:"confidence"`
	BankName             string                   `json:"bank_name"`
	AccountType          string                   `json:"account_type"`
	AccountNumber        string                   `json:"account_number,omitempty"`
	AccountNumberMask    string                   `json:"account_number_mask"`
	AccountHolderName    string                   `json:"account_holder_name,omitempty"`
	StartDate            string                   `json:"start_date"`
	EndDate              string                   `json:"end_date"`
	TotalTransactions    int                      `json:"total_transactions"`
	OpeningBalance       float64                  `json:"opening_balance"`
	ClosingBalance       float64                  `json:"closing_balance"`
	TotalDebits          float64                  `json:"total_debits"`
	TotalCredits         float64                  `json:"total_credits"`
	TotalDueAmount       float64                  `json:"total_due_amount"`
	MinimumDueAmount     float64                  `json:"minimum_due_amount"`
	PaymentDueDate       string                   `json:"payment_due_date"`
	CreditLimit          float64                  `json:"credit_limit"`
	AvailableCreditLimit float64                  `json:"available_credit_limit"`
	RewardPointsBalance  float64                  `json:"reward_points_balance"`
	CashbackEarned       float64                  `json:"cashback_earned"`
	CardNetwork          string                   `json:"card_network"`
	CardVariant          string                   `json:"card_variant"`
	Transactions         []PreviewTransactionItem `json:"transactions"`
	ExistingTxsCount     int                      `json:"existing_txs_count"`
	NewTxsCount          int                      `json:"new_txs_count"`
	Warnings             []string                 `json:"warnings,omitempty"`
	Error                string                   `json:"error,omitempty"`
	RequiresPassword     bool                     `json:"requires_password"`
}

type DatabaseInfo struct {
	Path              string    `json:"path"`
	FileSize          int64     `json:"file_size"`
	WALMode           bool      `json:"wal_mode"`
	LastModified      time.Time `json:"last_modified"`
	TotalAccounts     int       `json:"total_accounts"`
	TotalTransactions int       `json:"total_transactions"`
	TotalStatements   int       `json:"total_statements"`
	TotalRules        int       `json:"total_rules"`
}

type FullExportData struct {
	Splitwise []SplitwiseEntry `json:"splitwise"`
	Investments      []InvestmentSnapshot `json:"investments"`
	ExportedAt       time.Time            `json:"exported_at"`
	Version          string               `json:"version"`
	Accounts         []Account            `json:"accounts"`
	Transactions     []Transaction        `json:"transactions"`
	CreditCardBills  []CreditCardBill     `json:"credit_card_bills"`
	StatementImports []StatementImport    `json:"statement_imports"`
	Categories       []Category           `json:"categories"`
	Rules            []CategorizationRule `json:"rules"`
	Subscriptions    []Subscription       `json:"subscriptions,omitempty"`
}

type SubscriptionFrequency string

const (
	FrequencyMonthly   SubscriptionFrequency = "MONTHLY"
	FrequencyQuarterly SubscriptionFrequency = "QUARTERLY"
	FrequencyYearly    SubscriptionFrequency = "YEARLY"
	FrequencyWeekly    SubscriptionFrequency = "WEEKLY"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "ACTIVE"
	SubscriptionStatusPaused    SubscriptionStatus = "PAUSED"
	SubscriptionStatusCancelled SubscriptionStatus = "CANCELLED"
)

type Subscription struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	MerchantPattern string                `json:"merchant_pattern"`
	CategoryID      *string               `json:"category_id,omitempty"`
	CategoryName    *string               `json:"category_name,omitempty"`
	CategoryColor   *string               `json:"category_color,omitempty"`
	CategoryIcon    *string               `json:"category_icon,omitempty"`
	AccountID       *string               `json:"account_id,omitempty"`
	AccountName     *string               `json:"account_name,omitempty"`
	Frequency       SubscriptionFrequency `json:"frequency"`
	ExpectedAmount  float64               `json:"expected_amount"`
	Currency        string                `json:"currency"`
	BillingDay      int                   `json:"billing_day"`
	NextDueDate     *string               `json:"next_due_date,omitempty"`
	LastPaidDate    *string               `json:"last_paid_date,omitempty"`
	LastPaidAmount  *float64              `json:"last_paid_amount,omitempty"`
	Status          SubscriptionStatus    `json:"status"`
	IsAutoDetected  bool                  `json:"is_auto_detected"`
	Notes           *string               `json:"notes,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type SubscriptionsSummary struct {
	TotalActive       int            `json:"total_active"`
	MonthlyBurnRate   float64        `json:"monthly_burn_rate"`
	AnnualProjected   float64        `json:"annual_projected"`
	UpcomingIn30Days  int            `json:"upcoming_in_30_days"`
	Subscriptions     []Subscription `json:"subscriptions"`
}

type UpsertSubscriptionRequest struct {
	Name            string                `json:"name" binding:"required"`
	MerchantPattern string                `json:"merchant_pattern" binding:"required"`
	CategoryID      *string               `json:"category_id"`
	AccountID       *string               `json:"account_id"`
	Frequency       SubscriptionFrequency `json:"frequency"`
	ExpectedAmount  float64               `json:"expected_amount" binding:"required"`
	BillingDay      int                   `json:"billing_day"`
	NextDueDate     *string               `json:"next_due_date"`
	Status          SubscriptionStatus    `json:"status"`
	Notes           *string               `json:"notes"`
}

type CardRewardRule struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	MerchantPattern   string    `json:"merchant_pattern"`
	CategoryName      string    `json:"category_name"`
	RewardPercentage  float64   `json:"reward_percentage"`
	RewardDescription string    `json:"reward_description"`
	MaxCapPerMonth    *float64  `json:"max_cap_per_month,omitempty"`
	MinSpendPerTxn    *float64  `json:"min_spend_per_txn,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CardDetails struct {
	Account
	AnnualFee                 float64          `json:"annual_fee"`
	FeeWaiverThreshold        float64          `json:"fee_waiver_threshold"`
	BillingDay                int              `json:"billing_day"`
	PaymentDueDays            int              `json:"payment_due_days"`
	CardColor                 string           `json:"card_color"`
	RewardType                string           `json:"reward_type"`
	BaseRewardRate            float64          `json:"base_reward_rate"`
	TotalSpendThisYear        float64          `json:"total_spend_this_year"`
	FeeWaiverProgressPct      float64          `json:"fee_waiver_progress_pct"`
	FeeWaiverRemaining        float64          `json:"fee_waiver_remaining"`
	NextStatementDate         string           `json:"next_statement_date"`
	NextPaymentDueDate        string           `json:"next_payment_due_date"`
	DaysUntilStatement        int              `json:"days_until_statement"`
	InterestFreeDaysRemaining int              `json:"interest_free_days_remaining"`
	TotalCashbackEarned       float64          `json:"total_cashback_earned"`
	TotalRewardPoints         float64          `json:"total_reward_points"`
	TotalDueAmount            float64          `json:"total_due_amount"`
	AvailableCreditLimit      float64          `json:"available_credit_limit"`
	RewardRules               []CardRewardRule `json:"reward_rules"`
}

type CardRecommendationRequest struct {
	Merchant string  `json:"merchant"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

type CardRecommendation struct {
	AccountID         string  `json:"account_id"`
	BankName          string  `json:"bank_name"`
	CardVariant       string  `json:"card_variant"`
	CardNetwork       string  `json:"card_network"`
	CardColor         string  `json:"card_color"`
	RewardRate        float64 `json:"reward_rate"`
	RewardDescription string  `json:"reward_description"`
	EstimatedReward   float64 `json:"estimated_reward"`
	RewardType        string  `json:"reward_type"`
	InterestFreeDays  int     `json:"interest_free_days"`
	Rank              int     `json:"rank"`
	Notes             string  `json:"notes"`
}

type CardPortfolioOverview struct {
	Cards                     []CardDetails `json:"cards"`
	BestCardToSwipeToday      *CardDetails  `json:"best_card_to_swipe_today,omitempty"`
	TotalCreditLimit          float64       `json:"total_credit_limit"`
	TotalOutstanding          float64       `json:"total_outstanding"`
	OverallUtilizationRate    float64       `json:"overall_utilization_rate"`
	TotalAnnualFeeLiability   float64       `json:"total_annual_fee_liability"`
	TotalFeeSavingsProjected  float64       `json:"total_fee_savings_projected"`
	TotalCashbackEarned       float64       `json:"total_cashback_earned"`
	TotalRewardPoints         float64       `json:"total_reward_points"`
}

type UpdateCardMetadataRequest struct {
	CardVariant        *string  `json:"card_variant"`
	CardNetwork        *string  `json:"card_network"`
	CardColor          *string  `json:"card_color"`
	AccountHolderName  *string  `json:"account_holder_name"`
	CreditLimit        *float64 `json:"credit_limit"`
	AnnualFee          *float64 `json:"annual_fee"`
	FeeWaiverThreshold *float64 `json:"fee_waiver_threshold"`
	BillingDay         *int     `json:"billing_day"`
	PaymentDueDays     *int     `json:"payment_due_days"`
	BaseRewardRate     *float64 `json:"base_reward_rate"`
	RewardType         *string  `json:"reward_type"`
}

type CategoryBudget struct {
	ID                         string  `json:"id"`
	CategoryID                 string  `json:"category_id"`
	CategoryName               string  `json:"category_name"`
	CategoryColor              string  `json:"category_color"`
	CategoryIcon               string  `json:"category_icon"`
	MonthlyLimit               float64 `json:"monthly_limit"`
	Month                      string  `json:"month"` // e.g. "2026-08"
	ActualSpend                float64 `json:"actual_spend"`
	SpentPercentage            float64 `json:"spent_percentage"`
	RemainingAmount            float64 `json:"remaining_amount"`
	Status                     string  `json:"status"` // "SAFE", "WARNING", "EXCEEDED"
	DaysElapsedInMonth         int     `json:"days_elapsed_in_month"`
	DaysRemainingInMonth       int     `json:"days_remaining_in_month"`
	TotalDaysInMonth           int     `json:"total_days_in_month"`
	DailyRecommendedAllowance  float64 `json:"daily_recommended_allowance"`
	ProjectedMonthEndSpend     float64 `json:"projected_month_end_spend"`
	PacingStatus               string  `json:"pacing_status"` // "ON_TRACK", "PACING_HIGH", "PACING_EXCEEDED"
}

type BudgetSummary struct {
	Month                              string           `json:"month"`
	TotalBudget                        float64          `json:"total_budget"`
	TotalSpent                         float64          `json:"total_spent"`
	TotalRemaining                     float64          `json:"total_remaining"`
	OverallSpentPercentage             float64          `json:"overall_spent_percentage"`
	TotalDaysInMonth                   int              `json:"total_days_in_month"`
	DaysElapsedInMonth                 int              `json:"days_elapsed_in_month"`
	DaysRemainingInMonth               int              `json:"days_remaining_in_month"`
	OverallDailyRecommendedAllowance   float64          `json:"overall_daily_recommended_allowance"`
	CategoriesWithBudgets              int              `json:"categories_with_budgets"`
	OverspentCategoriesCount           int              `json:"overspent_categories_count"`
	WarningCategoriesCount             int              `json:"warning_categories_count"`
	Budgets                            []CategoryBudget `json:"budgets"`
}

type UpsertCategoryBudgetRequest struct {
	CategoryID   string  `json:"category_id" binding:"required"`
	MonthlyLimit float64 `json:"monthly_limit" binding:"required"`
	Month        string  `json:"month"` // optional; if empty defaults to current month
}

// ==========================================
// TRANSFER RECONCILIATION MODELS
// ==========================================

type TransferPair struct {
	ID                 string      `json:"id"`
	DebitTx            Transaction `json:"debit_tx"`
	CreditTx           Transaction `json:"credit_tx"`
	MatchConfidence    float64     `json:"match_confidence"`
	MatchReason        string      `json:"match_reason"`
	IsConfirmed        bool        `json:"is_confirmed"`
	DateDifferenceDays int         `json:"date_difference_days"`
	AmountDifference   float64     `json:"amount_difference"`
}

type ReconciliationSummary struct {
	TotalPairedTransfers       int            `json:"total_paired_transfers"`
	TotalPairedAmount          float64        `json:"total_paired_amount"`
	PendingCandidatesCount     int            `json:"pending_candidates_count"`
	DoubleCountPreventedAmount float64        `json:"double_count_prevented_amount"`
	WalletExcludedCount        int            `json:"wallet_excluded_count"`
	WalletExcludedAmount       float64        `json:"wallet_excluded_amount"`
	Pairs                      []TransferPair `json:"pairs"`
	Candidates                 []TransferPair `json:"candidates"`
	WalletTransactions         []Transaction  `json:"wallet_transactions"`
}

type LinkTransferPairRequest struct {
	DebitTxID   string `json:"debit_tx_id" binding:"required"`
	CreditTxID  string `json:"credit_tx_id" binding:"required"`
	MatchReason string `json:"match_reason"`
}

type UnlinkTransferPairRequest struct {
	TxID string `json:"tx_id" binding:"required"`
}

// ==========================================
// MERCHANT & PAYEE INTELLIGENCE MODELS
// ==========================================

type MonthlySpendDataPoint struct {
	Month       string  `json:"month"` // "YYYY-MM"
	SpendAmount float64 `json:"spend_amount"`
	TxCount     int     `json:"tx_count"`
}

type PaymentSourceShare struct {
	AccountName string  `json:"account_name"`
	SpendAmount float64 `json:"spend_amount"`
	TxCount     int     `json:"tx_count"`
	SharePct    float64 `json:"share_pct"`
}

type MerchantProfile struct {
	CleanedPayee           string                  `json:"cleaned_payee"`
	CategoryName           string                  `json:"category_name"`
	CategoryColor          string                  `json:"category_color"`
	CategoryIcon           string                  `json:"category_icon"`
	TotalSpend             float64                 `json:"total_spend"`
	TotalCredits           float64                 `json:"total_credits"`
	NetSpend               float64                 `json:"net_spend"`
	TotalTxCount           int                     `json:"total_tx_count"`
	DebitTxCount           int                     `json:"debit_tx_count"`
	CreditTxCount          int                     `json:"credit_tx_count"`
	AverageOrderValue      float64                 `json:"average_order_value"`
	FirstTxDate            string                  `json:"first_tx_date"`
	LastTxDate             string                  `json:"last_tx_date"`
	DaysSinceLastTx        int                     `json:"days_since_last_tx"`
	PreferredPaymentMode   string                  `json:"preferred_payment_mode"`
	PreferredPaymentSource string                  `json:"preferred_payment_source"`
	PaymentSources         []PaymentSourceShare    `json:"payment_sources"`
	MonthlySpendHistory    []MonthlySpendDataPoint `json:"monthly_spend_history"`
	RecentTransactions     []Transaction           `json:"recent_transactions"`
}

type MerchantSummaryItem struct {
	CleanedPayee      string  `json:"cleaned_payee"`
	CategoryName      string  `json:"category_name"`
	CategoryColor     string  `json:"category_color"`
	CategoryIcon      string  `json:"category_icon"`
	TotalSpend        float64 `json:"total_spend"`
	TotalCredits      float64 `json:"total_credits"`
	TxCount           int     `json:"tx_count"`
	AverageOrderValue float64 `json:"average_order_value"`
	FirstTxDate       string  `json:"first_tx_date"`
	LastTxDate        string  `json:"last_tx_date"`
	SpendSharePct     float64 `json:"spend_share_pct"`
	PrimarySource     string  `json:"primary_source"`
}

type MerchantListResponse struct {
	TotalMerchants    int                   `json:"total_merchants"`
	TotalSpend        float64               `json:"total_spend"`
	TopCategory       string                `json:"top_category"`
	AverageOrderValue float64               `json:"average_order_value"`
	Merchants         []MerchantSummaryItem `json:"merchants"`
}

// ==========================================
// CASH FLOW SANKEY & MOM ANOMALY MODELS
// ==========================================

type SankeyNodeType string

const (
	SankeyNodeIncomeSource SankeyNodeType = "INCOME_SOURCE"
	SankeyNodeAccount      SankeyNodeType = "ACCOUNT"
	SankeyNodeChannel      SankeyNodeType = "CHANNEL"
	SankeyNodeCategory     SankeyNodeType = "CATEGORY"
	SankeyNodeSurplus      SankeyNodeType = "SURPLUS"
	SankeyNodeDeficit      SankeyNodeType = "DEFICIT"
)

type SankeyNode struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       SankeyNodeType `json:"type"`
	ColorHex   string         `json:"color_hex"`
	Icon       string         `json:"icon"`
	TotalValue float64        `json:"total_value"`
	Layer      int            `json:"layer"` // 0=Sources, 1=Accounts, 2=Channels, 3=Categories/Surplus
	TxCount    int            `json:"tx_count"`
}

type SankeyLink struct {
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Value    float64 `json:"value"`
	ColorHex string  `json:"color_hex,omitempty"`
}

type SankeyData struct {
	Nodes        []SankeyNode `json:"nodes"`
	Links        []SankeyLink `json:"links"`
	TotalInflow  float64      `json:"total_inflow"`
	TotalOutflow float64      `json:"total_outflow"`
	NetSurplus   float64      `json:"net_surplus"`
	SavingsRate  float64      `json:"savings_rate"`
}

type AnomalySeverity string

const (
	AnomalySeverityWarning AnomalySeverity = "warning"
	AnomalySeverityInfo    AnomalySeverity = "info"
	AnomalySeveritySuccess AnomalySeverity = "success"
	AnomalySeverityDanger  AnomalySeverity = "danger"
)

type MoMAnomaly struct {
	ID               string          `json:"id"`
	Type             string          `json:"type"` // SPIKE, DROP, NEW_SPEND, DEFICIT, SAVINGS_MILESTONE, CC_RELIANCE_SHIFT
	Severity         AnomalySeverity `json:"severity"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	CategoryName     string          `json:"category_name,omitempty"`
	CategoryColor    string          `json:"category_color,omitempty"`
	CurrentAmount    float64         `json:"current_amount"`
	PreviousAmount   float64         `json:"previous_amount"`
	DeltaAmount      float64         `json:"delta_amount"`
	PercentageChange float64         `json:"percentage_change"`
	TopContributor   string          `json:"top_contributor,omitempty"`
}

type MonthlyCategoryDataPoint struct {
	Month  string  `json:"month"` // "YYYY-MM"
	Amount float64 `json:"amount"`
}

type CategoryComparisonItem struct {
	CategoryID       string                     `json:"category_id"`
	CategoryName     string                     `json:"category_name"`
	CategoryColor    string                     `json:"category_color"`
	CategoryIcon     string                     `json:"category_icon"`
	CurrentSpend     float64                    `json:"current_spend"`
	PreviousSpend    float64                    `json:"previous_spend"`
	ThreeMonthAvg    float64                    `json:"three_month_avg"`
	DeltaAmount      float64                    `json:"delta_amount"`
	PercentageChange float64                    `json:"percentage_change"`
	Trend            string                     `json:"trend"` // "UP", "DOWN", "FLAT"
	History          []MonthlyCategoryDataPoint `json:"history"`
	TopPayees        []PayeeSpend               `json:"top_payees,omitempty"`
}

type CashFlowIntelligenceSummary struct {
	TotalInflow         float64 `json:"total_inflow"`
	TotalOutflow        float64 `json:"total_outflow"`
	NetSurplus          float64 `json:"net_surplus"`
	SavingsRate         float64 `json:"savings_rate"`
	FixedNeedsSpend     float64 `json:"fixed_needs_spend"`
	DiscretionarySpend  float64 `json:"discretionary_spend"`
	CreditCardSharePct  float64 `json:"credit_card_share_pct"`
	DirectBankSharePct  float64 `json:"direct_bank_share_pct"`
	TopSpendingCategory string  `json:"top_spending_category"`
	TopSpendingAmount   float64 `json:"top_spending_amount"`
}

type CashFlowIntelligenceResponse struct {
	Period              string                      `json:"period"` // e.g. "2026-03", "ALL", "FY-2025-26"
	SelectedMonth       string                      `json:"selected_month"`
	PreviousMonth       string                      `json:"previous_month"`
	AvailableMonths     []string                    `json:"available_months"`
	Summary             CashFlowIntelligenceSummary `json:"summary"`
	Sankey              SankeyData                  `json:"sankey"`
	Anomalies           []MoMAnomaly                `json:"anomalies"`
	CategoryComparisons []CategoryComparisonItem    `json:"category_comparisons"`
}

type WrappedMerchantHighlight struct {
	Payee       string  `json:"payee"`
	TotalSpent  float64 `json:"total_spent"`
	OrderCount  int     `json:"order_count"`
	PaymentMode string  `json:"payment_mode"`
}

type WrappedStory struct {
	Year               string                     `json:"year"`
	AvailableYears     []string                   `json:"available_years"`
	TotalIncome        float64                    `json:"total_income"`
	TotalExpense       float64                    `json:"total_expense"`
	NetSavings         float64                    `json:"net_savings"`
	SavingsRate        float64                    `json:"savings_rate"`
	TotalTransactions  int                        `json:"total_transactions"`
	PersonaTitle       string                     `json:"persona_title"`
	PersonaBadge       string                     `json:"persona_badge"`
	PersonaDescription string                     `json:"persona_description"`
	CrownMerchant      *WrappedMerchantHighlight  `json:"crown_merchant,omitempty"`
	TopMerchants       []WrappedMerchantHighlight `json:"top_merchants"`
	BiggestPurchase    *Transaction               `json:"biggest_purchase,omitempty"`
	BusiestDay         string                     `json:"busiest_day,omitempty"` // YYYY-MM-DD
	BusiestDaySpend    float64                    `json:"busiest_day_spend,omitempty"`
	BusiestDayTxCount  int                        `json:"busiest_day_tx_count,omitempty"`
	TotalCashback      float64                    `json:"total_cashback"`
	TotalRewardPoints  int                        `json:"total_reward_points"`
	TopCategories      []CategorySpend            `json:"top_categories"`
	UPITxCount         int                        `json:"upi_tx_count"`
	CardTxCount        int                        `json:"card_tx_count"`
	TotalCardSpend     float64                    `json:"total_card_spend"`
	TotalUPISpend      float64                    `json:"total_upi_spend"`
}

// ==========================================
// LOCAL AUTH & APP SECURITY MODELS
// ==========================================

type SecuritySettings struct {
	AuthEnabled     bool      `json:"auth_enabled"`
	AutoLockMinutes int       `json:"auto_lock_minutes"`
	HasPassword     bool      `json:"has_password"`
	FilePermissions string    `json:"file_permissions"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AuthStatusResponse struct {
	AuthEnabled     bool   `json:"auth_enabled"`
	IsAuthenticated bool   `json:"is_authenticated"`
	AutoLockMinutes int    `json:"auto_lock_minutes"`
	FilePermissions string `json:"file_permissions"`
}

type AuthSetupRequest struct {
	Password        string `json:"password" binding:"required"`
	AutoLockMinutes int    `json:"auto_lock_minutes"`
}

type AuthLoginRequest struct {
	Password string `json:"password" binding:"required"`
}

type AuthChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

type AuthDisableRequest struct {
	Password string `json:"password" binding:"required"`
}

type UpdateSecuritySettingsRequest struct {
	AutoLockMinutes int `json:"auto_lock_minutes"`
}

// ==========================================
// SALARY & INCOME INSIGHTS MODELS
// ==========================================

type SalaryYearSummary struct {
	Year           int     `json:"year"`
	TotalEarned    float64 `json:"total_earned"`
	MonthlyAverage float64 `json:"monthly_average"`
	PaycheckCount  int     `json:"paycheck_count"`
	YoYGrowthPct   float64 `json:"yoy_growth_pct"`
}

type SalaryMonthlyDataPoint struct {
	Month    string  `json:"month"` // "YYYY-MM"
	Amount   float64 `json:"amount"`
	Employer string  `json:"employer"`
	IsHike   bool    `json:"is_hike"`
	HikePct  float64 `json:"hike_pct,omitempty"`
	IsBonus  bool    `json:"is_bonus"`
}

type EmployerSummary struct {
	EmployerName   string  `json:"employer_name"`
	TotalEarned    float64 `json:"total_earned"`
	FirstPaycheck  string  `json:"first_paycheck"`
	LastPaycheck   string  `json:"last_paycheck"`
	PaycheckCount  int     `json:"paycheck_count"`
	MonthlyAverage float64 `json:"monthly_average"`
}

type SalaryInsightsResponse struct {
	LifetimeEarned     float64                  `json:"lifetime_earned"`
	TotalPaychecks     int                      `json:"total_paychecks"`
	FirstSalaryAmount  float64                  `json:"first_salary_amount"`
	FirstSalaryDate    string                   `json:"first_salary_date"`
	LatestSalaryAmount float64                  `json:"latest_salary_amount"`
	LatestSalaryDate   string                   `json:"latest_salary_date"`
	LatestEmployer     string                   `json:"latest_employer"`
	AverageMonthly     float64                  `json:"average_monthly"`
	PeakSalaryAmount   float64                  `json:"peak_salary_amount"`
	PeakSalaryDate     string                   `json:"peak_salary_date"`
	PeakEmployer       string                   `json:"peak_employer"`
	OverallGrowthPct   float64                  `json:"overall_growth_pct"`
	CurrentYearEarned  float64                  `json:"current_year_earned"`
	CurrentFYEarned    float64                  `json:"current_fy_earned"`
	YearlyProgress     []SalaryYearSummary      `json:"yearly_progress"`
	MonthlyHistory     []SalaryMonthlyDataPoint `json:"monthly_history"`
	Employers          []EmployerSummary        `json:"employers"`
	RecentPaychecks    []Transaction            `json:"recent_paychecks"`
}
