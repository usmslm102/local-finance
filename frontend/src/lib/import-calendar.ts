import type { Account, CreditCardBill, StatementImport } from '@/types'

export interface SourceTracking {
  from?: string
  until?: string
  noActivity?: string[]
}
export interface CalendarPreferences {
  sources: Record<string, SourceTracking>
  importMonths: Record<string, string>
}
export type ImportStatus =
  | 'imported'
  | 'partial'
  | 'missing'
  | 'no-activity'
  | 'not-tracked'
  | 'pending'
  | 'upcoming'
  | 'unknown'
  | 'unconfigured'

export function validMonth(value?: string): value is string {
  return !!value && /^\d{4}-(0[1-9]|1[0-2])$/.test(value)
}
export function validDate(value?: string): value is string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const date = new Date(`${value}T00:00:00Z`)
  return (
    Number.isFinite(date.getTime()) && date.toISOString().slice(0, 10) === value
  )
}
function range(statement: StatementImport) {
  return validDate(statement.start_date) &&
    validDate(statement.end_date) &&
    statement.start_date <= statement.end_date
    ? { start: statement.start_date, end: statement.end_date }
    : null
}

// A card cycle belongs to one month. Prefer the dated bill over transaction-range
// fallbacks, which some parsers use when the statement has no explicit period.
export function importMonth(
  statement: StatementImport,
  bills: CreditCardBill[],
  preferences: CalendarPreferences,
) {
  const assigned = preferences.importMonths[statement.id]
  if (validMonth(assigned)) return assigned
  const bill = bills.find((bill) => bill.statement_import_id === statement.id)
  if (validDate(bill?.statement_date)) return bill.statement_date.slice(0, 7)
  return range(statement)?.end.slice(0, 7) ?? ''
}

export const CARD_IMPORT_GRACE_DAYS = 3

function isImportPending(account: Account, month: string, today: string) {
  if (account.account_type !== 'CREDIT_CARD' || !account.billing_cycle_day)
    return month === today.slice(0, 7)
  const monthEnd = new Date(`${month}-01T00:00:00Z`)
  monthEnd.setUTCMonth(monthEnd.getUTCMonth() + 1, 0)
  const deadline = new Date(`${month}-01T00:00:00Z`)
  deadline.setUTCDate(
    Math.min(Math.max(account.billing_cycle_day, 1), monthEnd.getUTCDate()) +
      CARD_IMPORT_GRACE_DAYS,
  )
  return today <= deadline.toISOString().slice(0, 10)
}

export type SourceMonth = ReturnType<typeof sourceMonth>
export interface CalendarMonth {
  month: string
  items: SourceMonth[]
  summary: string
}

export function sourceMonth(
  account: Account,
  statements: StatementImport[],
  bills: CreditCardBill[],
  preferences: CalendarPreferences,
  month: string,
  today: string,
) {
  const source = statements.filter(
    (statement) => statement.account_id === account.id,
  )
  const tracking = preferences.sources[account.id] ?? {}
  const card = account.account_type === 'CREDIT_CARD'
  const knownMonths = source
    .flatMap((statement) => {
      const assigned = preferences.importMonths[statement.id]
      if (validMonth(assigned)) return [assigned]
      const period = range(statement)
      if (card)
        return [importMonth(statement, bills, preferences)].filter(Boolean)
      return period ? [period.start.slice(0, 7)] : []
    })
    .sort()
  const from = validMonth(tracking.from) ? tracking.from : knownMonths[0]
  const until = validMonth(tracking.until) ? tracking.until : ''
  const imports = source
    .filter((statement) => {
      const assigned = preferences.importMonths[statement.id]
      if (validMonth(assigned) || card)
        return importMonth(statement, bills, preferences) === month
      const period = range(statement)
      return (
        period &&
        period.start.slice(0, 7) <= month &&
        period.end.slice(0, 7) >= month
      )
    })
    .sort((a, b) => b.imported_at.localeCompare(a.imported_at))
  const unknown = source.filter(
    (statement) => !importMonth(statement, bills, preferences),
  )
  let status: ImportStatus
  if (month > today.slice(0, 7)) status = 'upcoming'
  else if (imports.length) {
    if (
      card ||
      imports.some((statement) =>
        validMonth(preferences.importMonths[statement.id]),
      )
    )
      status = 'imported'
    else {
      const end = new Date(
        Date.UTC(Number(month.slice(0, 4)), Number(month.slice(5, 7)), 0),
      )
        .toISOString()
        .slice(0, 10)
      let cursor = `${month}-01`
      for (const period of imports
        .map(range)
        .filter((period) => period !== null)
        .sort((a, b) => a.start.localeCompare(b.start))) {
        if (period.start > cursor) break
        if (period.end >= cursor) {
          const next = new Date(`${period.end}T00:00:00Z`)
          next.setUTCDate(next.getUTCDate() + 1)
          cursor = next.toISOString().slice(0, 10)
        }
      }
      status =
        cursor > end
          ? 'imported'
          : month === today.slice(0, 7)
            ? 'pending'
            : 'partial'
    }
  } else if (tracking.noActivity?.includes(month)) status = 'no-activity'
  else if (!from) status = unknown.length ? 'unknown' : 'unconfigured'
  else if (month < from || (until && month > until)) status = 'not-tracked'
  else if (unknown.length) status = 'unknown'
  else if (isImportPending(account, month, today)) status = 'pending'
  else status = 'missing'
  return { account, status, imports, unknown, from: from ?? '', until }
}

export function calendarSummary(items: { status: ImportStatus }[]) {
  if (!items.length) return 'No sources'
  const missing = items.filter((item) => item.status === 'missing').length
  const partial = items.filter((item) => item.status === 'partial').length
  const unknown = items.filter((item) => item.status === 'unknown').length
  if (items.every((item) => item.status === 'upcoming')) return 'Upcoming'
  if (missing || partial)
    return [
      missing ? `${missing} missing` : '',
      partial ? `${partial} partial` : '',
    ]
      .filter(Boolean)
      .join(' · ')
  if (unknown) return 'Check dates'
  if (items.some((item) => item.status === 'unconfigured'))
    return 'Set tracking start'
  if (items.some((item) => item.status === 'pending')) return 'In progress'
  if (items.some((item) => item.status === 'imported')) return 'All imported'
  if (items.some((item) => item.status === 'no-activity')) return 'No activity'
  return 'Not tracked'
}
