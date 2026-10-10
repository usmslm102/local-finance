# LocalFinance security and bug audit

This report records the original findings before fixes. See [remediation and verification](remediation.md) for the subsequent fixes and current scan results.

Reviewed commit: `17bf6b43fa3181b8d10624944dcf8152cb84ec70` on `main`. `git pull --ff-only origin main` reported already up to date. Existing deleted `frontend/dist/.gitkeep` and untracked `docs/research/` were preserved. Application code was not changed.

This is a broad source review and targeted reproduction pass, not a guarantee that all bugs have been discovered. Review covered HTTP routing/authentication, MCP protection, database lifecycle and ingestion, reconciliation, parser/extractor resource handling, investment persistence, frontend authentication/update flows, CI, and dependency advisories. All reproduction data was disposable. No user financial database was opened.

## Findings, ordered by priority

### 1. P1 — Unauthenticated setup replaces an existing master password

**Locations:** `internal/api/auth.go:155`, `internal/api/auth.go:200`, `internal/db/db.go:3976`.

`/api/auth/setup` is public even when authentication is enabled. `SetupAuth` never checks whether setup already occurred. `SetSecurityPassword` uses an unconditional upsert that overwrites the existing hash. A client can submit a new password without a session or the current password, receive a valid session, and read/export/delete financial data. The loopback listener limits direct network reachability but does not fix the authorization bypass.

**Evidence:** `TestAuditAuthOverwriteAndHost` enabled security first, then sent unauthenticated setup. Setup and a subsequent full JSON export using the returned token both returned HTTP 200.

**Fix:** Atomically permit initial setup only when credentials have never been configured; reject subsequent setup and use the authenticated password-change flow. A handler-only check would still permit concurrent setup races.

### 2. P1 — Main HTTP API accepts arbitrary Host values

**Locations:** `internal/api/routes.go:17`, `internal/api/mcp.go:45`.

Host and origin protection exists only for MCP management routes; ordinary financial/authentication routes accept an attacker-controlled Host. A DNS-rebinding setup can potentially expose the loopback API through an attacker origin. Combined with finding 1, authentication cannot protect the ledger from this route to access. Binding to `127.0.0.1` alone does not establish a trusted HTTP origin.

**Evidence:** The HTTP-level probe supplied `Host` and `Origin` equal to `http://attacker.example:8080`; setup and export were accepted. This proves missing server validation, not a complete browser exploit. Browser protections against local-network access vary and were not tested.

**Fix:** Validate allowed loopback hosts and trusted origins globally before authentication; reject cross-site mutations and retain narrowly scoped development origins.

### 3. P1 — Authentication fails open on security-settings errors

**Location:** `internal/api/auth.go:143`.

On a database error, middleware calls the next handler for every route except MCP management. Corruption or an incompatible restore can therefore turn an enabled password gate into anonymous access wherever the downstream query still works. This is conditional on an operational database failure; it does not mean anonymous callers can directly drop tables.

**Evidence:** `TestAuditAuthFailOpen` removed `app_security` in a disposable database. An unauthenticated JSON export returned HTTP 200.

**Fix:** Treat unreadable security configuration as a server error and deny access. Distinguish a valid explicitly disabled configuration from a failed read.

### 4. P1 — Restore accepts incompatible databases and replaces the working ledger

**Locations:** `internal/db/db.go:1764`, `internal/db/db.go:1783`, `internal/db/db.go:1803`.

Validation checks only the SQLite header and presence of any table. It does not validate LocalFinance schema or integrity before swapping files. Migrations and seeding then run after replacement; seed errors are ignored and migration failures do not restore/reopen the old database. The `.bak` file offers manual recovery, but the active application can be broken despite a success response.

**Evidence:** `TestAuditInvalidRestoreReplacesDatabase` uploaded a SQLite file containing only `accounts(unrelated TEXT)`. Restore returned nil; `ListAccounts` subsequently failed with `no such column: id`.

