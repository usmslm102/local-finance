package db

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/pressly/goose/v3"
	"local-finance/internal/models"
	"local-finance/internal/parser"
)

func TestSplitwiseConsolidatedMigrationRoundTrip(t *testing.T) {
	d := safetyDB(t)
	account, err := d.GetOrCreateAccount("Demo", models.AccountTypeSavings, "demo", "demo", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	bank := &models.Transaction{AccountID: account.ID, TxHash: "migration-demo", TxDate: "2026-09-01", RawNarration: "Fictional taxi", Amount: 100, TxType: models.TxTypeDebit}
	if _, err := d.UpsertTransaction(bank); err != nil {
		t.Fatal(err)
	}
	entries, err := parser.ParseSplitwise(strings.NewReader("Date,Description,Category,Cost,Currency,Sanjay Example,Asha Example\n2026-09-01,Fictional taxi,Travel,100,INR,50,-50\n"), "Example", "Sanjay")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	if err := d.SaveSplitwiseMemberAliases(models.SplitwiseMember{Group: "Example", Name: "Asha Example", Pattern: "(?i)asha"}); err != nil {
		t.Fatal(err)
	}
	var version int
	if err := d.conn.QueryRow(`SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1`).Scan(&version); err != nil || version != 21 {
		t.Fatalf("migration version: %d %v", version, err)
	}
	migrationFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, d.conn, migrationFS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(context.Background(), 18); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := d.conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name IN ('splitwise_entries','splitwise_groups','splitwise_member_aliases','personal_transactions','ledger_transactions')`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("Splitwise objects after rollback: %d %v", remaining, err)
	}
	var amount float64
	if err := d.conn.QueryRow(`SELECT amount FROM transactions WHERE id=?`, bank.ID).Scan(&amount); err != nil || amount != 100 {
		t.Fatalf("bank cash changed on rollback: %v %v", amount, err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ImportSplitwiseAutomatically(entries); err != nil {
		t.Fatal(err)
	}
	mappings, err := d.ListSplitwiseMappings()
	if err != nil || len(mappings) != 1 || mappings[0].Bank == nil || mappings[0].Bank.ID != bank.ID {
		t.Fatalf("fresh schema after upgrade: %+v %v", mappings, err)
	}
	path := d.path
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	mappings, err = reopened.ListSplitwiseMappings()
	if err != nil || len(mappings) != 1 {
		t.Fatalf("reopen idempotency: %+v %v", mappings, err)
	}
}
