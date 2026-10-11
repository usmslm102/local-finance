import React from 'react'
import { BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts'
import type { MonthlyCashFlow } from '@/types'
import { formatINR } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
} from '@/components/ui/chart'
import { Empty, EmptyHeader, EmptyTitle, EmptyDescription } from '@/components/ui/empty'

export const CashFlowChart: React.FC<{ data: MonthlyCashFlow[] }> = ({ data }) => {
  const { isPrivacyMode } = usePrivacy()
  return (
    <Card>
      <CardHeader>
        <CardTitle>Monthly cash flow</CardTitle>
        <CardDescription>Income and spending by recorded month</CardDescription>
      </CardHeader>
      <CardContent>
        {!data?.length ? (
          <Empty className="h-60">
            <EmptyHeader>
              <EmptyTitle>No cash flow yet</EmptyTitle>
              <EmptyDescription>
                Import a statement to compare income and spending.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ChartContainer
            className="h-60 w-full aspect-auto"
            config={{
              income: { label: 'Income', color: 'var(--success)' },
              expense: { label: 'Spending', color: 'var(--destructive)' },
            }}
          >
            <BarChart
              accessibilityLayer
              data={data}
              margin={{ top: 10, right: 10, left: -15, bottom: 0 }}
            >
              <CartesianGrid vertical={false} />
              <XAxis dataKey="month" tickLine={false} axisLine={false} minTickGap={24} />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) =>
                  isPrivacyMode
                    ? '••••'
                    : `₹${value >= 1000 ? `${(value / 1000).toFixed(0)}k` : value}`
                }
              />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    formatter={(value, name) => (
                      <div className="flex w-full justify-between gap-4">
                        <span className="text-muted-foreground">
                          {name === 'income' ? 'Income' : 'Spending'}
                        </span>
                        <span className="tabular-nums">
                          {formatINR(Number(value), isPrivacyMode)}
                        </span>
                      </div>
                    )}
                  />
                }
              />
              <ChartLegend content={<ChartLegendContent />} />
              <Bar
                dataKey="income"
                fill="var(--color-income)"
                radius={4}
                maxBarSize={28}
                isAnimationActive={false}
              />
              <Bar
                dataKey="expense"
                fill="var(--color-expense)"
                radius={4}
                maxBarSize={28}
                isAnimationActive={false}
              />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
