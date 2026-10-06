import { useState, useCallback, useSyncExternalStore } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchSystemVersion } from '@/lib/api'
import type { SystemVersionInfo } from '@/types'

const AUTO_CHECK_STORAGE_KEY = 'localfinance_auto_check_updates'

// Reactive cross-component and cross-tab preference store
const listeners = new Set<() => void>()

function subscribePreference(callback: () => void) {
  listeners.add(callback)
  const onStorage = (e: StorageEvent) => {
    if (e.key === AUTO_CHECK_STORAGE_KEY) {
      callback()
    }
  }
  window.addEventListener('storage', onStorage)
  return () => {
    listeners.delete(callback)
    window.removeEventListener('storage', onStorage)
  }
}

function getPreferenceSnapshot(): boolean {
  try {
    const stored = localStorage.getItem(AUTO_CHECK_STORAGE_KEY)
    return stored !== 'false' // Default is enabled unless opted out
  } catch {
    return true
  }
}

function setPreferenceSnapshot(enabled: boolean) {
  try {
    localStorage.setItem(AUTO_CHECK_STORAGE_KEY, enabled ? 'true' : 'false')
  } catch {
    // ignore local storage quota / security errors
  }
  listeners.forEach((l) => l())
}

export function useSystemUpdate() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [isChecking, setIsChecking] = useState(false)
  const [checkError, setCheckError] = useState<string | null>(null)

  // Synchronized reactive preference across all components in all tabs
  const autoCheckEnabled = useSyncExternalStore(
    subscribePreference,
    getPreferenceSnapshot,
    () => true
  )

  const setAutoCheckEnabled = useCallback(
    (enabled: boolean) => {
      // Abort any in-flight query immediately so an active online check does not finish contacting GitHub
      queryClient.cancelQueries({ queryKey: ['system-version'] })
      // Update reactive preference store; changing queryKey ['system-version', autoCheckEnabled]
      // naturally transitions observers without refetching the old online key
      setPreferenceSnapshot(enabled)
    },
    [queryClient]
  )

  // Query is keyed by the shared reactive preference.
  // When autoCheckEnabled is false, offline is strictly true across the whole app.
  const { data: versionInfo, refetch } = useQuery<SystemVersionInfo>({
    queryKey: ['system-version', autoCheckEnabled],
    queryFn: () => fetchSystemVersion(false, !autoCheckEnabled),
    staleTime: 1000 * 60 * 60 * 4, // 4 hours
    refetchOnWindowFocus: false,
    refetchOnMount: true,
    refetchOnReconnect: false,
  })

  // Explicit user-triggered release check
  const checkNow = useCallback(async () => {
    setIsChecking(true)
    setCheckError(null)
    try {
      const data = await fetchSystemVersion(true, false)
      queryClient.setQueryData(['system-version', autoCheckEnabled], data)
      if (data.update_available) {
        setDialogOpen(true)
      } else if (data.auto_update_error) {
        setCheckError(data.auto_update_error)
      }
      return data
    } catch (err: any) {
      const msg = err.message || 'Failed to check GitHub releases'
      setCheckError(msg)
      throw err
    } finally {
      setIsChecking(false)
    }
  }, [queryClient, autoCheckEnabled])

  return {
    versionInfo: versionInfo || null,
    dialogOpen,
    setDialogOpen,
    isChecking,
    checkError,
    autoCheckEnabled,
    setAutoCheckEnabled,
    checkNow,
    refetch,
  }
}
