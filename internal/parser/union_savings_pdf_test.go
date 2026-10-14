package parser

import (
	"bytes"
	"slices"
	"testing"

	"local-finance/internal/models"
	"local-finance/internal/parser/extractor"
)

func syntheticUnionSignedRows() []extractor.PositionalRow {
	return []extractor.PositionalRow{
		line(1, 700, "IFSC UBIN0123456"),
		line(1, 650, "Account Number: 123456789012345"),
		line(1, 550, "Statement Details"),
		line(1, 495, "Statement Period 01-10-2026 to 14-10-2026"),
		{Page: 1, Y: 447, Elements: []extractor.PositionalElement{
			{X: 39, S: "Date"}, {X: 77, S: "Transaction Id", EndX: 143}, {X: 265, S: "Remarks"}, {X: 439, S: "Amount("}, {X: 483, S: ")"}, {X: 510, S: "Balance("}, {X: 555, S: ")"},
		}},
		{Page: 1, Y: 424, Elements: []extractor.PositionalElement{
			{X: 26, S: "01-10-2026"}, {X: 85, S: "DEMO001"}, {X: 156, S: "UPI/DR/123456789012/EXAMPLE/example@upi"}, {X: 437, S: "100.00(Dr)"}, {X: 501, S: "900.00(Cr)"},
		}},
		{Page: 1, Y: 402, Elements: []extractor.PositionalElement{
			{X: 26, S: "03-10-2026"}, {X: 85, S: "DEMO002"}, {X: 156, S: "Example deposit"}, {X: 437, S: "250.00(Cr)"}, {X: 501, S: "1150.00(Cr)"},
		}},
	}
}

func TestUnionSignedAmountRejectsMalformedMoneyAndBalances(t *testing.T) {
	for _, tc := range []struct{ name, amount, balance string }{
		{"missing direction", "100.00", "900.00(Cr)"},
		{"unknown direction", "100.00(XX)", "900.00(Cr)"},
		{"zero amount", "0.00(Dr)", "900.00(Cr)"},
		{"malformed balance", "100.00(Dr)", "broken(Cr)"},
		{"inconsistent balance", "100.00(Dr)", "901.00(Cr)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := syntheticUnionSignedRows()
			rows[5].Elements[3].S, rows[5].Elements[4].S = tc.amount, tc.balance
			if _, _, err := parseUnionSavingsRows(rows); err == nil {
				t.Fatal("invalid statement accepted")
			}
		})
	}
}

func TestUnionSignedAmountOverdraftAndReverseOrder(t *testing.T) {
	rows := syntheticUnionSignedRows()
	rows[5].Elements[4].S = "100.00(Dr)"
	rows[6].Elements[4].S = "150.00(Cr)"
	rows[5], rows[6] = rows[6], rows[5]
	tx, meta, err := parseUnionSavingsRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx) != 2 || *tx[0].RunningBalance != -100 || meta.OpeningBalance != 0 || meta.ClosingBalance != 150 {
		t.Fatalf("overdraft/reverse order lost: %+v %+v", tx, meta)
	}
}

func TestUnionSignedAmountRepeatedHeadersAndWrappedRemarks(t *testing.T) {
	rows := syntheticUnionSignedRows()
	rows[5].Elements[2].S = "Example"
	continuation := extractor.PositionalRow{Page: 1, Y: 413, Elements: []extractor.PositionalElement{{X: 156, S: "wrapped remarks"}}}
	rows = slices.Insert(rows, 6, continuation)
	header := rows[4]
	header.Page = 2
	header.Elements = slices.Clone(header.Elements)
	rows[7].Page = 2
	for i := range header.Elements {
		header.Elements[i].X += 20
		header.Elements[i].EndX += 20
	}
	for i := range rows[7].Elements {
		rows[7].Elements[i].X += 20
	}
	rows = slices.Insert(rows, 7, header)
	tx, _, err := parseUnionSavingsRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx) != 2 || tx[0].RawNarration != "Example wrapped remarks" || tx[1].ReferenceNumber != "DEMO002" {
		t.Fatalf("page/continuation lost: %+v", tx)
	}
}

func TestUnionSignedAmountPageBannerBeforeRepeatedHeader(t *testing.T) {
	rows := syntheticUnionSignedRows()
	header := rows[4]
	header.Page = 2
	rows[6].Page = 2
	banner := extractor.PositionalRow{Page: 2, Y: 786, Elements: []extractor.PositionalElement{{X: 416, S: "Savings Account"}}}
	rows = slices.Insert(rows, 6, banner, header)
	tx, _, err := parseUnionSavingsRows(rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx) != 2 || tx[1].ReferenceNumber != "DEMO002" {
		t.Fatalf("page banner lost transactions: %+v", tx)
	}
}

func TestUnionSignedAmountRejectsPageWithoutTableHeading(t *testing.T) {
	rows := syntheticUnionSignedRows()
	rows[6].Page = 2
	if _, _, err := parseUnionSavingsRows(rows); err == nil {
		t.Fatal("transaction on a page without a table heading was accepted")
	}
}

func TestUnionSignedAmountPDF(t *testing.T) {
	data := syntheticBaselinePDFWithFontSize(t, syntheticUnionSignedRows(), 4)
	p, _, _ := DefaultRegistry.Detect("DownloadStatement.pdf", data)
	if p == nil || p.ID() != "union_savings_pdf_v1" {
		t.Fatal("Union signed-amount PDF not detected")
	}
	tx, meta, err := p.Parse(bytes.NewReader(data), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tx) != 2 || tx[0].Amount != 100 || tx[0].TxType != models.TxTypeDebit || tx[1].Amount != 250 || tx[1].TxType != models.TxTypeCredit {
		t.Fatalf("incorrect transactions: %+v", tx)
	}
	if tx[0].ReferenceNumber != "DEMO001" || tx[0].RawNarration != "UPI/DR/123456789012/EXAMPLE/example@upi" {
		t.Fatalf("lost narration/reference: %+v", tx[0])
	}
	if meta.OpeningBalance != 1000 || meta.ClosingBalance != 1150 || meta.TotalDebits != 100 || meta.TotalCredits != 250 || meta.StartDate != "2026-10-01" || meta.EndDate != "2026-10-14" || meta.AccountNumber != "123456789012345" {
		t.Fatalf("incorrect metadata: %+v", meta)
	}
}
