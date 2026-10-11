import { Link } from '@tanstack/react-router'
import type { StatementImport } from '@/types'
import { validDate } from '@/lib/import-calendar'
import { formatDate } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'
import { ImportMonthControl } from './ImportMonthControl'

export function StatementImportRow({
  statement,
  label,
  isPrivacyMode,
  month,
  today,
  statementDate,
  onAssignMonth,
}: {
  statement: StatementImport
  label: string
  isPrivacyMode: boolean
  month?: string
  today: string
  statementDate?: string
  onAssignMonth: (id: string, month: string) => void
}) {
  const validPeriod =
    validDate(statement.start_date) &&
    validDate(statement.end_date) &&
    statement.start_date <= statement.end_date
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0 flex-1">
        <p className="break-words text-sm">
          {isPrivacyMode ? 'Filename hidden' : statement.filename}
        </p>
        <p className="text-xs text-muted-foreground">
          {validPeriod
            ? `${formatDate(statement.start_date!)} – ${formatDate(statement.end_date!)}`
            : 'Period unknown'}
          {month ? ' · Month confirmed by you' : ''} ·{' '}
          {statement.total_transactions} records
          {validDate(statementDate)
            ? ` · Statement dated ${formatDate(statementDate)}`
            : ''}
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <ImportMonthControl
          label={label}
          month={month}
          today={today}
          onSave={(next) => onAssignMonth(statement.id, next)}
        />
        <Link
          to="/transactions"
          search={{ statement: statement.id }}
          data-slot="button"
          aria-label={`View import for ${label}`}
          className={buttonVariants({ variant: 'outline', size: 'sm' })}
        >
          View import
        </Link>
      </div>
    </div>
  )
}
