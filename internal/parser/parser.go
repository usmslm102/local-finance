package parser

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

type StatementType string

const (
	TypeSavingsCSV    StatementType = "SAVINGS_CSV"
	TypeCreditCardCSV StatementType = "CREDIT_CARD_CSV"
	TypeSavingsPDF    StatementType = "SAVINGS_PDF"
	TypeCreditCardPDF StatementType = "CREDIT_CARD_PDF"
	TypeGenericCSV    StatementType = "GENERIC_CSV"
)

type ParseOptions struct {
	AccountID string
	Password  string
	Filename  string
}

type StatementMeta struct {
	BankName             string
	AccountType          models.AccountType
	AccountNumber        string
	AccountNumberMask    string
	AccountHolderName    string
	StartDate            string
	EndDate              string
	OpeningBalance       float64
	ClosingBalance       float64
	TotalDebits          float64
	TotalCredits         float64
	StatementFormat      string
	CustomerID           string
	IFSCCode             string
	BranchName           string
	CardNetwork          string // VISA, MASTERCARD, RUPAY, AMEX
	CardVariant          string // Regalia, Swiggy, Amazon Pay, Flipkart
	StatementDate        string
	PaymentDueDate       string
	TotalDueAmount       float64
	MinimumDueAmount     float64
	RewardPointsEarned   float64
	RewardPointsBalance  float64
	CashbackEarned       float64
	CashbackCredited     float64
	FinanceCharges       float64
	CreditLimit          float64
	AvailableCreditLimit float64
}

type ParsedTransaction struct {
	Date               string // YYYY-MM-DD
	ValueDate          *string
	RawNarration       string
	CleanedPayee       string
	PaymentMode        models.PaymentMode
	ReferenceNumber    string
	TxType             models.TxType // DEBIT or CREDIT
	Amount             float64
	RunningBalance     *float64
	UPIVPA             *string
	CardLast4          *string
	MerchantCategory   string
	CashbackAmount     float64
	RewardPointsEarned float64
	IsTransfer         bool
	OriginalCurrency   *string
	OriginalAmount     *float64
}

// StatementParser is the generic interface every bank plugin implements
type StatementParser interface {
	ID() string
	Name() string
	SupportedTypes() []StatementType
	CanParse(filename string, sample []byte) (confidence float64, bankName string, accType models.AccountType)
	Parse(r io.Reader, opts ParseOptions) ([]ParsedTransaction, StatementMeta, error)
}

type Registry struct {
	parsers map[string]StatementParser
}

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{
		parsers: make(map[string]StatementParser),
	}
}

func (r *Registry) Register(p StatementParser) {
	r.parsers[p.ID()] = p
}

func (r *Registry) Get(id string) (StatementParser, bool) {
	p, ok := r.parsers[id]
	return p, ok
}

func (r *Registry) List() []StatementParser {
	list := make([]StatementParser, 0, len(r.parsers))
	for _, p := range r.parsers {
		list = append(list, p)
	}
	return list
}

func (r *Registry) Detect(filename string, sample []byte) (StatementParser, float64, StatementMeta) {
	isPDF := bytes.HasPrefix(sample, []byte("%PDF")) || strings.HasSuffix(strings.ToUpper(filename), ".PDF")
	// PDF content streams are usually compressed. Inspect their extracted text
	// once for every adapter, rather than relying on institution-specific names.
	if bytes.HasPrefix(sample, []byte("%PDF")) {
		if text, err := extractor.ExtractPDFText(bytes.NewReader(sample), ""); err == nil {
			sample = []byte("%PDF\n" + text)
		}
	}
	var bestParser StatementParser
	var highestConfidence float64
	var bestMeta StatementMeta

	for _, p := range r.parsers {
		if isPDF {
			supportsPDF := false
			for _, typ := range p.SupportedTypes() {
				if typ == TypeSavingsPDF || typ == TypeCreditCardPDF {
					supportsPDF = true
					break
				}
			}
			if !supportsPDF {
				continue
			}
		}
		conf, bankName, accType := p.CanParse(filename, sample)
		if conf > highestConfidence {
			highestConfidence = conf
			bestParser = p
			bestMeta = StatementMeta{
				BankName:    bankName,
				AccountType: accType,
			}
		}
	}

	return bestParser, highestConfidence, bestMeta
}

// Standard date normalizer for Indian Bank Statements
func NormalizeDate(d string) string {
	d = strings.TrimSpace(d)
	// Formats: DD/MM/YYYY, DD-MM-YYYY, DD/MM/YY, DD-MM-YY, YYYY-MM-DD
	parts := strings.FieldsFunc(d, func(r rune) bool {
		return r == '/' || r == '-' || r == '.'
	})

	if len(parts) == 3 {
		if len(parts[0]) == 4 {
			// YYYY-MM-DD
			return fmt.Sprintf("%04s-%02s-%02s", parts[0], parts[1], parts[2])
		}
		year := parts[2]
		if len(year) == 2 {
			year = "20" + year
		}
		// DD-MM-YYYY -> YYYY-MM-DD
		return fmt.Sprintf("%04s-%02s-%02s", year, parts[1], parts[0])
	}
	return d
}
