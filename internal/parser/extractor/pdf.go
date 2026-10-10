package extractor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dslipak/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ErrPDFPasswordRequired distinguishes failed authentication from a damaged or
// unsupported PDF. Structural failures are not corrected by another password.
var ErrPDFPasswordRequired = errors.New("statement password required or incorrect")

// PositionalElement represents a word or token at a specific horizontal coordinate
type PositionalElement struct {
	X float64
	S string
	// EndX is the measured end of the text block; zero means unavailable.
	EndX float64
}

// PositionalRow represents a horizontal row of text elements on a specific page
type PositionalRow struct {
	Page     int
	Y        float64
	Elements []PositionalElement
}

// ExtractPDFPositionalRows extracts spatially ordered rows and words from each page
func ExtractPDFPositionalRows(r io.Reader, password string) ([]PositionalRow, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF content: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty PDF file")
	}

	// 1. Decrypt if needed
	decryptedBytes, err := DecryptPDFIfNeeded(data, password)
	if err != nil {
		return nil, err
	}

	// 2. Read with dslipak/pdf
	pdfReader, err := pdf.NewReader(bytes.NewReader(decryptedBytes), int64(len(decryptedBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse PDF structure: %w", err)
	}

	var allRows []PositionalRow
	numPages := pdfReader.NumPage()
	if numPages > 1000 {
		return nil, fmt.Errorf("statement exceeds 1000 PDF pages")
	}
	textElements := 0

	for pageIndex := 1; pageIndex <= numPages; pageIndex++ {
		p := pdfReader.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		content := p.Content()
		texts := content.Text
		textElements += len(texts)
		if textElements > 2_000_000 {
			return nil, fmt.Errorf("PDF statement contains too much text")
		}

		// Sort text by Y desc (top to bottom), X asc (left to right)
		sort.SliceStable(texts, func(i, j int) bool {
			diff := texts[i].Y - texts[j].Y
			if diff > 1.0 {
				return true
			} else if diff < -1.0 {
				return false
			}
			return texts[i].X < texts[j].X
		})

		// Group characters into continuous words/blocks with intelligent spacing
		var curY float64 = -999
		var curRow []PositionalElement
		var curWord strings.Builder
		var wordStartX float64 = 0
		var lastCharEndX float64 = 0

		flushWord := func() {
			if curWord.Len() > 0 {
				curRow = append(curRow, PositionalElement{X: wordStartX, S: strings.TrimSpace(curWord.String()), EndX: lastCharEndX})
				curWord.Reset()
			}
		}

		flushRow := func() {
			flushWord()
			if len(curRow) > 0 {
				allRows = append(allRows, PositionalRow{Page: pageIndex, Y: curY, Elements: curRow})
				curRow = nil
			}
		}

		for _, t := range texts {
			if curY == -999 || curY-t.Y > 1.0 || t.Y-curY > 1.0 {
				flushRow()
				curY = t.Y
				wordStartX = t.X
				lastCharEndX = t.X + t.W
				curWord.WriteString(t.S)
			} else {
				// Same line: check gap between end of previous char and start of this char
				gap := t.X - lastCharEndX
				if gap > 0.8 {
					if gap > 12.0 {
						// Large column gap -> new column element
						flushWord()
						wordStartX = t.X
					} else {
						// Inter-word space within same column text
						curWord.WriteString(" ")
					}
				}
				curWord.WriteString(t.S)
				lastCharEndX = t.X + t.W
			}
		}
		flushRow()
	}

	return allRows, nil
}

// ExtractPDFLines extracts clean non-empty text lines from a PDF file (supporting password decryption)
func ExtractPDFLines(r io.Reader, password string) ([]string, error) {
	rows, err := ExtractPDFPositionalRows(r, password)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, row := range rows {
		var parts []string
		for _, el := range row.Elements {
			parts = append(parts, el.S)
		}
		line := strings.Join(parts, " ")
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	return lines, nil
}

// ExtractPDFText extracts the entire raw plain text across all pages
func ExtractPDFText(r io.Reader, password string) (string, error) {
	lines, err := ExtractPDFLines(r, password)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

// IsPDFEncrypted checks if a PDF requires a password to open
func IsPDFEncrypted(r io.Reader) bool {
	data, err := io.ReadAll(r)
	if err != nil || len(data) < 4 || string(data[:4]) != "%PDF" {
		return false
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = ""
	var out bytes.Buffer
	err = api.Decrypt(bytes.NewReader(data), &out, conf)
	if err == nil {
		return false
	}
	return errors.Is(err, pdfcpu.ErrWrongPassword) || errors.Is(err, pdfcpu.ErrEncrypted)
}

// DecryptPDFIfNeeded checks if the PDF is encrypted and decrypts it with the given password.
// If the PDF is unencrypted, it returns the original data untouched.
func DecryptPDFIfNeeded(data []byte, password string) ([]byte, error) {
	if len(data) < 4 || string(data[:4]) != "%PDF" {
		return data, nil
	}

	conf := model.NewDefaultConfiguration()
	conf.UserPW = strings.TrimSpace(password)
	conf.OwnerPW = strings.TrimSpace(password)
	conf.WriteObjectStream = false
	conf.WriteXRefStream = false

	var decryptedBuf bytes.Buffer
	err := api.Decrypt(bytes.NewReader(data), &decryptedBuf, conf)
	// This failure follows successful password authentication. Some R5 exports
	// omit EncryptMetadata although their protected /Perms flag is false. Retry
	// only that missing flag; pdfcpu still verifies the password, magic, /P and
	// metadata flag. Never accept an invalid /Perms block or an explicit mismatch.
	// The compatibility path also checks reserved permission bytes and repairs
	// only legacy IV-only empty Form streams before the full decoding retry.
	if err != nil && strings.Contains(err.Error(), "password permissions: invalid permissions") {
		if normalized, normalizeErr := normalizeMissingPDFMetadataFlag(data, conf.UserPW); normalizeErr == nil {
			decryptedBuf.Reset()
			if retryErr := api.Decrypt(bytes.NewReader(normalized), &decryptedBuf, conf); retryErr == nil {
				return decryptedBuf.Bytes(), nil
			}
		}
	}
	if err != nil {
		if errors.Is(err, pdfcpu.ErrNotEncrypted) {
			return data, nil
		}
		if errors.Is(err, pdfcpu.ErrWrongPassword) {
			if strings.TrimSpace(password) == "" {
				return nil, fmt.Errorf("%w: please enter your statement password", ErrPDFPasswordRequired)
			}
			return nil, fmt.Errorf("%w: the supplied statement password was rejected", ErrPDFPasswordRequired)
		}
		return nil, fmt.Errorf("unable to decode PDF: %w", err)
	}

	return decryptedBuf.Bytes(), nil
}
