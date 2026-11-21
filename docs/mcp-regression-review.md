# Astra regression review of MCP workflow coverage

**Disposition** — The regression checks passed after correction. No blocking findings remain in the reviewed diff against `8308c98`.

## Corrections verified

| Finding in the first implementation | Final verification |
| --- | --- |
| A per-file invalid parser selection aborted a bank batch containing a valid sibling. | A two-file SDK request imported the valid CSV and returned an individual failure for the invalid parser. Aggregate byte limits remain a before-write check. |
| Bank preview exposed a full numeric account identifier passed through the CSV mask field. | The final response withheld the original identifier in both textual and structured output. Redaction runs on the MCP projection and leaves ledger account identity alone. |
| JSON Schema accepted an integer written as `25000.0` while Go's integer decoder returned an error. The confirmation handler silently changed the stored share from 25,000 paise to zero. | The same request returned an input error, leaving both status and share unchanged. A valid ignore request containing zero preserved the stored share across ignore and reset. |
| Empty record collections lost their pagination metadata, and the initial correction changed paging for historical month/year arrays. | A filtered/read result with no records included total zero. A two-month collection retained page-two behavior, while both members remained available on the same page. Explicit attribute names cover aliases, members and provider extensions. |
| A decimal-form page number could silently return the first page of a preview. | An independent SDK request with `page: 2.0` now returns a controlled input error before parsing. Typed integer page two continues to work in committed workflow tests. |

The final code also exposes the bank preview tool and host-path flag in discovery, validates the three defined Splitwise entry statuses, accepts the full parser-supported paise range and accurately indicates whether writes are enabled when multiple permissions apply.

## Independent verification

- Ran the Go MCP and database packages with CGO disabled and caching bypassed. Both passed.
- Ran separate real-SDK Go overlay tests against disposable SQLite databases. The overlay and fixtures lived outside the checkout. The initial probes reproduced the issues above, and final assertions passed after the fixes.
- Confirmed an expense with a reviewed amount difference through MCP. Without explicit approval the tool rejected the candidate. With approval the original bank debit stayed ₹125.50 and recognized personal spending became ₹50. Confirmed a separate ₹25 settlement from participant/payment evidence and verified its original amount and transfer classification.
- Inspected the eight registration policies, omitted-setting handling, source dispatch, size checks, migration and revoke/reset/restore paths. The three host-path previews require the matching permission before opening a local file. Existing write grants do not imply the new Splitwise permission. The new tests include migration from legacy settings, explicit grants before backup, and revocation on restore.
- Reviewed the independent settings toggle and discovery copy. The implementing agent reported full Go tests/vet and frontend build/lint/test results separately.

## Scope

This review covers the local MCP change and its shared Splitwise ignore behavior. It used synthetic finance data only. Splitwise list reads still materialize the app's existing result set before filtering and paging. Large-ledger performance was not benchmarked. The FIRE planning change belongs to a separate review.
