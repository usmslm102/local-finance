package parser

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"local-finance/internal/models"
)

const LocalFinanceCSVParserID = "localfinance_csv_v1"

var csvMoney = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]{1,2})?$`)
var localCSVRequired = []string{"bank_name", "account_type", "account_number_mask", "date", "narration", "amount", "tx_type"}
var localCSVOptional = []string{"account_number", "reference_number", "value_date", "running_balance", "cleaned_payee"}

// LocalFinanceCSVParser accepts verified agent transcriptions through the normal import path.
type LocalFinanceCSVParser struct{}

func init()                                 { DefaultRegistry.Register(&LocalFinanceCSVParser{}) }
func (*LocalFinanceCSVParser) ID() string   { return LocalFinanceCSVParserID }
func (*LocalFinanceCSVParser) Name() string { return "LocalFinance transaction CSV v1" }
func (*LocalFinanceCSVParser) SupportedTypes() []StatementType {
	return []StatementType{TypeGenericCSV}
}
func (*LocalFinanceCSVParser) CanParse(filename string, sample []byte) (float64, string, models.AccountType) {
	if !strings.EqualFold(filepath.Ext(filename), ".csv") {
		return 0, "", ""
	}
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(sample, []byte{0xef, 0xbb, 0xbf})))
	header, err := reader.Read()
	if err != nil {
		return 0, "", ""
	}
	if _, err := localCSVColumns(header); err != nil {
		return 0, "", ""
	}
	return 1, "", ""
}

func localCSVColumns(header []string) (map[string]int, error) {
	allowed := map[string]bool{}
	for _, column := range append(append([]string{}, localCSVRequired...), localCSVOptional...) {
		allowed[column] = true
	}
	columns := map[string]int{}
	for i, name := range header {
		if !allowed[name] {
			return nil, fmt.Errorf("unknown CSV column at position %d", i+1)
		}
		if _, exists := columns[name]; exists {
			return nil, fmt.Errorf("duplicate CSV column at position %d", i+1)
		}
		columns[name] = i
	}
	for _, name := range localCSVRequired {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("missing CSV column %s", name)
		}
	}
	return columns, nil
}

func (*LocalFinanceCSVParser) Parse(r io.Reader, opts ParseOptions) ([]ParsedTransaction, StatementMeta, error) {
	// The import module already bounds input; accept an optional UTF-8 BOM.
	reader := csv.NewReader(r)
	header, err := reader.Read()
	if err != nil {
		return nil, StatementMeta{}, fmt.Errorf("invalid CSV header")
	}
	if len(header) > 0 {
		header[0] = strings.TrimPrefix(header[0], "\ufeff")
	}
	columns, err := localCSVColumns(header)
	if err != nil {
		return nil, StatementMeta{}, err
	}
	meta := StatementMeta{StatementFormat: "LOCALFINANCE_CSV_V1"}
	var transactions []ParsedTransaction
	for rowNumber := 2; ; rowNumber++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, StatementMeta{}, fmt.Errorf("invalid CSV row %d", rowNumber)
		}
		for _, value := range row {
			if !utf8.ValidString(value) {
				return nil, StatementMeta{}, fmt.Errorf("CSV row %d must be UTF-8", rowNumber)
			}
		}
		get := func(name string) string {
			if index, ok := columns[name]; ok {
				return strings.TrimSpace(row[index])
			}
			return ""
		}
		fail := func(field string) ([]ParsedTransaction, StatementMeta, error) {
			return nil, StatementMeta{}, fmt.Errorf("invalid %s in CSV row %d", field, rowNumber)
		}
		bank, kind, mask, number := get("bank_name"), models.AccountType(get("account_type")), get("account_number_mask"), get("account_number")
		if bank == "" || mask == "" {
			return fail("account metadata")
		}
		switch kind {
		case models.AccountTypeSavings, models.AccountTypeCurrent, models.AccountTypeCreditCard, models.AccountTypeWallet:
		default:
			return fail("account_type")
		}
		if len(transactions) == 0 {
			meta.BankName, meta.AccountType, meta.AccountNumberMask, meta.AccountNumber = bank, kind, mask, number
		} else if bank != meta.BankName || kind != meta.AccountType || mask != meta.AccountNumberMask || number != meta.AccountNumber {
			return fail("mixed account metadata")
		}
		date := get("date")
		if !validCSVDate(date) {
			return fail("date")
		}
		narration := get("narration")
		if narration == "" {
			return fail("narration")
		}
		amount, err := parseCSVMoney(get("amount"))
		if err != nil || amount <= 0 {
			return fail("amount")
		}
		txType := models.TxType(get("tx_type"))
		if txType != models.TxTypeDebit && txType != models.TxTypeCredit {
			return fail("tx_type")
		}
		cleaned := CleanNarration(narration)
		pt := ParsedTransaction{Date: NormalizeDate(date), RawNarration: narration, Amount: amount, TxType: txType,
			CleanedPayee: cleaned.CleanedPayee, ReferenceNumber: get("reference_number"), PaymentMode: cleaned.PaymentMode,
			UPIVPA: cleaned.UPIVPA, CardLast4: cleaned.CardLast4, IsTransfer: cleaned.IsTransfer}
		if pt.ReferenceNumber == "" {
			pt.ReferenceNumber = cleaned.ReferenceNumber
		}
		if value := get("cleaned_payee"); value != "" {
			pt.CleanedPayee = value
		}
		if value := get("value_date"); value != "" {
			if !validCSVDate(value) {
				return fail("value_date")
			}
			pt.ValueDate = &value
		}
		if value := get("running_balance"); value != "" {
			balance, err := parseCSVMoney(value)
			if err != nil {
				return fail("running_balance")
			}
			pt.RunningBalance = &balance
		}
		if meta.StartDate == "" || date < meta.StartDate {
			meta.StartDate = date
		}
		if date > meta.EndDate {
			meta.EndDate = date
		}
		if txType == models.TxTypeDebit {
			meta.TotalDebits += amount
		} else {
			meta.TotalCredits += amount
		}
		transactions = append(transactions, pt)
	}
	if len(transactions) == 0 {
		return nil, StatementMeta{}, fmt.Errorf("CSV contains no transactions")
	}
	return transactions, meta, nil
}

func validCSVDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}
func parseCSVMoney(value string) (float64, error) {
	if !csvMoney.MatchString(value) {
		return 0, fmt.Errorf("invalid decimal")
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(amount, 0) || math.IsNaN(amount) {
		return 0, fmt.Errorf("invalid decimal")
	}
	return amount, nil
}
