import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

export function QueryError({
  message = 'Unable to load this view',
  onRetry,
}: {
  message?: string
  onRetry: () => unknown
}) {
  return (
    <Alert variant="destructive">
      <AlertCircle />
      <AlertTitle>{message}</AlertTitle>
      <AlertDescription className="flex flex-col items-start gap-3">
        <p>Check that LocalFinance is still running, then try again.</p>
        <Button variant="outline" onClick={() => void onRetry()}>
          Try again
        </Button>
      </AlertDescription>
    </Alert>
  )
}

export function PageLoading({ label = 'Loading your data' }: { label?: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-6">
      <span className="sr-only">{label}</span>
      <Skeleton className="h-8 w-56" />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[0, 1, 2, 3].map((item) => (
          <Skeleton key={item} className="h-28 w-full" />
        ))}
      </div>
      <Skeleton className="h-72 w-full" />
    </div>
  )
}
