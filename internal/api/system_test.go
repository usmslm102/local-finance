package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"local-finance/internal/db"
	"local-finance/internal/service"
	"local-finance/internal/updater"
)

func TestSystemVersionEndpoint(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer database.Close()

	svc := service.NewTransactionService(database)
	router := SetupRouter(database, svc, nil)

	// Test GET /api/system/version
	req := httptest.NewRequest(http.MethodGet, "/api/system/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res["current_version"] != updater.CurrentVersion {
		t.Errorf("expected current_version %q, got %v", updater.CurrentVersion, res["current_version"])
	}
}

func TestHealthCheckVersion(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer database.Close()

	svc := service.NewTransactionService(database)
	router := SetupRouter(database, svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res["version"] != updater.CurrentVersion {
		t.Errorf("expected health version %q, got %v", updater.CurrentVersion, res["version"])
	}
}
