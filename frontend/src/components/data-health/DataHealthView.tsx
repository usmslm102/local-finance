import { useState, useId } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, RefreshCw, Check, CircleAlert } from 'lucide-react'
import { fetchAccounts, fetchStatements, fetchDataHealth } from '@/lib/api'
import { localDate, monthName, statementCoverage } from '@/lib/planning'
import { formatDate } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { Button, buttonVariants } from '@/components/ui/button'
import { Field, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from '@/components/ui/empty'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { PageLoading, QueryError } from '@/components/ui/query-state'

export function DataHealthView() {
  const id = useId()
  const today = localDate()
  const { isPrivacyMode } = usePrivacy()
  const [month, setMonth] = useState(today.slice(0, 7))
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const statements = useQuery({ queryKey: ['statements'], queryFn: fetchStatements })
  const health = useQuery({ queryKey: ['data-health'], queryFn: fetchDataHealth })
  const refresh = () => {
    void accounts.refetch()
    void statements.refetch()
    void health.refetch()
  }
  if (accounts.isPending || statements.isPending || health.isPending)
    return <PageLoading label="Checking local data health" />
  if (accounts.isError || statements.isError || health.isError)
    return <QueryError message="Unable to inspect data health" onRetry={refresh} />
  const months = [
    ...new Set([
      today.slice(0, 7),
      ...(statements.data ?? [])
        .flatMap((s) => [s.start_date?.slice(0, 7), s.end_date?.slice(0, 7)])
        .filter((m): m is string => !!m && /^\d{4}-\d{2}$/.test(m) && m <= today.slice(0, 7)),
    ]),
  ]
    .sort()
    .reverse()
  // Include intermediate months for statements spanning several cycles.
  for (let n = 1; n <= 12; n++) {
    const date = new Date(`${today.slice(0, 7)}-01T00:00:00Z`)
    date.setUTCMonth(date.getUTCMonth() - n)
    const value = date.toISOString().slice(0, 7)
    if (!months.includes(value)) months.push(value)
  }
  months.sort().reverse()
  const items = (accounts.data ?? []).map((account) => ({
    account,
    coverage: statementCoverage(statements.data ?? [], account.id, month, today),
    stats: health.data?.find((a) => a.account_id === account.id),
  }))
  const complete = items.filter(
    (item) => item.coverage.covered === item.coverage.total && item.coverage.total > 0
  ).length
  const reviewCount = (health.data ?? []).reduce((sum, a) => sum + a.review_count, 0)
  const stale = items.filter(
    (item) =>
      !item.coverage.latest ||
      Date.parse(`${today}T00:00:00Z`) - Date.parse(`${item.coverage.latest}T00:00:00Z`) >
        35 * 86400000
  ).length
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Data health</h1>
          <p className="mt-1 max-w-prose text-sm text-muted-foreground">
            See what your imported statements cover before trusting a comparison or a forecast.
          </p>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={refresh}
          disabled={accounts.isFetching || statements.isFetching || health.isFetching}
        >
          <RefreshCw data-icon="inline-start" />
          Refresh checks
        </Button>
      </div>
      {items.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>No accounts to check yet</EmptyTitle>
            <EmptyDescription>
              Import a statement to see date coverage, freshness and transactions that need
              categorization.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Link to="/import" className={buttonVariants()}>
              Import statement
            </Link>
          </EmptyContent>
        </Empty>
      ) : (
        <>
          <Alert>
            <CircleAlert />
            <AlertTitle>
              {complete === items.length && stale === 0 && reviewCount === 0
                ? 'No issues in these checks'
                : 'A few checks before you plan'}
            </AlertTitle>
            <AlertDescription>
              {complete} of {items.length} accounts cover the selected period. {stale} accounts have
              no valid statement range or a latest end date over 35 days ago. {reviewCount}{' '}
              transactions need category review across all history.
            </AlertDescription>
          </Alert>
          <div className="flex flex-wrap items-end justify-between gap-4">
            <Field className="w-full sm:w-56">
              <FieldLabel htmlFor={id}>Coverage month</FieldLabel>
              <Select
                value={month}
                onValueChange={(value) => {
                  if (value) setMonth(value)
                }}
              >
                <SelectTrigger id={id}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {months.map((m) => (
                      <SelectItem key={m} value={m}>
                        {monthName(m)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Link to="/import" className={buttonVariants({ variant: 'outline', size: 'sm' })}>
              Fill gaps with statements
              <ArrowRight className="size-4" />
            </Link>
          </div>
          <p className="text-sm text-muted-foreground">
            {month === today.slice(0, 7)
              ? `Coverage is checked from the first of this month through ${formatDate(today)}.`
              : `Coverage is checked across all of ${monthName(month)}.`}{' '}
            Overlapping statements count once. Future dates do not fill past gaps.
          </p>
          <div className="flex flex-col divide-y border-y">
            {items.map(({ account, coverage, stats }) => (
              <section key={account.id} className="flex flex-col gap-3 py-5">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <h2 className="font-semibold">
                      {isPrivacyMode
                        ? 'Account hidden'
                        : account.nickname ||
                          `${account.bank_name} · ${account.account_number_mask}`}
                    </h2>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {account.account_type.replaceAll('_', ' ')} · latest reported end{' '}
                      {coverage.latest ? formatDate(coverage.latest) : 'unavailable'}
                    </p>
                  </div>
                  <Badge variant={coverage.covered === coverage.total ? 'secondary' : 'outline'}>
                    {coverage.covered}/{coverage.total} days covered
                  </Badge>
                </div>
                {coverage.gaps.length ? (
                  <div className="flex flex-wrap gap-2" aria-label="Missing statement date ranges">
                    {coverage.gaps.map((gap) => (
                      <Badge key={gap.start} variant="outline">
                        {formatDate(gap.start)}
                        {gap.end !== gap.start ? ` – ${formatDate(gap.end)}` : ''}
                      </Badge>
                    ))}
                  </div>
                ) : (
                  <p className="flex items-center gap-2 text-sm text-success">
                    <Check className="size-4" />
                    Reported date ranges cover this period.
                  </p>
                )}
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="text-sm text-muted-foreground">
                    {stats?.transaction_count ?? 0} transactions · latest activity{' '}
                    {stats?.latest_transaction
                      ? formatDate(stats.latest_transaction)
                      : 'unavailable'}
                    {(stats?.review_count ?? 0) > 0
                      ? ` · ${stats!.review_count} need category review`
                      : ''}
                  </p>
                  <Link
                    to="/transactions"
                    search={{
                      account: account.id,
                      ...(stats?.review_count ? { category: 'REVIEW_NEEDED' } : {}),
                    }}
                    className={buttonVariants({ variant: 'ghost', size: 'sm' })}
                  >
                    {stats?.review_count ? 'Review categories' : 'Open ledger'}
                    <ArrowRight className="size-4" />
                  </Link>
                </div>
              </section>
            ))}
          </div>
          <section className="flex flex-col gap-3">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-lg font-semibold">Finish categorizing your history</h2>
              <Link
                to="/transactions"
                search={{ category: 'REVIEW_NEEDED' }}
                className={buttonVariants({ variant: 'outline', size: 'sm' })}
              >
                Review {reviewCount} transactions
                <ArrowRight className="size-4" />
              </Link>
            </div>
            <p className="max-w-prose text-sm text-muted-foreground">
              This includes missing categories and “Others & Uncategorized”, excluding transfers and
              excluded activity. Edit a transaction or create a rule in the ledger to improve future
              imports.
            </p>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Check</TableHead>
                  <TableHead>Meaning</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow>
                  <TableCell>Coverage gaps</TableCell>
                  <TableCell className="whitespace-normal">
                    Missing reported statement dates, even if transactions exist during those dates.
                  </TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>Stale account</TableCell>
                  <TableCell className="whitespace-normal">
                    No valid statement range or the latest range ends more than 35 days before
                    today.
                  </TableCell>
                </TableRow>
                <TableRow>
                  <TableCell>Category review</TableCell>
                  <TableCell className="whitespace-normal">
                    An opportunity to make spending breakdowns more useful; it does not indicate an
                    incorrect transaction.
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </section>
        </>
      )}
      <p className="max-w-prose text-sm text-muted-foreground">
        These checks use local records only. Statement date coverage does not verify balances,
        missing rows, or whether you have imported every account.{' '}
        <Link to="/planning" className="underline underline-offset-4">
          Open savings forecast
        </Link>
        .
      </p>
    </div>
  )
}
