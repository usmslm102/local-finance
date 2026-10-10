import { buttonVariants } from '@/components/ui/button'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowRight, ChevronDown, ChevronUp, RefreshCw } from 'lucide-react'
import { fetchMonthlyReview, fetchMonthlyReviewEvidence } from '@/lib/api'
import { formatDate, formatINR } from '@/lib/utils'
import type { MonthlyReviewData, ReviewCategory } from '@/types/monthly-review'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function monthLabel(month: string) {
  return new Date(`${month}-01T12:00:00`).toLocaleDateString('en-IN', {
    month: 'long',
    year: 'numeric',
  })
}

function Coverage({ data }: { data: MonthlyReviewData }) {
  const { isPrivacyMode } = usePrivacy()
  return (
    <details className="border-y py-3 text-sm">
      <summary className="cursor-pointer font-medium focus-visible:outline-2 focus-visible:outline-offset-4">
        {data.coverage_complete
          ? 'Statement date ranges cover both periods'
          : 'Incomplete statement coverage — compare with care'}
      </summary>
      <div className="mt-3 flex flex-col gap-3">
        <p className="max-w-prose text-muted-foreground leading-relaxed">
          Coverage uses dates reported by imported statements across all tracked accounts. It does
          not verify statement completeness or balances. Missing dates can make spending look lower.
        </p>
        {data.coverage.length === 0 ? (
          <p>No accounts imported yet.</p>
        ) : (
          <ul className="divide-y">
            {data.coverage.map((account) => (
              <li
                key={account.account_id}
                className="flex flex-wrap justify-between gap-x-6 gap-y-1 py-3"
              >
                <div>
                  <p className="font-medium">{isPrivacyMode ? 'Account hidden' : account.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {account.latest_end
                      ? `Latest reported end: ${formatDate(account.latest_end)}`
                      : 'No valid statement dates'}
                  </p>
                </div>
                <p className="text-xs text-muted-foreground tabular-nums">
                  Selected period: {account.current_days}/{account.current_total} days
                  <br />
                  Previous period: {account.previous_days}/{account.previous_total} days
                </p>
              </li>
            ))}
          </ul>
        )}
        <Link to="/import" className={buttonVariants({ variant: 'outline' })}>
          Import missing statements
        </Link>
      </div>
    </details>
  )
}

function CategoryEvidence({ month, category }: { month: string; category: ReviewCategory }) {
  const [period, setPeriod] = useState<'current' | 'previous'>('current')
  const [page, setPage] = useState(1)
  const { isPrivacyMode, maskValue } = usePrivacy()
  const { data, isPending, isError, refetch } = useQuery({
    queryKey: ['analytics', 'monthly-review-evidence', month, category.id, period, page],
    queryFn: () => fetchMonthlyReviewEvidence(month, category.id, period, page),
  })
  return (
    <div
      className="mt-4 flex flex-col gap-3"
      aria-label={`${category.name} supporting transactions`}
    >
      <div className="flex flex-wrap items-center gap-2">
        {(['current', 'previous'] as const).map((value) => (
          <Button
            key={value}
            variant={period === value ? 'secondary' : 'ghost'}
            aria-pressed={period === value}
            onClick={() => {
              setPeriod(value)
              setPage(1)
            }}
          >
            {value === 'current' ? 'Selected period' : 'Previous period'}
          </Button>
        ))}
      </div>
      {isPending ? (
        <p role="status" className="text-sm text-muted-foreground">
          Loading supporting transactions…
        </p>
      ) : isError ? (
        <div role="alert">
          <p>Could not load these transactions.</p>
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        data && (
          <>
            <p className="text-xs text-muted-foreground">
              {formatDate(data.period.start)} – {formatDate(data.period.end)} · {data.total}{' '}
              {data.total === 1 ? 'transaction' : 'transactions'}. Transfers, excluded rows, and
              credits are omitted.
            </p>
            {data.total === 0 ? (
              <p className="py-3 text-sm text-muted-foreground">
                No spending recorded for this category in this period.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Date</TableHead>
                    <TableHead>Payee / account</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="whitespace-nowrap text-xs">
                        {formatDate(item.date)}
                      </TableCell>
                      <TableCell className="max-w-48 break-words whitespace-normal">
                        {isPrivacyMode ? 'Payee hidden' : item.payee}
                        <p className="text-xs text-muted-foreground">
                          {isPrivacyMode ? 'Account hidden' : item.account}
                        </p>
                      </TableCell>
                      <TableCell className="text-right tabular-nums whitespace-nowrap">
                        {maskValue(formatINR(item.amount, isPrivacyMode))}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {data.total > data.page_size && (
              <div className="flex items-center justify-end gap-3 text-xs">
                <Button variant="outline" disabled={page === 1} onClick={() => setPage(page - 1)}>
                  Previous
                </Button>
                <span>
                  Page {page} of {Math.ceil(data.total / data.page_size)}
                </span>
                <Button
                  variant="outline"
                  disabled={page * data.page_size >= data.total}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            )}
          </>
        )
      )}
    </div>
  )
}

function ReviewBody({ data, compact }: { data: MonthlyReviewData; compact: boolean }) {
  const [expanded, setExpanded] = useState<string | null>(null)
  const [showAll, setShowAll] = useState(false)
  const { maskValue, isPrivacyMode } = usePrivacy()
  const money = (amount: number) => maskValue(formatINR(amount, isPrivacyMode))
  const categories = showAll ? data.categories : data.categories.slice(0, 3)
  const change = data.delta > 0 ? 'higher' : data.delta < 0 ? 'lower' : 'unchanged'
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-2">
        <p className="text-base font-medium leading-relaxed">
          {data.current.count + data.previous.count === 0 ? (
            'No spending recorded in these periods.'
          ) : data.delta === 0 ? (
            'Recorded spending is unchanged from the comparison period.'
          ) : (
            <>
              Recorded spending is {money(Math.abs(data.delta))} {change} than the comparison
              period.
            </>
          )}
        </p>
        <p className="text-sm text-muted-foreground leading-relaxed">
          {money(data.current.amount)} across {data.current.count} purchases, compared with{' '}
          {money(data.previous.amount)} across {data.previous.count}. Transfers and excluded
          transactions are omitted. Refunds are not deducted.
        </p>
        <p className="text-xs text-muted-foreground leading-relaxed">
          {formatDate(data.current_period.start)} – {formatDate(data.current_period.end)} vs{' '}
          {formatDate(data.previous_period.start)} – {formatDate(data.previous_period.end)}.
          {data.is_partial_month &&
            ' Month to date; the previous period ends on the same day, or its month end if shorter.'}
        </p>
      </div>
      <Coverage data={data} />
      {compact ? (
        <Link
          to="/cashflow"
          search={{ month: data.month }}
          hash="monthly-review"
          className={buttonVariants({ variant: 'outline' })}
        >
          Understand what changed <ArrowRight />
        </Link>
      ) : (
        <>
          {categories.length > 0 ? (
            <div>
              <h3 className="text-base font-semibold">What changed</h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Largest category changes in the imported data. Open the evidence before changing
                your plan.
              </p>
              <ul className="mt-3 divide-y">
                {categories.map((category) => {
                  const isExpanded = expanded === category.id
                  return (
                    <li key={category.id} className="py-5">
                      <div className="flex flex-wrap items-baseline justify-between gap-2">
                        <h4 className="font-semibold">{category.name}</h4>
                        <span className="text-sm font-medium tabular-nums">
                          {category.delta === 0
                            ? 'Unchanged'
                            : `${money(Math.abs(category.delta))} ${category.delta > 0 ? 'more' : 'less'}`}
                        </span>
                      </div>
                      <p className="mt-2 text-sm text-muted-foreground leading-relaxed">
                        {category.previous.count} → {category.current.count} purchases.
                        {category.current.count > 0 && category.previous.count > 0
                          ? ` Average purchase: ${money(category.previous.amount / category.previous.count)} → ${money(category.current.amount / category.current.count)}.`
                          : category.current.count === 0
                            ? ' No purchases recorded in the selected period.'
                            : ' No purchases recorded in the previous period.'}
                      </p>
                      <ul className="mt-2 flex flex-col gap-1 text-sm text-muted-foreground">
                        {category.merchants
                          .filter((merchant) => merchant.delta !== 0)
                          .map((merchant) => (
                            <li key={merchant.name}>
                              {isPrivacyMode ? 'Payee hidden' : merchant.name}:{' '}
                              {money(Math.abs(merchant.delta))}{' '}
                              {merchant.delta > 0 ? 'more' : 'less'}.
                            </li>
                          ))}
                      </ul>
                      <div className="mt-3 flex flex-wrap gap-2">
                        <Button
                          variant="outline"
                          aria-expanded={isExpanded}
                          onClick={() => setExpanded(isExpanded ? null : category.id)}
                        >
                          {isExpanded ? 'Hide transactions' : 'View transactions'}{' '}
                          {isExpanded ? <ChevronUp /> : <ChevronDown />}
                        </Button>
                        <Link
                          to="/budget"
                          search={{
                            month: data.next_month,
                            search: category.id ? category.name : undefined,
                          }}
                          className={buttonVariants({ variant: 'ghost' })}
                        >
                          Plan next month <ArrowRight />
                        </Link>
                      </div>
                      {isExpanded && <CategoryEvidence month={data.month} category={category} />}
                    </li>
                  )
                })}
              </ul>
              {data.categories.length > 3 && (
                <Button variant="ghost" onClick={() => setShowAll(!showAll)}>
                  {showAll
                    ? 'Show largest changes'
                    : `Show all ${data.categories.length} categories`}
                </Button>
              )}
            </div>
          ) : (
            <Link to="/import" className={buttonVariants({ variant: 'outline' })}>
              Import a statement to begin
            </Link>
          )}
          <p className="max-w-prose text-xs text-muted-foreground leading-relaxed">
            These comparisons describe recorded purchases; they do not predict future spending or
            identify one-off expenses automatically. Your budgets change only when you save them.
          </p>
        </>
      )}
    </div>
  )
}

export function MonthlyReview({
  month,
  compact = false,
  onMonthChange,
}: {
  month?: string
  compact?: boolean
  onMonthChange?: (month: string) => void
}) {
  const queryClient = useQueryClient()
  const { data, isPending, isError, isFetching, refetch } = useQuery({
    queryKey: ['analytics', 'monthly-review', month || 'latest'],
    queryFn: () => fetchMonthlyReview(month),
  })
  const now = new Date()
  const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  const refreshReview = () => {
    void refetch()
    if (data)
      void queryClient.invalidateQueries({
        queryKey: ['analytics', 'monthly-review-evidence', data.month],
      })
  }
  return (
    <section
      id="monthly-review"
      aria-labelledby="monthly-review-heading"
      className="border bg-card p-5 sm:p-6 flex flex-col gap-5 scroll-mt-20"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 id="monthly-review-heading" className="text-lg font-semibold">
            Monthly review{data ? ` · ${monthLabel(data.month)}` : ''}
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Understand your spending. Make room for what comes next.
          </p>
        </div>
        <div className="flex items-end gap-2">
          {onMonthChange && (
            <label className="text-xs text-muted-foreground">
              Review month
              <Input
                type="month"
                className="mt-1 w-40"
                aria-label="Review month"
                value={month || data?.month || ''}
                min="0002-01"
                max={currentMonth}
                onInput={(event) => {
                  if (/^\d{4}-\d{2}$/.test(event.currentTarget.value))
                    onMonthChange(event.currentTarget.value)
                }}
              />
            </label>
          )}
          <Button
            variant="ghost"
            size="icon"
            aria-label="Refresh monthly review"
            disabled={isFetching}
            onClick={refreshReview}
          >
            <RefreshCw
              data-icon="inline-start"
              className={isFetching ? 'animate-spin motion-reduce:animate-none' : ''}
            />
          </Button>
        </div>
      </div>
      {isPending ? (
        <div role="status" aria-label="Loading monthly review" className="flex flex-col gap-3">
          <Skeleton className="h-5 w-3/4" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      ) : isError ? (
        <div role="alert" className="flex flex-col gap-3">
          <p className="text-sm">
            Monthly review could not load. Choose a past or current month, or try again.
          </p>
          <Button variant="outline" onClick={() => refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        data && <ReviewBody key={data.month} data={data} compact={compact} />
      )}
    </section>
  )
}
