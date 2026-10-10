import { cn } from '@/lib/utils'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { usePrivacy } from '@/components/privacy-provider'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import { Field, FieldLabel, FieldGroup } from '@/components/ui/field'
import React, { useState, useMemo, useCallback, useEffect, useId } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearch, useNavigate } from '@tanstack/react-router'
import {
  fetchAccounts,
  fetchCategories,
  fetchSubscriptions,
  scanSubscriptions,
  createSubscription,
  updateSubscription,
  deleteSubscription,
} from '@/lib/api'
import { formatDate, formatINR } from '@/lib/utils'
import {
  Subscription,
  SubscriptionFrequency,
  SubscriptionStatus,
  UpsertSubscriptionRequest,
  SubscriptionsSearchParams,
} from '@/types'
import {
  RefreshCw,
  Plus,
  Tv,
  Cloud,
  Zap,
  Landmark,
  Calendar,
  AlertCircle,
  CheckCircle2,
  Clock,
  Trash2,
  Edit2,
  TrendingUp,
  CreditCard,
  Layers,
  PauseCircle,
  PlayCircle,
  Search,
  Sparkles,
} from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

function getDaysUntilDue(
  dueDateStr?: string
): { days: number; label: string; color: string } | null {
  if (!dueDateStr) return null
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const [y, m, d] = dueDateStr.split('-').map(Number)
  const due = new Date(y, m - 1, d)

  const diffTime = due.getTime() - today.getTime()
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))

  if (diffDays < 0) {
    return {
      days: diffDays,
      label: `${Math.abs(diffDays)}d overdue`,
      color: 'text-destructive border-destructive/30 bg-destructive/10',
    }
  }
  if (diffDays === 0) {
    return {
      days: 0,
      label: 'Due today',
      color: 'text-warning border-warning/30 bg-warning/10 font-bold',
    }
  }
  if (diffDays === 1) {
    return { days: 1, label: 'Due tomorrow', color: 'text-warning border-warning/30 bg-warning/10' }
  }
  if (diffDays <= 7) {
    return {
      days: diffDays,
      label: `In ${diffDays} days`,
      color: 'text-warning border-warning/20 bg-warning/10',
    }
  }
  return {
    days: diffDays,
    label: `In ${diffDays} days`,
    color: 'text-muted-foreground border-border bg-muted/30',
  }
}

function getCategoryIcon(name: string, pattern: string) {
  const p = (name + ' ' + pattern).toUpperCase()
  if (
    p.includes('NETFLIX') ||
    p.includes('YOUTUBE') ||
    p.includes('SPOTIFY') ||
    p.includes('PRIME') ||
    p.includes('HOTSTAR') ||
    p.includes('SONYLIV')
  ) {
    return <Tv className="size-4 text-info" />
  }
  if (
    p.includes('APPLE') ||
    p.includes('GOOGLE') ||
    p.includes('OPENAI') ||
    p.includes('CHATGPT') ||
    p.includes('GITHUB')
  ) {
    return <Cloud className="size-4 text-info" />
  }
  if (
    p.includes('AIRTEL') ||
    p.includes('JIO') ||
    p.includes('FIBER') ||
    p.includes('BROADBAND') ||
    p.includes('ELECTRICITY') ||
    p.includes('GAS')
  ) {
    return <Zap className="size-4 text-warning" />
  }
  if (
    p.includes('EMI') ||
    p.includes('LOAN') ||
    p.includes('SIP') ||
    p.includes('INSURANCE') ||
    p.includes('NACH')
  ) {
    return <Landmark className="size-4 text-success" />
  }
  return <Layers className="size-4 text-primary" />
}

