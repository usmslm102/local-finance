# LocalFinance agent guide

LocalFinance is a privacy-first, offline personal finance application for Indian banking and credit cards. A Go backend embeds the React SPA and stores user data in local SQLite.

## Project constraints

- **Offline operation:** Keep application features and user data on the user's machine. No cloud services, telemetry, remote logging, SMS scraping, third-party account aggregators, or external API calls from the application. The sole exception is software release checking against public GitHub Releases, which checks for updates automatically in the background (cached for 4 hours) with a user-configurable opt-out toggle in Settings, or on-demand when the user clicks "Check for Updates".
- **Portable distribution:** Preserve the single binary with its embedded frontend and CGO-free dependencies, including `modernc.org/sqlite`. Builds must support `CGO_ENABLED=0` on Windows, macOS (Intel and ARM64), and Linux.
- **User edits survive re-import:** Import and upsert changes must preserve user-customized categories, manual tags, and notes.
- **Stable transaction identity:** Preserve `tx_hash = SHA256(account_id|date|amount|narration|ref_no|tx_type)` and the existing serialization in `internal/service/transaction_service.go` (two decimal places for amount; trimmed narration and reference). Identical statement re-uploads must remain idempotent.
- **Frontend tooling:** Use `pnpm` for all Node package management and script execution. Use shadcn/ui components exclusively; add components from `frontend/` with `pnpm dlx shadcn@latest add <component>`.

## Task-specific references

Read the sources relevant to the requested change:

- **Product scope and setup:** [ROADMAP.md](ROADMAP.md) for feature specifications; [README.md](README.md) for user-facing setup and usage.
- **Commands and versions:** [Makefile](Makefile), [frontend/package.json](frontend/package.json), and [go.mod](go.mod) are the sources of truth for available commands and dependencies.
- **Imports and deduplication:** [internal/service/transaction_service.go](internal/service/transaction_service.go) and [internal/db/db.go](internal/db/db.go) contain ingestion, fingerprinting, and upsert behavior.
- **Schema changes:** Add the next sequential SQL migration in [internal/db/migrations/](internal/db/migrations/) with Goose `-- +goose Up` and `-- +goose Down` sections. Migrations are embedded and applied at startup; users must not need a migration CLI. Preserve the WAL, busy-timeout, and single-writer connection setup in [internal/db/db.go](internal/db/db.go).
- **Bank parsers:** Follow `StatementParser` and registry contracts in [internal/parser/parser.go](internal/parser/parser.go), using an existing adapter for the relevant format as a reference. Register new adapters with `DefaultRegistry`, retain confidence-based detection, and reuse `NormalizeDate`, `CleanNarration`, and the shared [extractors](internal/parser/extractor/) for Indian statement normalization.

## Local development

| Task | Command |
| --- | --- |
| Rebuild frontend and run backend | `make dev` |
| Hot reload development, in separate terminals | `make dev-backend` and `make dev-frontend` |
| Build the embedded application without CGO | `CGO_ENABLED=0 make build` |
| Go tests | `CGO_ENABLED=0 go test ./...` (or affected packages) |
| Frontend lint / typecheck and build | `pnpm lint` / `pnpm build` from `frontend/` |

The backend embeds `frontend/dist`; build the frontend first if those assets are missing. Development backend targets use `./local_finance.db`. Use a disposable database for import or migration experiments. **`make clean` deletes `local_finance.db*` as well as build artifacts**; use targeted artifact cleanup when retaining local data.

## Completion

Carry the requested change through implementation and relevant verification, fixing failures caused by the change before handing it back. Routine local edits, builds, and checks against disposable data can proceed without additional confirmation. Match verification to the affected behavior; documentation-only edits need no application test suite. Report what changed, what was checked, and any remaining blocker.
