import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { cn } from '@/lib/utils'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert'
import { usePrivacy } from '@/components/privacy-provider'
import { Field, FieldLabel } from '@/components/ui/field'
import React, { useState, useMemo, useCallback, useEffect, useId } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { fetchBudgetSummary, upsertCategoryBudget, deleteCategoryBudget } from '../../lib/api'
import { formatINR } from '../../lib/utils'
import type { UpsertCategoryBudgetRequest, BudgetSearchParams } from '../../types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Badge } from '../ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../ui/dialog'
import {
  Target,
  ChevronLeft,
  ChevronRight,
  Calendar,
  AlertTriangle,
  CheckCircle2,
  TrendingUp,
  Sliders,
  Plus,
  Trash2,
  PieChart,
  Search,
  Sparkles,
  DollarSign,
} from 'lucide-react'

export const BudgetView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const fieldId = useId()
  const queryClient = useQueryClient()
  const searchParams = (useSearch({ strict: false }) as BudgetSearchParams) || {}
  const navigate = useNavigate()

  // Current selected month: format "YYYY-MM"
  const today = useMemo(() => new Date(), [])
  const currentMonthStr = useMemo(
    () => `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}`,
    [today]
  )

  const selectedMonth = searchParams.month || currentMonthStr
  const filterTab = (searchParams.tab as 'ALL' | 'BUDGETED' | 'WARNING' | 'UNBUDGETED') || 'ALL'
  const searchQuery = (searchParams.search || searchParams.q || '').trim()

  const [searchInput, setSearchInput] = useState<string>(searchQuery)

  useEffect(() => {
    setSearchInput(searchQuery)
  }, [searchQuery])

  const updateFilters = useCallback(
    (newParams: Partial<BudgetSearchParams>, replace: boolean = true) => {
      const current: BudgetSearchParams = {
        month: selectedMonth !== currentMonthStr ? selectedMonth : undefined,
        tab: filterTab !== 'ALL' ? filterTab : undefined,
        search: searchQuery || undefined,
      }
      const merged = { ...current, ...newParams }
      const cleaned: Record<string, string | undefined> = {}
      if (merged.month && merged.month !== currentMonthStr) cleaned.month = merged.month
      if (merged.tab && merged.tab !== 'ALL') cleaned.tab = merged.tab
      if (merged.search && merged.search.trim()) cleaned.search = merged.search.trim()

      navigate({
        to: '/budget',
        search: cleaned,
        replace,
      })
    },
    [selectedMonth, currentMonthStr, filterTab, searchQuery, navigate]
  )

  // Debounced search input sync
  useEffect(() => {
    if (searchInput.trim() === searchQuery) return
    const timer = setTimeout(() => {
      updateFilters({ search: searchInput.trim() || undefined }, true)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput, searchQuery, updateFilters])

  // Edit / Set Budget Modal state
  const [editingCategory, setEditingCategory] = useState<{
    id: string
    name: string
    color: string
    currentLimit: number
  } | null>(null)
  const [budgetLimitInput, setBudgetLimitInput] = useState<number>(10000)

  // Fetch Budget Summary for selected month
  const {
    data: summary,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['budgets', selectedMonth],
    queryFn: () => fetchBudgetSummary(selectedMonth),
  })

  // Mutations
  const upsertMutation = useMutation({
    mutationFn: (data: UpsertCategoryBudgetRequest) => upsertCategoryBudget(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      setEditingCategory(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: ({ categoryId, month }: { categoryId: string; month?: string }) =>
      deleteCategoryBudget(categoryId, month),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budgets'] })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
    },
  })

  // Navigation handlers
  const handlePrevMonth = () => {
    const [y, m] = selectedMonth.split('-').map(Number)
    const d = new Date(y, m - 2, 1)
    const nextMonth = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    updateFilters({ month: nextMonth }, false)
  }

  const handleNextMonth = () => {
    const [y, m] = selectedMonth.split('-').map(Number)
    const d = new Date(y, m, 1)
    const nextMonth = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
    updateFilters({ month: nextMonth }, false)
  }

  const handleCurrentMonth = () => {
    updateFilters({ month: undefined }, false)
  }

  // Format month for display (e.g., "August 2026")
  const displayMonthName = () => {
    const [y, m] = selectedMonth.split('-').map(Number)
    const d = new Date(y, m - 1, 1)
    return d.toLocaleString('en-US', { month: 'long', year: 'numeric' })
  }

  const openBudgetModal = (cat: {
    id: string
    name: string
    color: string
    currentLimit: number
  }) => {
    setEditingCategory(cat)
    setBudgetLimitInput(cat.currentLimit > 0 ? cat.currentLimit : 10000)
  }

  const handleSaveBudget = () => {
    if (!editingCategory) return
    upsertMutation.mutate({
      category_id: editingCategory.id,
      monthly_limit: budgetLimitInput,
      month: selectedMonth,
    })
  }

  // Filter budgets
  const filteredBudgets = (summary?.budgets || []).filter((b) => {
    const matchesSearch = b.category_name.toLowerCase().includes(searchQuery.toLowerCase())
    if (!matchesSearch) return false

    if (filterTab === 'BUDGETED') return b.monthly_limit > 0
    if (filterTab === 'WARNING') return b.status === 'WARNING' || b.status === 'EXCEEDED'
    if (filterTab === 'UNBUDGETED') return b.monthly_limit <= 0
    return true
  })

  const monthProgressPct = summary
    ? Math.min(
        (summary.days_elapsed_in_month / Math.max(1, summary.total_days_in_month)) * 100,
        100
      )
    : 0

  return (
    <div className="flex flex-col gap-6">
      {/* HEADER & MONTH SELECTOR */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2.5">
            <Target className="size-6 text-primary" /> Category Budgets
          </h1>
          <p className="text-sm text-muted-foreground">
            Set monthly limits and see how your spending compares.
          </p>
        </div>

        {/* Month Navigation Control */}
        <div className="flex items-center gap-2 bg-card border rounded-lg p-1 shadow-xs">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Previous budget month"
            onClick={handlePrevMonth}
            className="size-8"
          >
            <ChevronLeft data-icon="inline-start" />
          </Button>
          <div className="flex items-center gap-1.5 px-2.5 text-xs font-semibold">
            <Calendar className="size-3.5 text-primary" />
            <span>{displayMonthName()}</span>
            {selectedMonth === currentMonthStr && (
              <Badge variant="secondary" className="px-1.5 py-0 h-4">
                Current
              </Badge>
            )}
          </div>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Next budget month"
            onClick={handleNextMonth}
            className="size-8"
          >
            <ChevronRight data-icon="inline-start" />
          </Button>
          {selectedMonth !== currentMonthStr && (
            <Button variant="outline" size="sm" onClick={handleCurrentMonth} className="h-7 ml-1">
              Today
            </Button>
          )}
        </div>
      </div>

      {(upsertMutation.isError || deleteMutation.isError) && (
        <Alert variant="destructive">
          <AlertTitle>Budget changes could not be saved</AlertTitle>
          <AlertDescription>Please try the action again.</AlertDescription>
        </Alert>
      )}
      {isLoading ? (
        <PageLoading label="Loading budgets" />
      ) : isError || !summary ? (
        <QueryError message="Unable to load budgets" onRetry={refetch} />
      ) : (
        <>
          {/* OVERSPENT / WARNING ALERT BANNER */}
          {(summary.overspent_categories_count > 0 || summary.warning_categories_count > 0) && (
            <div className="rounded-xl border border-warning/30 bg-warning/10 p-4 flex flex-col gap-2">
              <div className="flex items-center gap-2 text-warning font-semibold text-sm">
                <AlertTriangle className="size-4.5" />
                <span>Overspend & Pacing Alerts for {displayMonthName()}</span>
              </div>
              <div className="flex flex-wrap gap-2 pt-1">
                {summary.budgets
                  .filter((b) => b.status === 'EXCEEDED' || b.status === 'WARNING')
                  .map((b) => (
                    <Badge
                      key={b.id}
                      variant="outline"
                      className={cn(
                        'text-xs gap-1.5 py-1 px-2.5',
                        b.status === 'EXCEEDED'
                          ? 'border-destructive/40 bg-destructive/10 text-destructive font-semibold'
                          : 'border-warning/40 bg-warning/10 text-warning font-semibold'
                      )}
                    >
                      <span
                        className="size-2 rounded-full"
                        style={{ backgroundColor: b.category_color }}
                      />
                      <strong>{b.category_name}:</strong>{' '}
                      {b.status === 'EXCEEDED'
                        ? `Exceeded by ${formatINR(Math.abs(b.remaining_amount), isPrivacyMode)} (${b.spent_percentage.toFixed(0)}%)`
                        : `${b.spent_percentage.toFixed(0)}% Spent (${formatINR(b.remaining_amount, isPrivacyMode)} remaining)`}
                    </Badge>
                  ))}
              </div>
            </div>
          )}

          {/* KPI RIBBON CARDS */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {/* 1. Total Monthly Budget */}
            <Card>
              <CardHeader className="pb-2">
                <CardDescription className="text-xs font-semibold text-muted-foreground flex items-center justify-between">
                  <span>TOTAL MONTHLY BUDGET</span>
                  <Target className="size-4 text-primary" />
                </CardDescription>
                <CardTitle className="text-2xl font-bold text-foreground">
                  {formatINR(summary.total_budget, isPrivacyMode)}
                </CardTitle>
              </CardHeader>
              <CardContent className="pt-0">
                <p className="text-xs text-muted-foreground">
                  {summary.categories_with_budgets} active category limit
                  {summary.categories_with_budgets === 1 ? '' : 's'} configured
                </p>
              </CardContent>
            </Card>

            {/* 2. Total Spend & Progress */}
            <Card>
              <CardHeader className="pb-2">
                <CardDescription className="text-xs font-semibold text-muted-foreground flex items-center justify-between">
                  <span>TOTAL MONTHLY SPENT</span>
                  <TrendingUp className="size-4 text-success" />
                </CardDescription>
                <CardTitle className="text-2xl font-bold text-foreground">
                  {formatINR(summary.total_spent, isPrivacyMode)}
                </CardTitle>
              </CardHeader>
              <CardContent className="pt-0 flex flex-col gap-1.5">
                <div className="flex justify-between text-xs font-mono">
                  <span className="text-muted-foreground">Budget Utilized:</span>
                  <span
                    className={cn(
                      'font-bold',
                      summary.overall_spent_percentage > 100
                        ? 'text-destructive'
                        : summary.overall_spent_percentage > 80
                          ? 'text-warning'
                          : 'text-success'
                    )}
                  >
                    {summary.overall_spent_percentage.toFixed(1)}%
                  </span>
                </div>
                <div className="h-1.5 w-full rounded-full bg-muted overflow-hidden">
                  <div
                    className={cn(
                      'h-full rounded-full transition-all',
                      summary.overall_spent_percentage > 100
                        ? 'bg-destructive'
                        : summary.overall_spent_percentage > 80
                          ? 'bg-warning'
                          : 'bg-success'
                    )}
                    style={{ width: `${Math.min(summary.overall_spent_percentage, 100)}%` }}
                  />
                </div>
              </CardContent>
            </Card>

            {/* 3. Total Remaining Budget */}
            <Card>
              <CardHeader className="pb-2">
                <CardDescription className="text-xs font-semibold text-muted-foreground flex items-center justify-between">
                  <span>REMAINING BUDGET POOL</span>
                  <PieChart className="size-4 text-info" />
                </CardDescription>
                <CardTitle
                  className={cn(
                    'text-2xl font-bold',
                    summary.total_remaining > 0 ? 'text-foreground' : 'text-destructive'
                  )}
                >
                  {formatINR(summary.total_remaining, isPrivacyMode)}
                </CardTitle>
              </CardHeader>
              <CardContent className="pt-0">
                <p className="text-xs text-muted-foreground font-mono">
                  {summary.days_remaining_in_month} day
                  {summary.days_remaining_in_month === 1 ? '' : 's'} remaining in month
                </p>
              </CardContent>
            </Card>

            {/* 4. Daily Recommended Safe Allowance */}
            <Card>
              <CardHeader className="pb-2">
                <CardDescription className="text-xs font-semibold text-primary flex items-center justify-between">
                  <span>DAILY SAFE ALLOWANCE</span>
                  <Sparkles className="size-4 text-primary" />
                </CardDescription>
                <CardTitle className="text-2xl font-bold text-primary font-mono">
                  {summary.overall_daily_recommended_allowance > 0
                    ? `${formatINR(summary.overall_daily_recommended_allowance, isPrivacyMode)} / day`
                    : `${formatINR(0, isPrivacyMode)} / day`}
                </CardTitle>
              </CardHeader>
              <CardContent className="pt-0">
                <p className="text-xs text-muted-foreground">
                  Safe daily spend runway across budgeted categories
                </p>
              </CardContent>
            </Card>
          </div>

          {/* MONTH PACING PROGRESS COMPARISON */}
          <Card>
            <CardContent className="py-4">
              <div className="flex flex-col gap-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between text-xs gap-1">
                  <span className="font-semibold text-foreground flex items-center gap-1.5">
                    <Calendar className="size-3.5 text-primary" /> Month Progress vs Spend Pacing
                  </span>
                  <div className="flex items-center gap-3 text-muted-foreground font-mono">
                    <span>
                      Day <strong>{summary.days_elapsed_in_month}</strong> of{' '}
                      {summary.total_days_in_month} ({monthProgressPct.toFixed(0)}% elapsed)
                    </span>
                    <span>•</span>
                    <span>
                      Spend: <strong>{summary.overall_spent_percentage.toFixed(0)}%</strong> of
                      budget
                    </span>
                  </div>
                </div>

                <div className="relative h-3 w-full rounded-full bg-muted overflow-hidden">
                  {/* Spend Bar */}
                  <div
                    className={cn(
                      'h-full rounded-full transition-all',
                      summary.overall_spent_percentage > 100
                        ? 'bg-destructive'
                        : summary.overall_spent_percentage > monthProgressPct + 10
                          ? 'bg-warning'
                          : 'bg-success'
                    )}
                    style={{ width: `${Math.min(summary.overall_spent_percentage, 100)}%` }}
                  />
                  {/* Month Timeline Marker Pin */}
                  <div
                    className="absolute top-0 bottom-0 w-0.5 bg-foreground/80 z-10"
                    style={{ left: `${monthProgressPct}%` }}
                    title={`Day ${summary.days_elapsed_in_month} Marker`}
                  />
                </div>

                <div className="flex justify-between text-[10px] text-muted-foreground">
                  <span>Day 1</span>
                  <span className="font-semibold text-foreground">
                    {summary.overall_spent_percentage <= monthProgressPct
                      ? '✨ Great job! You are spending under the expected calendar pace.'
                      : '⚡ Warning: Spending pace is slightly faster than calendar days elapsed.'}
                  </span>
                  <span>Day {summary.total_days_in_month}</span>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* CATEGORY BUDGETS LEDGER */}
          <Card>
            <CardHeader className="pb-3 border-b">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                <div>
                  <CardTitle className="text-base font-semibold flex items-center gap-2">
                    <Sliders className="size-4.5 text-primary" /> Category Budgets & Pacing
                    Breakdown
                  </CardTitle>
                  <CardDescription className="text-xs">
                    Inspect category limits, actual expenditures, and daily allowance
                    recommendations
                  </CardDescription>
                </div>

                {/* Filter Tabs & Search */}
                <div className="flex flex-wrap items-center gap-2">
                  <div className="relative w-44">
                    <Search className="size-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                    <Input
                      aria-label="Search category"
                      placeholder="Search category..."
                      value={searchInput}
                      onChange={(e) => setSearchInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          updateFilters({ search: searchInput.trim() || undefined }, true)
                        }
                      }}
                      className="pl-8 h-8"
                    />
                  </div>

                  <ToggleGroup
                    value={[filterTab]}
                    variant="outline"
                    size="sm"
                    aria-label="Budget filters"
                    className="flex-wrap"
                  >
                    <ToggleGroupItem
                      value="ALL"
                      onClick={() => updateFilters({ tab: undefined }, true)}
                      className="h-7 px-2.5"
                    >
                      All ({summary.budgets.length})
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="BUDGETED"
                      onClick={() => updateFilters({ tab: 'BUDGETED' }, true)}
                      className="h-7 px-2.5"
                    >
                      Active ({summary.categories_with_budgets})
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="WARNING"
                      onClick={() => updateFilters({ tab: 'WARNING' }, true)}
                      className="h-7 px-2.5"
                    >
                      Overspent (
                      {summary.overspent_categories_count + summary.warning_categories_count})
                    </ToggleGroupItem>
                    <ToggleGroupItem
                      value="UNBUDGETED"
                      onClick={() => updateFilters({ tab: 'UNBUDGETED' }, true)}
                      className="h-7 px-2.5"
                    >
                      Unset ({summary.budgets.length - summary.categories_with_budgets})
                    </ToggleGroupItem>
                  </ToggleGroup>
                </div>
              </div>
            </CardHeader>

            <CardContent className="pt-4">
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
                {filteredBudgets.map((b) => {
                  const hasBudget = b.monthly_limit > 0
                  const isExceeded = b.status === 'EXCEEDED'
                  const isWarning = b.status === 'WARNING'

                  return (
                    <div
                      key={b.id}
                      className={cn(
                        'rounded-xl border p-4 transition-all flex flex-col gap-3',
                        isExceeded
                          ? 'border-destructive/40 bg-destructive/5 ring-1 ring-destructive/20'
                          : isWarning
                            ? 'border-warning/40 bg-warning/5 ring-1 ring-warning/20'
                            : hasBudget
                              ? 'border-border/80 bg-card shadow-xs'
                              : 'border-dashed border-border/60 bg-muted/10 opacity-80'
                      )}
                    >
                      {/* Header: Name, Color & Badge */}
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span
                            className="size-3 rounded-full flex-shrink-0"
                            style={{ backgroundColor: b.category_color || '#64748B' }}
                          />
                          <h4 className="font-bold text-sm text-foreground">{b.category_name}</h4>
                        </div>

                        {isExceeded ? (
                          <Badge className="text-destructive gap-1">
                            <AlertTriangle className="size-3" /> Exceeded!
                          </Badge>
                        ) : isWarning ? (
                          <Badge className="text-warning gap-1">
                            <AlertTriangle className="size-3" /> {b.spent_percentage.toFixed(0)}%
                            Alert
                          </Badge>
                        ) : hasBudget ? (
                          <Badge className="text-success gap-1">
                            <CheckCircle2 className="size-3" /> On Track
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="border-dashed">
                            No Budget
                          </Badge>
                        )}
                      </div>

                      {/* Spend & Limit Numbers */}
                      <div className="flex justify-between items-end">
                        <div>
                          <p className="text-[10px] text-muted-foreground">Spent this month</p>
                          <p className="text-lg font-bold font-mono text-foreground">
                            {formatINR(b.actual_spend, isPrivacyMode)}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="text-[10px] text-muted-foreground">Monthly Target</p>
                          <p className="text-sm font-semibold font-mono text-foreground">
                            {hasBudget
                              ? formatINR(b.monthly_limit, isPrivacyMode)
                              : 'Not Configured'}
                          </p>
                        </div>
                      </div>

                      {/* Visual Pacing Bar */}
                      {hasBudget ? (
                        <div className="flex flex-col gap-1.5">
                          <div className="relative h-2 w-full rounded-full bg-muted overflow-hidden">
                            <div
                              className={cn(
                                'h-full rounded-full transition-all',
                                isExceeded
                                  ? 'bg-destructive'
                                  : isWarning
                                    ? 'bg-warning'
                                    : 'bg-primary'
                              )}
                              style={{ width: `${Math.min(b.spent_percentage, 100)}%` }}
                            />
                          </div>

                          <div className="flex justify-between items-center text-[11px] font-mono">
                            <span
                              className={
                                isExceeded ? 'text-destructive font-bold' : 'text-muted-foreground'
                              }
                            >
                              {isExceeded
                                ? `Overspent by ${formatINR(Math.abs(b.remaining_amount), isPrivacyMode)}`
                                : `${formatINR(b.remaining_amount, isPrivacyMode)} remaining`}
                            </span>
                            <span className="text-muted-foreground">
                              {b.spent_percentage.toFixed(0)}%
                            </span>
                          </div>
                        </div>
                      ) : (
                        <p className="text-xs text-muted-foreground italic">
                          Click below to set a monthly spending limit for this category.
                        </p>
                      )}

                      {/* Daily Allowance & Pacing Velocity */}
                      {hasBudget && (
                        <div className="rounded-lg border bg-muted/20 p-2 text-xs flex flex-col gap-1">
                          <div className="flex justify-between items-center">
                            <span className="text-muted-foreground text-[10px]">
                              Daily Allowance:
                            </span>
                            <span
                              className={cn(
                                'font-mono font-semibold text-[11px]',
                                b.daily_recommended_allowance > 0
                                  ? 'text-success'
                                  : 'text-destructive'
                              )}
                            >
                              {b.daily_recommended_allowance > 0
                                ? `${formatINR(b.daily_recommended_allowance, isPrivacyMode)} / day`
                                : `${formatINR(0, isPrivacyMode)} / day remaining`}
                            </span>
                          </div>
                          {b.projected_month_end_spend > 0 && (
                            <div className="flex justify-between items-center text-[10px]">
                              <span className="text-muted-foreground">Projected Spend:</span>
                              <span
                                className={cn(
                                  'font-mono',
                                  b.pacing_status === 'PACING_HIGH' ||
                                    b.pacing_status === 'PACING_EXCEEDED'
                                    ? 'text-warning font-semibold'
                                    : 'text-muted-foreground'
                                )}
                              >
                                ~{formatINR(b.projected_month_end_spend, isPrivacyMode)}
                              </span>
                            </div>
                          )}
                        </div>
                      )}

                      {/* Action Controls */}
                      <div className="flex items-center justify-end gap-2 pt-1 border-t border-border/50">
                        {hasBudget && (
                          <Button
                            aria-label="Remove budget target"
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              deleteMutation.mutate({
                                categoryId: b.category_id,
                                month: selectedMonth,
                              })
                            }
                            disabled={deleteMutation.isPending}
                            className="h-7 hover:text-destructive px-2"
                            title="Remove budget target"
                          >
                            <Trash2 data-icon="inline-start" />
                          </Button>
                        )}
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            openBudgetModal({
                              id: b.category_id,
                              name: b.category_name,
                              color: b.category_color,
                              currentLimit: b.monthly_limit,
                            })
                          }
                          className="h-7 gap-1"
                        >
                          <Plus data-icon="inline-start" />
                          {hasBudget ? 'Edit Target' : 'Set Budget'}
                        </Button>
                      </div>
                    </div>
                  )
                })}
              </div>

              {filteredBudgets.length === 0 && (
                <div className="py-12 text-center text-muted-foreground text-xs">
                  No categories match your search filter.
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}

      {/* SET / EDIT BUDGET MODAL */}
      <Dialog open={!!editingCategory} onOpenChange={(open) => !open && setEditingCategory(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <Target className="size-4.5 text-primary" />
              Set Budget for {editingCategory?.name}
            </DialogTitle>
            <DialogDescription className="text-xs">
              Configure target monthly spending limit for {displayMonthName()}
            </DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-4 py-2 text-xs">
            <Field className="flex flex-col gap-1.5">
              <FieldLabel htmlFor={`${fieldId}-694`} className="font-semibold text-foreground">
                Monthly Spending Limit (₹) *
              </FieldLabel>
              <div className="relative">
                <DollarSign className="size-4 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                <Input
                  id={`${fieldId}-694`}
                  type="number"
                  step="500"
                  placeholder="e.g. 10000"
                  value={budgetLimitInput || ''}
                  onChange={(e) => setBudgetLimitInput(parseFloat(e.target.value) || 0)}
                  className="pl-8 h-9"
                />
              </div>
            </Field>

            {/* Quick Preset Buttons */}
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] text-muted-foreground">Quick Preset Limits:</label>
              <div className="grid grid-cols-3 gap-1.5">
                {[2000, 5000, 10000, 15000, 25000, 50000].map((preset) => (
                  <Button
                    key={preset}
                    variant={budgetLimitInput === preset ? 'default' : 'outline'}
                    size="sm"
                    onClick={() => setBudgetLimitInput(preset)}
                    className="h-7"
                  >
                    {formatINR(preset, isPrivacyMode)}
                  </Button>
                ))}
              </div>
            </div>
          </div>

          <DialogFooter className="gap-2 pt-2">
            <Button variant="outline" size="sm" onClick={() => setEditingCategory(null)}>
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={handleSaveBudget}
              disabled={upsertMutation.isPending || budgetLimitInput <= 0}
            >
              {upsertMutation.isPending ? 'Saving...' : 'Save Budget Target'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
