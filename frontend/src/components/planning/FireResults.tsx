import { useId } from 'react'
import { Area, CartesianGrid, Cell, ComposedChart, Line, Pie, PieChart, ReferenceArea, ReferenceLine, XAxis, YAxis } from 'recharts'
import { ArrowRight, Flag, LockKeyhole, Sparkles, Sunrise, TrendingUp, Wallet } from 'lucide-react'
import { fireAge, fireChartPoints } from '@/lib/fire'
import { fireAssetBreakdown, fireStartingValues, fireTargetProgress } from '@/lib/fire-display'
import { formatINR } from '@/lib/utils'
import type { FireResult, FireScenario } from '@/types/fire'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

const compact = (v: number) => new Intl.NumberFormat('en-IN', { notation: 'compact', maximumFractionDigits: 1 }).format(v)
const labels: Record<string, string> = {portfolio: 'Total portfolio', accessible: 'Accessible portfolio', target: 'Initial spending target'}
const config = {portfolio: {label: labels.portfolio, color: '#10b981'}, accessible: {label: labels.accessible, color: '#38bdf8'}, target: {label: labels.target, color: '#f59e0b'}}

export function FireResults({ data, scenario, privateMode, basis, onBasisChange }: { data: FireResult; scenario: FireScenario; privateMode: boolean; basis: 'today' | 'future'; onBasisChange: (value: 'today' | 'future') => void }) {
  const id = useId().replace(/:/g, '')
  const real = basis === 'today'
  const money = (value: number) => formatINR(value, privateMode)
  const starting = fireStartingValues(scenario)
  const progress = fireTargetProgress(starting?.accessible ?? 0, data.today_target)
  const breakdown = fireAssetBreakdown(data, real)
  const warning = data.warnings.find(w => w.startsWith('Planned bank-funded'))
  const years = data.retirement_age == null ? null : data.retirement_age - scenario.age
  return <div className="space-y-5" aria-label="Retirement projection">
    <section className="relative overflow-hidden rounded-2xl border bg-gradient-to-br from-emerald-500/15 via-card to-sky-500/10 p-5 sm:p-7">
      <div className="pointer-events-none absolute -right-20 -top-24 h-64 w-64 rounded-full border border-emerald-400/15" aria-hidden="true" />
      <div className="relative grid gap-6 lg:grid-cols-[1.25fr_1fr]">
        <div className="space-y-4">
          <Badge variant="outline" className="gap-1.5 bg-background/60"><Sparkles className="size-3.5" /> Your scenario · through age {scenario.horizon}</Badge>
          <div>
            <p className="text-sm text-muted-foreground">Earliest modelled retirement</p>
            <h2 className="mt-2 text-3xl font-semibold tracking-tight sm:text-5xl">{data.retirement_age == null ? 'Not reached in this plan' : fireAge(data.retirement_age)}</h2>
            <p className="mt-3 max-w-xl text-sm text-muted-foreground">{years == null ? 'Try changing contributions, spending or assumptions. An unreachable plan is not an achieved retirement date.' : years === 0 ? 'The starting assets meet this scenario’s target and horizon checks.' : `About ${fireAge(years)} from your starting age. Contributions stop here; the model then funds your lifestyle through age ${scenario.horizon}.`}</p>
          </div>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-xs text-muted-foreground" aria-label="Plan phases">
            <span className="flex items-center gap-1.5"><TrendingUp className="size-4 text-emerald-500" /> Build · age {scenario.age}</span>
            <ArrowRight className="size-3.5" aria-hidden="true" />
            <span className="flex items-center gap-1.5"><Flag className="size-4 text-sky-500" /> {data.retirement_age == null ? 'Explore your target' : `Retire · ${fireAge(data.retirement_age)}`}</span>
            <ArrowRight className="size-3.5" aria-hidden="true" />
            <span className="flex items-center gap-1.5"><Sunrise className="size-4 text-amber-500" /> Plan through {scenario.horizon}</span>
          </div>
        </div>
        <div className="rounded-xl border bg-background/70 p-4 sm:p-5">
          <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Initial target · today’s purchasing power</p>
          <p className="mt-2 text-2xl font-semibold sm:text-3xl">{money(data.today_target)}</p>
          <p className="mt-1 text-xs text-muted-foreground">{money(scenario.retirement_expense)}/month at a {scenario.withdrawal_percent}% initial withdrawal target.</p>
          <div className="mt-5 flex justify-between gap-3 text-xs"><span>Starting accessible assets</span><span>{starting ? money(starting.accessible) : 'Conversion needed'}</span></div>
          {privateMode ? <p className="mt-2 text-xs text-muted-foreground">Progress graphic hidden in discreet mode.</p> : <>
            <div className="mt-2 h-2 overflow-hidden rounded-full bg-muted" role="progressbar" aria-label="Starting accessible assets as a portion of initial target" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.floor(progress.meter)} aria-valuetext={progress.label + ' of initial target'}><div className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-sky-400 motion-safe:transition-[width]" style={{width: `${progress.meter}%`}} /></div>
            <p className="mt-2 text-xs text-muted-foreground">{progress.label} of the initial target today · not a probability of success.</p>
          </>}
        </div>
      </div>
    </section>

    {warning && <Alert><AlertTitle>Check the funding behind your plan</AlertTitle><AlertDescription>{warning}</AlertDescription></Alert>}

    <div className="grid gap-3 sm:grid-cols-3">
      {[
        {label: 'At retirement · future rupees', value: data.at_retirement ? money(data.at_retirement.portfolio) : 'Not reached', icon: Wallet},
        {label: 'At retirement · today’s rupees', value: data.at_retirement ? money(data.at_retirement.today_value) : 'Not reached', icon: Sparkles},
        {label: 'Contributed after today', value: data.at_retirement ? money(data.at_retirement.invested) : 'See yearly figures', icon: TrendingUp},
      ].map(({label, value, icon: Icon}) => <Card key={label}><CardContent className="flex items-start gap-3 p-4"><div className="rounded-lg bg-primary/10 p-2"><Icon className="size-4 text-primary" /></div><div className="min-w-0"><p className="text-xs text-muted-foreground">{label}</p><p className="mt-1 text-lg font-semibold break-words">{value}</p></div></CardContent></Card>)}
    </div>

    <div className="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,2.3fr)_minmax(240px,1fr)]">
      <Card className="min-w-0 overflow-hidden">
        <CardHeader className="gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3"><CardTitle>Your portfolio journey</CardTitle><ToggleGroup value={[basis]} onValueChange={values => {const value = values[0]; if (value === 'today' || value === 'future') onBasisChange(value)}} aria-label="Projection money basis"><ToggleGroupItem value="today">Today’s ₹</ToggleGroupItem><ToggleGroupItem value="future">Future ₹</ToggleGroupItem></ToggleGroup></div>
          <CardDescription>{real ? 'Inflation-adjusted purchasing power, in today’s INR.' : 'Nominal INR at each future age.'} {data.retirement_age == null ? 'No retirement month qualifies; this path keeps working and contributing through the horizon.' : 'Contributions stop at the retirement marker.'} Total assets include funds that may still be locked.</CardDescription>
        </CardHeader>
        <CardContent className="px-2 pb-4 sm:px-4">
          {privateMode ? <div className="flex h-72 items-center justify-center gap-2 text-sm text-muted-foreground"><LockKeyhole className="size-4" /> Projection hidden in discreet mode.</div> : <ChartContainer className="h-80 w-full sm:h-96" config={config}>
            <ComposedChart data={fireChartPoints(data, real)} margin={{top: 18, right: 14, bottom: 0, left: 0}} accessibilityLayer>
              <defs><linearGradient id={`${id}-wealth`} x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor="#10b981" stopOpacity={0.25} /><stop offset="100%" stopColor="#10b981" stopOpacity={0.01} /></linearGradient></defs>
              <CartesianGrid vertical={false} strokeDasharray="3 5" />
              <XAxis dataKey="age" type="number" domain={['dataMin', 'dataMax']} tickFormatter={v => String(Math.floor(v))} tickLine={false} axisLine={false} minTickGap={28} />
              <YAxis tickFormatter={compact} width={62} tickLine={false} axisLine={false} />
              <ChartTooltip content={<ChartTooltipContent labelFormatter={(_label, payload) => {const age = Number(payload[0]?.payload?.age); return Number.isFinite(age) ? `Age ${fireAge(age)}` : 'Portfolio projection'}} formatter={(value, name) => <div className="flex w-full min-w-48 items-center justify-between gap-4"><span className="text-muted-foreground">{labels[String(name)] ?? String(name)}</span><span className="font-mono font-medium tabular-nums">{money(Number(value))}</span></div>} />} />
              <ChartLegend content={<ChartLegendContent />} />
              {data.retirement_age != null && <ReferenceArea x1={data.retirement_age} x2={scenario.horizon} fill="#38bdf8" fillOpacity={0.035} />}
              <Area dataKey="portfolio" type="monotone" stroke="#10b981" strokeWidth={2} fill={`url(#${id}-wealth)`} isAnimationActive={false} />
              <Line dataKey="accessible" type="monotone" stroke="#38bdf8" strokeWidth={2} dot={false} isAnimationActive={false} />
              <Line dataKey="target" type="monotone" stroke="#f59e0b" strokeDasharray="5 5" dot={false} isAnimationActive={false} />
              {data.retirement_age != null && <ReferenceLine x={data.retirement_age} stroke="#38bdf8" strokeDasharray="3 4" label={{value:'Retirement', position:'insideTopRight', fill:'var(--muted-foreground)'}} />}
            </ComposedChart>
          </ChartContainer>}
          <p className="mt-2 px-2 text-xs text-muted-foreground">{data.retirement_age == null ? 'The dashed line is the initial spending target at each potential retirement age. No retirement or withdrawal phase is shown.' : 'The dashed initial-target line ends at retirement. Later values reflect withdrawals.'} This is one deterministic scenario.</p>
        </CardContent>
      </Card>
      <Card className="min-w-0">
        <CardHeader><CardTitle>Accessible vs locked</CardTitle><CardDescription>{breakdown?.atRetirement ? 'At the modelled retirement month' : 'At your starting point'} · {real ? 'today’s INR' : 'future INR'}</CardDescription></CardHeader>
        <CardContent>
          {privateMode ? <div className="flex h-44 items-center justify-center text-sm text-muted-foreground">Asset graphic hidden in discreet mode.</div> : breakdown && breakdown.total > 0 ? <ChartContainer className="mx-auto h-48 w-full" config={{accessible:{label:'Accessible',color:'#38bdf8'},locked:{label:'Locked / unavailable',color:'#f59e0b'}}}><PieChart><ChartTooltip content={<ChartTooltipContent formatter={(value, name) => <span>{String(name)}: {money(Number(value))}</span>} />} /><Pie data={[{name:'Accessible',value:breakdown.accessible},{name:'Locked / unavailable',value:breakdown.locked}]} dataKey="value" nameKey="name" innerRadius={52} outerRadius={76} paddingAngle={2} strokeWidth={0} isAnimationActive={false}><Cell fill="#38bdf8" /><Cell fill="#f59e0b" /></Pie></PieChart></ChartContainer> : <p className="py-10 text-center text-sm text-muted-foreground">No starting assets entered yet.</p>}
          {breakdown && <div className="space-y-3 text-sm"><div className="flex justify-between gap-2"><span className="flex items-center gap-2"><span className="size-2 rounded-full bg-sky-400" /> Accessible</span><span className="font-medium">{money(breakdown.accessible)}</span></div><div className="flex justify-between gap-2"><span className="flex items-center gap-2"><span className="size-2 rounded-full bg-amber-500" /> Locked / unavailable</span><span className="font-medium">{money(breakdown.locked)}</span></div></div>}
          <p className="mt-5 text-xs leading-relaxed text-muted-foreground">Locked assets cannot bridge early retirement. PF/NPS access ages and available portions are your planning assumptions, not withdrawal-law claims.</p>
        </CardContent>
      </Card>
    </div>
  </div>
}
