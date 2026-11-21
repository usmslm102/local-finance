# Performance and experience improvements

Implemented against main commit `9da62e02810ebfd01baa17c41463e78e1cfa6a69`.
The original audit and independent correction reviews used Astra. Both final
reviews reported no remaining regression blockers in the reviewed paths.

## Audit closure

| Finding | Implementation |
| --- | --- |
| B1 Merchant source lookup | Directory explicitly omits unused source aggregation; default API and MCP retain it. |
| B2 Budget reads | One grouped spending read for all categories, with calendar month boundaries. |
| B3 Subscription scans | Lean candidate projection and chunked recurring updates; detection remains synchronous after imports. |
| B4 Cash-flow category reads | Two grouped/window reads replace queries per category; personal shares and blank-payee ranking are retained. |
| B5 Rule compilation | Match and exclusion expressions prepared once per operation, without shared mutable caches. |
| B6 Preview duplicates | Existing hashes checked in bounded batches; transaction fingerprints unchanged. |
| B7 Splitwise reconciliation | Only pending review rows loaded; finished history and unnecessary member reads skipped. |
| F1 Overview hierarchy | Period controls and headline results appear together; accounts, statements and trends use a disclosure. |
| F2 Card statement meaning | Latest distinct statement version counted once per account; archive and import ownership retained. Labels distinguish imported snapshots from verified unpaid debt. |
| F3 Exact category evidence | Overview categories open paginated purchases with the selected inclusive dates and URL state. |
| F4 Import ownership | Every staged file owns its reviewed parser, account and password settings; explicit re-preview applies changed defaults. Single-file imports cannot overlap a batch. |
| F5 Merchant history | Stable server pagination makes all history reachable; MCP supports page sizes 1 through 200. |
| F6 Failed edits | Annotation drafts survive errors, retry feedback is visible, and only edited fields are patched. |
| F7 Merchant directory | Persistent search and 25-item presentation pages; filters reset paging. Directory totals still cover the filtered result. |
| F8 Merchant summaries | Average uses purchase count, top category uses aggregate spending, and top merchant is independent of display sort with deterministic ties. |

## Data freshness and regression corrections

Authenticated clients observe a content-free database revision on first success,
focus and every five seconds while visible. Writes through HTTP, MCP, another
SQLite connection, and successful restores change the revision. Active reads are
canceled before invalidation so delayed initial responses cannot replace newer
data. Local mutation completion awaits this refresh too. Read-only POST requests
do not trigger an invalidation loop. These are imported ledger snapshots; there
is no live bank connection.

Correction tests cover an original bill followed by an amendment and a re-upload
of the original, including tied arrival timestamps. The amendment remains current
and all physical statement records remain available. Other correction cases
include restore revision changes, nullable payees, malformed month fallback,
out-of-range merchant pages, stale initial fetches and saves, mixed-account import
settings, and valid calendar URLs.

## Verification

- `CGO_ENABLED=0 go test ./...`: passed, including existing integration tests and
  new database, rules and actual MCP client/server regressions.
- `pnpm --dir frontend test`: 71 tests, 69 passed, zero failed, two existing
  environment-dependent setup skips (Claude and OpenCode).
- `pnpm --dir frontend lint`: passed with 12 existing Fast Refresh warnings.
- `pnpm --dir frontend build`: TypeScript and production Vite build passed.
- Portable server built with CGO disabled and run on port 8087 against a separate
  generated SQLite fixture. The production application and database were not edited.
- Local browser smoke checks: overview showed fixture income 50,000, spending
  12,200, liquidity 201,000 and canonical card statements 130; category detail
  exposed 122 purchases; directory next-page and search reset worked; merchant
  history exposed 50 then 37 rows for the same 87-row summary.
- Independent Astra backend review: five additional pinned-driver Go probes
  passed. Frontend final review: all nine focused tests passed. Review findings
  were corrected before PR creation.

No schema migration, destructive data repair or dependency upgrade is included.
Directory pagination bounds rendered cards; its aggregate directory response
still includes all matching payees. Merchant transaction history is server-paged.
Browser smoke checks complement the automated coverage; they do not establish
that every platform or interaction is regression-free.
