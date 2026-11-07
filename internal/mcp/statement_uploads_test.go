package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/jung-kurt/gofpdf"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"local-finance/internal/db"
	"local-finance/internal/parser"
)

func enableStatementUploads(t *testing.T, m *Manager, token string) *sdk.ClientSession {
	t.Helper()
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	return connect(t, m, token)
}

func importStatements(t *testing.T, session *sdk.ClientSession, items ...map[string]any) statementBatchResult {
	t.Helper()
	return decodeWrite[statementBatchResult](t, callWrite(t, session, "import_statements", map[string]any{"statements": items}, false))
}

func TestStatementSchemaDiscoveryWithoutWrites(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	session := connect(t, m, token)
	before, _ := database.ListStatementImports()
	schema := decodeWrite[map[string]any](t, callWrite(t, session, "get_statement_import_schema", map[string]any{}, false))
	if schema["format"] != "LOCALFINANCE_CSV_V1" || schema["currency"] != "INR" || schema["uploads"].(map[string]any)["uploads_enabled"] != false {
		t.Fatalf("schema discovery: %+v", schema)
	}
	// The complete schema must survive redaction and must not be paginated.
	row := schema["row_schema"].(map[string]any)
	properties := row["properties"].(map[string]any)
	if len(properties) != 12 || properties["account_number"] == nil || len(row["required"].([]any)) != 7 {
		t.Fatalf("incomplete CSV table schema: %+v", row)
	}
	encoded, _ := json.Marshal(row)
	var jsonSchema jsonschema.Schema
	if err := json.Unmarshal(encoded, &jsonSchema); err != nil {
		t.Fatal(err)
	}
	resolved, err := jsonSchema.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]any{"bank_name": "Fictional Bank", "account_type": "SAVINGS", "account_number_mask": "XXXX1234", "date": "2026-01-10", "narration": "Corner Shop", "amount": "125.50", "tx_type": "DEBIT", "running_balance": "-25.50", "value_date": ""}
	if err := resolved.Validate(valid); err != nil {
		t.Fatalf("valid row rejected by discoverable schema: %v", err)
	}
	for _, amount := range []string{"0", "0.00", "-1.00", "1e2", "1,000.00", "1.001"} {
		valid["amount"] = amount
		if err := resolved.Validate(valid); err == nil {
			t.Fatalf("schema accepted invalid amount %q", amount)
		}
	}
	p, _, _ := parser.DefaultRegistry.Detect("example.csv", []byte(schema["example_csv"].(string)))
	if p == nil || p.ID() != parser.LocalFinanceCSVParserID {
		t.Fatal("discovered CSV example cannot be detected")
	}
	if _, _, err := p.Parse(strings.NewReader(schema["example_csv"].(string)), parser.ParseOptions{}); err != nil {
		t.Fatalf("discovered CSV example cannot be imported: %v", err)
	}
	callWrite(t, session, "get_statement_import_schema", map[string]any{"page": 2}, true)
	if _, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_statements", Arguments: map[string]any{"statements": []any{map[string]any{"filename": "statement.csv", "csv_content": fictionalStatementCSV}}}}); err == nil {
		t.Fatal("discovery granted upload access")
	}
	after, _ := database.ListStatementImports()
	if len(before) != len(after) {
		t.Fatal("schema discovery mutated import history")
	}
	writeSession := enableStatementUploads(t, m, token)
	schema = decodeWrite[map[string]any](t, callWrite(t, writeSession, "get_statement_import_schema", map[string]any{}, false))
	if schema["uploads"].(map[string]any)["uploads_enabled"] != true {
		t.Fatal("discovery did not reflect granted upload access")
	}
	allowed := false
	if _, err := m.ConfigureAccess(true, m.Status().Port, nil, &allowed); err != nil {
		t.Fatal(err)
	}
	if _, err := writeSession.CallTool(context.Background(), &sdk.CallToolParams{Name: "import_statements", Arguments: map[string]any{"statements": []any{map[string]any{"filename": "statement.csv", "csv_content": fictionalStatementCSV}}}}); err == nil {
		t.Fatal("cached session retained upload permission")
	}
}

