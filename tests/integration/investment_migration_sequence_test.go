package integration_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"local-finance/internal/db"
)

func TestMigrationVersionsAreSequential(t *testing.T) {
	files, err := os.ReadDir("../../internal/db/migrations")
	if err != nil {
		t.Fatal(err)
	}
	expected := 1
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		version, err := strconv.Atoi(strings.SplitN(file.Name(), "_", 2)[0])
		if err != nil || version != expected {
			t.Fatalf("want migration %05d, found %s", expected, file.Name())
		}
		expected++
	}
}

func TestInvestmentMigrationsKeepOtherGapsStrict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-history.db")
	database, err := db.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	database.Close()
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM goose_db_version WHERE version_id IN (12,15)`); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	reopened, err := db.NewDB(path)
	if reopened != nil {
		reopened.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "12") {
		t.Fatalf("unrelated missing migration was permitted: %v", err)
	}
}

func TestInvestmentMigrationsUpgradeMainVersion14(t *testing.T) {
	path := filepath.Join(t.TempDir(), "main.db")
	database, err := db.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	database.Close()
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Exec(`DROP VIEW personal_transactions; DROP TABLE splitwise_entries; DROP TABLE investment_snapshots`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`DELETE FROM goose_db_version WHERE version_id>=15`); err != nil {
		t.Fatal(err)
	}
	upgraded, err := db.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	var applied int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM goose_db_version WHERE version_id=15 AND is_applied=1`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	snapshots, err := upgraded.ListInvestmentSnapshots()
	if err != nil || len(snapshots) != 0 || applied != 1 {
		t.Fatalf("main upgrade did not apply single migration 15: count=%d snapshots=%+v err=%v", applied, snapshots, err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.NewDB(path)
	if err != nil {
		t.Fatalf("repeat startup failed: %v", err)
	}
	defer reopened.Close()
	if err := conn.QueryRow(`SELECT COUNT(*) FROM goose_db_version WHERE version_id=15 AND is_applied=1`).Scan(&applied); err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Fatalf("repeat startup duplicated investment migration: count=%d", applied)
	}
}
