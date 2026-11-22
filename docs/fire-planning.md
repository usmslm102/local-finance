# Retirement / FIRE planning

Open **Savings & planning → Retirement / FIRE**. This adds a local, editable
scenario; it neither trades assets nor changes the ledger. The API exposes
authenticated `GET /api/planning/fire/baseline` and read-only
`POST /api/planning/fire/project` operations. No remote market or currency feed
is used, and the scenario is not saved in the database.

The read-only MCP tools `get_fire_baseline` and `calculate_fire` expose that same
engine. The calculator's schema derives from the shared scenario type; every
assumption is explicit, including zero benefits. Projection points are paginated
while the retirement age and retirement snapshot describe the whole calculation.
Use identical confirmed inputs when retrieving later pages. MCP does not persist
scenarios or grant any additional write permissions.

## Evidence and user inputs

The baseline uses the six complete calendar months before the current month.
It retains missing months and averages observed months, while warning that
activity alone is not proof of complete statement coverage. Living costs use
the app's personal spending view, including confirmed Splitwise shares.
Transfers, excluded rows and identified investment funding are omitted.
Investment-category descendants are included. Salary evidence shares the
existing Salary Insights predicate.

Zerodha/Groww funding is suggested as Indian investing, INDmoney as INR funding
for US investing, fixed-deposit labels as deposits, and other investment-category
activity stays unclassified. These are gross outgoing funding estimates, not
proof of holdings, historical CAGR or net deposits after withdrawals. Review the
month-by-month source evidence and change the suggested SIPs as needed.

Balances come only from the latest distinct imported portfolio snapshot per
provider/account/currency. Snapshot dates and unknown valuations stay visible;
INR and USD totals are separate. The supported adapters report Indian equity
and US-stock holdings; the planner does not infer an employer relationship or
invent a stock/fund return from bank transfers. Broad asset buckets and their
illustrative return assumptions are editable. Verify mixed holdings and avoid
duplicating a portfolio already represented by an imported snapshot.

Enter age, intended future living costs, employer shares, ESPP, unvested RSUs,
new grants, PF/NPS and other off-ledger balances explicitly. Starting balances
exclude unvested grants, your primary home and emergency reserves. The user can
add additional INR asset buckets. Input persistence is opt-in and browser-local;
turning it off removes that saved scenario. Refreshing evidence does not overwrite
manual inputs. The explicit replacement action lists the fields it replaces and
keeps future-spending and employer/PF/NPS inputs.

## Calculation contract

- Balance amounts use the asset's currency. Every monthly contribution and
  fixed monthly increase uses INR, including US SIPs and ESPP. Growth inputs are
  nominal annual assumptions; returns should be net of anticipated fees/taxes.
- Working expenses use the current level until the chosen lifestyle-change
  interval, then the future level. Retirement uses the future level immediately.
  Both levels are entered in today's INR and inflate from today.
- Monthly contributions equal `(starting monthly + monthly increase × elapsed
  months) × (1 + annual step-up) ^ completed years`. Salary growth is independent.
  Optional positive take-home surplus is added after living costs and planned
  bank-funded SIPs. It follows SIP weights, then existing market/deposit values,
  then the first market/deposit bucket. Payroll PF/NPS/ESPP are not deducted twice.
  An affordability warning flags explicit SIPs not covered by entered net salary.
- USD balances earn their native USD return; INR-per-USD changes by the separate
  depreciation assumption. Currency conversion and native returns combine
  multiplicatively. Do not subtract US inflation again from a nominal USD return.
- Withdrawals occur at the beginning of a retirement month, followed by asset
  growth. Working contributions are posted at month end and use month-end FX.
  Converted contribution totals use that same FX. Today's-money charts divide
  the resulting INR value by Indian inflation once.
- PF/NPS access ages and accessible percentages are user planning inputs, not
  legal withdrawal eligibility. Locked funds cannot finance early retirement;
  only the selected portion releases. The remainder stays unavailable. PF starts
  with an editable access age of 60; NPS starts at 0% accessible until configured.
  Annuity income and future statutory withdrawal changes are not modelled.
- Existing unvested RSUs vest evenly over the selected period while employed.
  Each new annual grant creates another evenly vesting cohort; grant growth and
  after-tax retention are independent inputs. Grant USD value stays fixed before
  vesting, so pre-vest share-price movement is not forecast. Vested value joins
  the employer-stock bucket. All SIP, ESPP, PF/NPS contributions, new grants and
  employment-dependent vesting stop at retirement; remaining unvested awards
  are conservatively forfeited.
- The initial target is future annual spending divided by the withdrawal rate.
  Every candidate retirement month must both meet that target with accessible
  assets and fund all later monthly spending through the chosen horizon without
  new contributions. Thus a target such as annual spending / 3% alone does not
  establish sustainability. The earliest passing month is shown; an infeasible
  scenario stays unretired and displays an accumulation-only path.

Graphs show nominal and today's-money portfolios, with total and accessible
assets distinct. The initial-target line ends at retirement; later values show
withdrawals rather than imposing a new initial target every year. The annual
table includes the exact retirement month, cumulative contributions and the zero
contribution boundary. Invalid or numerically unsupported scenarios return a
controlled error rather than NaN, division-by-zero or null chart points. Canceled
requests stop obsolete calculations and stale results are hidden while updating.

## Verification and limits

The independent [Astra regression review](fire-regression-review.md) records four
closed-form retirement-month comparisons and synthetic probes for FX timing,
locked funds, surplus allocation, numeric limits and unreachable retirement.
Committed engine, database, API and frontend tests cover the same boundaries,
read-only/authenticated API behavior, latest-snapshot grouping, personal shares,
source month gaps, saved input validation and chart conversion.

Final repository verification passed with `CGO_ENABLED=0 go test ./...`,
`go vet ./...`, frontend production build and lint. The frontend suite has 75
tests: 73 passed, zero failed and two existing environment-dependent setup skips.
The portable application ran on port 8089 with a separate fictional ledger.
Browser checks verified source prefills, automatic scenario/chart updates,
PF/NPS/ESPP/RSU inputs, persistence across reload and tab navigation, and discreet
mode hiding amounts and charts. Production data was not accessed or edited.

This is a deterministic plan, not a probability of success. It does not forecast
return sequences, healthcare shocks, tax-law changes, future benefits or live
prices. Change the assumptions to explore risk; do not interpret illustrative
defaults as a historical return or a guaranteed retirement date.
