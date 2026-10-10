import { useState } from 'react'
import { FileSpreadsheet, Upload, CheckCircle2 } from 'lucide-react'
import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { splitwiseUpload } from '@/lib/api'
import type { SplitwiseEntry } from '@/types/splitwise'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { SplitwiseSettlements } from './SplitwiseSettlements'

export interface SplitwiseImportResult {
  inserted: number
  duplicates: number
  matched: number
  paid_by_others: number
  skipped: number
  uninvolved: number
  transfers: number
}

export function SplitwiseImporter({ showMemberRules = true, onImported }: { showMemberRules?: boolean; onImported?: (result: SplitwiseImportResult) => void }) {
  const client = useQueryClient()
  const [person, setPerson] = useState('Sanjay')
  const [group, setGroup] = useState('')
  const [file, setFile] = useState<File>()
  const [preview, setPreview] = useState<SplitwiseEntry[]>()
  const [message, setMessage] = useState('')
  const [result, setResult] = useState<SplitwiseImportResult>()
  const [busy, setBusy] = useState(false)
  async function run(action: () => Promise<void>) {
    setBusy(true); setMessage(''); setResult(undefined)
    try { await action() } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to import Splitwise') }
    finally { setBusy(false) }
  }
  return <div className="space-y-4">
    <Card><CardContent className="space-y-4 pt-6">
      <div className="flex items-start gap-3"><div className="rounded-lg bg-primary/10 p-2.5"><FileSpreadsheet className="h-5 w-5 text-primary" /></div><div><h2 className="font-semibold">Import a Splitwise group</h2><p className="text-sm text-muted-foreground mt-1">Your shares become expenses. Matching payments become transfers.</p></div></div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2"><Label htmlFor="sw-person">Your member name</Label><Input id="sw-person" value={person} disabled={busy} onChange={e => { setPerson(e.target.value); setPreview(undefined) }} /></div>
        <div className="space-y-2"><Label htmlFor="sw-group">Group name</Label><Input id="sw-group" placeholder="e.g. Hyderabad flat" value={group} disabled={busy} onChange={e => { setGroup(e.target.value); setPreview(undefined) }} /></div>
      </div>
      <div className="rounded-lg border border-dashed bg-muted/20 p-4 space-y-2"><Label htmlFor="sw-file" className="text-sm">Group transaction history (.csv)</Label><Input id="sw-file" aria-label="Splitwise CSV" type="file" accept=".csv" disabled={busy} className="cursor-pointer" onChange={e => { setFile(e.target.files?.[0]); setPreview(undefined); setResult(undefined) }} /><p className="text-xs text-muted-foreground">Export your group history from Splitwise. The file stays on this device.</p></div>
      <p className="text-xs text-muted-foreground">Exact amounts first, then nearest bank date within the preceding 15 days (up to ₹5 difference). Unmatched and uninvolved entries are skipped. Your categories follow normal rules.</p>
      <div className="flex flex-col sm:flex-row sm:flex-wrap gap-2">
        <Button className="order-2" variant="outline" disabled={busy || !file || !group.trim() || !person.trim()} onClick={() => run(async () => { setPreview(await splitwiseUpload<SplitwiseEntry[]>('preview', file!, group, person)) })}>Preview CSV</Button>
        <Button className="order-1" disabled={busy || !file || !group.trim() || !person.trim()} onClick={() => run(async () => {
          const result = await splitwiseUpload<SplitwiseImportResult>('import', file!, group, person)
          setPreview(undefined); await client.invalidateQueries()
          setResult(result); onImported?.(result)
          setMessage(`Imported automatically: ${result.matched} bank matches, ${result.paid_by_others} expenses paid by others, ${result.transfers} CSV payments matched as transfers. Skipped ${result.skipped} unmatched and ${result.uninvolved} uninvolved entries; ${result.duplicates} existing entries kept.`)
        })}><Upload className="h-4 w-4" />{busy ? 'Processing…' : 'Import automatically'}</Button>
        <Link to="/splitwise" className="order-3 text-sm underline sm:self-center">View Splitwise matches</Link>
      </div>
      {preview && <p className="text-sm">{preview.length} entries · Members: {preview[0]?.members.join(', ')} · Preview only, nothing saved.</p>}
    </CardContent></Card>
    {message && <Alert>{result && <CheckCircle2 className="h-4 w-4" />}<AlertDescription>{result ? <div className="space-y-3"><p className="font-medium">Import complete</p><div className="grid grid-cols-2 gap-2 text-sm"><p>{result.matched} bank matches</p><p>{result.paid_by_others} paid-by-other expenses</p><p>{result.transfers} matched payments</p><p>{result.duplicates} existing entries kept</p></div><p className="text-xs">Skipped {result.skipped} unmatched and {result.uninvolved} uninvolved entries.</p><Link to="/splitwise" className="text-sm underline">View matches and suggested rules</Link></div> : message}</AlertDescription></Alert>}
    {showMemberRules && <SplitwiseSettlements />}
  </div>
}
