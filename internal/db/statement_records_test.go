package db

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"local-finance/internal/models"
)

func TestStatementRecordMigrationBackfillsExistingAssociations(t *testing.T) {
	database := safetyDB(t)
	account, err := database.GetOrCreateAccount("Demo Bank", models.AccountTypeSavings, "", "1234", "", "", "", "", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	statement := models.StatementImport{ID: "legacy-import", AccountID: account.ID, Filename: "legacy.csv", ImportedAt: time.Now()}
	if err := database.CreateStatementImport(&statement); err != nil {
		t.Fatal(err)
	}
	tx := models.Transaction{AccountID: account.ID, StatementImportID: &statement.ID, TxHash: "legacy-record", TxDate: "2026-09-01", RawNarration: "Demo expense", TxType: models.TxTypeDebit, Amount: 125.5, Notes: "Keep this note", Tags: "manual"}
	if _, err := database.UpsertTransaction(&tx); err != nil {
		t.Fatal(err)
	}
	migrations, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, database.conn, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.DownTo(context.Background(), 20); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := database.migrate(); err != nil {
			t.Fatal(err)
		}
		rows, total, err := database.ListTransactions(TransactionFilter{StatementImportID: statement.ID})
		if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != tx.ID || rows[0].Notes != "Keep this note" || rows[0].Tags != "manual" {
			t.Fatalf("backfilled records: %+v total=%d err=%v", rows, total, err)
		}
	}
}
