import type { MonthlyCashFlow, StatementImport } from '@/types'
import type { SavingsGoal } from '@/types/planning'

export function localDate(now = new Date()) {
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}
export function monthName(month: string) {
  return new Date(`${month}-01T12:00:00`).toLocaleDateString('en-IN', {
    month: 'short',
    year: 'numeric',
  })
}
function dayNumber(date: string): number | null {
  const clean = date.slice(0, 10)
  if (!/^\d{4}-\d{2}-\d{2}$/.test(clean)) return null
  const timestamp = Date.parse(`${clean}T00:00:00Z`)
  if (!Number.isFinite(timestamp) || new Date(timestamp).toISOString().slice(0, 10) !== clean)
    return null
  return timestamp / 86400000
}
function dayDate(day: number) {
  return new Date(day * 86400000).toISOString().slice(0, 10)
}

// Date ranges are evidence of reported coverage, not a bank balance audit.
export function statementCoverage(
  statements: StatementImport[],
  accountId: string,
  month: string,
  today = localDate()
) {
  const start = dayNumber(`${month}-01`)
  if (start === null || month > today.slice(0, 7))
    return { covered: 0, total: 0, gaps: [], latest: '' }
  const first = new Date(`${month}-01T00:00:00Z`)
  const monthEnd = Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0) / 86400000
  const end = Math.min(monthEnd, dayNumber(today) ?? monthEnd)
  const ranges: [number, number][] = []
  let latest = ''
  for (const statement of statements.filter((s) => s.account_id === accountId)) {
    const from = dayNumber(statement.start_date || '')
    const to = dayNumber(statement.end_date || '')
    if (from === null || to === null || from > to) continue
    if (dayDate(to) > latest) latest = dayDate(to)
    if (to >= start && from <= end) ranges.push([Math.max(from, start), Math.min(to, end)])
  }
  ranges.sort((a, b) => a[0] - b[0])
  let cursor = start
  const gaps: { start: string; end: string }[] = []
  for (const [from, to] of ranges) {
    if (from > cursor) gaps.push({ start: dayDate(cursor), end: dayDate(from - 1) })
    cursor = Math.max(cursor, to + 1)
  }
  if (cursor <= end) gaps.push({ start: dayDate(cursor), end: dayDate(end) })
  const missing = gaps.reduce(
    (sum, gap) => sum + (dayNumber(gap.end)! - dayNumber(gap.start)! + 1),
    0
  )
  return { covered: end - start + 1 - missing, total: end - start + 1, gaps, latest }
}

// Use consecutive calendar months; missing months stay visible instead of silently
// borrowing older observations or inventing zero spending.
export function forecastBaseline(trends: MonthlyCashFlow[], today = localDate()) {
  const current = new Date(`${today.slice(0, 7)}-01T00:00:00Z`)
  const months = Array.from({ length: 3 }, (_, index) => {
    const month = new Date(Date.UTC(current.getUTCFullYear(), current.getUTCMonth() - index - 1, 1))
      .toISOString()
      .slice(0, 7)
    return { month, recorded: trends.find((t) => t.month === month) }
  }).reverse()
  const recorded = months.flatMap((m) => (m.recorded ? [m.recorded] : []))
  const average = (key: 'income' | 'expense') =>
    recorded.length
      ? Math.round((recorded.reduce((sum, m) => sum + m[key], 0) / recorded.length) * 100) / 100
      : 0
  return { months, income: average('income'), expense: average('expense'), count: recorded.length }
}

export function goalProgress(goal: SavingsGoal, today = localDate()) {
  const remaining = Math.max(0, Math.round((goal.target_amount - goal.saved_amount) * 100) / 100)
  const due = dayNumber(goal.target_date)
  const now = dayNumber(today)
  const overdue = remaining > 0 && due !== null && now !== null && due < now
  const target = new Date(`${goal.target_date}T00:00:00Z`)
  const current = new Date(`${today}T00:00:00Z`)
  const monthsLeft = Math.max(
    1,
    (target.getUTCFullYear() - current.getUTCFullYear()) * 12 +
      target.getUTCMonth() -
      current.getUTCMonth() +
      1
  )
  const requiredMonthly = Math.ceil((remaining / monthsLeft) * 100) / 100
  return {
    remaining,
    monthsLeft,
    requiredMonthly,
    overdue,
    completed: remaining === 0,
    percent: Math.min(100, (goal.saved_amount / goal.target_amount) * 100),
    onTrack: remaining === 0 || (!overdue && goal.monthly_contribution >= requiredMonthly),
  }
}

export interface ForecastAssumptions {
  opening: number
  income: number
  expense: number
  oneOff: number
  goalContribution: number
  months: number
  goalReservations?: number[]
}
export function forecastScenario(assumptions: ForecastAssumptions, today = localDate()) {
  const cents = (amount: number) => Math.round(amount * 100)
  const current = new Date(`${today.slice(0, 7)}-01T00:00:00Z`)
  const monthly = cents(assumptions.income) - cents(assumptions.expense)
  return Array.from({ length: assumptions.months + 1 }, (_, index) => {
    const month = new Date(Date.UTC(current.getUTCFullYear(), current.getUTCMonth() + index, 1))
      .toISOString()
      .slice(0, 7)
    const balance =
      cents(assumptions.opening) + monthly * index - (index > 0 ? cents(assumptions.oneOff) : 0)
    return {
      month,
      label: index === 0 ? 'Now' : `+${index} mo`,
      balance: balance / 100,
      available:
        (balance -
          cents(assumptions.goalReservations?.[index] ?? assumptions.goalContribution * index)) /
        100,
    }
  })
}

export function goalReservations(goals: SavingsGoal[], months: number) {
  return Array.from(
    { length: months + 1 },
    (_, index) =>
      goals.reduce(
        (sum, goal) =>
          sum +
          Math.min(
            Math.max(0, Math.round((goal.target_amount - goal.saved_amount) * 100)),
            Math.round(goal.monthly_contribution * 100) * index
          ),
        0
      ) / 100
  )
}
