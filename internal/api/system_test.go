package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"local-finance/internal/db"
	"local-finance/internal/mcp"
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
	req := localTestRequest(http.MethodGet, "/api/system/version", nil)
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

	req := localTestRequest(http.MethodGet, "/api/health", nil)
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

func TestDesktopVersionSeparatesCheckFailuresFromInstallerGuidance(t *testing.T) {
	database, err := db.NewDB(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	router := SetupDesktopRouter(database, service.NewTransactionService(database), nil, &testDesktopActions{}, mcp.NewManager(database))
	request := func(path string, ctx context.Context) map[string]interface{} {
		req := localTestRequest(http.MethodGet, path, nil).WithContext(ctx)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d", response.Code)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	offline := request("/api/system/version?offline=true", context.Background())
	guidance, ok := offline["auto_update_error"].(string)
	if offline["can_auto_update"] != false || !ok || guidance == "" {
		t.Fatal("desktop installer guidance missing")
	}
	if offline["update_check_error"] != nil || offline["checked_at"] != "" {
		t.Fatal("offline desktop metadata reported a release-check failure")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Fail deterministically without contacting GitHub.
	failed := request("/api/system/version?refresh=true", ctx)
	if failed["update_check_error"] == nil || failed["update_check_error"] == "" || failed["checked_at"] != "" {
		t.Fatal("actual release-check failure not exposed separately")
	}
}
