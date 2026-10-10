package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecuritySetupCannotReplacePassword(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	send := func(password string) *httptest.ResponseRecorder {
		req := localTestRequest("POST", "http://127.0.0.1:8080/api/auth/setup", bytes.NewBufferString(`{"password":"`+password+`"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := send("original-password"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := send("replacement-password"); w.Code != http.StatusConflict {
		t.Fatalf("setup replaced enabled credentials: %d", w.Code)
	}
	req := localTestRequest("POST", "http://127.0.0.1:8080/api/auth/login", bytes.NewBufferString(`{"password":"original-password"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("original credentials lost")
	}
}

func TestSecurityRejectsForeignHostAndOrigin(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	for _, tc := range []struct {
		host, origin, site string
		want               int
	}{
		{"attacker.example:8080", "http://attacker.example:8080", "same-origin", 403},
		{"127.0.0.1:8080", "http://attacker.example", "", 403},
		{"127.0.0.1:8080", "", "cross-site", 403},
		{"127.0.0.1:8080", "null", "", 403},
		{"127.0.0.1:8080", "http://127.0.0.1:8080", "same-origin", 200},
		{"localhost:8080", "http://localhost:8080", "same-origin", 200},
		{"127.0.0.1:8080", "http://localhost:5173", "same-site", 200},
		{"[::1]:8080", "http://[::1]:8080", "same-origin", 200},
	} {
		req := localTestRequest("GET", "http://127.0.0.1:8080/api/accounts", nil)
		req.Host = tc.host
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Sec-Fetch-Site", tc.site)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%+v: got %d", tc, w.Code)
		}
	}
}

func TestSecurityReadErrorsDenyExport(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	if _, err := database.Exec("DROP TABLE app_security"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest("GET", "http://127.0.0.1:8080/api/database/export/json", nil))
	if w.Code != 500 {
		t.Fatalf("security read error allowed export: %d", w.Code)
	}
}

func TestSecurityPasswordChangeRevokesOtherSessions(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	send := func(path, payload, token string) *httptest.ResponseRecorder {
		req := localTestRequest("POST", "http://127.0.0.1:8080"+path, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		return w
	}
	var first, second, replacement struct {
		Token string `json:"token"`
	}
	json.Unmarshal(send("/api/auth/setup", `{"password":"old-password"}`, "").Body.Bytes(), &first)
	json.Unmarshal(send("/api/auth/login", `{"password":"old-password"}`, "").Body.Bytes(), &second)
	json.Unmarshal(send("/api/auth/change-password", `{"current_password":"old-password","new_password":"new-password"}`, first.Token).Body.Bytes(), &replacement)
	for _, token := range []string{first.Token, second.Token} {
		req := localTestRequest("GET", "http://127.0.0.1:8080/api/database/export/json", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 401 {
			t.Errorf("old session accepted: %d", w.Code)
		}
	}
	if replacement.Token == "" {
		t.Fatal("password change did not replace current session")
	}
	req := localTestRequest("GET", "http://127.0.0.1:8080/api/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+replacement.Token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("replacement session rejected")
	}
}

func TestSecurityMissingSettingsRowDeniesExport(t *testing.T) {
	_, router, database, cleanup := setupTestRouter(t)
	defer cleanup()
	if _, err := database.Exec("DELETE FROM app_security"); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, localTestRequest("GET", "http://127.0.0.1:8080/api/database/export/json", nil))
	if w.Code != 500 {
		t.Fatalf("missing security row allowed access: %d", w.Code)
	}
}

func TestSecurityUploadLimits(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	for _, path := range []string{"/api/statements/upload", "/api/statements/preview", "/api/statements/upload-batch", "/api/statements/preview-batch", "/api/database/restore", "/api/investments/import"} {
		req := localTestRequest("POST", "http://127.0.0.1:8080"+path, bytes.NewBufferString("small"))
		req.ContentLength = 600 << 20
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != 413 {
			t.Errorf("%s did not reject oversized request: %d", path, w.Code)
		}
	}
}

func TestSecurityMultipartBatchCount(t *testing.T) {
	_, router, _, cleanup := setupTestRouter(t)
	defer cleanup()
	for _, count := range []int{20, 21} {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for i := 0; i < count; i++ {
			part, e := form.CreateFormFile("files", fmt.Sprintf("sample-%d.csv", i))
			if e != nil {
				t.Fatal(e)
			}
			part.Write([]byte("invalid sample"))
		}
		form.Close()
		req := localTestRequest("POST", "http://127.0.0.1:8080/api/statements/preview-batch", &body)
		req.Header.Set("Content-Type", form.FormDataContentType())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		want := 200
		if count > 20 {
			want = 413
		}
		if w.Code != want {
			t.Errorf("%d files: got %d want %d", count, w.Code, want)
		}
	}
}
