package investment

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"local-finance/internal/models"
)

// INDmoney's US holdings export reports current values, without acquisition costs.
type INDmoneyHoldingsParser struct{}

func init() {
	DefaultRegistry.Register(INDmoneyHoldingsParser{})
}

func (INDmoneyHoldingsParser) ID() string { return "indmoney_us_holdings_xls_v1" }
func (INDmoneyHoldingsParser) Info() ParserInfo {
	return ParserInfo{Provider: "INDmoney", Name: "US stock holdings export", Extensions: []string{".xls"}, Requirements: []string{"Original INDmoney US stock holdings export in legacy .xls format, with INDmoney identification, Broker Account and Holdings as on YYYY-MM-DD.", "Holdings columns: Stock Symbol, Holding Since, Quantity, Avg. Price ($), Total Value ($).", "Values remain in USD. Acquisition costs and investment returns are unavailable; do not infer them from current average price."}}
}

var indmoneyHeaders = []string{"Stock Symbol", "Holding Since", "Quantity", "Avg. Price ($)", "Total Value ($)"}

func indmoneyHeader(row []string) bool {
	cols := headerIndexMap(row)
	for _, name := range indmoneyHeaders {
		if _, ok := cols[name]; !ok {
			return false
		}
	}
	return true
}

func (INDmoneyHoldingsParser) CanParse(filename string, data []byte) bool {
	if !strings.EqualFold(filepath.Ext(filename), ".xls") {
		return false
	}
	sheets, err := readLegacyInvestmentSheets(data)
	return err == nil && isINDmoneyHoldings(sheets)
}

func isINDmoneyHoldings(sheets []models.InvestmentSheet) bool {
	brand, header, account, date := false, false, false, false
	for _, sheet := range sheets {
		for _, row := range sheet.Rows {
			if indmoneyHeader(row) {
				header = true
			}
			if cell(row, 0) == "Broker Account" && cell(row, 1) != "" {
				account = true
			}
			if cell(row, 0) == "Holdings as on" && cell(row, 1) != "" {
				date = true
			}
			for _, v := range row {
				if strings.Contains(strings.ToLower(v), "indmoney") {
					brand = true
				}
			}
		}
	}
	return brand && header && account && date
}

func (INDmoneyHoldingsParser) Parse(data []byte) (*models.InvestmentSnapshot, error) {
	sheets, err := readLegacyInvestmentSheets(data)
	if err != nil {
		return nil, err
	}
	return parseINDmoneyHoldings(sheets)
}

func parseINDmoneyHoldings(sheets []models.InvestmentSheet) (*models.InvestmentSnapshot, error) {
	if !isINDmoneyHoldings(sheets) {
		return nil, fmt.Errorf("not an INDmoney US holdings report")
	}
	out := &models.InvestmentSnapshot{Provider: "INDmoney", Currency: "USD", Sheets: sheets, Holdings: []models.InvestmentHolding{}, Warnings: []string{
		"This statement provides current holding values in USD. Acquisition costs are not provided, so invested amount and returns are unavailable. No currency conversion is applied.",
	}}
	out.CurrentValue = new(float64)
	seen := map[string]bool{}
	for _, sheet := range sheets {
		header := -1
		var cols map[string]int
		for i, row := range sheet.Rows {
			switch cell(row, 0) {
			case "Broker Account":
				ref := cell(row, 1)
				if out.AccountRef != "" && out.AccountRef != ref {
					return nil, fmt.Errorf("inconsistent broker accounts")
				}
				out.AccountRef = ref
			case "Holdings as on":
				date := cell(row, 1)
				if _, err := time.Parse("2006-01-02", date); err != nil {
					return nil, fmt.Errorf("invalid holdings date")
				}
				if out.AsOf != "" && out.AsOf != date {
					return nil, fmt.Errorf("inconsistent holdings dates")
				}
				out.AsOf = date
			}
			if indmoneyHeader(row) {
				if header >= 0 {
					return nil, fmt.Errorf("multiple holdings tables in one worksheet")
				}
				header, cols = i, headerIndexMap(row)
			}
		}
		if header < 0 {
			continue
		}
		for i, row := range sheet.Rows[header+1:] {
			if strings.HasPrefix(strings.ToLower(cell(row, 0)), "disclaimer") {
				break
			}
			if len(headerIndexMap(row)) == 0 {
				continue
			}
			h := models.InvestmentHolding{Symbol: cell(row, cols["Stock Symbol"]), AssetClass: "US Stock", Fields: map[string]string{}}
			if h.Symbol == "" {
				return nil, fmt.Errorf("missing stock symbol in %s row %d", sheet.Name, header+i+2)
			}
			key := strings.ToUpper(h.Symbol)
			if seen[key] {
				return nil, fmt.Errorf("duplicate holding %s", h.Symbol)
			}
			seen[key] = true
			for name, c := range cols {
				h.Fields[name] = cell(row, c)
			}
			h.CurrentValue = new(float64)
			h.ClosingPrice = new(float64)
			for name, target := range map[string]*float64{"Quantity": &h.Quantity, "Avg. Price ($)": h.ClosingPrice, "Total Value ($)": h.CurrentValue} {
				n, err := number(h.Fields[name])
				if err != nil || n < 0 {
					return nil, fmt.Errorf("invalid %s for %s", name, h.Symbol)
				}
				*target = n
			}
			value := h.Quantity * *h.ClosingPrice
			if math.IsInf(value, 0) || math.Abs(value-*h.CurrentValue) > 0.01 {
				return nil, fmt.Errorf("holding value does not reconcile for %s", h.Symbol)
			}
			*out.CurrentValue += *h.CurrentValue
			if math.IsInf(*out.CurrentValue, 0) {
				return nil, fmt.Errorf("invalid portfolio value total")
			}
			out.Holdings = append(out.Holdings, h)
		}
	}
	if out.AccountRef == "" || out.AsOf == "" || len(out.Holdings) == 0 {
		return nil, fmt.Errorf("missing account, date, or holdings")
	}
	return out, nil
}
