package db

import (
	"bytes"
	"errors"
	"local-finance/internal/models"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"modernc.org/sqlite"
)

func TestRestoreValidatesQueriesBeforeReplacingLedger(t *testing.T) {
	for _, failure := range []string{"unreadable-account", "blocked-import", "missing-security-row"} {
		t.Run(failure, func(t *testing.T) {
			candidate := safetyDB(t)
			a, e := candidate.GetOrCreateAccount("Candidate", models.AccountTypeSavings, "2", "2", "", "", "", "", "", "", nil)
			if e != nil {
				t.Fatal(e)
			}
			switch failure {
			case "unreadable-account":
				_, e = candidate.Exec("UPDATE accounts SET current_balance='not-a-number' WHERE id=?", a.ID)
			case "blocked-import":
				_, e = candidate.Exec("CREATE TRIGGER block_import BEFORE INSERT ON transactions BEGIN SELECT RAISE(ABORT, 'blocked'); END")
			case "missing-security-row":
				_, e = candidate.Exec("DELETE FROM app_security")
			}
			if e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(t.TempDir(), "candidate.db")
			if e = candidate.BackupTo(path); e != nil {
				t.Fatal(e)
			}
			raw, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			active := safetyDB(t)
			keep, e := active.GetOrCreateAccount("Keep", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
			if e != nil {
				t.Fatal(e)
			}
			if e = active.RestoreFrom(bytes.NewReader(raw)); e == nil {
				t.Fatal("accepted backup with broken application queries")
			}
			accounts, e := active.ListAccounts()
			if e != nil || len(accounts) != 1 || accounts[0].ID != keep.ID {
				t.Fatalf("changed active ledger: %v", e)
			}
		})
	}
}

func TestInterruptedSQLiteRestorePreservesLiveLedger(t *testing.T) {
	candidate := safetyDB(t)
	if _, err := candidate.GetOrCreateAccount("Candidate", models.AccountTypeSavings, "2", "2", "", "", "", "", "", "", nil); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "candidate.db")
	if err := candidate.BackupTo(path); err != nil {
		t.Fatal(err)
	}
	active := safetyDB(t)
	keep, err := active.GetOrCreateAccount("Keep", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	interrupted := errors.New("simulated interruption after copying a page")
	steps := 0
	err = copySQLiteWithStep(active.conn, path, true, func(backup *sqlite.Backup) (bool, error) {
		steps++
		if steps > 1 {
			return false, interrupted
		}
		more, err := backup.Step(1)
		if err == nil && !more {
			t.Fatal("fixture must span multiple pages to exercise an incomplete copy")
		}
		return more, err
	})
	if !errors.Is(err, interrupted) {
		t.Fatalf("expected interrupted copy, got %v", err)
	}
	accounts, err := active.ListAccounts()
	if err != nil || len(accounts) != 1 || accounts[0].ID != keep.ID {
		t.Fatalf("interrupted copy changed ledger: %v", err)
	}
	if _, err := active.GetOrCreateAccount("Still writable", models.AccountTypeSavings, "3", "3", "", "", "", "", "", "", nil); err != nil {
		t.Fatalf("connection unusable after interruption: %v", err)
	}
}

func TestRestoreQueryProbesLeaveNoRecords(t *testing.T) {
	candidate := safetyDB(t)
	if _, err := candidate.GetOrCreateAccount("Candidate", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil); err != nil {
		t.Fatal(err)
	}
	counts := func(d *DB) map[string]int {
		result := map[string]int{}
		for _, table := range []string{"accounts", "statement_imports", "credit_card_bills", "transactions", "card_reward_rules"} {
			var resultValue int
			if err := d.conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&resultValue); err != nil {
				t.Fatal(err)
			}
			result[table] = resultValue
		}
		return result
	}
	before := counts(candidate)
	path := filepath.Join(t.TempDir(), "candidate.db")
	if err := candidate.BackupTo(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	active := safetyDB(t)
	if err := active.RestoreFrom(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	if after := counts(active); !reflect.DeepEqual(before, after) {
		t.Fatalf("probe records persisted: before=%v after=%v", before, after)
	}
}
