package investment

import (
	"bytes"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"local-finance/internal/models"
)

type ZerodhaHoldingsParser struct{}

func init() {
	DefaultRegistry.Register(ZerodhaHoldingsParser{})
}

func (ZerodhaHoldingsParser) Info() ParserInfo {
	return ParserInfo{Provider: "Zerodha", Name: "Holdings export", Extensions: []string{".xlsx"}, Requirements: []string{"Original Zerodha holdings export with Client ID and Holdings statement as on YYYY-MM-DD.", "Holdings table columns: Symbol, ISIN, Quantity Available, Average Price, Previous Closing Price, Unrealized P&L.", "Include the complete portfolio; leave source worksheets and values intact."}}
}

func (ZerodhaHoldingsParser) ID() string { return "zerodha_holdings_xlsx_v1" }

var holdingsDate = regexp.MustCompile(`(?i)holdings statement as on (\d{4}-\d{2}-\d{2})`)

func openHoldings(data []byte) (*excelize.File, error) {
	return excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 16 << 20})
}
func (ZerodhaHoldingsParser) CanParse(filename string, data []byte) bool {
	if !strings.EqualFold(filepath.Ext(filename), ".xlsx") {
		return false
	}
	f, err := openHoldings(data)
	if err != nil {
		return false
	}
	defer f.Close()
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		client, title, header := false, false, false
		for _, row := range rows {
			for _, v := range row {
				if v == "Client ID" {
					client = true
				}
				if holdingsDate.MatchString(v) {
					title = true
				}
			}
			cols := headerIndexMap(row)
			_, symbol := cols["Symbol"]
			_, isin := cols["ISIN"]
			_, price := cols["Previous Closing Price"]
			if symbol && isin && price {
				header = true
			}
		}
		if client && title && header {
			return true
		}
	}
	return false
}
func (ZerodhaHoldingsParser) Parse(data []byte) (*models.InvestmentSnapshot, error) {
	f, err := openHoldings(data)
	if err != nil {
		return nil, fmt.Errorf("cannot open investment workbook: %w", err)
	}
	defer f.Close()
	out := &models.InvestmentSnapshot{Provider: "Zerodha", Currency: "INR", Holdings: []models.InvestmentHolding{}, Sheets: []models.InvestmentSheet{}, Warnings: []string{}}
	out.InvestedValue, out.CurrentValue, out.UnrealizedReturn = new(float64), new(float64), new(float64)
	// Prefer the consolidated sheet. Asset-specific sheets repeat these holdings.
	combined := false
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, "Combined") {
			combined = true
		}
	}
	seen := map[string]bool{}
	var reportedCost, reportedValue *float64
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", name, err)
		}
		for i := range rows {
			if rows[i] == nil {
				rows[i] = []string{}
			}
		}
		out.Sheets = append(out.Sheets, models.InvestmentSheet{Name: name, Rows: rows})
		selected := !combined || strings.EqualFold(name, "Combined")
		header := -1
		var cols map[string]int
		for i, row := range rows {
			for j, v := range row {
				if v == "Client ID" {
					ref := cell(row, j+1)
					if out.AccountRef != "" && out.AccountRef != ref {
						return nil, fmt.Errorf("inconsistent client IDs")
					}
					out.AccountRef = ref
				}
				if m := holdingsDate.FindStringSubmatch(v); len(m) > 1 {
					if _, err := time.Parse("2006-01-02", m[1]); err != nil {
						return nil, fmt.Errorf("invalid holdings date")
					}
					if out.AsOf != "" && out.AsOf != m[1] {
						return nil, fmt.Errorf("inconsistent valuation dates")
					}
					out.AsOf = m[1]
				}
				if selected && combined && (v == "Invested Value" || v == "Present Value") {
					n, err := number(cell(row, j+1))
					if err != nil {
						return nil, err
					}
					if v == "Invested Value" {
						reportedCost = &n
					} else {
						reportedValue = &n
					}
				}
			}
			candidate := headerIndexMap(row)
			if _, ok := candidate["ISIN"]; ok {
				if _, ok := candidate["Symbol"]; ok {
					header = i
					cols = candidate
					break
				}
			}
		}
		if !selected {
			continue
		}
		if header < 0 {
			if combined || strings.EqualFold(name, "Equity") || strings.EqualFold(name, "Mutual Funds") {
				return nil, fmt.Errorf("sheet %s has no holdings table", name)
			}
			// Summary and disclaimer worksheets are retained for inspection.
			// A detected holdings table must still have all required columns.
			continue
		}
		for _, required := range []string{"Quantity Available", "Average Price", "Previous Closing Price", "Unrealized P&L"} {
			if _, ok := cols[required]; !ok {
				return nil, fmt.Errorf("sheet %s missing %s", name, required)
			}
		}
		for i, row := range rows[header+1:] {
			symbol, isin := cell(row, cols["Symbol"]), cell(row, cols["ISIN"])
			if symbol == "" && isin == "" {
				continue
			}
			if symbol == "" || isin == "" {
				return nil, fmt.Errorf("incomplete holding on sheet %s row %d", name, header+i+2)
			}
			if seen[isin] {
				return nil, fmt.Errorf("duplicate ISIN in holdings: %s", isin)
			}
			seen[isin] = true
			h := models.InvestmentHolding{Symbol: symbol, ISIN: isin, AssetClass: "Equity", Fields: map[string]string{}}
			for key, col := range cols {
				h.Fields[key] = cell(row, col)
			}
			h.ClosingPrice, h.CurrentValue, h.UnrealizedReturn = new(float64), new(float64), new(float64)
			h.InvestedValue = new(float64)
			h.AveragePrice = new(float64)
			values := map[string]*float64{"Quantity Available": &h.Quantity, "Average Price": h.AveragePrice, "Previous Closing Price": h.ClosingPrice, "Unrealized P&L": h.UnrealizedReturn}
			for key, target := range values {
				n, err := number(h.Fields[key])
				if err != nil {
					return nil, fmt.Errorf("%s row %d %s: %w", name, header+i+2, key, err)
				}
				*target = n
			}
			for _, key := range []string{"Quantity Pledged (Margin)", "Quantity Pledged (Loan)"} {
				if v := h.Fields[key]; v != "" && v != "-" {
					n, err := number(v)
					if err != nil || n < 0 {
						return nil, fmt.Errorf("invalid pledged quantity")
					}
					h.Quantity += n
				}
			}
			if h.Quantity < 0 || *h.AveragePrice < 0 || *h.ClosingPrice < 0 {
				return nil, fmt.Errorf("negative holding quantity or price")
			}
			instrumentType := h.Fields["Instrument Type"]
			if strings.EqualFold(name, "Mutual Funds") || strings.EqualFold(instrumentType, "MF") || strings.EqualFold(instrumentType, "Mutual Fund") {
				h.AssetClass = "Mutual Fund"
			}
			*h.CurrentValue = h.Quantity * *h.ClosingPrice
			// Reported P&L retains cost precision lost in the rounded average price.
			*h.InvestedValue = *h.CurrentValue - *h.UnrealizedReturn
			if *h.InvestedValue < 0 {
				return nil, fmt.Errorf("invalid negative cost basis for %s", symbol)
			}
			h.ReturnPercent = models.InvestmentReturnPercent(*h.InvestedValue, *h.UnrealizedReturn)
			out.Holdings = append(out.Holdings, h)
			*out.InvestedValue += *h.InvestedValue
			*out.CurrentValue += *h.CurrentValue
			*out.UnrealizedReturn += *h.UnrealizedReturn
		}
	}
	if out.AccountRef == "" || out.AsOf == "" || len(out.Holdings) == 0 {
		return nil, fmt.Errorf("missing account, valuation date, or holdings")
	}
	if reportedCost != nil && math.Abs(*reportedCost-*out.InvestedValue) > 1 || reportedValue != nil && math.Abs(*reportedValue-*out.CurrentValue) > 1 {
		return nil, fmt.Errorf("holdings do not reconcile with statement totals")
	}
	out.ReturnPercent = models.InvestmentReturnPercent(*out.InvestedValue, *out.UnrealizedReturn)
	return out, nil
}
