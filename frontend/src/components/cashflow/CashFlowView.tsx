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
import { QueryError, PageLoading } from '@/components/ui/query-state'
import { usePrivacy } from '@/components/privacy-provider'
import React, { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
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
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'

export const CashFlowView: React.FC = () => {
  const searchParams = (useSearch({ strict: false }) as CashFlowSearchParams) || {}
  const navigate = useNavigate()
  const { isPrivacyMode } = usePrivacy()
  const money = (amount: number) => (isPrivacyMode ? '₹••••••' : formatINR(amount, isPrivacyMode))
  const selectedPeriod = searchParams.month || searchParams.period || ''

  const handlePeriodChange = useCallback(
    (newPeriod: string) => {
      navigate({
        to: '/cashflow',
        search: newPeriod ? { month: newPeriod } : {},
        replace: false,
      })
    },
    [navigate]
  )

  const [activeNode, setActiveNode] = useState<SankeyNode | null>(null)
  const [isDrilldownOpen, setIsDrilldownOpen] = useState(false)

  // 1. Fetch Cash Flow Intelligence Data
  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['cashflow-intelligence', selectedPeriod],
    queryFn: () => fetchCashFlowIntelligence(selectedPeriod || undefined),
  })

  const [nodePage, setNodePage] = useState(1)
  const {
    data: nodeTxs,
    isLoading: isNodeTxsLoading,
    isError: isNodeError,
    refetch: retryNode,
  } = useQuery({
    queryKey: ['node-transactions', activeNode?.id, data?.selected_month, data?.available_months],
    queryFn: async () => {
      if (!activeNode || !data || ['SURPLUS', 'DEFICIT'].includes(activeNode.type)) return []
      const transactions = await fetchAllTransactions({
        ...cashFlowPeriodDates(data.selected_month, data.available_months),
        ...cashFlowNodeParams(activeNode),
      })
      return contributingTransactions(activeNode, transactions)
    },
    enabled: !!activeNode && isDrilldownOpen,
  })

  const handleNodeClick = (node: SankeyNode) => {
    setNodePage(1)
    setActiveNode(node)
    setIsDrilldownOpen(true)
  }

  const handleCategoryRowClick = (categoryId: string) => {
    setNodePage(1)
    const node = data?.sankey.nodes.find((n) => n.id === `cat_${categoryId}`)
    if (node) {
      setActiveNode(node)
    } else {
      setActiveNode({
        id: `cat_${categoryId}`,
        name:
          data?.category_comparisons.find((item) => item.category_id === categoryId)
            ?.category_name || categoryId,
        type: 'CATEGORY',
        color_hex: '#3B82F6',
        total_value: 0,
        layer: 3,
      })
    }
    setIsDrilldownOpen(true)
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
        <div className="flex items-center gap-2 bg-card/60 border rounded-lg p-1 shadow-xs">
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
            onClick={() => refetch()}
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

      {(!selectedPeriod || /^\d{4}-\d{2}$/.test(selectedPeriod)) && (
        <MonthlyReview month={selectedPeriod || undefined} onMonthChange={handlePeriodChange} />
      )}

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
      <AnomaliesSection
        anomalies={data?.anomalies || []}
        previousMonth={previousMonth}
        selectedMonth={currentSelectedMonth}
      />

      {/* Category Shifts & 6-Month Trajectory Table */}
      <CategoryComparisonTable
        items={data?.category_comparisons || []}
        selectedMonth={currentSelectedMonth}
        previousMonth={previousMonth}
        totalOutflow={summary.total_outflow}
        onCategoryClick={handleCategoryRowClick}
      />

      {/* Drilldown Modal for Selected Node */}
      <Dialog open={isDrilldownOpen} onOpenChange={setIsDrilldownOpen}>
        <DialogContent className="max-w-2xl max-h-[85dvh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center justify-between pr-6">
              <div className="flex items-center gap-2.5">
                <div
                  className="size-3 rounded-full"
                  style={{ backgroundColor: activeNode?.color_hex || '#3B82F6' }}
                />
                <span>{activeNode?.name} Transactions</span>
              </div>
              <Badge variant="outline">{activeNode && money(activeNode.total_value)}</Badge>
            </DialogTitle>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto mt-3">
            {activeNode && ['SURPLUS', 'DEFICIT'].includes(activeNode.type) ? (
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
                        {tx.cleaned_payee || tx.raw_narration}
                      </div>
                      <div className="text-[11px] text-muted-foreground flex items-center gap-2 mt-0.5">
                        <span>{formatDate(tx.tx_date)}</span>
                        {tx.payment_mode && (
                          <Badge variant="secondary" className="px-1 py-0">
                            {tx.payment_mode}
                          </Badge>
                        )}
                        {tx.account_name && (
                          <span className="truncate opacity-75">{tx.account_name}</span>
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
          {nodeTxs && nodeTxs.length > 0 && (
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
