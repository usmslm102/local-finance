import React from 'react'
import { PieChart, Pie, Cell } from 'recharts'
import type { CategorySpend } from '@/types'
import { formatINR } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart'
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from '@/components/ui/empty'

export const SpendingChart: React.FC<{ data: CategorySpend[] }> = ({ data }) => {
  const { isPrivacyMode } = usePrivacy()
  const chartData = (data ?? [])
    .filter((item) => item.total_amount > 0)
    .map((item, index) => ({
      name: item.category_name,
      value: item.total_amount,
      percentage: item.percentage,
      color: item.color_hex || `var(--chart-${(index % 5) + 1})`,
    }))
  return (
    <Card>
      <CardHeader>
        <CardTitle>Spending by category</CardTitle>
        <CardDescription>All recorded spending categories across your accounts</CardDescription>
      </CardHeader>
      <CardContent>
        {!chartData.length ? (
          <Empty className="h-60">
            <EmptyHeader>
              <EmptyTitle>No spending yet</EmptyTitle>
              <EmptyDescription>
                Import a statement to see your category breakdown.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className="grid items-center gap-4 sm:grid-cols-2">
            <ChartContainer
              className="h-60 w-full aspect-auto"
              config={{ value: { label: 'Spent' } }}
            >
              <PieChart>
                <Pie
                  data={chartData}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={55}
                  outerRadius={85}
                  paddingAngle={2}
                  isAnimationActive={false}
                >
                  {chartData.map((item) => (
                    <Cell key={item.name} fill={item.color} stroke="var(--card)" strokeWidth={2} />
                  ))}
                </Pie>
                <ChartTooltip
                  content={
                    <ChartTooltipContent
                      nameKey="name"
                      formatter={(value, name) => (
                        <div className="flex w-full justify-between gap-3">
                          <span>{name}</span>
                          <span className="tabular-nums">
                            {formatINR(Number(value), isPrivacyMode)}
                          </span>
                        </div>
                      )}
                    />
                  }
                />
              </PieChart>
            </ChartContainer>
            <ul
              className="flex max-h-60 min-w-0 flex-col gap-2 overflow-y-auto"
              aria-label="Spending category breakdown"
            >
              {chartData.map((item) => (
                <li key={item.name} className="flex items-start justify-between gap-2 text-xs">
                  <span className="flex min-w-0 items-start gap-2">
                    <span
                      className="mt-1 size-2 shrink-0 rounded-full"
                      style={{ backgroundColor: item.color }}
                    />
                    <span className="break-words">{item.name}</span>
                  </span>
                  <span className="flex shrink-0 flex-col items-end tabular-nums">
                    <span>{formatINR(item.value, isPrivacyMode)}</span>
                    <span className="text-muted-foreground">{item.percentage.toFixed(0)}%</span>
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
