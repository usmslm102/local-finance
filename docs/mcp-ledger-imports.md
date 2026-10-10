# MCP ledger writes: low-level design

## Scope

Extend the existing MCP listener without changing transport, credentials, transaction
identity, bank parsers or the application's manual-edit protections.

- Every MCP rule creation/update, including disabling a rule, atomically saves the
  rule and reapplies all active rules across all accounts. Manual categories remain
  untouched. Unmatched automatic categories become `cat_others`; tags and notes
  are preserved. The result retains rule fields and adds `ledger_updated_count`.
- A separate default-off `allow_statement_writes` setting exposes
  `import_statement_csv` and `delete_statement_import`. Category access alone does
  not grant file import/deletion. Revoke/reset/restore clear both permissions.
- An agent parses a source statement, verifies its transcription, writes a UTF-8
  CSV on the LocalFinance host, then calls `import_statement_csv` with its absolute
  `path`. Optional `account_id` selects an existing account; otherwise CSV account
  metadata resolves/creates it using the existing account-matching implementation.
- `delete_statement_import` accepts an exact `statement_import_id` obtained from
  import output or `list_statement_imports`. It atomically removes the import,
  transactions currently attributed to it, and its card bills; clears surviving
  reconciliation peer links; and recalculates the account balance. A missing ID is
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
Never skip a bad row. Bound input to the existing 20 MiB limit. Accept absolute
local `.csv` regular-file paths only; reject network paths and special files.
Read bytes through the existing bounded statement reader; never return file bytes
or raw filesystem errors through MCP. No application-side AI/network calls.

## Implementation seams

- `parser.LocalFinanceCSVParser` implements the existing `StatementParser`, is
  registered in `DefaultRegistry`, and has an explicit v1 ID. MCP pins this parser
  rather than permitting arbitrary parser selection or file formats.
- MCP handlers reuse `TransactionService.ImportStatement`, including atomic
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
