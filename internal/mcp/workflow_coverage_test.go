package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/models"
)

const groupCSV = "Date,Description,Category,Cost,Currency,Alex Example,Blair Example,Casey Example\n2026-01-10,Corner Shop,Food,125.50,INR,62.75,-62.75,0\n2026-01-11,Other paid,Food,80,INR,-40,40,0\n2026-01-12,Unmatched expense,Travel,500,INR,250,-250,0\n"

func TestMCPNativePreviewAndInlineInvestment(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	session := connect(t, m, token)
	before, _ := database.DataRevision()
	preview := decodeWrite[models.StatementPreviewResult](t, callWrite(t, session, "preview_statement", map[string]any{"filename": "fixture.csv", "csv_content": fictionalStatementCSV}, false))
	if preview.TotalTransactions != 1 || preview.TotalDebits != 125.5 || preview.NewTxsCount != 1 || preview.ParserID != "localfinance_csv_v1" {
		t.Fatalf("bank preview: %+v", preview)
	}
	input := map[string]any{"filename": "group.csv", "csv_content": groupCSV, "group": "Fixture group", "person": "Alex", "page_size": 1, "page": 2}
	decimalPage := map[string]any{"filename": "group.csv", "csv_content": groupCSV, "group": "Fixture group", "person": "Alex", "page_size": 1, "page": json.Number("2.0")}
	callWrite(t, session, "preview_splitwise_statement", decimalPage, true)
	sw := decodeWrite[struct {
		RowCount int                     `json:"row_count"`
		Members  []string                `json:"members"`
		Entries  []models.SplitwiseEntry `json:"entries"`
	}](t, callWrite(t, session, "preview_splitwise_statement", input, false))
	if sw.RowCount != 3 || len(sw.Members) != 3 || len(sw.Entries) != 1 || len(sw.Entries[0].Members) != 3 {
		t.Fatalf("Splitwise preview/paging: %+v", sw)
	}
	for _, file := range []string{"zerodha-fictional.xlsx", "indmoney-fictional.xls"} {
		data, err := os.ReadFile(filepath.Join("../../samples/investments", file))
		if err != nil {
			t.Fatal(err)
		}
		source := map[string]any{"filename": file, "content_base64": base64.StdEncoding.EncodeToString(data), "page_size": 1}
		p := decodeWrite[struct {
			investmentSummary
			Holdings []models.InvestmentHolding `json:"holdings"`
		}](t, callWrite(t, session, "preview_investment_statement", source, false))
		if p.ID != "" || p.HoldingCount < 1 || len(p.Holdings) != 1 || p.Holdings[0].Fields != nil {
			t.Fatalf("investment preview: %+v", p)
		}
		after, _ := database.DataRevision()
		if before != after {
			t.Fatal("preview wrote data")
		}
		yes := true
		m.ConfigureAccess(true, m.Status().Port, nil, &yes)
		write := connect(t, m, token)
		delete(source, "page_size")
		imported := decodeWrite[struct {
			Snapshot  investmentSummary `json:"snapshot"`
			Duplicate bool              `json:"duplicate"`
		}](t, callWrite(t, write, "import_investment_statement", source, false))
		absolute, _ := filepath.Abs(filepath.Join("../../samples/investments", file))
		repeated := decodeWrite[struct {
			Snapshot  investmentSummary `json:"snapshot"`
			Duplicate bool              `json:"duplicate"`
		}](t, callWrite(t, write, "import_investment_statement", map[string]any{"path": absolute}, false))
		if imported.Duplicate || !repeated.Duplicate || imported.Snapshot.ID != repeated.Snapshot.ID || imported.Snapshot.Currency != p.Currency || !reflect.DeepEqual(imported.Snapshot.CurrentValue, p.CurrentValue) {
			t.Fatalf("cross-source dedupe: %+v %+v", imported, repeated)
		}
		no := false
		m.ConfigureAccess(true, m.Status().Port, nil, &no)
		session = connect(t, m, token)
		before, _ = database.DataRevision()
	}
	// Read-only previews cannot use a host path, even if parsing would fail.
	for _, name := range []string{"preview_statement", "preview_investment_statement", "preview_splitwise_statement"} {
		args := map[string]any{"path": filepath.Join(t.TempDir(), "private.csv")}
		if name == "preview_investment_statement" {
			args["path"] = filepath.Join(t.TempDir(), "private.xlsx")
		}
		if name == "preview_splitwise_statement" {
			args["group"] = "Fixture"
			args["person"] = "Alex"
		}
		callWrite(t, session, name, args, true)
	}
	for _, args := range []map[string]any{
		{"filename": "bad.xlsx", "content_base64": "%%%"},
		{"filename": "../bad.xlsx", "content_base64": "YQ=="},
		{"path": "relative.xlsx", "filename": "bad.xlsx", "content_base64": "YQ=="},
		{},
	} {
		callWrite(t, session, "preview_investment_statement", args, true)
	}
	for _, args := range []map[string]any{{"filename": "bank.csv", "csv_content": fictionalStatementCSV, "parser_id": "wrong"}, {"filename": "bank.csv", "csv_content": fictionalStatementCSV, "parser_id": "hdfc_pdf"}} {
		callWrite(t, session, "preview_statement", args, true)
	}
	after, _ := database.DataRevision()
	if before != after {
		t.Fatal("invalid/denied preview changed data")
	}
	// A later provider still retains its complete extension and requirement attributes.
	result := callWrite(t, session, "list_investment_formats", map[string]any{"page": 2, "page_size": 1}, false)
	formats := decodeWrite[struct {
		Parsers []struct {
			Extensions   []string `json:"extensions"`
			Requirements []string `json:"requirements"`
		} `json:"parsers"`
	}](t, result)
	if len(formats.Parsers) != 1 || len(formats.Parsers[0].Extensions) == 0 || len(formats.Parsers[0].Requirements) == 0 {
		t.Fatalf("truncated format attributes: %+v", formats)
	}
}

