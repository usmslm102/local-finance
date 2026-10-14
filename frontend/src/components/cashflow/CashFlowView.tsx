import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardAction,
} from '@/components/ui/card'
import { cn } from '@/lib/utils'
import {
  cashFlowPeriodDates,
  cashFlowNodeParams,
  contributingTransactions,
} from '@/lib/cashflow-drilldown'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SpendingChart } from '@/components/dashboard/SpendingChart'
import { CategoryPurchases } from './CategoryPurchases'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cashFlowCategories, spendingMonthLabel, validSpendingMonth } from '@/lib/spending'
import { QueryError, PageLoading } from '@/components/ui/query-state'
import { usePrivacy } from '@/components/privacy-provider'
import React, { useState, useCallback, useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { fetchCashFlowIntelligence, fetchAllTransactions } from '@/lib/api'
import { SankeyDiagram } from './SankeyDiagram'
import { AnomaliesSection } from './AnomaliesSection'
import { CategoryComparisonTable } from './CategoryComparisonTable'
import { MonthlyReview } from './MonthlyReview'
import { formatINR, formatDate } from '@/lib/utils'
import { SankeyNode, Transaction, CashFlowSearchParams } from '@/types'
import {
  TrendingUp,
  TrendingDown,
  PiggyBank,
  PieChart,
  Calendar,
  Layers,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

export const CashFlowView: React.FC = () => {
  const searchParams = (useSearch({ strict: false }) as CashFlowSearchParams) || {}
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { isPrivacyMode } = usePrivacy()
  const money = (amount: number) => (isPrivacyMode ? '₹••••••' : formatINR(amount, isPrivacyMode))
  const selectedPeriod = searchParams.month || searchParams.period || ''

  const handlePeriodChange = useCallback(
    (newPeriod: string) => {
      navigate({
        to: '/cashflow',
        search: newPeriod ? { month: newPeriod } : {},
        replace: false,
        resetScroll: false,
      })
    },
    [navigate]
  )

  // 1. Fetch Cash Flow Intelligence Data
  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['cashflow-intelligence', selectedPeriod],
    queryFn: () => fetchCashFlowIntelligence(selectedPeriod || undefined),
  })

  const activeNode = data?.sankey.nodes.find((node) => node.id === searchParams.node) ?? null
  const isDrilldownOpen = !!activeNode
  const categories = cashFlowCategories(data?.sankey.nodes ?? [], data?.summary.total_outflow ?? 0)
  const selectedCategory = categories.find((item) => `cat_${item.category_id}` === activeNode?.id)
  const [nodePage, setNodePage] = useState(1)
  useEffect(() => { setNodePage(1) }, [activeNode?.id, data?.selected_month])
  const {
    data: nodeTxs,
    isLoading: isNodeTxsLoading,
    isError: isNodeError,
    refetch: retryNode,
  } = useQuery({
    queryKey: ['node-transactions', activeNode?.id, data?.selected_month, data?.available_months],
    queryFn: async () => {
      if (!activeNode || !data || ['SURPLUS', 'DEFICIT'].includes(activeNode.type)) return []
      const dates = cashFlowPeriodDates(data.selected_month, data.available_months)
      const params = cashFlowNodeParams(activeNode)
      if (params.account_id === 'splitwise') delete params.account_id
      const transactions =
        activeNode.type === 'ACCOUNT'
          ? (
              await Promise.all([
                fetchAllTransactions({ ...dates, ...params, spending_only: true }),
                fetchAllTransactions({ ...dates, ...params, tx_type: 'CREDIT' }),
              ])
            )
              .flat()
              .sort((a, b) => b.tx_date.localeCompare(a.tx_date))
          : await fetchAllTransactions({
              ...dates,
              ...params,
              spending_only: activeNode.type === 'CHANNEL' || undefined,
            })
      return contributingTransactions(activeNode, transactions)
    },
    enabled: !!activeNode && isDrilldownOpen && activeNode.type !== 'CATEGORY',
  })

  const handleNodeClick = (node: SankeyNode) => {
    setNodePage(1)
    void navigate({
      to: '/cashflow',
      search: { month: data?.selected_month || selectedPeriod, node: node.id },
      resetScroll: false,
    })
  }
  const handleCategoryRowClick = (categoryId: string) => {
    const node = data?.sankey.nodes.find((item) => item.id === `cat_${categoryId}`)
    if (node) handleNodeClick(node)
  }
  const closeDrilldown = () => {
    void navigate({
      to: '/cashflow',
      search: { month: data?.selected_month || selectedPeriod },
      resetScroll: false,
    })
  }
  if (isLoading) return <PageLoading label="Loading cash flow" />
  if (isError) return <QueryError message="Unable to load cash flow" onRetry={refetch} />

  const summary = data?.summary || {
    total_inflow: 0,
    total_outflow: 0,
    net_surplus: 0,
    savings_rate: 0,
    fixed_needs_spend: 0,
    discretionary_spend: 0,
    credit_card_share_pct: 0,
    direct_bank_share_pct: 0,
    top_spending_category: '',
    top_spending_amount: 0,
  }

  const currentSelectedMonth = data?.selected_month || ''
  const availableMonths = [
    ...new Set([
      ...(/^\d{4}-\d{2}$/.test(selectedPeriod || currentSelectedMonth)
        ? [selectedPeriod || currentSelectedMonth]
        : []),
      ...(data?.available_months || []),
    ]),
  ]
    .sort()
    .reverse()
  const previousMonth = data?.previous_month || ''
  const isMonthly = validSpendingMonth(currentSelectedMonth)
  const monthIndex = availableMonths.indexOf(currentSelectedMonth)
  const purchaseCount = categories.reduce((sum, category) => sum + category.tx_count, 0)
  const largestIncrease = isMonthly
    ? [...(data?.category_comparisons ?? [])]
        .filter((item) => item.delta_amount > 0)
        .sort((a, b) => b.delta_amount - a.delta_amount)[0]
    : undefined
  const now = new Date()
  const latestMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  const periodLabel = isMonthly
    ? spendingMonthLabel(currentSelectedMonth)
    : currentSelectedMonth === 'ALL'
      ? 'All time'
      : currentSelectedMonth === 'LAST_3_MONTHS'
        ? 'Latest 3 recorded months'
        : currentSelectedMonth

  return (
    <div className="flex flex-col gap-8">
      {/* Header & Month Filter Controls */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl flex items-center gap-2.5">
            <Layers className="size-6 text-primary" />
            Cash Flow
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            See where your money comes from, where it goes, and what changed this month.
          </p>
        </div>

        {/* Period Selector Bar */}
        <div className="flex flex-wrap items-center gap-2 bg-card/60 border rounded-lg p-1 shadow-xs">
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Previous recorded month"
            disabled={monthIndex < 0 || monthIndex + 1 >= availableMonths.length}
            onClick={() => handlePeriodChange(availableMonths[monthIndex + 1])}
          >
            <ChevronLeft />
          </Button>
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label="Next recorded month"
            disabled={monthIndex <= 0}
            onClick={() => handlePeriodChange(availableMonths[monthIndex - 1])}
          >
            <ChevronRight />
          </Button>
          <Calendar className="size-3.5 text-muted-foreground ml-2" />
          <Select
            value={selectedPeriod || currentSelectedMonth}
            onValueChange={(value) => value && handlePeriodChange(value)}
          >
            <SelectTrigger className="w-full sm:w-60" aria-label="Cash flow period">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {availableMonths.map((month) => (
                  <SelectItem key={month} value={month}>
                    {new Date(`${month}-01T00:00:00`).toLocaleDateString('en-IN', {
                      month: 'long',
                      year: 'numeric',
                    })}
                  </SelectItem>
                ))}
                <SelectItem value="LAST_3_MONTHS">Latest 3 recorded months</SelectItem>
                <SelectItem value="ALL">All time</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>

          <Button
            aria-label="Refresh Data"
            size="sm"
            variant="ghost"
            onClick={() => {
              void refetch()
              void queryClient.invalidateQueries({ queryKey: ['transactions', 'spending'] })
              void queryClient.invalidateQueries({ queryKey: ['node-transactions'] })
            }}
            disabled={isFetching}
            className="size-7 p-0"
            title="Refresh Data"
          >
            <RefreshCw
              data-icon="inline-start"
              className={cn(
                'size-3.5',
                isFetching ? 'animate-spin text-primary' : 'text-muted-foreground'
              )}
            />
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-end justify-between gap-3">
        <p className="text-sm text-muted-foreground" aria-live="polite">
          {periodLabel} · {purchaseCount} {purchaseCount === 1 ? 'purchase' : 'purchases'}
        </p>
        <div className="space-y-1.5">
          <Label htmlFor="cashflow-month">Choose month</Label>
          <Input
            id="cashflow-month"
            type="month"
            min="0002-01"
            max={latestMonth}
            className="w-44"
            value={isMonthly ? currentSelectedMonth : ''}
            onChange={(event) => {
              if (validSpendingMonth(event.target.value) && event.target.value <= latestMonth)
                handlePeriodChange(event.target.value)
            }}
          />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader>
            <CardTitle>Total inflow</CardTitle>
            <CardAction>
              <TrendingUp className="size-4 text-success" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold tabular-nums">{money(summary.total_inflow)}</p>
            <CardDescription>Salary, refunds and incoming deposits</CardDescription>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Total outflow</CardTitle>
            <CardAction>
              <TrendingDown className="size-4 text-destructive" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold tabular-nums">{money(summary.total_outflow)}</p>
            <CardDescription>
              {summary.credit_card_share_pct.toFixed(0)}% cards ·{' '}
              {summary.direct_bank_share_pct.toFixed(0)}% bank
            </CardDescription>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{summary.net_surplus >= 0 ? 'Net savings' : 'Shortfall'}</CardTitle>
            <CardAction>
              <PiggyBank className="size-4 text-primary" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold tabular-nums">{money(summary.net_surplus)}</p>
            <CardDescription>{summary.savings_rate.toFixed(1)}% of income retained</CardDescription>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Needs & wants</CardTitle>
            <CardAction>
              <PieChart className="size-4 text-info" />
            </CardAction>
          </CardHeader>
          <CardContent className="flex flex-col gap-1">
            <p className="tabular-nums">Needs: {money(summary.fixed_needs_spend)}</p>
            <p className="tabular-nums">Wants: {money(summary.discretionary_spend)}</p>
            <CardDescription>Essential and discretionary spending</CardDescription>
          </CardContent>
        </Card>
      </div>

      <section aria-label="Spending insights" className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Biggest category</CardTitle>
            </CardHeader>
            <CardContent>
              {categories[0] ? (
                <Button
                  variant="ghost"
                  className="h-auto w-full items-start justify-between gap-3 whitespace-normal text-left"
                  onClick={() => handleCategoryRowClick(categories[0].category_id)}
                >
                  <span>
                    {categories[0].category_name}
                    <span className="mt-1 block text-xs text-muted-foreground">
                      {categories[0].percentage.toFixed(1)}% of spending · View purchases
                    </span>
                  </span>
                  <span className="tabular-nums">{money(categories[0].total_amount)}</span>
                </Button>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No spending recorded for this period.
                </p>
              )}
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Average purchase</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-semibold tabular-nums">
                {money(purchaseCount ? summary.total_outflow / purchaseCount : 0)}
              </p>
              <CardDescription>
                Across {purchaseCount} purchases in {categories.length} categories
              </CardDescription>
            </CardContent>
          </Card>
        </div>
        {largestIncrease && (
          <Button
            variant="outline"
            className="h-auto w-full justify-start whitespace-normal text-left py-3"
            onClick={() => handleCategoryRowClick(largestIncrease.category_id)}
          >
            <TrendingUp className="shrink-0" /> {largestIncrease.category_name} rose the most:{' '}
            {money(largestIncrease.delta_amount)} more than {spendingMonthLabel(previousMonth)}.
            View purchases.
          </Button>
        )}
        <SpendingChart
          data={categories}
          selectedCategory={selectedCategory?.category_id}
          onCategorySelect={handleCategoryRowClick}
        />
        <p className="text-xs text-muted-foreground">
          Personal Splitwise shares are included. Transfers, excluded rows, and credits are omitted
          from spending; refunds are not deducted. Missing statements can make recorded spending
          look lower.
        </p>
      </section>

      <Card>
        <CardHeader>
          <CardTitle>
            <h2 className="flex items-center gap-2">
              <Layers className="size-4 text-primary" />
              Where your money goes
            </h2>
          </CardTitle>
          <CardDescription>
            Follow the flow from income to spending. Select a node for the supporting transactions.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <SankeyDiagram
            data={
              data?.sankey || {
                nodes: [],
                links: [],
                total_inflow: 0,
                total_outflow: 0,
                net_surplus: 0,
                savings_rate: 0,
              }
            }
            onNodeClick={handleNodeClick}
            selectedNodeId={activeNode?.id}
          />
        </CardContent>
      </Card>

      {/* MoM Anomaly & Shift Intelligence */}
      {isMonthly && (
        <AnomaliesSection
          anomalies={data?.anomalies || []}
          previousMonth={previousMonth}
          selectedMonth={currentSelectedMonth}
        />
      )}

      {/* Category Shifts & 6-Month Trajectory Table */}
      {isMonthly && (
        <CategoryComparisonTable
          items={data?.category_comparisons || []}
          selectedMonth={currentSelectedMonth}
          previousMonth={previousMonth}
          totalOutflow={summary.total_outflow}
          onCategoryClick={handleCategoryRowClick}
        />
      )}
      {isMonthly && (
        <details className="rounded-xl border bg-card p-4">
          <summary className="cursor-pointer font-medium">
            Monthly review and statement coverage
          </summary>
          <div className="mt-4">
            <MonthlyReview month={currentSelectedMonth} onMonthChange={handlePeriodChange} />
          </div>
        </details>
      )}

      {/* Drilldown Modal for Selected Node */}
      <Dialog
        open={isDrilldownOpen}
        onOpenChange={(open) => {
          if (!open) closeDrilldown()
        }}
      >
        <DialogContent className="max-w-2xl max-h-[85dvh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center justify-between pr-6">
              <div className="flex items-center gap-2.5">
                <div
                  className="size-3 rounded-full"
                  style={{ backgroundColor: activeNode?.color_hex || '#3B82F6' }}
                />
                <span>
                  {activeNode?.name} {selectedCategory ? 'purchases' : 'transactions'}
                </span>
              </div>
              <Badge variant="outline">{activeNode && money(activeNode.total_value)}</Badge>
            </DialogTitle>
            <DialogDescription>
              {periodLabel}
              {selectedCategory
                ? ` · ${selectedCategory.tx_count} purchases · ${selectedCategory.percentage.toFixed(1)}% of spending`
                : activeNode?.type === 'ACCOUNT'
                  ? ' · Incoming and outgoing personal flows'
                  : activeNode?.type === 'CHANNEL'
                    ? ' · Personal purchases on this card'
                    : ' · Supporting transactions'}
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto mt-3">
            {selectedCategory ? (
              <CategoryPurchases
                key={`${currentSelectedMonth}:${selectedCategory.category_id}`}
                category={selectedCategory}
                range={cashFlowPeriodDates(currentSelectedMonth, availableMonths)}
                periodLabel={periodLabel}
              />
            ) : activeNode && ['SURPLUS', 'DEFICIT'].includes(activeNode.type) ? (
              <p className="py-6 text-sm text-muted-foreground">
                {activeNode.type === 'SURPLUS'
                  ? 'Savings are the difference between recorded income and spending.'
                  : 'The shortfall is spending beyond recorded income for this period.'}{' '}
                This is a calculated total, with no individual transaction of its own.
              </p>
            ) : isNodeError ? (
              <QueryError message="Unable to load supporting transactions" onRetry={retryNode} />
            ) : isNodeTxsLoading ? (
              <div className="flex h-40 items-center justify-center text-xs text-muted-foreground">
                <RefreshCw className="size-4 animate-spin mr-2" />
                Loading node transactions...
              </div>
            ) : !nodeTxs || nodeTxs.length === 0 ? (
              <div className="flex h-32 items-center justify-center text-xs text-muted-foreground">
                No individual ledger records found for this node
              </div>
            ) : (
              <div className="divide-y divide-border/60 text-xs">
                {nodeTxs.slice((nodePage - 1) * 50, nodePage * 50).map((tx: Transaction) => (
                  <div
                    key={tx.id}
                    className="flex items-center justify-between py-2.5 px-1 hover:bg-muted/40 transition-colors"
                  >
                    <div className="min-w-0 pr-3">
                      <div className="font-semibold text-foreground truncate">
                        {isPrivacyMode ? 'Payee hidden' : tx.cleaned_payee || tx.raw_narration}
                      </div>
                      <div className="text-[11px] text-muted-foreground flex items-center gap-2 mt-0.5">
                        <span>{formatDate(tx.tx_date)}</span>
                        {tx.payment_mode && (
                          <Badge variant="secondary" className="px-1 py-0">
                            {tx.payment_mode}
                          </Badge>
                        )}
                        {tx.account_name && (
                          <span className="truncate opacity-75">
                            {isPrivacyMode ? 'Account hidden' : tx.account_name}
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="text-right shrink-0">
                      <div
                        className={cn(
                          'font-bold',
                          tx.tx_type === 'CREDIT' ? 'text-success ' : 'text-foreground'
                        )}
                      >
                        {tx.tx_type === 'CREDIT' ? '+' : '-'}
                        {money(tx.amount)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          {!selectedCategory && nodeTxs && nodeTxs.length > 0 && (
            <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-3 text-xs">
              <span>
                {nodeTxs.length} contributing transactions · Page {nodePage} of{' '}
                {Math.ceil(nodeTxs.length / 50)}
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={nodePage === 1}
                  onClick={() => setNodePage(nodePage - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={nodePage * 50 >= nodeTxs.length}
                  onClick={() => setNodePage(nodePage + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
