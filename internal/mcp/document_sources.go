package mcp

import (
	"encoding/base64"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

// Bounded original-document sources shared by investment and Splitwise tools.
// Host paths remain behind the matching upload permission, including previews.
type documentSource struct {
	Path          string `json:"path"`
	Filename      string `json:"filename"`
	ContentBase64 string `json:"content_base64"`
	CSVContent    string `json:"csv_content"`
}

func documentSourceSchema(max int64, csv bool) map[string]any {
	text := func(max int) map[string]any {
		return map[string]any{"type": "string", "minLength": 1, "maxLength": max}
	}
	properties := map[string]any{"path": text(4096), "filename": text(255), "content_base64": text(base64.StdEncoding.EncodedLen(int(max)))}
	choices := []any{
		map[string]any{"required": []string{"path"}, "properties": map[string]any{"filename": false, "content_base64": false, "csv_content": false}},
		map[string]any{"required": []string{"filename", "content_base64"}, "properties": map[string]any{"path": false, "csv_content": false}},
	}
	if csv {
		properties["csv_content"] = text(int(max))
		choices = append(choices, map[string]any{"required": []string{"filename", "csv_content"}, "properties": map[string]any{"path": false, "content_base64": false}})
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties, "oneOf": choices}
}

func (s documentSource) read(max int64, extensions []string, allowPath bool) (string, []byte, error) {
	name := s.Filename
	var data []byte
	if s.Path != "" {
		if !allowPath {
			return "", nil, errors.New("host paths require the corresponding upload permission; use bounded inline content for a read-only preview")
		}
		file, err := openLocalStatementFile(s.Path, max, extensions)
		if err != nil {
			return "", nil, err
		}
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, max+1))
		if err != nil {
			return "", nil, errors.New("unable to read local upload")
		}
		name = filepath.Base(s.Path)
	} else {
		valid := name != "" && !strings.ContainsAny(name, `/\:`) && strings.IndexFunc(name, unicode.IsControl) < 0
		supported := false
		for _, ext := range extensions {
			if strings.EqualFold(filepath.Ext(name), ext) {
				supported = true
			}
		}
		if !valid || !supported {
			return "", nil, errors.New("provide an original basename with a supported extension")
		}
		if s.CSVContent != "" {
			if !strings.EqualFold(filepath.Ext(name), ".csv") {
				return "", nil, errors.New("csv_content requires a .csv filename")
			}
			data = []byte(s.CSVContent)
		} else {
			var err error
			data, err = base64.StdEncoding.Strict().DecodeString(s.ContentBase64)
			if err != nil {
				return "", nil, errors.New("content_base64 must encode original document bytes")
			}
		}
	}
	if len(data) == 0 || int64(len(data)) > max {
		return "", nil, errors.New("document must be nonempty and within the advertised file-size limit")
	}
	return name, data, nil
}
