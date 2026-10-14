import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { usePrivacy } from '@/components/privacy-provider'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'
import React, { useState } from 'react'
import { CategoryComparisonItem } from '@/types'
import { formatINR } from '@/lib/utils'
import { ArrowUpRight, ArrowDownRight, ArrowRight, Search, SlidersHorizontal } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

interface CategoryComparisonTableProps {
  items: CategoryComparisonItem[]
  selectedMonth: string
  previousMonth: string
  totalOutflow: number
  onCategoryClick?: (categoryId: string) => void
}

export const CategoryComparisonTable: React.FC<CategoryComparisonTableProps> = ({
  items,
  selectedMonth,
  previousMonth,
  totalOutflow,
  onCategoryClick,
}) => {
  const { isPrivacyMode } = usePrivacy()
  const money = (amount: number) => (isPrivacyMode ? '₹••••••' : formatINR(amount, isPrivacyMode))
  const [search, setSearch] = useState('')
  const [sortBy, setSortBy] = useState<'spend' | 'delta_asc' | 'delta_desc' | 'pct'>('spend')

  const filteredItems = items
    .filter(
      (it) =>
        it.category_name.toLowerCase().includes(search.toLowerCase()) ||
        (it.top_payees &&
          it.top_payees.some((p) => p.payee.toLowerCase().includes(search.toLowerCase())))
    )
    .sort((a, b) => {
      if (sortBy === 'spend') return b.current_spend - a.current_spend
      if (sortBy === 'delta_desc') return b.delta_amount - a.delta_amount
      if (sortBy === 'delta_asc') return a.delta_amount - b.delta_amount
      if (sortBy === 'pct') return b.percentage_change - a.percentage_change
      return 0
    })

  // Helper to render mini sparkline
  const renderSparkline = (history: { month: string; amount: number }[], color: string) => {
    if (!history || history.length < 2) {
      return <span className="text-[10px] text-muted-foreground">-</span>
    }

    const width = 80
    const height = 22
    const maxVal = Math.max(...history.map((h) => h.amount), 1)
    const minVal = Math.min(...history.map((h) => h.amount), 0)
    const range = maxVal - minVal || 1

    const points = history
      .map((h, i) => {
        const x = (i / (history.length - 1)) * (width - 4) + 2
        const y = height - 2 - ((h.amount - minVal) / range) * (height - 6)
        return `${x.toFixed(1)},${y.toFixed(1)}`
      })
      .join(' ')

    return (
      <svg width={width} height={height} className="overflow-visible">
        <polyline
          fill="none"
          stroke={color || '#3B82F6'}
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={points}
        />
        {history.map((h, i) => {
          const x = (i / (history.length - 1)) * (width - 4) + 2
          const y = height - 2 - ((h.amount - minVal) / range) * (height - 6)
          return (
            <circle
              key={i}
              cx={x}
              cy={y}
              r={i === history.length - 1 ? 2.5 : 1}
              fill={color || '#3B82F6'}
            />
          )
        })}
      </svg>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
        <div>
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
            <SlidersHorizontal className="size-3.5 text-primary" />
            Category Shifts &amp; 6-Month Trajectory
          </h3>
          <p className="text-[11px] text-muted-foreground mt-0.5">
            Compare category spending against {previousMonth} and 3-month rolling averages
          </p>
        </div>

        <div className="flex flex-col lg:flex-row items-stretch gap-2 w-full lg:w-auto">
          <div className="relative flex-1 lg:w-56">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
            <Input
              aria-label="Search category / payee"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search category / payee..."
              className="h-8 pl-8"
            />
          </div>

          <ToggleGroup
            value={[sortBy]}
            onValueChange={(values) => {
              if (values[0]) setSortBy(values[0] as typeof sortBy)
            }}
            aria-label="Sort category changes"
            size="sm"
            variant="outline"
          >
            <ToggleGroupItem value="spend">Highest spend</ToggleGroupItem>
            <ToggleGroupItem value="delta_desc">Surges</ToggleGroupItem>
            <ToggleGroupItem value="delta_asc">Drops</ToggleGroupItem>
          </ToggleGroup>
        </div>
      </div>

      <div className="rounded-xl border bg-card/60 backdrop-blur-xs overflow-hidden shadow-xs">
        <div className="overflow-x-auto">
          <Table className="w-full text-left text-xs">
            <TableHeader className="bg-muted/50 border-b text-[11px] font-bold uppercase tracking-wider text-muted-foreground">
              <TableRow>
                <TableHead className="py-3 px-4">Category</TableHead>
                <TableHead className="py-3 px-4 text-right">Spend ({selectedMonth})</TableHead>
                <TableHead className="py-3 px-4 text-right">Share</TableHead>
                <TableHead className="py-3 px-4 text-right">MoM Shift</TableHead>
                <TableHead className="py-3 px-4 text-right">3-Mo Avg</TableHead>
                <TableHead className="py-3 px-4 text-center">6-Mo Trend</TableHead>
                <TableHead className="py-3 px-4">Top Payees</TableHead>
                <TableHead className="py-3 px-4">Purchases</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody className="divide-y divide-border/60">
              {filteredItems.length === 0 && (
                <TableRow>
                  <TableCell colSpan={8} className="py-8 text-center">
                    No categories match your search.
                  </TableCell>
                </TableRow>
              )}
              {filteredItems.map((item) => {
                const sharePct = totalOutflow > 0 ? (item.current_spend / totalOutflow) * 100 : 0

                return (
                  <TableRow
                    key={item.category_id}
                    onClick={() => onCategoryClick?.(item.category_id)}
                    tabIndex={onCategoryClick ? 0 : undefined}
                    onKeyDown={(event) => {
                      if (
                        event.target === event.currentTarget &&
                        (event.key === 'Enter' || event.key === ' ')
                      ) {
                        event.preventDefault()
                        onCategoryClick?.(item.category_id)
                      }
                    }}
                    className="outline-none focus-visible:ring-2 focus-visible:ring-ring hover:bg-muted/40 transition-colors cursor-pointer"
                  >
                    {/* Category */}
                    <TableCell className="py-3 px-4 font-semibold text-foreground">
                      <div className="flex items-center gap-2.5">
                        <div
                          className="size-2.5 rounded-full shrink-0"
                          style={{ backgroundColor: item.category_color }}
                        />
                        <span className="truncate">{item.category_name}</span>
                      </div>
                    </TableCell>

                    {/* Spend */}
                    <TableCell className="py-3 px-4 text-right font-bold text-foreground">
                      {money(item.current_spend)}
                    </TableCell>

                    {/* Share */}
                    <TableCell className="py-3 px-4 text-right text-muted-foreground">
                      {sharePct.toFixed(1)}%
                    </TableCell>

                    {/* MoM Shift */}
                    <TableCell className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5 font-semibold">
                        {item.delta_amount > 0 ? (
                          <span className="flex items-center gap-0.5 text-warning">
                            <ArrowUpRight className="size-3.5 shrink-0" />+
                            {money(item.delta_amount)}
                            <span className="text-[10px] font-normal opacity-85">
                              (+{item.percentage_change.toFixed(0)}%)
                            </span>
                          </span>
                        ) : item.delta_amount < 0 ? (
                          <span className="flex items-center gap-0.5 text-success">
                            <ArrowDownRight className="size-3.5 shrink-0" />-
                            {money(Math.abs(item.delta_amount))}
                            <span className="text-[10px] font-normal opacity-85">
                              ({item.percentage_change.toFixed(0)}%)
                            </span>
                          </span>
                        ) : (
                          <span className="flex items-center gap-0.5 text-muted-foreground">
                            <ArrowRight className="size-3 shrink-0" />
                            {money(0)} (0%)
                          </span>
                        )}
                      </div>
                    </TableCell>

                    {/* 3-Month Average */}
                    <TableCell className="py-3 px-4 text-right text-muted-foreground font-medium">
                      {item.three_month_avg > 0 ? money(item.three_month_avg) : '-'}
                    </TableCell>

                    {/* 6-Month Sparkline */}
                    <TableCell className="py-3 px-4 text-center">
                      <div className="flex justify-center">
                        {renderSparkline(item.history, item.category_color)}
                      </div>
                    </TableCell>

                    {/* Top Payees */}
                    <TableCell className="py-3 px-4">
                      {item.top_payees && item.top_payees.length > 0 ? (
                        <div className="flex flex-wrap gap-1">
                          {item.top_payees.map((p, idx) => (
                            <Badge key={idx} variant="secondary" className="px-1.5 py-0">
                              {isPrivacyMode ? 'Payee hidden' : p.payee}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        <span className="text-muted-foreground text-[11px]">-</span>
                      )}
                    </TableCell>
                    <TableCell className="py-3 px-4">
                      <Button variant="outline" size="sm" aria-label={`View ${item.category_name} purchases`}
                        onClick={(event) => { event.stopPropagation(); onCategoryClick?.(item.category_id) }}>View purchases</Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </div>
      </div>
    </div>
  )
}
