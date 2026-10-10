import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchCategories, splitwiseRequest, splitwiseUpload } from '@/lib/api'
import type { SplitwiseEntry } from '@/types/splitwise'
import type { Transaction } from '@/types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const money = (cents: number) => new Intl.NumberFormat('en-IN', { style: 'currency', currency: 'INR' }).format(cents / 100)

export function SplitwiseImporter() {
  const client = useQueryClient()
  const entries = useQuery({ queryKey: ['splitwise'], queryFn: () => splitwiseRequest<SplitwiseEntry[]>('') })
  const categories = useQuery({ queryKey: ['categories'], queryFn: fetchCategories })
  const [person, setPerson] = useState('Sanjay')
  const [group, setGroup] = useState('')
  const [file, setFile] = useState<File>()
  const [preview, setPreview] = useState<SplitwiseEntry[]>()
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  const [page, setPage] = useState(0)
  const [selected, setSelected] = useState<SplitwiseEntry>()
  const [share, setShare] = useState('')
  const [candidates, setCandidates] = useState<Transaction[]>([])
  const [link, setLink] = useState('')
  const [category, setCategory] = useState('uncategorized')
  const shareCents = Math.round(Number(share) * 100)
  const validShare = share.trim() !== '' && /^\d+(\.\d{1,2})?$/.test(share) && Number.isSafeInteger(shareCents) && shareCents <= (selected?.cost_cents ?? 0)
  const paid = selected?.kind === 'PAYMENT' ? Math.abs(selected.net_cents) : (selected?.net_cents ?? 0) + shareCents

  async function run(action: () => Promise<void>) {
    setBusy(true); setMessage('')
    try { await action() } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to update Splitwise') }
    finally { setBusy(false) }
  }
  function choose(entry: SplitwiseEntry) {
    setSelected(entry); setShare((entry.share_cents / 100).toFixed(2)); setCandidates([]); setLink(''); setMessage('')
    setCategory(entry.category_id ?? categories.data?.find(c => c.name.toLowerCase() === entry.category.toLowerCase())?.id ?? 'uncategorized')
  }
  async function confirm(ignore = false) {
    if (!selected) return
    await splitwiseRequest(`/${encodeURIComponent(selected.id)}/confirm`, { share_cents: ignore ? selected.share_cents : shareCents, transaction_id: ignore ? '' : link, category_id: category === 'uncategorized' ? '' : category, ignore })
    setSelected(undefined)
    await client.invalidateQueries()
    setMessage(ignore ? 'Entry ignored. Statement spending is unchanged.' : 'Confirmed. Personal spending has been updated.')
  }
  const rows = preview ?? entries.data ?? []
  return <div className="space-y-4">
    <Card><CardContent className="space-y-4 pt-6">
      <p className="text-sm text-muted-foreground">Import a Splitwise group CSV locally, then confirm your share and any matching statement payment. Your bank balances stay unchanged. Import your bank statements first or return here after importing them.</p>
      <p className="text-sm text-muted-foreground">The export contains net balances, so suggested shares assume one payer. Confirm or edit each entry, including zero balances, before it affects spending. Settlements are never expenses or income.</p>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2"><Label htmlFor="sw-person">Your member name</Label><Input id="sw-person" value={person} disabled={busy} onChange={e => { setPerson(e.target.value); setPreview(undefined) }} /></div>
        <div className="space-y-2"><Label htmlFor="sw-group">Group name (keep the same on re-import)</Label><Input id="sw-group" value={group} disabled={busy} onChange={e => { setGroup(e.target.value); setPreview(undefined) }} /></div>
      </div>
      <Input aria-label="Splitwise CSV" type="file" accept=".csv" disabled={busy} onChange={e => { setFile(e.target.files?.[0]); setPreview(undefined); setPage(0); setSelected(undefined) }} />
      <div className="flex gap-2">
        <Button disabled={busy || !file || !group.trim() || !person.trim()} onClick={() => run(async () => { setPreview(await splitwiseUpload<SplitwiseEntry[]>('preview', file!, group, person)); setPage(0); setSelected(undefined) })}>Preview CSV</Button>
        <Button disabled={busy || !preview} onClick={() => run(async () => { const result = await splitwiseUpload<{ inserted: number; duplicates: number }>('import', file!, group, person); setPreview(undefined); setPage(0); await client.invalidateQueries({ queryKey: ['splitwise'] }); setMessage(`Imported ${result.inserted} entries; ${result.duplicates} duplicates. Review entries below.`) })}>Import for review</Button>
        {preview && <Button variant="outline" disabled={busy} onClick={() => { setPreview(undefined); setPage(0) }}>Cancel preview</Button>}
      </div>
    </CardContent></Card>
    {(message || entries.error) && <Alert><AlertDescription>{message || entries.error?.message}</AlertDescription></Alert>}
    <p className="text-sm">{preview ? 'Preview — nothing saved' : 'Imported entries'} · {rows.length} records</p>
    <Table><TableHeader><TableRow><TableHead>Date / group</TableHead><TableHead>Description</TableHead><TableHead>Total</TableHead><TableHead>Your net balance</TableHead><TableHead>Your share</TableHead><TableHead>Status</TableHead><TableHead>Review</TableHead></TableRow></TableHeader>
      <TableBody>{rows.slice(page * 25, page * 25 + 25).map(e => <TableRow key={e.id}>
        <TableCell>{e.date}<div className="text-xs text-muted-foreground">{e.group} · {e.person}</div></TableCell><TableCell>{e.description}<div className="text-xs text-muted-foreground">{e.kind === 'PAYMENT' ? 'Settlement' : e.category}</div></TableCell><TableCell>{money(e.cost_cents)}</TableCell><TableCell>{money(e.net_cents)}</TableCell><TableCell>{e.kind === 'PAYMENT' ? '—' : money(e.share_cents)}</TableCell><TableCell>{e.status}</TableCell>
        <TableCell><Button size="sm" variant="outline" disabled={busy || !!preview} onClick={() => choose(e)}>Review</Button></TableCell>
      </TableRow>)}</TableBody></Table>
    <div className="flex items-center gap-2"><Button variant="outline" disabled={page === 0} onClick={() => setPage(page - 1)}>Previous</Button><span className="text-sm">Page {page + 1}</span><Button variant="outline" disabled={(page + 1) * 25 >= rows.length} onClick={() => setPage(page + 1)}>Next</Button></div>
    {selected && <Card><CardContent className="space-y-4 pt-6">
      <p className="font-medium">{selected.description} · {selected.date}</p>
      <div className="space-y-2"><Label htmlFor="sw-share">Your expense share (INR)</Label><Input id="sw-share" type="number" min="0" step="0.01" disabled={busy || selected.kind === 'PAYMENT'} value={share} onChange={e => { setShare(e.target.value); setLink(''); setCandidates([]) }} /></div>
      <p className="text-sm">{selected.kind === 'PAYMENT' ? (selected.net_cents < 0 ? 'You received' : 'You paid') : 'Amount you paid'}: {validShare ? money(paid) : 'Enter a valid share'}. {selected.kind !== 'PAYMENT' && 'Your share plus your net balance equals what you paid.'}</p>
      {selected.kind === 'EXPENSE' && paid === 0 && <div className="space-y-2"><Label>Expense category</Label><Select value={category} onValueChange={value => setCategory(value ?? 'uncategorized')} disabled={busy}><SelectTrigger aria-label="Expense category"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="uncategorized">Uncategorized</SelectItem>{categories.data?.filter(c => c.id !== 'cat_transfers').map(c => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}</SelectContent></Select></div>}
      <Button variant="outline" disabled={busy || !validShare || paid <= 0} onClick={() => run(async () => { setCandidates(await splitwiseRequest<Transaction[]>(`/${encodeURIComponent(selected.id)}/candidates?share_cents=${shareCents}`)); setLink(''); setMessage('Choose a matching payment below. If none appears, import the missing statement or check your share. Dates must be within three days.') })}>Find statement matches</Button>
      {candidates.map(t => <Button key={t.id} className="w-full justify-start" variant={link === t.id ? 'default' : 'outline'} disabled={busy} onClick={() => setLink(t.id)}>{t.tx_date} · {t.raw_narration} · {money(Math.round(t.amount * 100))}</Button>)}
      <div className="flex flex-wrap gap-2">
        <Button disabled={busy || !validShare || paid < 0 || paid > selected.cost_cents || (paid > 0 && !link)} onClick={() => run(() => confirm())}>Confirm {paid === 0 ? 'paid by someone else' : 'statement match'}</Button>
        <Button variant="outline" disabled={busy} onClick={() => run(() => confirm(true))}>Ignore entry</Button>
        <Button variant="outline" disabled={busy} onClick={() => run(async () => { await splitwiseRequest(`/${encodeURIComponent(selected.id)}/reset`, {}); setSelected(undefined); await client.invalidateQueries(); setMessage('Confirmation removed. Original statement spending restored.') })}>Undo confirmation</Button>
        <Button variant="ghost" disabled={busy} onClick={() => setSelected(undefined)}>Close</Button>
      </div>
    </CardContent></Card>}
  </div>
}
