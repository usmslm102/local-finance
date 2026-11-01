package parser

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

// Shared bank-history parsing utilities. Institution-specific adapters keep
// their own detection and statement profiles; these helpers handle the common
// positional-table parsing and reconciliation mechanics.
//
// bankHistoryProfile describes the adapter-provided headings and controls.
type bankHistoryProfile struct {
	bankName       string
	accountType    models.AccountType
	formatName     string
	recognition    []string
	identityTerm   string
	identityTerms  []string
	statementTerm  string
	statementTerms []string
	dateHeaders    []string
	description    []string
	debitHeaders   []string
	creditHeaders  []string
	balanceHeaders []string
	reference      []string
	accountPattern *regexp.Regexp
	periodPattern  *regexp.Regexp
	openingPattern *regexp.Regexp
	closingPattern *regexp.Regexp
	debitTotal     *regexp.Regexp
	creditTotal    *regexp.Regexp
	stopPattern    *regexp.Regexp
}

type bankHistoryColumns struct {
	serial, mode, valueDate                              float64
	date, description, debit, credit, balance, reference float64
	hasReference                                         bool
}

type bankHistoryRow struct {
	tx      ParsedTransaction
	balance float64
}

type bankHistoryBalances struct {
	openingFound bool
	opening      float64
	closingFound bool
	closing      float64
}

var (
	bankHistoryDatePrefix    = regexp.MustCompile(`^(?:\d{1,2}[./ -](?:\d{1,2}|[A-Za-z]{3,9})[./ -]\d{2,4}|\d{4}[./-]\d{1,2}[./-]\d{1,2})\b`)
	bankHistoryAmount        = regexp.MustCompile(`(?i)^\s*(?:₹|INR\s*)?([0-9][0-9,]*(?:\.\d{2})?)(?:\s*(CR|DR))?\s*$`)
	bankHistorySignedBalance = regexp.MustCompile(`^\s*(?:₹|INR\s*)?(-[0-9][0-9,]*(?:\.\d{2})?)\s*$`)
	bankHistoryMoneyToken    = regexp.MustCompile(`[0-9][0-9,]*\.[0-9]{2}`)
	bankHistoryDatePart      = regexp.MustCompile(`^\d{1,2}[./-](?:\d{1,2}|[A-Za-z]{3,9})[./-]$`)
	bankHistoryYearPart      = regexp.MustCompile(`^\d{4}$`)
	bankHistorySerialDate    = regexp.MustCompile(`^\d{1,6}\s+(\d{1,2}[./-]\d{1,2}[./-]\d{4})$`)
)

func parseBankHistoryPDF(r io.Reader, opts ParseOptions, profile bankHistoryProfile) ([]ParsedTransaction, StatementMeta, error) {
	rows, err := extractor.ExtractPDFPositionalRows(r, opts.Password)
	if err != nil {
		return nil, StatementMeta{}, fmt.Errorf("failed to extract %s statement: %w", profile.bankName, err)
	}
	return parseBankHistoryRows(rows, profile)
}

