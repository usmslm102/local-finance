import { useId, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { CircleAlert, Upload } from 'lucide-react'
import { fetchAccounts, fetchStatements, fetchCreditCardBills } from '@/lib/api'
import { localDate, monthName } from '@/lib/planning'
import {
  calendarSummary,
  sourceMonth,
  validMonth,
  validDate,
} from '@/lib/import-calendar'
import { useCalendarPreferences } from '@/hooks/use-calendar-preferences'
import { usePrivacy } from '@/components/privacy-provider'
import { Button, buttonVariants } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from '@/components/ui/empty'
import { Separator } from '@/components/ui/separator'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import { ImportCalendarGrid } from './ImportCalendarGrid'
import { MonthSourceDetail } from './MonthSourceDetail'
import { UndatedImportsSection } from './UndatedImportsSection'

export function DataHealthView() {
  const id = useId()
  const today = localDate()
  const { isPrivacyMode } = usePrivacy()
  const [year, setYear] = useState(today.slice(0, 4))
  const [selectedMonth, setSelectedMonth] = useState(today.slice(5, 7))
  const [selectedSources, setSelectedSources] = useState<string[]>([])
  const accounts = useQuery({ queryKey: ['accounts'], queryFn: fetchAccounts })
  const statements = useQuery({
    queryKey: ['statements'],
    queryFn: fetchStatements,
  })
  const bills = useQuery({
    queryKey: ['credit-card-bills'],
    queryFn: () => fetchCreditCardBills(),
  })
  const refresh = () => {
    void accounts.refetch()
    void statements.refetch()
    void bills.refetch()
  }
  const {
    preferences,
    storageError,
    assignMonth,
    updateTracking,
    toggleNoActivity,
  } = useCalendarPreferences(
    accounts.isSuccess ? accounts.data : undefined,
    statements.isSuccess ? statements.data : undefined,
  )
  if (accounts.isPending || statements.isPending || bills.isPending)
    return <PageLoading label="Loading your import calendar" />
  if (accounts.isError || statements.isError || bills.isError)
    return (
      <QueryError
        message="Unable to load your import calendar"
        onRetry={refresh}
      />
    )
  const activeSourceIds = selectedSources.filter((id) =>
    accounts.data.some((account) => account.id === id),
  )
  const sources = accounts.data.filter(
    (account) =>
      !activeSourceIds.length || activeSourceIds.includes(account.id),
  )
  const sourceName = (account: (typeof accounts.data)[number]) =>
    isPrivacyMode
      ? `Source ${accounts.data.findIndex((item) => item.id === account.id) + 1}`
      : account.nickname ||
        [
          account.bank_name,
          account.card_variant || account.account_type.replaceAll('_', ' '),
          account.account_number_mask,
        ]
          .filter(Boolean)
          .join(' · ')
  const years = [
    ...new Set([
      today.slice(0, 4),
      year,
      ...statements.data.flatMap((statement) =>
        [statement.start_date, statement.end_date]
          .filter(validDate)
          .map((date) => date.slice(0, 4)),
      ),
      ...bills.data
        .map((bill) => bill.statement_date)
        .filter(validDate)
        .map((date) => date.slice(0, 4)),
      ...Object.values(preferences.importMonths).map((month) =>
        month.slice(0, 4),
      ),
      ...Object.values(preferences.sources).flatMap((tracking) =>
        [tracking.from, tracking.until]
          .filter(validMonth)
          .map((month) => month.slice(0, 4)),
      ),
    ]),
  ]
    .sort()
    .reverse()
  // Include intervening years even when that entire year's imports are absent.
  const earliest = Math.min(...years.map(Number))
  for (let value = Number(today.slice(0, 4)) - 1; value >= earliest; value--)
    if (!years.includes(String(value))) years.push(String(value))
  years.sort().reverse()
  const months = Array.from({ length: 12 }, (_, index) => {
    const month = `${year}-${String(index + 1).padStart(2, '0')}`
    const items = sources.map((account) =>
      sourceMonth(
        account,
        statements.data,
        bills.data,
        preferences,
        month,
        today,
      ),
    )
    return { month, items, summary: calendarSummary(items) }
  })
  const selected = months[Number(selectedMonth) - 1]
  const unknownImports = selected.items.flatMap((item) => item.unknown)
  const missingMonths = months
    .filter((month) => month.items.some((item) => item.status === 'missing'))
    .map((month) => monthName(month.month).split(' ')[0])
  const needsDates = months.some((month) =>
    month.items.some((item) => item.status === 'unknown'),
  )
  const needsTracking = months.some((month) =>
    month.items.some((item) => item.status === 'unconfigured'),
  )
  const hasPartial = months.some((month) =>
    month.items.some((item) => item.status === 'partial'),
  )
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Data health</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Your imports, month by month. See what’s missing and pick a month to
            inspect.
          </p>
        </div>
        <Link to="/import" data-slot="button" className={buttonVariants()}>
          <Upload data-icon="inline-start" />
          Import statement
        </Link>
      </div>
      {storageError && (
        <Alert variant="destructive">
          <CircleAlert />
          <AlertTitle>Tracking changes could not be saved</AlertTitle>
          <AlertDescription>
            Your changes apply for this visit. Browser storage is unavailable.
          </AlertDescription>
        </Alert>
      )}
      {!accounts.data.length ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>No sources yet</EmptyTitle>
            <EmptyDescription>
              Import your first statement to start your calendar.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Link to="/import" data-slot="button" className={buttonVariants()}>
              Import statement
            </Link>
          </EmptyContent>
        </Empty>
      ) : (
        <>
          <FieldGroup className="gap-4">
            <Field className="w-36">
              <FieldLabel htmlFor={`${id}-year`}>Year</FieldLabel>
              <Select
                value={year}
                onValueChange={(value) => {
                  if (value) setYear(value)
                }}
              >
                <SelectTrigger id={`${id}-year`}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {years.map((value) => (
                      <SelectItem key={value} value={value}>
                        {value}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <div className="flex items-center gap-3">
                <FieldLabel id={`${id}-sources`}>Sources</FieldLabel>
                {activeSourceIds.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedSources([])}
                  >
                    Show all sources
                  </Button>
                )}
              </div>
              <ToggleGroup
                multiple
                value={activeSourceIds}
                onValueChange={setSelectedSources}
                variant="outline"
                className="max-w-full flex-wrap"
                aria-labelledby={`${id}-sources`}
              >
                {accounts.data.map((account) => (
                  <ToggleGroupItem
                    key={account.id}
                    value={account.id}
                    className="h-auto min-h-9 max-w-full whitespace-normal text-left"
                  >
                    {sourceName(account)}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
              <p className="text-xs text-muted-foreground">
                {activeSourceIds.length
                  ? `${activeSourceIds.length} selected`
                  : 'All sources shown. Select one or more to narrow the calendar.'}
              </p>
            </Field>
          </FieldGroup>
          <div className="flex flex-col gap-3">
            <p className="text-sm" aria-live="polite">
              {missingMonths.length
                ? `Missing imports in ${missingMonths.join(', ')}.`
                : needsDates
                  ? 'Assign undated imports to check for missing months.'
                  : needsTracking
                    ? 'Set a tracking start for sources without dated imports.'
                    : hasPartial
                      ? 'Some months have partial import ranges.'
                      : `No missing imports found in the tracked months of ${year}.`}
            </p>
            <ImportCalendarGrid
              months={months}
              year={year}
              selectedMonth={selected.month}
              detailsId={`${id}-details`}
              onSelectMonth={(month) => setSelectedMonth(month.slice(5, 7))}
              sourceName={sourceName}
            />
            <p className="text-xs text-muted-foreground">
              Cards count once in their statement month. Bank ranges can span
              several months. Bank months stay in progress until a full period
              is imported. Cards with a known billing day get a
              three-calendar-day grace period.
            </p>
          </div>
          <Separator />
          <section
            id={`${id}-details`}
            className="flex flex-col gap-4"
            aria-labelledby={`${id}-month-title`}
          >
            <div>
              <h2 id={`${id}-month-title`} className="text-lg font-semibold">
                {monthName(selected.month)}
              </h2>
              <p className="text-sm text-muted-foreground">
                {activeSourceIds.length
                  ? 'Imports for the selected sources.'
                  : 'All your sources and imports for this month.'}
              </p>
            </div>
            <div className="flex flex-col divide-y">
              {selected.items.map((item) => (
                <MonthSourceDetail
                  key={item.account.id}
                  item={item}
                  month={selected.month}
                  today={today}
                  name={sourceName(item.account)}
                  isPrivacyMode={isPrivacyMode}
                  bills={bills.data}
                  preferences={preferences}
                  onAssignMonth={assignMonth}
                  onTracking={(patch) => updateTracking(item.account.id, patch)}
                  onToggleNoActivity={() =>
                    toggleNoActivity(item.account.id, selected.month)
                  }
                />
              ))}
            </div>
          </section>
          <UndatedImportsSection
            statements={unknownImports}
            today={today}
            isPrivacyMode={isPrivacyMode}
            sourceName={(accountId) => {
              const account = accounts.data.find(
                (item) => item.id === accountId,
              )
              return account ? sourceName(account) : 'Source'
            }}
            onAssignMonth={assignMonth}
          />
          <p className="max-w-prose text-xs text-muted-foreground">
            “Imported” records a file for the period; it doesn’t verify every
            transaction. Some date ranges come from transaction dates. Tracking
            dates and no-activity confirmations are saved in this browser and
            are not included in database backups.
          </p>
        </>
      )}
    </div>
  )
}
