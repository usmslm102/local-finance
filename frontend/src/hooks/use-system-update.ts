import { useState, useCallback } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchSystemVersion } from '@/lib/api'
import type { SystemVersionInfo } from '@/types'

const AUTO_CHECK_STORAGE_KEY = 'localfinance_auto_check_updates'

export function useSystemUpdate() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [isChecking, setIsChecking] = useState(false)
  const [checkError, setCheckError] = useState<string | null>(null)

  // Stored preference: automatic release check enabled by default with opt-out
  const [autoCheckEnabled, setAutoCheckState] = useState<boolean>(() => {
    try {
      const stored = localStorage.getItem(AUTO_CHECK_STORAGE_KEY)
      return stored !== 'false' // defaults to true unless user explicitly opted out
    } catch {
      return true
    }
  })

  const setAutoCheckEnabled = useCallback(
    (enabled: boolean) => {
      try {
        localStorage.setItem(AUTO_CHECK_STORAGE_KEY, enabled ? 'true' : 'false')
        setAutoCheckState(enabled)
        queryClient.invalidateQueries({ queryKey: ['system-version'] })
      } catch {
        setAutoCheckState(enabled)
      }
    },
    [queryClient]
  )

  // Automatic background check runs on app mount when autoCheckEnabled is true (cached 4 hours)
  // If autoCheckEnabled is false (opted out), it queries with offline=true, touching 0 external endpoints
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
      queryClient.setQueryData(['system-version'], data)
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
