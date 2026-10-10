import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import { Field, FieldLabel, FieldGroup } from '@/components/ui/field'
import React, { useState, useId } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  CreditCard,
  Sparkles,
  Plus,
  Trash2,
  CheckCircle2,
  Edit2,
  Search,
  ShieldCheck,
  Flame,
  Clock,
  RefreshCw,
  Percent,
} from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectGroup,
} from '@/components/ui/select'
import {
  fetchCardPortfolioOverview,
  recommendBestCards,
  updateCardMetadata,
  createCardRewardRule,
  deleteCardRewardRule,
} from '@/lib/api'
import { formatINR, formatDate } from '@/lib/utils'
import { CardDetails, CardRewardRule, UpdateCardMetadataRequest } from '@/types'
import { InteractiveCreditCard } from '@/components/cards/InteractiveCreditCard'
import { AnimatedNumber } from '@/components/ui/animated-number'

const QUICK_MERCHANTS = [
  { label: '🍔 Swiggy', keyword: 'SWIGGY' },
  { label: '🛒 Blinkit', keyword: 'BLINKIT' },
  { label: '📦 Amazon', keyword: 'AMAZON' },
  { label: '🛍️ Flipkart', keyword: 'FLIPKART' },
  { label: '🚗 Uber', keyword: 'UBER' },
  { label: '👗 Myntra', keyword: 'MYNTRA' },
  { label: '⚡ Bills', keyword: 'ELECTRICITY' },
  { label: '⛽ Fuel', keyword: 'FUEL' },
  { label: '✈️ Flights', keyword: 'FLIGHT' },
]

