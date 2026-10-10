package db

import (
	"path/filepath"
	"testing"
)

func TestMCPWritePermissionUpgradeAndReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finance.db")
	database, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	settings := MCPSettings{Enabled: true, Port: 8081, TokenHash: "fictional-token-hash"}
	if err := database.SetMCPSettings(settings); err != nil {
		t.Fatal(err)
	}
	// Recreate the previous schema and migration history, preserving its access row.
	if _, err := database.Exec(`ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`DELETE FROM goose_db_version WHERE version_id = 16`); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	database, err = NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	got, err := database.GetMCPSettings()
	if err != nil || got != settings {
		t.Fatalf("upgrade changed existing access: %+v %v", got, err)
	}
	got.AllowCategorizationWrites = true
	if err := database.SetMCPSettings(got); err != nil {
		t.Fatal(err)
	}
	if err := database.ResetDatabase(); err != nil {
		t.Fatal(err)
	}
	got, err = database.GetMCPSettings()
	if err != nil || got.Enabled || got.TokenHash != "" || got.AllowCategorizationWrites {
		t.Fatalf("reset retained access: %+v %v", got, err)
	}
}