func TestNativeStatementBatchAndCSVFallback(t *testing.T) {
	m, database := testManager(t)
	session := enableStatementUploads(t, m, enable(t, m))
	nativePDF, err := os.ReadFile("../../samples/savings/ICICI_Savings_Synthetic.pdf")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "original.pdf")
	if err := os.WriteFile(path, nativePDF, 0600); err != nil {
		t.Fatal(err)
	}
	otherPDF, err := os.ReadFile("../../samples/savings/Union_Bank_Savings_Synthetic.pdf")
	if err != nil {
		t.Fatal(err)
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, "Unlisted Bank statement: 2026-01-10 Corner Shop debit INR 125.50 account XXXX1234")
	var unsupported bytes.Buffer
	if err := pdf.Output(&unsupported); err != nil {
		t.Fatal(err)
	}
	out := importStatements(t, session,
		map[string]any{"path": path},
		map[string]any{"filename": "another.pdf", "content_base64": base64.StdEncoding.EncodeToString(otherPDF)},
		map[string]any{"filename": "unsupported.pdf", "content_base64": base64.StdEncoding.EncodeToString(unsupported.Bytes())},
		map[string]any{"filename": "bad.csv", "csv_content": strings.ReplaceAll(fictionalStatementCSV, "125.50", "NaN")},
		map[string]any{"filename": "verified.csv", "csv_content": fictionalStatementCSV},
	)
	if out.Succeeded != 3 || out.Failed != 2 || out.NotAttempted != 0 || len(out.Results) != 5 {
		t.Fatalf("batch: %+v", out)
	}
	for i, item := range out.Results {
		if item.Index != i {
			t.Fatalf("lost input ordering: %+v", out.Results)
		}
	}
	for _, i := range []int{0, 1, 4} {
		if out.Results[i].Result == nil || out.Results[i].Result.StatementImportID == "" || out.Results[i].Result.InsertedCount == 0 {
			t.Fatalf("missing native/CSV import result: %+v", out.Results[i])
		}
	}
	for _, i := range []int{2, 3} {
		if out.Results[i].ErrorCode != "parse_failed" || out.Results[i].FallbackTool != "get_statement_import_schema" {
			t.Fatalf("missing conversion fallback: %+v", out.Results[i])
		}
	}
	if out.Results[0].Result.AccountID == out.Results[1].Result.AccountID || out.Results[0].Result.AccountID == out.Results[4].Result.AccountID {
		t.Fatal("batch merged distinct bank accounts")
	}
	_, savedPDF, err := database.GetStatementPDF(out.Results[0].Result.StatementImportID)
	if err != nil || !bytes.Equal(savedPDF, nativePDF) {
		t.Fatalf("native upload did not retain original PDF: %v", err)
	}
	// Only the failed source is converted and reuploaded; successful files remain.
	fallbackCSV := strings.ReplaceAll(fictionalStatementCSV, "Fictional Bank", "Unlisted Bank")
	fallback := importStatements(t, session, map[string]any{"filename": "converted.csv", "csv_content": fallbackCSV})
	if fallback.Succeeded != 1 || fallback.Results[0].Result.ParserUsed != "LocalFinance transaction CSV v1" {
		t.Fatalf("fallback: %+v", fallback)
	}
	imports, _ := database.ListStatementImports()
	if len(imports) != 4 {
		t.Fatalf("failed files wrote import records: %d", len(imports))
	}
	rows, _, err := database.ListTransactions(db.TransactionFilter{AccountID: fallback.Results[0].Result.AccountID, Limit: 10})
	if err != nil || len(rows) != 1 {
		t.Fatalf("fallback ledger: %+v %v", rows, err)
	}
	if _, err := database.Exec(`UPDATE transactions SET category_id='cat_groceries', is_manual_category=1, tags='manual', notes='keep' WHERE id=?`, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	reimport := importStatements(t, session, map[string]any{"filename": "converted.csv", "csv_content": fallbackCSV, "account_id": fallback.Results[0].Result.AccountID})
	if reimport.Succeeded != 1 || reimport.Results[0].Result.InsertedCount != 0 || reimport.Results[0].Result.DuplicateCount != 1 {
		t.Fatalf("reimport: %+v", reimport)
	}
	saved, err := database.GetTransaction(rows[0].ID)
	if err != nil || !saved.IsManualCategory || saved.Tags != "manual" || saved.Notes != "keep" || saved.CategoryID == nil || *saved.CategoryID != "cat_groceries" {
		t.Fatalf("batch reimport lost edits: %+v %v", saved, err)
	}
}

