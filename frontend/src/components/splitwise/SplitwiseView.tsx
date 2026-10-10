import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { splitwiseRequest } from '@/lib/api'
import type { SplitwiseEntry } from '@/types/splitwise'
import type { Transaction } from '@/types'
import { SplitwiseImporter } from '@/components/import/SplitwiseImporter'
import { SplitwiseSettlements } from '@/components/import/SplitwiseSettlements'
import { Upload, SlidersHorizontal } from 'lucide-react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { PrivacyAmount } from '@/components/ui/privacy-amount'

interface Mapping { entry: SplitwiseEntry; bank?: Transaction }

function personalShare(entry: SplitwiseEntry) {
  return entry.kind === 'PAYMENT' ? 0 : entry.share_cents / 100
}

function EntryBadge({ entry }: { entry: SplitwiseEntry }) {
  return <Badge variant="secondary">{entry.kind === 'PAYMENT' ? 'Self transfer · CSV payment matched' : 'Split expense'}</Badge>
}

function BankMovement({ bank }: { bank?: Transaction }) {
  return bank ? <div className="space-y-1"><p className="text-sm break-words">{bank.raw_narration}</p><p className="text-xs text-muted-foreground">{bank.tx_date.split('T')[0]} · {bank.account_name}</p><p className="text-xs text-muted-foreground">{bank.tx_type === 'CREDIT' ? 'Received' : 'Paid'} <PrivacyAmount amount={bank.amount} /></p></div> : <p className="text-sm text-muted-foreground">Paid by someone else · no bank movement</p>
}

