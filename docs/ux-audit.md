# UI and UX audit — 11 October 2026

Branch: `fix/shadcn-ux`, based on `origin/main` at `7236d89`.

The requested `pnpm dlx skills add shadcn/ui` installation is included in the working tree. The shadcn skill and the existing Impeccable skill informed this pass. The project already uses shadcn's Base UI `base-nova` preset; this work extends that composition across its views.

## Resolved findings

| Area | Finding and change |
| --- | --- |
| Cash flow | Proportional heights followed by minimum-height clamps overfilled crowded columns. The new allocator reserves label space before distributing remaining height. Ribbon stacks stay inside their nodes, including inconsistent routing totals. |
| Cash flow | Dense diagrams overwhelmed the available space. Compact mode groups smaller flows while preserving totals and keeping savings/shortfalls visible. Expand mode exposes every node. The desktop canvas adapts to its container; phones use a readable layer list. |
| Cash flow evidence | Category IDs were truncated, date ranges could use impossible month ends, and income-source drilldowns included unrelated credits. Drilldowns now use complete IDs, actual calendar boundaries, the backend's income classification and financial exclusions, and paginated supporting rows. Calculated savings/shortfalls explain that they have no direct ledger rows. |
| Cash flow meaning | The underlying account-to-card allocation is an overview, not a verified bill-payment relationship. The diagram now states that limitation. |
| Shared controls | Raw buttons/tables and loosely associated labels were replaced with shadcn primitives. Select/menu items use their groups. Forms use Fields with associated labels. Small option sets use ToggleGroup; reconciliation uses Tabs. Search-and-clear uses InputGroup. |
| Selects | Base UI controls could show internal category/account IDs before their popup mounted. The shared Select resolves option labels from its local composition. Selected labels were verified in the transaction filter and detail dialog. |
| Privacy | Financial amounts in charts, tooltips, cards, drilldowns and anomaly descriptions now honor Discreet Mode. PrivacyAmount masks its accessible text. Keyboard shortcuts ignore typing, composition, repeat events and active select/dialog controls. |
| Accessibility | Icon controls have names, clickable rows/cards support keyboards, filter-removal icons are Buttons, and the app has a skip link and route-specific document titles. The lock screen is a controlled shadcn modal with labeled validation and password controls. |
| Calendar | The initial month and latest-activity target follow actual imported data instead of a hardcoded August 2026. The grid fits phones; day details remain available in a dialog. |
| Responsive layouts | Transaction presets wrap; Settings has a mobile section Select; merchant filters stack; reconciliation tabs fit; release headings wrap. Dialogs have a viewport height limit and scrolling. |
| State feedback | Views have consistent Skeleton loading and retryable Alert errors. Merchant profile failures no longer appear as perpetual loading. Budget, card and reconciliation mutation failures are visible. Copy and annual-review clipboard failures are reported. |
| Charts and styling | Dashboard and salary charts use ChartContainer. The spending legend includes all categories. Semantic status colors, native component variants, merchant AvatarFallbacks and consistent typography replace scattered presentation rules. |
| Motion and themes | Number/card motion respects reduced-motion preferences. System theme changes are observed live, and invalid or unavailable theme storage is handled. Annual-review keyboard navigation avoids active controls. |
| Loading cost | Route views are loaded on demand. The production entry bundle is about 407 KB before compression, compared with about 1.6 MB before the change. |

## Verification

- All 17 application routes were inspected through the browser at desktop and 390-pixel phone widths, with focused confirmations of fixes. Disposable data contained 738 synthetic transactions across 18 accounts/cards, four income sources and multiple spending categories. No user database was used.
- Cash-flow compact/expanded modes, source drilldowns, privacy masking, category Select labels, transaction detail dialogs and light/dark themes were exercised. The final desktop flow canvas has no horizontal overflow at the checked desktop width. The corrected release page has no page-width overflow on the checked phone width.
- The source accessibility scan reports no unlabeled visible inputs/select triggers or click-only container findings. The browser's focused control-name checks found no unnamed visible Buttons in the confirmed views.
- `pnpm test`: **39 passed**, including eight new cash-flow/layout tests covering crowding, grouping, routing containment, exclusions, IDs, date boundaries and narrow desktop widths.
- `pnpm build`: passed, including TypeScript. `pnpm lint`: no errors; 12 Fast Refresh advisory warnings remain for files that intentionally export components alongside hooks, helpers or route definitions, including standard shadcn exports.
- `TMPDIR=/private/tmp CGO_ENABLED=0 go test ./...`: passed. The default macOS temp path caused two existing MCP file-import tests to fail on both this branch and an unchanged HEAD snapshot because of symlink protection; using the canonical temp directory resolved them.
- The server with the embedded frontend builds successfully with `CGO_ENABLED=0`. `git diff --check` passes.

The bundled Impeccable detector was run once. Its only finding was an opinion-based warning about Inter being commonly used. Inter remains intentional here: it is the established bundled font, works offline, and gives the finance tables consistent numerals. It is not treated as a usability defect.

This is a source and browser audit of the existing product surfaces. It does not claim exhaustive device/assistive-technology coverage. Password entry, software installation, destructive actions and real financial data were not exercised in the browser. Backend schema, imports, transaction identity and financial calculations are unchanged. Existing `docs/research/` files were preserved.