func parseBankHistoryRows(rows []extractor.PositionalRow, profile bankHistoryProfile) ([]ParsedTransaction, StatementMeta, error) {
	rows = coalesceBankHistoryRows(rows, profile)
	meta := StatementMeta{BankName: profile.bankName, AccountType: profile.accountType, StatementFormat: profile.formatName}
	headerIndex := -1
	var columns bankHistoryColumns
	documentText := strings.ToUpper(bankHistoryDocumentText(rows))
	if isInvestmentStatement(documentText) {
		return nil, meta, fmt.Errorf("investment statements are not supported by the %s bank statement parser", profile.bankName)
	}

	identityTerms := profile.identityTerms
	if len(identityTerms) == 0 && profile.identityTerm != "" {
		identityTerms = []string{profile.identityTerm}
	}
	identityMatches := len(identityTerms) == 0
	for _, term := range identityTerms {
		if strings.Contains(documentText, strings.ToUpper(term)) {
			identityMatches = true
			break
		}
	}

	statementTerms := profile.statementTerms
	if len(statementTerms) == 0 && profile.statementTerm != "" {
		statementTerms = []string{profile.statementTerm}
	}
	statementMatches := len(statementTerms) == 0
	for _, term := range statementTerms {
		if strings.Contains(documentText, strings.ToUpper(term)) {
			statementMatches = true
			break
		}
	}

	if !identityMatches || !statementMatches {
		return nil, meta, fmt.Errorf("document does not match the %s account statement format", profile.bankName)
	}
	for index, row := range rows {
		if candidate, ok := discoverBankHistoryColumns(row, profile); ok {
			headerIndex = index
			columns = candidate
			break
		}
	}
	if headerIndex < 0 {
		return nil, meta, fmt.Errorf("%s statement must contain one unambiguous transaction table", profile.bankName)
	}
	// Resolve only labeled identifiers in the account header, never numbers
	// embedded in transaction descriptions.
	if profile.accountPattern != nil {
		for _, row := range rows[:headerIndex] {
			match := profile.accountPattern.FindStringSubmatch(bankHistoryRowText(row))
			if len(match) < 2 {
				continue
			}
			number := strings.ToUpper(match[1])
			mask := "XX" + number[len(number)-4:]
			if meta.AccountNumberMask != "" && meta.AccountNumberMask != mask {
				return nil, meta, fmt.Errorf("%s statement contains conflicting account identifiers", profile.bankName)
			}
			meta.AccountNumberMask = mask
			if !strings.ContainsAny(number, "X*") {
				if meta.AccountNumber != "" && meta.AccountNumber != number {
					return nil, meta, fmt.Errorf("%s statement contains conflicting account identifiers", profile.bankName)
				}
				meta.AccountNumber = number
			}
		}
	}

	// Extract only explicitly labeled statement controls. They are optional for
	// bank exports that provide no summary; per-row balances still reconcile.
	openingFound, closingFound, debitTotalFound, creditTotalFound := false, false, false, false
	for _, row := range rows {
		line := bankHistoryRowText(row)
		upper := strings.ToUpper(line)
		if profile.periodPattern != nil && meta.StartDate == "" {
			if match := profile.periodPattern.FindStringSubmatch(line); len(match) == 3 {
				meta.StartDate = strictIndianDate(match[1])
				meta.EndDate = strictIndianDate(match[2])
			}
		}
		if profile.openingPattern != nil && strings.Contains(upper, "OPENING BALANCE") {
			amount, ok := parseLabeledHistoryAmount(profile.openingPattern, line)
			if !ok || openingFound && !historyMoneyEqual(meta.OpeningBalance, amount) {
				return nil, meta, fmt.Errorf("%s statement has a malformed or conflicting opening balance", profile.bankName)
			}
			meta.OpeningBalance, openingFound = amount, true
		}
		if profile.closingPattern != nil && strings.Contains(upper, "CLOSING BALANCE") {
			amount, ok := parseLabeledHistoryAmount(profile.closingPattern, line)
			if !ok || closingFound && !historyMoneyEqual(meta.ClosingBalance, amount) {
				return nil, meta, fmt.Errorf("%s statement has a malformed or conflicting closing balance", profile.bankName)
			}
			meta.ClosingBalance, closingFound = amount, true
		}
		if profile.debitTotal != nil && (strings.Contains(upper, "TOTAL DEBIT") || strings.Contains(upper, "TOTAL WITHDRAWAL")) {
			amount, ok := parseLabeledHistoryAmount(profile.debitTotal, line)
			if !ok || debitTotalFound && !historyMoneyEqual(meta.TotalDebits, amount) {
				return nil, meta, fmt.Errorf("%s statement has a malformed or conflicting debit total", profile.bankName)
			}
			meta.TotalDebits, debitTotalFound = amount, true
		}
		if profile.creditTotal != nil && (strings.Contains(upper, "TOTAL CREDIT") || strings.Contains(upper, "TOTAL DEPOSIT")) {
			amount, ok := parseLabeledHistoryAmount(profile.creditTotal, line)
			if !ok || creditTotalFound && !historyMoneyEqual(meta.TotalCredits, amount) {
				return nil, meta, fmt.Errorf("%s statement has a malformed or conflicting credit total", profile.bankName)
			}
			meta.TotalCredits, creditTotalFound = amount, true
		}
	}

	parsed := make([]bankHistoryRow, 0, len(rows)-headerIndex)
	active := true
	for index := headerIndex + 1; index < len(rows); index++ {
		row := rows[index]
		line := bankHistoryRowText(row)
		if candidate, ok := discoverBankHistoryColumns(row, profile); ok {
			columns = candidate
			continue
		}
		if profile.stopPattern != nil && profile.stopPattern.MatchString(line) {
			active = false
		}
		if !active || len(row.Elements) == 0 {
			continue
		}
		dateIndex, dateValue := bankHistoryDateCell(row, columns)
		if dateIndex < 0 {
			if bankHistoryDateAnchor(row, columns) {
				return nil, meta, fmt.Errorf("%s statement contains an incomplete or ambiguous transaction date", profile.bankName)
			}
			if len(parsed) > 0 && bankHistoryContinuation(row, columns) {
				previous := &parsed[len(parsed)-1].tx
				previous.RawNarration = strings.TrimSpace(previous.RawNarration + " " + bankHistoryNarration(row, columns))
				previous.ReferenceNumber += bankHistoryReference(row, columns)
			}
			continue
		}
		date := strictIndianDate(dateValue)
		if date == "" {
			return nil, meta, fmt.Errorf("%s statement contains an invalid transaction date", profile.bankName)
		}
		debitStart, debitEnd := bankHistoryColumnBounds(columns.debit, columns)
		creditStart, creditEnd := bankHistoryColumnBounds(columns.credit, columns)
		balanceStart, balanceEnd := bankHistoryColumnBounds(columns.balance, columns)
		debit, debitOK, err := amountInBankColumn(row, debitStart, debitEnd)
		if err != nil {
			return nil, meta, fmt.Errorf("%s statement contains an ambiguous withdrawal amount", profile.bankName)
		}
		credit, creditOK, err := amountInBankColumn(row, creditStart, creditEnd)
		if err != nil {
			return nil, meta, fmt.Errorf("%s statement contains an ambiguous deposit amount", profile.bankName)
		}
		balance, balanceOK, err := balanceInBankColumn(row, balanceStart, balanceEnd)
		if err != nil || !balanceOK {
			return nil, meta, fmt.Errorf("%s statement contains a missing or ambiguous running balance", profile.bankName)
		}
		if !debitOK && !creditOK {
			return nil, meta, fmt.Errorf("%s statement contains a dated row without a withdrawal or deposit", profile.bankName)
		}
		if debitOK && creditOK && debit > 0 && credit > 0 {
			return nil, meta, fmt.Errorf("%s statement row contains both a withdrawal and deposit", profile.bankName)
		}
		amount, direction := debit, models.TxTypeDebit
		if credit > 0 {
			amount, direction = credit, models.TxTypeCredit
		}
		if amount <= 0 {
			return nil, meta, fmt.Errorf("%s statement contains a zero or negative transaction amount", profile.bankName)
		}
		balanceCell := bankHistoryCellInColumn(row, balanceStart, balanceEnd)
		if strings.HasSuffix(strings.ToUpper(strings.TrimSpace(balanceCell)), "DR") {
			balance = -absFloat(balance)
		} else if strings.HasSuffix(strings.ToUpper(strings.TrimSpace(balanceCell)), "CR") {
			balance = absFloat(balance)
		}
		narration := bankHistoryNarration(row, columns)
		if narration == "" {
			return nil, meta, fmt.Errorf("%s statement contains a dated transaction without narration", profile.bankName)
		}
		ref := bankHistoryReference(row, columns)
		parsed = append(parsed, bankHistoryRow{tx: ParsedTransaction{
			Date: date, RawNarration: narration, ReferenceNumber: ref,
			TxType: direction, Amount: amount,
		}, balance: balance})
		if columns.valueDate >= 0 {
			start, end := bankHistoryColumnBounds(columns.valueDate, columns)
			if rawValueDate := bankHistoryCellInColumn(row, start, end); rawValueDate != "" {
				valueDate := strictIndianDate(rawValueDate)
				if valueDate == "" {
					return nil, meta, fmt.Errorf("%s statement contains an invalid value date", profile.bankName)
				}
				parsed[len(parsed)-1].tx.ValueDate = &valueDate
			}
		}
	}
	if len(parsed) == 0 {
		return nil, meta, fmt.Errorf("%s transaction table contains no complete rows", profile.bankName)
	}

	// Bank exports can run oldest-first or newest-first, including within a day.
	// Reverse the complete sequence when required; sorting by date alone loses
	// the order of same-day transactions.
	balances := bankHistoryBalances{
		openingFound: openingFound,
		opening:      meta.OpeningBalance,
		closingFound: closingFound,
		closing:      meta.ClosingBalance,
	}
	if !bankHistorySequenceReconciles(parsed, balances) {
		slices.Reverse(parsed)
		if !bankHistorySequenceReconciles(parsed, balances) {
			return nil, meta, fmt.Errorf("%s dated transactions and running balances do not reconcile in source or reverse order", profile.bankName)
		}
	}
	if !openingFound {
		meta.OpeningBalance = bankHistoryPreviousBalance(parsed[0])
	}
	if !closingFound {
		meta.ClosingBalance = parsed[len(parsed)-1].balance
	}
	debits, credits := 0.0, 0.0
	for _, row := range parsed {
		if row.tx.TxType == models.TxTypeDebit {
			debits += row.tx.Amount
		} else {
			credits += row.tx.Amount
		}
	}
	if debitTotalFound && !historyMoneyEqual(meta.TotalDebits, debits) {
		return nil, meta, fmt.Errorf("%s withdrawals do not match the printed debit total", profile.bankName)
	}
	if creditTotalFound && !historyMoneyEqual(meta.TotalCredits, credits) {
		return nil, meta, fmt.Errorf("%s deposits do not match the printed credit total", profile.bankName)
	}
	if !debitTotalFound {
		meta.TotalDebits = debits
	}
	if !creditTotalFound {
		meta.TotalCredits = credits
	}
	if !historyMoneyEqual(meta.OpeningBalance+credits-debits, meta.ClosingBalance) {
		return nil, meta, fmt.Errorf("%s opening balance, transactions, and closing balance do not reconcile", profile.bankName)
	}
	if meta.StartDate == "" || meta.EndDate == "" {
		meta.StartDate, meta.EndDate = parsed[0].tx.Date, parsed[len(parsed)-1].tx.Date
	}
	transactions := make([]ParsedTransaction, len(parsed))
	for index := range parsed {
		balance := parsed[index].balance
		transactions[index] = parsed[index].tx
		cleaned := CleanNarration(transactions[index].RawNarration)
		transactions[index].CleanedPayee = cleaned.CleanedPayee
		transactions[index].PaymentMode = cleaned.PaymentMode
		transactions[index].UPIVPA = cleaned.UPIVPA
		transactions[index].CardLast4 = cleaned.CardLast4
		transactions[index].IsTransfer = cleaned.IsTransfer
		if transactions[index].ReferenceNumber == "" {
			transactions[index].ReferenceNumber = cleaned.ReferenceNumber
		}
		transactions[index].RunningBalance = &balance
	}
	return transactions, meta, nil
}

