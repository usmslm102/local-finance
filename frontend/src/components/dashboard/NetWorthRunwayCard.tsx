import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import React from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { formatINR } from '@/lib/utils'
import { ShieldCheck, Clock, Wallet, CreditCard } from 'lucide-react'

interface NetWorthRunwayCardProps {
  totalLiquidity: number
  totalCreditDue: number
  monthlyExpenses: number[]
}

export const NetWorthRunwayCard: React.FC<NetWorthRunwayCardProps> = ({
  totalLiquidity,
  totalCreditDue,
  monthlyExpenses,
}) => {
  const { isPrivacyMode } = usePrivacy()
  const netWorth = totalLiquidity - totalCreditDue

  // Calculate average monthly expense (needs / baseline burn)
  const validExpenses = monthlyExpenses.filter((e) => e > 0)
  const avgMonthlyBurn =
    validExpenses.length > 0
      ? validExpenses.reduce((acc, v) => acc + v, 0) / validExpenses.length
      : 35000 // reasonable default baseline

  const runwayMonths = avgMonthlyBurn > 0 ? totalLiquidity / avgMonthlyBurn : 0

  // Calculate Debt-to-Liquid ratio
  const debtRatio = totalLiquidity > 0 ? (totalCreditDue / totalLiquidity) * 100 : 0

  // Runway health assessment
  let runwayStatusColor = 'text-success'
  let runwayBadgeVariant: 'default' | 'secondary' | 'outline' = 'default'
  let runwayLabel = 'Exceptional Cushion (6+ Months)'
  let gaugePercent = Math.min(100, (runwayMonths / 6) * 100)

  if (runwayMonths < 1.0) {
    runwayStatusColor = 'text-destructive'
    runwayLabel = 'Critical Runway (< 1 Month)'
    gaugePercent = Math.max(10, runwayMonths * 100)
  } else if (runwayMonths < 3.0) {
    runwayStatusColor = 'text-warning'
    runwayLabel = 'Lean Buffer (1-3 Months)'
    gaugePercent = (runwayMonths / 6) * 100
  } else if (runwayMonths < 6.0) {
    runwayStatusColor = 'text-info'
    runwayLabel = 'Healthy Runway (3-6 Months)'
    gaugePercent = (runwayMonths / 6) * 100
  }

  return (
    <Card className="overflow-hidden">
      <CardContent className="p-5 flex flex-col gap-4">
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border/50 pb-3">
          <div className="flex items-center gap-2">
            <div className="flex size-7 items-center justify-center rounded-md bg-success/10 text-success">
              <ShieldCheck className="size-4" />
            </div>
            <div>
              <h3 className="text-xs font-bold uppercase tracking-wider text-foreground">
                Liquid Net Worth &amp; Emergency Runway
              </h3>
            </div>
          </div>

          <Badge variant={runwayBadgeVariant} className="px-2 py-0.5">
            {runwayLabel}
          </Badge>
        </div>

        {/* 2-Column Gauge Section */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
          {/* Left: Net Worth Figure & Debt Breakdown */}
          <div className="flex flex-col gap-3">
            <div>
              <span className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground block">
                Total Liquid Net Worth
              </span>
              <div className="text-3xl font-extrabold font-mono tracking-tight text-foreground tabular-nums privacy-blur mt-0.5">
                {formatINR(netWorth, isPrivacyMode)}
              </div>
              <p className="text-[11px] text-muted-foreground mt-1">
                Liquid Bank Savings minus active Credit Card Dues
              </p>
            </div>

            {/* Inflow vs Debt Quick Chips */}
            <div className="flex flex-wrap gap-2 pt-1">
              <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/60 border text-[11px] font-mono">
                <Wallet className="size-3 text-success" />
                <span className="text-muted-foreground">Bank Balances:</span>
                <span className="font-bold text-foreground privacy-blur">
                  {formatINR(totalLiquidity, isPrivacyMode)}
                </span>
              </div>

              <div className="flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted/60 border text-[11px] font-mono">
                <CreditCard className="size-3 text-destructive" />
                <span className="text-muted-foreground">Card Dues:</span>
                <span className="font-bold text-destructive privacy-blur">
                  {formatINR(totalCreditDue, isPrivacyMode)}
                </span>
              </div>
            </div>
          </div>

          {/* Right: Emergency Runway Calculation */}
          <div className="flex flex-col gap-3 rounded-xl border bg-muted/20 p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1.5 text-xs font-semibold text-foreground">
                <Clock className="size-3.5 text-primary" />
                <span>Survival Runway without Income</span>
              </div>
              <span
                className={cn('text-sm font-bold font-mono', runwayStatusColor, 'privacy-blur')}
              >
                {runwayMonths.toFixed(1)} Months
              </span>
            </div>

            {/* Visual Runway Progress Bar */}
            <div className="flex flex-col gap-1">
              <div className="h-2.5 w-full bg-muted/60 rounded-full overflow-hidden p-0.5 border">
                <div
                  className={cn(
                    'h-full rounded-full transition-all duration-500',
                    runwayMonths >= 6
                      ? 'bg-success'
                      : runwayMonths >= 3
                        ? 'bg-info'
                        : runwayMonths >= 1
                          ? 'bg-warning'
                          : 'bg-destructive'
                  )}
                  style={{ width: `${gaugePercent}%` }}
                />
              </div>
              <div className="flex justify-between text-[9px] font-mono text-muted-foreground pt-0.5">
                <span>0 mo</span>
                <span>3 mo (Baseline)</span>
                <span>6+ mo (Ideal)</span>
              </div>
            </div>

            {/* Debt-to-Liquid Ratio Footer */}
            <div className="flex items-center justify-between text-[11px] text-muted-foreground pt-1 border-t border-border/50">
              <span>Card Debt Float Ratio:</span>
              <span className="font-mono font-semibold text-foreground privacy-blur">
                {debtRatio.toFixed(1)}% of savings
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
