import { Link } from '@tanstack/react-router'
import type { CreditCardBill } from '@/types'
import type {
  CalendarPreferences,
  SourceMonth,
  SourceTracking,
} from '@/lib/import-calendar'
import { Button, buttonVariants } from '@/components/ui/button'
import { CalendarStatus } from './CalendarStatus'
import { StatementImportRow } from './StatementImportRow'
import { TrackingControl } from './TrackingControl'

export function MonthSourceDetail({
  item,
  month,
  today,
  name,
  isPrivacyMode,
  bills,
  preferences,
  onAssignMonth,
  onTracking,
  onToggleNoActivity,
}: {
  item: SourceMonth
  month: string
  today: string
  name: string
  isPrivacyMode: boolean
  bills: CreditCardBill[]
  preferences: CalendarPreferences
  onAssignMonth: (id: string, month: string) => void
  onTracking: (patch: SourceTracking) => void
  onToggleNoActivity: () => void
}) {
  const { account, status, imports, from, until, unknown } = item
  return (
    <div className="flex flex-col gap-3 py-4 first:pt-0">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-col gap-1">
          <h3 className="font-medium">{name}</h3>
          <CalendarStatus status={status} />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {[
            'missing',
            'partial',
            'pending',
            'unknown',
            'unconfigured',
          ].includes(status) && (
            <Link
              to="/import"
              data-slot="button"
              aria-label={`Import statement for ${name}`}
              className={buttonVariants({ variant: 'outline', size: 'sm' })}
            >
              Import statement
            </Link>
          )}
          {!imports.length &&
            month <= today.slice(0, 7) &&
            status !== 'not-tracked' && (
              <Button
                variant="ghost"
                size="sm"
                aria-label={`${status === 'no-activity' ? 'Undo no activity' : 'Mark no activity'} for ${name} in ${month}`}
                onClick={onToggleNoActivity}
              >
                {status === 'no-activity'
                  ? 'Undo no activity'
                  : 'Mark no activity'}
              </Button>
            )}
          <TrackingControl
            name={name}
            from={from}
            until={until}
            onSave={onTracking}
          />
        </div>
      </div>
      {imports.map((statement, index) => (
        <div key={statement.id} className="sm:pl-5">
          <StatementImportRow
            statement={statement}
            label={
              isPrivacyMode
                ? `${name}, import ${index + 1}`
                : `${statement.filename}, ${name}, import ${index + 1}`
            }
            isPrivacyMode={isPrivacyMode}
            month={preferences.importMonths[statement.id]}
            today={today}
            statementDate={
              account.account_type === 'CREDIT_CARD'
                ? bills.find(
                    (bill) => bill.statement_import_id === statement.id,
                  )?.statement_date
                : undefined
            }
            onAssignMonth={onAssignMonth}
          />
        </div>
      ))}
      {status === 'unknown' && (
        <p className="text-sm text-muted-foreground">
          {unknown.length} {unknown.length === 1 ? 'file needs' : 'files need'}{' '}
          a month. Assign the undated files below before deciding whether this
          month’s import is missing.
        </p>
      )}
      {status === 'unconfigured' && (
        <p className="text-sm text-muted-foreground">
          No imports yet. Set a tracking start to check for missing months.
        </p>
      )}
      {status === 'partial' && (
        <p className="text-xs text-muted-foreground">
          The recorded ranges cover only part of this month. Import the
          remaining period if available.
        </p>
      )}
    </div>
  )
}
