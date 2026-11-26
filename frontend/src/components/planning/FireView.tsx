import { useEffect, useId, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { fetchFireBaseline, projectFire } from '@/lib/api'
import { fireDefaults, fireStorageKey, loadFireScenario } from '@/lib/fire'
import { formatINR } from '@/lib/utils'
import type { FireAsset, FireBaseline, FireScenario } from '@/types/fire'
import { usePrivacy } from '@/components/privacy-provider'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Flame, SlidersHorizontal, Database, ChevronDown, ArrowUpRight } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { FireResults } from './FireResults'
import { fireInputNeeds, fireNeedsFX } from '@/lib/fire-display'
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
  const [panel, setPanel] = useState('spending')
  const [expandedAsset, setExpandedAsset] = useState<number | null>(null)
  const [limitsOpen, setLimitsOpen] = useState(false)
  const [basis, setBasis] = useState<'today' | 'future'>('today')
  const [debounced, setDebounced] = useState(scenario)
  useEffect(() => { const timer = setTimeout(() => setDebounced(scenario), 400); return () => clearTimeout(timer) }, [scenario])
  useEffect(() => {
    try {
      if (remember) localStorage.setItem(fireStorageKey, JSON.stringify(scenario))
      else localStorage.removeItem(fireStorageKey)
      setStorageError('')
    } catch { setStorageError('This browser could not save your inputs. The current calculation still works.') }
  }, [remember, scenario])
  const needs = fireInputNeeds(scenario)
  const result = useQuery({ queryKey: ['fire-projection', debounced], queryFn: ({ signal }) => projectFire(debounced, signal), enabled: fireInputNeeds(debounced).length === 0, retry: false, gcTime: 60000 })
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

  const latest = data && !result.isError && !result.isFetching && !result.isPending && debounced === scenario && needs.length === 0
  const reviewSources = () => {setPanel('evidence'); requestAnimationFrame(() => {const heading = document.getElementById(id + '-source-heading'); heading?.focus(); heading?.scrollIntoView({block:'center'})})}
  return <div className="mx-auto flex w-full max-w-[1480px] flex-col gap-5">
    <header className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex items-center gap-3"><div className="rounded-2xl border border-emerald-500/20 bg-emerald-500/10 p-3"><Flame className="size-6 text-emerald-500" /></div><div><p className="text-xs font-medium uppercase tracking-[0.2em] text-muted-foreground">Your next chapter</p><h1 className="mt-1 text-2xl font-semibold tracking-tight sm:text-3xl">Retirement & FIRE</h1><p className="mt-1 text-sm text-muted-foreground">Turn today’s habits into a clearer view of life after work.</p></div></div>
      <Button variant="outline" onClick={reviewSources}><Database className="size-4" /> Review source data <ArrowUpRight className="size-3.5" /></Button>
    </header>

    <Card className="border-dashed">
      <CardContent className="p-4 sm:p-5">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2"><div className="flex items-center gap-2 text-sm font-medium"><SlidersHorizontal className="size-4 text-primary" /> Start with your lifestyle</div><Badge variant="outline">{baseline.observed_months}/6 months with recorded activity</Badge></div>
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {numeric('Current age', scenario.age, v => set('age', v), 'Required. Your retirement month is calculated, not chosen.')}
          {numeric('Current monthly living spend (₹)', scenario.monthly_expense, v => set('monthly_expense', v), 'Personal expenses, excluding identified investment funding.', true)}
          {numeric('Retirement monthly spend (today’s ₹)', scenario.retirement_expense, v => set('retirement_expense', v), 'The lifestyle you want, before future inflation.', true)}
          {fireNeedsFX(scenario) ? numeric('INR per USD', scenario.usd_inr, v => set('usd_inr', v), 'Required for your USD assets, funding or grants. No live rate is fetched.') : <div className="flex flex-col justify-center rounded-lg bg-muted/50 p-3"><p className="text-xs text-muted-foreground">Recorded personal spending estimate</p><p className="mt-1 font-semibold">{money(baseline.monthly_expense)}/month</p><p className="mt-1 text-xs text-muted-foreground">Editable · statement coverage may have gaps.</p></div>}
        </div>
        <p className="mt-3 text-xs text-muted-foreground">Recorded activity is not proof of complete coverage. Monthly investments, off-ledger assets and all return assumptions can be edited below.</p>
      </CardContent>
    </Card>

    <div aria-live="polite" aria-atomic="true" className="sr-only">{needs.length ? 'Complete your starting inputs.' : result.isError ? 'Check your scenario inputs.' : !latest ? 'Updating retirement projection.' : data.retirement_age == null ? 'Retirement target not reached under these assumptions.' : 'Retirement projection updated.'}</div>
    {needs.length > 0 ? <Card className="overflow-hidden bg-gradient-to-br from-emerald-500/10 via-card to-sky-500/10"><CardContent className="flex min-h-52 flex-col items-center justify-center gap-3 p-6 text-center"><div className="rounded-full bg-primary/10 p-3"><Flame className="size-7 text-primary" /></div><h2 className="text-xl font-semibold">Let’s find your retirement horizon</h2><p className="max-w-md text-sm text-muted-foreground">Add {needs.join(', ')} above. Your portfolio journey and estimated retirement age will appear here.</p><p className="text-xs text-muted-foreground">No fixed retirement age. No invented probability of success.</p></CardContent></Card> : result.isError ? <Alert variant="destructive"><AlertTitle>Check your scenario</AlertTitle><AlertDescription>{result.error.message}</AlertDescription></Alert> : !latest ? <Card><CardContent className="flex min-h-52 items-center justify-center gap-3 p-6" role="status"><span className="size-4 rounded-full border-2 border-muted border-t-primary motion-safe:animate-spin" /> Updating your retirement projection…</CardContent></Card> : <FireResults data={data} scenario={scenario} privateMode={isPrivacyMode} basis={basis} onBasisChange={setBasis} />}

    <section id={id + '-assumptions'} className="scroll-mt-20 space-y-3" aria-labelledby={id + '-edit-title'}>
      <div className="flex flex-wrap items-center justify-between gap-3"><div><h2 id={id + '-edit-title'} className="text-lg font-semibold">Make the plan yours</h2><p className="text-sm text-muted-foreground">Change a few assumptions at a time. The projection updates automatically.</p></div><div className="flex items-center gap-2"><Switch id={id + '-remember'} checked={remember} onCheckedChange={setRemember} /><FieldLabel htmlFor={id + '-remember'} className="text-xs">Remember inputs in this browser</FieldLabel></div></div>
      {storageError && <p role="alert">{storageError}</p>}
      <Tabs value={panel} onValueChange={setPanel}>
        <TabsList className="h-auto w-full flex-wrap justify-start gap-1 sm:w-auto">
          <TabsTrigger value="spending">Lifestyle & growth</TabsTrigger><TabsTrigger value="investments">Investments</TabsTrigger><TabsTrigger value="employer">Employer & USD</TabsTrigger><TabsTrigger value="evidence">Source evidence</TabsTrigger>
        </TabsList>
        <TabsContent value="spending" className="mt-4">
    <Card><CardHeader><CardTitle>Lifestyle, salary & assumptions</CardTitle><CardDescription>Fine-tune your lifestyle change, growth and withdrawal assumptions. Future spending is in today’s rupees; inflation applies from now.</CardDescription></CardHeader><CardContent className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {numeric('Plan through age', scenario.horizon, v => set('horizon', v), 'The model must fund spending through this age, up to 105.')}
      {numeric('Years until lifestyle cost changes', scenario.expense_change_years, v => set('expense_change_years', v), 'While working, current expenses switch to future expenses after this many years. Retirement always uses the future level.')}
      {numeric('Indian inflation (% per year)', scenario.inflation_percent, v => set('inflation_percent', v))}
      {numeric('Initial withdrawal target (%)', scenario.withdrawal_percent, v => set('withdrawal_percent', v), 'Annual spending ÷ this rate sets the initial target. Later spending follows inflation, not a fixed percentage of each year’s balance.')}
      {numeric('Net monthly salary received (₹)', scenario.monthly_income, v => set('monthly_income', v), 'Bank salary after payroll deductions; include bonuses only if repeatable. PF/ESPP are entered separately.', true)}
      {numeric('Annual salary growth (%)', scenario.income_growth_percent, v => set('income_growth_percent', v))}
      {numeric('Annual contribution step-up (%)', scenario.contribution_growth_percent, v => set('contribution_growth_percent', v), 'Applies to monthly SIP, ESPP, PF/NPS and the monthly increases below. RSU grants have their own growth input.')}
      <Field><FieldLabel htmlFor={`${id}-surplus`}>Invest remaining net salary too</FieldLabel><Switch id={`${id}-surplus`} checked={scenario.invest_surplus} onCheckedChange={v => set('invest_surplus', v)} /><FieldDescription>After living costs and planned bank-funded SIPs, add positive surplus using their contribution mix, or current market/deposit portfolio weights when no SIPs exist. With neither, the first market/deposit bucket receives it. Payroll deductions are not counted again.</FieldDescription></Field>
    </CardContent></Card>

        </TabsContent>
        <TabsContent value="investments" className="mt-4">
          <Card><CardHeader><CardTitle>Assets & contributions</CardTitle><CardDescription>Expand a bucket to edit it. Balances use each asset’s currency; monthly funding and increases are always INR. Keep your home and emergency fund separate.</CardDescription></CardHeader><CardContent className="space-y-2">
            {scenario.assets.map((a, i) => <div key={i} className="overflow-hidden rounded-xl border">
              <Button variant="ghost" className="h-auto w-full justify-between rounded-none px-4 py-3 text-left" aria-expanded={expandedAsset === i} aria-controls={id + '-asset-' + i} onClick={() => setExpandedAsset(expandedAsset === i ? null : i)}>
                <span className="min-w-0"><span className="block truncate font-medium">{a.name} <span className="text-xs font-normal text-muted-foreground">· {a.currency}</span></span><span className="mt-1 block text-xs font-normal text-muted-foreground">{isPrivacyMode ? 'Amounts hidden' : (a.currency === 'INR' ? money(a.balance) : 'USD ' + compact(a.balance)) + ' balance · ' + money(a.monthly) + '/month'}</span></span>
                <ChevronDown className={'ml-2 size-4 shrink-0 motion-safe:transition-transform ' + (expandedAsset === i ? 'rotate-180' : '')} />
              </Button>
              {expandedAsset === i && <div id={id + '-asset-' + i} className="space-y-4 border-t bg-muted/10 p-4">
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

                <div className="flex flex-wrap justify-between gap-2"><Button variant="outline" size="sm" aria-expanded={details} onClick={() => setDetails(!details)}>{details ? 'Hide' : 'Edit'} access ages & monthly increases</Button>{i >= 7 && <Button variant="ghost" size="sm" onClick={() => {set('assets', scenario.assets.filter((_, index) => index !== i)); setExpandedAsset(null)}}>Remove {a.name}</Button>}</div>
              </div>}
            </div>)}
            <Button className="mt-3" variant="outline" disabled={scenario.assets.length >= 12} onClick={() => {set('assets', [...scenario.assets, { kind: 'OTHER', name: 'Additional asset ' + (scenario.assets.length - 6), currency: 'INR', balance: 0, monthly: 0, monthly_increase: 0, return_percent: 5, access_age: 0, accessible_percent: 100 }]); setExpandedAsset(scenario.assets.length)}}>Add another asset</Button>
            <p className="pt-2 text-xs text-muted-foreground">PF defaults to planning access at 60. NPS defaults to 0% accessible until configured. These are editable planning assumptions, not withdrawal-law claims. Employer monthly funding is your ESPP deduction; do not duplicate it under US stocks.</p>
          </CardContent></Card>
        </TabsContent>
        <TabsContent value="employer" className="mt-4">
    <Card><CardHeader><CardTitle>US conversion and RSUs outside bank statements</CardTitle><CardDescription>Enter an INR-per-USD rate if using US assets. No live exchange rate is fetched. Employer grants are not salary receipts or starting wealth.</CardDescription></CardHeader><CardContent className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {!fireNeedsFX(scenario) && numeric('INR per USD', scenario.usd_inr, v => set('usd_inr', v), 'Required for USD balances, US contributions or grants. Monthly INR funding converts at the projected rate.')}
      {numeric('Annual INR depreciation vs USD (%)', scenario.fx_growth_percent, v => set('fx_growth_percent', v), '0% assumes a constant rate. USD nominal return and FX growth multiply; inflation adjusts INR spending separately.')}
      {numeric('Existing unvested RSUs (USD)', scenario.rsu_unvested_usd, v => set('rsu_unvested_usd', v), 'Excluded from starting portfolio. Assumed to vest evenly over the next vesting period while employed.', true)}
      {numeric('New annual RSU grant (USD)', scenario.rsu_annual_usd, v => set('rsu_annual_usd', v), 'A new grant each year while working; each cohort vests evenly over the selected years.', true)}
      {numeric('RSU vesting period (years)', scenario.rsu_vest_years, v => set('rsu_vest_years', v), '1–10 years. This smooth schedule is an assumption, not your employer’s actual vesting calendar.')}
      {numeric('Annual new-grant growth (%)', scenario.rsu_grant_growth_percent, v => set('rsu_grant_growth_percent', v))}
      {numeric('RSU value retained after taxes (%)', scenario.rsu_net_percent, v => set('rsu_net_percent', v), 'Illustrative default 70%; replace with your after-tax estimate. Unvested grant value is held constant until vesting; share-price changes before vesting are not forecast.')}
    </CardContent></Card>

        </TabsContent>
        <TabsContent value="evidence" className="mt-4 space-y-4"><h3 id={id + '-source-heading'} tabIndex={-1} className="rounded-sm text-lg font-medium focus-visible:outline-2 focus-visible:outline-primary">Source evidence</h3>
          <Alert><AlertTitle>Recorded evidence, editable estimates</AlertTitle><AlertDescription><p>Activity can have statement gaps. Investment funding is not a portfolio valuation or historical return.</p><Link to="/data-health" className="underline">Check missing statements</Link><div className="mt-3 flex flex-wrap gap-2"><Button size="sm" variant="outline" onClick={refresh}>Refresh source data</Button><Button size="sm" variant="outline" onClick={applyEvidence}>Replace ledger-derived inputs</Button></div><p className="text-xs">Replacing inputs updates current expenses, net salary, broker/deposit contributions and imported India/US balances. It keeps future spending, PF/NPS, employer inputs and manually added assets.</p></AlertDescription></Alert>
