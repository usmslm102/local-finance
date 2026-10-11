import { useId, useState } from 'react'
import { validMonth } from '@/lib/import-calendar'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  PopoverTitle,
} from '@/components/ui/popover'

export function ImportMonthControl({
  label,
  month,
  today,
  onSave,
}: {
  label: string
  month?: string
  today: string
  onSave: (month: string) => void
}) {
  const id = useId()
  const [draft, setDraft] = useState(month ?? '')
  const [open, setOpen] = useState(false)
  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) setDraft(month ?? '')
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            aria-label={`${month ? 'Change' : 'Assign'} month for ${label}`}
          />
        }
      >
        {month ? 'Change month' : 'Assign month'}
      </PopoverTrigger>
      <PopoverContent>
        <FieldGroup>
          <PopoverTitle>Import month for {label}</PopoverTitle>
          <Field>
            <FieldLabel htmlFor={id}>Import month</FieldLabel>
            <Input
              id={id}
              type="month"
              value={draft}
              max={today.slice(0, 7)}
              onChange={(event) => setDraft(event.target.value)}
            />
          </Field>
          <p className="text-xs text-muted-foreground">
            Use the month this statement belongs to, rather than the month you
            uploaded it.
          </p>
          <div className="flex gap-2">
            <Button
              disabled={!validMonth(draft) || draft > today.slice(0, 7)}
              onClick={() => {
                onSave(draft)
                setOpen(false)
              }}
            >
              Save month
            </Button>
            {month && (
              <Button
                variant="ghost"
                onClick={() => {
                  onSave('')
                  setOpen(false)
                }}
              >
                Use original dates
              </Button>
            )}
          </div>
        </FieldGroup>
      </PopoverContent>
    </Popover>
  )
}