func discoverBankHistoryColumns(row extractor.PositionalRow, profile bankHistoryProfile) (bankHistoryColumns, bool) {
	text := bankHistoryRowText(row)
	upper := strings.ToUpper(text)
	if !containsAllBankTerms(upper, profile.recognition) {
		return bankHistoryColumns{}, false
	}
	columns := bankHistoryColumns{date: -1, description: -1, debit: -1, credit: -1, balance: -1, reference: -1, serial: -1, mode: -1, valueDate: -1}
	for _, element := range row.Elements {
		label := strings.ToUpper(strings.Join(strings.Fields(element.S), " "))
		for _, field := range []struct {
			position float64
			terms    []string
		}{{columns.description, profile.description}, {columns.debit, profile.debitHeaders}, {columns.credit, profile.creditHeaders}, {columns.balance, profile.balanceHeaders}} {
			if field.position >= 0 && field.position != element.X && matchesBankTerm(label, field.terms) {
				return bankHistoryColumns{}, false
			}
		}
		switch {
		case label == "SI" || label == "SL" || label == "S.NO" || label == "SR NO":
			columns.serial = element.X
		case label == "MODE":
			columns.mode = element.X
		case label == "VALUE DATE" || label == "VALUEDATE" || label == "VAL DATE":
			columns.valueDate = element.X
		case columns.date < 0 && matchesBankTerm(label, profile.dateHeaders):
			columns.date = element.X
		case columns.description < 0 && matchesBankTerm(label, profile.description):
			columns.description = element.X
		case columns.debit < 0 && matchesBankTerm(label, profile.debitHeaders):
			columns.debit = element.X
		case columns.credit < 0 && matchesBankTerm(label, profile.creditHeaders):
			columns.credit = element.X
		case columns.balance < 0 && matchesBankTerm(label, profile.balanceHeaders):
			columns.balance = element.X
		case !columns.hasReference && matchesBankTerm(label, profile.reference):
			columns.reference, columns.hasReference = element.X, true
		}
	}
	if columns.date < 0 || columns.description < 0 || columns.debit < 0 || columns.credit < 0 || columns.balance < 0 {
		return bankHistoryColumns{}, false
	}
	if !(columns.date < columns.description && columns.description < columns.debit && columns.description < columns.credit && columns.debit < columns.balance && columns.credit < columns.balance && columns.debit != columns.credit) {
		return bankHistoryColumns{}, false
	}
	return columns, true
}

