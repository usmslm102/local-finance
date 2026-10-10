package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
	"local-finance/internal/api"
	"local-finance/internal/investment"
	"local-finance/internal/models"
	"local-finance/internal/service"
)

// Entirely fictional data. Never copy a user's workbook into test fixtures.
func investmentWorkbook(t *testing.T, modify func(*excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", "Combined"); err != nil {
		t.Fatal(err)
	}
	rows := [][]interface{}{
		{"Client ID", "DEMO0001"},
		{"Combined Holdings Statement as on 2026-04-01"},
		{"Invested Value", 800}, {"Present Value", 900},
		{"Symbol", "ISIN", "Sector", "Instrument Type", "Quantity Available", "Quantity Pledged (Margin)", "Quantity Pledged (Loan)", "Average Price", "Previous Closing Price", "Unrealized P&L", "Quantity Discrepant", "Quantity Long Term"},
		{"DEMO EQUITY", "INE000DEMO01", "Example sector", "-", 2, 1, 0, 100, 90, -30, 0, 1},
		{"DEMO FUND", "INF000DEMO01", "-", "MF", 5, 0, 0, 100, 126, 130, 0, "-"},
	}
	for i, row := range rows {
		axis, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow("Combined", axis, &row); err != nil {
			t.Fatal(err)
		}
	}
	index, err := f.NewSheet("Equity")
	if err != nil {
		t.Fatal(err)
	}
	_ = index
	for i, row := range rows[:6] {
		axis, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := f.SetSheetRow("Equity", axis, &row); err != nil {
			t.Fatal(err)
		}
	}
	if modify != nil {
		modify(f)
	}
	// Leading empty rows mirror broker exports and must survive JSON/UI handling.
	for _, name := range []string{"Combined", "Equity"} {
		if err := f.InsertRows(name, 1, 2); err != nil {
			t.Fatal(err)
		}
	}
	data, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestInvestmentImportIsolationAndHistory(t *testing.T) {
	database := testDatabase(t)
	svc := service.NewInvestmentService(database)
	data := investmentWorkbook(t, nil)
	preview, err := svc.Preview("demo-holdings.xlsx", bytes.NewReader(data))
	if err != nil || preview.ID != "" || len(preview.Holdings) != 2 {
		t.Fatalf("incorrect preview: %+v %v", preview, err)
	}
	before, err := database.ListInvestmentSnapshots()
	if err != nil || len(before) != 0 {
		t.Fatal("preview persisted an investment", err)
	}
	snapshot, duplicate, err := svc.Import("demo-holdings.xlsx", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if duplicate || snapshot.Provider != "Zerodha" || len(snapshot.Holdings) != 2 || len(snapshot.Sheets) != 2 || snapshot.InvestedValue == nil || *snapshot.InvestedValue != 800 || snapshot.CurrentValue == nil || *snapshot.CurrentValue != 900 || snapshot.UnrealizedReturn == nil || *snapshot.UnrealizedReturn != 100 || snapshot.ReturnPercent == nil || *snapshot.ReturnPercent != 12.5 {
		t.Fatalf("incorrect portfolio: %+v", snapshot)
	}
	if snapshot.Holdings[0].Quantity != 3 || snapshot.Holdings[0].Fields["Quantity Long Term"] != "1" {
		t.Fatal("pledged quantities or source fields lost")
	}
	repeated, duplicate, err := svc.Import("renamed.xlsx", bytes.NewReader(data))
	if err != nil || !duplicate || repeated.ID != snapshot.ID {
		t.Fatalf("non-idempotent import: %+v %v %v", repeated, duplicate, err)
	}
	newer := investmentWorkbook(t, func(f *excelize.File) {
		for _, name := range []string{"Combined", "Equity"} {
			if err := f.SetCellStr(name, "A2", "Holdings Statement as on 2026-04-02"); err != nil {
				t.Fatal(err)
			}
		}
	})
	if _, _, err := svc.Import("later.xlsx", bytes.NewReader(newer)); err != nil {
		t.Fatal(err)
	}
	list, err := database.ListInvestmentSnapshots()
	if err != nil || len(list) != 2 || list[0].AsOf != "2026-04-02" || list[0].CurrentValue == nil || *list[0].CurrentValue != 900 {
		t.Fatalf("snapshot history incorrect: %+v %v", list, err)
	}
	analytics, err := database.GetAnalyticsOverview()
	if err != nil {
		t.Fatal(err)
	}
	accounts, err := database.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if analytics.TotalTransactions != 0 || analytics.TotalExpense != 0 || analytics.TotalIncome != 0 || len(accounts) != 0 {
		t.Fatal("investments changed bank ledger or income/expenses")
	}
	exported, err := database.ExportAllDataJSON()
	if err != nil || len(exported.Investments) != 2 {
		t.Fatal("investments missing from JSON export", err)
	}
	if ok, err := database.DeleteInvestmentSnapshot(snapshot.ID); err != nil || !ok {
		t.Fatal("snapshot deletion failed", err)
	}
	if err := database.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	list, err = database.ListInvestmentSnapshots()
	if err != nil || len(list) != 0 {
		t.Fatal("reset left investment records", err)
	}
}

func TestInvestmentParserRejectsInvalidWorkbooks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		modify func(*excelize.File)
	}{
		{"malformed quantity", func(f *excelize.File) { f.SetCellStr("Combined", "E6", "unknown") }},
		{"missing price header", func(f *excelize.File) { f.SetCellStr("Combined", "I5", "Other field") }},
		{"mismatched totals", func(f *excelize.File) { f.SetCellInt("Combined", "B4", 10000) }},
		{"duplicate holding", func(f *excelize.File) { f.SetCellStr("Combined", "B7", "INE000DEMO01") }},
		{"inconsistent account", func(f *excelize.File) { f.SetCellStr("Equity", "B1", "OTHER0001") }},
		{"invalid date", func(f *excelize.File) { f.SetCellStr("Combined", "A2", "Holdings Statement as on 2026-99-99") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := investment.DefaultRegistry.Parse("holdings.xlsx", investmentWorkbook(t, tc.modify)); err == nil {
				t.Fatal("invalid workbook accepted")
			}
		})
	}
	if _, err := investment.DefaultRegistry.Parse("bank.xlsx", []byte("not an xlsx")); err == nil {
		t.Fatal("non-workbook accepted")
	}
	if models.InvestmentReturnPercent(0, 0) != nil {
		t.Fatal("zero cost should have undefined return percentage")
	}
}

