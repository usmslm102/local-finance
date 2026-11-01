package extractor

import (
	"fmt"
	"strings"

	"github.com/dslipak/pdf"
)

// The PDF reader decodes Identity-H strings, but computes their advances as
// single-byte fonts. Read these pages with CID widths so kerning cannot reorder
// letters and financial cells retain their actual printed coordinates.
type pdfTextGlyph struct {
	pdf.Text
	block int
}

func pdfPageText(page pdf.Page) (text []pdfTextGlyph, err error) {
	hasCID := false
	for _, name := range page.Fonts() {
		if page.Font(name).V.Key("Subtype").Name() == "Type0" {
			hasCID = true
		}
	}
	if !hasCID {
		for _, glyph := range page.Content().Text {
			text = append(text, pdfTextGlyph{Text: glyph})
		}
		return text, nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			text = nil
			err = fmt.Errorf("unsupported PDF text geometry: %v", recovered)
		}
	}()
	type state struct {
		ctm, tm, lm                                      pdfTextMatrix
		font                                             pdf.Font
		encoding                                         pdf.TextEncoding
		widths                                           map[int]float64
		size, scale, spacing, wordSpacing, leading, rise float64
	}
	g := state{ctm: pdfTextIdentity, tm: pdfTextIdentity, lm: pdfTextIdentity, scale: 1}
	var stack []state
	fontWidths := map[string]map[int]float64{}
	block := 0
	move := func(x, y float64) { g.tm = (pdfTextMatrix{1, 0, 0, 1, x, y}).mul(g.tm) }
	newLine := func() { g.lm = (pdfTextMatrix{1, 0, 0, 1, 0, -g.leading}).mul(g.lm); g.tm = g.lm }
	show := func(raw string) {
		cid := g.font.V.Key("Subtype").Name() == "Type0"
		step := 1
		if cid {
			if g.font.V.Key("Encoding").Name() != "Identity-H" {
				panic("unsupported composite font encoding")
			}
			step = 2
		}
		if len(raw)%step != 0 {
			panic("incomplete composite character")
		}
		for index := 0; index < len(raw); index += step {
			code := int(raw[index])
			if cid {
				code = code<<8 | int(raw[index+1])
			}
			width := g.font.Width(code)
			if cid {
				var ok bool
				width, ok = g.widths[code]
				if !ok {
					width = pdfCIDWidth(g.font, code)
					g.widths[code] = width
				}
			}
			matrix := (pdfTextMatrix{g.size * g.scale, 0, 0, g.size, 0, g.rise}).mul(g.tm).mul(g.ctm)
			if g.encoding == nil {
				panic("missing text font")
			}
			decoded := g.encoding.Decode(raw[index : index+step])
			text = append(text, pdfTextGlyph{Text: pdf.Text{Font: pdfUnsubsetFont(g.font.BaseFont()), FontSize: matrix[0], X: matrix[4], Y: matrix[5], W: width / 1000 * matrix[0], S: decoded}, block: block})
			if len(text) > 2_000_000 {
				panic("too much text")
			}
			advance := width/1000*g.size + g.spacing
			if !cid && code == 32 {
				advance += g.wordSpacing
			}
			move(advance*g.scale, 0)
		}
	}
	interpret := func(stream pdf.Value) {
		pdf.Interpret(stream, func(values *pdf.Stack, op string) {
			args := make([]pdf.Value, values.Len())
			for index := len(args) - 1; index >= 0; index-- {
				args[index] = values.Pop()
			}
			require := func(n int) {
				if len(args) != n {
					panic("invalid " + op + " operands")
				}
			}
			matrix := func() pdfTextMatrix {
				require(6)
				var m pdfTextMatrix
				for i := range m {
					m[i] = args[i].Float64()
				}
				return m
			}
			switch op {
			case "q":
				stack = append(stack, g)
			case "Q":
				if len(stack) == 0 {
					panic("unbalanced graphics state")
				}
				g = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
			case "cm":
				g.ctm = matrix().mul(g.ctm)
			case "BT":
				g.tm = pdfTextIdentity
				g.lm = pdfTextIdentity
			case "Tf":
				require(2)
				name := args[0].Name()
				g.font = page.Font(name)
				g.encoding = g.font.Encoder()
				if fontWidths[name] == nil {
					fontWidths[name] = map[int]float64{}
				}
				g.widths = fontWidths[name]
				g.size = args[1].Float64()
			case "Tm":
				block++
				g.tm = matrix()
				g.lm = g.tm
			case "Td", "TD":
				block++
				require(2)
				if op == "TD" {
					g.leading = -args[1].Float64()
				}
				g.lm = (pdfTextMatrix{1, 0, 0, 1, args[0].Float64(), args[1].Float64()}).mul(g.lm)
				g.tm = g.lm
			case "T*":
				newLine()
			case "Tc":
				require(1)
				g.spacing = args[0].Float64()
			case "Tw":
				require(1)
				g.wordSpacing = args[0].Float64()
			case "Tz":
				require(1)
				g.scale = args[0].Float64() / 100
			case "Ts":
				require(1)
				g.rise = args[0].Float64()
			case "TL":
				require(1)
				g.leading = args[0].Float64()
			case "Tj":
				require(1)
				show(args[0].RawString())
			case "'":
				require(1)
				newLine()
				show(args[0].RawString())
			case "\"":
				require(3)
				g.wordSpacing = args[0].Float64()
				g.spacing = args[1].Float64()
				newLine()
				show(args[2].RawString())
			case "TJ":
				require(1)
				for i := 0; i < args[0].Len(); i++ {
					v := args[0].Index(i)
					if v.Kind() == pdf.String {
						show(v.RawString())
					} else {
						move(-v.Float64()/1000*g.size*g.scale, 0)
					}
				}
			}
		})
	}
	contents := page.V.Key("Contents")
	if contents.Kind() == pdf.Array {
		for i := 0; i < contents.Len(); i++ {
			interpret(contents.Index(i))
		}
	} else {
		interpret(contents)
	}
	return text, nil
}

type pdfTextMatrix [6]float64

var pdfTextIdentity = pdfTextMatrix{1, 0, 0, 1, 0, 0}

func (m pdfTextMatrix) mul(n pdfTextMatrix) pdfTextMatrix {
	return pdfTextMatrix{m[0]*n[0] + m[1]*n[2], m[0]*n[1] + m[1]*n[3], m[2]*n[0] + m[3]*n[2], m[2]*n[1] + m[3]*n[3], m[4]*n[0] + m[5]*n[2] + n[4], m[4]*n[1] + m[5]*n[3] + n[5]}
}

func pdfCIDWidth(font pdf.Font, code int) float64 {
	descendant := font.V.Key("DescendantFonts").Index(0)
	widths := descendant.Key("W")
	for i := 0; i < widths.Len(); {
		start := int(widths.Index(i).Int64())
		i++
		next := widths.Index(i)
		i++
		if next.Kind() == pdf.Array {
			if code >= start && code < start+next.Len() {
				return next.Index(code - start).Float64()
			}
		} else {
			end := int(next.Int64())
			width := widths.Index(i).Float64()
			i++
			if code >= start && code <= end {
				return width
			}
		}
	}
	if width := descendant.Key("DW"); !width.IsNull() {
		return width.Float64()
	}
	return 1000
}

// Retain the font name convention used by the ordinary reader.
func pdfUnsubsetFont(name string) string {
	if index := strings.Index(name, "+"); index >= 0 {
		return name[index+1:]
	}
	return name
}
