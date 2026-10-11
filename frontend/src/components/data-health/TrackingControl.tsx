import { useId, useState } from 'react'
import { Settings2 } from 'lucide-react'
import { validMonth, type SourceTracking } from '@/lib/import-calendar'
import { Button } from '@/components/ui/button'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  PopoverTitle,
} from '@/components/ui/popover'

export function TrackingControl({
  name,
  from,
  until,
  onSave,
}: {
  name: string
  from: string
  until: string
  onSave: (patch: SourceTracking) => void
}) {
  const id = useId()
  const [open, setOpen] = useState(false)
  const [draftFrom, setDraftFrom] = useState(from)
  const [draftUntil, setDraftUntil] = useState(until)
  const valid =
    (!draftFrom || validMonth(draftFrom)) &&
    (!draftUntil || validMonth(draftUntil)) &&
    (!draftFrom || !draftUntil || draftFrom <= draftUntil)
  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setDraftFrom(from)
          setDraftUntil(until)
        }
      }}
    >
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            aria-label={`Tracking dates for ${name}`}
          />
        }
      >
        <Settings2 data-icon="inline-start" />
        Tracking
      </PopoverTrigger>
      <PopoverContent>
        <FieldGroup>
          <PopoverTitle>Tracking dates for {name}</PopoverTitle>
          <Field>
            <FieldLabel htmlFor={`${id}-from`}>Track from</FieldLabel>
            <Input
              id={`${id}-from`}
              type="month"
              value={draftFrom}
              max={draftUntil || undefined}
              onChange={(event) => setDraftFrom(event.target.value)}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor={`${id}-until`}>
              Stopped using (optional)
            </FieldLabel>
            <Input
              id={`${id}-until`}
              type="month"
              value={draftUntil}
              min={draftFrom || undefined}
              onChange={(event) => setDraftUntil(event.target.value)}
            />
          </Field>
          <p className="text-xs text-muted-foreground">
            Defaults to your earliest dated import. Earlier months won’t be
            marked missing.
          </p>
          <Button
            disabled={!valid}
            onClick={() => {
              onSave({ from: draftFrom, until: draftUntil })
              setOpen(false)
            }}
          >
            Save tracking dates
          </Button>
        </FieldGroup>
      </PopoverContent>
    </Popover>
  )
}
