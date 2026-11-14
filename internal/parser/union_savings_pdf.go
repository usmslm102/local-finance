package parser

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

type UnionSavingsPDFParser struct{}

func init() { DefaultRegistry.Register(&UnionSavingsPDFParser{}) }

func (p *UnionSavingsPDFParser) ID() string   { return "union_savings_pdf_v1" }
func (p *UnionSavingsPDFParser) Name() string { return "Union Bank of India Savings Account (PDF)" }
func (p *UnionSavingsPDFParser) SupportedTypes() []StatementType {
	return []StatementType{TypeSavingsPDF}
}

func (p *UnionSavingsPDFParser) CanParse(filename string, sample []byte) (float64, string, models.AccountType) {
	name := strings.ToUpper(filename)
	content := strings.ToUpper(strings.Join(strings.Fields(string(sample)), " "))
	isPDF := strings.HasSuffix(name, ".PDF") || strings.HasPrefix(string(sample), "%PDF")
	hasSignedTable := containsAllBankTerms(content, []string{"STATEMENT DETAILS", "DATE", "TRANSACTION ID", "REMARKS", "AMOUNT", "BALANCE"})
	hasTable := hasBankHistoryTableText(content, unionSavingsProfile.description) || hasSignedTable
	if !isPDF || isInvestmentStatement(string(sample)) || strings.Contains(name, "_CC") || strings.Contains(content, "TOTAL AMOUNT DUE") || strings.Contains(content, "CREDIT CARD") && !hasTable {
		return 0, "", models.AccountTypeSavings
	}
	confidence := 0.0
	if strings.Contains(name, "UNION") || strings.Contains(content, "UNION BANK OF INDIA") || strings.Contains(content, "UBIN") {
		confidence += 0.55
	}
	if strings.Contains(content, "DETAILS OF STATEMENT") && strings.Contains(content, "WITHDRAWAL") && strings.Contains(content, "DEPOSIT") || hasSignedTable {
		confidence += 0.4
	}
	return confidence, "Union Bank of India", models.AccountTypeSavings
}

var unionSavingsProfile = bankHistoryProfile{
	bankName:       "Union Bank of India",
	accountType:    models.AccountTypeSavings,
	formatName:     "PDF",
	identityTerm:   "UBIN",
	identityTerms:  []string{"UBIN", "UNION BANK OF INDIA", "UNION BANK"},
	statementTerm:  "DETAILS OF STATEMENT",
	statementTerms: []string{"DETAILS OF STATEMENT", "STATEMENT OF ACCOUNT", "ACCOUNT STATEMENT"},
	recognition:    []string{"DATE", "WITHDRAWAL", "DEPOSIT", "BALANCE"},
	dateHeaders:    []string{"TRANSACTION DATE", "DATE"},
	description:    []string{"TRANSACTION REMARKS", "PARTICULARS", "NARRATION"},
	debitHeaders:   []string{"WITHDRAWAL", "DEBIT"},
	creditHeaders:  []string{"DEPOSIT", "CREDIT"},
	balanceHeaders: []string{"BALANCE"},
	reference:      []string{"CHQ NUM", "CHEQUE NUMBER", "CHEQUE NO", "REFERENCE"},
	accountPattern: regexp.MustCompile(`(?i)(?:account\s*(?:number|no\.?)|a/c\s*(?:number|no\.?))\s*:?\s*([0-9X*]{8,20})\b`),
	periodPattern:  regexp.MustCompile(`(?i)(?:period\s+from|statement\s+period)\s*:?\s*(\d{2}[./-]\d{2}[./-]\d{2,4})\s*(?:to|-)\s*(\d{2}[./-]\d{2}[./-]\d{2,4})`),
	openingPattern: regexp.MustCompile(`(?i)opening\s+balance\s*:?\s*([0-9][0-9,]*(?:\.\d{2})?)\s*(CR|DR)?`),
	closingPattern: regexp.MustCompile(`(?i)closing\s+balance\s*:?\s*([0-9][0-9,]*(?:\.\d{2})?)\s*(CR|DR)?`),
	debitTotal:     regexp.MustCompile(`(?i)total\s+debits\s*:?\s*([0-9][0-9,]*(?:\.\d{2})?)`),
	creditTotal:    regexp.MustCompile(`(?i)total\s+credits\s*:?\s*([0-9][0-9,]*(?:\.\d{2})?)`),
	stopPattern:    regexp.MustCompile(`(?i)^\s*(?:total\s+debits|closing\s+balance|linked\s+casa|end\s+of\s+statement)\b`),
}

func (p *UnionSavingsPDFParser) Parse(r io.Reader, opts ParseOptions) ([]ParsedTransaction, StatementMeta, error) {
	rows, err := extractor.ExtractPDFPositionalRows(r, opts.Password)
	if err != nil {
		return nil, StatementMeta{}, fmt.Errorf("failed to extract Union Bank of India statement: %w", err)
	}
	return parseUnionSavingsRows(rows)
}