export const SubscriptionsView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const fieldId = useId()
  const queryClient = useQueryClient()
  const searchParams = (useSearch({ strict: false }) as SubscriptionsSearchParams) || {}
  const navigate = useNavigate()

  const activeCategoryFilter =
    searchParams.category && searchParams.category !== 'ALL'
      ? searchParams.category
      : searchParams.service && searchParams.service !== 'ALL'
        ? searchParams.service
        : 'ALL'
  const activeStatusFilter =
    searchParams.status && searchParams.status !== 'ALL' ? searchParams.status : 'ALL'
  const searchQuery = (searchParams.search || searchParams.q || '').trim()

  const [searchInput, setSearchInput] = useState<string>(searchQuery)

  useEffect(() => {
    setSearchInput(searchQuery)
  }, [searchQuery])

  const updateFilters = useCallback(
    (newParams: Partial<SubscriptionsSearchParams>, replace: boolean = true) => {
      const current: SubscriptionsSearchParams = {
        category: activeCategoryFilter !== 'ALL' ? activeCategoryFilter : undefined,
        status: activeStatusFilter !== 'ALL' ? activeStatusFilter : undefined,
        search: searchQuery || undefined,
      }
      const merged = { ...current, ...newParams }
      const cleaned: Record<string, string | undefined> = {}
      if (merged.category && merged.category !== 'ALL') cleaned.category = merged.category
      if (merged.status && merged.status !== 'ALL') cleaned.status = merged.status
      if (merged.search && merged.search.trim()) cleaned.search = merged.search.trim()

      navigate({
        to: '/subscriptions',
        search: cleaned,
        replace,
      })
    },
    [activeCategoryFilter, activeStatusFilter, searchQuery, navigate]
  )

  // Debounced search input sync
  useEffect(() => {
    if (searchInput.trim() === searchQuery) return
    const timer = setTimeout(() => {
      updateFilters({ search: searchInput.trim() || undefined }, true)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput, searchQuery, updateFilters])

  const [statusMessage, setStatusMessage] = useState<{
    type: 'success' | 'error'
    title: string
    text: string
  } | null>(null)

  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [showAddModal, setShowAddModal] = useState(false)
  const [showDeleteModal, setShowDeleteModal] = useState<Subscription | null>(null)

  // Form State for Add / Edit
  const [formData, setFormData] = useState<UpsertSubscriptionRequest>({
    name: '',
    merchant_pattern: '',
    category_id: undefined,
    account_id: undefined,
    frequency: 'MONTHLY',
    expected_amount: 0,
    billing_day: 1,
    next_due_date: '',
    status: 'ACTIVE',
    notes: '',
  })

  const {
    data: summary,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['subscriptions'],
    queryFn: fetchSubscriptions,
  })

  const { data: accounts } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })

  const scanMutation = useMutation({
    mutationFn: scanSubscriptions,
    onSuccess: (data) => {
      queryClient.setQueryData(['subscriptions'], data)
      setStatusMessage({
        type: 'success',
        title: 'Ledger Scanned',
        text: `Successfully detected ${data.total_active} active recurring subscriptions and bills!`,
      })
      setTimeout(() => setStatusMessage(null), 5000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Scan Failed',
        text: err.message || 'Failed to scan transactions for subscriptions',
      })
    },
  })

  const createMutation = useMutation({
    mutationFn: (data: UpsertSubscriptionRequest) => createSubscription(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] })
      setShowAddModal(false)
      setStatusMessage({
        type: 'success',
        title: 'Subscription Added',
        text: 'Custom subscription created successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Creation Failed',
        text: err.message || 'Failed to create subscription',
      })
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<UpsertSubscriptionRequest> }) =>
      updateSubscription(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] })
      setEditingSub(null)
      setStatusMessage({
        type: 'success',
        title: 'Subscription Updated',
        text: 'Subscription changes saved successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Update Failed',
        text: err.message || 'Failed to update subscription',
      })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteSubscription(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['subscriptions'] })
      setShowDeleteModal(null)
      setStatusMessage({
        type: 'success',
        title: 'Subscription Deleted',
        text: 'Subscription removed successfully!',
      })
      setTimeout(() => setStatusMessage(null), 4000)
    },
    onError: (err: any) => {
      setStatusMessage({
        type: 'error',
        title: 'Delete Failed',
        text: err.message || 'Failed to delete subscription',
      })
    },
  })

  const handleOpenAdd = () => {
    setFormData({
      name: '',
      merchant_pattern: '',
      category_id: categories?.[0]?.id,
      account_id: accounts?.[0]?.id,
      frequency: 'MONTHLY',
      expected_amount: 0,
      billing_day: 1,
      next_due_date: '',
      status: 'ACTIVE',
      notes: '',
    })
    setShowAddModal(true)
  }

  const handleOpenEdit = (sub: Subscription) => {
    setFormData({
      name: sub.name,
      merchant_pattern: sub.merchant_pattern,
      category_id: sub.category_id,
      account_id: sub.account_id,
      frequency: sub.frequency,
      expected_amount: sub.expected_amount,
      billing_day: sub.billing_day,
      next_due_date: sub.next_due_date || '',
      status: sub.status,
      notes: sub.notes || '',
    })
    setEditingSub(sub)
  }

  const handleToggleStatus = (sub: Subscription) => {
    const nextStatus: SubscriptionStatus = sub.status === 'ACTIVE' ? 'PAUSED' : 'ACTIVE'
    updateMutation.mutate({
      id: sub.id,
      data: { status: nextStatus },
    })
  }

  // Filter subscriptions
  const filteredSubscriptions = useMemo(() => {
    if (!summary?.subscriptions) return []
    return summary.subscriptions.filter((sub) => {
      // Category filter
      if (activeCategoryFilter !== 'ALL') {
        const cat = (sub.category_name || '').toLowerCase()
        const pat = (sub.name + ' ' + sub.merchant_pattern).toLowerCase()
        if (
          activeCategoryFilter === 'OTT' &&
          !pat.includes('netflix') &&
          !pat.includes('youtube') &&
          !pat.includes('spotify') &&
          !pat.includes('prime') &&
          !pat.includes('hotstar') &&
          !pat.includes('sonyliv') &&
          !cat.includes('entertainment')
        ) {
          return false
        }
        if (
          activeCategoryFilter === 'TECH' &&
          !pat.includes('apple') &&
          !pat.includes('google') &&
          !pat.includes('openai') &&
          !pat.includes('chatgpt') &&
          !pat.includes('github') &&
          !cat.includes('tech')
        ) {
          return false
        }
        if (
          activeCategoryFilter === 'UTILITY' &&
          !pat.includes('airtel') &&
          !pat.includes('jio') &&
          !pat.includes('fiber') &&
          !pat.includes('broadband') &&
          !cat.includes('utilities')
        ) {
          return false
        }
        if (
          activeCategoryFilter === 'EMI' &&
          !pat.includes('emi') &&
          !pat.includes('loan') &&
          !pat.includes('nach') &&
          !pat.includes('sip') &&
          !cat.includes('financial')
        ) {
          return false
        }
      }

      // Status filter
      if (activeStatusFilter !== 'ALL' && sub.status !== activeStatusFilter) {
        return false
      }

      // Search query
      if (searchQuery.trim() !== '') {
        const q = searchQuery.toLowerCase()
        return (
          sub.name.toLowerCase().includes(q) ||
          sub.merchant_pattern.toLowerCase().includes(q) ||
          (sub.category_name && sub.category_name.toLowerCase().includes(q)) ||
          (sub.account_name && sub.account_name.toLowerCase().includes(q))
        )
      }

      return true
    })
  }, [summary, activeCategoryFilter, activeStatusFilter, searchQuery])

  // Sort upcoming renewals chronologically
  const upcomingRenewals = useMemo(() => {
    if (!summary?.subscriptions) return []
    return summary.subscriptions
      .filter((s) => s.status === 'ACTIVE' && s.next_due_date)
      .sort((a, b) => (a.next_due_date || '').localeCompare(b.next_due_date || ''))
      .slice(0, 5)
  }, [summary])

  const nextUpcoming = upcomingRenewals[0]
  const nextDueInfo = nextUpcoming ? getDaysUntilDue(nextUpcoming.next_due_date) : null

  if (isLoading) return <PageLoading label="Loading subscriptions" />
  if (isError) return <QueryError message="Unable to load subscriptions" onRetry={refetch} />

  return (
    <div className="flex flex-col gap-6">
      {/* Header & Main Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl flex items-center gap-2.5">
            <Sparkles className="size-6 text-primary" /> Subscriptions & Recurring Bills
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            Detect, track, and forecast recurring digital streaming, software, broadband, insurance,
            and loan EMI auto-debits.
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <Button
            variant="outline"
            size="sm"
            onClick={() => scanMutation.mutate()}
            disabled={scanMutation.isPending}
            className="gap-1.5 h-8.5"
          >
            <RefreshCw
              data-icon="inline-start"
              className={cn('size-3.5', scanMutation.isPending ? 'animate-spin' : '')}
            />
            {scanMutation.isPending ? 'Scanning...' : 'Scan Ledger'}
          </Button>

          <Button size="sm" onClick={handleOpenAdd} className="gap-1.5 h-8.5">
            <Plus data-icon="inline-start" /> Add Subscription
          </Button>
        </div>
      </div>

      {statusMessage && (
        <Alert
          variant={statusMessage.type === 'error' ? 'destructive' : 'default'}
          className={
            statusMessage.type === 'success' ? 'border-success/30 bg-success/10 text-success' : ''
          }
        >
          {statusMessage.type === 'success' ? (
            <CheckCircle2 className="size-4 text-success" />
          ) : (
            <AlertCircle className="size-4" />
          )}
          <AlertTitle>{statusMessage.title}</AlertTitle>
          <AlertDescription className="text-xs">{statusMessage.text}</AlertDescription>
        </Alert>
      )}

      {/* KPI Summary Cards Ribbon */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Monthly Burn Rate */}
        <Card>
          <CardHeader className="pb-2 flex flex-row items-center justify-between flex-col gap-0">
            <CardTitle className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Monthly Recurring Outflow
            </CardTitle>
            <div className="rounded-md bg-primary/10 p-2 text-primary">
              <TrendingUp className="size-4" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono text-foreground">
              {formatINR(summary?.monthly_burn_rate || 0, isPrivacyMode)}
              <span className="text-xs font-normal text-muted-foreground ml-1">/ mo</span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">
              Active subscriptions & fixed monthly debits
            </p>
          </CardContent>
        </Card>

        {/* Annual Projected Spend */}
        <Card>
          <CardHeader className="pb-2 flex flex-row items-center justify-between flex-col gap-0">
            <CardTitle className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Annual Projected Cost
            </CardTitle>
            <div className="rounded-md bg-warning/10 p-2 text-warning">
              <Calendar className="size-4" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono text-foreground">
              {formatINR(summary?.annual_projected || 0, isPrivacyMode)}
              <span className="text-xs font-normal text-muted-foreground ml-1">/ yr</span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">12-month commitment forecast</p>
          </CardContent>
        </Card>

        {/* Active Subscriptions Count */}
        <Card>
          <CardHeader className="pb-2 flex flex-row items-center justify-between flex-col gap-0">
            <CardTitle className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Active Services
            </CardTitle>
            <div className="rounded-md bg-success/10 p-2 text-success">
              <Layers className="size-4" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono text-foreground">
              {summary?.total_active || 0}
              <span className="text-xs font-normal text-muted-foreground ml-1.5 font-sans">
                Tracked
              </span>
            </div>
            <p className="text-[11px] text-muted-foreground mt-1">
              Across credit cards and bank mandates
            </p>
          </CardContent>
        </Card>

        {/* Next Upcoming Bill */}
        <Card>
          <CardHeader className="pb-2 flex flex-row items-center justify-between flex-col gap-0">
            <CardTitle className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Next Expected Debit
            </CardTitle>
            <div className="rounded-md bg-destructive/10 p-2 text-destructive">
              <Clock className="size-4" />
            </div>
          </CardHeader>
          <CardContent>
            {nextUpcoming ? (
              <div>
                <div className="text-base font-bold text-foreground truncate">
                  {nextUpcoming.name}
                </div>
                <div className="flex items-center gap-2 mt-1">
                  <span className="font-mono font-bold text-sm text-foreground">
                    {formatINR(nextUpcoming.expected_amount, isPrivacyMode)}
                  </span>
                  {nextDueInfo && (
                    <Badge variant="outline" className={cn('text-[10px]', nextDueInfo.color)}>
                      {nextDueInfo.label}
                    </Badge>
                  )}
                </div>
              </div>
            ) : (
              <div>
                <div className="text-sm font-semibold text-muted-foreground">No upcoming bills</div>
                <p className="text-[11px] text-muted-foreground mt-1">All clear for this cycle</p>
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Upcoming Renewals Timeline Ribbon */}
      {upcomingRenewals.length > 0 && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <Calendar className="size-4 text-primary" /> Upcoming Renewal Timeline
            </CardTitle>
            <CardDescription className="text-xs">
              Chronological schedule of expected renewal debits
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
              {upcomingRenewals.map((sub) => {
                const dueInfo = getDaysUntilDue(sub.next_due_date)
                return (
                  <div
                    key={sub.id}
                    className="rounded-lg border border-border/80 bg-muted/20 p-3 flex flex-col justify-between gap-2 hover:border-primary/50 transition-colors"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2 min-w-0">
                        <div className="flex size-7 shrink-0 items-center justify-center rounded-md bg-background border">
                          {getCategoryIcon(sub.name, sub.merchant_pattern)}
                        </div>
                        <div className="min-w-0">
                          <p className="text-xs font-semibold text-foreground truncate">
                            {sub.name}
                          </p>
                          <p className="text-[10px] text-muted-foreground font-mono">
                            {sub.next_due_date
                              ? formatDate(sub.next_due_date)
                              : 'Day ' + sub.billing_day}
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center justify-between pt-1 border-t border-border/50">
                      <span className="font-mono font-bold text-xs text-foreground">
                        {formatINR(sub.expected_amount, isPrivacyMode)}
                      </span>
                      {dueInfo && (
                        <Badge
                          variant="outline"
                          className={cn('text-[9px] px-1.5 py-0', dueInfo.color)}
                        >
                          {dueInfo.label}
                        </Badge>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Subscriptions Table & Filters */}
      <Card>
        <CardHeader className="pb-3 border-b">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base font-semibold">
                Active Subscriptions & Recurring Mandates
              </CardTitle>
              <CardDescription className="text-xs">
                Manage frequency, amounts, linked payment sources, and renewal schedules
              </CardDescription>
            </div>

            {/* Quick Filter Buttons */}
            <ToggleGroup
              value={[activeCategoryFilter]}
              onValueChange={(values) => {
                const value = values[0]
                if (value) updateFilters({ category: value !== 'ALL' ? value : undefined }, true)
              }}
              aria-label="Subscription category"
              className="w-full flex-wrap justify-start"
              variant="outline"
              size="sm"
            >
              {[
                { id: 'ALL', label: 'All' },
                { id: 'OTT', label: 'Media' },
                { id: 'TECH', label: 'Software' },
                { id: 'UTILITY', label: 'Utilities' },
                { id: 'EMI', label: 'Loans & EMIs' },
              ].map((option) => (
                <ToggleGroupItem key={option.id} value={option.id}>
                  {option.label}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          </div>

          {/* Search and Status Bar */}
          <div className="grid grid-cols-1 sm:grid-cols-12 gap-3 pt-3">
            <div className="relative sm:col-span-8">
              <Search className="absolute left-3 top-2.5 size-4 text-muted-foreground" />
              <Input
                aria-label="Search subscription name, merchant keyword, account"
                type="text"
                placeholder="Search subscription name, merchant keyword, account..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    updateFilters({ search: searchInput.trim() || undefined }, true)
                  }
                }}
                className="pl-9 h-8.5"
              />
            </div>
            <div className="sm:col-span-4">
              <Select
                value={activeStatusFilter}
                onValueChange={(val) =>
                  updateFilters({ status: val && val !== 'ALL' ? val : undefined }, true)
                }
              >
                <SelectTrigger aria-label="All Statuses" className="h-8.5">
                  <SelectValue placeholder="All Statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="ALL">All Statuses</SelectItem>
                    <SelectItem value="ACTIVE">Active Only</SelectItem>
                    <SelectItem value="PAUSED">Paused Only</SelectItem>
                    <SelectItem value="CANCELLED">Cancelled</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>

        <CardContent className="p-0">
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="text-xs font-semibold uppercase">
                    Service & Merchant
                  </TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Category</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Payment Source</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Frequency</TableHead>
                  <TableHead className="text-xs font-semibold uppercase text-right">
                    Amount
                  </TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Next Due</TableHead>
                  <TableHead className="text-xs font-semibold uppercase">Status</TableHead>
                  <TableHead className="text-xs font-semibold uppercase text-right">
                    Actions
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className="h-32 text-center text-xs text-muted-foreground"
                    >
                      Loading subscriptions...
                    </TableCell>
                  </TableRow>
                ) : filteredSubscriptions.length > 0 ? (
                  filteredSubscriptions.map((sub) => {
                    const dueInfo = getDaysUntilDue(sub.next_due_date)
                    const isPaused = sub.status === 'PAUSED'

                    return (
                      <TableRow key={sub.id} className="hover:bg-muted/40 text-xs">
                        {/* Service & Merchant */}
                        <TableCell>
                          <div className="flex items-center gap-2.5">
                            <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-background border text-primary">
                              {getCategoryIcon(sub.name, sub.merchant_pattern)}
                            </div>
                            <div>
                              <p className="font-semibold text-foreground">{sub.name}</p>
                              <p className="text-[10px] font-mono text-muted-foreground truncate max-w-[180px]">
                                Match: {sub.merchant_pattern}
                              </p>
                            </div>
                          </div>
                        </TableCell>

                        {/* Category */}
                        <TableCell>
                          <span className="inline-flex items-center gap-1.5 rounded-full bg-muted/60 px-2 py-0.5 text-[11px] font-medium text-foreground border border-border/50">
                            <span
                              className="size-1.5 rounded-full"
                              style={{ backgroundColor: sub.category_color || '#94a3b8' }}
                            />
                            {sub.category_name || 'Bills & Utilities'}
                          </span>
                        </TableCell>

                        {/* Account */}
                        <TableCell>
                          <div className="flex items-center gap-1 text-muted-foreground font-mono text-[11px]">
                            <CreditCard className="size-3" />
                            <span className="truncate max-w-[150px]">
                              {sub.account_name || 'Primary Account'}
                            </span>
                          </div>
                        </TableCell>

                        {/* Frequency */}
                        <TableCell>
                          <Badge variant="outline" className="uppercase">
                            {sub.frequency}
                          </Badge>
                        </TableCell>

                        {/* Expected Amount */}
                        <TableCell className="text-right">
                          <div className="font-mono font-bold text-sm text-foreground">
                            {formatINR(sub.expected_amount, isPrivacyMode)}
                          </div>
                          {sub.last_paid_amount && sub.last_paid_amount !== sub.expected_amount && (
                            <span className="text-[10px] text-warning font-mono">
                              Last: {formatINR(sub.last_paid_amount, isPrivacyMode)}
                            </span>
                          )}
                        </TableCell>

                        {/* Next Due */}
                        <TableCell>
                          {sub.next_due_date ? (
                            <div>
                              <p className="font-mono text-xs text-foreground">
                                {formatDate(sub.next_due_date)}
                              </p>
                              {dueInfo && (
                                <Badge
                                  variant="outline"
                                  className={cn('text-[9px] px-1 py-0 mt-0.5', dueInfo.color)}
                                >
                                  {dueInfo.label}
                                </Badge>
                              )}
                            </div>
                          ) : (
                            <span className="text-muted-foreground font-mono">
                              Day {sub.billing_day}
                            </span>
                          )}
                        </TableCell>

                        {/* Status */}
                        <TableCell>
                          <Badge
                            className={cn(
                              'text-[10px] font-semibold',
                              sub.status === 'ACTIVE'
                                ? 'bg-success/10 text-success border-success/20'
                                : sub.status === 'PAUSED'
                                  ? 'bg-warning/10 text-warning border-warning/20'
                                  : 'bg-muted text-muted-foreground'
                            )}
                          >
                            {sub.status}
                          </Badge>
                        </TableCell>

                        {/* Actions */}
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-1">
                            <Button
                              aria-label={isPaused ? 'Resume' : 'Pause'}
                              variant="ghost"
                              size="sm"
                              onClick={() => handleToggleStatus(sub)}
                              className="size-7 p-0"
                              title={isPaused ? 'Resume' : 'Pause'}
                            >
                              {isPaused ? (
                                <PlayCircle className="size-3.5 text-success" />
                              ) : (
                                <PauseCircle className="size-3.5 text-warning" />
                              )}
                            </Button>

                            <Button
                              aria-label="Edit"
                              variant="ghost"
                              size="sm"
                              onClick={() => handleOpenEdit(sub)}
                              className="size-7 p-0"
                              title="Edit"
                            >
                              <Edit2 data-icon="inline-start" />
                            </Button>

                            <Button
                              aria-label="Delete"
                              variant="ghost"
                              size="sm"
                              onClick={() => setShowDeleteModal(sub)}
                              className="size-7 p-0 hover:text-destructive"
                              title="Delete"
                            >
                              <Trash2 data-icon="inline-start" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                ) : (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className="h-32 text-center text-xs text-muted-foreground"
                    >
                      No matching recurring subscriptions found. Tap <strong>Scan Ledger</strong> to
                      auto-detect from transactions!
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Add / Edit Subscription Dialog */}
      <Dialog
        open={showAddModal || !!editingSub}
        onOpenChange={(open) => {
          if (!open) {
            setShowAddModal(false)
            setEditingSub(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              {editingSub ? (
                <Edit2 className="size-4 text-primary" />
              ) : (
                <Plus className="size-4 text-primary" />
              )}
              {editingSub ? 'Edit Subscription' : 'Add Custom Subscription'}
            </DialogTitle>
            <DialogDescription className="text-xs">
              Configure name, merchant matcher, amount, frequency, and payment details
            </DialogDescription>
          </DialogHeader>

          <FieldGroup className="grid grid-cols-1 sm:grid-cols-2 gap-3.5 py-2 text-xs">
            {/* Name */}
            <Field className="sm:col-span-2 flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-853`} className="font-semibold text-foreground">
                Service Name *
              </FieldLabel>
              <Input
                id={`${fieldId}-853`}
                placeholder="e.g. Netflix, Cult.fit, Broadband"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="h-8.5"
              />
            </Field>

            {/* Merchant Pattern */}
            <Field className="sm:col-span-2 flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-864`} className="font-semibold text-foreground">
                Merchant Keyword / Pattern *
              </FieldLabel>
              <Input
                id={`${fieldId}-864`}
                placeholder="e.g. NETFLIX, CULT FIT, AIRTEL"
                value={formData.merchant_pattern}
                onChange={(e) => setFormData({ ...formData, merchant_pattern: e.target.value })}
                className="h-8.5"
              />
              <p className="text-[10px] text-muted-foreground">
                Matches transactions containing this keyword in payee or raw narration
              </p>
            </Field>

            {/* Expected Amount */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-878`} className="font-semibold text-foreground">
                Expected Amount (₹) *
              </FieldLabel>
              <Input
                id={`${fieldId}-878`}
                type="number"
                step="0.01"
                placeholder="199.00"
                value={formData.expected_amount || ''}
                onChange={(e) =>
                  setFormData({ ...formData, expected_amount: parseFloat(e.target.value) || 0 })
                }
                className="h-8.5"
              />
            </Field>

            {/* Frequency */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-895`} className="font-semibold text-foreground">
                Billing Frequency
              </FieldLabel>
              <Select
                value={formData.frequency}
                onValueChange={(val) =>
                  setFormData({
                    ...formData,
                    frequency: (val || 'MONTHLY') as SubscriptionFrequency,
                  })
                }
              >
                <SelectTrigger id={`${fieldId}-895`} className="h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="MONTHLY">Monthly</SelectItem>
                    <SelectItem value="QUARTERLY">Quarterly</SelectItem>
                    <SelectItem value="YEARLY">Yearly</SelectItem>
                    <SelectItem value="WEEKLY">Weekly</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {/* Category */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-914`} className="font-semibold text-foreground">
                Category
              </FieldLabel>
              <Select
                value={formData.category_id || 'none'}
                onValueChange={(val) =>
                  setFormData({
                    ...formData,
                    category_id: !val || val === 'none' ? undefined : val,
                  })
                }
              >
                <SelectTrigger id={`${fieldId}-914`} className="h-8.5">
                  <SelectValue placeholder="Select Category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="none">Default Category</SelectItem>
                    {categories?.map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {/* Payment Source */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-935`} className="font-semibold text-foreground">
                Linked Account / Card
              </FieldLabel>
              <Select
                value={formData.account_id || 'none'}
                onValueChange={(val) =>
                  setFormData({ ...formData, account_id: !val || val === 'none' ? undefined : val })
                }
              >
                <SelectTrigger id={`${fieldId}-935`} className="h-8.5">
                  <SelectValue placeholder="Select Account" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="none">Any Account</SelectItem>
                    {accounts?.map((a) => (
                      <SelectItem key={a.id} value={a.id}>
                        {a.bank_name} {a.account_number_mask ? `(${a.account_number_mask})` : ''}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {/* Billing Day */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-952`} className="font-semibold text-foreground">
                Billing Day of Month
              </FieldLabel>
              <Input
                id={`${fieldId}-952`}
                type="number"
                min="1"
                max="31"
                placeholder="1 to 31"
                value={formData.billing_day || ''}
                onChange={(e) =>
                  setFormData({ ...formData, billing_day: parseInt(e.target.value) || 1 })
                }
                className="h-8.5"
              />
            </Field>

            {/* Next Due Date */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-966`} className="font-semibold text-foreground">
                Next Due Date
              </FieldLabel>
              <Input
                id={`${fieldId}-966`}
                type="date"
                value={formData.next_due_date || ''}
                onChange={(e) => setFormData({ ...formData, next_due_date: e.target.value })}
                className="h-8.5"
              />
            </Field>

            {/* Status */}
            <Field className="flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-981`} className="font-semibold text-foreground">
                Status
              </FieldLabel>
              <Select
                value={formData.status || 'ACTIVE'}
                onValueChange={(val) =>
                  setFormData({ ...formData, status: (val || 'ACTIVE') as SubscriptionStatus })
                }
              >
                <SelectTrigger id={`${fieldId}-981`} className="h-8.5">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="ACTIVE">Active</SelectItem>
                    <SelectItem value="PAUSED">Paused</SelectItem>
                    <SelectItem value="CANCELLED">Cancelled</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>

            {/* Notes */}
            <Field className="sm:col-span-2 flex flex-col gap-1">
              <FieldLabel htmlFor={`${fieldId}-995`} className="font-semibold text-foreground">
                Notes / Purpose
              </FieldLabel>
              <Input
                id={`${fieldId}-995`}
                placeholder="Optional notes or plan description..."
                value={formData.notes || ''}
                onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                className="h-8.5"
              />
            </Field>
          </FieldGroup>

          <DialogFooter className="gap-2 pt-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setShowAddModal(false)
                setEditingSub(null)
              }}
            >
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => {
                if (editingSub) {
                  updateMutation.mutate({ id: editingSub.id, data: formData })
                } else {
                  createMutation.mutate(formData)
                }
              }}
              disabled={
                createMutation.isPending ||
                updateMutation.isPending ||
                !formData.name ||
                !formData.merchant_pattern
              }
            >
              {editingSub ? 'Save Changes' : 'Create Subscription'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!showDeleteModal} onOpenChange={(open) => !open && setShowDeleteModal(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-destructive flex items-center gap-2">
              <Trash2 className="size-5" /> Delete Subscription
            </DialogTitle>
            <DialogDescription className="text-xs">
              Are you sure you want to delete{' '}
              <strong className="text-foreground">{showDeleteModal?.name}</strong>? Past transaction
              records in your ledger will not be deleted.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setShowDeleteModal(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => {
                if (showDeleteModal) {
                  deleteMutation.mutate(showDeleteModal.id)
                }
              }}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? 'Deleting...' : 'Confirm Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
