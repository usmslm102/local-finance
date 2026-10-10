package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/investment"
	"local-finance/internal/models"
	"local-finance/internal/service"
)

func TestInvestmentMCPViewsUploadsAndPermission(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	read := connect(t, m, token)
	formats := decodeWrite[struct {
		Parsers []investment.ParserInfo `json:"parsers"`
		Max     int64                   `json:"max_file_size"`
	}](t, callWrite(t, read, "list_investment_formats", map[string]any{}, false))
	if formats.Max != service.MaxInvestmentFileSize || len(formats.Parsers) != 2 {
		t.Fatalf("formats: %+v", formats)
	}
	for _, format := range formats.Parsers {
		if len(format.Requirements) == 0 {
			t.Fatalf("missing requirements: %+v", format)
		}
	}
	if _, err := read.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_investment_statement", Arguments: map[string]any{"path": "relative.xlsx"}}); err == nil {
		t.Fatal("read access exposed investment upload")
	}
	callWrite(t, connect(t, m, token), "get_investment_snapshot", map[string]any{"id": "missing"}, true)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	for _, name := range []string{"zerodha-fictional.xlsx", "indmoney-fictional.xls"} {
		t.Run(name, func(t *testing.T) {
			path, err := filepath.Abs(filepath.Join("../../samples/investments", name))
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := service.NewInvestmentService(database).Preview(name, strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			first := decodeWrite[struct {
				Snapshot  investmentSummary `json:"snapshot"`
				Duplicate bool              `json:"duplicate"`
			}](t, callWrite(t, session, "import_investment_statement", map[string]any{"path": path}, false))
			if first.Duplicate || first.Snapshot.ID == "" || first.Snapshot.Currency != expected.Currency || first.Snapshot.HoldingCount != len(expected.Holdings) || !reflect.DeepEqual(first.Snapshot.CurrentValue, expected.CurrentValue) || !reflect.DeepEqual(first.Snapshot.InvestedValue, expected.InvestedValue) {
				t.Fatalf("import: %+v expected %+v", first, expected)
			}
			second := decodeWrite[struct {
				Snapshot  investmentSummary `json:"snapshot"`
				Duplicate bool              `json:"duplicate"`
			}](t, callWrite(t, session, "import_investment_statement", map[string]any{"path": path}, false))
			if !second.Duplicate || second.Snapshot.ID != first.Snapshot.ID {
				t.Fatalf("reimport: %+v", second)
			}
			result := callWrite(t, session, "get_investment_snapshot", map[string]any{"id": first.Snapshot.ID, "page_size": 1}, false)
			var detail struct {
				Data struct {
					investmentSummary
					Holdings []models.InvestmentHolding `json:"holdings"`
				}
				Pagination map[string]pagination `json:"pagination"`
			}
			raw, _ := json.Marshal(result.StructuredContent)
			if err := json.Unmarshal(raw, &detail); err != nil {
				t.Fatal(err)
			}
			if len(detail.Data.Holdings) != 1 || detail.Pagination["/holdings"].Total != len(expected.Holdings) || detail.Data.HoldingCount != len(expected.Holdings) {
				t.Fatalf("holdings pagination: %+v", detail)
			}
			want := expected.Holdings[0]
			want.Fields = nil
			if !reflect.DeepEqual(detail.Data.Holdings[0], want) {
				t.Fatalf("holdings parity: %+v want %+v", detail.Data.Holdings[0], want)
			}
			if strings.Contains(string(raw), expected.AccountRef) || strings.Contains(string(raw), `"sheets"`) || strings.Contains(string(raw), `"account_ref"`) {
				t.Fatalf("raw source/account metadata leaked: %s", raw)
			}
			if len(expected.Holdings) > 1 {
				next := decodeWrite[struct {
					Holdings []models.InvestmentHolding `json:"holdings"`
				}](t, callWrite(t, session, "get_investment_snapshot", map[string]any{"id": first.Snapshot.ID, "page": 2, "page_size": 1}, false))
				if len(next.Holdings) != 1 || next.Holdings[0].Symbol != expected.Holdings[1].Symbol {
					t.Fatalf("next holdings page: %+v", next)
				}
			}
		})
	}
	summaries := decodeWrite[[]investmentSummary](t, callWrite(t, session, "list_investments", map[string]any{}, false))
	if len(summaries) != 2 || summaries[0].PortfolioKey == summaries[1].PortfolioKey {
		t.Fatalf("summaries: %+v", summaries)
	}
	bankData, err := database.ExportAllDataJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(bankData.Transactions) != 0 || len(bankData.Accounts) != 0 || len(bankData.StatementImports) != 0 {
		t.Fatal("investment import changed bank ledger")
	}
	allowed = false
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_investment_statement", Arguments: map[string]any{"path": "relative.xlsx"}}); err == nil {
		t.Fatal("old session retained investment writes")
	}
	callWrite(t, connect(t, m, token), "list_investments", map[string]any{}, false)
}

func TestInvestmentMCPRejectsInvalidFiles(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	session := connect(t, m, token)
	root := t.TempDir()
	bad := filepath.Join(root, "broken.xlsx")
	if err := os.WriteFile(bad, []byte("invalid workbook"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{bad, "relative.xlsx", filepath.Join(root, "missing.xls"), filepath.Join(root, "investment.csv"), `\/server/share/investment.xlsx`, root} {
		callWrite(t, session, "import_investment_statement", map[string]any{"path": path}, true)
	}
	file, err := os.Create(bad)
	if err != nil {
		t.Fatal(err)
	}
	if err = file.Truncate(service.MaxInvestmentFileSize + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	callWrite(t, session, "import_investment_statement", map[string]any{"path": bad}, true)
	snapshots, err := database.ListInvestmentSnapshots()
	if err != nil || len(snapshots) != 0 {
		t.Fatalf("invalid file persisted snapshot: %+v %v", snapshots, err)
	}
}

func TestInvestmentPortfolioKeysDisambiguateMaskedReferences(t *testing.T) {
	a := models.InvestmentSnapshot{Provider: "Fictional", AccountRef: "AAA1234", Currency: "INR"}
	b := a
	b.AccountRef = "BBB1234"
	first, second := summarizeInvestment(a), summarizeInvestment(b)
	if first.AccountRefMask != second.AccountRefMask || first.PortfolioKey == second.PortfolioKey {
		t.Fatal("masked reference collision merged portfolios")
	}
	a.AsOf = "2026-01-10"
	if summarizeInvestment(a).PortfolioKey != first.PortfolioKey {
		t.Fatal("snapshot date changed portfolio identity")
	}
}
