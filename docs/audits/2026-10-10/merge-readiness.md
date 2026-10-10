# Final production merge review

The remaining genuine review gaps have been corrected. No known blocking issue remains within the audited scope. This is a code merge assessment, not proof that every production environment or undiscovered failure is covered. The final commit must still pass the repository's required CI checks. Changes are staged; no commit, push, or merge has been performed.

## Remaining comments

| Comment | Assessment / resolution |
| --- | --- |
| Restore needs a second copy from .bak after a mid-copy failure | Incorrect for this implementation. SQLite keeps the destination copy in a write transaction and rolls back an unfinished operation in backup_finish. copySQLite always defers Finish. The modernc driver calls sqlite3_backup_finish and closes only the remote connection. A regression test now copies one real page, injects an interruption, and verifies that the old ledger remains intact and the existing connection remains writable. A second copy would be redundant and introduce another possible failure. The .bak recovery file is still retained. |
| Explicit restore sidecar permissions | Valid hardening gap. A shared permission helper now protects the active database and existing WAL/SHM sidecars before and after restore, including after copy failure. Startup uses the same helper. The Unix regression test deliberately widens live database/sidecar modes after startup and verifies restore returns them to 0600. |
| Ambiguous reconciliation alternatives hidden | Valid UI correctness issue. Candidate generation now returns all sorted competing suggestions instead of greedily hiding alternatives. Multiple strong matches for either debit or credit are still downgraded below the automatic-link threshold. The UI can display/select either alternative; database linking still enforces one-to-one reciprocal relationships. Regression tests inspect the summary as well as automatic linking. |
| API GET protection lacks legacy-browser fallback | Valid defense-in-depth gap; accepting a request does not alone prove a cross-site browser can read its response because same-origin/CORS protections also apply. API reads now validate Referer as well as Origin and Fetch Metadata. With no trusted browser metadata, the explicit X-LocalFinance-Request: 1 header is required. The frontend sends it for all API calls. Native scripts must include it for reads too; README documents this compatibility change. External document navigation to the SPA remains allowed. |
| Move pnpm override into package.json | Incorrect for pnpm 11. The installed pnpm implementation and official configuration documentation use pnpm-workspace.yaml. The existing override works with pnpm 10 and 11 and prevents reinstalling the vulnerable source-map parser. The YAML file is retained, and CONTRIBUTING documents where dependency declarations and overrides live. |
| Request body limiter mixes cache policy | Addressed. Authentication and MCP no-store behavior moved into one dedicated sensitiveResponseNoStore middleware, registered before authentication so error responses are covered too. |
| Duplicated development origins | Addressed with one shared origin list used by both CORS and local origin validation. |
| Confidence literals | Named constants now identify the automatic threshold, suggestion score, and scoring bonuses. |
| SQL parameterization, truthful lock-screen copy, dependency/CI changes are scope creep | Disagree for the reasons documented in the previous review: the user explicitly authorized whole-project security fixes and regression prevention. These changes remain. |

Primary references: [SQLite backup API rollback guarantee](https://www.sqlite.org/c3ref/backup_finish.html) and [pnpm project configuration](https://pnpm.io/settings). Both were checked alongside the locally installed implementations.

## Verification

- New tests failed on the earlier code for legacy GET requests, widened restore permissions, and hidden reconciliation alternatives; they pass after the fixes.
- Full CGO-free Go test suite and Go vet passed.
- Full Go race suite passed.
- Frontend lint, seven reported tests, TypeScript checking, and production build passed. Existing Fast Refresh, Vite configuration, and bundle-size warnings remain.
- CGO-free builds passed for macOS Intel/ARM64, Windows AMD64, and Linux AMD64/ARM64.
- The previous review verified a fresh frozen dependency install, complete tests, and embedded build from an isolated staged-file checkout. Final source files are staged and checked for whitespace errors before handoff.

No financial database was used. Original unstaged frontend/dist/.gitkeep deletion and untracked docs/research files remain untouched. Unix mode testing does not verify Windows ACLs, and Windows/Linux builds were not executed. Existing dependency scans show zero frontend advisories and zero reachable Go vulnerabilities; the unused OpenPGP module-only warning remains non-applicable.
