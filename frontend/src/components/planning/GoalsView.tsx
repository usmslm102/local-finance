import { useId, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Pencil, Trash2, CircleCheck } from 'lucide-react'
import { fetchSavingsGoals, saveSavingsGoal, deleteSavingsGoal } from '@/lib/api'
import { goalProgress } from '@/lib/planning'
import { formatINR, formatDate } from '@/lib/utils'
import type { GoalInput, SavingsGoal } from '@/types/planning'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Input } from '@/components/ui/input'
import { Field, FieldGroup, FieldLabel, FieldDescription } from '@/components/ui/field'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import {
  Empty,
  EmptyHeader,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from '@/components/ui/empty'
import { PageLoading, QueryError } from '@/components/ui/query-state'

function GoalEditor({ goal, onClose }: { goal?: SavingsGoal; onClose: () => void }) {
  const id = useId()
  const { isPrivacyMode } = usePrivacy()
  const client = useQueryClient()
  const [name, setName] = useState(goal?.name ?? '')
  const [target, setTarget] = useState(String(goal?.target_amount ?? ''))
  const [saved, setSaved] = useState(String(goal?.saved_amount ?? 0))
  const [monthly, setMonthly] = useState(String(goal?.monthly_contribution ?? 0))
  const [date, setDate] = useState(goal?.target_date ?? `${new Date().getFullYear() + 1}-01-01`)
  const mutation = useMutation({
    mutationFn: (input: GoalInput) => saveSavingsGoal(input, goal?.id),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['savings-goals'] })
      onClose()
    },
  })
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !mutation.isPending) onClose()
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{goal ? 'Edit savings goal' : 'New savings goal'}</DialogTitle>
          <DialogDescription>
            Record your plan. Saving a goal does not move money or change your ledger.
          </DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(e) => {
            e.preventDefault()
            const form = new FormData(e.currentTarget)
            mutation.mutate({
              name: String(form.get('name')),
              target_amount: Number(form.get('target')),
              saved_amount: Number(form.get('saved')),
              monthly_contribution: Number(form.get('monthly')),
              target_date: String(form.get('target_date')),
            })
          }}
          className="flex flex-col gap-6"
        >
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor={`${id}-name`}>Goal name</FieldLabel>
              <Input
                id={`${id}-name`}
                name="name"
                value={isPrivacyMode ? 'Hidden in Discreet Mode' : name}
                disabled={isPrivacyMode}
                onChange={(e) => setName(e.target.value)}
                maxLength={100}
                required
                placeholder="Emergency fund, holiday, new laptop…"
              />
            </Field>
            {[
              {
                key: 'target',
                label: 'Target amount (₹)',
                value: target,
                set: setTarget,
                min: 0.01,
              },
              { key: 'saved', label: 'Already saved (₹)', value: saved, set: setSaved, min: 0 },
              {
                key: 'monthly',
                label: 'Planned monthly contribution (₹)',
                value: monthly,
                set: setMonthly,
                min: 0,
              },
            ].map((field) => (
              <Field key={field.key}>
                <FieldLabel htmlFor={`${id}-${field.key}`}>{field.label}</FieldLabel>
                <Input
                  id={`${id}-${field.key}`}
                  name={field.key}
                  type={isPrivacyMode ? 'password' : 'number'}
                  value={isPrivacyMode ? '••••••' : field.value}
                  disabled={isPrivacyMode}
                  min={field.min}
                  max={1e12}
                  step="0.01"
                  required
                  onChange={(e) => field.set(e.target.value)}
                />
              </Field>
            ))}
            <Field>
              <FieldLabel htmlFor={`${id}-date`}>Target date</FieldLabel>
              <Input
                id={`${id}-date`}
                name="target_date"
                type="date"
                min="1900-01-01"
                max="2200-12-31"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                required
              />
              <FieldDescription>
                Monthly pacing includes this month and the target month. Update saved amounts
                yourself as you contribute.
              </FieldDescription>
            </Field>
          </FieldGroup>
          {isPrivacyMode && (
            <p className="text-sm text-muted-foreground">
              Turn off Discreet Mode to edit financial values.
            </p>
          )}
          {mutation.error && (
            <Alert variant="destructive">
              <AlertTitle>Goal could not be saved</AlertTitle>
              <AlertDescription>{mutation.error.message}</AlertDescription>
            </Alert>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" disabled={mutation.isPending} onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={mutation.isPending || isPrivacyMode}>
              {mutation.isPending ? 'Saving…' : 'Save goal'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

export function GoalsView() {
  const { isPrivacyMode } = usePrivacy()
  const money = (amount: number) => formatINR(amount, isPrivacyMode)
  const query = useQuery({ queryKey: ['savings-goals'], queryFn: fetchSavingsGoals })
  const client = useQueryClient()
  const [editor, setEditor] = useState<SavingsGoal | 'new' | null>(null)
  const [deleting, setDeleting] = useState<SavingsGoal | null>(null)
  const deletion = useMutation({
    mutationFn: deleteSavingsGoal,
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ['savings-goals'] })
      setDeleting(null)
    },
  })
  if (query.isPending) return <PageLoading label="Loading savings goals" />
  if (query.isError) return <QueryError message={query.error.message} onRetry={query.refetch} />
  const goals = query.data
  const contribution = goals
    .filter((g) => g.saved_amount < g.target_amount)
    .reduce((sum, g) => sum + g.monthly_contribution, 0)
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-xl font-semibold">Give your savings a destination</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {goals.length} {goals.length === 1 ? 'goal' : 'goals'} · {money(contribution)} planned per month across unfinished goals.
          </p>
        </div>
        <Button onClick={() => setEditor('new')}>
          <Plus data-icon="inline-start" />
          Add goal
        </Button>
      </div>
      {goals.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>Start with one goal</EmptyTitle>
            <EmptyDescription>
              Plan an emergency buffer or a future purchase. Your goal balances stay separate from
              imported bank balances.
            </EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button variant="outline" onClick={() => setEditor('new')}>
              Create your first goal
            </Button>
          </EmptyContent>
        </Empty>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {goals.map((goal) => {
            const progress = goalProgress(goal)
            return (
              <Card key={goal.id}>
                <CardHeader>
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <CardTitle className="break-words">
                      {isPrivacyMode ? 'Savings goal' : goal.name}
                    </CardTitle>
                    <Badge variant={progress.onTrack ? 'secondary' : 'outline'}>
                      {progress.completed
                        ? 'Reached'
                        : progress.overdue
                          ? 'Past target date'
                          : progress.onTrack
                            ? 'On pace'
                            : 'Contribution below target'}
                    </Badge>
                  </div>
                  <CardDescription>
                    Target {formatDate(goal.target_date)} · updated {formatDate(goal.updated_at)}
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-4">
                  <div className="flex flex-wrap justify-between gap-2 text-sm">
                    <span>{money(goal.saved_amount)} saved</span>
                    <span className="text-muted-foreground">of {money(goal.target_amount)}</span>
                  </div>
                  {isPrivacyMode ? (
                    <p className="text-sm text-muted-foreground">
                      Progress hidden in Discreet Mode
                    </p>
                  ) : (
                    <Progress
                      value={progress.percent}
                      aria-label={`${goal.name} savings progress`}
                    />
                  )}
                  <dl className="grid grid-cols-2 gap-3 text-sm">
                    <div>
                      <dt className="text-muted-foreground">Monthly plan</dt>
                      <dd className="font-medium">{money(goal.monthly_contribution)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">
                        {progress.completed
                          ? 'Remaining'
                          : progress.overdue
                            ? 'Still needed'
                            : 'Monthly amount needed'}
                      </dt>
                      <dd className="font-medium">
                        {money(
                          progress.completed || progress.overdue
                            ? progress.remaining
                            : progress.requiredMonthly
                        )}
                      </dd>
                    </div>
                  </dl>
                  <p className="text-sm text-muted-foreground">
                    {progress.completed ? (
                      <span className="flex items-center gap-2">
                        <CircleCheck className="size-4 text-success" />
                        You reached your recorded target.
                      </span>
                    ) : progress.overdue ? (
                      'Choose a new date or update your saved amount to keep the plan useful.'
                    ) : (
                      `Spread ${money(progress.remaining)} over ${progress.monthsLeft} monthly contributions, starting this month.`
                    )}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button variant="outline" size="sm" onClick={() => setEditor(goal)}>
                      <Pencil data-icon="inline-start" />
                      Update progress
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        deletion.reset()
                        setDeleting(goal)
                      }}
                    >
                      <Trash2 data-icon="inline-start" />
                      Delete
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
      <p className="text-sm text-muted-foreground">
        Goal amounts are your own records, not additional assets. Monthly contributions appear as
        reserved money in the forecast until the recorded target is reached.
      </p>
      {editor && (
        <GoalEditor goal={editor === 'new' ? undefined : editor} onClose={() => setEditor(null)} />
      )}
      <Dialog
        open={!!deleting}
        onOpenChange={(open) => {
          if (!open && !deletion.isPending) setDeleting(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete savings goal?</DialogTitle>
            <DialogDescription>
              {isPrivacyMode ? 'This goal' : deleting?.name} will be removed from your plan. Bank
              balances and transactions stay unchanged.
            </DialogDescription>
          </DialogHeader>
          {deletion.error && (
            <Alert variant="destructive">
              <AlertTitle>Delete failed</AlertTitle>
              <AlertDescription>{deletion.error.message}</AlertDescription>
            </Alert>
          )}
          <DialogFooter>
            <Button
              variant="outline"
              disabled={deletion.isPending}
              onClick={() => setDeleting(null)}
            >
              Keep goal
            </Button>
            <Button
              variant="destructive"
              disabled={deletion.isPending}
              onClick={() => deleting && deletion.mutate(deleting.id)}
            >
              {deletion.isPending ? 'Deleting…' : 'Delete goal'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