func TestInvestmentAPI(t *testing.T) {
	database := testDatabase(t)
	router := api.SetupRouter(database, service.NewTransactionService(database), nil)
	upload := func(path string, data []byte, filenames ...string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		filename := "fictional.xlsx"
		if len(filenames) > 0 {
			filename = filenames[0]
		}
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		writer.Close()
		req := localTestRequest(http.MethodPost, path, &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest(http.MethodGet, "/api/investments/formats", nil))
	var formats struct {
		Parsers []investment.ParserInfo `json:"parsers"`
		MaxSize int                     `json:"max_file_size"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &formats); err != nil || w.Code != 200 || formats.MaxSize != service.MaxInvestmentFileSize {
		t.Fatal("invalid format discovery", err)
	}
	byID := make(map[string]investment.ParserInfo)
	for _, format := range formats.Parsers {
		byID[format.ID] = format
	}
	for id, expected := range map[string]investment.ParserInfo{
		"zerodha_holdings_xlsx_v1":    {Provider: "Zerodha", Extensions: []string{".xlsx"}},
		"indmoney_us_holdings_xls_v1": {Provider: "INDmoney", Extensions: []string{".xls"}},
	} {
		format := byID[id]
		if format.Provider != expected.Provider || len(format.Extensions) != 1 || format.Extensions[0] != expected.Extensions[0] {
			t.Fatalf("missing format %s: %+v", id, format)
		}
	}
	w = upload("/api/investments/preview", investmentWorkbook(t, nil))
	if w.Code != 200 {
		t.Fatal("preview failed", w.Body.String())
	}
	list, err := database.ListInvestmentSnapshots()
	if err != nil || len(list) != 0 {
		t.Fatal("preview saved data", err)
	}
	w = upload("/api/investments/import", investmentWorkbook(t, nil))
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Snapshot models.InvestmentSnapshot `json:"snapshot"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	usData, err := os.ReadFile("../../samples/investments/indmoney-fictional.xls")
	if err != nil {
		t.Fatal(err)
	}
	w = upload("/api/investments/preview", usData, "renamed.xls")
	if w.Code != 200 {
		t.Fatal("US preview failed", w.Body.String())
	}
	var usSnapshot models.InvestmentSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &usSnapshot); err != nil || usSnapshot.Currency != "USD" || usSnapshot.InvestedValue != nil || usSnapshot.CurrentValue == nil || usSnapshot.Holdings[0].Quantity != 0.123456789 {
		t.Fatal("US preview fabricated valuation or lost precision", err)
	}
	list, err = database.ListInvestmentSnapshots()
	if err != nil || len(list) != 1 {
		t.Fatal("US preview persisted data", err)
	}
	w = upload("/api/investments/import", usData, "renamed.xls")
	if w.Code != 200 {
		t.Fatal("US import failed", w.Body.String())
	}
	w = upload("/api/investments/import", usData, "renamed-again.xls")
	var usResult struct {
		Snapshot  models.InvestmentSnapshot `json:"snapshot"`
		Duplicate bool                      `json:"duplicate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &usResult); err != nil || !usResult.Duplicate || usResult.Snapshot.Provider != "INDmoney" || usResult.Snapshot.UnrealizedReturn != nil {
		t.Fatal("US import not idempotent or changed valuation", err)
	}
	analytics, err := database.GetAnalyticsOverview()
	if err != nil || analytics.TotalTransactions != 0 || analytics.TotalIncome != 0 || analytics.TotalExpense != 0 {
		t.Fatal("US holdings changed bank ledger", err)
	}
	w = upload("/api/investments/import", []byte("invalid"))
	if w.Code != 400 {
		t.Fatal("bad upload should return 400")
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest(http.MethodDelete, "/api/investments/"+result.Snapshot.ID, nil))
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	list, err = database.ListInvestmentSnapshots()
	if err != nil || len(list) != 1 || list[0].Provider != "INDmoney" || list[0].InvestedValue != nil || list[0].CurrentValue == nil {
		t.Fatal("deletion changed another provider snapshot", err)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest(http.MethodDelete, "/api/investments/"+usResult.Snapshot.ID, nil))
	if w.Code != 204 {
		t.Fatal("US deletion failed", w.Code)
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest(http.MethodGet, "/api/investments", nil))
	if w.Code != 200 || w.Body.String() != "[]" {
		t.Fatal("deleted portfolio still listed", w.Body.String())
	}
}
