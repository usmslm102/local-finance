import {
  validMonth,
  type CalendarPreferences,
  type SourceTracking,
} from '@/lib/import-calendar'

export const CALENDAR_STORAGE_KEY = 'local-finance-import-calendar-v1'
export const emptyPreferences = (): CalendarPreferences => ({
  sources: {},
  importMonths: {},
})
function record(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
export function parsePreferences(raw: string | null): CalendarPreferences {
  const result = emptyPreferences()
  try {
    const parsed: unknown = JSON.parse(raw ?? 'null')
    if (!record(parsed)) return result
    if (record(parsed.sources))
      for (const [id, entry] of Object.entries(parsed.sources)) {
        if (
          !record(entry) ||
          ['__proto__', 'constructor', 'prototype'].includes(id)
        )
          continue
        const from =
          typeof entry.from === 'string' && validMonth(entry.from)
            ? entry.from
            : undefined
        const until =
          typeof entry.until === 'string' &&
          validMonth(entry.until) &&
          (!from || from <= entry.until)
            ? entry.until
            : undefined
        result.sources[id] = {
          from,
          until,
          noActivity: Array.isArray(entry.noActivity)
            ? [
                ...new Set(
                  entry.noActivity.filter(
                    (month): month is string =>
                      typeof month === 'string' && validMonth(month),
                  ),
                ),
              ]
            : [],
        }
      }
    if (record(parsed.importMonths))
      for (const [id, month] of Object.entries(parsed.importMonths)) {
        if (
          typeof month === 'string' &&
          validMonth(month) &&
          !['__proto__', 'constructor', 'prototype'].includes(id)
        )
          result.importMonths[id] = month
      }
  } catch {
    /* Corrupt browser data must not prevent opening the calendar. */
  }
  return result
}
export function prunePreferences(
  preferences: CalendarPreferences,
  accountIds: string[],
  statementIds: string[],
): CalendarPreferences {
  const accounts = new Set(accountIds)
  const statements = new Set(statementIds)
  return {
    sources: Object.fromEntries(
      Object.entries(preferences.sources).filter(([id]) => accounts.has(id)),
    ),
    importMonths: Object.fromEntries(
      Object.entries(preferences.importMonths).filter(([id]) =>
        statements.has(id),
      ),
    ),
  }
}
export function withTracking(
  preferences: CalendarPreferences,
  id: string,
  patch: SourceTracking,
): CalendarPreferences {
  return {
    ...preferences,
    sources: {
      ...preferences.sources,
      [id]: { ...preferences.sources[id], ...patch },
    },
  }
}
