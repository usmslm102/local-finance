package parser

import (
	"fmt"
	"regexp"
	"strings"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

// Portal downloads print the bank logo as an image. Recognize the complete
// layout signature rather than assigning generic card headings to ICICI.
func isICICICCPortalStatement(text string) bool {
	text = strings.ToUpper(strings.Join(strings.Fields(text), " "))
	return containsAllBankTerms(text, []string{"VIEW LAST STATEMENT", "CARD HOLDER NAME", "CARD TYPE", "PREVIOUS BALANCE", "PURCHASES AND OTHER CHARGES", "PAYMENTS AND OTHER CREDITS", "STATEMENT PERIOD", "TRANSACTION DATE", "DETAILS", "AMOUNT (INR)", "REWARD POINTS", "REFERENCE NUMBER"})
}

var iciciPortalDate = regexp.MustCompile(`^\d{2}-\d{2}-\d{4}$`)
var iciciPortalAmount = regexp.MustCompile(`(?i)^([0-9][0-9,]*(?:\.\d{2})?)\s+(DR|CR)\.$`)
var iciciPortalPeriod = regexp.MustCompile(`(?i)^(\d{2}-\d{2}-\d{4})\s+TO\s+(\d{2}-\d{2}-\d{4})$`)

func parseICICICCPortalRows(rows []extractor.PositionalRow) ([]ParsedTransaction, StatementMeta, error) {
	meta := StatementMeta{BankName: "ICICI Bank", AccountType: models.AccountTypeCreditCard, StatementFormat: "PDF", RequiresAccountSelection: true}
	values := map[string]string{}
	labels := []string{"Card Holder Name", "Card Type", "Statement Date", "Payment Due Date", "Statement Period", "Previous Balance", "Purchases and Other Charges", "Payments and Other Credits", "Total Amount Due", "Minimum Amount Due", "Total Credit Limit", "Available Credit Limit", "Total Cash Limit", "Cash Advances"}
	for _, row := range rows {
		line := bankHistoryRowText(row)
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "TRANSACTION DATE") && strings.Contains(upper, "REFERENCE NUMBER") {
			break
		}
		for _, label := range labels {
			if start := strings.Index(upper, strings.ToUpper(label)); start >= 0 {
				start += len(label)
				end := len(line)
				for _, other := range labels {
					if at := strings.Index(upper[start:], strings.ToUpper(other)); at >= 0 && start+at < end {
						end = start + at
					}
				}
				value := strings.TrimSpace(line[start:end])
				if old, ok := values[label]; ok && old != value {
					return nil, meta, fmt.Errorf("ICICI portal statement has conflicting %s", label)
				}
				values[label] = value
			}
		}
	}
	meta.AccountHolderName = extractor.CleanCustomerName(values["Card Holder Name"])
	meta.StatementDate = strictIndianDate(values["Statement Date"])
	meta.PaymentDueDate = strictIndianDate(values["Payment Due Date"])
	period := iciciPortalPeriod.FindStringSubmatch(values["Statement Period"])
	if len(period) != 3 || meta.StatementDate == "" || meta.PaymentDueDate == "" {
		return nil, meta, fmt.Errorf("ICICI portal statement has missing or invalid statement dates")
	}
	meta.StartDate, meta.EndDate = strictIndianDate(period[1]), strictIndianDate(period[2])
	if meta.StartDate == "" || meta.EndDate == "" || meta.StartDate > meta.EndDate {
		return nil, meta, fmt.Errorf("ICICI portal statement has invalid statement period")
	}
	for _, field := range []struct {
		label  string
		target *float64
	}{
		{"Previous Balance", &meta.OpeningBalance}, {"Purchases and Other Charges", &meta.TotalDebits}, {"Payments and Other Credits", &meta.TotalCredits},
		{"Total Amount Due", &meta.TotalDueAmount}, {"Minimum Amount Due", &meta.MinimumDueAmount}, {"Total Credit Limit", &meta.CreditLimit}, {"Available Credit Limit", &meta.AvailableCreditLimit},
	} {
		raw := strings.TrimSpace(strings.TrimPrefix(values[field.label], "INR"))
		if !bankHistoryAmount.MatchString(raw) {
			return nil, meta, fmt.Errorf("ICICI portal statement has missing or invalid %s", field.label)
		}
		amount, err := extractor.ParseIndianAmount(raw)
		if err != nil {
			return nil, meta, err
		}
		*field.target = amount
	}
	// This portal layout does not print a card number, network or product.
	// Leave those unset rather than inventing an Amazon Pay/VISA identity.
	var txs []ParsedTransaction
	var debitTotal, creditTotal float64
	var dateX, detailsX, amountX, rewardX, referenceX float64
	active := false
	for _, row := range rows {
		if strings.Contains(strings.ToUpper(bankHistoryRowText(row)), "TRANSACTION DATE") && strings.Contains(strings.ToUpper(bankHistoryRowText(row)), "REFERENCE NUMBER") {
			positions := map[string]float64{}
			for _, el := range row.Elements {
				label := strings.ToUpper(strings.TrimSpace(el.S))
				if _, exists := positions[label]; exists {
					return nil, meta, fmt.Errorf("ICICI portal statement has ambiguous columns")
				}
				positions[label] = el.X
			}
			var ok bool
			for _, field := range []struct {
				label  string
				target *float64
			}{{"TRANSACTION DATE", &dateX}, {"DETAILS", &detailsX}, {"AMOUNT (INR)", &amountX}, {"REWARD POINTS", &rewardX}, {"REFERENCE NUMBER", &referenceX}} {
				*field.target, ok = positions[field.label]
				if !ok {
					*field.target, ok = positions[strings.Fields(field.label)[0]]
				}
				if !ok {
					return nil, meta, fmt.Errorf("ICICI portal statement has incomplete table headings")
				}
			}
			if !(dateX < detailsX && detailsX < amountX && amountX < rewardX && rewardX < referenceX) {
				return nil, meta, fmt.Errorf("ICICI portal statement has invalid column order")
			}
			active = true
			continue
		}
		if !active {
			continue
		}
		dateCell := bankHistoryCellInColumn(row, dateX-2, detailsX)
		if dateCell == "" {
			if len(txs) > 0 && len(row.Elements) > 0 {
				continuation := true
				for _, el := range row.Elements {
					if el.X < detailsX || el.X >= amountX {
						continuation = false
					}
				}
				if continuation {
					txs[len(txs)-1].RawNarration = strings.TrimSpace(txs[len(txs)-1].RawNarration + " " + bankHistoryCellInColumn(row, detailsX, amountX))
				}
			}
			continue
		}
		if !iciciPortalDate.MatchString(dateCell) || strictIndianDate(dateCell) == "" {
			return nil, meta, fmt.Errorf("ICICI portal statement contains an invalid transaction date")
		}
		date := strictIndianDate(dateCell)
		if date < meta.StartDate || date > meta.EndDate {
			return nil, meta, fmt.Errorf("ICICI portal transaction lies outside the statement period")
		}
		narration := bankHistoryCellInColumn(row, detailsX, amountX)
		match := iciciPortalAmount.FindStringSubmatch(bankHistoryCellInColumn(row, amountX, rewardX))
		if narration == "" || len(match) != 3 {
			return nil, meta, fmt.Errorf("ICICI portal statement contains an incomplete transaction at row %d", len(txs)+1)
		}
		amount, err := extractor.ParseIndianAmount(match[1])
		if err != nil || amount <= 0 {
			return nil, meta, fmt.Errorf("ICICI portal statement has an invalid transaction amount")
		}
		reference := bankHistoryCellInColumn(row, referenceX, 0)
		if reference == "" || !isDigitsOnly(reference) {
			return nil, meta, fmt.Errorf("ICICI portal statement has an invalid reference number")
		}
		rewardsRaw := bankHistoryCellInColumn(row, rewardX, referenceX)
		if !bankHistoryAmount.MatchString(rewardsRaw) {
			return nil, meta, fmt.Errorf("ICICI portal statement has invalid reward points")
		}
		rewards, err := extractor.ParseIndianAmount(rewardsRaw)
		if err != nil {
			return nil, meta, err
		}
		direction := models.TxTypeDebit
		if strings.EqualFold(match[2], "CR") {
			direction = models.TxTypeCredit
			creditTotal += amount
		} else {
			debitTotal += amount
		}
		txs = append(txs, ParsedTransaction{Date: date, RawNarration: narration, ReferenceNumber: reference, Amount: amount, TxType: direction, RewardPointsEarned: rewards, PaymentMode: models.PaymentModeCardPOS})
	}
	if len(txs) == 0 || !historyMoneyEqual(debitTotal, meta.TotalDebits) || !historyMoneyEqual(creditTotal, meta.TotalCredits) || !historyMoneyEqual(meta.OpeningBalance+debitTotal-creditTotal, meta.TotalDueAmount) {
		return nil, meta, fmt.Errorf("ICICI portal transactions do not reconcile with the statement totals")
	}
	for i := range txs {
		cleaned := CleanNarration(txs[i].RawNarration)
		txs[i].CleanedPayee = cleaned.CleanedPayee
		txs[i].UPIVPA = cleaned.UPIVPA
		txs[i].CardLast4 = cleaned.CardLast4
		txs[i].IsTransfer = cleaned.IsTransfer || txs[i].TxType == models.TxTypeCredit && strings.Contains(strings.ToUpper(txs[i].RawNarration), "PAYMENT RECEIVED")
	}
	return txs, meta, nil
}
