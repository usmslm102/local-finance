import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchTransactions } from '@/lib/api'
import { formatDate, formatINR } from '@/lib/utils'
import type { OverviewRange } from '@/lib/overview-range'
import type { CategorySpend } from '@/types'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { QueryError } from '@/components/ui/query-state'
import { Skeleton } from '@/components/ui/skeleton'

export function CategoryPurchases({
  category,
  range,
  periodLabel,
}: {
  category: CategorySpend
  range: OverviewRange
  periodLabel: string
}) {
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  useEffect(() => {
    if (searchInput.trim() === search) return
    const timer = window.setTimeout(() => {
      setSearch(searchInput.trim())
      setPage(1)
    }, 250)
    return () => window.clearTimeout(timer)
  }, [searchInput, search])
  const { isPrivacyMode } = usePrivacy()
  const { data, isPending, isError, refetch } = useQuery({
    queryKey: ['transactions', 'spending', range, category.category_id, search, page],
    queryFn: () =>
      fetchTransactions({
        spending_only: true,
        category_id: category.category_id,
        ...range,
        page,
        page_size: 10,
        search: search || undefined,
      }),
  })
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Input
          aria-label="Search purchases"
          placeholder="Search payee, narration or reference"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
        />
        {searchInput && (
          <Button
            variant="ghost"
            size="sm"
            aria-label="Clear purchase search"
            onClick={() => {
              setSearchInput('')
              setSearch('')
              setPage(1)
            }}
          >
            Clear
          </Button>
        )}
      </div>
      <div className="space-y-4">
        {isPending ? (
          <div role="status" aria-label="Loading category purchases" className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : isError ? (
          <QueryError message="Unable to load category purchases" onRetry={refetch} />
        ) : data && data.items.length > 0 ? (
          <>
            <div className="max-h-[50dvh] overflow-auto rounded-lg border">
              <Table aria-label={`${category.category_name} purchases · ${periodLabel}`}>
                <TableHeader className="sticky top-0 z-10 bg-card">
                  <TableRow>
                    <TableHead className="hidden sm:table-cell">Date</TableHead>
                    <TableHead>Purchase</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.items.map((transaction) => (
                    <TableRow key={transaction.id}>
                      <TableCell className="hidden whitespace-nowrap text-xs sm:table-cell">
                        {formatDate(transaction.tx_date)}
                      </TableCell>
                      <TableCell className="max-w-64 whitespace-normal break-words">
                        {isPrivacyMode
                          ? 'Payee hidden'
                          : transaction.cleaned_payee || 'Unknown payee'}
                        <p className="mt-1 text-xs text-muted-foreground sm:hidden">
                          {formatDate(transaction.tx_date)}
                        </p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {isPrivacyMode ? 'Account hidden' : transaction.account_name}
                        </p>
                      </TableCell>
                      <TableCell className="text-right text-xs whitespace-nowrap tabular-nums sm:text-sm">
                        {formatINR(transaction.amount, isPrivacyMode)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
              <span aria-live="polite">
                {data.total} {search ? 'matching ' : ''}
                {data.total === 1 ? 'purchase' : 'purchases'} · Page {data.page} of{' '}
                {data.total_pages}
              </span>
              {data.total_pages > 1 && (
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page === 1}
                    onClick={() => setPage(page - 1)}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= data.total_pages}
                    onClick={() => setPage(page + 1)}
                  >
                    Next
                  </Button>
                </div>
              )}
            </div>
          </>
        ) : (
          <p role="status" className="text-sm text-muted-foreground">
            {search
              ? 'No purchases match your search. Try another payee or clear the search.'
              : 'No purchases found for this category and period.'}
          </p>
        )}
      </div>
    </div>
  )
}
