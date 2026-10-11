import { useMemo } from 'react'
import type { InvestmentSnapshot } from '@/types/investments'
import { investmentTotals } from '@/lib/investment-summary'
import { formatMoney } from '@/lib/formatters'
import { usePrivacy } from '@/components/privacy-provider'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

export function InvestmentTotals({ snapshots }: { snapshots: InvestmentSnapshot[] }) {
  const totals = useMemo(() => investmentTotals(snapshots), [snapshots])
  const { isPrivacyMode } = usePrivacy()
  const money = (value: number, currency: string) => formatMoney(value, currency, isPrivacyMode)
  if (!totals.length) return null
  return (
    <section aria-label="Investment totals by currency" className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">Total investment holdings</h2>
      <p className="text-xs text-muted-foreground">
        Latest statement per account. Currencies stay separate; these dated values are separate from
        bank balances, income, and expenses.
      </p>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {totals.map((total) => (
          <Card key={total.currency}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">
                Current holdings in {total.currency}
              </CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
              <p className="text-2xl font-semibold">
                {total.currentValue === null
                  ? 'Not provided'
                  : money(total.currentValue, total.currency)}
              </p>
              <p className="text-sm">
                Invested amount:{' '}
                {total.investedValue === null
                  ? 'Not provided'
                  : money(total.investedValue, total.currency)}
              </p>
              <p className="text-xs text-muted-foreground">
                {total.accounts} {total.accounts === 1 ? 'account' : 'accounts'} · {total.holdings}{' '}
                holdings ·{' '}
                {total.oldestDate === total.newestDate
                  ? `As of ${total.newestDate}`
                  : `Statement dates ${total.oldestDate} to ${total.newestDate}`}
              </p>
              {total.missingValuations > 0 && (
                <p className="text-xs text-muted-foreground">
                  {total.missingValuations}{' '}
                  {total.missingValuations === 1 ? 'account has' : 'accounts have'} no market
                  valuation.
                  {total.missingValuations < total.accounts &&
                    ` Known value: ${money(total.knownCurrentValue, total.currency)}. The full total is unavailable.`}
                </p>
              )}
              {total.missingCosts > 0 && (
                <p className="text-xs text-muted-foreground">
                  {total.missingCosts} {total.missingCosts === 1 ? 'account has' : 'accounts have'}{' '}
                  no acquisition cost.
                  {total.missingCosts < total.accounts &&
                    ` Known invested amount: ${money(total.knownInvestedValue, total.currency)}. The full invested total is unavailable.`}
                </p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  )
}
