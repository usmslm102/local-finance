# MCP imports and shared-expense workflows

The [Astra coverage audit](mcp-coverage-audit.md) compares the available workflows
and records deliberately deferred operations. These additions reuse the app's
parsers, investment service and Splitwise accounting methods.

## Discovery and permissions

`get_app_info` reports the effective category/rule, statement/investment and
Splitwise write flags. `get_statement_import_schema`, `list_investment_formats`
and `get_splitwise_import_schema` describe format contracts, limits and source
modes. All finance reads require the existing authenticated local MCP channel.

The new **Allow Splitwise uploads and matching** setting defaults to off, including
on upgrade. Existing statement permissions do not enable it. It gates group CSV
imports, full member alias/pattern replacement, match/ignore/reset and settlement
confirmation. Imports and alias replacement can reconcile pending entries across
groups and change personal spending. Revoke, restore and reset clear all write
permissions. Omitted settings fields retain existing preferences.

## Preview and import

- `preview_statement` accepts the bank import source modes, optional account,
  registered parser ID and password. It reports whole-file totals/duplicates and
  paginated transaction evidence without saving an account or import. Explicit
  parser selection is also accepted by `import_statements`; verified `csv_content`
  always pins CSV v1 and rejects a conflicting override.
- `preview_investment_statement` and the existing
  `import_investment_statement` accept either a regular local path or an original
  basename with `content_base64`, up to 10 MiB decoded. Provider formats come from
  the existing registry. Same bytes across transports retain the saved snapshot
  ID; missing values remain null, currencies separate, and raw worksheets/account
  references are omitted.
- `preview_splitwise_statement` and `import_splitwise_statement` accept an original
  CSV path, basename/base64, or basename/CSV text, plus explicit `group` and
  `person`. Input uses decimal INR; output `_cents` fields use integer paise.
  Limits are 10 MiB, 100 member columns and 50,000 rows. The app's exact or unique
  first-name selection, balanced member shares and bank-evidence rules apply.

Inline previews are available read-only. Opening host paths requires the matching
upload permission, even for previews. Network/device paths, symlinks, unsupported
extensions and mixed source modes are rejected. The authenticated request limit
remains 32 MiB. Previews do not reserve a file or future matches; imports revalidate
their input. No source bytes, passwords or filesystem error details are returned.

Splitwise imports return the app counters. Inserted/duplicate counts refer to the
incoming rows; reconciliation counters can include existing pending records.
Skipped/uninvolved rows can be removed, so a repeated export need not report every
row as a duplicate. Bank amounts and balances remain original cash movements.

## Review and supporting evidence

Use `list_splitwise_entries` (exact group/status filters),
`list_splitwise_members`, `list_splitwise_mappings`,
`list_splitwise_candidates` and `list_splitwise_settlement_candidates`.
Filtering happens before result paging. Entry candidate shortlists cap at 50 and
settlement suggestions at 100 before MCP paging; these are not all-ledger totals.
Primitive member/alias/format arrays remain complete when a parent record is paged.

`confirm_splitwise_entry` requires the fresh entry's `expected_status` and proposed
share. Amount-difference approval is explicit; existing evidence and single-use
bank-link constraints remain enforced. `save_splitwise_member_aliases` replaces
both the complete aliases and pattern; empty values clear them. Suggested regexes
and candidate matches do not themselves authorize confirmation. Reset undoes a
decision and can change spending; settlement confirmation does not move money.

`list_transactions` now also accepts `statement_import_id`, `review_needed`,
`spending_only`, `min_amount` and `max_amount`. Inclusive zero bounds are retained.
`spending_only` uses the same personal-share view as spending reports, so do not
sum it with original bank amounts. `list_credit_card_bills(latest=true)` selects
one current statement cycle/version per card; the default remains archive history.
`list_savings_goals` returns user plans, and `get_data_health` returns the app's
per-account coverage/review counts.

## Validation

Actual MCP SDK/SQLite tests cover all eight permission combinations, old-session
revocation, readonly preview nonmutation, both provider workbooks through inline
and host-path sources, cross-source deduplication, Splitwise auto matching and
personal-spend parity, ignore/reset, alias validation, complete nested attributes
and malformed input. Legacy migration, reset and restore tests include the new
default-off permission. Full Go tests/vet and frontend build/lint/tests pass;
frontend tests have two preexisting environment-dependent setup skips.
