package service

import (
	"bytes"
	"local-finance/internal/db"
	"local-finance/internal/models"
	"local-finance/internal/testutil"
	"path/filepath"
	"testing"
)

func TestICICICCPortalPreviewImportAndReimport(t *testing.T) {
	database, err := db.NewDB(filepath.Join(t.TempDir(), "ledger.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	svc := NewTransactionService(database)
	data := testutil.ICICIPortalPDF(t)
	for _, filename := range []string{"CCLastStatement29-10-2026.pdf", "statement.pdf"} {
		preview, err := svc.PreviewStatement(filename, bytes.NewReader(data), "", "", "")
		if err != nil || preview.Error != "" || preview.ParserID != "icici_cc_pdf_v1" || preview.TotalTransactions != 3 {
			t.Fatalf("auto preview failed: %+v %v", preview, err)
		}
		manual, err := svc.PreviewStatement(filename, bytes.NewReader(data), "", "icici_cc_pdf_v1", "")
		if err != nil || manual.Error != "" || manual.TotalTransactions != preview.TotalTransactions {
			t.Fatalf("manual preview differs: %+v %v", manual, err)
		}
	}
	first, err := database.GetOrCreateAccount("ICICI Bank", models.AccountTypeCreditCard, "", "XX1234", "", "", "", "", "", "EXAMPLE CARDHOLDER", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.GetOrCreateAccount("ICICI Bank", models.AccountTypeCreditCard, "", "XX5678", "", "", "", "", "", "EXAMPLE CARDHOLDER", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range []models.Account{*first, *second} {
		result, err := svc.ImportStatement("statement.pdf", bytes.NewReader(data), account.ID, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if result.AccountID != account.ID || result.TotalParsed != 3 || result.InsertedCount != 3 {
			t.Fatalf("selected account not preserved: %+v", result)
		}
		result, err = svc.ImportStatement("renamed.pdf", bytes.NewReader(data), account.ID, "", "")
		if err != nil {
			t.Fatal(err)
		}
		if result.InsertedCount != 0 || result.DuplicateCount != 3 {
			t.Fatalf("reimport duplicated transactions: %+v", result)
		}
	}
}
