package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"local-finance/internal/db"
	"local-finance/internal/mcp"
	"local-finance/internal/service"
)

type testDesktopActions struct {
	saved  string
	opened string
}

func (d *testDesktopActions) SaveFile(name string, r io.Reader) error {
	data, err := io.ReadAll(r)
	d.saved = name + ":" + string(data)
	return err
}
func (d *testDesktopActions) OpenURL(url string) error { d.opened = url; return nil }

func TestDesktopActionsUseUIAuthorization(t *testing.T) {
	database, err := db.NewDB(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	desktop := &testDesktopActions{}
	router := SetupDesktopRouter(database, service.NewTransactionService(database), nil, desktop, mcp.NewManager(database))
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := localTestRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}
	if response := request("POST", "/api/desktop/save?name=ledger.csv", "a,b"); response.Code != 204 || desktop.saved != "ledger.csv:a,b" {
		t.Fatal(response.Body.String())
	}
	for _, path := range []string{"/api/desktop/save?name=../ledger", "/api/desktop/open-url"} {
		if response := request("POST", path, `{"url":"file:///etc/passwd"}`); response.Code != 400 {
			t.Fatalf("accepted unsafe OS action: %d", response.Code)
		}
	}
	if response := request("POST", "/api/auth/setup", `{"password":"supersecret123","auto_lock_minutes":30}`); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if response := request("GET", "/api/desktop", ""); response.Code != 200 {
		t.Fatal("host metadata should be available on login screen")
	}
	if response := request("POST", "/api/desktop/open-url", `{"url":"https://example.com/docs"}`); response.Code != 204 || desktop.opened != "https://example.com/docs" {
		t.Fatal("locked users cannot open documentation", response.Body.String())
	}
	for _, origin := range []string{"https://evil.example", "null"} {
		req := localTestRequest("POST", "/api/desktop/open-url", bytes.NewBufferString(`{"url":"https://example.com/blocked"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != 403 || desktop.opened != "https://example.com/docs" {
			t.Fatalf("untrusted origin opened a link: %s %d", origin, response.Code)
		}
	}
	if response := request("POST", "/api/desktop/open-url", `{"url":"file:///etc/passwd"}`); response.Code != 400 {
		t.Fatal("locked link opener accepted file URL")
	}
	for _, path := range []string{"/api/desktop/save?name=ledger.csv", "/api/system/update"} {
		if response := request(http.MethodPost, path, `{"url":"https://example.com"}`); response.Code != 401 {
			t.Fatalf("OS action bypassed UI lock: %s %d", path, response.Code)
		}
	}
}
