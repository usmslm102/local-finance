import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { splitwiseRequest } from '@/lib/api'
import type { SplitwiseEntry } from '@/types/splitwise'
import type { Transaction } from '@/types'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { PrivacyAmount } from '@/components/ui/privacy-amount'

function ReviewMatch({ entry, onClose, onDismiss }: { entry: SplitwiseEntry; onClose: () => void; onDismiss: (entry: SplitwiseEntry) => void }) {
  const client = useQueryClient()
  const expected = (entry.net_cents + entry.share_cents) / 100
  const [search, setSearch] = useState('')
  const [showMore, setShowMore] = useState(false)
  const candidates = useQuery({
    queryKey: ['splitwise-review-candidates', entry.id, entry.share_cents, search],
    queryFn: () => splitwiseRequest<Transaction[]>(`/${encodeURIComponent(entry.id)}/candidates?share_cents=${entry.share_cents}&review=true&search=${encodeURIComponent(search)}`),
  })
  const [selected, setSelected] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const bank = candidates.data?.find(candidate => candidate.id === selected)
  const difference = bank ? Math.abs(Math.round(bank.amount * 100) - Math.round(expected * 100)) / 100 : 0
  async function apply(ignore: boolean) {
    if (!ignore && !bank) return
    setBusy(true); setError('')
    try {
      await splitwiseRequest(`/${encodeURIComponent(entry.id)}/confirm`, {
        share_cents: entry.share_cents,
        category_id: entry.category_id ?? '',
        transaction_id: ignore ? undefined : bank?.id,
        allow_amount_difference: !ignore && difference > 5,
        ignore,
        expected_status: 'REVIEW',
      })
      await client.invalidateQueries()
      if (ignore) onDismiss(entry)
      onClose()
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Unable to update this match')
      await client.invalidateQueries({ queryKey: ['splitwise-pending'] })
    }
    finally { setBusy(false) }
  }
  return <Dialog open onOpenChange={open => { if (!open && !busy) onClose() }}>
    <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
      <DialogHeader><DialogTitle>Review bank match</DialogTitle><DialogDescription>{entry.description} · {entry.date} · {entry.group}</DialogDescription></DialogHeader>
      <div className="grid gap-3 sm:grid-cols-2 rounded-lg bg-muted/30 p-4">
        <div><p className="text-xs text-muted-foreground">Splitwise amount you paid</p><p className="font-semibold"><PrivacyAmount amount={expected} /></p></div>
        <div><p className="text-xs text-muted-foreground">Your expense after approval</p><p className="font-semibold"><PrivacyAmount amount={entry.share_cents / 100} /></p></div>
      </div>
      <p className="text-sm text-muted-foreground">Choose the correct bank debit. Suggestions use the entry date and preceding 15 days, closest amount first. Nothing changes until you approve.</p>
      <Input aria-label="Search bank candidates" placeholder="Search bank payee or narration" value={search} onInput={event => { setSearch(event.currentTarget.value); setSelected(''); setShowMore(false) }} onChange={event => { setSearch(event.target.value); setSelected(''); setShowMore(false) }} />
      {candidates.isPending && <p role="status">Finding nearby bank transactions…</p>}
      {candidates.error && <Alert variant="destructive"><AlertDescription>Unable to load candidates. <Button variant="link" onClick={() => { void candidates.refetch() }}>Retry</Button></AlertDescription></Alert>}
      {!candidates.isPending && !candidates.error && !candidates.data?.length && <Alert><AlertDescription>No eligible bank debit found. Import missing statements, then try again. You can also dismiss this entry.</AlertDescription></Alert>}
      <div role="group" aria-label="Bank transaction candidates" className="space-y-2">
        {candidates.data?.slice(0, showMore ? 50 : 5).map(candidate => <Button key={candidate.id} variant="outline" disabled={busy} aria-pressed={selected === candidate.id} onClick={() => setSelected(candidate.id)} className="w-full h-auto whitespace-normal justify-start text-left p-3">
          <div className="min-w-0 space-y-1">
            <p className="font-medium break-words">{candidate.cleaned_payee || candidate.raw_narration}</p>
            <p className="text-xs text-muted-foreground break-words">{candidate.raw_narration}</p>
            <p className="text-xs text-muted-foreground">{candidate.tx_date.split('T')[0]} · {candidate.account_name}</p>
            <p><PrivacyAmount amount={candidate.amount} /> <span className="text-xs text-muted-foreground">· Difference <PrivacyAmount amount={Math.abs(candidate.amount - expected)} /></span></p>
            {selected === candidate.id && <Badge>Selected</Badge>}
          </div>
        </Button>)}
      </div>
      {(candidates.data?.length ?? 0) > 5 && <Button variant="ghost" onClick={() => setShowMore(!showMore)}>{showMore ? 'Show closest five' : `Show more (${candidates.data?.length} found)`}</Button>}
      <p className="text-xs text-muted-foreground">Up to 50 candidates are shown. Search to find a specific payee. Dismissing keeps this entry out of future CSV imports; you can undo the latest dismissal while this page remains open.</p>
      {bank && <Alert><AlertDescription>
        <p>Bank paid <PrivacyAmount amount={bank.amount} />; Splitwise recorded <PrivacyAmount amount={expected} />. Difference: <PrivacyAmount amount={difference} />.</p>
        <p className="mt-1">Approving counts only your Splitwise share of <PrivacyAmount amount={entry.share_cents / 100} /> as spending. The original bank amount stays intact; the difference is not added as a separate expense.</p>
      </AlertDescription></Alert>}
      {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
      <div className="flex flex-wrap justify-end gap-2">
        <Button variant="ghost" disabled={busy} onClick={onClose}>Cancel</Button>
        <Button variant="outline" disabled={busy} onClick={() => { void apply(true) }}>Dismiss entry</Button>
        <Button disabled={busy || !bank} onClick={() => { void apply(false) }}>{busy ? 'Saving…' : difference > 5 ? 'Approve amount difference' : 'Approve match'}</Button>
      </div>
    </DialogContent>
  </Dialog>
}

export function SplitwiseReview() {
  const entries = useQuery({ queryKey: ['splitwise-pending'], queryFn: () => splitwiseRequest<SplitwiseEntry[]>('') })
  const [selected, setSelected] = useState<SplitwiseEntry>()
  const [dismissed, setDismissed] = useState<SplitwiseEntry>()
  const [undoBusy, setUndoBusy] = useState(false)
  const [undoError, setUndoError] = useState('')
  const client = useQueryClient()
  const [page, setPage] = useState(0)
  const pending = (entries.data ?? []).filter(entry => entry.status === 'REVIEW' && entry.kind === 'EXPENSE' && entry.net_cents + entry.share_cents > 0)
  const current = Math.min(page, Math.max(0, Math.ceil(pending.length / 5) - 1))
  async function undo() {
    if (!dismissed) return
    const dismissedID = dismissed.id
    setUndoBusy(true); setUndoError('')
    try {
      await splitwiseRequest(`/${encodeURIComponent(dismissedID)}/reset`, {})
      await client.invalidateQueries()
      setDismissed(current => current?.id === dismissedID ? undefined : current)
    } catch (error) { setUndoError(error instanceof Error ? error.message : 'Unable to undo dismissal') }
    finally { setUndoBusy(false) }
  }
  return <Card><CardContent className="p-4 sm:p-6 space-y-4">
    <div><h2 className="text-lg font-semibold">Expenses you paid · bank match needed {pending.length > 0 && <Badge variant="secondary">{pending.length}</Badge>}</h2>
      <p className="text-sm text-muted-foreground">These entries have not changed your expenses. Review a nearby bank debit or dismiss the entry.</p>
      <p className="text-xs text-muted-foreground mt-2">After updating, saved pending entries are recalculated automatically. Older versions discarded unmatched entries; re-import your CSV once to recover those.</p>
    </div>
    {dismissed && <Alert><AlertDescription><p>Dismissed {dismissed.description}. Re-imports will keep this choice.</p><Button variant="link" disabled={undoBusy} onClick={() => { void undo() }}>Undo dismissal</Button></AlertDescription></Alert>}
    {undoError && <p role="alert" className="text-sm text-destructive">{undoError}</p>}
    {entries.isPending && <p role="status">Loading pending expenses…</p>}
    {entries.error && <Alert variant="destructive"><AlertDescription>Unable to load pending expenses. <Button variant="link" onClick={() => { void entries.refetch() }}>Retry</Button></AlertDescription></Alert>}
    {!entries.isPending && !entries.error && !pending.length && <p className="text-sm text-muted-foreground">No expenses waiting for a bank match.</p>}
    {pending.slice(current * 5, current * 5 + 5).map(entry => <div key={entry.id} className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-lg border p-4">
      <div className="min-w-0"><p className="font-medium break-words">{entry.description}</p><p className="text-xs text-muted-foreground">{entry.date} · {entry.group}</p><p className="text-sm mt-1">You paid <PrivacyAmount amount={(entry.net_cents + entry.share_cents) / 100} /> · Your share <PrivacyAmount amount={entry.share_cents / 100} /></p></div>
      <Button variant="outline" onClick={() => setSelected(entry)}>Review bank match</Button>
    </div>)}
    {pending.length > 5 && <div className="flex flex-wrap justify-between items-center gap-2"><p className="text-xs text-muted-foreground">{current * 5 + 1}–{Math.min(current * 5 + 5, pending.length)} of {pending.length}</p><div className="flex gap-2"><Button variant="outline" size="sm" disabled={current === 0} onClick={() => setPage(current - 1)}>Previous</Button><Button variant="outline" size="sm" disabled={(current + 1) * 5 >= pending.length} onClick={() => setPage(current + 1)}>Next</Button></div></div>}
    {selected && <ReviewMatch key={selected.id} entry={selected} onDismiss={entry => { setDismissed(entry); setUndoError('') }} onClose={() => setSelected(undefined)} />}
  </CardContent></Card>
}
