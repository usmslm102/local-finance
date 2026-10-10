import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import type { SplitwiseMember } from '@/types/splitwise'
import { useSplitwiseMembers, useSaveSplitwiseMember } from '@/hooks/use-splitwise-members'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { TableCell, TableRow } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

function MemberPattern({ member }: { member: SplitwiseMember }) {
  const mutation = useSaveSplitwiseMember()
  const [draft, setDraft] = useState({ base: member.pattern, value: member.pattern })
  const pattern = draft.base === member.pattern ? draft.value : member.pattern
  const busy = mutation.isPending
  const [message, setMessage] = useState('')
  const id = `sw-pattern-${encodeURIComponent(member.group)}-${encodeURIComponent(member.name)}`
  async function save(nextPattern = pattern) {
    setMessage('')
    try {
      await mutation.mutateAsync({ ...member, pattern: nextPattern })
      setDraft({ base: nextPattern, value: nextPattern })
      setMessage('Saved. Import or re-import your CSV to match its Payment entries using this rule. Bank transactions without a matching Splitwise Payment are unchanged.')
    } catch (error) { setMessage(error instanceof Error ? error.message : 'Unable to save member matching') }
  }
  return <div className="space-y-2">
    <Label htmlFor={id}>{member.name} · {member.group}</Label>
    <p className="text-xs text-muted-foreground">{member.pattern ? <>Saved regex: <code className="break-all">{member.pattern}</code></> : 'Full member name matching; no custom regex saved.'} <Badge variant="secondary">{pattern !== member.pattern ? 'Unsaved changes' : 'Saved'}</Badge></p>
    <p className="text-xs text-muted-foreground">{member.matched_payments} matched Splitwise payments</p>
    {!member.pattern && member.suggested_pattern && <div className="rounded-md border p-3 space-y-2"><p className="text-sm">Suggested from a matched bank payment: <code className="break-all">{member.suggested_pattern}</code></p><Button variant="outline" size="sm" disabled={busy} onClick={() => { void save(member.suggested_pattern) }}>Add suggested rule</Button></div>}
    <div className="flex flex-col sm:flex-row gap-2"><Input id={id} value={pattern} disabled={busy} placeholder="Optional regex, e.g. (?i)asha@fictional" onChange={e => setDraft({ base: member.pattern, value: e.target.value })} />
      <Button variant="outline" disabled={busy || pattern === member.pattern} onClick={() => { void save() }}>{busy ? 'Saving…' : 'Save rule'}</Button></div>
    {message && <Alert><AlertDescription>{message}</AlertDescription></Alert>}
  </div>
}

export function SplitwiseMemberRuleRow({ member }: { member: SplitwiseMember }) {
  const [open, setOpen] = useState(false)
  return <TableRow className="text-xs">
    <TableCell>Splitwise payment</TableCell>
    <TableCell>Payee / narration / UPI</TableCell>
    <TableCell><Badge variant="outline">REGEX</Badge></TableCell>
    <TableCell className="max-w-[240px]"><code className="break-all">{member.pattern}</code><p className="text-muted-foreground">{member.name} · {member.group}</p></TableCell>
    <TableCell>Self transfer</TableCell>
    <TableCell><Badge variant="secondary">Active · Payment required</Badge></TableCell>
    <TableCell className="text-right"><Button variant="outline" size="sm" onClick={() => setOpen(true)}>Edit member rule</Button>
      <Dialog open={open} onOpenChange={setOpen}><DialogContent><DialogHeader><DialogTitle>Splitwise member rule</DialogTitle><DialogDescription>Matches the other member of an imported Splitwise Payment to a bank payee. Amount, direction and the preceding 15-day window must also match. Names alone never create transfers.</DialogDescription></DialogHeader>
        <MemberPattern member={member} />
      </DialogContent></Dialog>
    </TableCell>
  </TableRow>
}

function MemberSummary({ member }: { member: SplitwiseMember }) {
  const [open, setOpen] = useState(false)
  const mutation = useSaveSplitwiseMember()
  return <div className="rounded-lg border bg-background/50 p-4 space-y-3">
    <div className="flex items-start justify-between gap-3"><div><p className="font-medium text-sm">{member.name}</p><p className="text-xs text-muted-foreground">{member.group} · {member.matched_payments} matched payments</p></div><Badge variant={member.pattern ? 'secondary' : 'outline'}>{member.pattern ? 'Rule saved' : 'Name matching'}</Badge></div>
    {member.pattern && <code className="block text-xs break-all text-muted-foreground">{member.pattern}</code>}
    {!member.pattern && member.suggested_pattern && <div className="space-y-2"><p className="text-xs text-muted-foreground">Suggested from a matched bank payment</p><code className="block text-xs break-all">{member.suggested_pattern}</code><Button size="sm" disabled={mutation.isPending} onClick={() => mutation.mutate({ ...member, pattern: member.suggested_pattern })}>{mutation.isPending ? 'Adding…' : 'Add suggested rule'}</Button></div>}
    {mutation.error && <Alert><AlertDescription>{mutation.error.message}</AlertDescription></Alert>}
    <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>{member.pattern ? 'Edit rule' : 'Set a custom regex'}</Button>
    <Dialog open={open} onOpenChange={setOpen}><DialogContent><DialogHeader><DialogTitle>Payment matching · {member.name}</DialogTitle><DialogDescription>Identify this member in bank payee, narration or UPI fields. A matching Splitwise Payment, amount and direction are required.</DialogDescription></DialogHeader><MemberPattern member={member} /></DialogContent></Dialog>
  </div>
}

export function SplitwiseSettlements() {
  const members = useSplitwiseMembers()
  return <Card><CardContent className="space-y-4 pt-6">
    <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-2"><div><h2 className="font-semibold">Payment matching rules</h2><p className="text-sm text-muted-foreground mt-1">Names and regexes identify a member. Transfers always require a matching Splitwise Payment and bank amount.</p></div><Link to="/settings" search={{ tab: 'rules' }} className="text-sm underline shrink-0">All rules</Link></div>
    {members.error && <Alert><AlertDescription>{members.error.message}</AlertDescription></Alert>}
    {members.isPending && <p className="text-sm text-muted-foreground">Loading member rules…</p>}
    {!members.isPending && !members.error && !members.data?.length && <p className="text-sm text-muted-foreground">Import your group CSV to load members. Rule suggestions appear after payments are matched.</p>}
    <div className="grid gap-3 lg:grid-cols-2">{members.data?.map(member => <MemberSummary key={`${member.group}\0${member.name}`} member={member} />)}</div>
  </CardContent></Card>
}
