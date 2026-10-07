package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
)

func testManager(t *testing.T) (*Manager, *db.DB) {
	t.Helper()
	database, err := db.NewDB(filepath.Join(t.TempDir(), "finance.db"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager(database)
	t.Cleanup(func() { m.Close(); database.Close() })
	return m, database
}
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
func enable(t *testing.T, m *Manager) string {
	t.Helper()
	token, err := m.RotateToken()
	if err != nil {
		t.Fatal(err)
	}
	status, err := m.Configure(true, freePort(t))
	if err != nil || !status.Listening {
		t.Fatalf("enable: %+v, %v", status, err)
	}
	return token
}
func request(m *Manager, token, origin, host string) int {
	req := httptest.NewRequest(http.MethodPost, m.Status().Endpoint, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if host != "" {
		req.Host = host
	}
	w := httptest.NewRecorder()
	m.ServeHTTP(w, req)
	return w.Code
}

func TestAccessLifecycle(t *testing.T) {
	m, database := testManager(t)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if m.Status().Enabled || m.Status().Listening || m.Status().HasToken {
		t.Fatal("fresh install must be disabled")
	}
	if _, err := m.Configure(true, 8081); err == nil {
		t.Fatal("enabled without credential")
	}
	token := enable(t, m)
	if request(m, "", "", "") != 401 || request(m, "bad", "", "") != 401 {
		t.Fatal("unauthenticated access")
	}
	if request(m, token, "", "") != 200 {
		t.Fatal("valid credential rejected")
	}
	if request(m, token, "http://attacker.example", "") != 403 || request(m, token, "", "attacker.example:8081") != 403 {
		t.Fatal("host/origin protection")
	}
	settings, _ := database.GetMCPSettings()
	if settings.TokenHash == token || strings.Contains(settings.TokenHash, token) {
		t.Fatal("stored raw secret")
	}
	encoded, _ := json.Marshal(settings)
	if bytes.Contains(encoded, []byte(settings.TokenHash)) {
		t.Fatal("serialized hash")
	}
	next, err := m.RotateToken()
	if err != nil {
		t.Fatal(err)
	}
	if request(m, token, "", "") != 401 || request(m, next, "", "") != 200 {
		t.Fatal("rotation did not revoke old token")
	}
	// UI protection and its sessions are independent of the MCP credential.
	if err := database.SetSecurityPassword("password-hash", 1); err != nil {
		t.Fatal(err)
	}
	if request(m, next, "", "") != 200 {
		t.Fatal("UI lock incorrectly blocked MCP")
	}
	port := m.Status().Port
	if _, err := m.Configure(false, port); err != nil {
		t.Fatal(err)
	}
	if request(m, next, "", "") != 503 {
		t.Fatal("disabled access accepted")
	}
	if _, err := m.Configure(true, port); err != nil {
		t.Fatal(err)
	}
	m.Close()
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if request(m, next, "", "") != 200 {
		t.Fatal("restart lost opted-in access")
	}
}

func TestPortConflict(t *testing.T) {
	m, _ := testManager(t)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	_, err = m.RotateToken()
	if err != nil {
		t.Fatal(err)
	}
	status, err := m.Configure(true, l.Addr().(*net.TCPAddr).Port)
	if err != nil || status.Listening || !status.Enabled || status.Error == "" {
		t.Fatalf("conflict not surfaced: %+v %v", status, err)
	}
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}
func connect(t *testing.T, m *Manager, token string) *sdk.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	client := sdk.NewClient(&sdk.Implementation{Name: "integration-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.StreamableClientTransport{Endpoint: m.Status().Endpoint, HTTPClient: &http.Client{Transport: bearerTransport{token}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestToolsReadOnlyAndParity(t *testing.T) {
	m, database := testManager(t)
	if _, err := database.Exec(`INSERT INTO accounts (id,bank_name,account_type,account_number_mask,account_number,customer_id,account_holder_name) VALUES ('a','Test Bank','SAVINGS','XXXX1234','987654321234','secret-customer','Secret Holder')`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := database.Exec(`INSERT INTO transactions (id,account_id,tx_hash,tx_date,raw_narration,cleaned_payee,tx_type,amount,category_id,notes,payment_mode,reference_number,tags) VALUES (?, 'a', ?, '2026-01-10','Groceries','Shop','DEBIT',100,'cat_groceries','manual note','UPI','','manual')`, fmt.Sprint(i), fmt.Sprint(i)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := database.ExportAllDataJSON()
	if err != nil {
		t.Fatal(err)
	}
	before.ExportedAt = time.Time{}
	snapshot, _ := json.Marshal(before)
	token := enable(t, m)
	session := connect(t, m, token)
	list, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 23 {
		t.Fatalf("got %d tools", len(list.Tools))
	}
	args := map[string]map[string]any{
		"get_transaction": {"id": "0"}, "get_merchant_profile": {"name": "Shop"}, "get_monthly_review": {"month": "2026-01"}, "list_monthly_review_evidence": {"month": "2026-01", "category": "cat_groceries"}, "get_wrapped": {"year": "2026"}, "get_cashflow": {"period": "2026-01"}, "get_budget_summary": {"month": "2026-01"},
	}
	for _, tool := range list.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			if !tool.Annotations.ReadOnlyHint {
				t.Fatal("missing read-only hint")
			}
			a := args[tool.Name]
			if a == nil {
				a = map[string]any{}
			}
			result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: tool.Name, Arguments: a})
			if err != nil || result.IsError {
				t.Fatalf("call failed: %v %+v", err, result)
			}
			raw, _ := json.Marshal(result.StructuredContent)
			for _, secret := range []string{"987654321234", "secret-customer", "Secret Holder"} {
				if bytes.Contains(raw, []byte(secret)) {
					t.Fatalf("leaked account metadata: %s", secret)
				}
			}
			if len(result.Content) == 0 {
				t.Fatal("missing text fallback")
			}
		})
	}
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "list_transactions", Arguments: map[string]any{"page": 2, "page_size": 2}})
	if err != nil || result.IsError {
		t.Fatalf("pagination: %v %+v", err, result)
	}
	raw, _ := json.Marshal(result.StructuredContent)
	var out struct {
		Data struct {
			Total int   `json:"total"`
			Items []any `json:"items"`
		}
		Pagination map[string]pagination
	}
	json.Unmarshal(raw, &out)
	if out.Data.Total != 3 || len(out.Data.Items) != 1 || out.Pagination["/items"].HasMore {
		t.Fatalf("bad pagination: %s", raw)
	}
	overview, _ := database.GetAnalyticsOverview()
	expected, _ := prepareOutput(overview, arguments{Page: 1, PageSize: 50}, false)
	result, _ = session.CallTool(context.Background(), &sdk.CallToolParams{Name: "get_overview", Arguments: map[string]any{}})
	actualJSON, _ := json.Marshal(result.StructuredContent)
	expectedJSON, _ := json.Marshal(expected)
	var canonical any
	_ = json.Unmarshal(expectedJSON, &canonical)
	expectedJSON, _ = json.Marshal(canonical)
	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Fatal("overview differs from app")
	}
	for _, args := range []map[string]any{{"page_size": 201}, {"page": 0}, {"unknown": true}, {"tx_type": "INVALID"}, {"start_date": "garbage"}, {"start_date": "2026-02-01", "end_date": "2026-01-01"}} {
		result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: "list_transactions", Arguments: args})
		if err != nil || !result.IsError {
			t.Fatalf("invalid arguments accepted: %+v %v", args, err)
		}
	}
	result, err = session.CallTool(context.Background(), &sdk.CallToolParams{Name: "get_transaction", Arguments: map[string]any{"id": "missing"}})
	if err != nil || !result.IsError {
		t.Fatal("missing ID not reported")
	}
	if _, err = session.CallTool(context.Background(), &sdk.CallToolParams{Name: "reset_database", Arguments: map[string]any{}}); err == nil {
		t.Fatal("write tool exposed")
	}
	after, _ := database.ExportAllDataJSON()
	after.ExportedAt = time.Time{}
	afterJSON, _ := json.Marshal(after)
	if !bytes.Equal(snapshot, afterJSON) {
		t.Fatal("finance tools modified data")
	}
}

