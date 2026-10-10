package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/service"
)

func setupTestRouter(t *testing.T) (*ginEngineWrapper, *db.DB) {
	tempDB := "test_auth_" + t.Name() + ".db"
	_ = os.Remove(tempDB)
	_ = os.Remove(tempDB + "-wal")
	_ = os.Remove(tempDB + "-shm")

	database, err := db.NewDB(tempDB)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}

	svc := service.NewTransactionService(database)
	router := SetupRouter(database, svc, nil)

	t.Cleanup(func() {
		database.Close()
		_ = os.Remove(tempDB)
		_ = os.Remove(tempDB + "-wal")
		_ = os.Remove(tempDB + "-shm")
	})

	return &ginEngineWrapper{router: router}, database
}

type ginEngineWrapper struct {
	router http.Handler
}

func (g *ginEngineWrapper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	g.router.ServeHTTP(w, req)
}

func TestAuthWorkflow(t *testing.T) {
	server, _ := setupTestRouter(t)

	// 1. Initial status check: auth should be disabled by default
	req := localTestRequest("GET", "/api/auth/status", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var status models.AuthStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to parse auth status: %v", err)
	}
	if status.AuthEnabled {
		t.Errorf("expected AuthEnabled = false initially")
	}

	// 2. Access protected endpoint before setup: should succeed because auth is disabled
	req = localTestRequest("GET", "/api/accounts", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for public accounts when auth disabled, got %d", w.Code)
	}

	// 3. Setup password
	setupPayload, _ := json.Marshal(models.AuthSetupRequest{
		Password:        "supersecret123",
		AutoLockMinutes: 30,
	})
	req = localTestRequest("POST", "/api/auth/setup", bytes.NewReader(setupPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("setup failed with %d: %s", w.Code, w.Body.String())
	}

	var setupResp struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &setupResp)
	if setupResp.Token == "" {
		t.Fatalf("expected non-empty token in setup response")
	}

	// 4. Now protected endpoint without token should be 401 Unauthorized
	req = localTestRequest("GET", "/api/accounts", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
	}

	// 5. With Bearer token should be 200 OK
	req = localTestRequest("GET", "/api/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+setupResp.Token)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with Bearer token, got %d", w.Code)
	}

	// 6. Test Login with wrong password
	badLogin, _ := json.Marshal(models.AuthLoginRequest{
		Password: "wrongpassword",
	})
	req = localTestRequest("POST", "/api/auth/login", bytes.NewReader(badLogin))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad password, got %d", w.Code)
	}

	// 7. Test Login with correct password
	goodLogin, _ := json.Marshal(models.AuthLoginRequest{
		Password: "supersecret123",
	})
	req = localTestRequest("POST", "/api/auth/login", bytes.NewReader(goodLogin))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for good login, got %d", w.Code)
	}

	// 8. Test Disable auth
	disableReq, _ := json.Marshal(models.AuthDisableRequest{
		Password: "supersecret123",
	})
	req = localTestRequest("POST", "/api/auth/disable", bytes.NewReader(disableReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+setupResp.Token)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 disabling auth, got %d", w.Code)
	}

	// 9. Protected endpoint should now succeed without token
	req = localTestRequest("GET", "/api/accounts", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 after disabling auth, got %d", w.Code)
	}
}
