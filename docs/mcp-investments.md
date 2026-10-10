# Investment MCP support

The application already supports investment views, provider format discovery and
workbook imports. Expose these through the existing MCP module; reuse investment
storage and the adapter registry without introducing a second portfolio model.

- `list_investments`: paginated snapshot summaries (newest first), existing totals,
  date, currency and holding counts. Never add holdings from different snapshots.
- `get_investment_snapshot(id)`: one snapshot's normalized holdings with pagination.
  Preserve unavailable valuation/return fields as null. Do not expose original
  worksheets or arbitrary provider fields through MCP; they may contain account
  details. Replace account references with a mask and a stable opaque portfolio key
  for grouping snapshots by provider/account/currency without revealing the reference.
  The key is the earliest stored snapshot ID, stable across imports while that
  anchor exists; deleting it through the app selects another anchor. Upload responses
  omit the key; obtain it from the read tools.
- `list_investment_formats`: current registry metadata, file limit and upload
  requirements. Native provider workbooks are the existing supported formats:
  Zerodha `.xlsx`, INDmoney `.xls`. Do not transcribe into the bank transaction CSV.
- `import_investment_statement(path)`: regular local file, absolute path, supported
  registry extension, at most 10 MiB. Reuse the statement upload file-policy helper
  (including Windows drive classification and symlink rejection) and
  `InvestmentService.Import`. Return snapshot ID, summary and duplicate status;
  holdings remain available through the detail tool. Identical file retries retain
  the existing snapshot ID. Never write bank transactions or perform live valuations.

Investment reads are available to every authenticated MCP client. Investment
uploads share `allow_statement_writes`; extend the setting's explanation to name
investment uploads. No new permission, migration or deletion capability is needed.
Tool descriptions explain complete snapshots, currency separation, original export
requirements, unknown valuations and exact-file deduplication. No application-side
AI parsing or remote price lookup. User financial fields remain untrusted data.

Verify through real SDK, SQLite and fictional provider fixtures: discovery and
permission revocation, both native formats, upload and repeat import, read parity,
account masking, paginated holdings, missing IDs, bounded/local file rejection,
and unchanged bank ledger. Review the delta from PR #15's previous head, then
update that PR and run the required checks.

Snapshot views retrieve portfolio identities together with snapshots in one database query, keeping grouping consistent during concurrent deletion and avoiding per-snapshot lookups.
