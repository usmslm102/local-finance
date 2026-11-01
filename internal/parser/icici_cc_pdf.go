package parser

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

type ICICICCPDFParser struct{}

func init() {
	DefaultRegistry.Register(&ICICICCPDFParser{})
}

func (p *ICICICCPDFParser) ID() string {
	return "icici_cc_pdf_v1"
}

func (p *ICICICCPDFParser) Name() string {
	return "ICICI Bank Credit Card (PDF)"
}

func (p *ICICICCPDFParser) SupportedTypes() []StatementType {
	return []StatementType{TypeCreditCardPDF}
}

func (p *ICICICCPDFParser) CanParse(filename string, sample []byte) (float64, string, models.AccountType) {
	nameUpper := strings.ToUpper(filename)
	content := strings.ToUpper(string(sample))

	isPDF := strings.HasSuffix(nameUpper, ".PDF") || (len(sample) > 4 && string(sample[:4]) == "%PDF")
	if !isPDF {
		return 0.0, "", models.AccountTypeCreditCard
	}
	if isICICICCPortalStatement(string(sample)) {
		return 0.95, "ICICI Bank", models.AccountTypeCreditCard
	}

	confidence := 0.0
	if strings.Contains(nameUpper, "ICICI") || strings.Contains(content, "ICICI BANK") {
		confidence += 0.4
	}
	if strings.Contains(nameUpper, "AMAZON") || strings.Contains(nameUpper, "CORAL") ||
		strings.Contains(nameUpper, "RUBYX") || strings.Contains(nameUpper, "SAPPHIRO") ||
		strings.Contains(content, "AMAZON PAY") || strings.Contains(content, "PURCHASES / CHARGES") {
		confidence += 0.4
	}

	return confidence, "ICICI Bank", models.AccountTypeCreditCard
}

var (
	iciciCCCardNoRegex     = regexp.MustCompile(`(?i)(\d{4}[X*]{6,10}\d{4})`)
	iciciCCPeriodRegex     = regexp.MustCompile(`(?i)Statement period\s*:\s*([A-Za-z]+\s+\d{1,2},?\s+\d{4})\s+to\s+([A-Za-z]+\s+\d{1,2},?\s+\d{4})`)
	iciciCCLineDateRegex   = regexp.MustCompile(`^(\d{2}/\d{2}/\d{4})`)
	iciciCCAmountAtEnd     = regexp.MustCompile(`(?i)([0-9,]+\.\d{2})\s*(CR)?$`)
	iciciCCNamePrefixRegex = regexp.MustCompile(`(?i)^(?:MR|MS|MRS|DR|PROF)\.?\s+([A-Z\s]{3,50})$`)
	iciciCCCustNameRegex   = regexp.MustCompile(`(?i)Customer\s+Name\s*[:\s]*([A-Z\s]{3,50})`)
)