export function SplitwiseView() {
  const client = useQueryClient()
  const mappings = useQuery({ queryKey: ['splitwise-mappings'], queryFn: () => splitwiseRequest<Mapping[]>('/mappings') })
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  const [importOpen, setImportOpen] = useState(false)
  const [rulesOpen, setRulesOpen] = useState(false)
  const [kind, setKind] = useState('ALL')
  const rows = (mappings.data ?? []).filter(m => (kind === 'ALL' || m.entry.kind === kind) && `${m.entry.description} ${m.entry.group} ${m.entry.date} ${m.bank?.raw_narration ?? ''} ${m.bank?.account_name ?? ''}`.toLowerCase().includes(search.toLowerCase()))
  const currentPage = Math.min(page, Math.max(0, Math.ceil(rows.length / 25) - 1))
  const visible = rows.slice(currentPage * 25, currentPage * 25 + 25)
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
    <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4"><div><h1 className="text-2xl font-bold tracking-tight">Splitwise</h1><p className="text-sm text-muted-foreground mt-1">Your share of group expenses, connected to your bank.</p></div><div className="flex gap-2"><Button variant="outline" onClick={() => setRulesOpen(true)}><SlidersHorizontal className="h-4 w-4" />Matching rules</Button><Button onClick={() => setImportOpen(true)}><Upload className="h-4 w-4" />Import CSV</Button></div></div>
    <div className="grid gap-3 sm:grid-cols-3">
      <Card><CardContent className="p-5"><p className="text-xs text-muted-foreground">Your expense share</p><p className="text-2xl font-semibold mt-2"><PrivacyAmount amount={expense / 100} /></p><p className="text-xs text-muted-foreground mt-1">Repayments excluded</p></CardContent></Card>
      <Card><CardContent className="p-5"><p className="text-xs text-muted-foreground">Applied entries</p><p className="text-2xl font-semibold mt-2">{mappings.data?.length ?? 0}</p></CardContent></Card>
      <Card><CardContent className="p-5"><p className="text-xs text-muted-foreground">Matched payments</p><p className="text-2xl font-semibold mt-2">{mappings.data?.filter(m => m.entry.kind === 'PAYMENT').length ?? 0}</p><p className="text-xs text-muted-foreground mt-1">CSV and bank evidence required</p></CardContent></Card>
    </div>
    {(message || mappings.error) && <Alert><AlertDescription>{message || mappings.error?.message}</AlertDescription></Alert>}
    <Card><CardContent className="p-4 sm:p-6 space-y-4">
      <div><h2 className="text-lg font-semibold">Matched transactions</h2><p className="text-sm text-muted-foreground">Original bank amounts stay intact. Only your share counts as an expense.</p></div>
      <Input aria-label="Search Splitwise mappings" placeholder="Search description, group or bank narration" value={search} onChange={e => { setSearch(e.target.value); setPage(0) }} />
      <div className="flex gap-1">{[['ALL','All'],['EXPENSE','Expenses'],['PAYMENT','Payments']].map(([value,label]) => <Button key={value} size="sm" variant={kind === value ? 'secondary' : 'ghost'} aria-pressed={kind === value} onClick={() => { setKind(value); setPage(0) }}>{label}</Button>)}</div>
      <div className="md:hidden space-y-3">{visible.map(({entry:e,bank}) => <div key={e.id} className="border rounded-lg p-4 space-y-3">
        <div><p className="font-medium break-words">{e.description}</p><p className="text-xs text-muted-foreground mt-1">{e.date} · {e.group}</p></div><EntryBadge entry={e} />
        <div className="border-t pt-3"><p className="text-xs text-muted-foreground mb-1">Bank movement</p><BankMovement bank={bank} /></div>
        <div className="flex justify-between items-center gap-2 border-t pt-3"><div><p className="text-xs text-muted-foreground">Your expense</p><PrivacyAmount amount={personalShare(e)} /></div><Button variant="outline" size="sm" disabled={busy} aria-label={`Remove ${e.description}`} onClick={() => { void remove(e) }}>Remove</Button></div>
      </div>)}</div>
      <div className="hidden md:block"><Table><TableHeader><TableRow><TableHead>Splitwise entry</TableHead><TableHead>Your expense</TableHead><TableHead>Matched bank transaction</TableHead><TableHead>Action</TableHead></TableRow></TableHeader>
        <TableBody>{visible.map(({ entry: e, bank }) => <TableRow key={e.id}>
          <TableCell className="max-w-[280px] whitespace-normal"><p className="font-medium">{e.description}</p><p className="text-xs text-muted-foreground">{e.date} · {e.group}</p><EntryBadge entry={e} /></TableCell>
          <TableCell><PrivacyAmount amount={personalShare(e)} /></TableCell>
          <TableCell className="max-w-[360px] whitespace-normal"><BankMovement bank={bank} /></TableCell>
          <TableCell><Button variant="outline" size="sm" disabled={busy} onClick={() => { void remove(e) }}>Remove</Button></TableCell>
        </TableRow>)}</TableBody></Table>
      </div>
      {mappings.isPending && <p className="text-sm">Loading matches…</p>}
      {!mappings.isPending && !mappings.error && !rows.length && <p className="text-sm text-muted-foreground">No applied entries found.</p>}
      <div className="flex flex-wrap items-center justify-between gap-2 border-t pt-4"><span className="text-xs text-muted-foreground">{rows.length ? `${currentPage*25+1}–${Math.min((currentPage+1)*25,rows.length)} of ${rows.length}` : '0 entries'}</span><div className="flex gap-2"><Button size="sm" variant="outline" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}>Previous</Button><Button size="sm" variant="outline" disabled={(currentPage + 1) * 25 >= rows.length} onClick={() => setPage(currentPage + 1)}>Next</Button></div></div>
    </CardContent></Card>
    <SplitwiseSettlements />
    <Dialog open={importOpen} onOpenChange={setImportOpen}><DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto"><DialogHeader><DialogTitle>Import Splitwise</DialogTitle><DialogDescription>Import bank statements first, then match your group history automatically.</DialogDescription></DialogHeader><SplitwiseImporter showMemberRules={false} onImported={result => { setImportOpen(false); setPage(0); setMessage(`Imported: ${result.matched} bank matches, ${result.paid_by_others} paid-by-other expenses and ${result.transfers} matched payments. Skipped ${result.skipped} unmatched and ${result.uninvolved} uninvolved entries.`) }} /></DialogContent></Dialog>
    <Dialog open={rulesOpen} onOpenChange={setRulesOpen}><DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto"><DialogHeader><DialogTitle>Payment matching rules</DialogTitle><DialogDescription>Add a suggested rule in one step, or edit a member’s bank pattern. These rules also appear in Settings.</DialogDescription></DialogHeader><SplitwiseSettlements /></DialogContent></Dialog>
  </div>
}