**Fix:** Validate integrity, migration compatibility, required columns, and essential queries on a staged database before replacement. Make replacement recoverable through all failure paths, including connection reopen and migration failures.

### 5. P1 — Restore and backups lose private file permissions

**Locations:** `internal/db/db.go:1700`, `internal/db/db.go:1742`, `internal/api/handlers.go:446`.

Startup applies mode 0600, but restore creates a file using ordinary `os.Create` permissions and renames it over the protected database without chmod. `VACUUM INTO` backups likewise inherit the process umask. With umask 022 these copies contain the entire unencrypted ledger and password hash while being readable by other local users, wherever directory permissions allow access. Temporary download backups are placed in a temporary directory; update backups persist.

**Evidence:** `TestAuditRestorePermissions` observed both backup and restored database mode 0644, compared with startup's intended 0600. This is a Unix permissions finding; Windows ACL behavior was not validated.

**Fix:** Create staged files privately, enforce 0600 for backups/restores and sidecars, and use unique private temporary files for downloads. Do not rely on the caller's umask.

### 6. P1 — Import discards write errors and reports success

**Locations:** `internal/service/transaction_service.go:160`, `internal/service/transaction_service.go:238`.

Statement-log and bill errors are ignored; failed transaction upserts simply continue. Import is not atomic. Disk, constraint, or other write failures can leave a partially imported ledger while the upload appears successful, producing inaccurate financial totals and an import log claiming more transactions than were saved.

**Evidence:** `TestAuditImportSilentlyLosesRows` used a SQLite trigger to simulate transaction write failures. Import returned no error with eight parsed transactions, zero inserted, and zero duplicates.

**Fix:** Persist account/import/bill/transaction changes in a transaction, abort on any write failure, and roll back. Return an explicit failure rather than silently skipping rows; preserve transaction hashes and user edits.

### 7. P1 — Automatic reconciliation hides unrelated purchases

**Locations:** `internal/service/reconciliation_service.go:96`, `internal/service/reconciliation_service.go:258`.

An exact amount match plus dates within one day yields confidence 0.90, exceeding the 0.85 automatic-link threshold, even when neither narration suggests a card payment. A savings purchase and unrelated card refund can be marked internal transfers and removed from financial activity totals.

**Evidence:** `TestAuditUnrelatedAutoTransfer` inserted a ₹500 grocery debit and a ₹500 merchant refund on the same date. Scanning automatically linked them without payment evidence.

**Fix:** Require affirmative card-payment evidence for automatic linking. Amount/date matches alone should remain suggestions, especially when refunds or multiple possible matches exist.

### 8. P2 — Manual relinking corrupts reciprocal transfer relationships

**Location:** `internal/db/db.go:2653`.

Linking does not validate existence, transaction direction, distinct IDs/accounts, or existing peers. Linking A–B and then A–C leaves B pointing to A. Unlinking B subsequently clears A's new relationship while C still points to A. This creates orphaned transfer flags and incorrect exclusions.

**Evidence:** `TestAuditTransferRelink` reproduced this with three transactions; it also demonstrated that two DEBIT transactions can be linked successfully.

**Fix:** Validate pair invariants inside the database transaction. Reject occupied peers or atomically detach the previous reciprocal relationships before creating a new pair. Unlink only the verified reciprocal peer.

### 9. P2 — Password changes retain previously issued sessions

**Location:** `internal/api/auth.go:328`.

Changing the master password updates the hash without invalidating sessions. A leaked session token retains full access until its original expiry, even after the owner changes credentials to recover control.

**Evidence:** `TestAuditPasswordChangeKeepsOldSessions` created two sessions, changed the password using one, and successfully exported the database using the other.

**Fix:** Revoke existing sessions on credential replacement and issue a fresh session for the current user. Clear sessions across disable/re-enable and restore boundaries as well.

### 10. P2 — Statement and restore uploads have no request-size ceiling

