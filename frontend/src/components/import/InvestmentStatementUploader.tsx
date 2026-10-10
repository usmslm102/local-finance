import { cn } from '@/lib/utils'
import { Field, FieldLabel } from '@/components/ui/field'
import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { UploadCloud, Eye, X } from 'lucide-react'
import { fetchInvestmentFormats, importInvestment, previewInvestment } from '@/lib/api'
import { formatMoney } from '@/lib/formatters'
import type { InvestmentSnapshot } from '@/types/investments'
import { usePrivacy } from '@/components/privacy-provider'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

type QueuedInvestment = {
  id: string
  file: File
  status: 'previewing' | 'ready' | 'importing' | 'imported' | 'error'
  preview?: InvestmentSnapshot
  error?: string
  duplicate?: boolean
}

export function InvestmentStatementUploader() {
  const client = useQueryClient()
  const { isPrivacyMode, maskValue } = usePrivacy()
  const formats = useQuery({ queryKey: ['investment-formats'], queryFn: fetchInvestmentFormats })
  const [queue, setQueue] = useState<QueuedInvestment[]>([])
  const [active, setActive] = useState<InvestmentSnapshot | null>(null)
  const [dragging, setDragging] = useState(false)
  const [busy, setBusy] = useState(false)
  const extensions = [
    ...new Set(formats.data?.parsers.flatMap((parser) => parser.extensions) ?? []),
  ]
  const update = (id: string, patch: Partial<QueuedInvestment>) =>
    setQueue((items) => items.map((item) => (item.id === id ? { ...item, ...patch } : item)))
  const addFiles = async (files: File[]) => {
    if (!extensions.length) return
    const added: QueuedInvestment[] = files.map((file) => ({
      id: crypto.randomUUID(),
      file,
      status: 'previewing',
    }))
    setQueue((items) => [...items, ...added])
    for (const item of added) {
      try {
        if (item.file.size > (formats.data?.max_file_size ?? 0))
          throw new Error('File exceeds the supported upload size')
        const preview = await previewInvestment(item.file)
        update(item.id, { status: 'ready', preview })
      } catch (err) {
        update(item.id, {
          status: 'error',
          error: err instanceof Error ? err.message : 'Unable to preview statement',
        })
      }
    }
  }
  const save = async (item: QueuedInvestment) => {
    update(item.id, { status: 'importing', error: undefined })
    try {
      const result = await importInvestment(item.file)
      update(item.id, { status: 'imported', duplicate: result.duplicate })
      await client.invalidateQueries({ queryKey: ['investments'] })
    } catch (err) {
      update(item.id, {
        status: 'error',
        error: err instanceof Error ? err.message : 'Import failed',
      })
    }
  }
  const saveReady = async () => {
    setBusy(true)
    try {
      for (const item of queue.filter((item) => item.status === 'ready')) await save(item)
    } finally {
      setBusy(false)
    }
  }
  const ready = queue.filter((item) => item.status === 'ready').length
  const money = (value: number | null, currency: string) =>
    formatMoney(value, currency, isPrivacyMode)

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UploadCloud className="size-4" />
            Import investment statements
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">
            Attach one or multiple statements. The provider is detected automatically. Inspect
            holdings before saving them to Investments.
          </p>
          <Field
            onDragOver={(event) => {
              event.preventDefault()
              setDragging(true)
            }}
            onDragLeave={() => setDragging(false)}
            onDrop={(event) => {
              event.preventDefault()
              setDragging(false)
              void addFiles(Array.from(event.dataTransfer.files))
            }}
            className={cn(
              'rounded-xl border-2 border-dashed p-8 flex flex-col gap-3 text-center',
              dragging ? 'border-primary bg-primary/5' : 'border-border bg-muted/20'
            )}
          >
            <UploadCloud className="size-8 mx-auto text-muted-foreground" />
            <p>Drop investment statements here or choose files</p>
            <FieldLabel htmlFor="investment-files" className="sr-only">
              Investment statement files
            </FieldLabel>
            <Input
              id="investment-files"
              type="file"
              multiple
              accept={extensions.join(',')}
              disabled={!extensions.length}
              onChange={(event) => {
                void addFiles(Array.from(event.target.files ?? []))
                event.target.value = ''
              }}
            />
          </Field>
          {formats.isPending && <p>Loading supported formats…</p>}
          {formats.error && (
            <p role="alert" className="text-destructive">
              {formats.error.message}
            </p>
          )}
          {formats.data && (
            <div className="text-sm text-muted-foreground">
              <p>
                Supported statements (up to {formats.data.max_file_size / (1024 * 1024)} MB each):
              </p>
              <ul className="list-disc pl-5">
                {formats.data.parsers.map((parser) => (
                  <li key={parser.id}>
                    {parser.provider} — {parser.name} ({parser.extensions.join(', ')})
                  </li>
                ))}
              </ul>
              {!formats.data.parsers.length && <p>No statement formats are currently available.</p>}
            </div>
          )}
          <p className="text-xs text-muted-foreground">
            Investment holdings remain separate from bank transactions, income, and expenses.
          </p>
        </CardContent>
      </Card>
      {queue.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Statement queue</CardTitle>
            <Button disabled={!ready || busy} onClick={() => void saveReady()}>
              {busy ? 'Importing…' : `Import ${ready} ready ${ready === 1 ? 'file' : 'files'}`}
            </Button>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            {queue.map((item) => (
              <div key={item.id} className="rounded-lg border p-4 flex flex-col gap-2">
                <div className="flex flex-wrap justify-between gap-3">
                  <div>
                    <p className="font-medium break-all">{maskValue(item.file.name)}</p>
                    <p className="text-sm text-muted-foreground">
                      {item.preview
                        ? `${item.preview.provider} · ${maskValue(item.preview.account_ref)} · ${item.preview.as_of} · ${item.preview.holdings.length} holdings`
                        : 'Investment statement'}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    {item.preview && (
                      <Button variant="outline" onClick={() => setActive(item.preview ?? null)}>
                        <Eye data-icon="inline-start" />
                        Preview
                      </Button>
                    )}
                    {item.status === 'ready' && (
                      <Button disabled={busy} onClick={() => void save(item)}>
                        Import
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      aria-label={`Remove ${item.file.name} from queue`}
                      disabled={item.status === 'previewing' || item.status === 'importing' || busy}
                      onClick={() =>
                        setQueue((items) => items.filter((entry) => entry.id !== item.id))
                      }
                    >
                      <X data-icon="inline-start" />
                    </Button>
                  </div>
                </div>
                <p role="status" className="text-sm">
                  {item.status === 'imported'
                    ? item.duplicate
                      ? 'Already imported — no duplicate snapshot added.'
                      : 'Imported. View your holdings in Investments.'
                    : item.status === 'ready'
                      ? 'Preview ready. Review before importing.'
                      : item.status === 'previewing'
                        ? 'Detecting format and preparing preview…'
                        : item.status === 'importing'
                          ? 'Importing…'
                          : 'Import needs attention.'}
                </p>
                {item.error && (
                  <p role="alert" className="text-sm text-destructive">
                    {item.error}
                  </p>
                )}
              </div>
            ))}
            <Link to="/investments" className="text-sm underline">
              Open Investments
            </Link>
          </CardContent>
        </Card>
      )}
      <Dialog
        open={active !== null}
        onOpenChange={(open) => {
          if (!open) setActive(null)
        }}
      >
        <DialogContent className="sm:max-w-4xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Investment statement preview</DialogTitle>
          </DialogHeader>
          {active && (
            <div className="flex flex-col gap-4">
              <p>
                {active.provider} · {maskValue(active.account_ref)} · Values as of {active.as_of}
              </p>
              <div className="grid sm:grid-cols-3 gap-3">
                <p>Invested: {money(active.invested_value, active.currency)}</p>
                <p>Current: {money(active.current_value, active.currency)}</p>
                <p>Unrealized return: {money(active.unrealized_return, active.currency)}</p>
              </div>
              <p className="text-xs text-muted-foreground">
                Preview only. Nothing is saved until you import.
              </p>
              {active.warnings.map((warning) => (
                <p key={warning} role="status" className="text-sm">
                  {warning}
                </p>
              ))}
              <Table>
                <TableHeader>
                  <TableRow>
                    {['Holding', 'Asset class', 'Quantity', 'Invested', 'Current', 'Return'].map(
                      (label) => (
                        <TableHead key={label}>{label}</TableHead>
                      )
                    )}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {active.holdings.map((holding) => (
                    <TableRow key={holding.isin || holding.symbol}>
                      <TableCell>
                        {holding.symbol}
                        <div className="text-xs text-muted-foreground">{holding.isin}</div>
                      </TableCell>
                      <TableCell>{holding.asset_class}</TableCell>
                      <TableCell>{maskValue(holding.quantity)}</TableCell>
                      <TableCell>{money(holding.invested_value, active.currency)}</TableCell>
                      <TableCell>{money(holding.current_value, active.currency)}</TableCell>
                      <TableCell>{money(holding.unrealized_return, active.currency)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