export const CardsView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const fieldId = useId()
  const queryClient = useQueryClient()

  // State for Best-Card Finder
  const [selectedMerchant, setSelectedMerchant] = useState('SWIGGY')
  const [customSearch, setCustomSearch] = useState('')
  const [spendAmount, setSpendAmount] = useState<number>(1000)

  // Modals
  const [editingCard, setEditingCard] = useState<CardDetails | null>(null)
  const [cardFormData, setCardFormData] = useState<UpdateCardMetadataRequest>({})
  const [managingRulesCard, setManagingRulesCard] = useState<CardDetails | null>(null)
  const [newRuleData, setNewRuleData] = useState<Partial<CardRewardRule>>({
    merchant_pattern: '',
    category_name: 'General',
    reward_percentage: 5,
    reward_description: '',
  })
  const [showAddRuleDialog, setShowAddRuleDialog] = useState(false)

  // Fetch Portfolio
  const {
    data: portfolio,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['card-portfolio'],
    queryFn: fetchCardPortfolioOverview,
  })

  // Active query keyword for recommender
  const activeKeyword = customSearch.trim() || selectedMerchant

  // Fetch Recommendations for current merchant & spend
  const {
    data: recData,
    isLoading: recLoading,
    isError: recError,
    refetch: refetchRecommendations,
  } = useQuery({
    queryKey: ['card-recommendation', activeKeyword, spendAmount],
    queryFn: () =>
      recommendBestCards({
        merchant: activeKeyword,
        amount: spendAmount || 1000,
      }),
    enabled: !!portfolio && (portfolio?.cards?.length ?? 0) > 0,
  })

  // Card Metadata Mutation
  const updateCardMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateCardMetadataRequest }) =>
      updateCardMetadata(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['card-portfolio'] })
      queryClient.invalidateQueries({ queryKey: ['card-recommendation'] })
      setEditingCard(null)
    },
  })

  // Rule Creation Mutation
  const createRuleMutation = useMutation({
    mutationFn: (data: Partial<CardRewardRule>) => createCardRewardRule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['card-portfolio'] })
      queryClient.invalidateQueries({ queryKey: ['card-recommendation'] })
      setShowAddRuleDialog(false)
      setNewRuleData({
        merchant_pattern: '',
        category_name: 'General',
        reward_percentage: 5,
        reward_description: '',
      })
    },
  })

  // Rule Deletion Mutation
  const deleteRuleMutation = useMutation({
    mutationFn: (ruleId: string) => deleteCardRewardRule(ruleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['card-portfolio'] })
      queryClient.invalidateQueries({ queryKey: ['card-recommendation'] })
    },
  })

  const handleOpenEditCard = (card: CardDetails) => {
    setEditingCard(card)
    setCardFormData({
      card_variant: card.card_variant || undefined,
      card_network: card.card_network || 'VISA',
      card_color: card.card_color || '#1E293B',
      account_holder_name: card.account_holder_name || undefined,
      credit_limit: card.credit_limit || undefined,
      annual_fee: card.annual_fee || 0,
      fee_waiver_threshold: card.fee_waiver_threshold || 0,
      billing_day: card.billing_day || 12,
      payment_due_days: card.payment_due_days || 20,
      base_reward_rate: card.base_reward_rate || 1.0,
      reward_type: card.reward_type || 'CASHBACK',
    })
  }

  if (isLoading) return <PageLoading label="Loading credit cards" />
  if (isError) return <QueryError message="Unable to load credit cards" onRetry={refetch} />

  const cards = portfolio?.cards || []
  const topRecommendation =
    recData?.recommendations && recData.recommendations.length > 0
      ? recData.recommendations[0]
      : null

  return (
    <div className="flex flex-col gap-4 pb-12">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div>
          <h1 className="text-xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <CreditCard className="size-5 text-primary" /> Credit Card Optimizer
          </h1>
          <p className="text-xs text-muted-foreground">
            Compact portfolio intelligence: grace runway, annual fee milestones, reward perks &amp;
            settings
          </p>
        </div>
      </div>

      {/* KPI Top Summary Row - Dense */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {/* 1. Credit Limit & Utilization */}
        <Card className="transition-all">
          <CardContent className="p-3 flex flex-col gap-1.5">
            <div className="flex items-center justify-between text-[11px] text-muted-foreground">
              <span>Total Limit</span>
              <Badge variant="outline" className="px-1 py-0">
                {portfolio?.overall_utilization_rate.toFixed(1)}% Used
              </Badge>
            </div>
            <div className="text-lg font-bold font-mono text-foreground tabular-nums leading-none">
              <AnimatedNumber
                value={portfolio?.total_credit_limit || 0}
                formatFn={(value) => formatINR(value, isPrivacyMode)}
              />
            </div>
            <div className="h-1 w-full rounded-full bg-muted overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  (portfolio?.overall_utilization_rate || 0) > 30 ? 'bg-warning' : 'bg-success'
                )}
                style={{ width: `${Math.min(portfolio?.overall_utilization_rate || 0, 100)}%` }}
              />
            </div>
            <div className="flex justify-between text-[10px] text-muted-foreground font-mono truncate">
              <span>Due: {formatINR(portfolio?.total_outstanding || 0, isPrivacyMode)}</span>
              <span>
                Avail:{' '}
                {formatINR(
                  (portfolio?.total_credit_limit || 0) - (portfolio?.total_outstanding || 0),
                  isPrivacyMode
                )}
              </span>
            </div>
          </CardContent>
        </Card>

        {/* 2. Lifetime Rewards Earned */}
        <Card className="transition-all">
          <CardContent className="p-3 flex flex-col gap-1.5">
            <div className="flex items-center justify-between text-[11px] text-muted-foreground">
              <span>Cashback Earned</span>
              <Sparkles className="size-3 text-success" />
            </div>
            <div className="text-lg font-bold font-mono text-success tabular-nums leading-none">
              <AnimatedNumber
                value={portfolio?.total_cashback_earned || 0}
                formatFn={(value) => formatINR(value, isPrivacyMode)}
              />
            </div>
            <p className="text-[10px] text-muted-foreground font-mono pt-1">
              Points:{' '}
              <strong className="text-warning font-mono">
                {(portfolio?.total_reward_points || 0).toLocaleString('en-IN')} pts
              </strong>
            </p>
          </CardContent>
        </Card>

        {/* 3. Fee Waiver Savings */}
        <Card className="transition-all">
          <CardContent className="p-3 flex flex-col gap-1.5">
            <div className="flex items-center justify-between text-[11px] text-muted-foreground">
              <span>Projected Fee Savings</span>
              <ShieldCheck className="size-3 text-primary" />
            </div>
            <div className="text-lg font-bold font-mono text-foreground tabular-nums leading-none">
              <AnimatedNumber
                value={portfolio?.total_fee_savings_projected || 0}
                formatFn={(value) => formatINR(value, isPrivacyMode)}
              />
            </div>
            <p className="text-[10px] text-muted-foreground font-mono pt-1">
              Fees Liability:{' '}
              <span className="text-foreground">
                {formatINR(portfolio?.total_annual_fee_liability || 0, isPrivacyMode)}
              </span>
            </p>
          </CardContent>
        </Card>

        {/* 4. Best Card Runway Today */}
        <Card className="relative overflow-hidden transition-all">
          <CardContent className="p-3 flex flex-col gap-1.5">
            <div className="flex items-center justify-between text-[11px] text-primary font-semibold">
              <span className="flex items-center gap-1">
                <Flame className="size-3 text-warning fill-amber-400" /> Best Runway
              </span>
              <Badge className="text-warning px-1 py-0">
                {portfolio?.best_card_to_swipe_today?.interest_free_days_remaining || 0}d
              </Badge>
            </div>
            <div className="text-sm font-bold text-foreground truncate leading-none pt-0.5">
              {portfolio?.best_card_to_swipe_today?.card_variant ||
                portfolio?.best_card_to_swipe_today?.bank_name ||
                'None'}
            </div>
            <p className="text-[10px] text-muted-foreground truncate font-mono pt-1">
              Billing Day: {portfolio?.best_card_to_swipe_today?.billing_day || 0}th
            </p>
          </CardContent>
        </Card>
      </div>

      {/* ⚡ Best-Card Recommendation Engine Bar - Compact */}
      <Card className="from-card via-card to-primary/5 overflow-hidden">
        <CardContent className="p-3.5 flex flex-col gap-3">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Sparkles className="size-4 text-warning" />
              <span className="text-xs font-semibold text-foreground">Best Card Recommender</span>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-[11px] text-muted-foreground whitespace-nowrap">Spend:</span>
              <div className="relative w-28">
                <span className="absolute left-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
                  ₹
                </span>
                <Input
                  aria-label="Spend amount"
                  type="number"
                  value={spendAmount}
                  onChange={(e) => setSpendAmount(Math.max(1, Number(e.target.value)))}
                  className="h-7 pl-5"
                />
              </div>
            </div>
          </div>

          {recError && (
            <QueryError
              message="Unable to load card recommendations"
              onRetry={refetchRecommendations}
            />
          )}

          {/* Quick Merchant Buttons */}
          <div className="flex flex-wrap gap-1 items-center">
            {QUICK_MERCHANTS.map((m) => {
              const isSelected = selectedMerchant === m.keyword && !customSearch
              return (
                <Button
                  variant="ghost"
                  aria-pressed={isSelected}
                  key={m.keyword}
                  type="button"
                  onClick={() => {
                    setSelectedMerchant(m.keyword)
                    setCustomSearch('')
                  }}
                  className={cn(
                    'px-2.5 py-1 rounded-md text-[11px] font-medium transition-all duration-150 active:scale-95 cursor-pointer border',
                    isSelected
                      ? 'bg-primary text-primary-foreground border-primary shadow-xs'
                      : 'bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground border-transparent'
                  )}
                >
                  {m.label}
                </Button>
              )
            })}

            {/* Custom Search Input */}
            <div className="relative flex-1 min-w-[140px] max-w-xs">
              <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3 text-muted-foreground" />
              <Input
                aria-label="Or custom merchant"
                placeholder="Or custom merchant..."
                value={customSearch}
                onChange={(e) => setCustomSearch(e.target.value)}
                className="h-7 pl-7"
              />
            </div>
          </div>

          {/* Recommender Result Banner */}
          {recLoading ? (
            <div className="py-2 text-center text-xs text-muted-foreground flex items-center justify-center gap-2">
              <RefreshCw className="size-3.5 animate-spin text-primary" />
              <span>Checking best rates...</span>
            </div>
          ) : topRecommendation ? (
            <div className="rounded-lg border border-primary/30 bg-primary/5 px-3 py-2 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2.5">
                <span className="text-base">🏆</span>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-bold text-xs text-foreground">
                      {topRecommendation.card_variant || topRecommendation.bank_name}
                    </span>
                    <Badge className="text-success gap-0.5 px-1 py-0">
                      <Percent className="size-2" /> {topRecommendation.reward_rate}%
                    </Badge>
                  </div>
                  <p className="text-[10px] text-muted-foreground">
                    {topRecommendation.reward_description || 'Standard reward applies'}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-4 text-xs">
                <div className="text-right">
                  <span className="text-[9px] uppercase tracking-wider text-muted-foreground">
                    Estimated Reward
                  </span>
                  <div className="font-bold font-mono text-success">
                    {formatINR(topRecommendation.estimated_reward, isPrivacyMode)}
                  </div>
                </div>

                <div className="text-right border-l pl-3 border-border/60">
                  <span className="text-[9px] uppercase tracking-wider text-muted-foreground">
                    Runway
                  </span>
                  <div className="font-bold font-mono text-foreground">
                    {topRecommendation.interest_free_days}d
                  </div>
                </div>
              </div>
            </div>
          ) : null}
        </CardContent>
      </Card>

      {/* 💳 2-Column Dense Grid Portfolio */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-bold tracking-tight text-foreground flex items-center gap-1.5">
            <CreditCard className="size-4 text-primary" /> Active Cards ({cards.length})
          </h2>
        </div>

        {cards.length === 0 ? (
          <Card className="p-8 text-center border-dashed">
            <CreditCard className="size-8 text-muted-foreground mx-auto mb-2 opacity-50" />
            <h3 className="font-semibold text-sm text-foreground">No Credit Cards Registered</h3>
            <p className="text-xs text-muted-foreground mt-0.5 max-w-sm mx-auto">
              Import a credit card statement (PDF / CSV) from HDFC, ICICI, Axis, or SBI to activate
              portfolio intelligence.
            </p>
          </Card>
        ) : (
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
            {cards.map((c) => {
              const isBestRunway = portfolio?.best_card_to_swipe_today?.id === c.id
              const hasFee = c.annual_fee > 0
              const isFeeWaived =
                c.fee_waiver_threshold > 0 && c.total_spend_this_year >= c.fee_waiver_threshold
              const creditLimit = c.credit_limit || 0
              const utilizationPct =
                creditLimit > 0 ? ((c.total_due_amount || 0) / creditLimit) * 100 : 0

              // Determine card color scheme
              const cardScheme = (c.card_variant || '').toLowerCase().includes('amazon')
                ? 'amber'
                : (c.card_variant || '').toLowerCase().includes('swiggy')
                  ? 'rose'
                  : (c.card_variant || '').toLowerCase().includes('flipkart')
                    ? 'indigo'
                    : (c.card_variant || '').toLowerCase().includes('rupay')
                      ? 'emerald'
                      : 'slate'

              return (
                <Card
                  key={c.id}
                  className={cn(
                    'border-border/80 bg-card shadow-xs transition-all duration-200 overflow-hidden flex flex-col justify-between',
                    isBestRunway ? 'ring-1 ring-warning/40' : ''
                  )}
                >
                  {/* Card Header */}
                  <div className="border-b bg-muted/20 px-3.5 py-2 flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <h3 className="font-bold text-xs text-foreground truncate">
                        {c.card_variant || c.bank_name}
                      </h3>
                      <Badge variant="outline" className="px-1 py-0 shrink-0">
                        {c.card_network || 'VISA'} •{' '}
                        {c.account_number_mask ? c.account_number_mask.slice(-4) : '0000'}
                      </Badge>
                      {isBestRunway && (
                        <Badge className="text-warning gap-0.5 px-1 py-0 shrink-0">
                          <Flame className="size-2.5 fill-amber-500" /> Best Runway
                        </Badge>
                      )}
                    </div>

                    <div className="flex items-center gap-1 shrink-0">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setManagingRulesCard(c)
                          setShowAddRuleDialog(false)
                        }}
                        className="h-6 px-2 gap-1"
                      >
                        <Sparkles data-icon="inline-start" className="text-warning" />
                        <span>Rules ({c.reward_rules?.length || 0})</span>
                      </Button>

                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`Edit ${c.nickname || c.card_variant || c.bank_name}`}
                        onClick={() => handleOpenEditCard(c)}
                        className="h-6 px-1.5 gap-1"
                      >
                        <Edit2 data-icon="inline-start" />
                      </Button>
                    </div>
                  </div>

                  {/* Card Body: Single Column Animated Card with Dense Details Below */}
                  <CardContent className="p-4 flex flex-col gap-3.5">
                    {/* 1. Hero 3D Animated Card (Single Column Centered) */}
                    <div className="flex justify-center w-full">
                      <InteractiveCreditCard
                        cardName={c.card_variant || 'Credit Card'}
                        bankName={c.bank_name}
                        cardholderName={c.account_holder_name || c.nickname || 'CARDHOLDER'}
                        variant={c.card_variant || 'Credit Card'}
                        network={c.card_network || 'VISA'}
                        last4={
                          c.account_number_mask
                            ? c.account_number_mask.replace(/[^0-9]/g, '').slice(-4)
                            : '8888'
                        }
                        creditLimit={c.credit_limit}
                        totalDue={c.total_due_amount}
                        dueDate={
                          c.next_payment_due_date ? formatDate(c.next_payment_due_date) : undefined
                        }
                        colorScheme={cardScheme}
                      />
                    </div>

                    {/* 2. Credit Utilization Gauge (Full Width) */}
                    <div className="rounded-xl border border-border/70 bg-muted/20 p-2.5 flex flex-col gap-1.5">
                      <div className="flex justify-between items-center text-xs">
                        <span className="text-muted-foreground">Credit Utilization</span>
                        <span className="font-mono font-semibold text-foreground">
                          {utilizationPct.toFixed(1)}% (
                          {formatINR(c.total_due_amount || 0, isPrivacyMode)} used)
                        </span>
                      </div>
                      <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                        <div
                          className={cn(
                            'h-full rounded-full transition-all',
                            utilizationPct > 30 ? 'bg-warning' : 'bg-success'
                          )}
                          style={{ width: `${Math.min(utilizationPct, 100)}%` }}
                        />
                      </div>
                      <div className="flex justify-between items-center text-[10px] font-mono text-muted-foreground">
                        <span>
                          Total Limit:{' '}
                          <strong className="text-foreground">
                            {c.credit_limit ? formatINR(c.credit_limit, isPrivacyMode) : 'N/A'}
                          </strong>
                        </span>
                        <span>
                          Available:{' '}
                          <strong className="text-foreground">
                            {c.credit_limit
                              ? formatINR(c.credit_limit - (c.total_due_amount || 0), isPrivacyMode)
                              : 'N/A'}
                          </strong>
                        </span>
                      </div>
                    </div>

                    {/* 3. 2 Dense Mini-Panels (Runway & Fee Waiver) */}
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5 text-xs">
                      {/* Panel A: Grace Runway & Billing Cycle */}
                      <div className="rounded-xl bg-muted/20 border border-border/70 p-2.5 flex flex-col gap-1.5">
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground flex items-center gap-1 text-[11px] font-medium">
                            <Clock className="size-3 text-info" /> Grace Runway
                          </span>
                          <span className="font-mono font-bold text-foreground text-[11px]">
                            {c.interest_free_days_remaining}d Left
                          </span>
                        </div>
                        <div className="text-[11px] font-medium text-foreground flex justify-between pt-0.5">
                          <span>
                            Statement:{' '}
                            {c.next_statement_date
                              ? formatDate(c.next_statement_date)
                              : `Day ${c.billing_day}th`}
                          </span>
                          <span className="text-muted-foreground text-[10px]">
                            ({c.days_until_statement}d)
                          </span>
                        </div>
                        <div className="text-[11px] text-muted-foreground font-mono flex justify-between">
                          <span>Payment Due:</span>
                          <strong className="text-foreground">
                            {c.next_payment_due_date
                              ? formatDate(c.next_payment_due_date)
                              : `+${c.payment_due_days}d`}
                          </strong>
                        </div>
                      </div>

                      {/* Panel B: Annual Fee Waiver */}
                      <div className="rounded-xl bg-muted/20 border border-border/70 p-2.5 flex flex-col gap-1.5">
                        <div className="flex items-center justify-between text-[11px]">
                          <span className="text-muted-foreground flex items-center gap-1 font-medium">
                            <ShieldCheck className="size-3 text-primary" /> Annual Fee Waiver
                          </span>
                          {isFeeWaived ? (
                            <span className="text-success font-semibold flex items-center gap-0.5 text-[10px]">
                              <CheckCircle2 className="size-2.5" /> Waived
                            </span>
                          ) : hasFee ? (
                            <span className="font-mono text-warning font-semibold text-[10px]">
                              {c.fee_waiver_progress_pct.toFixed(0)}%
                            </span>
                          ) : (
                            <span className="text-muted-foreground text-[10px]">LTF</span>
                          )}
                        </div>

                        {c.fee_waiver_threshold > 0 ? (
                          <div className="flex flex-col gap-1 pt-0.5">
                            <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                              <div
                                className={cn(
                                  'h-full rounded-full',
                                  isFeeWaived ? 'bg-success' : 'bg-primary'
                                )}
                                style={{ width: `${Math.min(c.fee_waiver_progress_pct, 100)}%` }}
                              />
                            </div>
                            <div className="flex justify-between text-[10px] font-mono text-muted-foreground">
                              <span>
                                Spent: {formatINR(c.total_spend_this_year, isPrivacyMode)}
                              </span>
                              <span>
                                Target: {formatINR(c.fee_waiver_threshold, isPrivacyMode)}
                              </span>
                            </div>
                          </div>
                        ) : (
                          <p className="text-[10px] text-muted-foreground pt-1">
                            Lifetime Free (LTF) • Zero renewal liability
                          </p>
                        )}
                      </div>
                    </div>

                    {/* 4. Active Perks Badges */}
                    {c.reward_rules && c.reward_rules.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5 items-center pt-0.5">
                        <span className="text-[10px] text-muted-foreground font-medium mr-0.5">
                          Perks:
                        </span>
                        <span className="rounded-md bg-muted px-1.5 py-0.5 text-[10px] font-mono text-foreground border border-border/60">
                          Base {c.base_reward_rate}% {c.reward_type}
                        </span>
                        {c.reward_rules.map((r) => (
                          <span
                            key={r.id}
                            className="inline-flex items-center gap-1 rounded-md border border-border/60 bg-muted/40 px-1.5 py-0.5 text-[10px] text-foreground"
                          >
                            <strong className="text-warning font-mono">
                              {r.reward_percentage}%
                            </strong>
                            <span className="text-muted-foreground truncate max-w-[110px]">
                              {r.merchant_pattern || r.category_name}
                            </span>
                          </span>
                        ))}
                      </div>
                    ) : (
                      <div className="flex items-center justify-between text-[11px] text-muted-foreground pt-0.5">
                        <span>
                          Standard Base Reward Rate:{' '}
                          <strong className="text-foreground font-mono">
                            {c.base_reward_rate}% {c.reward_type}
                          </strong>
                        </span>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setManagingRulesCard(c)
                            setShowAddRuleDialog(true)
                          }}
                          className="h-5 px-1.5 gap-1"
                        >
                          <Plus data-icon="inline-start" /> Add Perks
                        </Button>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>

      {/* Edit Card Metadata Dialog */}
      <Dialog open={!!editingCard} onOpenChange={(open) => !open && setEditingCard(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Edit Credit Card Details</DialogTitle>
            <DialogDescription className="text-xs">
              Configure billing cycle, limits, and fee waiver targets for{' '}
              {editingCard?.card_variant || editingCard?.bank_name}
            </DialogDescription>
          </DialogHeader>

          <FieldGroup className="flex flex-col gap-3 py-2 text-xs">
            <Field>
              <FieldLabel htmlFor={`${fieldId}-608`} className="text-muted-foreground font-medium">
                Card Variant Name
              </FieldLabel>
              <Input
                id={`${fieldId}-608`}
                value={cardFormData.card_variant || ''}
                onChange={(e) => setCardFormData({ ...cardFormData, card_variant: e.target.value })}
                placeholder="e.g. Regalia Gold / Amazon Pay"
                className="mt-1 h-8"
              />
            </Field>

            <Field>
              <FieldLabel htmlFor={`${fieldId}-618`} className="text-muted-foreground font-medium">
                Cardholder Name
              </FieldLabel>
              <Input
                id={`${fieldId}-618`}
                value={cardFormData.account_holder_name || ''}
                onChange={(e) =>
                  setCardFormData({ ...cardFormData, account_holder_name: e.target.value })
                }
                placeholder="e.g. RAHUL SHARMA"
                className="mt-1 h-8"
              />
            </Field>

            <FieldGroup className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-633`}
                  className="text-muted-foreground font-medium"
                >
                  Card Network
                </FieldLabel>
                <Select
                  value={cardFormData.card_network || 'VISA'}
                  onValueChange={(val) =>
                    setCardFormData({ ...cardFormData, card_network: val ?? 'VISA' })
                  }
                >
                  <SelectTrigger id={`${fieldId}-633`} className="mt-1 h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="VISA">VISA</SelectItem>
                      <SelectItem value="MASTERCARD">Mastercard</SelectItem>
                      <SelectItem value="RUPAY">RuPay</SelectItem>
                      <SelectItem value="AMEX">American Express</SelectItem>
                      <SelectItem value="DINERS">Diners Club</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>

              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-648`}
                  className="text-muted-foreground font-medium"
                >
                  Credit Limit (₹)
                </FieldLabel>
                <Input
                  id={`${fieldId}-648`}
                  type="number"
                  value={cardFormData.credit_limit || ''}
                  onChange={(e) =>
                    setCardFormData({ ...cardFormData, credit_limit: Number(e.target.value) })
                  }
                  placeholder="e.g. 500000"
                  className="mt-1 h-8"
                />
              </Field>
            </FieldGroup>

            <FieldGroup className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-663`}
                  className="text-muted-foreground font-medium"
                >
                  Billing Day (1-31)
                </FieldLabel>
                <Input
                  id={`${fieldId}-663`}
                  type="number"
                  min={1}
                  max={31}
                  value={cardFormData.billing_day || ''}
                  onChange={(e) =>
                    setCardFormData({ ...cardFormData, billing_day: Number(e.target.value) })
                  }
                  className="mt-1 h-8"
                />
              </Field>

              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-677`}
                  className="text-muted-foreground font-medium"
                >
                  Payment Due Days
                </FieldLabel>
                <Input
                  id={`${fieldId}-677`}
                  type="number"
                  min={10}
                  max={30}
                  value={cardFormData.payment_due_days || ''}
                  onChange={(e) =>
                    setCardFormData({ ...cardFormData, payment_due_days: Number(e.target.value) })
                  }
                  className="mt-1 h-8"
                />
              </Field>
            </FieldGroup>

            <FieldGroup className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-693`}
                  className="text-muted-foreground font-medium"
                >
                  Annual Fee (₹)
                </FieldLabel>
                <Input
                  id={`${fieldId}-693`}
                  type="number"
                  value={cardFormData.annual_fee || 0}
                  onChange={(e) =>
                    setCardFormData({ ...cardFormData, annual_fee: Number(e.target.value) })
                  }
                  placeholder="0 for LTF"
                  className="mt-1 h-8"
                />
              </Field>

              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-706`}
                  className="text-muted-foreground font-medium"
                >
                  Waiver Threshold (₹)
                </FieldLabel>
                <Input
                  id={`${fieldId}-706`}
                  type="number"
                  value={cardFormData.fee_waiver_threshold || 0}
                  onChange={(e) =>
                    setCardFormData({
                      ...cardFormData,
                      fee_waiver_threshold: Number(e.target.value),
                    })
                  }
                  placeholder="0 for none"
                  className="mt-1 h-8"
                />
              </Field>
            </FieldGroup>

            <FieldGroup className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-721`}
                  className="text-muted-foreground font-medium"
                >
                  Base Reward Rate (%)
                </FieldLabel>
                <Input
                  id={`${fieldId}-721`}
                  type="number"
                  step="0.1"
                  value={cardFormData.base_reward_rate || 1.0}
                  onChange={(e) =>
                    setCardFormData({ ...cardFormData, base_reward_rate: Number(e.target.value) })
                  }
                  className="mt-1 h-8"
                />
              </Field>

              <Field>
                <FieldLabel
                  htmlFor={`${fieldId}-738`}
                  className="text-muted-foreground font-medium"
                >
                  Reward Type
                </FieldLabel>
                <Select
                  value={cardFormData.reward_type || 'CASHBACK'}
                  onValueChange={(val) =>
                    setCardFormData({ ...cardFormData, reward_type: val ?? 'CASHBACK' })
                  }
                >
                  <SelectTrigger id={`${fieldId}-738`} className="mt-1 h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="CASHBACK">Direct Cashback</SelectItem>
                      <SelectItem value="POINTS">Reward Points</SelectItem>
                      <SelectItem value="MILES">Air Miles</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
            </FieldGroup>
          </FieldGroup>

          {updateCardMutation.isError && (
            <Alert variant="destructive">
              <AlertTitle>Card changes could not be saved</AlertTitle>
              <AlertDescription>Please try saving again.</AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setEditingCard(null)}>
              Cancel
            </Button>
            <Button
              size="sm"
              disabled={updateCardMutation.isPending}
              onClick={() => {
                if (editingCard) {
                  updateCardMutation.mutate({ id: editingCard.id, data: cardFormData })
                }
              }}
            >
              {updateCardMutation.isPending ? 'Saving...' : 'Save Settings'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Manage Rules Dialog */}
      <Dialog
        open={!!managingRulesCard}
        onOpenChange={(open) => !open && setManagingRulesCard(null)}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Sparkles className="size-4 text-warning" /> Reward Rules &amp; Multipliers
            </DialogTitle>
            <DialogDescription className="text-xs">
              Merchant cashback multipliers for{' '}
              {managingRulesCard?.card_variant || managingRulesCard?.bank_name}
            </DialogDescription>
          </DialogHeader>

          {(createRuleMutation.isError || deleteRuleMutation.isError) && (
            <Alert variant="destructive">
              <AlertTitle>Reward rules could not be saved</AlertTitle>
              <AlertDescription>Please try the action again.</AlertDescription>
            </Alert>
          )}

          <div className="flex flex-col gap-4 py-2">
            {/* Rules List */}
            <div className="flex flex-col gap-2 max-h-60 overflow-y-auto">
              {managingRulesCard?.reward_rules && managingRulesCard.reward_rules.length > 0 ? (
                managingRulesCard.reward_rules.map((rule) => (
                  <div
                    key={rule.id}
                    className="flex items-center justify-between rounded-lg border border-border/70 bg-muted/30 p-2.5 text-xs"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <strong className="font-mono text-success font-bold">
                          {rule.reward_percentage}%
                        </strong>
                        <span className="font-semibold text-foreground">
                          {rule.merchant_pattern || rule.category_name}
                        </span>
                      </div>
                      {rule.reward_description && (
                        <p className="text-[11px] text-muted-foreground mt-0.5">
                          {rule.reward_description}
                        </p>
                      )}
                    </div>

                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => deleteRuleMutation.mutate(rule.id)}
                      className="size-7 hover:text-destructive"
                    >
                      <Trash2 data-icon="inline-start" />
                    </Button>
                  </div>
                ))
              ) : (
                <div className="text-center py-6 text-xs text-muted-foreground">
                  No merchant multipliers configured. Uses base rate (
                  {managingRulesCard?.base_reward_rate}%).
                </div>
              )}
            </div>

            {/* Add Rule Form */}
            {showAddRuleDialog ? (
              <div className="rounded-xl border border-border/80 bg-muted/40 p-3.5 flex flex-col gap-3">
                <h4 className="font-semibold text-xs text-foreground">Add New Merchant Rule</h4>
                <FieldGroup className="grid grid-cols-2 gap-2 text-xs">
                  <Field>
                    <FieldLabel
                      htmlFor={`${fieldId}-825`}
                      className="text-muted-foreground text-[10px]"
                    >
                      Merchant / Keyword
                    </FieldLabel>
                    <Input
                      id={`${fieldId}-825`}
                      placeholder="e.g. SWIGGY / BLINKIT"
                      value={newRuleData.merchant_pattern || ''}
                      onChange={(e) =>
                        setNewRuleData({
                          ...newRuleData,
                          merchant_pattern: e.target.value.toUpperCase(),
                        })
                      }
                      className="h-8 mt-1"
                    />
                  </Field>

                  <Field>
                    <FieldLabel
                      htmlFor={`${fieldId}-837`}
                      className="text-muted-foreground text-[10px]"
                    >
                      Reward Percentage (%)
                    </FieldLabel>
                    <Input
                      id={`${fieldId}-837`}
                      type="number"
                      step="0.5"
                      value={newRuleData.reward_percentage || 5}
                      onChange={(e) =>
                        setNewRuleData({
                          ...newRuleData,
                          reward_percentage: Number(e.target.value),
                        })
                      }
                      className="h-8 mt-1"
                    />
                  </Field>
                </FieldGroup>

                <Field>
                  <FieldLabel
                    htmlFor={`${fieldId}-851`}
                    className="text-muted-foreground text-[10px]"
                  >
                    Description (Optional)
                  </FieldLabel>
                  <Input
                    id={`${fieldId}-851`}
                    placeholder="e.g. 5% accelerated cashback"
                    value={newRuleData.reward_description || ''}
                    onChange={(e) =>
                      setNewRuleData({ ...newRuleData, reward_description: e.target.value })
                    }
                    className="h-8 mt-1"
                  />
                </Field>

                <div className="flex justify-end gap-2 pt-1">
                  <Button variant="ghost" size="sm" onClick={() => setShowAddRuleDialog(false)}>
                    Cancel
                  </Button>
                  <Button
                    size="sm"
                    disabled={createRuleMutation.isPending || !newRuleData.merchant_pattern}
                    onClick={() => {
                      if (managingRulesCard) {
                        createRuleMutation.mutate({
                          ...newRuleData,
                          account_id: managingRulesCard.id,
                        })
                      }
                    }}
                  >
                    {createRuleMutation.isPending ? 'Saving...' : 'Save Rule'}
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowAddRuleDialog(true)}
                className="w-full gap-1.5"
              >
                <Plus data-icon="inline-start" />
                <span>Add Merchant Multiplier Rule</span>
              </Button>
            )}
          </div>

          <DialogFooter>
            <Button size="sm" onClick={() => setManagingRulesCard(null)}>
              Done
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
