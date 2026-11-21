package service

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"local-finance/internal/db"
	"local-finance/internal/models"
)

type SubscriptionService struct {
	db *db.DB
}

func NewSubscriptionService(database *db.DB) *SubscriptionService {
	return &SubscriptionService{
		db: database,
	}
}

type CatalogRule struct {
	Name            string
	MerchantPattern string
	Regex           *regexp.Regexp
	Frequency       models.SubscriptionFrequency
	DefaultCategory string
}

var knownCatalog = []CatalogRule{
	{
		Name:            "Netflix",
		MerchantPattern: "NETFLIX",
		Regex:           regexp.MustCompile(`(?i)\bNETFLIX\b`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "YouTube Premium",
		MerchantPattern: "YOUTUBE",
		Regex:           regexp.MustCompile(`(?i)\bYOUTUBE\b`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "Apple Media & Services",
		MerchantPattern: "APPLEMEDIA",
		Regex:           regexp.MustCompile(`(?i)(APPLEMEDIA|APPLE SERVICES|ITUNES\.COM|APPLE\.COM)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Electronics & Tech",
	},
	{
		Name:            "Spotify",
		MerchantPattern: "SPOTIFY",
		Regex:           regexp.MustCompile(`(?i)\bSPOTIFY\b`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "Amazon Prime",
		MerchantPattern: "AMAZON PRIME",
		Regex:           regexp.MustCompile(`(?i)(AMAZON PRIME|PRIME VIDEO|AMAZON DIGITAL)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "Google One & Cloud",
		MerchantPattern: "GOOGLE ONE",
		Regex:           regexp.MustCompile(`(?i)(GOOGLE STORAGE|GOOGLE ONE|GOOGLE CLOUD|GOOGLE PLAY)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Electronics & Tech",
	},
	{
		Name:            "ChatGPT / OpenAI",
		MerchantPattern: "OPENAI",
		Regex:           regexp.MustCompile(`(?i)(OPENAI|CHATGPT)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Electronics & Tech",
	},
	{
		Name:            "GitHub",
		MerchantPattern: "GITHUB",
		Regex:           regexp.MustCompile(`(?i)\bGITHUB\b`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Electronics & Tech",
	},
	{
		Name:            "Disney+ Hotstar",
		MerchantPattern: "HOTSTAR",
		Regex:           regexp.MustCompile(`(?i)\bHOTSTAR\b`),
		Frequency:       models.FrequencyYearly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "SonyLIV",
		MerchantPattern: "SONYLIV",
		Regex:           regexp.MustCompile(`(?i)\bSONYLIV\b`),
		Frequency:       models.FrequencyYearly,
		DefaultCategory: "Entertainment",
	},
	{
		Name:            "Airtel Broadband / Postpaid",
		MerchantPattern: "AIRTEL",
		Regex:           regexp.MustCompile(`(?i)(AIRTEL POSTPAID|AIRTEL BROADBAND|AIRTEL FIBER|BHARTI AIRTEL)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Bills & Utilities",
	},
	{
		Name:            "JioFiber / Postpaid",
		MerchantPattern: "JIOFIBER",
		Regex:           regexp.MustCompile(`(?i)(JIO FIBER|JIO POSTPAID|RELIANCE JIO)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Bills & Utilities",
	},
	{
		Name:            "ACT Fibernet",
		MerchantPattern: "ACT FIBERNET",
		Regex:           regexp.MustCompile(`(?i)(ACT FIBERNET|ACT BROADBAND)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Bills & Utilities",
	},
	{
		Name:            "Tata Play DTH",
		MerchantPattern: "TATA PLAY",
		Regex:           regexp.MustCompile(`(?i)(TATA PLAY|TATA SKY)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Bills & Utilities",
	},
	{
		Name:            "Cult.fit / Curefit",
		MerchantPattern: "CULT.FIT",
		Regex:           regexp.MustCompile(`(?i)(CULT\.FIT|CUREFIT|CULT FIT)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Personal Care",
	},
	{
		Name:            "Credit Card / Loan EMI",
		MerchantPattern: "EMI",
		Regex:           regexp.MustCompile(`(?i)(EMI PRINCIPAL|OFFUS EMI|EMI UPI|EMI\b)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Financial",
	},
	{
		Name:            "UPI AutoPay Mandate",
		MerchantPattern: "AUTOPAY",
		Regex:           regexp.MustCompile(`(?i)(UPI-AUTOPAY|MANDATEEXECUTE|EXECUTIONTEST)`),
		Frequency:       models.FrequencyMonthly,
		DefaultCategory: "Bills & Utilities",
	},
}

func (s *SubscriptionService) ScanAndDetectSubscriptions() (*models.SubscriptionsSummary, error) {
	if err := s.DetectSubscriptions(); err != nil {
		return nil, err
	}
	return s.db.GetSubscriptionsSummary()
}

// DetectSubscriptions keeps per-import cadence semantics but avoids presentation reads.
func (s *SubscriptionService) DetectSubscriptions() error {
	txs, err := s.db.ListSubscriptionCandidates()
	if err != nil {
		return fmt.Errorf("failed to list transactions for subscription scan: %w", err)
	}

	// 2. Fetch existing categories and subscriptions
	categories, err := s.db.ListCategories()
	if err != nil {
		return err
	}
	catMap := make(map[string]string)
	for _, c := range categories {
		catMap[strings.ToLower(c.Name)] = c.ID
	}

	existingSubs, err := s.db.ListSubscriptions()
	if err != nil {
		return err
	}
	existingMap := make(map[string]models.Subscription)
	for _, sub := range existingSubs {
		existingMap[strings.ToUpper(sub.MerchantPattern)] = sub
		existingMap[strings.ToUpper(sub.Name)] = sub
	}

	var recurringTxIDs []string

	// Map of pattern -> slice of transactions
	detectedGroups := make(map[string][]models.Transaction)
	detectedNames := make(map[string]string)
	detectedCategories := make(map[string]string)
	detectedFreqs := make(map[string]models.SubscriptionFrequency)

	// 3. Scan with Catalog
	for _, tx := range txs {
		if tx.IsTransfer || tx.IsExcluded {
			continue
		}

		fullText := tx.CleanedPayee + " " + tx.RawNarration

		matched := false
		for _, rule := range knownCatalog {
			if rule.Regex.MatchString(fullText) {
				detectedGroups[rule.MerchantPattern] = append(detectedGroups[rule.MerchantPattern], tx)
				detectedNames[rule.MerchantPattern] = rule.Name
				detectedCategories[rule.MerchantPattern] = rule.DefaultCategory
				detectedFreqs[rule.MerchantPattern] = rule.Frequency
				recurringTxIDs = append(recurringTxIDs, tx.ID)
				matched = true
				break
			}
		}

		// Also check standalone AUTOPAY keywords
		if !matched && (strings.Contains(strings.ToUpper(tx.RawNarration), "AUTOPAY") ||
			strings.Contains(strings.ToUpper(tx.RawNarration), "MANDATEEXECUTE")) {
			payeeKey := strings.ToUpper(strings.TrimSpace(tx.CleanedPayee))
			if payeeKey == "" {
				payeeKey = "AUTOPAY"
			}
			detectedGroups[payeeKey] = append(detectedGroups[payeeKey], tx)
			detectedNames[payeeKey] = tx.CleanedPayee
			detectedCategories[payeeKey] = "Bills & Utilities"
			detectedFreqs[payeeKey] = models.FrequencyMonthly
			recurringTxIDs = append(recurringTxIDs, tx.ID)
		}
	}

	// 4. Statistical Cadence Engine for remaining uncatalogued payees
	payeeGroups := make(map[string][]models.Transaction)
	for _, tx := range txs {
		if tx.IsTransfer || tx.IsExcluded || tx.CleanedPayee == "" {
			continue
		}
		payeeGroups[strings.ToUpper(tx.CleanedPayee)] = append(payeeGroups[strings.ToUpper(tx.CleanedPayee)], tx)
	}

	for payeeUpper, pTxs := range payeeGroups {
		if _, already := detectedGroups[payeeUpper]; already || len(pTxs) < 2 {
			continue
		}

		// Sort by date ASC
		sort.Slice(pTxs, func(i, j int) bool {
			return pTxs[i].TxDate < pTxs[j].TxDate
		})

		// Check if intervals are consistently monthly (25 - 35 days)
		intervals := []int{}
		for i := 1; i < len(pTxs); i++ {
			d1, e1 := time.Parse("2006-01-02", pTxs[i-1].TxDate)
			d2, e2 := time.Parse("2006-01-02", pTxs[i].TxDate)
			if e1 == nil && e2 == nil {
				days := int(d2.Sub(d1).Hours() / 24)
				intervals = append(intervals, days)
			}
		}

		isMonthly := true
		for _, days := range intervals {
			if days < 25 || days > 35 {
				isMonthly = false
				break
			}
		}

		// Check amount consistency (within 15% variance)
		if isMonthly && len(intervals) >= 1 {
			var sumAmt float64
			for _, t := range pTxs {
				sumAmt += t.Amount
			}
			avgAmt := sumAmt / float64(len(pTxs))
			amountConsistent := true
			for _, t := range pTxs {
				if math.Abs(t.Amount-avgAmt)/avgAmt > 0.15 {
					amountConsistent = false
					break
				}
			}

			if amountConsistent {
				detectedGroups[payeeUpper] = pTxs
				detectedNames[payeeUpper] = pTxs[0].CleanedPayee
				detectedCategories[payeeUpper] = "Bills & Utilities"
				detectedFreqs[payeeUpper] = models.FrequencyMonthly
				for _, t := range pTxs {
					recurringTxIDs = append(recurringTxIDs, t.ID)
				}
			}
		}
	}

	// 5. Mark recurring transactions in database
	if len(recurringTxIDs) > 0 {
		if err := s.db.MarkTransactionsRecurring(recurringTxIDs); err != nil {
			return err
		}
	}

	// 6. Create / Update Subscriptions in database
	for pattern, groupTxs := range detectedGroups {
		if len(groupTxs) == 0 {
			continue
		}

		// Sort DESC to get latest transaction
		sort.Slice(groupTxs, func(i, j int) bool {
			return groupTxs[i].TxDate > groupTxs[j].TxDate
		})

		latestTx := groupTxs[0]
		name := detectedNames[pattern]
		if name == "" {
			name = pattern
		}
		freq := detectedFreqs[pattern]
		if freq == "" {
			freq = models.FrequencyMonthly
		}

		// Parse billing day from latest tx date
		billingDay := 1
		if txD, err := time.Parse("2006-01-02", latestTx.TxDate); err == nil {
			billingDay = txD.Day()
		}

		nextDue := nextDueDateFromLastPaid(latestTx.TxDate, billingDay, freq)

		var catID *string
		if cName, ok := detectedCategories[pattern]; ok {
			if id, exists := catMap[strings.ToLower(cName)]; exists {
				catID = &id
			}
		}
		if catID == nil && latestTx.CategoryID != nil {
			catID = latestTx.CategoryID
		}

		accountID := latestTx.AccountID
		lastPaidAmt := latestTx.Amount
		lastPaidDate := latestTx.TxDate

		if existing, exists := existingMap[pattern]; exists {
			// Update existing subscription record with latest values if active
			if existing.Status == models.SubscriptionStatusActive && existing.IsAutoDetected {
				existing.ExpectedAmount = latestTx.Amount
				existing.LastPaidDate = &lastPaidDate
				existing.LastPaidAmount = &lastPaidAmt
				existing.BillingDay = billingDay
				existing.NextDueDate = &nextDue
				if existing.AccountID == nil {
					existing.AccountID = &accountID
				}
				if err := s.db.UpdateSubscription(&existing); err != nil {
					return err
				}
			}
		} else {
			// Create new subscription record
			sub := &models.Subscription{
				Name:            name,
				MerchantPattern: pattern,
				CategoryID:      catID,
				AccountID:       &accountID,
				Frequency:       freq,
				ExpectedAmount:  latestTx.Amount,
				Currency:        "INR",
				BillingDay:      billingDay,
				NextDueDate:     &nextDue,
				LastPaidDate:    &lastPaidDate,
				LastPaidAmount:  &lastPaidAmt,
				Status:          models.SubscriptionStatusActive,
				IsAutoDetected:  true,
			}
			if err := s.db.CreateSubscription(sub); err != nil {
				return err
			}
		}
	}

	return nil
}

func nextDueDateFromLastPaid(lastPaid string, billingDay int, freq models.SubscriptionFrequency) string {
	t, err := time.Parse("2006-01-02", lastPaid)
	if err != nil {
		return ""
	}

	var next time.Time
	switch freq {
	case models.FrequencyYearly:
		next = t.AddDate(1, 0, 0)
	case models.FrequencyQuarterly:
		next = t.AddDate(0, 3, 0)
	case models.FrequencyWeekly:
		next = t.AddDate(0, 0, 7)
	case models.FrequencyMonthly:
	default:
		next = t.AddDate(0, 1, 0)
	}

	if billingDay > 0 && billingDay <= 31 {
		year, month, _ := next.Date()
		// Make sure day does not overflow month
		maxDays := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		targetDay := billingDay
		if targetDay > maxDays {
			targetDay = maxDays
		}
		next = time.Date(year, month, targetDay, 0, 0, 0, 0, time.UTC)
	}

	return next.Format("2006-01-02")
}
