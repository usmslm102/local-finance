import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { fetchWithAuth } from '@/lib/api'
import { refreshActiveFinance } from '@/lib/finance-refresh'

// Keep active reports current after out-of-band MCP writes without repeatedly
// fetching every report. Hidden tabs catch up on visibility/focus instead.
export function DataFreshness() {
  const client = useQueryClient()
  useEffect(() => {
    let previous: string | undefined
    let active = true
    let pending = false
    const controller = new AbortController()
    const check = async () => {
      if (pending || document.visibilityState === 'hidden') return
      pending = true
      try {
        const response = await fetchWithAuth('/api/data-revision', { signal: controller.signal, cache: 'no-store' })
        if (!response.ok || !active) return
        const { revision } = await response.json() as { revision: string }
        if (!active) return
        if (previous !== revision) {
          void refreshActiveFinance(client, () => active).catch(() => { if (active) previous = undefined })
        }
        previous = revision
      } catch {
        // Existing query errors remain visible; a failed freshness check never
        // clears cached financial data. The next focus/tick retries.
      } finally { pending = false }
    }
    void check()
    const timer = window.setInterval(() => void check(), 5000)
    window.addEventListener('focus', check)
    document.addEventListener('visibilitychange', check)
    return () => {
      active = false
      controller.abort()
      window.clearInterval(timer)
      window.removeEventListener('focus', check)
      document.removeEventListener('visibilitychange', check)
    }
  }, [client])
  return null
}
