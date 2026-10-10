package service

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"local-finance/internal/db"
	"local-finance/internal/models"
)

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	s = strings.Split(s, "T")[0]
	s = strings.Split(s, " ")[0]
	return time.Parse("2006-01-02", s)
}

type ReconciliationService struct {
	db *db.DB
}

func NewReconciliationService(database *db.DB) *ReconciliationService {
	return &ReconciliationService{db: database}
}

var (
	ccBillPaymentPayeeRegex = regexp.MustCompile(`(?i)(\bCRED\b|\bBILLDESK\b|(?:CREDIT\s*CARD|CC|CARD)\s*(?:BILL\s*)?PAYMENT|AUTODEBIT\s*CC|BBPS\s*CC|AUTO\s*DEBIT|PAYMENT\s+(?:RECEIVED|THANKS))`)
	cardRefundRegex         = regexp.MustCompile(`(?i)\b(?:REFUND|REVERSAL|REVERSED|CASHBACK|REWARDS?)\b`)
	walletLoadRegex         = regexp.MustCompile(`(?i)(UPI\s*LITE|PAYTM\s*WALLET|AMAZON\s*PAY\s*WALLET|MOBIKWIK|FREECHARGE\s*WALLET|WALLET\s*LOAD|WALLET\s*TOPUP|LITE\s*LOAD)`)
)

const (
	autoTransferConfidence         = 0.85
	suggestionConfidence           = 0.70
	exactAmountConfidenceBonus     = 0.15
	paymentEvidenceConfidenceBonus = 0.15
	nearDateConfidenceBonus        = 0.05
)

func (s *ReconciliationService) GetSummary() (*models.ReconciliationSummary, error) {
	// 1. Fetch Confirmed Pairs
	pairs, err := s.getConfirmedPairs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch confirmed pairs: %w", err)
	}
	if pairs == nil {
		pairs = []models.TransferPair{}
	}

	// 2. Fetch Candidates
	candidates, err := s.getCandidatePairs()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch candidate pairs: %w", err)
	}
	if candidates == nil {
		candidates = []models.TransferPair{}
	}

	// 3. Fetch Excluded / Wallet Transactions
	walletTxs, err := s.getWalletTransactions()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallet transactions: %w", err)
	}
	if walletTxs == nil {
		walletTxs = []models.Transaction{}
	}

	totalPairedAmount := 0.0
	for _, p := range pairs {
		totalPairedAmount += p.DebitTx.Amount
	}

	totalWalletAmount := 0.0
	for _, w := range walletTxs {
		totalWalletAmount += w.Amount
	}

	return &models.ReconciliationSummary{
		TotalPairedTransfers:       len(pairs),
		TotalPairedAmount:          totalPairedAmount,
		PendingCandidatesCount:     len(candidates),
		DoubleCountPreventedAmount: totalPairedAmount,
		WalletExcludedCount:        len(walletTxs),
		WalletExcludedAmount:       totalWalletAmount,
		Pairs:                      pairs,
		Candidates:                 candidates,
		WalletTransactions:         walletTxs,
	}, nil
}

func (s *ReconciliationService) ScanAndAutoReconcile() (int, int, error) {
	// 1. Fetch unlinked Credit Card Payments from Savings / Current Accounts
	candidates, err := s.getCandidatePairs()
	if err != nil {
		return 0, 0, err
	}

	autoLinkedCount := 0
	for _, c := range candidates {
		// Auto-link if high confidence (>= 0.85)
		if c.MatchConfidence >= autoTransferConfidence {
			if err := s.LinkPair(c.DebitTx.ID, c.CreditTx.ID, c.MatchReason); err == nil {
				autoLinkedCount++
			}
		}
	}

	// 2. Auto-exclude wallet load transactions
	walletExcludedCount, err := s.autoExcludeWalletLoads()
	if err != nil {
		return autoLinkedCount, 0, err
	}

	return autoLinkedCount, walletExcludedCount, nil
}

func (s *ReconciliationService) LinkPair(debitTxID, creditTxID, reason string) error {
	if reason == "" {
		reason = "MANUAL_PAIR"
	}
	return s.db.LinkTransferPair(debitTxID, creditTxID, reason)
}

func (s *ReconciliationService) UnlinkPair(txID string) error {
	return s.db.UnlinkTransferPair(txID)
}

