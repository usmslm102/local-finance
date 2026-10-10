import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { splitwiseRequest } from '@/lib/api'
import type { SplitwiseEntry } from '@/types/splitwise'
import type { Transaction } from '@/types'
import { SplitwiseImporter } from '@/components/import/SplitwiseImporter'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PrivacyAmount } from '@/components/ui/privacy-amount'

interface Mapping { entry: SplitwiseEntry; bank?: Transaction }

export function SplitwiseView() {
  const client = useQueryClient()
  const mappings = useQuery({ queryKey: ['splitwise-mappings'], queryFn: () => splitwiseRequest<Mapping[]>('/mappings') })
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const rows = (mappings.data ?? []).filter(m => `${m.entry.description} ${m.entry.group} ${m.entry.date} ${m.bank?.raw_narration ?? ''} ${m.bank?.account_name ?? ''}`.toLowerCase().includes(search.toLowerCase()))
  const expense = (mappings.data ?? []).reduce((sum, m) => sum + (m.entry.kind === 'EXPENSE' ? m.entry.share_cents : 0), 0)
  async function remove(entry: SplitwiseEntry) {
    setBusy(true); setMessage('')
    try {
      await splitwiseRequest(`/${encodeURIComponent(entry.id)}/confirm`, { ignore: true, share_cents: entry.share_cents })
      setPage(0); await client.invalidateQueries(); setMessage('Removed. The bank transaction’s original treatment is restored. Re-imports will keep this removal.')
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to remove mapping') }
    finally { setBusy(false) }
  }
  return <div className="space-y-6">
    <div><h1 className="text-2xl font-bold tracking-tight">Splitwise</h1><p className="text-sm text-muted-foreground">Your expense shares, bank matches and member transfers.</p></div>
    <Card><CardContent className="flex flex-wrap gap-6 pt-6"><p>{mappings.data?.length ?? 0} applied entries</p><p>Your expenses: <PrivacyAmount amount={expense / 100} /></p><p>{mappings.data?.filter(m => m.entry.kind === 'PAYMENT').length ?? 0} transfers / settlements</p></CardContent></Card>
    <SplitwiseImporter />
    {(message || mappings.error) && <Alert><AlertDescription>{message || mappings.error?.message}</AlertDescription></Alert>}
    <section className="space-y-3">
      <h2 className="text-lg font-semibold">Applied matches</h2>
      <p className="text-sm text-muted-foreground">See which Splitwise entry maps to which bank movement. Remove an incorrect match to restore its original bank spending. Expenses paid by others have no bank payment.</p>
      <Input aria-label="Search Splitwise mappings" placeholder="Search description, group or bank narration" value={search} onChange={e => { setSearch(e.target.value); setPage(0) }} />
      <Table><TableHeader><TableRow><TableHead>Splitwise entry</TableHead><TableHead>Your expense</TableHead><TableHead>Matched bank transaction</TableHead><TableHead>Action</TableHead></TableRow></TableHeader>
        <TableBody>{rows.slice(page * 25, page * 25 + 25).map(({ entry: e, bank }) => <TableRow key={e.id}>
          <TableCell><p className="font-medium">{e.description}</p><p className="text-xs text-muted-foreground">{e.date} · {e.group}</p><Badge variant="secondary">{e.kind === 'PAYMENT' ? 'Transfer / settlement' : 'Split expense'}</Badge></TableCell>
          <TableCell><PrivacyAmount amount={e.kind === 'PAYMENT' ? 0 : e.share_cents / 100} /></TableCell>
          <TableCell>{bank ? <><p>{bank.raw_narration}</p><p className="text-xs text-muted-foreground">{bank.tx_date.split('T')[0]} · {bank.account_name} · {bank.tx_type === 'CREDIT' ? 'Received' : 'Paid'} <PrivacyAmount amount={bank.amount} /></p></> : <span className="text-sm text-muted-foreground">Paid by someone else · no bank movement</span>}</TableCell>
          <TableCell><Button variant="outline" size="sm" disabled={busy} onClick={() => { void remove(e) }}>Remove</Button></TableCell>
        </TableRow>)}</TableBody></Table>
      {mappings.isPending && <p className="text-sm">Loading matches…</p>}
      {!mappings.isPending && !mappings.error && !rows.length && <p className="text-sm text-muted-foreground">No applied entries found.</p>}
      <div className="flex items-center gap-2"><Button variant="outline" disabled={page === 0} onClick={() => setPage(page - 1)}>Previous</Button><span className="text-sm">Page {page + 1} · {rows.length} entries</span><Button variant="outline" disabled={(page + 1) * 25 >= rows.length} onClick={() => setPage(page + 1)}>Next</Button></div>
    </section>
  </div>
}