func TestEncryptedPDFUploadsRequirePasswordAndRetainSource(t *testing.T) {
	m, database := testManager(t)
	session := enableStatementUploads(t, m, enable(t, m))
	pdf, err := os.ReadFile("../../samples/savings/ICICI_Savings_Synthetic.pdf")
	if err != nil {
		t.Fatal(err)
	}
	password := "Fictional-Upload-Pass-123"
	var encrypted bytes.Buffer
	if err := api.Encrypt(bytes.NewReader(pdf), &encrypted, model.NewAESConfiguration(password, password, 256)); err != nil {
		t.Fatal(err)
	}
	for _, supplied := range []string{"", "incorrect", password} {
		item := map[string]any{"filename": "protected.pdf", "content_base64": base64.StdEncoding.EncodeToString(encrypted.Bytes())}
		if supplied != "" {
			item["password"] = supplied
		}
		out := importStatements(t, session, item)
		if supplied != password {
			if out.Failed != 1 || out.Results[0].ErrorCode != "password_required" || out.Results[0].FallbackTool != "" {
				t.Fatalf("password failure requested conversion: %+v", out)
			}
		} else {
			if out.Succeeded != 1 {
				t.Fatalf("password import: %+v", out)
			}
			_, retained, err := database.GetStatementPDF(out.Results[0].Result.StatementImportID)
			if err != nil || !bytes.Equal(retained, encrypted.Bytes()) {
				t.Fatalf("encrypted source altered: %v", err)
			}
		}
		encoded, _ := json.Marshal(out)
		if bytes.Contains(encoded, []byte(password)) || bytes.Contains(encoded, []byte(item["content_base64"].(string))) {
			t.Fatal("upload response echoed secret or source bytes")
		}
	}
	imports, _ := database.ListStatementImports()
	if len(imports) != 1 {
		t.Fatal("password failures saved import records")
	}
}

func TestStatementUploadValidationAndIndependentFailures(t *testing.T) {
	m, database := testManager(t)
	session := enableStatementUploads(t, m, enable(t, m))
	valid := map[string]any{"filename": "verified.csv", "csv_content": fictionalStatementCSV}
	for _, items := range []any{
		[]any{},
		[]any{valid, map[string]any{"filename": "missing.csv"}},
		[]any{map[string]any{"path": "/private/tmp/source.pdf", "filename": "source.csv", "csv_content": fictionalStatementCSV}},
		[]any{map[string]any{"filename": "source.csv", "csv_content": fictionalStatementCSV, "content_base64": "YWJj"}},
		[]any{map[string]any{"filename": "source.csv", "csv_content": fictionalStatementCSV, "unknown": true}},
		[]any{map[string]any{"filename": "source.csv", "csv_content": fictionalStatementCSV, "password": nil}},
		make([]any, maxStatementBatchFiles+1),
	} {
		callWrite(t, session, "import_statements", map[string]any{"statements": items}, true)
	}
	imports, _ := database.ListStatementImports()
	if len(imports) != 0 {
		t.Fatal("structurally invalid batch partially imported")
	}
	out := importStatements(t, session,
		map[string]any{"filename": "../bad.csv", "csv_content": fictionalStatementCSV},
		map[string]any{"filename": "bad.pdf", "csv_content": fictionalStatementCSV},
		map[string]any{"filename": "bad.pdf", "content_base64": "invalid base64"},
		map[string]any{"filename": "bad.exe", "content_base64": "YWJj"},
		map[string]any{"path": "relative.pdf"},
		map[string]any{"filename": "verified.csv", "csv_content": fictionalStatementCSV, "account_id": "missing"},
		valid,
	)
	if out.Succeeded != 1 || out.Failed != 6 || out.Results[6].Result == nil {
		t.Fatalf("individual validation failures: %+v", out)
	}
	for _, item := range out.Results[:5] {
		if item.ErrorCode != "invalid_source" {
			t.Fatalf("unexpected source error: %+v", item)
		}
	}
	if out.Results[5].ErrorCode != "import_failed" {
		t.Fatalf("wrong account failure: %+v", out.Results[5])
	}
	imports, _ = database.ListStatementImports()
	if len(imports) != 1 {
		t.Fatal("source validation or invalid account saved an import")
	}
}

func TestInlineUploadOverOneMiBAndBatchByteLimit(t *testing.T) {
	m, database := testManager(t)
	session := enableStatementUploads(t, m, enable(t, m))
	large := strings.Replace(fictionalStatementCSV, "Corner Shop", strings.Repeat("Fictional ", 120000), 1)
	out := importStatements(t, session, map[string]any{"filename": "large.csv", "csv_content": large})
	if out.Succeeded != 1 {
		t.Fatalf("inline upload larger than old transport limit failed: %+v", out)
	}
	imports, _ := database.ListStatementImports()
	tooLarge := strings.Repeat(" ", 11<<20)
	callWrite(t, session, "import_statements", map[string]any{"statements": []any{
		map[string]any{"filename": "first.csv", "csv_content": tooLarge},
		map[string]any{"filename": "second.csv", "csv_content": tooLarge},
	}}, true)
	after, _ := database.ListStatementImports()
	if len(imports) != len(after) {
		t.Fatal("oversized inline batch changed the ledger")
	}
}
