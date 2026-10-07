package parser

import (
	"regexp"
	"strings"

	"local-finance/internal/models"
)

var (
	// UPI Patterns:
	upiSlashRegex1 = regexp.MustCompile(`(?i)UPI/(?:DR|CR)/(\d+)/([^/]+)/([^/]+)/([^/]+)`)
	upiSlashRegex2 = regexp.MustCompile(`(?i)UPI/(\d+)/([^/]+)/([^/]+)`)
	upiSlashRegex3 = regexp.MustCompile(`(?i)UPI/([^/]+)/([^/]+)`)

	// Card Mask Patterns (extract last 4)
	cardMaskRegex = regexp.MustCompile(`(?i)(?:X{4,}|[*]{4,})([0-9]{4})`)

	// POS / Card Swipe patterns:
	posPattern = regexp.MustCompile(`(?i)(?:POS|ECOM|IPS|PUR)\s*(?:\d+)?\s*[-/]?\s*([A-Z0-9\s\.\*]+)`)

	// IMPS / NEFT / RTGS
	impsPattern = regexp.MustCompile(`(?i)IMPS(?:-P2A)?-(\d+)-([^-]+)`)
	neftPattern = regexp.MustCompile(`(?i)NEFT\s*(?:CR|DR)?-([A-Z0-9]+)-([^-]+)`)
	rtgsPattern = regexp.MustCompile(`(?i)RTGS\s*(?:CR|DR)?-([A-Z0-9]+)-([^-]+)`)

	// ATM
	atmPattern = regexp.MustCompile(`(?i)(?:ATM|NWD|ATW|WDL|EAW)\s*[-/]?\s*([A-Z0-9\s]+)`)

	// Internal Transfers
	transferPattern = regexp.MustCompile(`(?i)(?:IB FUNDS TRANSFER|TPT|INT-TRF|TRANSFER TO|TRANSFER FROM)\s*(?:CR|DR)?-([^-]+)`)

	// Noise keywords and merchant prefixes to strip
	noiseRegex = regexp.MustCompile(`(?i)\b(PVT LTD|PRIVATE LIMITED|LTD|INDIA|BANGALORE|BENGALURU|MUMBAI|DELHI|GURGAON|HYDERABAD|CHENNAI|PUNE|UPI|PAYTM|PYTM|GPAY|PHONEPE|OKAXIS|OKHDFCBANK|OKICICI|OKSBI|IBL|YBL|AXISB)\b`)
	prefixNoiseRegex = regexp.MustCompile(`(?i)^(?:RAZ\*|RSP\*|PAY\*|BIL\*|CC\s*)`)
	refInParenRegex = regexp.MustCompile(`(?i)\(Ref#?\s*([A-Z0-9]+)\)`)
)

type CleanedNarration struct {
	CleanedPayee    string
	PaymentMode     models.PaymentMode
	ReferenceNumber string
	UPIVPA          *string
	CardLast4       *string
	IsTransfer      bool
}

var (
	timePrefixRegex = regexp.MustCompile(`^\d{1,2}:\d{2}(?::\d{2})?\s*`)
)

