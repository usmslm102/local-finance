import { ChartContainer } from '@/components/ui/chart'
import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import { buttonVariants } from '@/components/ui/button'
import { PageLoading, QueryError } from '@/components/ui/query-state'
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@/components/ui/table'
import React, { useState, useMemo, useEffect, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearch, useNavigate } from '@tanstack/react-router'
import { BarChart, Bar, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid } from 'recharts'
import { fetchSalaryInsights } from '@/lib/api'
import { formatINR, formatDate } from '@/lib/utils'
import { PrivacyAmount } from '@/components/ui/privacy-amount'
import { MerchantAvatar } from '@/components/ui/merchant-avatar'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SelectGroup,
} from '@/components/ui/select'
import {
  TrendingUp,
  ArrowUpRight,
  Briefcase,
  Building2,
  Calendar,
  Sparkles,
  Award,
  RefreshCw,
  Search,
  ReceiptText,
  UploadCloud,
  CheckCircle2,
  Gift,
  Zap,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Filter,
  X,
} from 'lucide-react'
import { SalarySearchParams } from '@/types'

function getPaginationItems(current: number, total: number): (number | 'ellipsis')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }
  if (current <= 4) {
    return [1, 2, 3, 4, 5, 'ellipsis', total]
  }
  if (current >= total - 3) {
    return [1, 'ellipsis', total - 4, total - 3, total - 2, total - 1, total]
  }
  return [1, 'ellipsis', current - 1, current, current + 1, 'ellipsis', total]
}

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 0]

