# Claims ledger
Verified 2026-10-25 against this checkout. All listed claims have high confidence from implementation; film graphics simplify actual interface layouts.

| ID | Claim | Primary repository evidence | Qualification |
|---|---|---|---|
| C01 | Private offline finance application for Indian banking and cards | AGENTS.md; README.md highlights; internal/db/db.go | Refers to the local workflow shown. Optional external AI clients have separate privacy policies. |
| C02 | Statements imported into local records | frontend/src/components/import/StatementUploader.tsx; internal/service/transaction_service.go | User selects a supported file; no bank sync. |
| C03 | PDF/CSV/Excel support and preview before import | README.md bank matrix; frontend/src/components/import/StatementUploader.tsx | Supported formats depend on bank/account/parser. |
| C04 | Rule categorization, manual tags and notes | internal/service/transaction_service.go matchCategory; frontend/src/components/transactions/TransactionTable.tsx | Rules help; user reviews results. |
| C05 | Duplicate identity and preservation of customized category/tags/notes on repeat import | internal/service/transaction_service.go; internal/db/db.go; README.md Duplicate Detection | Reimport same statement, not arbitrary fuzzy deduplication. |
| C06 | Category spending, cash flow, monthly comparison, category budgets | frontend/src/components/dashboard/SpendingChart.tsx; CashFlowChart.tsx; frontend/src/components/cashflow/MonthlyReview.tsx; frontend/src/components/budget/BudgetView.tsx | Fictional amounts; no guaranteed outcomes. |
| C07 | Go backend, embedded React frontend, local SQLite and optional release checks | AGENTS.md; go.mod; internal/db/db.go; frontend/src/hooks/use-system-update.ts | Release checks are on by default, can be disabled. Do not claim all network activity is impossible. |

Renderer references: https://www.remotion.dev/docs/cli/render and https://www.remotion.dev/docs/audio, consulted 2026-10-25. No remote assets are fetched while rendering or playing the film.
