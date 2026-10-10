package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"local-finance/internal/models"
	"modernc.org/sqlite"
)

const MaxRestoreSize int64 = 512 << 20

// BackupTo uses SQLite's online backup API, including committed WAL pages.
// It never copies a live database file or changes the caller's connection.
func (d *DB) BackupTo(targetPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.backupTo(targetPath)
}

func (d *DB) backupTo(targetPath string) error {
	absolute, err := filepath.Abs(targetPath)
	if err != nil {
		return err
	}
	if absolute == d.path {
		return fmt.Errorf("backup target must differ from the active database")
	}
	temp, err := os.CreateTemp(filepath.Dir(absolute), ".localfinance-backup-*.db")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer func() { os.Remove(name); os.Remove(name + "-wal"); os.Remove(name + "-shm") }()
	if err := temp.Close(); err != nil {
		return err
	}
	if err := copySQLite(d.conn, name, false); err != nil {
		return fmt.Errorf("database backup failed: %w", err)
	}
	if err := os.Chmod(name, 0600); err != nil {
		return err
	}
	return os.Rename(name, absolute)
}

// copySQLite holds the single writer connection for the entire atomic copy.
// Finish rolls back an incomplete restore, and closes only the remote connection.
func copySQLite(database *sql.DB, path string, restore bool) error {
	return copySQLiteWithStep(database, path, restore, func(backup *sqlite.Backup) (bool, error) {
		return backup.Step(-1)
	})
}

func copySQLiteWithStep(database *sql.DB, path string, restore bool, step func(*sqlite.Backup) (bool, error)) error {
	connection, err := database.Conn(context.Background())
	if err != nil {
		return err
	}
	defer connection.Close()
	return connection.Raw(func(raw any) (result error) {
		driver, ok := raw.(interface {
			NewBackup(string) (*sqlite.Backup, error)
			NewRestore(string) (*sqlite.Backup, error)
		})
		if !ok {
			return fmt.Errorf("SQLite backup API unavailable")
		}
		var backup *sqlite.Backup
		if restore {
			backup, err = driver.NewRestore(path)
		} else {
			backup, err = driver.NewBackup(path)
		}
		if err != nil {
			return err
		}
		defer func() {
			if finishErr := backup.Finish(); result == nil {
				result = finishErr
			}
		}()
		for more := true; more; {
			more, err = step(backup)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) RestoreFrom(r io.Reader) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	temp, err := os.CreateTemp(filepath.Dir(d.path), ".localfinance-restore-*.db")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer func() { os.Remove(name); os.Remove(name + "-wal"); os.Remove(name + "-shm") }()
	count, copyErr := io.Copy(temp, io.LimitReader(r, MaxRestoreSize+1))
	closeErr := temp.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to stage backup: %w", copyErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if count > MaxRestoreSize {
		return fmt.Errorf("backup exceeds 512 MB")
	}
	staged, err := sql.Open("sqlite", name+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	staged.SetMaxOpenConns(1)
	defer staged.Close()
	var integrity string
	if err := staged.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		return fmt.Errorf("backup is not an intact SQLite database")
	}
	// Identify a LocalFinance backup before migrations can create missing tables.
	for _, table := range []string{"accounts", "transactions", "statement_imports", "categories", "categorization_rules"} {
		var exists int
		if err := staged.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists); err != nil || exists != 1 {
			return fmt.Errorf("backup is missing LocalFinance table %s", table)
		}
	}
	candidate := &DB{conn: staged, path: name}
	if err := candidate.migrate(); err != nil {
		return fmt.Errorf("backup migration failed: %w", err)
	}
	if err := validateRestoreSchema(staged); err != nil {
		return err
	}
	if err := candidate.clearMCPAccess(); err != nil {
		return err
	}
	if err := candidate.seedDefaultCategories(); err != nil {
		return err
	}
	if err := candidate.seedDefaultRules(); err != nil {
		return err
	}
	if err := candidate.validateRestoreQueries(); err != nil {
		return fmt.Errorf("backup application validation failed: %w", err)
	}
	// Checkpoint staging before opening it through the raw restore API.
	if _, err := staged.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}
	if err := staged.Close(); err != nil {
		return err
	}
	if err := protectDatabaseFiles(d.path); err != nil {
		return err
	}
	if err := d.backupTo(d.path + ".bak"); err != nil {
		return fmt.Errorf("failed to preserve current database: %w", err)
	}
	restoreErr := copySQLite(d.conn, name, true)
	permissionErr := protectDatabaseFiles(d.path)
	if restoreErr != nil {
		// copySQLite always calls Finish: SQLite rolls back an incomplete
		// destination write transaction before the connection returns to use.
		return errors.Join(fmt.Errorf("database restore failed: %w", restoreErr), permissionErr)
	}
	return permissionErr
}