func (p *ICICICCPDFParser) Parse(r io.Reader, opts ParseOptions) ([]ParsedTransaction, StatementMeta, error) {
	rows, err := extractor.ExtractPDFPositionalRows(r, opts.Password)
	if err != nil {
		return nil, StatementMeta{}, fmt.Errorf("failed to extract PDF rows: %w", err)
	}
	if isICICICCPortalStatement(bankHistoryDocumentText(rows)) {
		return parseICICICCPortalRows(rows)
	}

	meta := StatementMeta{
		BankName:        "ICICI Bank",
		AccountType:     models.AccountTypeCreditCard,
		CardNetwork:     "VISA",
		CardVariant:     "Amazon Pay ICICI",
		StatementFormat: "PDF",
	}

	// 1. Scan metadata
	for _, row := range rows {
		var lineParts []string
		for _, el := range row.Elements {
			lineParts = append(lineParts, el.S)
		}
		lineStr := strings.Join(lineParts, " ")

		if matches := iciciCCCustNameRegex.FindStringSubmatch(lineStr); len(matches) > 1 && meta.AccountHolderName == "" {
			meta.AccountHolderName = extractor.CleanCustomerName(matches[1])
		}

		if matches := iciciCCCardNoRegex.FindStringSubmatch(lineStr); len(matches) > 1 && meta.AccountNumberMask == "" {
			rawCard := matches[1]
			meta.AccountNumberMask = "XX" + rawCard[max(0, len(rawCard)-4):]
		}
		if matches := iciciCCPeriodRegex.FindStringSubmatch(lineStr); len(matches) > 2 {
			if meta.StartDate == "" {
				meta.StartDate = extractor.NormalizeIndianDate(matches[1])
			}
			if meta.EndDate == "" {
				meta.EndDate = extractor.NormalizeIndianDate(matches[2])
			}
		}

		// Spatial extraction on Page 1
		if row.Page == 1 {
			// Scan top address block (Y: 710..760, X < 100) for cardholder name
			if meta.AccountHolderName == "" && row.Y >= 710.0 && row.Y <= 760.0 {
				for _, el := range row.Elements {
					if el.X < 100.0 {
						trimmed := strings.TrimSpace(el.S)
						if m := iciciCCNamePrefixRegex.FindStringSubmatch(trimmed); len(m) > 1 {
							meta.AccountHolderName = extractor.CleanCustomerName(m[1])
							break
						}
					}
				}
			}
			// Statement Date at Y ~ 636
			if row.Y >= 630.0 && row.Y <= 645.0 {
				for _, el := range row.Elements {
					if el.X < 120.0 {
						if dNorm := extractor.NormalizeIndianDate(el.S); dNorm != "" && meta.StatementDate == "" {
							meta.StatementDate = dNorm
						}
					}
				}
			}
			// Payment Due Date at Y ~ 600..615
			if row.Y >= 595.0 && row.Y <= 615.0 {
				for _, el := range row.Elements {
					if el.X < 120.0 {
						if dNorm := extractor.NormalizeIndianDate(el.S); dNorm != "" && meta.PaymentDueDate == "" {
							meta.PaymentDueDate = dNorm
						}
					}
				}
			}
			// Total Amount Due at Y ~ 552
			if row.Y >= 545.0 && row.Y <= 560.0 {
				for _, el := range row.Elements {
					if el.X < 120.0 {
						if a, err := extractor.ParseIndianAmount(el.S); err == nil && a > 0 {
							meta.TotalDueAmount = a
						}
					}
				}
			}
			// Minimum Due at Y ~ 513
			if row.Y >= 505.0 && row.Y <= 520.0 {
				for _, el := range row.Elements {
					if el.X < 120.0 {
						if a, err := extractor.ParseIndianAmount(el.S); err == nil && a > 0 {
							meta.MinimumDueAmount = a
						}
					}
				}
			}
			// Credit Limit & Available Credit Limit at Y ~ 487
			if row.Y >= 480.0 && row.Y <= 495.0 {
				for _, el := range row.Elements {
					if el.X >= 200.0 && el.X <= 260.0 && meta.CreditLimit == 0 {
						if a, err := extractor.ParseIndianAmount(el.S); err == nil && a > 0 {
							meta.CreditLimit = a
						}
					}
					if el.X >= 300.0 && el.X <= 360.0 && meta.AvailableCreditLimit == 0 {
						if a, err := extractor.ParseIndianAmount(el.S); err == nil && a > 0 {
							meta.AvailableCreditLimit = a
						}
					}
				}
			}
		}
	}

	// 2. Parse transactions
	var transactions []ParsedTransaction

	for _, row := range rows {
		if len(row.Elements) == 0 {
			continue
		}

		// First element starts with date DD/MM/YYYY
		firstEl := row.Elements[0]
		firstText := strings.TrimSpace(firstEl.S)

		if matches := iciciCCLineDateRegex.FindStringSubmatch(firstText); len(matches) > 1 {
			normDate := extractor.NormalizeIndianDate(matches[1])
			if normDate == "" {
				continue
			}

			// Collect line elements
			var otherParts []string
			var refNo string
			var amount float64
			var rewardPoints float64
			txType := models.TxTypeDebit
			isTransfer := false

			// Check remainder of first element if concatenated
			rem := strings.TrimSpace(strings.TrimPrefix(firstText, matches[1]))
			if rem != "" {
				otherParts = append(otherParts, rem)
			}

			for _, el := range row.Elements[1:] {
				sTrim := strings.TrimSpace(el.S)
				if sTrim == "" {
					continue
				}

				// Check serial/ref number (9-12 digits)
				if len(sTrim) >= 9 && len(sTrim) <= 14 && isDigitsOnly(sTrim) && refNo == "" {
					refNo = sTrim
					continue
				}

				// Check Reward Points column (X ~ 380..460)
				if (el.X >= 380.0 && el.X <= 460.0) || (row.Page > 1 && el.X >= 380.0 && el.X <= 440.0) {
					if pts, err := extractor.ParseIndianAmount(sTrim); err == nil {
						rewardPoints = pts
						continue
					}
				}

				// Check Amount at end (e.g. "8,100.00" or "55,601.18 CR")
				if amtMatches := iciciCCAmountAtEnd.FindStringSubmatch(sTrim); len(amtMatches) > 1 {
					if parsedAmt, err := extractor.ParseIndianAmount(amtMatches[1]); err == nil && parsedAmt > 0 {
						amount = parsedAmt
						if amtMatches[2] != "" || strings.EqualFold(amtMatches[2], "CR") {
							txType = models.TxTypeCredit
							isTransfer = true
						}
						continue
					}
				}

				otherParts = append(otherParts, sTrim)
			}

			if amount == 0 {
				continue
			}

			fullNarr := strings.Join(otherParts, " ")
			fullNarr = strings.TrimSpace(fullNarr)
			if strings.Contains(strings.ToUpper(fullNarr), "PAYMENT RECEIVED") {
				txType = models.TxTypeCredit
				isTransfer = true
			}

			cleaned := CleanNarration(fullNarr)
			if txType == models.TxTypeCredit {
				cleaned.IsTransfer = true
			}

			cbAmount := 0.0
			if rewardPoints > 0 {
				// For Amazon Pay ICICI, 1 Reward Point = ₹1 direct Cashback into Amazon Pay Balance
				cbAmount = rewardPoints
			}

			transactions = append(transactions, ParsedTransaction{
				Date:               normDate,
				RawNarration:       fullNarr,
				CleanedPayee:       cleaned.CleanedPayee,
				PaymentMode:        models.PaymentModeCardPOS,
				ReferenceNumber:    refNo,
				TxType:             txType,
				Amount:             amount,
				RewardPointsEarned: rewardPoints,
				CashbackAmount:     cbAmount,
				UPIVPA:             cleaned.UPIVPA,
				CardLast4:          cleaned.CardLast4,
				IsTransfer:         cleaned.IsTransfer || isTransfer,
			})
		}
	}

	if len(transactions) > 0 {
		if meta.StartDate == "" {
			meta.StartDate = transactions[len(transactions)-1].Date
		}
		if meta.EndDate == "" {
			meta.EndDate = transactions[0].Date
		}
		if meta.StartDate > meta.EndDate {
			meta.StartDate, meta.EndDate = meta.EndDate, meta.StartDate
		}
	}

	return transactions, meta, nil
}
