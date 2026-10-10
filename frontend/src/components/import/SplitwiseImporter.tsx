import { useState } from 'react'
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

interface ImportResult {
  inserted: number
  duplicates: number
  matched: number
  paid_by_others: number
  skipped: number
  uninvolved: number
  transfers: number
}

export function SplitwiseImporter() {
  const client = useQueryClient()
  const [person, setPerson] = useState('Sanjay')
  const [group, setGroup] = useState('')
  const [file, setFile] = useState<File>()
  const [preview, setPreview] = useState<SplitwiseEntry[]>()
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState(false)
  async function run(action: () => Promise<void>) {
    setBusy(true); setMessage('')
    try { await action() } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to import Splitwise') }
    finally { setBusy(false) }
  }
  return <div className="space-y-4">
    <Card><CardContent className="space-y-4 pt-6">
      <p className="text-sm text-muted-foreground">Import your group CSV. Your expense shares appear in Transactions with their descriptions, starting in Others and following your normal category rules. Payments you make to group members are transfers, so repayments do not count as another expense.</p>
      <p className="text-sm text-muted-foreground">Bank matches use exact amounts first, then the nearest date in the preceding 15 days, with up to ₹5 difference. Entries needing a bank match are skipped when none is found. Expenses paid by others need no bank match.</p>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2"><Label htmlFor="sw-person">Your member name</Label><Input id="sw-person" value={person} disabled={busy} onChange={e => { setPerson(e.target.value); setPreview(undefined) }} /></div>
        <div className="space-y-2"><Label htmlFor="sw-group">Group name</Label><Input id="sw-group" value={group} disabled={busy} onChange={e => { setGroup(e.target.value); setPreview(undefined) }} /></div>
      </div>
      <Input aria-label="Splitwise CSV" type="file" accept=".csv" disabled={busy} onChange={e => { setFile(e.target.files?.[0]); setPreview(undefined) }} />
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="outline" disabled={busy || !file || !group.trim() || !person.trim()} onClick={() => run(async () => { setPreview(await splitwiseUpload<SplitwiseEntry[]>('preview', file!, group, person)) })}>Preview CSV</Button>
        <Button disabled={busy || !file || !group.trim() || !person.trim()} onClick={() => run(async () => {
          const result = await splitwiseUpload<ImportResult>('import', file!, group, person)
          setPreview(undefined); await client.invalidateQueries()
          setMessage(`Imported automatically: ${result.matched} bank matches, ${result.paid_by_others} expenses paid by others, ${result.transfers} member payments treated as transfers. Skipped ${result.skipped} unmatched and ${result.uninvolved} uninvolved entries; ${result.duplicates} existing entries kept.`)
        })}>Import automatically</Button>
        <Link to="/transactions" className="text-sm underline">View transactions</Link>
      </div>
      {preview && <p className="text-sm">{preview.length} entries · Members: {preview[0]?.members.join(', ')} · Preview only, nothing saved.</p>}
    </CardContent></Card>
    {message && <Alert><AlertDescription>{message}</AlertDescription></Alert>}
    <SplitwiseSettlements />
  </div>
}