func containsAllBankTerms(text string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(text, strings.ToUpper(term)) {
			return false
		}
	}
	return true
}

// Card-payment narration is valid bank data. Strong savings-table headings
// distinguish it from a credit-card bill, without using transaction merchants
// or personal identifiers as format evidence.
func hasBankHistoryTableText(text string, descriptions []string) bool {
	text = strings.ToUpper(strings.Join(strings.Fields(text), " "))
	return containsAllBankTerms(text, []string{"DATE", "WITHDRAWAL", "DEPOSIT", "BALANCE"}) && matchesBankTerm(text, descriptions)
}

func matchesBankTerm(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, strings.ToUpper(term)) {
			return true
		}
	}
	return false
}

func bankHistoryRowText(row extractor.PositionalRow) string {
	parts := make([]string, 0, len(row.Elements))
	for _, element := range row.Elements {
		parts = append(parts, strings.TrimSpace(element.S))
	}
	return strings.Join(parts, " ")
}

func bankHistoryDocumentText(rows []extractor.PositionalRow) string {
	var lines []string
	for _, row := range rows {
		lines = append(lines, bankHistoryRowText(row))
	}
	return strings.Join(lines, "\n")
}

func bankHistoryDateCell(row extractor.PositionalRow, columns bankHistoryColumns) (int, string) {
	start, end := bankHistoryColumnBounds(columns.date, columns)
	descriptionStart, descriptionEnd := bankHistoryColumnBounds(columns.description, columns)
	serialIndex, serialDate := -1, ""
	for index, element := range row.Elements {
		if date := bankHistoryMergedSerialDate(element, columns); date != "" {
			if serialIndex >= 0 {
				return -1, ""
			}
			serialIndex, serialDate = index, date
		}
	}
	var values []string
	var indexes []int
	for index, element := range row.Elements {
		if element.X < start || end > start && element.X >= end {
			continue
		}
		// The measured description-boundary exception belongs to narration,
		// not to a second date cell on the same or neighboring baseline.
		if element.X < descriptionStart && bankHistoryDescriptionElement(element, descriptionStart, descriptionEnd) {
			continue
		}
		value := strings.TrimSpace(element.S)
		if value != "" {
			if serialIndex >= 0 && bankHistoryDatePrefix.MatchString(value) {
				return -1, ""
			}
			values = append(values, value)
			indexes = append(indexes, index)
		}
	}
	if serialIndex >= 0 {
		return serialIndex, serialDate
	}
	if len(values) == 1 && bankHistoryDatePrefix.MatchString(values[0]) {
		return indexes[0], bankHistoryDatePrefix.FindString(values[0])
	}
	// Some exports put the year on the next baseline inside the date cell.
	// Join only this exact two-fragment shape; other numeric fragments do not
	// establish a date and multiple dated cells remain ambiguous.
	if len(values) == 2 && bankHistoryDatePart.MatchString(values[0]) && bankHistoryYearPart.MatchString(values[1]) {
		date := values[0] + values[1]
		if strictIndianDate(date) != "" {
			return indexes[0], date
		}
	}
	return -1, ""
}

