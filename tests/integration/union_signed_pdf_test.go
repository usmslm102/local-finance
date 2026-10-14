package integration_test

import (
	"bytes"
	"testing"

	"github.com/jung-kurt/gofpdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"local-finance/internal/service"
)

func TestUnionSignedPDFEncryptedPreviewAndIdempotentImport(t *testing.T) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{UnitStr: "pt", Size: gofpdf.SizeType{Wd: 600, Ht: 800}})
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 4)
	pdf.Text(30, 50, "IFSC UBIN0123456")
	pdf.Text(30, 80, "Account Number: 123456789012345")
	pdf.Text(30, 120, "Statement Details")
	pdf.Text(30, 150, "Statement Period 01-10-2026 to 14-10-2026")
	for i, label := range []string{"Date", "Transaction Id", "Remarks", "Amount(INR)", "Balance(INR)"} {
		pdf.Text([]float64{39, 77, 265, 439, 510}[i], 200, label)
	}
	for n, row := range [][]string{
		{"01-10-2026", "DEMO001", "Example withdrawal", "100.00(Dr)", "900.00(Cr)"},
		{"03-10-2026", "DEMO002", "Example deposit", "250.00(Cr)", "1150.00(Cr)"},
	} {
		for i, value := range row {
			pdf.Text([]float64{26, 85, 156, 437, 501}[i], float64(230+n*25), value)
		}
	}
	var plain, encrypted bytes.Buffer
	if err := pdf.Output(&plain); err != nil {
		t.Fatal(err)
	}
	if err := api.Encrypt(bytes.NewReader(plain.Bytes()), &encrypted, model.NewAESConfiguration("fixture-open", "fixture-owner", 256)); err != nil {
		t.Fatal(err)
	}
	svc := service.NewTransactionService(testDatabase(t))
	for _, password := range []string{"", "wrong", "fixture-open"} {
		preview, err := svc.PreviewStatement("DownloadStatement.pdf", bytes.NewReader(encrypted.Bytes()), "", "", password)
		if err != nil {
			t.Fatal(err)
		}
		if password != "fixture-open" {
			if !preview.RequiresPassword {
				t.Fatal("wrong password accepted")
			}
			continue
		}
		if preview.Error != "" || preview.RequiresPassword || preview.TotalTransactions != 2 || preview.ParserID != "union_savings_pdf_v1" || preview.AccountNumberMask != "XX2345" || preview.TotalDebits != 100 || preview.TotalCredits != 250 {
			t.Fatalf("signed PDF preview failed: %+v", preview)
		}
	}
	result, err := svc.ImportStatement("DownloadStatement.pdf", bytes.NewReader(encrypted.Bytes()), "", "", "fixture-open")
	if err != nil {
		t.Fatal(err)
	}
	if result.InsertedCount != 2 {
		t.Fatalf("import failed: %+v", result)
	}
	repeat, err := svc.ImportStatement("DownloadStatement.pdf", bytes.NewReader(encrypted.Bytes()), "", "", "fixture-open")
	if err != nil {
		t.Fatal(err)
	}
	if repeat.AccountID != result.AccountID || repeat.InsertedCount != 0 || repeat.DuplicateCount != 2 {
		t.Fatalf("repeat import duplicated rows: %+v", repeat)
	}
}
