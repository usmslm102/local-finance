import type { QueryClient } from '@tanstack/react-query'

// Invalidation alone reuses an initial in-flight query with no cached data. A
// pre-write response must be cancelled before the post-write reads can begin.
export async function refreshActiveFinance(client: QueryClient, active = () => true) {
  await client.cancelQueries({ type: 'active', fetchStatus: 'fetching' })
  if (active()) await client.invalidateQueries()
}