func (s *ReconciliationService) getConfirmedPairs() ([]models.TransferPair, error) {
	// Query pairs where transfer_peer_id is set
	query := `
		SELECT 
			d.id, d.account_id, da.bank_name || ' (' || da.account_type || ')',
			d.tx_date, COALESCE(d.raw_narration, ''), COALESCE(d.cleaned_payee, ''), d.amount, d.transfer_match_reason,
			c.id, c.account_id, ca.bank_name || ' (' || ca.account_type || ')',
			c.tx_date, COALESCE(c.raw_narration, ''), COALESCE(c.cleaned_payee, ''), c.amount
		FROM transactions d
		JOIN transactions c ON d.transfer_peer_id = c.id
		JOIN accounts da ON d.account_id = da.id
		JOIN accounts ca ON c.account_id = ca.id
		WHERE d.tx_type = 'DEBIT' AND c.tx_type = 'CREDIT' AND d.transfer_peer_id IS NOT NULL
		ORDER BY d.tx_date DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pairs := make([]models.TransferPair, 0)
	for rows.Next() {
		var p models.TransferPair
		var reason sql.NullString
		if err := rows.Scan(
			&p.DebitTx.ID, &p.DebitTx.AccountID, &p.DebitTx.AccountName,
			&p.DebitTx.TxDate, &p.DebitTx.RawNarration, &p.DebitTx.CleanedPayee, &p.DebitTx.Amount, &reason,
			&p.CreditTx.ID, &p.CreditTx.AccountID, &p.CreditTx.AccountName,
			&p.CreditTx.TxDate, &p.CreditTx.RawNarration, &p.CreditTx.CleanedPayee, &p.CreditTx.Amount,
		); err != nil {
			return nil, err
		}

		p.ID = fmt.Sprintf("%s_%s", p.DebitTx.ID, p.CreditTx.ID)
		p.IsConfirmed = true
		if reason.Valid {
			p.MatchReason = reason.String
		} else {
			p.MatchReason = "CC_BILL_PAYMENT_MATCH"
		}
		p.MatchConfidence = 1.0

		// Date difference
		dDate, _ := parseDate(p.DebitTx.TxDate)
		cDate, _ := parseDate(p.CreditTx.TxDate)
		p.DateDifferenceDays = int(math.Abs(cDate.Sub(dDate).Hours() / 24))
		p.AmountDifference = math.Abs(p.DebitTx.Amount - p.CreditTx.Amount)

		pairs = append(pairs, p)
	}

	return pairs, nil
}

func (s *ReconciliationService) getCandidatePairs() ([]models.TransferPair, error) {
	// Query unlinked debits from Savings/Current and unlinked credits from Credit Card accounts
	debitQuery := `
		SELECT t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')', t.tx_date, COALESCE(t.raw_narration, ''), COALESCE(t.cleaned_payee, ''), t.amount
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.tx_type = 'DEBIT' AND (t.transfer_peer_id IS NULL OR t.transfer_peer_id = '') AND a.account_type IN ('SAVINGS', 'CURRENT')
		AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=t.id)
		ORDER BY t.tx_date DESC
		LIMIT 200
	`
	creditQuery := `
		SELECT t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')', t.tx_date, COALESCE(t.raw_narration, ''), COALESCE(t.cleaned_payee, ''), t.amount
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.tx_type = 'CREDIT' AND (t.transfer_peer_id IS NULL OR t.transfer_peer_id = '') AND a.account_type = 'CREDIT_CARD'
		AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=t.id)
		ORDER BY t.tx_date DESC
		LIMIT 200
	`

	dRows, err := s.db.Query(debitQuery)
	if err != nil {
		return nil, err
	}
	var debits []models.Transaction
	for dRows.Next() {
		var t models.Transaction
		if err := dRows.Scan(&t.ID, &t.AccountID, &t.AccountName, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount); err == nil {
			debits = append(debits, t)
		}
	}
	dRows.Close()

	cRows, err := s.db.Query(creditQuery)
	if err != nil {
		return nil, err
	}
	var credits []models.Transaction
	for cRows.Next() {
		var t models.Transaction
		if err := cRows.Scan(&t.ID, &t.AccountID, &t.AccountName, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount); err == nil {
			credits = append(credits, t)
		}
	}
	cRows.Close()

	candidates := make([]models.TransferPair, 0)

	for _, d := range debits {
		dDate, err := parseDate(d.TxDate)
		if err != nil {
			continue
		}

		isDebitPayeeMatch := ccBillPaymentPayeeRegex.MatchString(d.RawNarration) || ccBillPaymentPayeeRegex.MatchString(d.CleanedPayee)

		for _, c := range credits {
			cDate, err := parseDate(c.TxDate)
			if err != nil {
				continue
			}

			// Date difference within 4 days
			dayDiff := math.Abs(cDate.Sub(dDate).Hours() / 24)
			if dayDiff > 4 {
				continue
			}

			// Amount difference within ₹1
			amtDiff := math.Abs(d.Amount - c.Amount)
			if amtDiff > 1.0 {
				continue
			}

			isCreditPayeeMatch := ccBillPaymentPayeeRegex.MatchString(c.RawNarration) || ccBillPaymentPayeeRegex.MatchString(c.CleanedPayee)

			confidence := suggestionConfidence
			if amtDiff < 0.05 {
				confidence += exactAmountConfidenceBonus
			}
			if isDebitPayeeMatch || isCreditPayeeMatch {
				confidence += paymentEvidenceConfidenceBonus
			}
			if dayDiff <= 1 {
				confidence += nearDateConfidenceBonus
			}

			// An amount/date coincidence alone is never sufficient evidence
			// of an internal payment. Refunds must remain financial activity.
			if !(isDebitPayeeMatch || isCreditPayeeMatch) || cardRefundRegex.MatchString(c.RawNarration+" "+c.CleanedPayee) {
				confidence = suggestionConfidence
			}
			if confidence > 1.0 {
				confidence = 1.0
			}

			candidates = append(candidates, models.TransferPair{
				ID:                 fmt.Sprintf("cand_%s_%s", d.ID, c.ID),
				DebitTx:            d,
				CreditTx:           c,
				MatchConfidence:    confidence,
				MatchReason:        "CC_BILL_PAYMENT_MATCH",
				IsConfirmed:        false,
				DateDifferenceDays: int(dayDiff),
				AmountDifference:   amtDiff,
			})
		}
	}

	// Assess ambiguity before selecting any pairs. Neither a refund nor a
	// low-confidence suggestion should consume a stronger payment candidate.
	strongDebits, strongCredits := map[string]int{}, map[string]int{}
	for _, pair := range candidates {
		if pair.MatchConfidence >= autoTransferConfidence {
			strongDebits[pair.DebitTx.ID]++
			strongCredits[pair.CreditTx.ID]++
		}
	}
	for i := range candidates {
		pair := &candidates[i]
		if strongDebits[pair.DebitTx.ID] > 1 || strongCredits[pair.CreditTx.ID] > 1 {
			pair.MatchConfidence = suggestionConfidence
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.MatchConfidence != b.MatchConfidence {
			return a.MatchConfidence > b.MatchConfidence
		}
		if a.DateDifferenceDays != b.DateDifferenceDays {
			return a.DateDifferenceDays < b.DateDifferenceDays
		}
		if a.AmountDifference != b.AmountDifference {
			return a.AmountDifference < b.AmountDifference
		}
		return a.ID < b.ID
	})
	// Keep competing suggestions visible for manual selection. Automatic
	// linking sees only unambiguous high-confidence pairs, and LinkPair also
	// validates reciprocal one-to-one relationships atomically.
	return candidates, nil
}

func (s *ReconciliationService) getWalletTransactions() ([]models.Transaction, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.account_id, a.bank_name || ' (' || a.account_type || ')', t.tx_date, t.raw_narration, t.cleaned_payee, t.amount, t.is_excluded, t.is_transfer
		FROM transactions t
		JOIN accounts a ON t.account_id = a.id
		WHERE t.raw_narration LIKE '%UPI LITE%' OR t.raw_narration LIKE '%PAYTM WALLET%' OR t.raw_narration LIKE '%AMAZON PAY WALLET%' OR t.cleaned_payee LIKE '%UPI Lite%' OR t.cleaned_payee LIKE '%Wallet%'
		ORDER BY t.tx_date DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]models.Transaction, 0)
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.AccountName, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount, &t.IsExcluded, &t.IsTransfer); err == nil {
			list = append(list, t)
		}
	}
	return list, nil
}

func (s *ReconciliationService) autoExcludeWalletLoads() (int, error) {
	res, err := s.db.Exec(`
		UPDATE transactions
		SET is_excluded = 1
		WHERE (raw_narration LIKE '%UPI LITE%' OR raw_narration LIKE '%PAYTM WALLET%' OR raw_narration LIKE '%WALLET TOPUP%' OR raw_narration LIKE '%WALLET LOAD%')
		  AND is_excluded = 0
		  AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=transactions.id)
	`)
	if err != nil {
		return 0, err
	}
	rowsAffected, _ := res.RowsAffected()
	return int(rowsAffected), nil
}
