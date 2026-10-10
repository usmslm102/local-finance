package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3/database"
)

// recoverMCPMigrationHistory handles the observed legacy state: Goose history
// stops at 13, but the original migration-14 table already exists. A later SQL
// migration cannot repair this because migration 14 fails before it is reached.
// Published migrations remain unchanged; only an exactly matching table is
// adopted. Seeding an empty table and recording its version are atomic.
func (d *DB) recoverMCPMigrationHistory(ctx context.Context) error {
	tx, err := d.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tableSQL string
	err = tx.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'mcp_settings'`).Scan(&tableSQL)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // Fresh database: let Goose create the table normally.
	}
	if err != nil {
		return err
	}
	var hasHistory bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'goose_db_version')`).Scan(&hasHistory); err != nil {
		return err
	}
	if !hasHistory {
		return nil // Unknown legacy state; do not infer other migrations.
	}
	var maxVersion sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT MAX(version_id) FROM goose_db_version`).Scan(&maxVersion); err != nil {
		return err
	}
	if !maxVersion.Valid || maxVersion.Int64 != 13 {
		return nil // Already migrated or outside the known recovery case.
	}
	var latestVersion int64
	var applied bool
	if err := tx.QueryRowContext(ctx, `SELECT version_id, is_applied FROM goose_db_version ORDER BY id DESC LIMIT 1`).Scan(&latestVersion, &applied); err != nil {
		return err
	}
	if latestVersion != 13 || !applied {
		return nil
	}

	// Compare against the embedded, published CREATE statement, including its
	// constraints and defaults. Formatting differences are allowed; alternative
	// schemas are rejected rather than silently marked as migrated.
	migration, err := embedMigrations.ReadFile("migrations/00014_mcp_settings.sql")
	if err != nil {
		return err
	}
	_, createSQL, found := strings.Cut(string(migration), "CREATE TABLE")
	createSQL, _, terminated := strings.Cut(createSQL, ";")
	if !found || !terminated {
		return fmt.Errorf("cannot locate published MCP table definition")
	}
	normalize := func(statement string) string {
		return strings.ToLower(strings.Join(strings.Fields(statement), ""))
	}
	if normalize(tableSQL) != normalize("CREATE TABLE"+createSQL) {
		return fmt.Errorf("cannot recover MCP migration history: existing mcp_settings schema differs from migration 14")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mcp_settings (id) VALUES (1) ON CONFLICT(id) DO NOTHING`); err != nil {
		return err
	}
	store, err := database.NewStore(database.DialectSQLite3, "goose_db_version")
	if err != nil {
		return err
	}
	if err := store.Insert(ctx, tx, database.InsertRequest{Version: 14}); err != nil {
		return err
	}
	return tx.Commit()
}
