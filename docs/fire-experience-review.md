# FIRE experience regression review

Reviewed the implementation on `feature/fire-experience` against `5e92176aca28c6b2b4d295b30333c5c0765dcce2`. I compared the displayed values with the existing engine point and scenario types, checked the route and legacy redirect, traced stale/error/empty states and discreet mode, and ran the focused tests. All fixtures were synthetic. I did not access production or user records.

## Final disposition after corrections

**No remaining merge blocker in the reviewed FIRE frontend.** All five findings below were corrected. I independently reran both FIRE frontend test files against the final working tree. Nine tests passed, with zero failures and zero skips.

- I extracted the trajectory formatter from the corrected `FireResults.tsx`, rendered it through the actual shared ChartContainer and ChartTooltipContent, and checked the output. A synthetic age of 43.25 produced **Age 43 years 3 months**. A missing point age produced the neutral fallback. The above-target regression test also confirms the meter saturates at 100 while the label retains 200%.
- The unreachable scenario now describes working through the horizon and explicitly labels the absence of a withdrawal phase. The evidence shortcut focuses the mounted evidence heading, whose next interactive item is the Data Health link. The guide names the dedicated sidebar page and legacy redirect.
- The runtime checks in this pass used synthetic data and server-rendered tooltip markup. I inspected the keyboard-focus change in source. The coordinator owns the separate browser smoke test and CI/release checks.

## Findings on the initial working tree

| Priority | Reproduction and consequence | Source and acceptance check |
| --- | --- | --- |
| P2 | **Hovering or keyboard-selecting a trajectory point shows an invalid age.** The shared tooltip passes the configured series label to the FIRE formatter for a numeric X value. Parsing “Total portfolio” as a number produces NaN. I rendered the actual ChartContainer and ChartTooltipContent with an example age 43.25. The rendered text was “Age NaN years.” | `frontend/src/components/planning/FireResults.tsx:82`, `frontend/src/components/ui/chart.tsx:154`–`165`. Use the source point's numeric age. A point at 43.25 must read “Age 43 years 3 months” in both money bases. |
| P2 | **The progress description understates above-target assets.** With starting accessible assets of ₹20 lakh and an initial target of ₹10 lakh, the page displays exactly 100%. Its caption claims this is the percentage of the initial target. The entered amounts give 200%. | `frontend/src/components/planning/FireResults.tsx:23`, `52`–`53`. Bound the graphical fill independently of the textual percentage, or explicitly label a saturated figure as 100%+. Cover above, at and below target. |
| P3 | **An unreachable scenario describes withdrawals that never occur in the depicted projection.** The chart caption says contributions stop at the retirement marker and subsequent values reflect withdrawals, even when retirement is never reached. The engine simulates continued employment through the horizon in that case. | `frontend/src/components/planning/FireResults.tsx:73`, `91`; `internal/retirement/projection.go:192`, `273`. State that this scenario keeps accumulating through the horizon. Render the retirement/withdrawal caption only when the phase exists. |
| P3 | **The new evidence shortcut scrolls without moving keyboard focus.** Activating Review source data selects a tab deep in the page, but focus remains on the header button. The next Tab lands on Current age, and a keyboard user has to traverse the page to reach the requested evidence. | `frontend/src/components/planning/FireView.tsx:70`, `74`, `81`, `152`–`154`. Give the evidence tab or mounted panel heading programmatic focus. Pressing Tab after the shortcut should reach an evidence control. |
| P3 | **The user guide directs people to the removed tab.** The old URL redirects correctly, but a user following the written instruction inside Savings & Planning can no longer find that tab. | `docs/fire-planning.md:3`, `frontend/src/components/planning/PlanningView.tsx:26`. Name the dedicated sidebar destination and link to the current route. |

## Checks that passed

- `/fire` is lazy-loaded, has its own shell title and command-palette entry, and precedes Savings & Planning in the sidebar. The replacement redirect specifically matches `/planning?tab=fire`. Forecast and goals retain their original routes.
- The result, annual figures and projection-specific warnings all require the same current input reference as the debounced request. Incomplete inputs, an unachieved horizon and a legitimate retirement outcome have separate states. Missing FX for USD-funded assets or grants is surfaced alongside the quick inputs.
- Both chart bases use `fireChartPoints`. For the asset split, real accessible value uses the same point's real-to-nominal factor as total value. The remaining locked amount is the difference in that basis. The starting-accessibility calculation matches the engine's currency, age and accessible-percentage rules.
- Discreet mode unmounts both charts and their tooltips and omits progress percentages. Financial labels and table cells use the masked formatting path, and sensitive numeric inputs use password presentation. Nonfinancial assumptions and retirement ages stay visible as before.
- All original scenario fields remain editable. Imported evidence refresh and explicit replacement remain distinct, and the existing opt-in storage key and scenario decoder are unchanged. The reviewed diff makes no model, API or MCP changes.
- I ran `pnpm exec node --test tests/fire.test.mjs tests/fire-display.test.mjs`. All eight passed without skips. These tests cover FX requirements, asset access, retirement-point basis conversion, missing outcomes, stored input compatibility and target cutoffs. Browser navigation, narrow-viewport and live keyboard smoke testing are being performed separately by the coordinator.

**Initial disposition.** Correct the two numerical display defects before merging. The three other findings are narrow corrections within the requested workflow.
