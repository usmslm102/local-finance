package db

import (
	"bytes"
	"database/sql"
	"local-finance/internal/models"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func safetyDB(t *testing.T) *DB {
	t.Helper()
	d, e := NewDB(filepath.Join(t.TempDir(), "ledger.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestSafetyRestoreRejectsIncompatibleSchemaWithoutChangingLedger(t *testing.T) {
	d := safetyDB(t)
	a, e := d.GetOrCreateAccount("Keep Bank", models.AccountTypeSavings, "123", "123", "", "", "", "", "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "unrelated.db")
	c, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Exec("CREATE TABLE accounts (unrelated TEXT)"); e != nil {
		t.Fatal(e)
	}
	c.Close()
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.RestoreFrom(bytes.NewReader(raw)); e == nil {
		t.Fatal("accepted incompatible schema")
	}
	accounts, e := d.ListAccounts()
	if e != nil || len(accounts) != 1 || accounts[0].ID != a.ID {
		t.Fatalf("working ledger changed: %v %+v", e, accounts)
	}
}

func TestSafetyBackupAndRestoreKeepPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file mode test")
	}
	d := safetyDB(t)
	path := filepath.Join(t.TempDir(), "backup.db")
	if e := d.BackupTo(path); e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	// Simulate permissions widened after startup. Restore must protect the
	// active inode and any existing sidecars, not just its private staging file.
	for _, p := range []string{d.GetPath(), d.GetPath() + "-wal", d.GetPath() + "-shm"} {
		if _, e := os.Stat(p); e == nil {
			if e := os.Chmod(p, 0644); e != nil {
				t.Fatal(e)
			}
		}
	}
	if e = d.RestoreFrom(bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	for _, p := range []string{path, d.GetPath(), d.GetPath() + ".bak"} {
		info, e := os.Stat(p)
		if e != nil {
			t.Fatal(e)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("%s permissions: %04o", filepath.Base(p), info.Mode().Perm())
		}
	}
	for _, p := range []string{d.GetPath() + "-wal", d.GetPath() + "-shm"} {
		info, e := os.Stat(p)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("sidecar %s permissions: %04o", filepath.Base(p), info.Mode().Perm())
		}
	}
}

func TestSafetyConcurrentInitialSetup(t *testing.T) {
	d := safetyDB(t)
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- d.SetupSecurityPassword("hash", 60) }()
	}
	wg.Wait()
	close(results)
	wins := 0
	for e := range results {
		if e == nil {
			wins++
		} else if e != ErrSecurityConfigured {
			t.Fatal(e)
		}
	}
	if wins != 1 {
		t.Fatalf("initial setup successes=%d", wins)
	}
}

func TestSafetyTransferPairRejectsInvalidAndOccupiedPeers(t *testing.T) {
	d := safetyDB(t)
	a, _ := d.GetOrCreateAccount("Bank", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
	b, _ := d.GetOrCreateAccount("Card", models.AccountTypeCreditCard, "2", "2", "", "", "", "", "", "", nil)
	makeTx := func(hash, account string, kind models.TxType) *models.Transaction {
		tx := &models.Transaction{AccountID: account, TxHash: hash, TxDate: "2026-10-01", TxType: kind, Amount: 100, RawNarration: hash}
		if _, e := d.UpsertTransaction(tx); e != nil {
			t.Fatal(e)
		}
		return tx
	}
	debit := makeTx("debit", a.ID, models.TxTypeDebit)
	other := makeTx("other", a.ID, models.TxTypeDebit)
	credit := makeTx("credit", b.ID, models.TxTypeCredit)
	next := makeTx("next", b.ID, models.TxTypeCredit)
	for _, pair := range [][2]string{{debit.ID, "missing"}, {debit.ID, debit.ID}, {debit.ID, other.ID}, {credit.ID, debit.ID}} {
		if e := d.LinkTransferPair(pair[0], pair[1], "MANUAL_PAIR"); e == nil {
			t.Errorf("accepted invalid pair %v", pair)
		}
	}
	if e := d.LinkTransferPair(debit.ID, credit.ID, "MANUAL_PAIR"); e != nil {
		t.Fatal(e)
	}
	if e := d.LinkTransferPair(debit.ID, next.ID, "MANUAL_PAIR"); e == nil {
		t.Fatal("relinked occupied peer")
	}
	if e := d.UnlinkTransferPair(debit.ID); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{debit.ID, credit.ID, next.ID} {
		tx, e := d.GetTransaction(id)
		if e != nil {
			t.Fatal(e)
		}
		if tx.IsTransfer || tx.TransferPeerID != nil {
			t.Errorf("orphaned transfer %+v", tx)
		}
	}
}

func TestSafetyFailedRestoreBackupLeavesActiveLedger(t *testing.T) {
	d := safetyDB(t)
	backup := filepath.Join(t.TempDir(), "backup.db")
	if e := d.BackupTo(backup); e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(backup)
	if e != nil {
		t.Fatal(e)
	}
	a, e := d.GetOrCreateAccount("Active", models.AccountTypeSavings, "1", "1", "", "", "", "", "", "", nil)
	if e != nil {
		t.Fatal(e)
	}
	// A non-empty directory prevents replacing .bak on every supported platform.
	if e = os.Mkdir(d.GetPath()+".bak", 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(d.GetPath()+".bak", "keep"), []byte("keep"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = d.RestoreFrom(bytes.NewReader(raw)); e == nil {
		t.Fatal("restore continued after backup failed")
	}
	accounts, e := d.ListAccounts()
	if e != nil || len(accounts) != 1 || accounts[0].ID != a.ID {
		t.Fatalf("failed restore changed active ledger: %v", e)
	}
}

func TestSafetyRestoreMigratesOlderLedger(t *testing.T) {
	d := safetyDB(t)
	backup := filepath.Join(t.TempDir(), "older.db")
	if e := d.BackupTo(backup); e != nil {
		t.Fatal(e)
	}
	old, e := sql.Open("sqlite", backup)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = old.Exec("DROP VIEW ledger_transactions; DROP VIEW personal_transactions; DROP TABLE splitwise_entries; DROP TABLE splitwise_member_aliases; DROP TABLE splitwise_groups; DROP TABLE investment_snapshots; ALTER TABLE mcp_settings DROP COLUMN allow_categorization_writes; DELETE FROM goose_db_version WHERE version_id >= 15"); e != nil {
		t.Fatal(e)
	}
	old.Close()
	raw, e := os.ReadFile(backup)
	if e != nil {
		t.Fatal(e)
	}
	if e = d.RestoreFrom(bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	if _, e = d.ListInvestmentSnapshots(); e != nil {
		t.Fatalf("restored old backup not migrated: %v", e)
	}
}

func TestSafetyRecurringIDsAreParameters(t *testing.T) {
	d := safetyDB(t)
	if e := d.MarkTransactionsRecurring([]string{"'); DROP TABLE transactions; --"}); e != nil {
		t.Fatal(e)
	}
	if _, _, e := d.ListTransactions(TransactionFilter{Limit: 1}); e != nil {
		t.Fatal("transaction IDs were executed as SQL")
	}
}
