import { useCallback, useEffect, useRef, useState } from 'react'
import type { Account, StatementImport } from '@/types'
import type { CalendarPreferences, SourceTracking } from '@/lib/import-calendar'
import {
  CALENDAR_STORAGE_KEY,
  emptyPreferences,
  parsePreferences,
  prunePreferences,
  withTracking,
} from '@/lib/calendar-preferences'

function readPreferences() {
  try {
    return parsePreferences(localStorage.getItem(CALENDAR_STORAGE_KEY))
  } catch {
    return emptyPreferences()
  }
}
export function useCalendarPreferences(
  accounts?: Account[],
  statements?: StatementImport[],
) {
  const [preferences, setPreferences] = useState(readPreferences)
  const [storageError, setStorageError] = useState(false)
  const current = useRef(preferences)
  current.current = preferences
  const constrain = useCallback(
    (next: CalendarPreferences) =>
      accounts && statements
        ? prunePreferences(
            next,
            accounts.map((account) => account.id),
            statements.map((statement) => statement.id),
          )
        : next,
    [accounts, statements],
  )
  const save = useCallback(
    (change: (previous: CalendarPreferences) => CalendarPreferences) => {
      // Merge with the latest saved values so another tab's unrelated edits survive.
      let previous = current.current
      if (!storageError) {
        try {
          previous = parsePreferences(
            localStorage.getItem(CALENDAR_STORAGE_KEY),
          )
        } catch {
          /* Keep session state when storage is inaccessible. */
        }
      }
      const next = constrain(change(previous))
      current.current = next
      setPreferences(next)
      try {
        localStorage.setItem(CALENDAR_STORAGE_KEY, JSON.stringify(next))
        setStorageError(false)
      } catch {
        setStorageError(true)
      }
    },
    [constrain, storageError],
  )
  useEffect(() => {
    const next = constrain(current.current)
    if (JSON.stringify(next) !== JSON.stringify(current.current))
      save((previous) => constrain(previous))
  }, [constrain, save])
  useEffect(() => {
    const sync = (event: StorageEvent) => {
      if (
        event.storageArea !== localStorage ||
        (event.key !== CALENDAR_STORAGE_KEY && event.key !== null)
      )
        return
      // A queued event can be older than another tab's latest write.
      let next: CalendarPreferences
      try {
        next = constrain(
          parsePreferences(localStorage.getItem(CALENDAR_STORAGE_KEY)),
        )
      } catch {
        setStorageError(true)
        return
      }
      current.current = next
      setPreferences(next)
      setStorageError(false)
    }
    window.addEventListener('storage', sync)
    return () => window.removeEventListener('storage', sync)
  }, [constrain])
  const assignMonth = (id: string, month: string) =>
    save((previous) => {
      const importMonths = { ...previous.importMonths }
      if (month) importMonths[id] = month
      else delete importMonths[id]
      return { ...previous, importMonths }
    })
  const updateTracking = (id: string, patch: SourceTracking) =>
    save((previous) => withTracking(previous, id, patch))
  const toggleNoActivity = (id: string, month: string) =>
    save((previous) => {
      const months = previous.sources[id]?.noActivity ?? []
      return withTracking(previous, id, {
        noActivity: months.includes(month)
          ? months.filter((value) => value !== month)
          : [...months, month],
      })
    })
  return {
    preferences,
    storageError,
    assignMonth,
    updateTracking,
    toggleNoActivity,
  }
}
