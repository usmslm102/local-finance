# Independent FIRE regression review

Compared the FIRE worktree against `8308c98`. The review covered deterministic projection math, database inference, the local API, browser persistence and chart behavior. All numerical examples used fictional records. No user database or deployed application was accessed.

## Findings and corrections

| Behavior | Independent evidence |
| --- | --- |
| Under the initial implementation, an unrepresentable target or a projection exceeding the numeric ceiling could return a nominally successful result with missing chart points. | The recheck of a tiny positive withdrawal rate and a growing locked USD portfolio returned controlled errors. A successful response must be finite and carry a usable trajectory. |
| A scenario that never reached retirement still labeled its final horizon point retired. | An accumulation-only one-year scenario now ends with no retirement age and no retired point. |
| INR funded into a USD bucket was converted at the start-of-month rate after market growth had already occurred. | In a zero-return asset with 12% annual FX growth, the original code turned the first ₹8,000 contribution into ₹8,075.91 immediately. The corrected month-end booking reports ₹8,000 and begins both market and currency exposure after investment. |
| With positive net salary surplus and no regular market contributions, the initial fallback could drop contributions if every market asset was locked. | A year of ₹1,000 monthly take-home and ₹100 living costs now adds ₹10,800 to a locked non-payroll bucket, with access still controlled by its release age. |
| The new fallback could invest positive surplus into an initially empty USD bucket with no supplied exchange rate. | A separate API-level scenario with only that USD bucket exercises the distinction between an unused USD placeholder and a destination that needs conversion. |

## Checks

- Compared the earliest modelled retirement against an independent closed-form solution. Each candidate had to clear its inflation-adjusted withdrawal target and the present value of all future beginning-of-month withdrawals. Four cases covered positive, zero and negative asset returns, different inflation assumptions, positive contributions, and an unreachable target. The results agreed to the month. A separate ten-deposit boundary case agreed as well.
- Reviewed the monthly order of access releases, withdrawals, growth, employment contributions and vesting. Committed tests cover the halt in employee and employer contributions, vesting cohorts and grant retention. They also cover locked assets, inaccessible portions, independent INR inflation and native returns, FX, and take-home pay without duplicate PF contributions.
- Ran the targeted database baseline tests on temporary SQLite databases. Those tests cover Splitwise personal shares, transfer and exclusion handling, non-INR ledger entries, investment funding and calendar gaps. They separately verify latest distinct snapshots, multiple accounts, separate INR/USD totals and missing valuations.
- Inspected input validation, chart loading/error branches, optional browser-local scenario storage and source refresh controls. No remote market, payroll or exchange-rate data feed is involved. The main agent handles browser smoke and repository-wide checks separately.

The monthly salary and broker figures are estimates based on imported records. In particular, deposits to an investment service provide funding evidence rather than a current asset value. The retirement horizon test describes this deterministic scenario; it does not estimate a probability of success.

## Final disposition

**Passed after correction. No blocking calculation or accounting finding remains.** The assertion-based overlay now rejects an empty USD surplus destination without a conversion rate. The earliest-month closed-form checks, end-of-month conversion, locked-bucket surplus, unreachable-horizon state, and both numeric-limit cases all passed together on the final working tree.

FIRE now shares the salary evidence predicate with Salary Insights. The database fixtures include a payroll credit outside the salary category. The projection also warns when planned bank-funded contributions plus living costs exceed projected take-home pay during employment. Targeted FIRE database and API tests passed with CGO disabled. These results clear the independent calculation review; repository-wide CI and browser verification are tracked by the implementing agent.

## MCP integration addendum

The final FIRE tools pass the targeted read-only SDK and general read-tool parity tests. The first run found an unused test import; after its removal both tests compiled and passed independently with Go caching bypassed. The baseline invokes the existing database read. The calculator validates the derived scenario schema, checks typed decoding and delegates to the same cancellable projection with no database writer or persistence. A one-item page retained the full scenario's retirement age and retirement point and returned the next trajectory point on page two. No further protocol or access finding remains. The display-only chart helper ends the initial-target line after retirement and leaves projection results untouched.
