package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/models"
)

func TestRevokeDrainsSDKWriteBeforeRestore(t *testing.T) {
	m, database := testManager(t)
	token := enable(t, m)
	allowed := true
	if _, err := m.ConfigureAccess(true, m.Status().Port, &allowed, &allowed); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "before-write.db")
	if err := database.BackupTo(backup); err != nil {
		t.Fatal(err)
	}

	entered, stopping, release, saved := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	// Pause after shared schema/cancellation validation, modeling a write that
	// cannot finish until its database operation completes. The SDK and DB are real.
	server := sdk.NewServer(&sdk.Implementation{Name: "write-lifecycle-test", Version: "1"}, nil)
	registerWriteTool(server, "save_category", "Save a fictional category", map[string]any{"name": map[string]any{"type": "string"}}, []string{"name"}, func(raw json.RawMessage) (any, error) {
		close(entered)
		<-release
		category := models.Category{ID: "inflight-category", Name: "Fictional in-flight category"}
		err := database.CreateCategory(&category)
		close(saved)
		return category, err
	})
	// Observe the existing lifecycle cancellation without replacing the SDK's
	// handler. SDK tool contexts may outlive their HTTP request while Close drains.
	m.mu.Lock()
	cancelListener := m.cancel
	m.cancel = func() { cancelListener(); close(stopping) }
	m.mu.Unlock()
	m.handler = sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	session := connect(t, m, token)
	callDone := make(chan struct{})
	go func() {
		defer close(callDone)
		_, _ = session.CallTool(context.Background(), &sdk.CallToolParams{Name: "save_category", Arguments: map[string]any{"name": "Fictional"}})
	}()
	// Release the callback on every exit, before Manager cleanup can wait for it.
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("SDK write did not enter")
	}
	revokeDone := make(chan error, 1)
	go func() { revokeDone <- m.Revoke() }()
	select {
	case <-stopping:
	case <-time.After(5 * time.Second):
		t.Fatal("revocation did not begin stopping the listener")
	}
	select {
	case err := <-revokeDone:
		t.Fatalf("revocation returned while write was pending: %v", err)
	default:
	}
	close(release)
	select {
	case err := <-revokeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("revocation did not finish after write")
	}
	select {
	case <-saved:
	default:
		t.Fatal("revocation returned before database write finished")
	}
	select {
	case <-callDone:
	case <-time.After(5 * time.Second):
		t.Fatal("SDK call did not finish")
	}
	beforeRestore, err := database.ListCategories()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, category := range beforeRestore {
		if category.ID == "inflight-category" {
			found = true
		}
	}
	if !found {
		t.Fatal("paused write did not persist before restore")
	}

	file, err := os.Open(backup)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := database.RestoreFrom(file); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	settings, err := database.GetMCPSettings()
	if err != nil || settings.Enabled || settings.AllowCategorizationWrites || settings.AllowStatementWrites || settings.TokenHash != "" {
		t.Fatalf("restore retained access: %+v %v", settings, err)
	}
	categories, err := database.ListCategories()
	if err != nil {
		t.Fatal(err)
	}
	for _, category := range categories {
		if category.ID == "inflight-category" {
			t.Fatal("in-flight write reached restored database")
		}
	}
	if request(m, token, "", "") != http.StatusServiceUnavailable {
		t.Fatal("old credential remained usable")
	}
}
