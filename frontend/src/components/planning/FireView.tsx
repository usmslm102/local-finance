import { useEffect, useId, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { CartesianGrid, Line, LineChart, ReferenceLine, XAxis, YAxis } from 'recharts'
import { fetchFireBaseline, projectFire } from '@/lib/api'
import { fireAge, fireChartPoints, fireDefaults, fireStorageKey, loadFireScenario } from '@/lib/fire'
import { formatINR } from '@/lib/utils'
import type { FireAsset, FireBaseline, FireScenario } from '@/types/fire'
import { usePrivacy } from '@/components/privacy-provider'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PageLoading, QueryError } from '@/components/ui/query-state'

const compact = (v: number) => new Intl.NumberFormat('en-IN', { notation: 'compact', maximumFractionDigits: 1 }).format(v)

export function FireView() {
  const baseline = useQuery({ queryKey: ['fire-baseline'], queryFn: fetchFireBaseline })
  if (baseline.isPending) return <PageLoading label="Loading retirement planning evidence" />
  if (baseline.isError) return <QueryError message="Unable to load retirement inputs" onRetry={() => void baseline.refetch()} />
  return <FirePlanner baseline={baseline.data} refresh={() => void baseline.refetch()} />
}

function FirePlanner({ baseline, refresh }: { baseline: FireBaseline; refresh: () => void }) {
  const id = useId()
  const { isPrivacyMode } = usePrivacy()
  const [saved] = useState(() => { try { return loadFireScenario(localStorage.getItem(fireStorageKey)) } catch { return null } })
  const [scenario, setScenario] = useState<FireScenario>(() => saved ?? fireDefaults(baseline))
  const [remember, setRemember] = useState(!!saved)
  const [storageError, setStorageError] = useState('')
  const [details, setDetails] = useState(false)
  const [evidence, setEvidence] = useState(false)
  const [debounced, setDebounced] = useState(scenario)
  useEffect(() => { const timer = setTimeout(() => setDebounced(scenario), 400); return () => clearTimeout(timer) }, [scenario])
  useEffect(() => {
    try {
      if (remember) localStorage.setItem(fireStorageKey, JSON.stringify(scenario))
      else localStorage.removeItem(fireStorageKey)
      setStorageError('')
    } catch { setStorageError('This browser could not save your inputs. The current calculation still works.') }
  }, [remember, scenario])
  const result = useQuery({ queryKey: ['fire-projection', debounced], queryFn: ({ signal }) => projectFire(debounced, signal), enabled: debounced.age >= 18 && debounced.monthly_expense > 0 && debounced.retirement_expense > 0, retry: false, gcTime: 60000 })
  const data = result.data
  const money = (v: number) => formatINR(v, isPrivacyMode)
  const set = <K extends keyof FireScenario>(key: K, value: FireScenario[K]) => setScenario(s => ({ ...s, [key]: value }))
  const assetSet = (i: number, key: keyof FireAsset, value: string | number) => setScenario(s => ({ ...s, assets: s.assets.map((a, index) => index === i ? { ...a, [key]: value } : a) }))
  let fieldIndex = 0
  const numeric = (label: string, value: number, change: (v: number) => void, description?: string, sensitive = false) => {
    const fieldID = `${id}-field-${fieldIndex++}`
    return <Field key={label}><FieldLabel htmlFor={fieldID}>{label}</FieldLabel><Input id={fieldID} type={sensitive && isPrivacyMode ? 'password' : 'number'} inputMode="decimal" value={value === 0 ? '' : value} placeholder="0" step="any" onChange={e => change(Number(e.target.value))} />{description && <FieldDescription>{description}</FieldDescription>}</Field>
  }
  const applyEvidence = () => {
    const inferred = fireDefaults(baseline, scenario.usd_inr)
    setScenario(s => ({ ...s, monthly_expense: inferred.monthly_expense, monthly_income: inferred.monthly_income, assets: s.assets.map((a, i) => [0, 1, 3, 6].includes(i) ? { ...a, balance: i === 0 || i === 1 ? inferred.assets[i].balance : a.balance, monthly: inferred.assets[i].monthly } : a) }))
  }
  const charts = [
    { title: 'Portfolio and target · future rupees', description: 'Total includes locked assets; the accessible line shows what can fund withdrawals. Contributions stop at the calculated retirement month. The initial-target line ends there; later portfolio values reflect withdrawals.', real: false },
    { title: 'The same portfolio · today’s purchasing power', description: 'Future values divided by Indian inflation. USD growth and currency effects are converted before this adjustment.', real: true },
  ]
  return <div className="flex flex-col gap-6">
    <div><h2 className="text-xl font-semibold">When could you retire?</h2><p className="mt-1 text-sm text-muted-foreground">Start with recorded spending, review the estimates, then add investments outside your bank statements. This is an editable planning scenario.</p></div>
    <Alert><AlertTitle>Prefilled from {baseline.observed_months} observed months out of six</AlertTitle><AlertDescription><p>Current spending {money(baseline.monthly_expense)}/month excludes identified investment funding and uses your personal Splitwise shares. Salary and broker funding are estimates from recorded activity, not verified complete income or holdings.</p><Link to="/data-health" className="underline">Check missing statements</Link><div className="flex flex-wrap gap-2 mt-2"><Button size="sm" variant="outline" onClick={refresh}>Refresh source data</Button><Button size="sm" variant="outline" onClick={applyEvidence}>Replace ledger-derived inputs</Button><Button size="sm" variant="ghost" aria-expanded={evidence} onClick={() => setEvidence(!evidence)}>Show source evidence</Button></div><p className="text-xs">Replacing inputs updates current expenses, net salary, broker/deposit contributions and imported India/US balances. It keeps future spending, PF/NPS and employer inputs.</p></AlertDescription></Alert>
    <Card><CardHeader><CardTitle>{scenario.age < 18 ? 'Enter your age to start' : result.isError ? 'Review the highlighted scenario inputs' : result.isFetching || debounced !== scenario ? 'Updating your retirement estimate…' : data?.retirement_age != null ? `Modelled retirement: ${fireAge(data.retirement_age)}` : data ? 'Retirement target not reached in this plan' : 'Complete spending inputs to calculate'}</CardTitle><CardDescription>Inputs update the estimate and both graphs automatically. Default returns and growth rates are illustrative assumptions, not inferred investment performance.</CardDescription></CardHeader>{data && !result.isFetching && debounced === scenario && <CardContent className="flex flex-wrap items-center gap-3"><span>Initial target in today’s money: {money(data.today_target)}</span><Button variant="outline" onClick={() => document.getElementById(`${id}-results`)?.scrollIntoView({behavior:'smooth'})}>View projection charts</Button></CardContent>}</Card>
    {evidence && <Card><CardHeader><CardTitle>What the records support</CardTitle><CardDescription>Missing months are unknown. A month with records may still have incomplete statement coverage. Investment funding does not establish returns.</CardDescription></CardHeader><CardContent className="overflow-x-auto"><Table><TableHeader><TableRow>{['Month', 'Personal living spend', 'Salary credits', 'India funding', 'US funding (₹)', 'Deposits', 'Other investing'].map(h => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{baseline.months.map(m => <TableRow key={m.month}><TableCell>{m.month}{!m.observed && ' · missing'}</TableCell>{[m.expense, m.salary, m.india, m.us, m.deposits, m.other].map((v, i) => <TableCell key={i}>{m.observed ? money(v) : 'Unknown'}</TableCell>)}</TableRow>)}</TableBody></Table><div className="mt-4 text-sm space-y-1">{baseline.portfolios.map((p, i) => <p key={i}>{p.provider} · as of {p.as_of} · {p.value == null ? 'Valuation unavailable' : isPrivacyMode ? '••••••' : `${p.currency} ${compact(p.value)}`}</p>)}</div></CardContent></Card>}
    <Card><CardHeader><CardTitle>Your spending and timeline</CardTitle><CardDescription>Age cannot be inferred. Enter future lifestyle spending in today’s rupees; inflation is applied from now, including before retirement.</CardDescription></CardHeader><CardContent className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {numeric('Current age', scenario.age, v => set('age', v), 'Required: 18–80. The retirement month is calculated, not fixed at 40.')}
      {numeric('Plan through age', scenario.horizon, v => set('horizon', v), 'The model must fund spending through this age, up to 105.')}
      {numeric('Current monthly living spend (₹)', scenario.monthly_expense, v => set('monthly_expense', v), 'Excludes identified investments, transfers and excluded transactions. Review inferred classifications.', true)}
      {numeric('Future / retirement monthly spend (today’s ₹)', scenario.retirement_expense, v => set('retirement_expense', v), 'Use your intended lifestyle, dependants and housing costs. Not multiplied automatically.', true)}
      {numeric('Years until lifestyle cost changes', scenario.expense_change_years, v => set('expense_change_years', v), 'While working, current expenses switch to future expenses after this many years. Retirement always uses the future level.')}
      {numeric('Indian inflation (% per year)', scenario.inflation_percent, v => set('inflation_percent', v))}
      {numeric('Initial withdrawal target (%)', scenario.withdrawal_percent, v => set('withdrawal_percent', v), 'Annual spending ÷ this rate sets the initial target. Later spending follows inflation, not a fixed percentage of each year’s balance.')}
      {numeric('Net monthly salary received (₹)', scenario.monthly_income, v => set('monthly_income', v), 'Bank salary after payroll deductions; include bonuses only if repeatable. PF/ESPP are entered separately.', true)}
      {numeric('Annual salary growth (%)', scenario.income_growth_percent, v => set('income_growth_percent', v))}
      {numeric('Annual contribution step-up (%)', scenario.contribution_growth_percent, v => set('contribution_growth_percent', v), 'Applies to monthly SIP, ESPP, PF/NPS and the monthly increases below. RSU grants have their own growth input.')}
      <Field><FieldLabel htmlFor={`${id}-surplus`}>Invest remaining net salary too</FieldLabel><Switch id={`${id}-surplus`} checked={scenario.invest_surplus} onCheckedChange={v => set('invest_surplus', v)} /><FieldDescription>After living costs and planned bank-funded SIPs, add positive surplus using their contribution mix, or current market/deposit portfolio weights when no SIPs exist. With neither, the first market/deposit bucket receives it. Payroll deductions are not counted again.</FieldDescription></Field>
    </CardContent></Card>
    <Card><CardHeader><CardTitle>Assets and monthly contributions</CardTitle><CardDescription>Balance uses the asset currency; every monthly contribution and monthly increase below is in INR. Enter current balances once, excluding your home and emergency fund. Do not add transfers as balances.</CardDescription></CardHeader><CardContent className="space-y-5">
      {scenario.assets.map((a, i) => <div key={i} className="border rounded-lg p-4 space-y-3"><div className="flex items-center justify-between"><h3 className="font-medium">{a.name} · {a.currency}</h3>{i >= 7 && <Button variant="ghost" size="sm" onClick={() => set('assets', scenario.assets.filter((_, index) => index !== i))}>Remove {a.name}</Button>}</div>
        {i >= 7 && <Field><FieldLabel htmlFor={`${id}-name-${i}`}>Asset name</FieldLabel><Input id={`${id}-name-${i}`} value={a.name} maxLength={100} onChange={e => assetSet(i, 'name', e.target.value)} /></Field>}
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {numeric(`${a.name} balance (${a.currency})`, a.balance, v => assetSet(i, 'balance', v), a.kind === 'EMPLOYER' ? 'Vested shares only. Unvested RSUs go in the section below.' : i < 2 ? 'Latest imported portfolio value when available; verify dated snapshots and avoid duplication.' : 'Not reliably visible in bank records; enter your actual balance.', true)}
          {numeric(`${a.name} monthly contribution (₹)`, a.monthly, v => assetSet(i, 'monthly', v), a.kind === 'EMPLOYER' ? 'Your ESPP deduction in INR, in addition to RSU vesting. Do not duplicate this under US stocks.' : a.kind === 'PF' ? 'Include the employer contribution if you want it modelled.' : undefined, true)}
          {numeric(`${a.name} expected net return (%)`, a.return_percent, v => assetSet(i, 'return_percent', v), `Nominal annual ${a.currency} return after assumed fees/taxes; illustrative, not historical CAGR.`)}
        </div>
        {details && <div className="grid sm:grid-cols-3 gap-4">
          {numeric(`${a.name} monthly increase (₹)`, a.monthly_increase, v => assetSet(i, 'monthly_increase', v), 'Contribution adds this amount each month, then the annual step-up applies.', true)}
          {numeric(`${a.name} planning access age`, a.access_age, v => assetSet(i, 'access_age', v), '0 = available now. Confirm actual withdrawal eligibility independently.')}
          {numeric(`${a.name} accessible portion (%)`, a.accessible_percent, v => assetSet(i, 'accessible_percent', v), a.kind === 'NPS' ? 'Default 0: NPS cannot fund retirement until you choose a realistic accessible lump-sum assumption. Annuity income is not modelled.' : 'Only this portion can fund spending at the access age; the rest stays unavailable.')}
        </div>}
      </div>)}
      <div className="flex flex-wrap gap-2"><Button variant="outline" onClick={() => setDetails(!details)} aria-expanded={details}>{details ? 'Hide' : 'Edit'} access ages and monthly increases</Button><Button variant="outline" disabled={scenario.assets.length >= 12} onClick={() => set('assets', [...scenario.assets, { kind: 'OTHER', name: `Additional asset ${scenario.assets.length - 6}`, currency: 'INR', balance: 0, monthly: 0, monthly_increase: 0, return_percent: 5, access_age: 0, accessible_percent: 100 }])}>Add another asset</Button></div>
      <p className="text-xs text-muted-foreground">PF defaults to planning access at 60. NPS defaults to 0% accessible. These are conservative placeholders, not withdrawal-law or annuity assumptions. Open access settings to configure your plan.</p>
    </CardContent></Card>
    <Card><CardHeader><CardTitle>US conversion and RSUs outside bank statements</CardTitle><CardDescription>Enter an INR-per-USD rate if using US assets. No live exchange rate is fetched. Employer grants are not salary receipts or starting wealth.</CardDescription></CardHeader><CardContent className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {numeric('INR per USD', scenario.usd_inr, v => set('usd_inr', v), 'Required for USD balances, US contributions or grants. Monthly INR funding converts at the projected rate.')}
      {numeric('Annual INR depreciation vs USD (%)', scenario.fx_growth_percent, v => set('fx_growth_percent', v), '0% assumes a constant rate. USD nominal return and FX growth multiply; inflation adjusts INR spending separately.')}
      {numeric('Existing unvested RSUs (USD)', scenario.rsu_unvested_usd, v => set('rsu_unvested_usd', v), 'Excluded from starting portfolio. Assumed to vest evenly over the next vesting period while employed.', true)}
      {numeric('New annual RSU grant (USD)', scenario.rsu_annual_usd, v => set('rsu_annual_usd', v), 'A new grant each year while working; each cohort vests evenly over the selected years.', true)}
      {numeric('RSU vesting period (years)', scenario.rsu_vest_years, v => set('rsu_vest_years', v), '1–10 years. This smooth schedule is an assumption, not your employer’s actual vesting calendar.')}
      {numeric('Annual new-grant growth (%)', scenario.rsu_grant_growth_percent, v => set('rsu_grant_growth_percent', v))}
      {numeric('RSU value retained after taxes (%)', scenario.rsu_net_percent, v => set('rsu_net_percent', v), 'Illustrative default 70%; replace with your after-tax estimate. Unvested grant value is held constant until vesting; share-price changes before vesting are not forecast.')}
    </CardContent></Card>
    <div className="flex items-center gap-3"><Switch id={`${id}-remember`} checked={remember} onCheckedChange={setRemember} /><FieldLabel htmlFor={`${id}-remember`}>Remember these inputs in this browser</FieldLabel></div>
    {storageError && <p role="alert">{storageError}</p>}
    {scenario.age < 18 || scenario.monthly_expense <= 0 || scenario.retirement_expense <= 0 ? <Alert><AlertTitle>Complete your starting inputs</AlertTitle><AlertDescription>Enter your age and positive current/future spending to calculate a retirement month.</AlertDescription></Alert> : result.isError ? <Alert variant="destructive"><AlertTitle>Check your scenario</AlertTitle><AlertDescription>{result.error.message}</AlertDescription></Alert> : result.isPending || result.isFetching || debounced !== scenario ? <p role="status">Recalculating your retirement scenario…</p> : data && <>
      <div id={`${id}-results`} className="scroll-mt-20 grid sm:grid-cols-2 lg:grid-cols-4 gap-4">{[
        ['Earliest modelled retirement', data.retirement_age == null ? 'Not reached in this plan' : fireAge(data.retirement_age)],
        ['Initial FIRE target · today’s money', money(data.today_target)],
        ['Portfolio at retirement · future ₹', data.at_retirement ? money(data.at_retirement.portfolio) : 'Not reached'],
        ['Portfolio at retirement · today’s ₹', data.at_retirement ? money(data.at_retirement.today_value) : 'Not reached'],
      ].map(([label, value]) => <Card key={label}><CardHeader><CardDescription>{label}</CardDescription><CardTitle className="text-xl">{value}</CardTitle></CardHeader></Card>)}</div>
      <p className="text-sm text-muted-foreground">The calculated month must satisfy the initial withdrawal target using accessible assets and fund every subsequent monthly expense through age {scenario.horizon}. A target alone does not prove sustainability. Contributions, payroll investments and employment-dependent vesting stop at that month.</p>
      {data.at_retirement && <p className="text-sm">Accessible at retirement: {money(data.at_retirement.accessible)} · total contributed after today: {money(data.at_retirement.invested)} · first retirement month spending: {money(data.at_retirement.monthly_expense)}.</p>}
      {charts.map(chart => <Card key={chart.title}><CardHeader><CardTitle>{chart.title}</CardTitle><CardDescription>{chart.description}</CardDescription></CardHeader><CardContent>{isPrivacyMode ? <p>Chart hidden in discreet mode.</p> : <ChartContainer className="h-72 w-full" config={{ portfolio: { label: 'Total portfolio', color: 'var(--primary)' }, accessible: { label: 'Accessible portfolio', color: '#3b82f6' }, target: { label: 'Initial withdrawal target', color: '#f59e0b' } }}><LineChart data={fireChartPoints(data, chart.real)}><CartesianGrid vertical={false} /><XAxis dataKey="age" type="number" domain={['dataMin', 'dataMax']} tickFormatter={v => String(Math.floor(v))} /><YAxis tickFormatter={compact} width={70} /><ChartTooltip content={<ChartTooltipContent formatter={value => formatINR(Number(value))} />} /><ChartLegend content={<ChartLegendContent />} /><Line dataKey="portfolio" type="monotone" stroke="var(--color-portfolio)" dot={false} /><Line dataKey="accessible" type="monotone" stroke="var(--color-accessible)" dot={false} /><Line dataKey="target" type="monotone" stroke="var(--color-target)" strokeDasharray="4 4" dot={false} />{data.retirement_age != null && <ReferenceLine x={data.retirement_age} label="Retirement" />}</LineChart></ChartContainer>}</CardContent></Card>)}
      <Card><CardHeader><CardTitle>Contributions and spending through the plan</CardTitle><CardDescription>All amounts below are nominal INR. Monthly contribution is zero from retirement. Cumulative contributions exclude your starting balances.</CardDescription></CardHeader><CardContent className="overflow-x-auto"><Table><TableHeader><TableRow>{['Age', 'Monthly investing', 'Contributed since today', 'Portfolio', 'Today’s purchasing power', 'Monthly spending'].map(h => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{data.points.map(p => <TableRow key={p.age}><TableCell>{p.age.toFixed(1)}{p.retired && ' · retired'}</TableCell>{[p.monthly_investment, p.invested, p.portfolio, p.today_value, p.monthly_expense].map((v, i) => <TableCell key={i}>{money(v)}</TableCell>)}</TableRow>)}</TableBody></Table></CardContent></Card>
      <Alert><AlertTitle>Assumptions and limits</AlertTitle><AlertDescription><ul className="list-disc pl-4 space-y-2">{[...baseline.warnings, ...data.warnings].map(w => <li key={w}>{w}</li>)}</ul></AlertDescription></Alert>
    </>}
  </div>
}
