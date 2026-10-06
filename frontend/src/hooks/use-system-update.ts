import { useState, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchSystemVersion } from '@/lib/api'
import type { SystemVersionInfo } from '@/types'

const SYSTEM_VERSION_QUERY_KEY = ['system-version']
const AUTO_CHECK_STORAGE_KEY = 'localfinance_auto_check_updates'

export function useSystemUpdate() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [isChecking, setIsChecking] = useState(false)
  const [checkError, setCheckError] = useState<string | null>(null)

  // Stored preference: strictly opt-in, default is false (100% offline-by-default)
  const [autoCheckEnabled, setAutoCheckState] = useState<boolean>(() => {
    try {
      return localStorage.getItem(AUTO_CHECK_STORAGE_KEY) === 'true'
    } catch {
      return false
    }
  })

  const setAutoCheckEnabled = useCallback((enabled: boolean) => {
    try {
      localStorage.setItem(AUTO_CHECK_STORAGE_KEY, enabled ? 'true' : 'false')
      setAutoCheckState(enabled)
    } catch {
      setAutoCheckState(enabled)
    }
  }, [])

  // Offline-by-default: Only executes with refresh=false on mount,
  // which backend answers locally with current version info and checked_at="" (no network call).
  const { data: versionInfo, refetch } = useQuery<SystemVersionInfo>({
    queryKey: SYSTEM_VERSION_QUERY_KEY,
    queryFn: () => fetchSystemVersion(false),
    staleTime: 1000 * 60 * 60, // 1 hour cache
    refetchOnWindowFocus: false,
    refetchOnMount: false,
    refetchOnReconnect: false,
  })

  // Explicit user-triggered release check
  const checkNow = useCallback(async () => {
    setIsChecking(true)
    setCheckError(null)
    try {
      const data = await fetchSystemVersion(true)
      queryClient.setQueryData(SYSTEM_VERSION_QUERY_KEY, data)
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
  }, [queryClient])

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