func bankHistoryMergedSerialDate(element extractor.PositionalElement, columns bankHistoryColumns) string {
	if columns.serial < 0 || element.X < columns.serial-5 || element.X >= columns.description || element.EndX <= element.X || element.EndX > columns.description {
		return ""
	}
	match := bankHistorySerialDate.FindStringSubmatch(strings.TrimSpace(element.S))
	if len(match) != 2 || strictIndianDate(match[1]) == "" {
		return ""
	}
	return match[1]
}

// Leave the generic extractor's 1pt baseline grouping unchanged. Bank tables
// can place date/financial cells on adjacent baselines. Prefer a whole complete
// transaction window within 12pt on one page, preserving all narration. When
// surrounding baselines exceed that span, require a unique smallest complete
// window. Conflicting dates or numeric cells never establish one transaction.
func coalesceBankHistoryRows(rows []extractor.PositionalRow, profile bankHistoryProfile) []extractor.PositionalRow {
	var output []extractor.PositionalRow
	var columns bankHistoryColumns
	active, boundary := false, 0
	for index := 0; index < len(rows); index++ {
		row := rows[index]
		if candidate, ok := discoverBankHistoryColumns(row, profile); ok {
			columns, active, boundary = candidate, true, index+1
			output = append(output, row)
			continue
		}
		if profile.stopPattern != nil && profile.stopPattern.MatchString(bankHistoryRowText(row)) {
			active = false
		}
		if !active || completeBankHistoryRow(row, columns) || !bankHistoryDateAnchor(row, columns) {
			output = append(output, row)
			continue
		}
		left, right := index, index
		for left > boundary && rows[left-1].Page == row.Page && absFloat(rows[left-1].Y-row.Y) <= 12 {
			if _, header := discoverBankHistoryColumns(rows[left-1], profile); header {
				break
			}
			left--
		}
		for right+1 < len(rows) && rows[right+1].Page == row.Page && absFloat(rows[right+1].Y-row.Y) <= 12 {
			candid := rows[right+1]
			if _, header := discoverBankHistoryColumns(candid, profile); header || profile.stopPattern != nil && profile.stopPattern.MatchString(bankHistoryRowText(candid)) {
				break
			}
			right++
		}
		// Narration may straddle one date/financial anchor symmetrically.
		// Prefer the whole bounded window when it still has one unambiguous
		// date, debit/credit and balance. Different descriptions surrounding
		// that same anchor are continuations, not competing transactions.
		if absFloat(rows[left].Y-rows[right].Y) <= 12 {
			merged := extractor.PositionalRow{Page: row.Page, Y: row.Y}
			for _, fragment := range rows[left : right+1] {
				merged.Elements = append(merged.Elements, fragment.Elements...)
			}
			if completeBankHistoryRow(merged, columns) {
				output = append(output[:len(output)-(index-left)], merged)
				index, boundary = right, right+1
				continue
			}
			// Do not salvage a smaller subset by dropping conflicting date or
			// financial fragments from this same bounded printed transaction.
			output = append(output, row)
			continue
		}
		bestStart, bestEnd, bestSize := -1, -1, len(rows)+1
		var best extractor.PositionalRow
		ambiguous := false
		for start := left; start <= index; start++ {
			for end := index; end <= right; end++ {
				if end-start+1 > bestSize || absFloat(rows[start].Y-rows[end].Y) > 12 {
					continue
				}
				merged := extractor.PositionalRow{Page: row.Page, Y: row.Y}
				for _, fragment := range rows[start : end+1] {
					merged.Elements = append(merged.Elements, fragment.Elements...)
				}
				if !completeBankHistoryRow(merged, columns) {
					continue
				}
				if end-start+1 < bestSize {
					bestStart, bestEnd, bestSize, best, ambiguous = start, end, end-start+1, merged, false
				} else if bankHistoryRowText(best) != bankHistoryRowText(merged) {
					ambiguous = true
				}
			}
		}
		if bestStart < 0 || ambiguous {
			output = append(output, row)
			continue
		}
		output = append(output[:len(output)-(index-bestStart)], best)
		index, boundary = bestEnd, bestEnd+1
	}
	return output
}

