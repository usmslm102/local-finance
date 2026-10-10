# Follow-up review of Gemini comments

The subsequent [final merge-readiness assessment](merge-readiness.md) supersedes the remaining candidate-selection and API-read behavior described below.

The following assessment addresses the comments against the security remediation working tree. The historical audit is evidence of the original issues, not a complete implementation specification.

| Comment | Assessment and action |
| --- | --- |
| Essential queries missing from restore validation | Valid gap. Seeding already executed writes, but schema checks alone missed broken typed reads, missing security data, and import-blocking triggers. Staging now exercises security, accounts, categories, rules, and all transaction pages, followed by account/import/bill/transaction/upsert/balance writes in a transaction that is always rolled back. Invalid candidates cannot replace the active ledger. |
| Missing Origin / Fetch Metadata mutation protection | Valid gap. Mutations now validate a supplied Referer and require a trusted Origin/Referer, same-origin Fetch Metadata, or the explicit `X-LocalFinance-Request: 1` header. The frontend sends the header automatically; CORS permits it only for the existing development origins. Native scripts without browser metadata must add this header. The header is CSRF protection, not authentication. |
| Cross-site document navigation blocked | Valid regression. External top-level GET/HEAD navigation to the SPA is permitted. Cross-site API requests, iframe loads, and mutations remain blocked. |
| Reconciliation stops at the first weak match | Valid regression. All candidates are examined before selection, stronger candidates take priority globally, and multiple strong candidates for either transaction are downgraded to suggestions. A refund encountered first no longer hides a subsequent real payment. |
| Contributor Go prerequisite mismatch | Valid documentation inconsistency. CONTRIBUTING now agrees with go.mod, README, and ROADMAP: Go 1.27.2+. |
| Untracked CSS missing from a future checkout | Valid packaging concern, not a failure of the current build. Required new source, stylesheet, license, workspace configuration, tests, and audit files are staged with the related changes. No commit or push is made. Clean staged-source build verification is recorded below. |
| Duplicated HTTP test helper | Maintenance concern addressed by one implementation in internal/testutil; each test package retains only an alias. |
| Many tests adjusted for the host guard | Necessary fixture correction. The application's real loopback boundary should not be relaxed to accept httptest's default example.com host. The shared helper represents an explicit local API client; security tests for absent/hostile browser headers construct their own requests. |
| MCP middleware name suggests access control | Valid naming concern. Renamed to mcpManagementNoStore, reflecting its actual cache-header purpose. |
| DB methods delegate to statementStore | Intentional interface boundary retained. The same SQL implementation works with both sql.DB and sql.Tx; this is how single writes and atomic imports preserve identical behavior without duplicate persistence code. |
| TransactionFilter in the import module | Valid placement concern. Moved back beside ListTransactions in db.go; public type and behavior unchanged. |
| Recurring SQL parameterization is scope creep | Disagree. The user authorized whole-project security fixes, and bound IDs remove unsafe SQL interpolation. The original ten findings were not a restriction against additional genuine security fixes. Retained. |
| Removing “decrypt” is scope creep | Disagree. SQLite is unencrypted; the original text falsely described password protection as decryption. Retained the accurate text. |
| shadcn removal, CSS vendoring, overrides, and CI are scope creep | Disagree. These address the dependency findings and the request to prevent regressions. Moving the unused CLI to devDependencies would retain its vulnerable dependency tree. Existing shadcn components and the documented pnpm dlx component workflow remain. The original stylesheet and MIT license are preserved, with identical built CSS. CI security checks and the patched source-map override remain. |

## Verification

The new middleware, restore, and reconciliation tests failed on the previous implementation before the fixes. They then passed with the fixes. Additional tests verify that successful restore probes leave no records and that multipart/bodyless frontend mutations include the explicit local-request header.

- Full CGO-free Go suite passed.
- Frontend lint, seven reported tests, TypeScript checking, and production build passed. Existing Fast Refresh, Vite configuration, and bundle-size warnings remain.
- Full race detection and Go vet passed.
- An isolated checkout containing only staged files passed a fresh `pnpm install --frozen-lockfile --ignore-scripts`, frontend typecheck/build, the full CGO-free Go test suite, and an embedded binary build. This verification used its own dependency installation, not untracked files or the working tree's node_modules.
- CGO-free builds passed for macOS Intel/ARM64, Windows AMD64, and Linux AMD64/ARM64.
- Staged and working-tree whitespace checks passed.

No real financial database was used. The pre-existing frontend/dist/.gitkeep deletion and docs/research files are preserved outside the staged security changes. Changes are staged for review, not committed or pushed.