<Card><CardHeader><CardTitle>What the records support</CardTitle><CardDescription>Missing months are unknown. A month with records may still have incomplete statement coverage. Investment funding does not establish returns.</CardDescription></CardHeader><CardContent className="overflow-x-auto"><Table><TableHeader><TableRow>{['Month', 'Personal living spend', 'Salary credits', 'India funding', 'US funding (₹)', 'Deposits', 'Other investing'].map(h => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{baseline.months.map(m => <TableRow key={m.month}><TableCell>{m.month}{!m.observed && ' · missing'}</TableCell>{[m.expense, m.salary, m.india, m.us, m.deposits, m.other].map((v, i) => <TableCell key={i}>{m.observed ? money(v) : 'Unknown'}</TableCell>)}</TableRow>)}</TableBody></Table><div className="mt-4 text-sm space-y-1">{baseline.portfolios.map((p, i) => <p key={i}>{p.provider} · as of {p.as_of} · {p.value == null ? 'Valuation unavailable' : isPrivacyMode ? '••••••' : `${p.currency} ${compact(p.value)}`}</p>)}</div></CardContent></Card>
        </TabsContent>
      </Tabs>
    </section>

    <section className="space-y-3">
      <Button variant="ghost" className="w-full justify-between" aria-expanded={limitsOpen} aria-controls={id + '-limits'} onClick={() => setLimitsOpen(!limitsOpen)}>Assumptions, data coverage & model limits <ChevronDown className="size-4" /></Button>
      {limitsOpen && <Alert id={id + '-limits'}><AlertTitle>Know what this plan assumes</AlertTitle><AlertDescription><ul className="list-disc space-y-2 pl-4">{[...baseline.warnings, ...(latest ? data.warnings : [])].map(w => <li key={w}>{w}</li>)}</ul></AlertDescription></Alert>}
      <FireYearly data={latest ? data : undefined} privateMode={isPrivacyMode} />
    </section>
  </div>
}

function FireYearly({data, privateMode}: {data?: import('@/types/fire').FireResult; privateMode: boolean}) {
  const [open, setOpen] = useState(false)
  const id = useId()
  const money = (v: number) => formatINR(v, privateMode)
  return <div className="space-y-3"><Button variant="ghost" className="w-full justify-between" aria-expanded={open} aria-controls={id} onClick={() => setOpen(!open)}>Year-by-year contributions & spending <ChevronDown className="size-4" /></Button>{open && <div id={id}>{data ? <>
      <Card><CardHeader><CardTitle>Contributions and spending through the plan</CardTitle><CardDescription>All amounts below are nominal INR. Monthly contribution is zero from retirement. Cumulative contributions exclude your starting balances.</CardDescription></CardHeader><CardContent className="overflow-x-auto"><Table><TableHeader><TableRow>{['Age', 'Monthly investing', 'Contributed since today', 'Portfolio', 'Today’s purchasing power', 'Monthly spending'].map(h => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{data.points.map(p => <TableRow key={p.age}><TableCell>{p.age.toFixed(1)}{p.retired && ' · retired'}</TableCell>{[p.monthly_investment, p.invested, p.portfolio, p.today_value, p.monthly_expense].map((v, i) => <TableCell key={i}>{money(v)}</TableCell>)}</TableRow>)}</TableBody></Table></CardContent></Card>

  </> : <p className="p-4 text-sm text-muted-foreground">Complete or correct the inputs above to see current yearly figures.</p>}</div>}</div>
}
