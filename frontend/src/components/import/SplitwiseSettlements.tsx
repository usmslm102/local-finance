import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { splitwiseRequest } from '@/lib/api'
import type { SplitwiseMember } from '@/types/splitwise'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'

function MemberPattern({ member, busy, save }: { member: SplitwiseMember; busy: boolean; save: (member: SplitwiseMember) => void }) {
  const [pattern, setPattern] = useState(member.pattern)
  const id = `sw-pattern-${encodeURIComponent(member.group)}-${encodeURIComponent(member.name)}`
  return <div className="space-y-2">
    <Label htmlFor={id}>{member.name} · {member.group}</Label>
    <div className="flex gap-2"><Input id={id} value={pattern} disabled={busy} placeholder="Optional regex, e.g. (?i)asha(@|\\s)" onChange={e => setPattern(e.target.value)} />
      <Button variant="outline" disabled={busy} onClick={() => save({ ...member, pattern })}>Save & apply</Button></div>
  </div>
}

export function SplitwiseSettlements() {
  const client = useQueryClient()
  const members = useQuery({ queryKey: ['splitwise-members'], queryFn: () => splitwiseRequest<SplitwiseMember[]>('/members') })
  const [busy, setBusy] = useState(false)
  const [message, setMessage] = useState('')
  async function save(member: SplitwiseMember) {
    setBusy(true); setMessage('')
    try { await splitwiseRequest('/members', member); await client.invalidateQueries(); setMessage('Saved. Matching outgoing payments are now transfers; the rule also applies to future statement imports.') }
    catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to save member matching') }
    finally { setBusy(false) }
  }
  return <Card><CardContent className="space-y-4 pt-6">
    <h2 className="font-medium">Automatic member transfers</h2>
    <p className="text-sm text-muted-foreground">Full member names identify outgoing payments automatically. Add a regex to match how each person appears in your bank payee, narration or UPI ID. Use (?i) for case-insensitive matching. Repayments change bank cash but add no expense. Incoming payments are not automatically treated as self transfers.</p>
    {(message || members.error) && <Alert><AlertDescription>{message || members.error?.message}</AlertDescription></Alert>}
    {!members.isPending && !members.data?.length && <p className="text-sm">Import a group CSV to load its members.</p>}
    {members.data?.map(member => <MemberPattern key={`${member.group}\0${member.name}`} member={member} busy={busy} save={m => { void save(m) }} />)}
  </CardContent></Card>
}