var unionSignedMoney = regexp.MustCompile(`(?i)^\s*([0-9][0-9,]*(?:\.\d{2})?)\s*\(\s*(DR|CR)\s*\)\s*$`)

// New retail exports have centered headings and one signed amount column.
// Map that table to the existing history schema so both layouts use the same
// date validation, narration cleaning, account identity and reconciliation.
func parseUnionSavingsRows(rows []extractor.PositionalRow) ([]ParsedTransaction, StatementMeta, error) {
	var normalized []extractor.PositionalRow
	var dateEnd, referenceEnd, remarksEnd, amountEnd float64
	active, found := false, false
	page := 0
	for _, row := range rows {
		if row.Page != page {
			// Page furniture precedes the repeated table heading and must not
			// be interpreted using the previous page's financial columns.
			active, page = false, row.Page
		}
		date, reference, remarks, amount, balance := -1.0, -1.0, -1.0, -1.0, -1.0
		refEnd := 0.0
		for _, el := range row.Elements {
			label := strings.ToUpper(strings.Join(strings.Fields(el.S), " "))
			switch {
			case label == "DATE":
				if date >= 0 {
					return nil, StatementMeta{}, fmt.Errorf("Union statement has ambiguous date headings")
				}
				date = el.X
			case label == "TRANSACTION ID":
				if reference >= 0 {
					return nil, StatementMeta{}, fmt.Errorf("Union statement has ambiguous reference headings")
				}
				reference, refEnd = el.X, el.EndX
			case label == "REMARKS":
				if remarks >= 0 {
					return nil, StatementMeta{}, fmt.Errorf("Union statement has ambiguous remarks headings")
				}
				remarks = el.X
			case strings.HasPrefix(label, "AMOUNT(") || strings.HasPrefix(label, "AMOUNT ("):
				if amount >= 0 {
					return nil, StatementMeta{}, fmt.Errorf("Union statement has ambiguous amount headings")
				}
				amount = el.X
			case strings.HasPrefix(label, "BALANCE(") || strings.HasPrefix(label, "BALANCE ("):
				if balance >= 0 {
					return nil, StatementMeta{}, fmt.Errorf("Union statement has ambiguous balance headings")
				}
				balance = el.X
			}
		}
		if date >= 0 && reference > date && remarks > reference && amount > remarks && balance > amount {
			dateEnd = (date + reference) / 2
			// The reference heading ends at the left edge of the remarks cell;
			// its centered Remarks label is far to the right of short narrations.
			referenceEnd = refEnd
			if referenceEnd <= reference || referenceEnd >= remarks {
				referenceEnd = reference + (remarks-reference)/3
			}
			remarksEnd, amountEnd = (remarks+amount)/2, (amount+balance)/2
			active, found = true, true
			normalized = append(normalized, extractor.PositionalRow{Page: row.Page, Y: row.Y, Elements: []extractor.PositionalElement{
				{X: 0, S: "Date"}, {X: 100, S: "Particulars"}, {X: 200, S: "Reference"},
				{X: 300, S: "Withdrawal"}, {X: 400, S: "Deposit"}, {X: 500, S: "Balance"},
			}})
			continue
		}
		if !active {
			if found && len(row.Elements) > 0 && row.Elements[0].X < dateEnd && bankHistoryDatePrefix.MatchString(strings.TrimSpace(row.Elements[0].S)) {
				return nil, StatementMeta{}, fmt.Errorf("Union statement contains a transaction before its page table heading")
			}
			normalized = append(normalized, row)
			continue
		}
		converted := extractor.PositionalRow{Page: row.Page, Y: row.Y}
		for _, el := range row.Elements {
			switch {
			case el.X < dateEnd:
				el.X = 0
			case el.X < referenceEnd:
				el.X = 200
			case el.X < remarksEnd:
				el.X = 100
			default:
				match := unionSignedMoney.FindStringSubmatch(el.S)
				if el.X < amountEnd {
					// Missing or unknown direction must fail rather than guessing
					// from narration or silently omitting a dated transaction.
					if len(match) != 3 {
						return nil, StatementMeta{}, fmt.Errorf("Union statement contains an amount without a valid (Dr)/(Cr) marker")
					}
					el.X, el.S = 300, match[1]
					if strings.EqualFold(match[2], "CR") {
						el.X = 400
					}
				} else {
					el.X = 500
					if len(match) == 3 {
						el.S = match[1] + " " + strings.ToUpper(match[2])
					}
				}
			}
			el.EndX = 0
			converted.Elements = append(converted.Elements, el)
		}
		normalized = append(normalized, converted)
	}
	if !found {
		return parseBankHistoryRows(rows, unionSavingsProfile)
	}
	profile := unionSavingsProfile
	profile.statementTerms = append([]string{"STATEMENT DETAILS"}, profile.statementTerms...)
	return parseBankHistoryRows(normalized, profile)
}
