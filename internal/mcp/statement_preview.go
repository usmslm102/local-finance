package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"local-finance/internal/db"
	"local-finance/internal/parser"
	"local-finance/internal/service"
)

func statementParserID(input statementUploadInput) (string, error) {
	if input.CSVContent != "" {
		if input.ParserID != "" && input.ParserID != parser.LocalFinanceCSVParserID {
			return "", errors.New("csv_content requires localfinance_csv_v1; do not override the verified CSV parser")
		}
		return parser.LocalFinanceCSVParserID, nil
	}
	if input.ParserID != "" {
		if _, ok := parser.DefaultRegistry.Get(input.ParserID); !ok {
			return "", errors.New("unknown parser_id; discover list_parsers")
		}
	}
	return input.ParserID, nil
}

func registerStatementPreviewTool(server *sdk.Server, database *db.DB, allowPath bool) {
	registerPreviewToolSchema(server, "preview_statement", "Preview one bank/card document without saving accounts, transactions or imports. Same source modes, account_id, parser_id and password as import_statements. Inline data is available read-only; host paths require statement-upload permission. Discover list_parsers and get_statement_import_schema first. Totals and duplicate counts describe the whole file; transactions are paginated. Passwords, full account identities and source bytes are omitted. A preview is not a reservation: import revalidates its input.", statementUploadInputSchema(), func(ctx context.Context, raw json.RawMessage) (any, error) {
		var input statementUploadInput
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, errors.New("invalid argument types")
		}
		if input.Path != "" && !allowPath {
			return nil, errors.New("host paths require statement-upload permission; use inline content")
		}
		parserID, err := statementParserID(input)
		if err != nil {
			return nil, err
		}
		data, err := readStatementUpload(input)
		if err != nil {
			return nil, err
		}
		name := input.Filename
		if input.Path != "" {
			name = filepath.Base(input.Path)
		}
		preview, err := service.NewTransactionService(database).PreviewStatement(name, bytes.NewReader(data), input.AccountID, parserID, input.Password)
		if err != nil {
			return nil, errors.New("unable to preview statement; check source, account and parser")
		}
		if preview.Error != "" {
			preview.Error = "Unable to parse statement; check source/parser, or verify and convert using get_statement_import_schema."
		}
		// A safe name avoids echoing arbitrary host paths. prepareOutput redacts identities.
		preview.Filename = "statement" + filepath.Ext(name)
		return preview, nil
	})
}
