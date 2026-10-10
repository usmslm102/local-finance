import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
  EmptyContent,
} from '@/components/ui/empty'
import { QueryError, PageLoading } from '@/components/ui/query-state'
import React, { useState, useEffect, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { fetchMerchants, fetchMerchantProfile, fetchCategories } from '../../lib/api'
import { formatINR } from '../../lib/utils'
import type { MerchantSummaryItem, MerchantProfile, MerchantsSearchParams } from '../../types'
import { Card, CardContent } from '../ui/card'
import { Input } from '../ui/input'
import { Badge } from '../ui/badge'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '../ui/dialog'
import {
  Store,
  Search,
  SlidersHorizontal,
  TrendingUp,
  CreditCard,
  Receipt,
  Calendar,
  PieChart,
  ShoppingBag,
  ChevronRight,
} from 'lucide-react'
import { MerchantAvatar } from '../ui/merchant-avatar'

export const MerchantsView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const searchParams = (useSearch({ strict: false }) as MerchantsSearchParams) || {}
  const navigate = useNavigate()

  const searchQuery = (searchParams.search || searchParams.q || '').trim()
  const selectedCategory =
    searchParams.category && searchParams.category !== 'ALL' ? searchParams.category : ''
  const sortBy = searchParams.sortBy || searchParams.sort || searchParams.sort_by || 'total_spend'
  const selectedMerchantPayee = searchParams.payee || searchParams.merchant || null

  const [searchInput, setSearchInput] = useState<string>(searchQuery)

  useEffect(() => {
    setSearchInput(searchQuery)
  }, [searchQuery])

  const updateFilters = useCallback(
    (newParams: Partial<MerchantsSearchParams>, replace: boolean = true) => {
      const current: MerchantsSearchParams = {
        search: searchQuery || undefined,
        category: selectedCategory || undefined,
        sortBy: sortBy !== 'total_spend' ? sortBy : undefined,
        payee: selectedMerchantPayee || undefined,
      }
      const merged = { ...current, ...newParams }
      const cleaned: Record<string, string | undefined> = {}
      if (merged.search && merged.search.trim()) cleaned.search = merged.search.trim()
      if (merged.category && merged.category !== 'ALL') cleaned.category = merged.category
      if (merged.sortBy && merged.sortBy !== 'total_spend') cleaned.sortBy = merged.sortBy
      if (merged.payee) cleaned.payee = merged.payee

      navigate({
        to: '/merchants',
        search: cleaned,
        replace,
      })
    },
    [searchQuery, selectedCategory, sortBy, selectedMerchantPayee, navigate]
  )

  // Debounced search input sync
  useEffect(() => {
    if (searchInput.trim() === searchQuery) return
    const timer = setTimeout(() => {
      updateFilters({ search: searchInput.trim() || undefined }, true)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput, searchQuery, updateFilters])

  // Fetch Merchants List
  const {
    data: listResponse,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['merchants', searchQuery, selectedCategory, sortBy],
    queryFn: () =>
      fetchMerchants({
        search: searchQuery,
        category: selectedCategory,
        sort_by: sortBy,
      }),
  })

  // Fetch Categories for filter dropdown
  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })

  // Fetch Selected Merchant Profile
  const {
    data: merchantProfile,
    isLoading: isProfileLoading,
    isError: isProfileError,
    refetch: refetchProfile,
  } = useQuery({
    queryKey: ['merchant-profile', selectedMerchantPayee],
    queryFn: () => (selectedMerchantPayee ? fetchMerchantProfile(selectedMerchantPayee) : null),
    enabled: !!selectedMerchantPayee,
  })

  if (isLoading) return <PageLoading label="Loading merchants" />
  if (isError || !listResponse)
    return <QueryError message="Unable to load merchants" onRetry={refetch} />

  const merchants = listResponse.merchants || []

  return (
    <div className="flex flex-col gap-8 max-w-7xl mx-auto pb-12">
      {/* Header section */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
              Merchant Intelligence
            </h1>
            <Badge variant="secondary" className="gap-1 py-0.5">
              <Store className="size-3.5 text-primary" />
              {listResponse.total_merchants} Tracked Payees
            </Badge>
          </div>
          <p className="text-xs text-muted-foreground mt-1">
            Deep dive into your lifetime relationships, average order values, and preferred payment
            methods across every merchant.
          </p>
        </div>
      </div>

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Total Merchant Spend
              </p>
              <div className="rounded-md bg-primary/10 p-2 text-primary">
                <ShoppingBag className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {formatINR(listResponse.total_spend, isPrivacyMode)}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Across {listResponse.total_merchants} distinct merchants
            </p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Average Order Value
              </p>
              <div className="rounded-md bg-info/10 p-2 text-info">
                <TrendingUp className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground">
              {formatINR(listResponse.average_order_value, isPrivacyMode)}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">Mean transaction size</p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Top Spend Category
              </p>
              <div className="rounded-md bg-info/10 p-2 text-info">
                <PieChart className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground truncate">
              {listResponse.top_category || 'General'}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">Highest merchant volume</p>
          </CardContent>
        </Card>

        <Card className="backdrop-blur-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                #1 Top Merchant
              </p>
              <div className="rounded-md bg-warning/10 p-2 text-warning">
                <Store className="size-4" />
              </div>
            </div>
            <p className="mt-2 text-2xl font-bold tracking-tight text-foreground truncate">
              {merchants.length > 0 ? merchants[0].cleaned_payee : 'None'}
            </p>
            <p className="mt-1 text-[11px] text-muted-foreground">
              {merchants.length > 0
                ? `${formatINR(merchants[0].total_spend, isPrivacyMode)} (${merchants[0].spend_share_pct.toFixed(1)}% of total)`
                : 'No transactions'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Search, Filter & Sort Bar */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex w-full min-w-0 flex-col sm:flex-row sm:flex-1 items-stretch gap-3">
          <div className="relative min-w-0 flex-1 sm:max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
            <Input
              aria-label="Search by merchant name (e.g. Swiggy, Amazon, Uber)"
              type="text"
              placeholder="Search by merchant name (e.g. Swiggy, Amazon, Uber)..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  updateFilters({ search: searchInput.trim() || undefined }, true)
                }
              }}
              className="pl-9"
            />
          </div>

          <Select
            value={selectedCategory || 'ALL'}
            onValueChange={(value) =>
              updateFilters({ category: value === 'ALL' ? undefined : value || undefined }, true)
            }
          >
            <SelectTrigger aria-label="Filter merchants by category" className="w-full sm:w-56">
              <SelectValue>{selectedCategory || 'All categories'}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="ALL">All categories</SelectItem>
                {categories?.map((category) => (
                  <SelectItem key={category.id} value={category.name}>
                    {category.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className="flex w-full sm:w-auto items-center gap-2">
          <SlidersHorizontal className="size-4 text-muted-foreground" />
          <span className="text-xs text-muted-foreground font-medium">Sort By:</span>
          <Select
            value={sortBy}
            onValueChange={(value) => value && updateFilters({ sortBy: value }, true)}
          >
            <SelectTrigger aria-label="Sort merchants" className="w-full sm:w-52">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="total_spend">Highest spend</SelectItem>
                <SelectItem value="tx_count">Most orders</SelectItem>
                <SelectItem value="aov">Highest average order</SelectItem>
                <SelectItem value="recent">Most recent</SelectItem>
                <SelectItem value="name">Alphabetical</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Merchant Cards Grid */}
      {merchants.length === 0 ? (
        <Empty className="border py-12">
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <Store />
            </EmptyMedia>
            <EmptyTitle>No merchants found</EmptyTitle>
            <EmptyDescription>
              Import a statement or clear your filters to see your merchants.
            </EmptyDescription>
          </EmptyHeader>
          {(searchQuery || selectedCategory) && (
            <EmptyContent>
              <Button
                variant="outline"
                onClick={() => updateFilters({ search: undefined, category: undefined })}
              >
                Clear filters
              </Button>
            </EmptyContent>
          )}
        </Empty>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {merchants.map((merchant) => (
            <MerchantCard
              key={merchant.cleaned_payee}
              merchant={merchant}
              onClick={() => updateFilters({ payee: merchant.cleaned_payee }, false)}
            />
          ))}
        </div>
      )}

      {/* Merchant Profile Deep-Dive Modal */}
      <Dialog
        open={!!selectedMerchantPayee}
        onOpenChange={(open) => {
          if (!open) updateFilters({ payee: undefined }, true)
        }}
      >
        <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
          {isProfileError ? (
            <>
              <DialogTitle>Merchant details</DialogTitle>
              <QueryError message="Unable to load merchant details" onRetry={refetchProfile} />
            </>
          ) : isProfileLoading || !merchantProfile ? (
            <>
              <DialogTitle>Merchant details</DialogTitle>
              <PageLoading label="Loading merchant details" />
            </>
          ) : (
            <MerchantProfileModalContent profile={merchantProfile} />
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}

// Individual Merchant Card
const MerchantCard: React.FC<{
  merchant: MerchantSummaryItem
  onClick: () => void
}> = ({ merchant, onClick }) => {
  const { isPrivacyMode } = usePrivacy()
  return (
    <Card
      onClick={onClick}
      role="button"
      tabIndex={0}
      aria-label={`View ${merchant.cleaned_payee} transactions`}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onClick()
        }
      }}
      className="group outline-none focus-visible:ring-2 focus-visible:ring-ring cursor-pointer transition-all"
    >
      <CardContent className="p-5 flex flex-col gap-4">
        {/* Top Payee Info */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3">
            <MerchantAvatar payee={merchant.cleaned_payee} size="md" />
            <div>
              <h3 className="font-semibold text-sm text-foreground group-hover:text-primary transition-colors flex items-center gap-1.5">
                {merchant.cleaned_payee}
                <ChevronRight className="size-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
              </h3>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Badge
                  variant="outline"
                  className="px-1.5 py-0"
                  style={{
                    borderColor: `${merchant.category_color}40`,
                    color: merchant.category_color || 'inherit',
                  }}
                >
                  {merchant.category_name}
                </Badge>
                <span className="text-[11px] text-muted-foreground font-mono">
                  {merchant.tx_count} {merchant.tx_count === 1 ? 'order' : 'orders'}
                </span>
              </div>
            </div>
          </div>

          <div className="text-right">
            <p className="text-base font-bold text-foreground">
              {formatINR(merchant.total_spend, isPrivacyMode)}
            </p>
            <p className="text-[10px] text-muted-foreground">
              {merchant.spend_share_pct.toFixed(1)}% of spend
            </p>
          </div>
        </div>

        {/* Spend Share Bar */}
        <div className="flex flex-col gap-1">
          <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
            <div
              className="h-full rounded-full transition-all duration-500"
              style={{
                width: `${Math.min(100, Math.max(2, merchant.spend_share_pct))}%`,
                backgroundColor: merchant.category_color || '#3B82F6',
              }}
            />
          </div>
        </div>

        {/* Bottom Key Stats */}
        <div className="flex items-center justify-between pt-2 border-t border-border/60 text-xs text-muted-foreground">
          <div>
            <span className="text-[10px] uppercase tracking-wider block">Avg Order</span>
            <span className="font-semibold text-foreground">
              {formatINR(merchant.average_order_value, isPrivacyMode)}
            </span>
          </div>

          <div className="text-right">
            <span className="text-[10px] uppercase tracking-wider block">Last Active</span>
            <span className="font-medium font-mono text-[11px]">{merchant.last_tx_date}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

// Modal Content for Deep Dive
const MerchantProfileModalContent: React.FC<{ profile: MerchantProfile }> = ({ profile }) => {
  const { isPrivacyMode } = usePrivacy()
  return (
    <div className="flex flex-col gap-6">
      <DialogHeader>
        <div className="flex items-center gap-3">
          <MerchantAvatar payee={profile.cleaned_payee} size="md" className="size-12 text-lg" />
          <div>
            <DialogTitle className="text-xl font-bold text-foreground">
              {profile.cleaned_payee}
            </DialogTitle>
            <DialogDescription className="flex items-center gap-2 mt-1">
              <Badge
                variant="outline"
                className="px-2 py-0"
                style={{
                  borderColor: `${profile.category_color}40`,
                  color: profile.category_color || 'inherit',
                }}
              >
                {profile.category_name}
              </Badge>
              <span>•</span>
              <span>
                Active since {profile.first_tx_date} ({profile.days_since_last_tx} days ago)
              </span>
            </DialogDescription>
          </div>
        </div>
      </DialogHeader>

      {/* KPI Ribbon */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="rounded-lg border border-border bg-card/60 p-3">
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground block">
            Total Spend
          </span>
          <span className="text-lg font-bold text-foreground mt-0.5 block">
            {formatINR(profile.total_spend, isPrivacyMode)}
          </span>
          <span className="text-[10px] text-muted-foreground">{profile.debit_tx_count} debits</span>
        </div>

        <div className="rounded-lg border border-border bg-card/60 p-3">
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground block">
            Avg Order Value
          </span>
          <span className="text-lg font-bold text-foreground mt-0.5 block">
            {formatINR(profile.average_order_value, isPrivacyMode)}
          </span>
          <span className="text-[10px] text-muted-foreground">per transaction</span>
        </div>

        <div className="rounded-lg border border-border bg-card/60 p-3">
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground block">
            Total Orders
          </span>
          <span className="text-lg font-bold text-foreground mt-0.5 block">
            {profile.total_tx_count}
          </span>
          <span className="text-[10px] text-muted-foreground">
            {profile.credit_tx_count > 0 ? `${profile.credit_tx_count} refunds` : '0 refunds'}
          </span>
        </div>

        <div className="rounded-lg border border-border bg-card/60 p-3">
          <span className="text-[10px] uppercase tracking-wider text-muted-foreground block">
            Net Outflow
          </span>
          <span className="text-lg font-bold text-foreground mt-0.5 block">
            {formatINR(profile.net_spend, isPrivacyMode)}
          </span>
          <span className="text-[10px] text-muted-foreground">after cashbacks/refunds</span>
        </div>
      </div>

      {/* Payment Sources & Cards Breakdown */}
      {profile.payment_sources && profile.payment_sources.length > 0 && (
        <div className="flex flex-col gap-3">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
            <CreditCard className="size-3.5 text-primary" /> Payment Sources Used
          </h4>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {profile.payment_sources.map((src) => (
              <div
                key={src.account_name}
                className="rounded-lg border border-border bg-muted/30 p-3 flex items-center justify-between"
              >
                <div className="flex flex-col gap-0.5">
                  <p className="text-xs font-semibold text-foreground">{src.account_name}</p>
                  <p className="text-[10px] text-muted-foreground">
                    {src.tx_count} transactions ({src.share_pct.toFixed(0)}% of spend)
                  </p>
                </div>
                <span className="text-xs font-bold font-mono text-foreground">
                  {formatINR(src.spend_amount, isPrivacyMode)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Monthly Spend History Bar Chart */}
      {profile.monthly_spend_history && profile.monthly_spend_history.length > 0 && (
        <div className="flex flex-col gap-3">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
            <Calendar className="size-3.5 text-primary" /> Monthly Spend Velocity
          </h4>
          <div className="rounded-xl border border-border bg-card/40 p-4 flex flex-col gap-3">
            {profile.monthly_spend_history.map((m) => {
              const maxSpend = Math.max(
                ...profile.monthly_spend_history.map((item) => item.spend_amount),
                1
              )
              const pct = (m.spend_amount / maxSpend) * 100

              return (
                <div key={m.month} className="flex flex-col gap-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-mono text-muted-foreground">{m.month}</span>
                    <div className="flex items-center gap-2">
                      <span className="text-[11px] text-muted-foreground">
                        {m.tx_count} {m.tx_count === 1 ? 'order' : 'orders'}
                      </span>
                      <span className="font-bold text-foreground">
                        {formatINR(m.spend_amount, isPrivacyMode)}
                      </span>
                    </div>
                  </div>
                  <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                    <div
                      className="h-full rounded-full transition-all duration-300"
                      style={{
                        width: `${Math.max(3, pct)}%`,
                        backgroundColor: profile.category_color || '#3B82F6',
                      }}
                    />
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Chronological Recent Transactions */}
      {profile.recent_transactions && profile.recent_transactions.length > 0 && (
        <div className="flex flex-col gap-3">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-1.5">
            <Receipt className="size-3.5 text-primary" /> Transaction History (
            {profile.recent_transactions.length})
          </h4>
          <div className="rounded-xl border border-border overflow-hidden divide-y divide-border text-xs">
            {profile.recent_transactions.map((tx) => (
              <div
                key={tx.id}
                className="flex items-center justify-between p-3 hover:bg-muted/30 transition-colors gap-3"
              >
                <div className="flex flex-col gap-0.5">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[11px] text-muted-foreground">
                      {tx.tx_date}
                    </span>
                    <Badge variant="outline" className="px-1 py-0">
                      {tx.payment_mode}
                    </Badge>
                    <span className="text-[11px] text-muted-foreground font-medium">
                      {tx.account_name}
                    </span>
                  </div>
                  <p className="text-xs text-foreground font-mono truncate max-w-md">
                    {tx.raw_narration}
                  </p>
                </div>

                <span
                  className={cn(
                    'font-bold font-mono shrink-0',
                    tx.tx_type === 'CREDIT' ? 'text-success ' : 'text-foreground'
                  )}
                >
                  {tx.tx_type === 'CREDIT' ? '+' : '-'} {formatINR(tx.amount, isPrivacyMode)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
