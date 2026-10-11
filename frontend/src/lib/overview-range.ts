export type OverviewPeriod = 'all' | '1' | '3' | '6' | '12' | 'custom'
export type OverviewRange = { start_date?: string; end_date?: string }

function localDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// Rolling calendar months through today; clamp month-end dates instead of
// letting JavaScript overflow a short month into the following month.
export function overviewRange(period: OverviewPeriod, now = new Date()): OverviewRange {
  if (period === 'all' || period === 'custom') return {}
  const start = new Date(now.getFullYear(), now.getMonth() - Number(period), 1)
  const lastDay = new Date(start.getFullYear(), start.getMonth() + 1, 0).getDate()
  start.setDate(Math.min(now.getDate(), lastDay))
  return { start_date: localDate(start), end_date: localDate(now) }
}
