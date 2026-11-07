# MCP ledger writes: low-level design

## Scope

Extend the existing MCP listener without changing transport, credentials, transaction
identity, bank parsers or the application's manual-edit protections.

- Every MCP rule creation/update, including disabling a rule, atomically saves the
  rule and reapplies all active rules across all accounts. Manual categories remain
  untouched. Unmatched automatic categories become `cat_others`; tags and notes
  are preserved. The result retains rule fields and adds `ledger_updated_count`.
- A separate default-off `allow_statement_writes` setting exposes
  `import_statements`, `import_statement_csv` and `delete_statement_import`. Category access alone does
  not grant file import/deletion. Revoke/reset/restore clear both permissions.
- `get_statement_import_schema` is always discoverable while MCP is enabled,
  including in read-only mode. It returns the CSV table's columns as JSON Schema,
  examples, native upload modes, limits, current permissions and fallback workflow.
  This static import contract bypasses ledger redaction and pagination; it contains
  no user data and provides no SQL or internal database access.
- Agents call `import_statements` with one or multiple original bank PDFs, CSVs or
  Excel files. They use the existing parser registry and import service. For
  `parse_failed`, the agent verifies and converts only that source to CSV v1 and
  retries it. `password_required` requests a password rather than conversion.
- Uploads accept local `path`, or `filename` with either `content_base64` for
  original bytes or `csv_content` for verified CSV v1. Inline content needs no
  shared filesystem. The existing `import_statement_csv` remains a single-file
  local-path tool. Optional `account_id` selects an existing account; otherwise
  statement metadata resolves/creates it through the existing implementation.
- `delete_statement_import` accepts an exact `statement_import_id` obtained from
  import output or `list_statement_imports`. It atomically removes the import,
  transactions currently attributed to it, and its card bills; clears surviving
  reconciliation peer links and reapplies active rules to surviving automatic
  counterparts; and recalculates the account balance. A missing ID is
  an error. Accounts, categories and rules remain.

## CSV v1 contract

Required columns (any order, exact lowercase names):
`bank_name,account_type,account_number_mask,date,narration,amount,tx_type`.
Account metadata must be nonempty and identical on every row. Account types:
`SAVINGS`, `CURRENT`, `CREDIT_CARD`, `WALLET`. Currency is INR (existing default).
Dates must be valid `YYYY-MM-DD`. Amounts must be positive plain decimal values
with at most two fractional digits, no grouping separators or exponent notation.
`tx_type` is `DEBIT` or `CREDIT`; narration must reproduce the source text.

Optional columns: `account_number`, `reference_number`, `value_date`,
`running_balance`, `cleaned_payee`. Full account number, when present, must be
identical on every row. Preserve source references to retain deduplication.
Absent references fall back to references extracted from narration.
Absent cleaned payees are normalized with the shared narration cleaner. Other
payment metadata is extracted from narration using the existing cleaner.
Statement date range and debit/credit totals are derived from the rows. This
transaction CSV does not invent credit-card billing dates or due amounts.

```csv
bank_name,account_type,account_number_mask,date,narration,amount,tx_type,reference_number,running_balance
Fictional Bank,SAVINGS,XXXX1234,2026-01-10,UPI/DR/123/Corner Shop,125.50,DEBIT,123,9874.50
```

Reject unknown/duplicate/missing headers, inconsistent row widths, malformed or
empty statements, invalid values and mixed account metadata before any writes.
Never skip a bad row. Bound each file to the existing 20 MiB limit. The legacy
`import_statement_csv` accepts absolute local `.csv` regular-file paths only;
reject network paths and special files. Never return source bytes, PDF passwords
or raw filesystem errors through MCP. No application-side AI/network calls.

## Native and batch uploads

`import_statements` takes `statements` with 1–20 items. Every item has exactly one
source: `path`, `filename` + `content_base64`, or `.csv` `filename` + `csv_content`.
Optional `account_id` and PDF `password` are per file. Inline filenames must be
basenames with a supported `.pdf`, `.csv`, `.xls` or `.xlsx` extension. Local paths
use the existing no-follow regular-file reader and the same extensions. Original
files use automatic detection; `csv_content` pins `localfinance_csv_v1`.

Per-file size is 20 MiB. Total decoded inline content per batch is 20 MiB, checked
before any writes. The authenticated MCP request limit is 32 MiB, including JSON
escaping/base64, enough for a single maximum-size PDF. Split larger batches into
separate calls. Original bytes are bounded/read once, previewed to classify parser
and password failures, then imported through `TransactionService.ImportStatement`.
Original PDFs retain their source bytes and parser-provided billing metadata.

Request-schema errors reject the entire call before writing. File processing is
sequential and each import is atomic; the batch is not one database transaction.
Results preserve input order and zero-based `index`, `filename`, `status`, and
either `result` (`ImportResult`) or `error_code`/`error`. `parse_failed` also returns
`fallback_tool: get_statement_import_schema`. `invalid_source`, `preview_failed`,
`password_required` and `import_failed` distinguish other recovery paths. A failed
file creates no import/account/transaction record. A cancellation stops subsequent
files (`not_attempted`) while reporting already completed imports. Batch summaries
count `succeeded`, `failed`, and `not_attempted` files. Inspect every result and
retry failed files only; after uncertain/time-out responses, check import history
before retrying. No source bytes or passwords are echoed.

## Implementation seams

- `parser.LocalFinanceCSVParser` implements the existing `StatementParser`, is
  registered in `DefaultRegistry`, and has an explicit v1 ID. MCP pins this parser
  rather than permitting arbitrary parser selection or file formats.
- MCP handlers reuse `TransactionService.PreviewStatement` and `ImportStatement`, including atomic
  account/import/transaction writes, existing hashes/upserts and subscription scan.
- Rule operations share `statementStore` methods so save + reapplication use one
  SQL transaction under the existing writer lock. UI reapply retains its existing
  unmatched-category behavior; MCP recomputes unmatched automatic categories.
- Deletion uses the same writer lock and SQL transaction; it does not introduce a
  parallel ledger or new ownership schema. Existing upserts associate duplicate
  transactions with the latest upload, so deleting an older overlapping upload
  preserves those rows, while deleting the latest removes them. This is stated
  explicitly in the destructive tool description and documentation.

## Verification

Exercise real parsers, SQLite and SDK connections: strict CSV errors, create and
reimport/manual edits, all-account rule reapplication/disable/rollback, deletion
and overlapping imports, balances/peer links/bills, independent permission
discovery/execution and migration/reset/restore. Run Go tests/vet and frontend
build/lint/tests. Review the committed diff from merged main along Standards and
Spec axes, fix meaningful findings, and create a new PR.

Security/design review: file uploads use platform no-follow handle traversal (pinned ancestor handles on Windows; directory-relative nonblocking opens on Linux/macOS). Rule matching is shared with statement imports. Unlinking reconciled transfers restores automatic categories atomically while preserving manual categories.