func TestRestoreAndResetRevokeCredentials(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	backup := filepath.Join(t.TempDir(), "backup.db")
	if err := database.BackupTo(backup); err != nil {
		t.Fatal(err)
	}
	if err := m.Revoke(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := database.RestoreFrom(f); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	settings, _ := database.GetMCPSettings()
	if settings.Enabled || settings.TokenHash != "" || request(m, token, "", "") != 503 {
		t.Fatal("restore reactivated credential")
	}
	enable(t, m)
	if err := m.Revoke(); err != nil {
		t.Fatal(err)
	}
	if err := database.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	settings, _ = database.GetMCPSettings()
	if settings.Enabled || settings.TokenHash != "" {
		t.Fatal("reset retained credential")
	}
}

func TestNestedPaginationAndMasking(t *testing.T) {
	data := map[string]any{"accounts": []any{map[string]any{"account_number": "secret", "customer_id": "secret", "account_holder_name": "secret", "account_number_mask": "XXXX1234"}, map[string]any{"account_number": "secret"}, map[string]any{"account_number": "secret"}}}
	out, err := prepareOutput(data, arguments{Page: 2, PageSize: 2}, false)
	if err != nil {
		t.Fatal(err)
	}
	if out.Pagination["/accounts"].Total != 3 || out.Pagination["/accounts"].HasMore {
		t.Fatal("nested pagination incorrect")
	}
	raw, _ := json.Marshal(out)
	if bytes.Contains(raw, []byte("secret")) {
		t.Fatal("nested metadata leaked")
	}
}

func TestDisableDrainsInFlightReads(t *testing.T) {
	m, _ := testManager(t)
	entered := make(chan struct{})
	drained := make(chan struct{})
	m.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(drained) })
	token := enable(t, m)
	done := make(chan struct{})
	go func() {
		defer close(done)
		req, _ := http.NewRequest(http.MethodPost, m.Status().Endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response, _ := http.DefaultClient.Do(req)
		if response != nil {
			response.Body.Close()
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not start")
	}
	port := m.Status().Port
	if _, err := m.Configure(false, port); err != nil {
		t.Fatal(err)
	}
	select {
	case <-drained:
	default:
		t.Fatal("disable returned before reads drained")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not finish")
	}
}

func TestLegacyProtocolNegotiation(t *testing.T) {
	m, _ := testManager(t)
	token := enable(t, m)
	for _, version := range []string{"2025-03-26", "2025-06-18", "2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":%q,"capabilities":{},"clientInfo":{"name":"compatibility-test","version":"1"}}}`, version)
			req := httptest.NewRequest("POST", m.Status().Endpoint, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			w := httptest.NewRecorder()
			m.ServeHTTP(w, req)
			expectedVersion := version
			if version == "2026-07-28" {
				expectedVersion = "2025-11-25"
			} // initialize is the legacy handshake; modern SDK clients use server/discover.
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"protocolVersion":"`+expectedVersion+`"`) {
				t.Fatalf("negotiation failed: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
