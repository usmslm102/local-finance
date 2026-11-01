package testutil

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"strings"
	"testing"
)

// CIDText uses two-byte glyph IDs and a kerned TJ array, as browser PDF
// exports do. All characters are fictional ASCII mapped through ToUnicode.
func CIDText(x, y float64, text string) string {
	var codes []string
	for _, r := range text {
		codes = append(codes, fmt.Sprintf("%04X", r+256))
	}
	if len(codes) == 0 {
		return ""
	}
	return fmt.Sprintf("BT /F1 12 Tf 1 0 0 1 %.2f %.2f Tm [<%s> 20 <%s>] TJ ET\n", x, y, codes[0], strings.Join(codes[1:], ""))
}

func CIDPDF(t testing.TB, content string) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	cmap := "/CIDInit /ProcSet findresource begin 12 dict begin begincmap /CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def /CMapName /Fixture def /CMapType 2 def 1 begincodespacerange <0000> <FFFF> endcodespacerange 1 beginbfrange <0120> <017E> <0020> endbfrange endcmap CMapName currentdict /CMap defineresource pop end end"
	var widths strings.Builder
	for i := 32; i <= 126; i++ {
		widths.WriteString("500 ")
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 842 1191] /Resources << /Font << /F1 4 0 R >> >> /Contents 7 0 R >>",
		"<< /Type /Font /Subtype /Type0 /BaseFont /Fixture /Encoding /Identity-H /DescendantFonts [5 0 R] /ToUnicode 6 0 R >>",
		fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /Fixture /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor 8 0 R /CIDToGIDMap /Identity /DW 750 /W [288 [%s] 500 501 600] >>", widths.String()),
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(cmap), cmap),
		fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.Bytes()),
		"<< /Type /FontDescriptor /FontName /Fixture /Flags 32 /FontBBox [0 -200 1000 900] /ItalicAngle 0 /Ascent 800 /Descent -200 /CapHeight 700 /StemV 80 >>",
	}
	var output bytes.Buffer
	output.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = output.Len()
		fmt.Fprintf(&output, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := output.Len()
	fmt.Fprintf(&output, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&output, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&output, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return output.Bytes()
}

func ICICIPortalPDF(t testing.TB) []byte {
	t.Helper()
	var content strings.Builder
	content.WriteString(CIDText(292, 1100, "VIEW LAST STATEMENT"))
	for i, pair := range [][2]string{
		{"Card Holder Name", "EXAMPLE CARDHOLDER"}, {"Card Type", "primary"},
		{"Previous Balance", "INR 50"}, {"Statement Date", "29-10-2026"},
		{"Purchases and Other Charges", "INR 100"}, {"Payment Due Date", "15-11-2026"},
		{"Payments and Other Credits", "INR 50"}, {"Total Credit Limit", "INR 10000"},
		{"Total Amount Due", "INR 100"}, {"Available Credit Limit", "INR 9900"},
		{"Minimum Amount Due", "INR 5"}, {"Total Cash Limit", "INR 1000"},
		{"Cash Advances", "INR 0"}, {"Statement Period", "01-10-2026 TO 29-10-2026"},
	} {
		x, y := 40.0, 1040-float64(i/2)*34
		if i%2 == 1 {
			x = 416
		}
		content.WriteString(CIDText(x, y, pair[0]))
		content.WriteString(CIDText(x+188, y, pair[1]))
	}
	content.WriteString(CIDText(40, 760, "Transaction Details"))
	positions := []float64{45, 204, 363, 522, 681}
	for i, label := range []string{"Transaction Date", "Details", "Amount (INR)", "Reward Points", "Reference Number"} {
		content.WriteString(CIDText(positions[i], 716, label))
	}
	for i, row := range [][5]string{
		{"27-10-2026", "EXAMPLE MERCHANT SHOP STORE", "100 Dr.", "2", "12345678901"},
		{"26-10-2026", "EXAMPLE REFUND", "20 Cr.", "0", "12345678902"},
		{"26-10-2026", "INFINITY PAYMENT", "30 Cr.", "0", "12345678903"},
	} {
		y := 697 - float64(i)*35
		for j, cell := range row {
			content.WriteString(CIDText(positions[j], y, cell))
		}
		if i == 0 {
			content.WriteString(CIDText(204, y-14, "MUMBAI"))
		}
		if i == 2 {
			content.WriteString(CIDText(204, y-14, "RECEIVED, THANK YOU"))
		}
	}
	return CIDPDF(t, content.String())
}
