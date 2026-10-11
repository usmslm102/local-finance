import { useId, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Line, LineChart, CartesianGrid, XAxis, YAxis, ReferenceLine } from 'recharts'
import { RotateCcw, ArrowRight } from 'lucide-react'
import { fetchAnalytics, fetchSavingsGoals, fetchDataHealth } from '@/lib/api'
import {
  forecastBaseline,
  forecastScenario,
  goalReservations,
  localDate,
  monthName,
} from '@/lib/planning'
import { formatDate, formatINR } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { Button, buttonVariants } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel, FieldDescription } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartLegend, ChartLegendContent } from '@/components/ui/chart'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { PageLoading, QueryError } from '@/components/ui/query-state'

const axisCurrency = new Intl.NumberFormat('en-IN', {
  style: 'currency', currency: 'INR', notation: 'compact', maximumFractionDigits: 1,
})

export function ForecastView() {
  const id = useId()
  const { isPrivacyMode } = usePrivacy()
  const analytics = useQuery({ queryKey: ['analytics'], queryFn: fetchAnalytics })
  const goals = useQuery({ queryKey: ['savings-goals'], queryFn: fetchSavingsGoals })
  const health = useQuery({ queryKey: ['data-health'], queryFn: fetchDataHealth })
  const [overrides, setOverrides] = useState<
    Partial<Record<'opening' | 'income' | 'expense' | 'oneOff', string>>
  >({})
  const [months, setMonths] = useState(3)
  const [reserveGoals, setReserveGoals] = useState(true)
  const [error, setError] = useState('')
  if (analytics.isPending || goals.isPending || health.isPending)
    return <PageLoading label="Loading forecast inputs" />
  if (analytics.isError || goals.isError || health.isError)
    return (
      <QueryError
        message="Unable to load forecast inputs"
        onRetry={() => {
          void analytics.refetch()
          void goals.refetch()
          void health.refetch()
        }}
      />
    )
  const data = analytics.data!
  const baseline = forecastBaseline(data.monthly_trends ?? [])
  const defaults = {
    opening: Math.round((data.total_bank_liquidity - data.total_credit_due) * 100) / 100,
    income: baseline.income,
    expense: baseline.expense,
    oneOff: 0,
  }
  const fields = [
    {
      key: 'opening',
      label: 'Starting available money (₹)',
      description:
        'Recorded bank liquidity minus recorded card dues. Adjust this to your actual balance.',
      min: -1e12,
    },
    {
      key: 'income',
      label: 'Monthly income (₹)',
      description:
        'Recorded credits include refunds and other non-recurring credits. Set a realistic monthly amount.',
      min: 0,
    },
    {
      key: 'expense',
      label: 'Monthly spending (₹)',
      description:
        'Includes bank and card purchases; internal transfers and excluded transactions are omitted.',
      min: 0,
    },
    {
      key: 'oneOff',
      label: 'One-off expense in the first month (₹)',
      description: 'An extra purchase or expense outside your usual monthly spending.',
      min: 0,
    },
  ] as const
  const values = Object.fromEntries(
    fields.map((f) => [f.key, Number(overrides[f.key] ?? defaults[f.key])])
  ) as typeof defaults
  const valid = fields.every((f) => {
    const value = values[f.key]
    return (
      Number.isFinite(value) &&
      value >= f.min &&
      value <= 1e12 &&
      (overrides[f.key] ?? String(defaults[f.key])).trim() !== ''
    )
  })
  const reservations = reserveGoals
    ? goalReservations(goals.data ?? [], months)
    : Array(months + 1).fill(0)
  const points = valid
    ? forecastScenario({ ...values, goalContribution: 0, goalReservations: reservations, months })
    : []
  const final = points.at(-1)
  const shortage = points.find((point) => point.available < 0)
  const latest = health.data
    ?.map((a) => a.latest_transaction)
    .filter(Boolean)
    .sort()
    .at(-1)
  const stale =
    !latest ||
    latest.slice(0, 10) <
      new Date(Date.parse(`${localDate()}T00:00:00Z`) - 35 * 86400000).toISOString().slice(0, 10)
  const money = (amount: number) => formatINR(amount, isPrivacyMode)
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">Look ahead before you commit</h2>
          <p className="mt-1 max-w-prose text-sm text-muted-foreground">
            A monthly savings scenario from today. Change the assumptions to see how much remains
            available after planned goals.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            setOverrides({})
            setReserveGoals(true)
            setError('')
          }}
        >
          <RotateCcw data-icon="inline-start" />
          Reset assumptions
        </Button>
      </div>
      {(baseline.count < 3 || stale) && (
        <Alert>
          <AlertTitle>History needs a closer look</AlertTitle>
          <AlertDescription>
            {baseline.count} of the last 3 complete calendar months contain recorded activity.{' '}
            {latest
              ? `Latest recorded transaction: ${formatDate(latest)}.`
              : 'No recorded activity yet.'}{' '}
            Defaults may understate income or spending.{' '}
            <Link to="/data-health" className="underline underline-offset-4">
              Check statement coverage
            </Link>{' '}
            and enter your own assumptions.
          </AlertDescription>
        </Alert>
      )}
      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <form
          className="flex flex-col gap-5"
          onSubmit={(e) => {
            e.preventDefault()
            setError(
              valid
                ? ''
                : 'Enter valid amounts in every assumption. Spending and income cannot be negative.'
            )
          }}
        >
          <FieldGroup>
            {fields.map((field) => (
              <Field key={field.key}>
                <FieldLabel htmlFor={`${id}-${field.key}`}>{field.label}</FieldLabel>
                <Input
                  id={`${id}-${field.key}`}
                  type={isPrivacyMode ? 'password' : 'number'}
                  value={isPrivacyMode ? '••••••' : (overrides[field.key] ?? defaults[field.key])}
                  disabled={isPrivacyMode}
                  min={field.min}
                  max={1e12}
                  step="0.01"
                  required
                  aria-invalid={!valid || undefined}
                  onChange={(e) => {
                    setOverrides((previous) => ({ ...previous, [field.key]: e.target.value }))
                    setError('')
                  }}
                />
                <FieldDescription>{field.description}</FieldDescription>
              </Field>
            ))}
          </FieldGroup>
          <Field orientation="horizontal">
            <FieldLabel htmlFor={`${id}-reserve`}>Reserve planned goal contributions</FieldLabel>
            <Switch id={`${id}-reserve`} checked={reserveGoals} onCheckedChange={setReserveGoals} />
          </Field>
          <p className="text-xs text-muted-foreground">
            {isPrivacyMode ? 'Turn off Discreet Mode to edit amounts. ' : ''}Assumptions are
            temporary and reset when you leave this view.
          </p>
          {!valid && (
            <p role="alert" className="text-sm text-destructive">
              {error || 'Enter valid amounts to display the forecast.'}
            </p>
          )}
        </form>
        <Card>
          <CardHeader>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <CardTitle>Available money over time</CardTitle>
              <ToggleGroup
                value={[String(months)]}
                onValueChange={(v) => {
                  if (v[0]) setMonths(Number(v[0]))
                }}
                variant="outline"
                size="sm"
                aria-label="Forecast horizon"
              >
                {[3, 6, 12].map((n) => (
                  <ToggleGroupItem key={n} value={String(n)}>
                    {n} months
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            </div>
            <CardDescription>
              Full monthly assumptions start today. Goal reserves are capped at each goal’s
              remaining target.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-5">
            {valid ? (
              <>
                <dl className="flex flex-wrap justify-between gap-4 text-sm">
                  <div>
                    <dt className="text-muted-foreground">After {months} months</dt>
                    <dd className="text-2xl font-semibold tabular-nums">
                      {money(final!.available)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">Reserved for goals</dt>
                    <dd className="font-semibold tabular-nums">{money(reservations[months])}</dd>
                  </div>
                </dl>
                {isPrivacyMode ? (
                  <div className="flex h-64 items-center justify-center rounded-md bg-muted text-sm text-muted-foreground">
                    Forecast chart hidden in Discreet Mode
                  </div>
                ) : (
                  <ChartContainer
                    className="h-64 w-full"
                    config={{
                      balance: { label: 'Before goals', color: 'var(--muted-foreground)' },
                      available: { label: 'Available', color: 'var(--primary)' },
                    }}
                  >
                    <LineChart data={points} accessibilityLayer margin={{ left: 8, right: 12 }}>
                      <CartesianGrid vertical={false} />
                      <XAxis dataKey="label" tickLine={false} axisLine={false} />
                      <YAxis
                        width={72}
                        tickFormatter={(v) => axisCurrency.format(v)}
                        tickLine={false}
                        axisLine={false}
                      />
                      <ReferenceLine y={0} stroke="var(--destructive)" strokeDasharray="4 4" />
                      <ChartTooltip
                        content={
                          <ChartTooltipContent
                            formatter={(value, name) => (
                              <span>
                                {name === 'available'
                                  ? 'Available after goals'
                                  : 'Before goal reserves'}
                                : {formatINR(Number(value))}
                              </span>
                            )}
                          />
                        }
                      />
                      <ChartLegend content={<ChartLegendContent />} />
                      <Line
                        dataKey="balance"
                        stroke="var(--color-balance)"
                        strokeDasharray="4 4"
                        dot={false}
                        isAnimationActive={false}
                      />
                      <Line
                        dataKey="available"
                        stroke="var(--color-available)"
                        strokeWidth={2}
                        dot={false}
                        isAnimationActive={false}
                      />
                    </LineChart>
                  </ChartContainer>
                )}
                <p role="status" className="text-sm">
                  {shortage
                    ? `This scenario goes below zero ${shortage.label === 'Now' ? 'at the starting point' : `within ${shortage.label.replace('+', '').replace(' mo', ' months')}`}. Reduce spending, revise contributions, or update your starting balance.`
                    : 'This scenario stays above zero for the selected horizon.'}
                </p>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Horizon</TableHead>
                      <TableHead className="text-right">Before goals</TableHead>
                      <TableHead className="text-right">Available</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {points
                      .filter((_, i) => i === 0 || i === months || i % 3 === 0)
                      .map((point) => (
                        <TableRow key={point.month}>
                          <TableCell>{point.label}</TableCell>
                          <TableCell className="text-right tabular-nums">
                            {money(point.balance)}
                          </TableCell>
                          <TableCell className="text-right tabular-nums">
                            {money(point.available)}
                          </TableCell>
                        </TableRow>
                      ))}
                  </TableBody>
                </Table>
              </>
            ) : (
              <p className="py-12 text-center text-sm text-muted-foreground">
                Complete the assumptions to calculate a scenario.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
      <details className="border-y py-4 text-sm">
        <summary className="cursor-pointer font-medium">How this scenario is calculated</summary>
        <div className="mt-4 flex flex-col gap-3 text-muted-foreground">
          <p className="max-w-prose">
            Starting money + monthly income − monthly spending − the one-off expense − planned goal
            reserves. Existing card dues are reserved once in the default starting value.
            Subscription costs and future card purchases are part of monthly spending, so they are
            not added again.
          </p>
          <p className="max-w-prose">
            History defaults use averages of the recorded months below. A recorded month may still
            have incomplete statements. This is a planning estimate, not a daily bank-balance
            prediction: salary dates, bill dates, interest, inflation and investment returns are not
            modeled.
          </p>
          <ul className="flex flex-wrap gap-3">
            {baseline.months.map((m) => (
              <li key={m.month}>
                {monthName(m.month)}:{' '}
                {m.recorded
                  ? `${money(m.recorded.income)} in / ${money(m.recorded.expense)} out`
                  : 'no recorded activity'}
              </li>
            ))}
          </ul>
          <Link
            to="/data-health"
            className={buttonVariants({ variant: 'outline', size: 'sm', className: 'self-start' })}
          >
            Review data health
            <ArrowRight className="size-4" />
          </Link>
        </div>
      </details>
    </div>
  )
}
