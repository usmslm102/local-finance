import React from 'react'
import ReactDOM from 'react-dom/client'
import { MutationCache, QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { router } from './router'
import './index.css'
import { installDesktopActions } from './lib/desktop'
import { refreshActiveFinance } from './lib/finance-refresh'

void installDesktopActions()

const queryClient: QueryClient = new QueryClient({
  mutationCache: new MutationCache({ onSuccess: () => refreshActiveFinance(queryClient) }),
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      refetchOnWindowFocus: false,
    },
  },
})

// Desktop hosts can request a refresh after focus returns from an external MCP client.
window.addEventListener('localfinance:refresh', () => {
  void queryClient.invalidateQueries()
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>
)
