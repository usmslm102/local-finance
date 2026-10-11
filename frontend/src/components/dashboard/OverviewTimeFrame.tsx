import { useId, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { overviewRange, type OverviewPeriod, type OverviewRange } from '@/lib/overview-range'
import { formatDate } from '@/lib/utils'

export function OverviewTimeFrame({ range, onChange }: { range: OverviewRange; onChange: (range: OverviewRange) => void }) {
  const id = useId()
  const [period, setPeriod] = useState<OverviewPeriod>('all')
  const initial = overviewRange('1')
  const [start, setStart] = useState(initial.start_date ?? '')
  const [end, setEnd] = useState(initial.end_date ?? '')
  const invalid = !start || !end || start > end
  return (
    <div className="rounded-xl border bg-card p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold">Overview time frame</h2>
          <p className="text-xs text-muted-foreground" aria-live="polite">
            {range.start_date && range.end_date ? `${formatDate(range.start_date)} – ${formatDate(range.end_date)}` : 'All time'} · Income, spending and charts
          </p>
        </div>
        <div className="w-full sm:w-44">
          <Select value={period} onValueChange={(value) => {
            if (!value) return
            setPeriod(value as OverviewPeriod)
            if (value !== 'custom') onChange(overviewRange(value as OverviewPeriod))
          }}>
            <SelectTrigger aria-label="Overview time frame"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All time</SelectItem>
              <SelectItem value="1">Last 1 month</SelectItem>
              <SelectItem value="3">Last 3 months</SelectItem>
              <SelectItem value="6">Last 6 months</SelectItem>
              <SelectItem value="12">Last 1 year</SelectItem>
              <SelectItem value="custom">Custom range</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
      {period === 'custom' && (
        <form className="flex flex-wrap items-end gap-3" onSubmit={(event) => {
          event.preventDefault()
          if (!invalid) onChange({ start_date: start, end_date: end })
        }}>
          <div className="grid gap-1.5 flex-1 min-w-40">
            <Label htmlFor={`${id}-start`}>From</Label>
            <Input id={`${id}-start`} type="date" required value={start} max={end || undefined} onInput={(event) => setStart(event.currentTarget.value)} onChange={(event) => setStart(event.target.value)} />
          </div>
          <div className="grid gap-1.5 flex-1 min-w-40">
            <Label htmlFor={`${id}-end`}>To</Label>
            <Input id={`${id}-end`} type="date" required value={end} min={start || undefined} onInput={(event) => setEnd(event.currentTarget.value)} onChange={(event) => setEnd(event.target.value)} />
          </div>
          <Button type="submit" disabled={invalid}>Apply range</Button>
          {invalid && <p role="alert" className="w-full text-xs text-destructive">Choose both dates, with From on or before To.</p>}
        </form>
      )}
      <p className="text-xs text-muted-foreground">Balances, investments and card dues are current snapshots. Monthly review and emergency runway use their own periods.</p>
    </div>
  )
}
