import type { StatementImport } from '@/types'
import { StatementImportRow } from './StatementImportRow'

export function UndatedImportsSection({
  statements,
  today,
  isPrivacyMode,
  sourceName,
  onAssignMonth,
}: {
  statements: StatementImport[]
  today: string
  isPrivacyMode: boolean
  sourceName: (id: string) => string
  onAssignMonth: (id: string, month: string) => void
}) {
  if (!statements.length) return null
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">Imports needing a month</h2>
      <p className="text-sm text-muted-foreground">
        These files have no usable period. Confirm their month to include them
        in the calendar.
      </p>
      {statements.map((statement, index) => (
        <StatementImportRow
          key={statement.id}
          statement={statement}
          label={
            isPrivacyMode
              ? `${sourceName(statement.account_id)}, undated import ${index + 1}`
              : `${statement.filename}, ${sourceName(statement.account_id)}, undated import ${index + 1}`
          }
          today={today}
          isPrivacyMode={isPrivacyMode}
          onAssignMonth={onAssignMonth}
        />
      ))}
    </section>
  )
}
