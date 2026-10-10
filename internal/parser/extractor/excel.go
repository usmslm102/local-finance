package extractor

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
	"golang.org/x/net/html"
)

// ExtractExcel extracts tabular data from any Excel file (.xlsx, legacy .xls BIFF8, or HTML-disguised .xls)
func ExtractExcel(r io.Reader, password string) ([][]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read excel content: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty excel file")
	}

	// 1. Check if Modern XLSX (ZIP archive signature: 50 4B 03 04)
	if len(data) > 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		return extractXLSX(data, password)
	}

	// 2. Check if Legacy OLE2 BIFF8 XLS (Signature: D0 CF 11 E0 A1 B1 1A E1)
	if len(data) > 8 && data[0] == 0xD0 && data[1] == 0xCF && data[2] == 0x11 && data[3] == 0xE0 {
		return extractLegacyXLS(data)
	}

	// 3. Check if HTML table disguised with .xls extension
	contentSnippet := strings.ToUpper(string(data[:min(1024, len(data))]))
	if strings.Contains(contentSnippet, "<HTML") || strings.Contains(contentSnippet, "<!DOCTYPE") || strings.Contains(contentSnippet, "<TABLE") {
		return extractHTMLTable(data)
	}

	// Fallback attempt: try XLSX first, then XLS
	if rows, err := extractXLSX(data, password); err == nil && len(rows) > 0 {
		return rows, nil
	}
	if rows, err := extractLegacyXLS(data); err == nil && len(rows) > 0 {
		return rows, nil
	}

	return nil, fmt.Errorf("unrecognized or unsupported excel format")
}

func extractXLSX(data []byte, password string) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{Password: password, UnzipSizeLimit: 64 << 20, UnzipXMLSizeLimit: 16 << 20})
	if err != nil {
		return nil, fmt.Errorf("failed to open .xlsx file: %w", err)
	}
	defer f.Close()

	sheetList := f.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("no sheets found in .xlsx file")
	}

	// Read first non-empty sheet
	for _, sheetName := range sheetList {
		rows, err := f.GetRows(sheetName)
		if err == nil && len(rows) > 0 {
			var cleanRows [][]string
			for _, row := range rows {
				var cleanRow []string
				hasVal := false
				for _, cell := range row {
					trimmed := strings.TrimSpace(cell)
					if trimmed != "" {
						hasVal = true
					}
					cleanRow = append(cleanRow, trimmed)
				}
				if hasVal {
					cleanRows = append(cleanRows, cleanRow)
				} else {
					cleanRows = append(cleanRows, []string{})
				}
			}
			if len(cleanRows) > 0 {
				return cleanRows, nil
			}
		}
	}

	return nil, fmt.Errorf("no data found in .xlsx workbook")
}

func extractLegacyXLS(data []byte) ([][]string, error) {
	xlFile, err := xls.OpenReader(bytes.NewReader(data), "utf-8")
	if err != nil {
		return nil, fmt.Errorf("failed to open legacy .xls: %w", err)
	}

	sheet := xlFile.GetSheet(0)
	if sheet == nil {
		return nil, fmt.Errorf("sheet 0 is nil in legacy .xls")
	}

	var rows [][]string
	for i := 0; i <= int(sheet.MaxRow); i++ {
		row := sheet.Row(i)
		if row == nil {
			rows = append(rows, []string{})
			continue
		}
		var rowVals []string
		for j := 0; j < row.LastCol(); j++ {
			rowVals = append(rowVals, strings.TrimSpace(row.Col(j)))
		}
		rows = append(rows, rowVals)
	}

	return rows, nil
}

func extractHTMLTable(data []byte) ([][]string, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML table: %w", err)
	}

	var rows [][]string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.ToLower(n.Data) == "tr" {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (strings.ToLower(c.Data) == "td" || strings.ToLower(c.Data) == "th") {
					cellText := extractTextFromNode(c)
					row = append(row, strings.TrimSpace(cellText))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	if len(rows) == 0 {
		return nil, fmt.Errorf("no table rows found in HTML format")
	}
	return rows, nil
}

func extractTextFromNode(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractTextFromNode(c))
	}
	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
