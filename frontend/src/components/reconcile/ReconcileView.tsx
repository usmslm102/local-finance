import { cn } from '@/lib/utils'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { usePrivacy } from '@/components/privacy-provider'
import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  fetchReconciliationSummary,
  scanReconciliation,
  linkTransferPair,
  unlinkTransferPair,
  toggleExcludeTransaction,
} from '../../lib/api'
import { formatINR } from '../../lib/utils'
import type { TransferPair } from '../../types'
import { Card, CardContent } from '../ui/card'
import { Button } from '../ui/button'
import { Badge } from '../ui/badge'
import { MerchantAvatar } from '../ui/merchant-avatar'
import {
  ArrowLeftRight,
  ShieldCheck,
  Sparkles,
  Ban,
  CheckCircle2,
  RefreshCw,
  Wallet,
  Landmark,
  CreditCard,
  Unlink,
  Check,
  HelpCircle,
} from 'lucide-react'

export const ReconcileView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = useState<'CANDIDATES' | 'PAIRED' | 'WALLET'>('CANDIDATES')
  const [notification, setNotification] = useState<string | null>(null)

  const {
    data: summary,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['reconciliation-summary'],
    queryFn: fetchReconciliationSummary,
  })

  // Mutations
  const scanMutation = useMutation({
    mutationFn: scanReconciliation,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['reconciliation-summary'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      setNotification(data.message)
      setTimeout(() => setNotification(null), 5000)
    },
  })

  const linkMutation = useMutation({
    mutationFn: (pair: { debit_tx_id: string; credit_tx_id: string; match_reason?: string }) =>
      linkTransferPair(pair),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reconciliation-summary'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
    },
  })

  const unlinkMutation = useMutation({
    mutationFn: (txId: string) => unlinkTransferPair(txId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reconciliation-summary'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
    },
  })

  const toggleExcludeMutation = useMutation({
    mutationFn: (txId: string) => toggleExcludeTransaction(txId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['reconciliation-summary'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
    },
  })

  if (isLoading) return <PageLoading label="Loading transfer reconciliation" />
  if (isError || !summary)
    return <QueryError message="Unable to load transfer reconciliation" onRetry={refetch} />

  const candidates = summary.candidates || []
  const pairs = summary.pairs || []
  const walletTransactions = summary.wallet_transactions || []

  return (
    <Tabs
      value={activeTab}
      onValueChange={(value) => {
        if (value === 'CANDIDATES' || value === 'PAIRED' || value === 'WALLET') setActiveTab(value)
      }}
      className="gap-6 max-w-7xl mx-auto pb-12"
    >
      {/* Header Section */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
              Transfer Reconciler
            </h1>
            <Badge variant="secondary" className="gap-1 py-0.5">
              <ShieldCheck className="size-3.5 text-success" />
              Double-Count Protection
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Pair credit card bill payments from your bank with card credits, and exclude wallet load
            transfers to keep expense analytics accurate.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <Button
            onClick={() => scanMutation.mutate()}
            disabled={scanMutation.isPending}
            className="gap-2"
          >
            <RefreshCw
              data-icon="inline-start"
              className={cn('size-4', scanMutation.isPending ? 'animate-spin' : '')}
            />
            <span>{scanMutation.isPending ? 'Scanning Ledger...' : 'Scan & Auto-Pair'}</span>
          </Button>
        </div>
      </div>

      {[scanMutation, linkMutation, unlinkMutation, toggleExcludeMutation].some(
        (mutation) => mutation.isError
      ) && (
        <Alert variant="destructive">
          <AlertTitle>Reconciliation action failed</AlertTitle>
          <AlertDescription>Please try the action again.</AlertDescription>
        </Alert>
      )}
      {/* Notification Toast Banner */}
      {notification && (
        <div className="flex items-center gap-3 rounded-lg border border-success/30 bg-success/10 p-4 text-sm text-success animate-in fade-in slide-in-from-top-2">
          <CheckCircle2 className="size-5 shrink-0" />
          <span className="font-medium">{notification}</span>
        </div>
      )}

      {/* KPI Ribbon */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Double-Count Prevented
              </p>
              <div className="rounded-md bg-success/10 p-2 text-success">
                <ShieldCheck className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {formatINR(summary.double_count_prevented_amount || 0, isPrivacyMode)}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Across {summary.total_paired_transfers || 0} linked internal transfers
            </p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Pending Pair Candidates
              </p>
              <div className="rounded-md bg-warning/10 p-2 text-warning">
                <Sparkles className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {summary.pending_candidates_count || 0}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Potential unlinked bill payments found
            </p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Linked CC Payments
              </p>
              <div className="rounded-md bg-info/10 p-2 text-info">
                <ArrowLeftRight className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {summary.total_paired_transfers || 0}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">Marked with is_transfer = 1</p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Excluded Wallet Loads
              </p>
              <div className="rounded-md bg-info/10 p-2 text-info">
                <Wallet className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {formatINR(summary.wallet_excluded_amount || 0, isPrivacyMode)}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              {summary.wallet_excluded_count || 0} UPI Lite & wallet top-up transactions
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Tabs Navigation */}
      <div>
        <TabsList
          className="w-full flex-wrap h-auto"
          variant="line"
          aria-label="Transfer reconciliation"
        >
          <TabsTrigger value="CANDIDATES">
            Candidates ({summary.pending_candidates_count || 0})
          </TabsTrigger>
          <TabsTrigger value="PAIRED">Linked ({summary.total_paired_transfers || 0})</TabsTrigger>
          <TabsTrigger value="WALLET">Wallet ({summary.wallet_excluded_count || 0})</TabsTrigger>
        </TabsList>
      </div>

      {/* Tab 1: Candidates to Pair */}
      <TabsContent value="CANDIDATES">
        <div className="flex flex-col gap-4">
          {candidates.length === 0 ? (
            <Card className="p-12 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-success/10 text-success mb-3">
                <CheckCircle2 className="size-6" />
              </div>
              <h3 className="text-base font-semibold text-foreground">
                All Set! No Unlinked Bill Payments
              </h3>
              <p className="text-xs text-muted-foreground mt-1 max-w-md mx-auto">
                All identified bank debits and credit card bill payment credits have been matched
                and reconciled.
              </p>
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-4">
              {candidates.map((pair) => (
                <CandidatePairCard
                  key={pair.id}
                  pair={pair}
                  onLink={() =>
                    linkMutation.mutate({
                      debit_tx_id: pair.debit_tx.id,
                      credit_tx_id: pair.credit_tx.id,
                      match_reason: pair.match_reason,
                    })
                  }
                  isLinking={linkMutation.isPending}
                />
              ))}
            </div>
          )}
        </div>
      </TabsContent>

      {/* Tab 2: Linked Transfers */}
      <TabsContent value="PAIRED">
        <div className="flex flex-col gap-4">
          {pairs.length === 0 ? (
            <Card className="p-12 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground mb-3">
                <HelpCircle className="size-6" />
              </div>
              <h3 className="text-base font-semibold text-foreground">No Linked Transfers Yet</h3>
              <p className="text-xs text-muted-foreground mt-1 max-w-md mx-auto">
                Click &quot;Scan &amp; Auto-Pair&quot; above to automatically detect credit card
                bill payment transactions.
              </p>
            </Card>
          ) : (
            <div className="grid grid-cols-1 gap-4">
              {pairs.map((pair) => (
                <PairedTransferCard
                  key={pair.id}
                  pair={pair}
                  onUnlink={() => unlinkMutation.mutate(pair.debit_tx.id)}
                  isUnlinking={unlinkMutation.isPending}
                />
              ))}
            </div>
          )}
        </div>
      </TabsContent>

      {/* Tab 3: Wallet Loads */}
      <TabsContent value="WALLET">
        <div className="flex flex-col gap-4">
          {walletTransactions.length === 0 ? (
            <Card className="p-12 text-center">
              <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted text-muted-foreground mb-3">
                <Wallet className="size-6" />
              </div>
              <h3 className="text-base font-semibold text-foreground">
                No Wallet Load Transactions Found
              </h3>
              <p className="text-xs text-muted-foreground mt-1 max-w-md mx-auto">
                Transactions from UPI Lite or Paytm Wallet top-ups will appear here to prevent
                double counting.
              </p>
            </Card>
          ) : (
            <div className="rounded-xl border border-border bg-card overflow-hidden">
              <div className="divide-y divide-border">
                {walletTransactions.map((tx) => (
                  <div
                    key={tx.id}
                    className="flex flex-wrap items-center justify-between p-4 hover:bg-muted/30 transition-colors gap-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex size-9 items-center justify-center rounded-lg bg-info/10 text-info">
                        <Wallet className="size-4" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-sm text-foreground">
                            {tx.cleaned_payee || tx.raw_narration}
                          </span>
                          <Badge variant="outline">{tx.tx_date}</Badge>
                          {tx.is_excluded && (
                            <Badge variant="secondary" className="text-warning">
                              Excluded from Spend
                            </Badge>
                          )}
                        </div>
                        <p className="text-xs text-muted-foreground mt-0.5 font-mono truncate max-w-lg">
                          {tx.raw_narration} • {tx.account_name}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-4">
                      <span className="text-sm font-bold text-foreground">
                        {formatINR(tx.amount, isPrivacyMode)}
                      </span>
                      <Button
                        size="sm"
                        variant={tx.is_excluded ? 'outline' : 'secondary'}
                        onClick={() => toggleExcludeMutation.mutate(tx.id)}
                        disabled={toggleExcludeMutation.isPending}
                        className="gap-1.5"
                      >
                        <Ban data-icon="inline-start" />
                        <span>{tx.is_excluded ? 'Include in Expenses' : 'Exclude'}</span>
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </TabsContent>
    </Tabs>
  )
}

// Candidate Match Card Component
const CandidatePairCard: React.FC<{
  pair: TransferPair
  onLink: () => void
  isLinking: boolean
}> = ({ pair, onLink, isLinking }) => {
  const { isPrivacyMode } = usePrivacy()
  const confidencePercent = Math.round(pair.match_confidence * 100)

  return (
    <Card className="overflow-hidden">
      <div className="p-5">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3 mb-4">
          <div className="flex items-center gap-2">
            <Badge
              variant="outline"
              className={cn(
                'text-xs font-semibold px-2 py-0.5',
                confidencePercent >= 90
                  ? 'border-success/50 bg-success/10 text-success '
                  : 'border-warning/50 bg-warning/10 text-warning '
              )}
            >
              <Sparkles className="size-3 mr-1 inline" />
              {confidencePercent}% Match Confidence
            </Badge>
            <span className="text-xs text-muted-foreground font-medium">
              {pair.date_difference_days === 0
                ? 'Same-day transaction'
                : `${pair.date_difference_days} day difference`}
            </span>
          </div>

          <Button size="sm" onClick={onLink} disabled={isLinking} className="gap-2">
            <Check data-icon="inline-start" />
            <span>Confirm &amp; Link Transfer</span>
          </Button>
        </div>

        <div className="relative grid grid-cols-1 md:grid-cols-2 gap-4 items-stretch">
          {/* Debit Bank Side */}
          <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-4 flex flex-col gap-2.5 transition-all group-hover:border-destructive/40">
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-destructive flex items-center gap-1.5">
                <Landmark className="size-3.5" /> Savings Bank Outflow
              </span>
              <span className="text-xs font-mono text-muted-foreground">
                {pair.debit_tx.tx_date}
              </span>
            </div>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2.5 min-w-0">
                <MerchantAvatar
                  payee={pair.debit_tx.cleaned_payee || pair.debit_tx.raw_narration}
                  size="sm"
                />
                <div className="min-w-0">
                  <p className="font-semibold text-sm text-foreground truncate">
                    {pair.debit_tx.cleaned_payee || pair.debit_tx.raw_narration}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    {pair.debit_tx.account_name}
                  </p>
                </div>
              </div>
              <p className="text-base font-bold font-mono text-destructive shrink-0 tabular-nums">
                - {formatINR(pair.debit_tx.amount, isPrivacyMode)}
              </p>
            </div>
            <p className="text-[11px] font-mono text-muted-foreground/80 truncate pt-1 border-t border-destructive/10">
              {pair.debit_tx.raw_narration}
            </p>
          </div>

          {/* Credit Card Side */}
          <div className="rounded-xl border border-success/20 bg-success/5 p-4 flex flex-col gap-2.5 transition-all group-hover:border-success/40">
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-success flex items-center gap-1.5">
                <CreditCard className="size-3.5" /> Credit Card Bill Credit
              </span>
              <span className="text-xs font-mono text-muted-foreground">
                {pair.credit_tx.tx_date}
              </span>
            </div>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2.5 min-w-0">
                <MerchantAvatar
                  payee={pair.credit_tx.cleaned_payee || pair.credit_tx.raw_narration}
                  size="sm"
                />
                <div className="min-w-0">
                  <p className="font-semibold text-sm text-foreground truncate">
                    {pair.credit_tx.cleaned_payee || pair.credit_tx.raw_narration}
                  </p>
                  <p className="text-xs text-muted-foreground mt-0.5 truncate">
                    {pair.credit_tx.account_name}
                  </p>
                </div>
              </div>
              <p className="text-base font-bold font-mono text-success shrink-0 tabular-nums">
                + {formatINR(pair.credit_tx.amount, isPrivacyMode)}
              </p>
            </div>
            <p className="text-[11px] font-mono text-muted-foreground/80 truncate pt-1 border-t border-success/10">
              {pair.credit_tx.raw_narration}
            </p>
          </div>
        </div>
      </div>
    </Card>
  )
}

// Paired Transfer Card Component
const PairedTransferCard: React.FC<{
  pair: TransferPair
  onUnlink: () => void
  isUnlinking: boolean
}> = ({ pair, onUnlink, isUnlinking }) => {
  const { isPrivacyMode } = usePrivacy()
  return (
    <Card>
      <div className="p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <div className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <ArrowLeftRight className="size-4" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-semibold text-sm text-foreground">
                  {pair.debit_tx.account_name} ➔ {pair.credit_tx.account_name}
                </span>
                <Badge variant="outline" className="text-success">
                  Linked Transfer
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">
                {pair.debit_tx.tx_date} • {pair.debit_tx.raw_narration}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <span className="text-base font-bold text-foreground">
              {formatINR(pair.debit_tx.amount, isPrivacyMode)}
            </span>
            <Button
              size="sm"
              variant="outline"
              onClick={onUnlink}
              disabled={isUnlinking}
              className="gap-1.5 hover:text-destructive"
            >
              <Unlink data-icon="inline-start" />
              <span>Unlink</span>
            </Button>
          </div>
        </div>
      </div>
    </Card>
  )
}
