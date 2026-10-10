# Security and bug remediation

The [follow-up Gemini review](gemini-review.md) records additional corrections, the assessment of every review comment, and subsequent verification.

All ten findings in the [original audit](report.md) have been addressed in the working tree based on main commit `17bf6b43fa3181b8d10624944dcf8152cb84ec70`. No commit or push was made. Existing local changes were retained. Verification used disposable databases, never the user's financial database.

| Finding | Implemented fix | Regression coverage |
| --- | --- | --- |
| Unauthenticated password replacement | Atomic initial setup rejects already configured credentials; concurrent credential operations are serialized. | Existing password remains valid; concurrent setup has exactly one winner. |
| Missing HTTP host/origin boundary | Global loopback host and origin validation protects the SPA and API, with explicit development origins. | Foreign host, origin, and cross-site requests rejected; supported local origins accepted. |
| Authentication fails open | Security read failures and a missing configuration row deny access. | Export denied when the security table or row is missing. |
| Unsafe restore | Stage, integrity-check, migrate, and validate the complete required schema before atomic SQLite online restore into the existing connection pool. Preserve a private backup first and clear restored MCP credentials. | Invalid schema and backup failure preserve the active ledger; older backups migrate successfully. |
| Public database copies | Private temporary files, enforced Unix mode 0600, checked sidecar permissions, unique download filenames, and private default application directory. | Backup, active restored database, and recovery backup remain private on Unix. |
| Silent partial imports | Account, default rules, statement log, bill, transactions, and balance writes commit together; errors roll back and reach the caller. | Injected failures at multiple write stages leave no partial import; repeated sample import preserves hashes, categories, notes, and tags. |
| Unrelated automatic transfers | Automatic linking requires payment evidence; coincidental amount/date and refund matches remain suggestions. | Unrelated purchases/refunds remain visible; genuine bill payments still reconcile. |
| Invalid transfer relationships | Validate existence, directions, distinct accounts, and occupied peers within the transaction; unlink only reciprocal peers. | Invalid/occupied pairs rejected; unlink preserves unrelated relationships. |
| Sessions survive credential replacement | Password changes revoke all previous sessions and issue a replacement token; frontend stores it. Restore, reset, and disable revoke sessions after success. | Previous sessions lose access; replacement session and frontend follow-up requests work. |
| Unbounded processing | HTTP body limits, multipart cleanup, bounded service readers, batch limits, PDF/Excel expansion limits, and server timeouts. | Oversized requests/readers and excess file counts rejected; supported batch count accepted. |

Bank statements are limited to 20 MiB per file, with 20 files and 64 MiB combined per batch. Investment statements retain a 10 MiB limit. Database restore allows 512 MiB. Multipart requests allow an additional 1 MiB of framing overhead. PDF extraction permits at most 1,000 pages and two million text elements; Excel expansion is capped at 64 MiB, with 16 MiB XML expansion.

Recurring transaction IDs also use SQL parameters rather than interpolation. Transaction fingerprint serialization is unchanged. Imports retain the existing account and user-edit semantics.

## Dependencies and CI

The minimum Go version is now 1.27.2, with patched Go modules. TanStack Router and its Seroval dependency were updated. The unused installed shadcn CLI was removed; component additions still use the documented `pnpm dlx shadcn@latest add` workflow. Its exact original 4.19.0 Tailwind stylesheet and MIT license are retained locally. The rebuilt CSS is byte-for-byte identical to the original (SHA-256 `8be8fcb64fe07bc4880b6eb44856d719ea13973142b74c8127336b6fcffc6d1f`). React and existing component versions remain unchanged. A pnpm workspace override selects the patched source-map parser.

- [Frontend audit after fixes](frontend-audit-after.json): zero advisories at every severity.
- [Go scan after fixes](go-vulnerability-scan-after.txt): zero reachable vulnerabilities and zero vulnerabilities in imported packages. One module-only warning remains for unmaintained `golang.org/x/crypto/openpgp`, which this application does not import or call and which has no patch. Other required x/crypto packages remain in use.
- CI now runs Go vet, reachable Go vulnerability scanning, frontend lint/tests, and frontend dependency auditing alongside existing tests/builds. Frozen lockfile compatibility was checked with pnpm 10.

## Verification

- Full `CGO_ENABLED=0 go test ./... -count=1` and `go vet ./...` passed.
- Full `go test -race ./... -count=1` passed. Final security regression tests also passed under the race detector.
- Eighteen new Go regression tests cover the cases above, including subcases and injected storage failures. A frontend regression test verifies token replacement.
- Frontend lint, six reported frontend tests, TypeScript checking, and production build passed. Existing Fast Refresh, Vite configuration, and bundle-size warnings remain.
- CGO-free embedded binary builds passed for macOS ARM64 and Intel, Windows AMD64, and Linux AMD64 and ARM64.
- The rebuilt macOS ARM64 binary ran against a disposable database. Dashboard, settings, and import routes rendered, their API requests succeeded, and the browser error log was empty.
- `git diff --check` passed.

No regressions were found in these checks. Cross-platform builds are compilation checks; the Windows/Linux binaries were not executed. Unix permission checks do not establish Windows ACL behavior. The review and tests cannot establish the absence of every undiscovered vulnerability or regression.