func bankHistoryDateAnchor(row extractor.PositionalRow, columns bankHistoryColumns) bool {
	start, end := bankHistoryColumnBounds(columns.date, columns)
	if index, _ := bankHistoryDateCell(row, columns); index >= 0 {
		return true
	}
	for _, element := range row.Elements {
		if element.X >= start && element.X < end && bankHistoryDatePart.MatchString(strings.TrimSpace(element.S)) {
			return true
		}
	}
	return false
}

func completeBankHistoryRow(row extractor.PositionalRow, columns bankHistoryColumns) bool {
	if _, date := bankHistoryDateCell(row, columns); strictIndianDate(date) == "" || bankHistoryNarration(row, columns) == "" {
		return false
	}
	start, end := bankHistoryColumnBounds(columns.debit, columns)
	debit, _, debitErr := amountInBankColumn(row, start, end)
	start, end = bankHistoryColumnBounds(columns.credit, columns)
	credit, _, creditErr := amountInBankColumn(row, start, end)
	start, end = bankHistoryColumnBounds(columns.balance, columns)
	_, hasBalance, balanceErr := balanceInBankColumn(row, start, end)
	return debitErr == nil && creditErr == nil && balanceErr == nil && hasBalance && (debit > 0 && credit == 0 || credit > 0 && debit == 0)
}

