import { Check, CircleAlert, Clock } from 'lucide-react'
import type { ImportStatus } from '@/lib/import-calendar'
import { cn } from '@/lib/utils'

const labels: Record<ImportStatus, string> = {
  imported: 'Imported',
  partial: 'Partial range',
  missing: 'Missing import',
  'no-activity': 'No activity',
  'not-tracked': 'Not tracked',
  pending: 'In progress',
  upcoming: 'Upcoming',
  unknown: 'Check dates',
  unconfigured: 'Set tracking start',
}
export function CalendarStatus({ status }: { status: ImportStatus }) {
  const Icon =
    status === 'imported' || status === 'no-activity'
      ? Check
      : status === 'missing' || status === 'partial' || status === 'unknown'
        ? CircleAlert
        : Clock
  return (
    <span
      className={cn(
        'flex items-center gap-1.5 text-sm',
        (status === 'imported' || status === 'no-activity') && 'text-success',
        (status === 'missing' ||
          status === 'partial' ||
          status === 'unknown') &&
          'text-warning',
        (status === 'pending' ||
          status === 'upcoming' ||
          status === 'not-tracked') &&
          'text-muted-foreground',
      )}
    >
      <Icon className="size-4" aria-hidden="true" />
      {labels[status]}
    </span>
  )
}