func CleanNarration(raw string) CleanedNarration {
	trimmed := strings.TrimSpace(raw)
	trimmed = timePrefixRegex.ReplaceAllString(trimmed, "")
	trimmed = strings.TrimSpace(trimmed)
	upper := strings.ToUpper(trimmed)
	res := CleanedNarration{
		CleanedPayee:    trimmed,
		PaymentMode:     models.PaymentModeOther,
		ReferenceNumber: "",
		// Detect explicit self transfers before payment-specific parsing returns.
		IsTransfer: strings.Contains(upper, "SELF TRANSFER"),
	}

	// Check card mask last 4 digits
	if matches := cardMaskRegex.FindStringSubmatch(trimmed); len(matches) > 1 {
		last4 := matches[1]
		res.CardLast4 = &last4
	}

	// Extract Ref# from parentheses if present (e.g. "BPPY CC PAYMENT DP... (Ref# ST262100083000010039463)")
	if refMatches := refInParenRegex.FindStringSubmatch(trimmed); len(refMatches) > 1 {
		res.ReferenceNumber = refMatches[1]
	}

	// 1. UPI Match
	if strings.HasPrefix(upper, "UPI") || strings.Contains(upper, "UPI/") || strings.Contains(upper, "UPI-") {
		res.PaymentMode = models.PaymentModeUPI

		// Try Hyphen Delimited (e.g. HDFC format)
		if strings.HasPrefix(upper, "UPI-") {
			parts := strings.Split(trimmed, "-")
			if len(parts) >= 2 {
				// Special check for UPI-LITE
				if strings.EqualFold(parts[1], "LITE") {
					res.CleanedPayee = "UPI Lite Balance"
					if len(parts) >= 4 {
						res.ReferenceNumber = parts[3]
					}
					res.IsTransfer = true
					return res
				}

				// Check for UPI-AUTOPAY-<Merchant>
				if strings.EqualFold(parts[1], "AUTOPAY") && len(parts) >= 3 {
					res.CleanedPayee = cleanMerchantName(parts[2])
					if len(parts) >= 4 && strings.Contains(parts[3], "@") {
						vpa := strings.ToLower(parts[3])
						res.UPIVPA = &vpa
					}
					if len(parts) >= 6 {
						res.ReferenceNumber = parts[5]
					}
					return res
				}

				// Standard: UPI-<Name>-<VPA/Acc>-<IFSC/Bank>-<RefNo>-<Remarks>
				res.CleanedPayee = cleanMerchantName(parts[1])
				if len(parts) >= 3 && strings.Contains(parts[2], "@") {
					vpa := strings.ToLower(parts[2])
					res.UPIVPA = &vpa
				}
				for _, part := range parts[2:] {
					partTrimmed := strings.TrimSpace(part)
					if len(partTrimmed) >= 10 && len(partTrimmed) <= 16 && isDigitsOnly(partTrimmed) {
						res.ReferenceNumber = partTrimmed
						break
					}
				}
				if strings.Contains(upper, "CRED.CLUB") || strings.Contains(upper, "CRED CLUB") {
					res.IsTransfer = true
				}
				return res
			}
		}

		// Try Slash Delimited
		if matches := upiSlashRegex1.FindStringSubmatch(trimmed); len(matches) > 4 {
			res.ReferenceNumber = matches[1]
			res.CleanedPayee = cleanMerchantName(matches[2])
			if strings.Contains(matches[4], "@") {
				vpa := strings.ToLower(matches[4])
				res.UPIVPA = &vpa
			}
			return res
		}
		if matches := upiSlashRegex2.FindStringSubmatch(trimmed); len(matches) > 3 {
			res.ReferenceNumber = matches[1]
			res.CleanedPayee = cleanMerchantName(matches[2])
			if strings.Contains(matches[3], "@") {
				vpa := strings.ToLower(matches[3])
				res.UPIVPA = &vpa
			}
			return res
		}
		if matches := upiSlashRegex3.FindStringSubmatch(trimmed); len(matches) > 2 {
			res.CleanedPayee = cleanMerchantName(matches[1])
			if strings.Contains(matches[2], "@") {
				vpa := strings.ToLower(matches[2])
				res.UPIVPA = &vpa
			}
			return res
		}
	}

	// 2. POS / Card swipes / E-Commerce
	if strings.HasPrefix(upper, "POS") || strings.HasPrefix(upper, "ECOM") || strings.Contains(upper, "PURCHASE") {
		res.PaymentMode = models.PaymentModeCardPOS
		if matches := posPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			res.CleanedPayee = cleanMerchantName(matches[1])
			return res
		}
	}

	// 3. IMPS
	if strings.Contains(upper, "IMPS") {
		res.PaymentMode = models.PaymentModeIMPS
		if matches := impsPattern.FindStringSubmatch(trimmed); len(matches) > 2 {
			res.ReferenceNumber = matches[1]
			res.CleanedPayee = cleanMerchantName(matches[2])
			return res
		}
	}

	// 4. NEFT
	if strings.Contains(upper, "NEFT") {
		res.PaymentMode = models.PaymentModeNEFT
		if strings.Contains(upper, "SALARY") {
			res.PaymentMode = models.PaymentModeSalary
		}
		if matches := neftPattern.FindStringSubmatch(trimmed); len(matches) > 2 {
			res.ReferenceNumber = matches[1]
			res.CleanedPayee = cleanMerchantName(matches[2])
			return res
		}
	}

	// 5. RTGS
	if strings.Contains(upper, "RTGS") {
		res.PaymentMode = models.PaymentModeRTGS
		if matches := rtgsPattern.FindStringSubmatch(trimmed); len(matches) > 2 {
			res.ReferenceNumber = matches[1]
			res.CleanedPayee = cleanMerchantName(matches[2])
			return res
		}
	}

	// 6. ATM / Cash Withdrawal
	if strings.HasPrefix(upper, "EAW") || strings.Contains(upper, "ATM") || strings.Contains(upper, "NWD") || strings.Contains(upper, "CASH WDL") {
		res.PaymentMode = models.PaymentModeATM
		res.CleanedPayee = "ATM Cash Withdrawal"
		return res
	}

	// 7. Internal Transfers / TPT / NetBanking Transfers
	if strings.Contains(upper, "IB FUNDS TRANSFER") || strings.Contains(upper, "TPT") || strings.Contains(upper, "INT-TRF") || strings.Contains(upper, "SELF TRANSFER") {
		res.PaymentMode = models.PaymentModeOther
		res.IsTransfer = true
		parts := strings.Split(trimmed, "-")
		if len(parts) >= 3 {
			res.CleanedPayee = cleanMerchantName(parts[len(parts)-1])
			return res
		} else if len(parts) == 2 {
			res.CleanedPayee = cleanMerchantName(parts[1])
			return res
		}
		if matches := transferPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			res.CleanedPayee = cleanMerchantName(matches[1])
			return res
		}
		res.CleanedPayee = "Self / Internal Account Transfer"
		return res
	}

	// 8. Credit Card Bill Payment / BBPS / BPPY / Payment received
	if strings.Contains(upper, "BPPY CC PAYMENT") || strings.Contains(upper, "BBPS PAYMENT") ||
		strings.Contains(upper, "PAYMENT RECEIVED") || strings.Contains(upper, "AUTO DEBIT") ||
		strings.Contains(upper, "AUTOPAY DEBIT") || strings.Contains(upper, "CRED CC PAYMENT") {
		res.PaymentMode = models.PaymentModeOther
		res.IsTransfer = true
		res.CleanedPayee = "Credit Card Bill Payment"
		return res
	}

	// 9. Cashback / Reward credits
	if strings.Contains(upper, "CASHBACK") || strings.Contains(upper, "CASH BACK") {
		res.PaymentMode = models.PaymentModeOther
		res.IsTransfer = false
		if strings.Contains(upper, "SWIGGY") {
			res.CleanedPayee = "Swiggy Cashback"
		} else {
			res.CleanedPayee = cleanMerchantName(trimmed)
		}
		return res
	}

	// 10. Salary
	if strings.Contains(upper, "SALARY") || strings.Contains(upper, "SAL CREDIT") {
		res.PaymentMode = models.PaymentModeSalary
		res.CleanedPayee = "Salary Credit"
		return res
	}

	// 11. Bank Charges / Interest
	if strings.Contains(upper, "CHG") || strings.Contains(upper, "FEE") || strings.Contains(upper, "GST") {
		res.PaymentMode = models.PaymentModeCharges
		res.CleanedPayee = "Bank Charges / Taxes"
		return res
	}
	if strings.Contains(upper, "INT.PD") || strings.Contains(upper, "INTEREST") {
		res.PaymentMode = models.PaymentModeInterest
		res.CleanedPayee = "Interest Earned"
		return res
	}

	// Default cleanup
	res.CleanedPayee = cleanMerchantName(trimmed)
	return res
}

func cleanMerchantName(name string) string {
	cleaned := prefixNoiseRegex.ReplaceAllString(name, "")
	cleaned = noiseRegex.ReplaceAllString(cleaned, "")
	cleaned = strings.ReplaceAll(cleaned, "/", " ")
	cleaned = strings.ReplaceAll(cleaned, "-", " ")
	cleaned = strings.ReplaceAll(cleaned, "_", " ")
	cleaned = strings.ReplaceAll(cleaned, "*", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	if len(cleaned) == 0 {
		return strings.TrimSpace(name)
	}
	return strings.Title(strings.ToLower(cleaned))
}

func isDigitsOnly(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