func strictIndianDate(raw string) string {
	normalized := extractor.NormalizeIndianDate(raw)
	if _, err := time.Parse("2006-01-02", normalized); err != nil {
		return ""
	}
	return normalized
}

func amountInBankColumn(row extractor.PositionalRow, start, end float64) (float64, bool, error) {
	cell := bankHistoryCellInColumn(row, start, end)
	if cell == "" || cell == "-" || cell == "--" {
		return 0, false, nil
	}
	match := bankHistoryAmount.FindStringSubmatch(cell)
	if len(match) != 3 {
		return 0, false, fmt.Errorf("invalid amount cell")
	}
	amount, err := extractor.ParseIndianAmount(match[1])
	if err != nil || amount < 0 {
		return 0, false, fmt.Errorf("invalid amount value")
	}
	return amount, true, nil
}

// A running balance may be below zero even though withdrawal/deposit amounts
// must remain positive. Use the same rule during row assembly and parsing.
func balanceInBankColumn(row extractor.PositionalRow, start, end float64) (float64, bool, error) {
	cell := bankHistoryCellInColumn(row, start, end)
	if match := bankHistorySignedBalance.FindStringSubmatch(cell); len(match) == 2 {
		amount, err := extractor.ParseIndianAmount(match[1])
		return amount, err == nil, err
	}
	return amountInBankColumn(row, start, end)
}

func bankHistoryCellInColumn(row extractor.PositionalRow, start, end float64) string {
	var values []string
	for _, element := range row.Elements {
		if element.X < start || (end > start && element.X >= end) {
			continue
		}
		value := strings.TrimSpace(element.S)
		if value != "" {
			values = append(values, value)
		}
	}
	return strings.Join(values, " ")
}

