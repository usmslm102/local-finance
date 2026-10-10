package db

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPSettingsMigrationWithoutHistory(t *testing.T) {
	for _, hasSettings := range []bool{false, true} {
		name := "empty table"
		if hasSettings {
			name = "existing access settings"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "finance.db")
			database, err := NewDB(path)
			if err != nil {
				t.Fatal(err)
			}
			settings := MCPSettings{Enabled: true, Port: 18081, TokenHash: "fictional-token-hash"}
			if hasSettings {
				if err := database.SetMCPSettings(settings); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := database.Exec(`DELETE FROM mcp_settings`); err != nil {
					t.Fatal(err)
				}
				settings = MCPSettings{Port: 8081}
			}
			// Mimic the observed legacy database: migration history stops at 13
			// even though the original MCP table is already present.
			if _, err := database.Exec(`
                ALTER TABLE mcp_settings DROP COLUMN allow_statement_writes;
                ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes;
                DROP TABLE investment_snapshots;
                DELETE FROM goose_db_version WHERE version_id >= 14;
                INSERT INTO accounts (id, bank_name, account_type) VALUES ('fixture-account', 'Fixture bank', 'savings');
                INSERT INTO transactions (id, account_id, tx_hash, tx_date, raw_narration, tx_type, amount, category_id, notes, tags, is_manual_category)
                VALUES ('fixture-tx', 'fixture-account', 'fixture-stable-hash', '2026-01-01', 'Fixture narration', 'debit', 100, 'cat_food', 'Keep my note', 'manual-tag', 1);
            `); err != nil {
				t.Fatal(err)
			}
			if err := database.Close(); err != nil {
				t.Fatal(err)
			}
			// Startup repairs missing migration history, then stays idempotent.
			for i := 0; i < 2; i++ {
				database, err = NewDB(path)
				if err != nil {
					t.Fatalf("startup %d failed: %v", i, err)
				}
				got, err := database.GetMCPSettings()
				if err != nil {
					t.Fatal(err)
				}
				if got != settings {
					t.Fatal("migration changed existing MCP access settings")
				}
				var hash, category, notes, tags string
				var manual bool
				if err := database.conn.QueryRow(`SELECT tx_hash, category_id, notes, tags, is_manual_category FROM transactions WHERE id = 'fixture-tx'`).Scan(&hash, &category, &notes, &tags, &manual); err != nil {
					t.Fatal(err)
				}
				if hash != "fixture-stable-hash" || category != "cat_food" || notes != "Keep my note" || tags != "manual-tag" || !manual {
					t.Fatal("migration changed transaction identity or user edits")
				}
				var count int
				if err := database.conn.QueryRow(`SELECT COUNT(*) FROM goose_db_version WHERE version_id BETWEEN 14 AND 17 AND is_applied = 1`).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 4 {
					t.Fatalf("expected migrations 14–17 to be recorded, got %d", count)
				}
				if err := database.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestMCPMigrationRecoveryFailsWithoutPartialChanges(t *testing.T) {
	for _, fixture := range []struct {
		name     string
		mutation string
		wantErr  string
	}{
		{"unknown schema", `ALTER TABLE mcp_settings ADD COLUMN unexpected TEXT`, "schema differs"},
		{"history write fails", `CREATE TRIGGER reject_history BEFORE INSERT ON goose_db_version WHEN NEW.version_id = 14 BEGIN SELECT RAISE(ABORT, 'fixture history failure'); END`, "fixture history failure"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			database, err := NewDB(filepath.Join(t.TempDir(), "finance.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			if _, err := database.Exec(`
				ALTER TABLE mcp_settings DROP COLUMN allow_statement_writes;
                ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes;
				DROP TABLE investment_snapshots;
				DELETE FROM goose_db_version WHERE version_id >= 14;
				DELETE FROM mcp_settings;
			` + fixture.mutation); err != nil {
				t.Fatal(err)
			}
			if err := database.recoverMCPMigrationHistory(context.Background()); err == nil || !strings.Contains(err.Error(), fixture.wantErr) {
				t.Fatalf("expected %q, got %v", fixture.wantErr, err)
			}
			var count int
			if err := database.conn.QueryRow(`SELECT COUNT(*) FROM mcp_settings`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("failed recovery inserted settings")
			}
			if err := database.conn.QueryRow(`SELECT COUNT(*) FROM goose_db_version WHERE version_id >= 14`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatal("failed recovery changed migration history")
			}
		})
	}
}