func TestMCPSplitwisePermissionAndAccountingWorkflow(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	yes, no := true, false
	for policy := 0; policy < 8; policy++ {
		cat, statements, split := policy&1 != 0, policy&2 != 0, policy&4 != 0
		if _, err := m.ConfigureAccess(true, m.Status().Port, &cat, &statements, &split); err != nil {
			t.Fatal(err)
		}
		session := connect(t, m, token)
		tools, err := session.ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, tool := range tools.Tools {
			seen[tool.Name] = true
		}
		if seen["import_splitwise_statement"] != split || seen["confirm_splitwise_entry"] != split || seen["save_splitwise_member_aliases"] != split || seen["import_investment_statement"] != statements || seen["save_category"] != cat {
			t.Fatalf("policy %d tool isolation: %+v", policy, seen)
		}
		if !split {
			if _, err := connect(t, m, token).CallTool(context.Background(), &sdk.CallToolParams{Name: "import_splitwise_statement", Arguments: map[string]any{}}); err == nil {
				t.Fatalf("policy %d allowed Splitwise write", policy)
			}
		}
		info := decodeWrite[struct {
			Permissions map[string]bool `json:"permissions"`
		}](t, callWrite(t, session, "get_app_info", map[string]any{}, false))
		if info.Permissions["splitwise_writes"] != split {
			t.Fatal("capability differs from policy")
		}
	}
	m.ConfigureAccess(true, m.Status().Port, &no, &yes, &no)
	session := connect(t, m, token)
	callWrite(t, session, "import_statements", map[string]any{"statements": []any{map[string]any{"filename": "bank.csv", "csv_content": fictionalStatementCSV, "parser_id": "localfinance_csv_v1"}}}, false)
	m.ConfigureAccess(true, m.Status().Port, nil, &no, &yes)
	session = connect(t, m, token)
	source := map[string]any{"filename": "group.csv", "csv_content": groupCSV, "group": "Fixture group", "person": "Alex Example"}
	imported := decodeWrite[models.SplitwiseImportResult](t, callWrite(t, session, "import_splitwise_statement", source, false))
	if imported.Inserted != 3 || imported.Matched != 1 || imported.PaidByOthers != 1 || imported.Review != 1 {
		t.Fatalf("import counters: %+v", imported)
	}
	repeated := decodeWrite[models.SplitwiseImportResult](t, callWrite(t, session, "import_splitwise_statement", source, false))
	if repeated.Inserted != 0 || repeated.Duplicates != 3 {
		t.Fatalf("reimport: %+v", repeated)
	}
	entries := decodeWrite[[]models.SplitwiseEntry](t, callWrite(t, session, "list_splitwise_entries", map[string]any{"group": "Fixture group", "status": "REVIEW", "page_size": 1}, false))
	if len(entries) != 1 || entries[0].Description != "Unmatched expense" {
		t.Fatalf("filtered entries: %+v", entries)
	}
	pending := entries[0]
	// Schema accepts mathematical integers such as 25000.0. Typed decode failure
	// must reject before any mutation rather than silently substituting zero.
	callWrite(t, session, "confirm_splitwise_entry", map[string]any{"id": pending.ID, "share_cents": json.Number("25000.0"), "expected_status": "REVIEW", "ignore": true}, true)
	callWrite(t, session, "list_splitwise_candidates", map[string]any{"id": pending.ID, "share_cents": pending.Share, "include_near_matches": true}, false)
	callWrite(t, session, "list_splitwise_candidates", map[string]any{"id": pending.ID, "share_cents": 1.5}, true)
	callWrite(t, session, "confirm_splitwise_entry", map[string]any{"id": pending.ID, "share_cents": pending.Share, "expected_status": "CONFIRMED", "ignore": true}, true)
	callWrite(t, session, "confirm_splitwise_entry", map[string]any{"id": pending.ID, "share_cents": 0, "expected_status": "REVIEW", "ignore": true}, false)
	ignored := decodeWrite[[]models.SplitwiseEntry](t, callWrite(t, session, "list_splitwise_entries", map[string]any{"status": "IGNORED"}, false))
	if len(ignored) != 1 || ignored[0].Share != pending.Share {
		t.Fatalf("ignore rewrote source share: %+v", ignored)
	}
	callWrite(t, session, "reset_splitwise_entry", map[string]any{"id": pending.ID}, false)
	callWrite(t, session, "save_splitwise_member_aliases", map[string]any{"group": "Fixture group", "name": "Casey Example", "aliases": []string{"Casey Example", "casey@fixture"}, "pattern": ""}, false)
	members := decodeWrite[[]models.SplitwiseMember](t, callWrite(t, session, "list_splitwise_members", map[string]any{"group": "Fixture group", "page_size": 1, "page": 2}, false))
	if len(members) != 1 || len(members[0].Aliases) != 2 {
		t.Fatalf("aliases truncated: %+v", members)
	}
	callWrite(t, session, "save_splitwise_member_aliases", map[string]any{"group": "Fixture group", "name": "Casey Example", "aliases": []string{}, "pattern": "["}, true)
	callWrite(t, session, "confirm_splitwise_settlement", map[string]any{"transaction_id": "missing", "group": "Fixture group", "member": "Blair Example"}, true)
	gross := decodeWrite[struct {
		Items []models.Transaction `json:"items"`
		Total int                  `json:"total"`
	}](t, callWrite(t, session, "list_transactions", map[string]any{}, false))
	spend := decodeWrite[struct {
		Items []models.Transaction `json:"items"`
		Total int                  `json:"total"`
	}](t, callWrite(t, session, "list_transactions", map[string]any{"spending_only": true, "min_amount": 0, "max_amount": 70}, false))
	bankAmount := 0.0
	for _, item := range gross.Items {
		if item.AccountID != "" {
			bankAmount += item.Amount
		}
	}
	if gross.Total != 2 || bankAmount != 125.5 || spend.Total != 2 {
		t.Fatalf("bank/personal views: %+v %+v", gross, spend)
	}
	total := 0.0
	for _, item := range spend.Items {
		total += item.Amount
	}
	if total != 102.75 {
		t.Fatalf("personal share total %v", total)
	}
	callWrite(t, session, "list_transactions", map[string]any{"min_amount": 70, "max_amount": 0}, true)
	m.ConfigureAccess(true, m.Status().Port, nil, nil, &no)
	if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_splitwise_statement", Arguments: source}); err == nil {
		t.Fatal("old session retained revoked writes")
	}
	m.ConfigureAccess(true, m.Status().Port, nil, nil, &yes)
	m.Configure(true, m.Status().Port)
	if !m.Status().AllowSplitwiseWrites {
		t.Fatal("omitted field cleared preference")
	}
	if err := m.Revoke(); err != nil {
		t.Fatal(err)
	}
	saved, _ := database.GetMCPSettings()
	if saved.AllowSplitwiseWrites || saved.TokenHash != "" {
		t.Fatal("revoke retained permission")
	}
}