func bankHistoryNarration(row extractor.PositionalRow, columns bankHistoryColumns) string {
	start, end := bankHistoryColumnBounds(columns.description, columns)
	// When serial and date occupy one measured block before PARTICULARS, the
	// unused DATE region can hold narration on its neighboring baseline.
	for _, element := range row.Elements {
		if bankHistoryMergedSerialDate(element, columns) != "" {
			start = columns.date
			break
		}
	}
	var parts []string
	for _, element := range row.Elements {
		value := strings.TrimSpace(element.S)
		if !bankHistoryDescriptionElement(element, start, end) {
			continue
		}
		if bankHistoryDatePrefix.MatchString(value) {
			continue
		}
		if _, err := extractor.ParseIndianAmount(value); err == nil && bankHistoryMoneyToken.MatchString(value) {
			continue
		}
		parts = append(parts, value)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func bankHistoryReference(row extractor.PositionalRow, columns bankHistoryColumns) string {
	if !columns.hasReference {
		return ""
	}
	start, end := bankHistoryColumnBounds(columns.reference, columns)
	var parts []string
	for _, element := range row.Elements {
		if element.X < start || element.X >= end {
			continue
		}
		value := strings.TrimSpace(element.S)
		if value != "" && !bankHistoryDatePrefix.MatchString(value) {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "")
}

// Continuation lines contain only description/reference cells. Account headers,
// page footers and numeric summary lines must not become transaction narration.
func bankHistoryContinuation(row extractor.PositionalRow, columns bankHistoryColumns) bool {
	if len(row.Elements) == 0 {
		return false
	}
	start, end := bankHistoryColumnBounds(columns.description, columns)
	if columns.hasReference {
		_, end = bankHistoryColumnBounds(columns.reference, columns)
	}
	for _, element := range row.Elements {
		if strings.TrimSpace(element.S) != "" && !bankHistoryDescriptionElement(element, start, end) {
			return false
		}
	}
	return true
}

// A centered heading's midpoint can miss a left-aligned narration by a few
// points. Accept that small overlap only for measured text crossing into the
// description region; never widen date or financial columns, infer missing
// extents, or treat date fragments/numeric cells as narration.
func bankHistoryDescriptionElement(element extractor.PositionalElement, start, end float64) bool {
	if element.X >= end {
		return false
	}
	if element.X >= start {
		return true
	}
	value := strings.TrimSpace(element.S)
	if element.X < start-3 || element.EndX <= start || bankHistoryDatePrefix.MatchString(value) || bankHistoryDatePart.MatchString(value) || bankHistoryYearPart.MatchString(value) || bankHistoryAmount.MatchString(value) {
		return false
	}
	return strings.ContainsFunc(value, unicode.IsLetter)
}

// Centered headings and differently aligned cells share the regions bounded by
// midpoints between adjacent known columns, regardless of debit/credit order.
func bankHistoryColumnBounds(position float64, columns bankHistoryColumns) (float64, float64) {
	previous, next := -1.0, -1.0
	for _, candidate := range []float64{columns.serial, columns.date, columns.mode, columns.description, columns.reference, columns.valueDate, columns.debit, columns.credit, columns.balance} {
		if candidate < 0 {
			continue
		}
		if candidate < position && candidate > previous {
			previous = candidate
		}
		if candidate > position && (next < 0 || candidate < next) {
			next = candidate
		}
	}
	start, end := 0.0, 0.0
	if previous >= 0 {
		start = (previous + position) / 2
	}
	if next >= 0 {
		end = (position + next) / 2
	}
	return start, end
}

func bankHistorySequenceReconciles(rows []bankHistoryRow, balances bankHistoryBalances) bool {
	for index, row := range rows {
		previous := bankHistoryPreviousBalance(row)
		if index == 0 {
			if balances.openingFound && !historyMoneyEqual(balances.opening, previous) {
				return false
			}
		} else if row.tx.Date < rows[index-1].tx.Date || !historyMoneyEqual(rows[index-1].balance, previous) {
			return false
		}
	}
	return !balances.closingFound || historyMoneyEqual(balances.closing, rows[len(rows)-1].balance)
}

func bankHistoryPreviousBalance(row bankHistoryRow) float64 {
	if row.tx.TxType == models.TxTypeCredit {
		return row.balance - row.tx.Amount
	}
	return row.balance + row.tx.Amount
}

func parseLabeledHistoryAmount(pattern *regexp.Regexp, line string) (float64, bool) {
	if pattern == nil {
		return 0, false
	}
	match := pattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return 0, false
	}
	amount, err := extractor.ParseIndianAmount(match[1])
	if err != nil || amount < 0 {
		return 0, false
	}
	if len(match) > 2 && strings.EqualFold(strings.TrimSpace(match[2]), "DR") {
		return -absFloat(amount), true
	}
	return absFloat(amount), true
}

func historyMoneyEqual(a, b float64) bool { return absFloat(a-b) <= 0.01 }

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
