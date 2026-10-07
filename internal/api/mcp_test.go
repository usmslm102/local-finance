package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPManagementProtection(t *testing.T) {
	server, database := setupTestRouter(t)
	send := func(method, path, body, token, origin, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
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
		server.ServeHTTP(w, req)
		return w
	}
	if w := send("GET", "/api/mcp/settings", "", "", "", ""); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"port":8081`)) {
		t.Fatalf("settings: %d %s", w.Code, w.Body.String())
	}
	for _, input := range []struct{ origin, host string }{{"http://evil.example", ""}, {"", "evil.example:8080"}, {"null", ""}} {
		if w := send("POST", "/api/mcp/token/rotate", "", "", input.origin, input.host); w.Code != 403 {
			t.Fatalf("foreign management allowed: %d", w.Code)
		}
	}
	w := send("POST", "/api/mcp/token/rotate", "", "", "", "")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("rotation: %d %s", w.Code, w.Body.String())
	}
	var credential struct{ Token string }
	json.Unmarshal(w.Body.Bytes(), &credential)
	if len(credential.Token) != 64 {
		t.Fatal("bad token")
	}
	w = send("GET", "/api/mcp/settings", "", "", "", "")
	if bytes.Contains(w.Body.Bytes(), []byte(credential.Token)) || bytes.Contains(w.Body.Bytes(), []byte("token_hash")) {
		t.Fatal("credential exposed in status")
	}
	if w := send("PUT", "/api/mcp/settings", `{"enabled":false,"port":1}`, "", "", ""); w.Code != 400 {
		t.Fatal("invalid port accepted")
	}
	if w := send("PUT", "/api/mcp/settings", `{"enabled":false,"port":8081}`, "", "http://localhost:5173", ""); w.Code != 200 {
		t.Fatalf("dev origin rejected: %d", w.Code)
	}
	hash, err := HashPassword("test-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SetSecurityPassword(hash, 60); err != nil {
		t.Fatal(err)
	}
	if w := send("GET", "/api/mcp/settings", "", credential.Token, "", ""); w.Code != 401 {
		t.Fatal("MCP credential authenticated management")
	}
	if w := send("GET", "/api/accounts", "", credential.Token, "", ""); w.Code != 401 {
		t.Fatal("MCP credential authenticated REST")
	}
	w = send("POST", "/api/auth/login", `{"password":"test-password"}`, "", "", "")
	var login struct{ Token string }
	json.Unmarshal(w.Body.Bytes(), &login)
	if w.Code != 200 || login.Token == "" {
		t.Fatalf("login: %s", w.Body.String())
	}
	if w := send("GET", "/api/mcp/settings", "", login.Token, "", ""); w.Code != http.StatusOK {
		t.Fatal("UI session rejected")
	}
}
