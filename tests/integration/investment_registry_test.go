package integration_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"local-finance/internal/api"
	"local-finance/internal/investment"
	"local-finance/internal/models"
	"local-finance/internal/service"
)

// This adapter exercises the extension contract using a checked-in fictional
// workbook. It deliberately has no dependency on either built-in adapter.
type exampleInvestmentParser struct{ fixture []byte }

func (exampleInvestmentParser) ID() string { return "example_holdings_xlsx_v1" }
func (exampleInvestmentParser) Info() investment.ParserInfo {
	return investment.ParserInfo{Provider: "Example Broker", Name: "Example holdings", Extensions: []string{".xlsx"}}
}
func (p exampleInvestmentParser) CanParse(filename string, data []byte) bool {
	return bytes.Equal(data, p.fixture)
}
func (exampleInvestmentParser) Parse(data []byte) (*models.InvestmentSnapshot, error) {
	return &models.InvestmentSnapshot{
		Provider: "Example Broker", AccountRef: "FICTIONAL", AsOf: "2026-04-01", Currency: "INR",
		Holdings: []models.InvestmentHolding{{Symbol: "DEMO", Quantity: 2, Fields: map[string]string{"Provider detail": "retained"}}},
	}, nil
}

func TestInvestmentAPIUsesRegisteredProvider(t *testing.T) {
	data, err := os.ReadFile("../../samples/investments/zerodha-fictional.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	previous := investment.DefaultRegistry
	investment.DefaultRegistry = investment.NewRegistry()
	t.Cleanup(func() { investment.DefaultRegistry = previous })
	investment.DefaultRegistry.Register(exampleInvestmentParser{fixture: data})
	database := testDatabase(t)
	router := api.SetupRouter(database, service.NewTransactionService(database), nil)
	formats := httptest.NewRecorder()
	router.ServeHTTP(formats, localTestRequest(http.MethodGet, "/api/investments/formats", nil))
	var capabilities struct {
		Parsers []investment.ParserInfo `json:"parsers"`
	}
	if err := json.Unmarshal(formats.Body.Bytes(), &capabilities); err != nil {
		t.Fatal(err)
	}
	if formats.Code != http.StatusOK || len(capabilities.Parsers) != 1 || capabilities.Parsers[0].ID != "example_holdings_xlsx_v1" || capabilities.Parsers[0].Extensions[0] != ".xlsx" {
		t.Fatalf("registered provider missing from upload capabilities: %s", formats.Body.String())
	}
	upload := func(endpoint, filename string) []byte {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := localTestRequest(http.MethodPost, endpoint, &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s failed: %s", endpoint, response.Body.String())
		}
		return response.Body.Bytes()
	}
	var preview models.InvestmentSnapshot
	if err := json.Unmarshal(upload("/api/investments/preview", "example.xlsx"), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Provider != "Example Broker" || preview.ParserID != "example_holdings_xlsx_v1" {
		t.Fatalf("generic preview did not use registered adapter: %+v", preview)
	}
	if preview.Sheets == nil || preview.Warnings == nil {
		t.Fatal("optional adapter collections must be arrays for the shared UI")
	}
	before, err := database.ListInvestmentSnapshots()
	if err != nil || len(before) != 0 {
		t.Fatalf("preview persisted a snapshot: %+v, %v", before, err)
	}
	var imported struct {
		Snapshot  models.InvestmentSnapshot `json:"snapshot"`
		Duplicate bool                      `json:"duplicate"`
	}
	if err := json.Unmarshal(upload("/api/investments/import", "example.xlsx"), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Duplicate || imported.Snapshot.ID == "" || imported.Snapshot.Provider != preview.Provider || len(imported.Snapshot.Holdings) != 1 || imported.Snapshot.Holdings[0].Fields["Provider detail"] != "retained" {
		t.Fatalf("generic import lost adapter data: %+v", imported)
	}
	originalID := imported.Snapshot.ID
	if err := json.Unmarshal(upload("/api/investments/import", "renamed.xlsx"), &imported); err != nil {
		t.Fatal(err)
	}
	if !imported.Duplicate || imported.Snapshot.ID != originalID {
		t.Fatalf("registered provider bypassed deduplication: %+v", imported)
	}
	listed := httptest.NewRecorder()
	router.ServeHTTP(listed, localTestRequest(http.MethodGet, "/api/investments", nil))
	var saved []models.InvestmentSnapshot
	if err := json.Unmarshal(listed.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if listed.Code != http.StatusOK || len(saved) != 1 || saved[0].ID != originalID || saved[0].Provider != preview.Provider || len(saved[0].Holdings) != 1 || saved[0].Holdings[0].Fields["Provider detail"] != "retained" {
		t.Fatalf("generic listing lost registered provider data: %s", listed.Body.String())
	}
}