export const SalaryView: React.FC = () => {
  const { isPrivacyMode } = usePrivacy()
  const searchParams = (useSearch({ strict: false }) as SalarySearchParams) || {}
  const navigate = useNavigate()

  const activeTab = searchParams.tab || 'yearly'
  const selectedCompany =
    searchParams.company && searchParams.company !== 'ALL' ? searchParams.company : ''
  const searchQuery = (searchParams.search || searchParams.q || '').trim()
  const page = searchParams.page && searchParams.page > 0 ? searchParams.page : 1
  const pageSizeParam =
    searchParams.pageSize !== undefined ? searchParams.pageSize : searchParams.page_size
  const pageSize =
    pageSizeParam !== undefined &&
    pageSizeParam !== null &&
    PAGE_SIZE_OPTIONS.includes(Number(pageSizeParam))
      ? Number(pageSizeParam)
      : 25

  const [searchInput, setSearchInput] = useState<string>(searchQuery)
  const [jumpPageInput, setJumpPageInput] = useState<string>('')

  useEffect(() => {
    setSearchInput(searchQuery)
  }, [searchQuery])

  const { data, isLoading, isError, refetch, isFetching } = useQuery({
    queryKey: ['salary-insights'],
    queryFn: fetchSalaryInsights,
  })

  const updateFilters = useCallback(
    (newParams: Partial<SalarySearchParams>, replace: boolean = true) => {
      const current: SalarySearchParams = {
        tab: activeTab !== 'yearly' ? activeTab : undefined,
        company: selectedCompany || undefined,
        search: searchQuery || undefined,
        page: page > 1 ? page : undefined,
        pageSize: pageSize !== 25 ? pageSize : undefined,
      }
      const merged = { ...current, ...newParams }
      const cleaned: Record<string, string | number | undefined> = {}
      if (merged.tab && merged.tab !== 'yearly') cleaned.tab = merged.tab
      if (merged.company && merged.company !== 'ALL') cleaned.company = merged.company
      if (merged.search && merged.search.trim()) cleaned.search = merged.search.trim()
      if (merged.page && merged.page > 1) cleaned.page = merged.page
      if (merged.pageSize !== undefined && merged.pageSize !== 25)
        cleaned.pageSize = merged.pageSize

      navigate({
        to: '/salary',
        search: cleaned as any,
        replace,
      })
    },
    [activeTab, selectedCompany, searchQuery, page, pageSize, navigate]
  )

  const handlePageSizeChange = (val: string | null) => {
    if (val === null || val === undefined || val === '') return
    const size = Number(val)
    updateFilters({ pageSize: size !== 25 ? size : undefined, page: undefined }, true)
  }

  const handleJumpToPage = (e: React.FormEvent) => {
    e.preventDefault()
    const target = Number(jumpPageInput)
    if (!isNaN(target) && target >= 1 && target <= totalPages) {
      updateFilters({ page: target > 1 ? target : undefined }, false)
      setJumpPageInput('')
    }
  }

  // Filtered paychecks by search and company
  const filteredPaychecks = useMemo(() => {
    if (!data?.recent_paychecks) return []
    let list = data.recent_paychecks

    // Filter by company
    if (selectedCompany) {
      const comp = selectedCompany.toLowerCase()
      list = list.filter(
        (tx) =>
          (tx.cleaned_payee && tx.cleaned_payee.toLowerCase().includes(comp)) ||
          (tx.raw_narration && tx.raw_narration.toLowerCase().includes(comp))
      )
    }

    // Filter by search term
    if (searchQuery) {
      const term = searchQuery.toLowerCase()
      list = list.filter(
        (tx) =>
          (tx.cleaned_payee && tx.cleaned_payee.toLowerCase().includes(term)) ||
          (tx.raw_narration && tx.raw_narration.toLowerCase().includes(term)) ||
          (tx.account_name && tx.account_name.toLowerCase().includes(term)) ||
          (tx.reference_number && tx.reference_number.toLowerCase().includes(term))
      )
    }

    return list
  }, [data?.recent_paychecks, selectedCompany, searchQuery])

  const totalCount = filteredPaychecks.length
  const totalPages = pageSize === 0 ? 1 : Math.max(1, Math.ceil(totalCount / pageSize))
  const currentPage = Math.min(page, totalPages)
  const startItem = totalCount === 0 ? 0 : pageSize === 0 ? 1 : (currentPage - 1) * pageSize + 1
  const endItem = pageSize === 0 ? totalCount : Math.min(currentPage * pageSize, totalCount)

  const paginatedPaychecks = useMemo(() => {
    if (pageSize === 0) return filteredPaychecks
    const start = (currentPage - 1) * pageSize
    return filteredPaychecks.slice(start, start + pageSize)
  }, [filteredPaychecks, currentPage, pageSize])

  const paginationPages = useMemo(
    () => getPaginationItems(currentPage, totalPages),
    [currentPage, totalPages]
  )

  // Significant hikes list (growth >= 5%)
  const significantHikes = useMemo(() => {
    if (!data?.monthly_history) return []
    return data.monthly_history.filter((m) => m.is_hike && (m.hike_pct ?? 0) >= 5)
  }, [data?.monthly_history])

  // Bonuses list
  const bonuses = useMemo(() => {
    if (!data?.monthly_history) return []
    return data.monthly_history.filter((m) => m.is_bonus)
  }, [data?.monthly_history])

  if (isLoading) return <PageLoading label="Loading salary insights" />
  if (isError) return <QueryError message="Unable to load salary insights" onRetry={refetch} />

  // Empty state if no salary credits exist
  if (!data || data.total_paychecks === 0) {
    return (
      <div className="p-4 sm:p-8 flex flex-col gap-6 max-w-6xl mx-auto">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b pb-5">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2.5">
              <Briefcase className="size-6 text-primary" />
              Salary & Career Growth Insights
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              Offline intelligence on lifetime earnings, pay progression, and employer history.
            </p>
          </div>
        </div>

        <Card className="flex flex-col items-center justify-center p-12 text-center border-dashed border-2">
          <div className="size-14 rounded-full bg-primary/10 flex items-center justify-center text-primary mb-4">
            <Briefcase className="size-7" />
          </div>
          <h2 className="text-lg font-semibold text-foreground">No Salary Credits Detected Yet</h2>
          <p className="text-sm text-muted-foreground max-w-md mt-2 mb-6">
            Transactions categorized under <strong>Salary</strong>, with payment mode{' '}
            <strong>SALARY</strong>, or matching employer payroll narrations will automatically
            populate your career timeline here.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-3">
            <Link to="/import" className={buttonVariants({})}>
              <UploadCloud className="size-4 mr-2" />
              Import Bank Statements
            </Link>
            <Link to="/transactions" className={buttonVariants({ variant: 'outline' })}>
              <ReceiptText className="size-4 mr-2" />
              Manage Categories & Rules
            </Link>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-8 flex flex-col gap-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b pb-5">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl sm:text-3xl font-bold tracking-tight text-foreground flex items-center gap-2.5">
              <Briefcase className="size-7 text-success" />
              Salary & Income Insights
            </h1>
            <Badge variant="outline" className="text-success">
              100% Offline
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Career compensation trajectory, appraisal milestones, and employer compensation
            breakdown.
          </p>
        </div>

        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="h-9 gap-1.5"
          >
            <RefreshCw
              data-icon="inline-start"
              className={cn('size-4', isFetching ? 'animate-spin text-primary' : '')}
            />
            Refresh
          </Button>
          <Link
            to="/transactions"
            search={{ category_id: 'cat_salary' }}
            className={buttonVariants({
              variant: 'secondary',
              size: 'sm',
              className: 'h-9 gap-1.5',
            })}
          >
            <ReceiptText className="size-4" />
            Salary Ledger
          </Link>
        </div>
      </div>

      {/* KPI Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        {/* Lifetime Earnings */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 size-24 bg-success/5 rounded-full blur-2xl pointer-events-none" />
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center justify-between">
              <span>Lifetime Earnings</span>
              <span className="p-1 rounded-md bg-success/10 text-success">
                <TrendingUp className="size-3.5" />
              </span>
            </CardDescription>
            <CardTitle className="text-xl sm:text-2xl font-bold text-foreground mt-1">
              <PrivacyAmount amount={data.lifetime_earned} />
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <p className="text-xs text-muted-foreground flex items-center gap-1.5">
              <CheckCircle2 className="size-3.5 text-success shrink-0" />
              <span>{data.total_paychecks} paychecks recorded</span>
            </p>
          </CardContent>
        </Card>

        {/* Latest Monthly Salary */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 size-24 bg-info/5 rounded-full blur-2xl pointer-events-none" />
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center justify-between">
              <span>Current Take-Home</span>
              <span className="p-1 rounded-md bg-info/10 text-info">
                <Building2 className="size-3.5" />
              </span>
            </CardDescription>
            <CardTitle className="text-xl sm:text-2xl font-bold text-foreground mt-1">
              <PrivacyAmount amount={data.latest_salary_amount} />
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <p className="text-xs text-muted-foreground truncate" title={data.latest_employer}>
              {data.latest_employer}
            </p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5">
              {formatDate(data.latest_salary_date)}
            </p>
          </CardContent>
        </Card>

        {/* Career Pay Growth */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 size-24 bg-info/5 rounded-full blur-2xl pointer-events-none" />
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center justify-between">
              <span>Career Pay Growth</span>
              <span className="p-1 rounded-md bg-info/10 text-info">
                <ArrowUpRight className="size-3.5" />
              </span>
            </CardDescription>
            <CardTitle className="text-xl sm:text-2xl font-bold text-success mt-1 flex items-center gap-1">
              <span>+{data.overall_growth_pct.toFixed(0)}%</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <p className="text-xs text-muted-foreground">
              From <PrivacyAmount amount={data.first_salary_amount} className="font-medium" />
            </p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5">
              Started {formatDate(data.first_salary_date)}
            </p>
          </CardContent>
        </Card>

        {/* Peak Paycheck */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 size-24 bg-warning/5 rounded-full blur-2xl pointer-events-none" />
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center justify-between">
              <span>Peak Paycheck</span>
              <span className="p-1 rounded-md bg-warning/10 text-warning">
                <Award className="size-3.5" />
              </span>
            </CardDescription>
            <CardTitle className="text-xl sm:text-2xl font-bold text-warning mt-1">
              <PrivacyAmount amount={data.peak_salary_amount} />
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <p className="text-xs text-muted-foreground truncate" title={data.peak_employer}>
              {data.peak_employer}
            </p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5">
              {formatDate(data.peak_salary_date)}
            </p>
          </CardContent>
        </Card>

        {/* Current FY Earned */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 size-24 bg-info/5 rounded-full blur-2xl pointer-events-none" />
          <CardHeader className="pb-2">
            <CardDescription className="text-xs font-medium uppercase tracking-wider text-muted-foreground flex items-center justify-between">
              <span>Current FY Take-Home</span>
              <span className="p-1 rounded-md bg-info/10 text-info">
                <Calendar className="size-3.5" />
              </span>
            </CardDescription>
            <CardTitle className="text-xl sm:text-2xl font-bold text-foreground mt-1">
              <PrivacyAmount amount={data.current_fy_earned} />
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-0">
            <p className="text-xs text-muted-foreground">
              Career Avg: <PrivacyAmount amount={data.average_monthly} className="font-medium" />
              /mo
            </p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5">
              Across active earning months
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Visualizations Section */}
      <Card>
        <CardHeader className="pb-3 border-b flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <CardTitle className="text-base font-semibold flex items-center gap-2">
              <TrendingUp className="size-4 text-success" />
              Salary Trajectory & Annual Progression
            </CardTitle>
            <CardDescription className="text-xs mt-0.5">
              Track year-over-year compensation growth and monthly take-home evolution.
            </CardDescription>
          </div>

          {/* Tab Selector */}
          <div className="flex items-center gap-1 bg-muted/60 p-1 rounded-lg border border-border/50 self-start sm:self-auto">
            <Button
              variant={activeTab === 'yearly' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => updateFilters({ tab: undefined }, true)}
              className="h-7 px-3 cursor-pointer"
            >
              Annual Take-Home
            </Button>
            <Button
              variant={activeTab === 'monthly' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => updateFilters({ tab: 'monthly' }, true)}
              className="h-7 px-3 cursor-pointer"
            >
              Monthly Trajectory
            </Button>
          </div>
        </CardHeader>

        <CardContent className="pt-6">
          {activeTab === 'yearly' ? (
            <div className="h-72 w-full">
              <ChartContainer
                className="h-full w-full aspect-auto"
                config={{ total_earned: { label: 'Income', color: 'var(--success)' } }}
              >
                <BarChart
                  data={data.yearly_progress}
                  margin={{ top: 15, right: 15, left: -10, bottom: 0 }}
                >
                  <defs>
                    <linearGradient id="salaryYearGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--success)" stopOpacity={0.9} />
                      <stop offset="100%" stopColor="var(--success)" stopOpacity={0.4} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    stroke="var(--border)"
                    opacity={0.6}
                    vertical={false}
                  />
                  <XAxis
                    dataKey="year"
                    stroke="var(--muted-foreground)"
                    fontSize={12}
                    tickLine={false}
                    axisLine={false}
                  />
                  <YAxis
                    stroke="var(--muted-foreground)"
                    fontSize={11}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) =>
                      isPrivacyMode
                        ? '••••'
                        : `₹${v >= 100000 ? `${(v / 100000).toFixed(1)}L` : `${(v / 1000).toFixed(0)}k`}`
                    }
                  />
                  <Tooltip
                    cursor={{ fill: 'var(--muted)' }}
                    content={({ active, payload }) => {
                      if (!active || !payload || !payload.length) return null
                      const item = payload[0].payload
                      return (
                        <div className="bg-popover border border-border text-popover-foreground rounded-lg p-3 shadow-lg text-xs flex flex-col gap-1.5 min-w-[190px]">
                          <div className="flex items-center justify-between border-b pb-1">
                            <span className="font-semibold text-sm">Year {item.year}</span>
                            {item.yoy_growth_pct ? (
                              <Badge
                                variant="outline"
                                className={
                                  item.yoy_growth_pct >= 0
                                    ? 'bg-success/10 text-success border-success/30'
                                    : 'bg-destructive/10 text-destructive border-destructive/30'
                                }
                              >
                                {item.yoy_growth_pct >= 0
                                  ? `+${item.yoy_growth_pct.toFixed(1)}%`
                                  : `${item.yoy_growth_pct.toFixed(1)}%`}{' '}
                                YoY
                              </Badge>
                            ) : null}
                          </div>
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Total In-Hand:</span>
                            <span className="font-bold text-foreground">
                              {formatINR(item.total_earned, isPrivacyMode)}
                            </span>
                          </div>
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Monthly Avg:</span>
                            <span className="font-medium text-foreground">
                              {formatINR(item.monthly_average, isPrivacyMode)}
                            </span>
                          </div>
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Paychecks:</span>
                            <span className="font-medium text-foreground">
                              {item.paycheck_count}
                            </span>
                          </div>
                        </div>
                      )
                    }}
                  />
                  <Bar
                    dataKey="total_earned"
                    name="Total Earned"
                    fill="url(#salaryYearGradient)"
                    radius={[6, 6, 0, 0]}
                    maxBarSize={48}
                  />
                </BarChart>
              </ChartContainer>
            </div>
          ) : (
            <div className="h-72 w-full">
              <ChartContainer
                className="h-full w-full aspect-auto"
                config={{ total_earned: { label: 'Income', color: 'var(--success)' } }}
              >
                <AreaChart
                  data={data.monthly_history}
                  margin={{ top: 15, right: 15, left: -10, bottom: 0 }}
                >
                  <defs>
                    <linearGradient id="salaryMonthlyGradient" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="var(--success)" stopOpacity={0.4} />
                      <stop offset="100%" stopColor="var(--success)" stopOpacity={0.02} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid
                    strokeDasharray="3 3"
                    stroke="var(--border)"
                    opacity={0.6}
                    vertical={false}
                  />
                  <XAxis
                    dataKey="month"
                    stroke="var(--muted-foreground)"
                    fontSize={11}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) => {
                      const parts = v.split('-')
                      return parts.length === 2 ? `${parts[1]}/${parts[0].slice(2)}` : v
                    }}
                  />
                  <YAxis
                    stroke="var(--muted-foreground)"
                    fontSize={11}
                    tickLine={false}
                    axisLine={false}
                    tickFormatter={(v) =>
                      isPrivacyMode
                        ? '••••'
                        : `₹${v >= 100000 ? `${(v / 100000).toFixed(1)}L` : `${(v / 1000).toFixed(0)}k`}`
                    }
                  />
                  <Tooltip
                    content={({ active, payload }) => {
                      if (!active || !payload || !payload.length) return null
                      const item = payload[0].payload
                      return (
                        <div className="bg-popover border border-border text-popover-foreground rounded-lg p-3 shadow-lg text-xs flex flex-col gap-1.5 min-w-[190px]">
                          <div className="flex items-center justify-between border-b pb-1">
                            <span className="font-semibold text-sm">{item.month}</span>
                            {item.is_hike && item.hike_pct ? (
                              <Badge variant="outline" className="text-success">
                                +{item.hike_pct.toFixed(1)}% Hike
                              </Badge>
                            ) : item.is_bonus ? (
                              <Badge variant="outline" className="text-warning">
                                Bonus Included
                              </Badge>
                            ) : null}
                          </div>
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Amount:</span>
                            <span className="font-bold text-foreground">
                              {formatINR(item.amount, isPrivacyMode)}
                            </span>
                          </div>
                          <div className="flex items-center justify-between text-muted-foreground">
                            <span>Employer:</span>
                            <span className="font-medium text-foreground truncate max-w-[120px]">
                              {item.employer}
                            </span>
                          </div>
                        </div>
                      )
                    }}
                  />
                  <Area
                    type="monotone"
                    dataKey="amount"
                    stroke="#10b981"
                    strokeWidth={2.5}
                    fill="url(#salaryMonthlyGradient)"
                  />
                </AreaChart>
              </ChartContainer>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Appraisal Milestones & Bonuses */}
      {(significantHikes.length > 0 || bonuses.length > 0) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Pay Hikes Detected */}
          <Card>
            <CardHeader className="pb-3 border-b">
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Zap className="size-4 text-success" />
                Detected Pay Hikes & Appraisals
              </CardTitle>
              <CardDescription className="text-xs">
                Months where take-home increased by &ge; 5% compared to preceding month.
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-4 flex flex-col gap-2.5 max-h-60 overflow-y-auto">
              {significantHikes.length === 0 ? (
                <p className="text-xs text-muted-foreground py-2">No pay spikes detected yet.</p>
              ) : (
                significantHikes.map((hike, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between p-2.5 rounded-lg border bg-muted/20 hover:bg-muted/40 transition-colors text-xs"
                  >
                    <div className="flex items-center gap-2.5">
                      <div className="p-1.5 rounded-md bg-success/10 text-success">
                        <ArrowUpRight className="size-4" />
                      </div>
                      <div>
                        <p className="font-semibold text-foreground">{hike.month}</p>
                        <p className="text-[11px] text-muted-foreground truncate max-w-[160px] sm:max-w-[200px]">
                          {hike.employer}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge className="text-success">+{hike.hike_pct?.toFixed(0)}% Hike</Badge>
                      <p className="text-[11px] text-muted-foreground mt-0.5 font-medium">
                        <PrivacyAmount amount={hike.amount} />
                      </p>
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          {/* Bonus Paychecks */}
          <Card>
            <CardHeader className="pb-3 border-b">
              <CardTitle className="text-sm font-semibold flex items-center gap-2">
                <Gift className="size-4 text-warning" />
                Bonus / Variable Pay Outliers
              </CardTitle>
              <CardDescription className="text-xs">
                Paychecks exceeding 1.8&times; your historical monthly average.
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-4 flex flex-col gap-2.5 max-h-60 overflow-y-auto">
              {bonuses.length === 0 ? (
                <p className="text-xs text-muted-foreground py-2">
                  No outlier bonus paychecks detected.
                </p>
              ) : (
                bonuses.map((bonus, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between p-2.5 rounded-lg border bg-muted/20 hover:bg-muted/40 transition-colors text-xs"
                  >
                    <div className="flex items-center gap-2.5">
                      <div className="p-1.5 rounded-md bg-warning/10 text-warning">
                        <Sparkles className="size-4" />
                      </div>
                      <div>
                        <p className="font-semibold text-foreground">{bonus.month}</p>
                        <p className="text-[11px] text-muted-foreground truncate max-w-[160px] sm:max-w-[200px]">
                          {bonus.employer}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <Badge className="text-warning">Outlier Paycheck</Badge>
                      <p className="text-[11px] text-muted-foreground mt-0.5 font-medium">
                        <PrivacyAmount amount={bonus.amount} />
                      </p>
                    </div>
                  </div>
                ))
              )}
            </CardContent>
          </Card>
        </div>
      )}

      {/* Employers Breakdown */}
      <Card>
        <CardHeader className="pb-3 border-b">
          <CardTitle className="text-base font-semibold flex items-center gap-2">
            <Building2 className="size-4 text-info" />
            Employer & Client Breakdown
          </CardTitle>
          <CardDescription className="text-xs">
            Cumulative compensation and paycheck history categorized by employer or payroll entity.
          </CardDescription>
        </CardHeader>
        <CardContent className="pt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.employers.map((emp, idx) => {
              const isSelected = selectedCompany.toLowerCase() === emp.employer_name.toLowerCase()
              return (
                <div
                  key={idx}
                  role="button"
                  tabIndex={0}
                  aria-pressed={isSelected}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      updateFilters(
                        { company: isSelected ? undefined : emp.employer_name, page: undefined },
                        false
                      )
                    }
                  }}
                  onClick={() =>
                    updateFilters(
                      { company: isSelected ? undefined : emp.employer_name, page: undefined },
                      false
                    )
                  }
                  className={cn(
                    'p-4 rounded-xl border transition-all shadow-xs flex flex-col justify-between flex flex-col gap-3 cursor-pointer',
                    isSelected
                      ? 'border-primary ring-2 ring-primary/20 bg-primary/5 shadow-sm'
                      : 'bg-card/60 hover:bg-muted/30 hover:border-border'
                  )}
                  title={
                    isSelected
                      ? 'Click to clear filter'
                      : `Click to filter paychecks by ${emp.employer_name}`
                  }
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-center gap-3 min-w-0">
                      <MerchantAvatar payee={emp.employer_name} size="md" />
                      <div className="min-w-0">
                        <p
                          className="font-semibold text-sm text-foreground truncate"
                          title={emp.employer_name}
                        >
                          {emp.employer_name}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {emp.paycheck_count} {emp.paycheck_count === 1 ? 'paycheck' : 'paychecks'}
                        </p>
                      </div>
                    </div>
                    <Badge variant={isSelected ? 'default' : 'outline'} className="shrink-0">
                      {isSelected ? 'Filtered' : `Rank #${idx + 1}`}
                    </Badge>
                  </div>

                  <div className="border-t pt-2.5 flex items-center justify-between text-xs">
                    <div>
                      <span className="text-muted-foreground text-[11px]">Total Received:</span>
                      <p className="font-bold text-foreground text-sm">
                        <PrivacyAmount amount={emp.total_earned} />
                      </p>
                    </div>
                    <div className="text-right">
                      <span className="text-muted-foreground text-[11px]">Monthly Avg:</span>
                      <p className="font-medium text-foreground">
                        <PrivacyAmount amount={emp.monthly_average} />
                      </p>
                    </div>
                  </div>

                  <div className="text-[11px] text-muted-foreground/80 bg-muted/40 px-2.5 py-1.5 rounded-md flex items-center justify-between">
                    <span>Tenure:</span>
                    <span className="font-medium">
                      {formatDate(emp.first_paycheck)} — {formatDate(emp.last_paycheck)}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        </CardContent>
      </Card>

      {/* Paycheck Ledger Table */}
      <Card>
        <CardHeader className="pb-3 border-b flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <ReceiptText className="size-4 text-primary" />
                Paycheck Transaction History
              </CardTitle>
              <Badge variant="secondary">{totalCount}</Badge>
            </div>
            <CardDescription className="text-xs mt-0.5">
              Showing {totalCount} {totalCount === 1 ? 'salary credit' : 'salary credits'}
              {selectedCompany ? ` from ${selectedCompany}` : ''}
              {searchQuery ? ` matching "${searchQuery}"` : ''} across your bank statements.
            </CardDescription>
          </div>

          <div className="flex items-center gap-2.5 flex-wrap sm:flex-nowrap">
            {/* Company / Employer Selector Filter */}
            <div className="flex items-center gap-1.5 w-full sm:w-auto">
              <Select
                value={selectedCompany || 'ALL'}
                onValueChange={(val: string | null) =>
                  updateFilters(
                    { company: val && val !== 'ALL' ? val : undefined, page: undefined },
                    false
                  )
                }
              >
                <SelectTrigger aria-label="All Companies" className="h-9 w-full sm:w-[170px]">
                  <Building2 className="size-3.5 mr-1.5 text-muted-foreground shrink-0" />
                  <SelectValue placeholder="All Companies" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="ALL" className="text-xs">
                      All Companies ({data.total_paychecks})
                    </SelectItem>
                    {data.employers.map((emp) => (
                      <SelectItem
                        key={emp.employer_name}
                        value={emp.employer_name}
                        className="text-xs"
                      >
                        {emp.employer_name} ({emp.paycheck_count})
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            {/* Search Input Form */}
            <form
              onSubmit={(e) => {
                e.preventDefault()
                updateFilters({ search: searchInput || undefined, page: undefined }, true)
              }}
              className="relative w-full sm:w-56"
            >
              <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
              <Input
                aria-label="Filter payee, narration"
                type="text"
                placeholder="Filter payee, narration..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                className="pl-8 pr-7 h-9"
              />
              {searchInput && (
                <Button
                  aria-label="Clear search"
                  variant="ghost"
                  type="button"
                  onClick={() => {
                    setSearchInput('')
                    updateFilters({ search: undefined, page: undefined }, true)
                  }}
                  className="absolute right-2.5 top-2.5 cursor-pointer"
                  title="Clear search"
                >
                  <X data-icon="inline-start" />
                </Button>
              )}
            </form>

            {/* Limit Filter / Rows Selector in Header */}
            <div className="flex items-center gap-1.5 shrink-0">
              <span className="text-muted-foreground text-[11px] hidden sm:inline">Rows:</span>
              <Select value={String(pageSize)} onValueChange={handlePageSizeChange}>
                <SelectTrigger aria-label="Salary table rows per page" className="h-9 w-[100px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent align="end">
                  <SelectGroup>
                    {PAGE_SIZE_OPTIONS.map((size) => (
                      <SelectItem key={size} value={String(size)} className="font-mono text-xs">
                        {size === 0 ? 'All' : `${size} / page`}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>

        {/* Active Filter Pills Bar */}
        {(selectedCompany || searchQuery) && (
          <div className="flex items-center gap-2 flex-wrap px-4 py-2 bg-muted/20 border-b text-xs">
            <span className="text-muted-foreground flex items-center gap-1 font-medium">
              <Filter className="size-3" /> Filters:
            </span>
            {selectedCompany && (
              <Badge variant="secondary" className="gap-1 py-0.5">
                Company: <strong className="font-semibold">{selectedCompany}</strong>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Remove filter"
                  onClick={() => updateFilters({ company: undefined, page: undefined }, false)}
                >
                  <X data-icon="inline-start" />
                </Button>
              </Badge>
            )}
            {searchQuery && (
              <Badge variant="secondary" className="gap-1 py-0.5">
                Search: <strong className="font-semibold">"{searchQuery}"</strong>
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Remove filter"
                  onClick={() => {
                    setSearchInput('')
                    updateFilters({ search: undefined, page: undefined }, true)
                  }}
                >
                  <X data-icon="inline-start" />
                </Button>
              </Badge>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchInput('')
                updateFilters({ company: undefined, search: undefined, page: undefined }, true)
              }}
              className="h-6 px-2 cursor-pointer"
            >
              Reset Filters
            </Button>
          </div>
        )}

        <CardContent className="pt-0 p-0">
          <div className="overflow-x-auto">
            <Table className="w-full text-xs text-left">
              <TableHeader className="bg-muted/40 text-muted-foreground font-medium border-b">
                <TableRow>
                  <TableHead className="py-2.5 px-4">Date</TableHead>
                  <TableHead className="py-2.5 px-4">Employer / Narration</TableHead>
                  <TableHead className="py-2.5 px-4">Account</TableHead>
                  <TableHead className="py-2.5 px-4">Mode / Ref</TableHead>
                  <TableHead className="py-2.5 px-4 text-right">Credit Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody className="divide-y divide-border/60">
                {paginatedPaychecks.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="py-12 text-center text-muted-foreground">
                      No paycheck transactions match your filter criteria.
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedPaychecks.map((tx) => (
                    <TableRow key={tx.id} className="hover:bg-muted/30 transition-colors">
                      <TableCell className="py-3 px-4 font-medium whitespace-nowrap text-foreground">
                        {formatDate(tx.tx_date)}
                      </TableCell>
                      <TableCell className="py-3 px-4 max-w-xs sm:max-w-md">
                        <div className="flex items-center gap-2.5">
                          <MerchantAvatar payee={tx.cleaned_payee || tx.raw_narration} size="xs" />
                          <div className="min-w-0">
                            <p className="font-semibold text-foreground truncate">
                              {tx.cleaned_payee || 'Salary Credit'}
                            </p>
                            <p className="text-[11px] text-muted-foreground truncate font-mono">
                              {tx.raw_narration}
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="py-3 px-4 text-muted-foreground whitespace-nowrap">
                        {tx.account_name || 'Bank Account'}
                      </TableCell>
                      <TableCell className="py-3 px-4 whitespace-nowrap">
                        <Badge variant="outline" className="uppercase">
                          {tx.payment_mode || 'SALARY'}
                        </Badge>
                      </TableCell>
                      <TableCell className="py-3 px-4 text-right font-bold text-success whitespace-nowrap font-mono">
                        + <PrivacyAmount amount={tx.amount} />
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          {/* Full Navigation Options (Matching /transactions) */}
          <div className="p-4 border-t bg-muted/20 flex flex-col md:flex-row items-center justify-between gap-4 text-xs">
            {/* Item Counter */}
            <div className="text-muted-foreground font-mono flex items-center gap-1.5">
              <span>Showing</span>
              <span className="font-semibold text-foreground">
                {startItem}–{endItem}
              </span>
              <span>of</span>
              <span className="font-semibold text-foreground">{totalCount}</span>
              <span>paychecks</span>
              {isFetching && (
                <span
                  className="inline-block size-2 rounded-full bg-primary animate-ping ml-1"
                  title="Updating..."
                />
              )}
            </div>

            {/* Full Navigation Buttons: First, Prev, Dynamic Page Pills, Next, Last */}
            <div className="flex items-center gap-1.5 flex-wrap justify-center">
              {/* First Page Button */}
              <Button
                aria-label="First Page"
                variant="outline"
                size="sm"
                onClick={() => updateFilters({ page: undefined }, false)}
                disabled={currentPage <= 1 || pageSize === 0}
                className="size-8 p-0 cursor-pointer"
                title="First Page"
              >
                <ChevronsLeft data-icon="inline-start" />
                <span className="sr-only">First Page</span>
              </Button>

              {/* Previous Page Button */}
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  updateFilters(
                    { page: Math.max(1, currentPage - 1) > 1 ? currentPage - 1 : undefined },
                    false
                  )
                }
                disabled={currentPage <= 1 || pageSize === 0}
                className="h-8 gap-1 px-2.5 cursor-pointer"
              >
                <ChevronLeft data-icon="inline-start" />
                <span className="hidden sm:inline">Prev</span>
              </Button>

              {/* Direct Page Number Pills */}
              {pageSize > 0 && (
                <div className="flex items-center gap-1">
                  {paginationPages.map((item, idx) => {
                    if (item === 'ellipsis') {
                      return (
                        <span
                          key={`ellipsis-${idx}`}
                          className="px-1.5 text-xs text-muted-foreground select-none"
                        >
                          …
                        </span>
                      )
                    }
                    const isCurrent = item === currentPage
                    return (
                      <Button
                        variant="ghost"
                        key={`page-${item}`}
                        onClick={() =>
                          updateFilters(
                            { page: (item as number) > 1 ? (item as number) : undefined },
                            false
                          )
                        }
                        className={cn(
                          'h-8 min-w-[32px] px-2 rounded-md text-xs font-mono transition-all cursor-pointer',
                          isCurrent
                            ? 'bg-primary text-primary-foreground font-bold shadow-xs'
                            : 'bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground border border-transparent hover:border-border'
                        )}
                      >
                        {item}
                      </Button>
                    )
                  })}
                </div>
              )}

              {/* Next Page Button */}
              <Button
                variant="outline"
                size="sm"
                onClick={() =>
                  updateFilters({ page: Math.min(totalPages, currentPage + 1) }, false)
                }
                disabled={currentPage >= totalPages || pageSize === 0}
                className="h-8 gap-1 px-2.5 cursor-pointer"
              >
                <span className="hidden sm:inline">Next</span>
                <ChevronRight data-icon="inline-start" />
              </Button>

              {/* Last Page Button */}
              <Button
                aria-label={`Last Page (${totalPages})`}
                variant="outline"
                size="sm"
                onClick={() => updateFilters({ page: totalPages }, false)}
                disabled={currentPage >= totalPages || pageSize === 0}
                className="size-8 p-0 cursor-pointer"
                title={`Last Page (${totalPages})`}
              >
                <ChevronsRight data-icon="inline-start" />
                <span className="sr-only">Last Page</span>
              </Button>
            </div>

            {/* Jump to page form & Rows Selector */}
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground text-[11px]">Rows:</span>
                <Select value={String(pageSize)} onValueChange={handlePageSizeChange}>
                  <SelectTrigger aria-label="Income history rows per page" className="h-8 w-[95px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent align="end">
                    <SelectGroup>
                      {PAGE_SIZE_OPTIONS.map((size) => (
                        <SelectItem key={size} value={String(size)} className="font-mono text-xs">
                          {size === 0 ? 'All' : `${size} / page`}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              {pageSize > 0 && totalPages > 1 && (
                <form onSubmit={handleJumpToPage} className="flex items-center gap-1">
                  <span className="text-muted-foreground text-[11px] hidden md:inline">Go to:</span>
                  <Input
                    aria-label="Jump to income history page"
                    type="number"
                    min={1}
                    max={totalPages}
                    placeholder={String(currentPage)}
                    value={jumpPageInput}
                    onChange={(e) => setJumpPageInput(e.target.value)}
                    className="h-8 w-16 text-center"
                  />
                  <Button
                    type="submit"
                    variant="secondary"
                    size="sm"
                    className="h-8 px-2.5 cursor-pointer"
                  >
                    Go
                  </Button>
                </form>
              )}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