// Exercise the application's typed reads and its import write path on staging.
// Schema metadata alone cannot detect invalid stored values or broken triggers.
// Probe writes are always rolled back and never enter the restored ledger.
func (d *DB) validateRestoreQueries() error {
	if _, _, err := d.GetSecuritySettings(); err != nil {
		return fmt.Errorf("security settings: %w", err)
	}
	if _, err := d.ListAccounts(); err != nil {
		return err
	}
	if _, err := d.ListCategories(); err != nil {
		return err
	}
	if _, err := d.ListRules(); err != nil {
		return err
	}
	// Keep memory bounded while checking every transaction, including old rows.
	for offset := 0; ; offset += 1000 {
		list, total, err := d.ListTransactions(TransactionFilter{Limit: 1000, Offset: offset})
		if err != nil {
			return err
		}
		if offset+len(list) >= total {
			break
		}
		if len(list) == 0 {
			return fmt.Errorf("transaction query returned fewer rows than its count")
		}
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	store := &statementStore{conn: tx}
	identity := uuid.NewString()
	account, err := store.GetOrCreateAccount("Restore validation", models.AccountTypeCreditCard, identity, identity, "", "", "", "", "", "", nil)
	if err != nil {
		return err
	}
	statement := &models.StatementImport{ID: uuid.NewString(), AccountID: account.ID, Filename: "restore-validation", FileHash: identity, ImportedAt: time.Now()}
	if err := store.CreateStatementImport(statement); err != nil {
		return err
	}
	if err := store.CreateOrUpdateCreditCardBill(&models.CreditCardBill{AccountID: account.ID, StatementImportID: &statement.ID}); err != nil {
		return err
	}
	transaction := &models.Transaction{AccountID: account.ID, StatementImportID: &statement.ID, TxHash: identity, TxDate: "2000-01-01", TxType: models.TxTypeDebit, Amount: 1, RawNarration: "Restore validation"}
	if _, err := store.UpsertTransaction(transaction); err != nil {
		return err
	}
	if _, err := store.UpsertTransaction(transaction); err != nil {
		return err
	}
	if err := store.RecalculateAccountBalance(account.ID); err != nil {
		return err
	}
	return tx.Rollback()
}

// Compare required columns to this binary's migrated schema, allowing additional
// columns from compatible backups. An arbitrary SQLite database is not a ledger.
func validateRestoreSchema(candidate *sql.DB) error {
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}
	defer reference.Close()
	reference.SetMaxOpenConns(1)
	if err := (&DB{conn: reference}).migrate(); err != nil {
		return err
	}
	rows, err := reference.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return err
	}
	tables := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, table := range tables {
		expected, err := tableColumns(reference, table)
		if err != nil {
			return err
		}
		actual, err := tableColumns(candidate, table)
		if err != nil {
			return err
		}
		for name, definition := range expected {
			if actual[name] != definition {
				return fmt.Errorf("incompatible backup schema: %s.%s", table, name)
			}
		}
	}
	return nil
}

func tableColumns(conn *sql.DB, table string) (map[string]string, error) {
	rows, err := conn.Query(`PRAGMA table_info("` + strings.ReplaceAll(table, `"`, `""`) + `")`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]string{}
	for rows.Next() {
		var ordinal, required, primary int
		var name, kind string
		var value any
		if err := rows.Scan(&ordinal, &name, &kind, &required, &value, &primary); err != nil {
			return nil, err
		}
		result[name] = fmt.Sprintf("%s/%d/%d", strings.ToUpper(kind), required, primary)
	}
	return result, rows.Err()
}