func TestMCPAdditionalReadParity(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	session := connect(t, m, token)
	for _, tool := range []string{"list_savings_goals", "get_data_health", "list_credit_card_bills"} {
		result := callWrite(t, session, tool, map[string]any{}, false)
		if result.StructuredContent == nil {
			t.Fatalf("%s missing result", tool)
		}
	}
	before, _ := database.DataRevision()
	callWrite(t, session, "list_credit_card_bills", map[string]any{"latest": true}, false)
	callWrite(t, session, "get_splitwise_import_schema", map[string]any{}, false)
	after, _ := database.DataRevision()
	if before != after {
		t.Fatal("read mutated ledger")
	}
	// Strict schemas reject mixed modes without revealing file contents or errors.
	invalid := callWrite(t, session, "preview_splitwise_statement", map[string]any{"group": "Fixture", "person": "Alex", "filename": "bad.csv", "csv_content": "bad", "content_base64": "YmFk"}, true)
	raw, _ := json.Marshal(invalid.Content)
	if strings.Contains(string(raw), "private") {
		t.Fatal("unexpected source echo")
	}
}

func TestMCPReviewCorrections(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	session := connect(t, m, token)
	source := strings.ReplaceAll(fictionalStatementCSV, "XXXX1234", "987654321234")
	preview := callWrite(t, session, "preview_statement", map[string]any{"filename": "fixture.csv", "csv_content": source}, false)
	raw, _ := json.Marshal(preview.StructuredContent)
	if strings.Contains(string(raw), "987654321234") {
		t.Fatal("preview leaked full identifier in mask")
	}
	empty := callWrite(t, session, "list_splitwise_entries", map[string]any{"group": "missing"}, false)
	encoded, _ := json.Marshal(empty.StructuredContent)
	var out toolOutput
	_ = json.Unmarshal(encoded, &out)
	if p, ok := out.Pagination[""]; !ok || p.Total != 0 || p.HasMore {
		t.Fatalf("empty collection metadata: %s", encoded)
	}
	yes := true
	m.ConfigureAccess(true, m.Status().Port, nil, &yes)
	session = connect(t, m, token)
	batch := decodeWrite[statementBatchResult](t, callWrite(t, session, "import_statements", map[string]any{"statements": []any{map[string]any{"filename": "bad.csv", "csv_content": fictionalStatementCSV, "parser_id": "missing"}, map[string]any{"filename": "valid.csv", "csv_content": fictionalStatementCSV}}}, false))
	if batch.Failed != 1 || batch.Succeeded != 1 || batch.Results[0].ErrorCode != "invalid_source" {
		t.Fatalf("parser error aborted independent imports: %+v", batch)
	}
	imports, _ := database.ListStatementImports()
	if len(imports) != 1 {
		t.Fatal("incorrect batch ownership")
	}
}
