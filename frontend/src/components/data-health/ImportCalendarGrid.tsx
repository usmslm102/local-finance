import { Check, CircleAlert } from 'lucide-react'
import type { Account } from '@/types'
import type { CalendarMonth } from '@/lib/import-calendar'
import { monthName } from '@/lib/planning'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

export function ImportCalendarGrid({
  months,
  year,
  selectedMonth,
  detailsId,
  onSelectMonth,
  sourceName,
}: {
  months: CalendarMonth[]
  year: string
  selectedMonth: string
  detailsId: string
  onSelectMonth: (month: string) => void
  sourceName: (account: Account) => string
}) {
  return (
    <div
      role="group"
      className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4"
      aria-label={`Import calendar for ${year}`}
    >
      {months.map(({ month, items, summary }) => {
        const problems = items.filter(
          (item) => item.status === 'missing' || item.status === 'partial',
        )
        const needsDates = items.some(
          (item) => item.status === 'unknown' || item.status === 'unconfigured',
        )
        return (
          <Button
            key={month}
            variant={selectedMonth === month ? 'secondary' : 'outline'}
            aria-pressed={selectedMonth === month}
            aria-controls={detailsId}
            onClick={() => onSelectMonth(month)}
            className="h-auto min-h-28 min-w-0 flex-col items-start justify-start gap-3 p-4 whitespace-normal text-left"
          >
            <span className="flex w-full items-center justify-between gap-2">
              <span>{monthName(month).split(' ')[0]}</span>
              {summary === 'All imported' && (
                <Check aria-hidden="true" className="text-success" />
              )}
              {(problems.length > 0 || needsDates) && (
                <CircleAlert aria-hidden="true" className="text-warning" />
              )}
            </span>
            <span
              className={cn(
                'text-sm',
                problems.length || needsDates
                  ? 'text-warning'
                  : summary === 'All imported'
                    ? 'text-success'
                    : 'text-muted-foreground',
              )}
            >
              {problems.length === 1
                ? `${sourceName(problems[0].account)} · ${problems[0].status === 'missing' ? 'Missing' : 'Partial'}`
                : summary}
            </span>
            {items.some((item) => item.imports.length) && (
              <span className="text-xs text-muted-foreground">
                {items.reduce((total, item) => total + item.imports.length, 0)}{' '}
                {items.reduce(
                  (total, item) => total + item.imports.length,
                  0,
                ) === 1
                  ? 'import'
                  : 'imports'}
              </span>
            )}
          </Button>
        )
      })}
    </div>
  )
}