**Locations:** `internal/api/handlers.go:163`, `internal/api/handlers.go:205`, `internal/api/handlers.go:462`, `internal/service/transaction_service.go:34`, `cmd/server/main.go:75`.

Unlike the investment upload endpoint, bank statements and restore do not use `http.MaxBytesReader`. Multipart memory thresholds spill to disk rather than rejecting large uploads. Statement processing then reads the entire file into memory, and restore copies it without a bound. The main HTTP server also omits read/header/idle timeouts. Large or slow requests can exhaust memory, disk, or connections.

**Evidence:** Source inspection; resource exhaustion was not deliberately triggered. Reachability depends on authentication and local API exposure.

**Fix:** Apply documented total and per-file limits before multipart parsing, limit batch count and aggregate bytes, bound parser expansion/work, and configure appropriate HTTP timeouts.

## Dependency scan results

The full [Go scanner output](go-vulnerability-scan.txt) contains **12 symbol-level findings**. This is static reachability evidence, not twelve demonstrated exploitable application vulnerabilities. Some traces are broad interface matches; HTTP/2/HTTP/3 server advisories require protocol/configuration analysis because the application listener serves plain HTTP. The installed scanning toolchain was `go1.27.0`; `go.mod` declares `1.26.6`, so build-toolchain results must be checked against the actual release compiler.

The scanner flags `golang.org/x/text v0.40.0`, with a fix in v0.41.0, along the PDF compatibility and Excel paths. The underlying advisory describes a crafted-input panic in the Nickname profile; profile-specific application exploitability still needs verification. See the [official Go advisory](https://pkg.go.dev/vuln/GO-2026-6629). Other symbol findings identify `x/net` fixes in v0.60.0, `quic-go` in v0.59.1, and Go standard-library fixes in Go 1.27.2 for the installed toolchain.

The [frontend registry audit](frontend-audit.json) reports **25 advisories: 1 critical, 10 high, 11 moderate, 3 low**. The critical `proxy-addr` finding follows `shadcn → MCP SDK → Express`; no shipped-browser runtime exposure was established. Most high findings also sit beneath the shadcn CLI, which is currently declared as a production dependency. Treat them as tooling/dependency hygiene issues unless a reachable use is established.

One high advisory is `seroval v1.6.2` under TanStack Router. [The advisory](https://github.com/advisories/GHSA-jp82-f5mq-hwhp) covers memory exhaustion while deserializing untrusted Seroval JSON and gives v1.6.3 as the patch. This SPA has no identified application path deserializing attacker-supplied Seroval JSON, so app exploitability is unconfirmed. Update affected lockfile resolutions and move the CLI to development dependencies as appropriate; do not equate advisory severity with application severity.

## Verification and reproduction

- `CGO_ENABLED=0 go test ./...`: passed on the original application code.
- `pnpm lint`: passed with Fast Refresh warnings.
- `pnpm test`: five reported tests passed.
- `pnpm build`: typecheck and production build passed, with bundle-size and future Vite config warnings.
- Eight temporary audit probes passed by asserting the observed vulnerable behavior; they are evidence probes, not regression tests asserting secure behavior.
- Go and frontend vulnerability scanners completed and returned nonzero because advisories were found. An initial sandboxed frontend audit could not reach the registry; the network-enabled run completed.

The [probe source](audit_probe_test.go.txt) is preserved outside normal test discovery. To repeat on disposable data from the repository root:

```sh
cp docs/audits/2026-10-10/audit_probe_test.go.txt internal/api/audit_probe_test.go
CGO_ENABLED=0 go test ./internal/api -run TestAudit -v
rm internal/api/audit_probe_test.go
```

The permissions probe assumes a Unix environment with a conventional umask such as 022. Browser DNS-rebinding, malicious parser payloads, Windows ACLs, race stress, and all release platforms were not exercised. No fixes or dependency changes were made during this audit. Prioritize setup authorization and global host protection, followed by restore safety, permissions, atomic ingestion, and reconciliation correctness.
