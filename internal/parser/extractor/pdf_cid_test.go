package extractor

import (
	"bytes"
	"github.com/dslipak/pdf"
	"local-finance/internal/testutil"
	"math"
	"strings"
	"testing"
)

func TestIdentityHTextWidthsAndKerning(t *testing.T) {
	data := testutil.CIDPDF(t, testutil.CIDText(45, 700, "Transaction Date")+testutil.CIDText(204, 700, "Details"))
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	text, err := pdfPageText(reader.Page(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(text) < 2 || text[0].S != "T" || text[0].W != 6 || math.Abs(text[1].X-50.76) > 0.001 {
		t.Fatalf("CID glyph advance/kerning incorrect: %+v", text)
	}
	font := reader.Page(1).Font("F1")
	for code, want := range map[int]float64{288: 500, 500: 600, 501: 600, 502: 750} {
		if got := pdfCIDWidth(font, code); got != want {
			t.Fatalf("CID width %d: %v want %v", code, got, want)
		}
	}
	rows, err := ExtractPDFPositionalRows(bytes.NewReader(data), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || len(rows[0].Elements) != 2 || rows[0].Elements[0].S != "Transaction Date" || rows[0].Elements[1].S != "Details" || rows[0].Elements[0].EndX <= rows[0].Elements[0].X {
		t.Fatalf("CID words corrupted: %+v", rows)
	}
}

func TestIdentityHTextTransformsAndMalformedGlyphs(t *testing.T) {
	content := "q 2 0 0 2 10 20 cm\n" + testutil.CIDText(45, 700, "AB") + "Q\n"
	data := testutil.CIDPDF(t, content)
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	text, err := pdfPageText(reader.Page(1))
	if err != nil {
		t.Fatal(err)
	}
	if text[0].X != 100 || text[0].Y != 1420 || text[0].W != 12 {
		t.Fatalf("CID transform incorrect: %+v", text)
	}
	bad := testutil.CIDPDF(t, "BT /F1 12 Tf 1 0 0 1 45 700 Tm <01> Tj ET")
	if _, err := ExtractPDFPositionalRows(bytes.NewReader(bad), ""); err == nil || !strings.Contains(err.Error(), "incomplete composite character") {
		t.Fatalf("malformed CID accepted: %v", err)
	}
}

func TestIdentityHContiguousPositionedRuns(t *testing.T) {
	for name, content := range map[string]string{
		"CID Tm":         "BT /F1 12 Tf 1 0 0 1 45 700 Tm <01310132> Tj 1 0 0 1 57 700 Tm <01330134> Tj ET",
		"CID Td":         "BT /F1 12 Tf 1 0 0 1 45 700 Tm <01310132> Tj 12 0 Td <01330134> Tj ET",
		"simple font Tm": "BT /F2 12 Tf 1 0 0 1 45 700 Tm (12) Tj 1 0 0 1 57 700 Tm (34) Tj ET",
		"simple font Td": "BT /F2 12 Tf 1 0 0 1 45 700 Tm (12) Tj 12 0 Td (34) Tj ET",
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := ExtractPDFPositionalRows(bytes.NewReader(testutil.CIDPDF(t, content)), "")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || len(rows[0].Elements) != 1 || rows[0].Elements[0].S != "1234" {
				t.Fatalf("contiguous positioned number split: %+v", rows)
			}
		})
	}
}

func TestIdentityHSeparatedPositionedCells(t *testing.T) {
	for name, content := range map[string]string{
		"printed space": testutil.CIDText(45, 700, "12 ") + testutil.CIDText(62.76, 700, "34"),
		"small gap":     testutil.CIDText(45, 700, "12") + testutil.CIDText(60, 700, "34"),
	} {
		t.Run(name, func(t *testing.T) {
			rows, err := ExtractPDFPositionalRows(bytes.NewReader(testutil.CIDPDF(t, content)), "")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || len(rows[0].Elements) != 2 || rows[0].Elements[0].S != "12" || rows[0].Elements[1].S != "34" {
				t.Fatalf("separate printed cells merged: %+v", rows)
			}
		})
	}
}
