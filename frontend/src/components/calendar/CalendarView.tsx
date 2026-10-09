import React, { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { fetchAccounts, fetchAllTransactions } from '@/lib/api'
import { Transaction, Account } from '@/types'
import { formatDate, formatINR } from '@/lib/utils'
import {
  Calendar as CalendarIcon,
  ChevronLeft,
  ChevronRight,
  TrendingDown,
  TrendingUp,
  CreditCard,
  Landmark,
  ArrowUpRight,
  Sparkles,
  Search,
} from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

const MONTH_NAMES = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
]

const DAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

function normalizeDateStr(d?: string): string {
  if (!d) return ''
  const trimmed = d.trim()
  if (trimmed.length >= 10) {
    const candidate = trimmed.slice(0, 10)
    const parts = candidate.split('-')
    if (parts.length === 3) {
      return `${parts[0]}-${parts[1].padStart(2, '0')}-${parts[2].padStart(2, '0')}`
    }
  }
  return trimmed
}

function makeDateKey(year: number, monthIndex: number, day: number): string {
  const yyyy = String(year)
  const mm = String(monthIndex + 1).padStart(2, '0')
  const dd = String(day).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function isTodayDate(year: number, monthIndex: number, day: number): boolean {
  const today = new Date()
  return (
    today.getFullYear() === year &&
    today.getMonth() === monthIndex &&
    today.getDate() === day
  )
}

export const CalendarView: React.FC = () => {
  // Default to August 2026 (or current active dataset date)
  const [currentDate, setCurrentDate] = useState(() => {
    return new Date(2026, 7, 1) // August 2026
  })
  const [selectedDayKey, setSelectedDayKey] = useState<string | null>(null)
  const [daySearch, setDaySearch] = useState('')

  const year = currentDate.getFullYear()
  const month = currentDate.getMonth()

  // Calculate start and end date for current month query
  const startOfMonthStr = makeDateKey(year, month, 1)
  const lastDayOfMonth = new Date(year, month + 1, 0).getDate()
  const endOfMonthStr = makeDateKey(year, month, lastDayOfMonth)

  const { data: rawTransactions } = useQuery({
    queryKey: ['transactions', 'calendar', startOfMonthStr, endOfMonthStr],
    queryFn: () =>
      fetchAllTransactions({
        start_date: startOfMonthStr,
        end_date: endOfMonthStr,
      }),
  })

  const { data: accounts } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const accountsMap = useMemo(() => {
    const map = new Map<string, Account>()
    accounts?.forEach((a) => map.set(a.id, a))
    return map
  }, [accounts])

  const transactions = useMemo(() => rawTransactions || [], [rawTransactions])

  // Group transactions by normalized date (YYYY-MM-DD)
  const txByDay = useMemo(() => {
    const map = new Map<string, Transaction[]>()
    transactions.forEach((tx) => {
      const key = normalizeDateStr(tx.tx_date)
      if (!key) return
      const existing = map.get(key) || []
      existing.push(tx)
      map.set(key, existing)
    })
    return map
  }, [transactions])

  // Calculate Month Summary Statistics
  const monthStats = useMemo(() => {
    let totalOutflow = 0
    let totalInflow = 0
    let peakDayKey = ''
    let peakDayAmount = 0

    // Source breakdown map: accountId -> { account, totalSpent, count }
    const sourceMap = new Map<
      string,
      { accountId: string; bankName: string; accountType: string; mask: string; variant?: string; totalSpent: number; count: number }
    >()

    transactions.forEach((tx) => {
      if (tx.is_transfer || tx.is_excluded) return

      if (tx.tx_type === 'DEBIT') {
        totalOutflow += tx.amount

        // Track per-source spend
        const acc = accountsMap.get(tx.account_id)
        const bankName = acc?.bank_name || tx.account_name || 'Bank Account'
        const accountType = acc?.account_type || 'ACCOUNT'
        const mask = acc?.account_number_mask || ''
        const variant = acc?.card_variant

        const existing = sourceMap.get(tx.account_id) || {
          accountId: tx.account_id,
          bankName,
          accountType,
          mask,
          variant,
          totalSpent: 0,
          count: 0,
        }
        existing.totalSpent += tx.amount
        existing.count += 1
        sourceMap.set(tx.account_id, existing)
      } else if (tx.tx_type === 'CREDIT') {
        totalInflow += tx.amount
      }
    })

    // Find peak spend day
    txByDay.forEach((dayTxs, dateKey) => {
      const dayDebit = dayTxs
        .filter((t) => t.tx_type === 'DEBIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)

      if (dayDebit > peakDayAmount) {
        peakDayAmount = dayDebit
        peakDayKey = dateKey
      }
    })

    const daysInMonth = new Date(year, month + 1, 0).getDate()
    const dailyAverage = totalOutflow / daysInMonth
    const netCashFlow = totalInflow - totalOutflow

    // Sort sources by total spend descending
    const sourcesList = Array.from(sourceMap.values()).sort(
      (a, b) => b.totalSpent - a.totalSpent
    )

    return {
      totalOutflow,
      totalInflow,
      netCashFlow,
      dailyAverage,
      peakDayKey,
      peakDayAmount,
      sourcesList,
    }
  }, [transactions, txByDay, accountsMap, year, month])

  // Calendar Grid Days Builder
  const calendarDays = useMemo(() => {
    const firstDay = new Date(year, month, 1)
    const daysInCurrentMonth = new Date(year, month + 1, 0).getDate()
    const startDayOfWeek = firstDay.getDay()
    const prevMonthLastDay = new Date(year, month, 0).getDate()

    const days: {
      date: Date
      dateKey: string
      dayNumber: number
      isCurrentMonth: boolean
      isToday: boolean
      debitTotal: number
      creditTotal: number
      txCount: number
      sources: string[]
    }[] = []

    // Previous month padding
    const prevMonthYear = month === 0 ? year - 1 : year
    const prevMonthIndex = month === 0 ? 11 : month - 1
    for (let i = startDayOfWeek - 1; i >= 0; i--) {
      const dayNum = prevMonthLastDay - i
      const d = new Date(prevMonthYear, prevMonthIndex, dayNum)
      const dateKey = makeDateKey(prevMonthYear, prevMonthIndex, dayNum)
      const dayTxs = txByDay.get(dateKey) || []
      const debitTotal = dayTxs
        .filter((t) => t.tx_type === 'DEBIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)
      const creditTotal = dayTxs
        .filter((t) => t.tx_type === 'CREDIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)

      days.push({
        date: d,
        dateKey,
        dayNumber: dayNum,
        isCurrentMonth: false,
        isToday: isTodayDate(prevMonthYear, prevMonthIndex, dayNum),
        debitTotal,
        creditTotal,
        txCount: dayTxs.length,
        sources: Array.from(new Set(dayTxs.map((t) => t.account_name || 'Bank'))),
      })
    }

    // Current month days
    for (let dayNum = 1; dayNum <= daysInCurrentMonth; dayNum++) {
      const d = new Date(year, month, dayNum)
      const dateKey = makeDateKey(year, month, dayNum)
      const dayTxs = txByDay.get(dateKey) || []
      const debitTotal = dayTxs
        .filter((t) => t.tx_type === 'DEBIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)
      const creditTotal = dayTxs
        .filter((t) => t.tx_type === 'CREDIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)

      days.push({
        date: d,
        dateKey,
        dayNumber: dayNum,
        isCurrentMonth: true,
        isToday: isTodayDate(year, month, dayNum),
        debitTotal,
        creditTotal,
        txCount: dayTxs.length,
        sources: Array.from(new Set(dayTxs.map((t) => t.account_name || 'Bank'))),
      })
    }

    // Next month padding to fill row
    const nextMonthYear = month === 11 ? year + 1 : year
    const nextMonthIndex = month === 11 ? 0 : month + 1
    const remaining = (7 - (days.length % 7)) % 7
    for (let dayNum = 1; dayNum <= remaining; dayNum++) {
      const d = new Date(nextMonthYear, nextMonthIndex, dayNum)
      const dateKey = makeDateKey(nextMonthYear, nextMonthIndex, dayNum)
      const dayTxs = txByDay.get(dateKey) || []
      const debitTotal = dayTxs
        .filter((t) => t.tx_type === 'DEBIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)
      const creditTotal = dayTxs
        .filter((t) => t.tx_type === 'CREDIT' && !t.is_transfer && !t.is_excluded)
        .reduce((sum, t) => sum + t.amount, 0)

      days.push({
        date: d,
        dateKey,
        dayNumber: dayNum,
        isCurrentMonth: false,
        isToday: isTodayDate(nextMonthYear, nextMonthIndex, dayNum),
        debitTotal,
        creditTotal,
        txCount: dayTxs.length,
        sources: Array.from(new Set(dayTxs.map((t) => t.account_name || 'Bank'))),
      })
    }

    return days
  }, [year, month, txByDay])

  // Navigation handlers
  const handlePrevMonth = () => {
    setCurrentDate((prev) => new Date(prev.getFullYear(), prev.getMonth() - 1, 1))
  }

  const handleNextMonth = () => {
    setCurrentDate((prev) => new Date(prev.getFullYear(), prev.getMonth() + 1, 1))
  }

  const handleJumpToLatestData = () => {
    setCurrentDate(new Date(2026, 7, 1)) // August 2026
  }

  // Selected Day Transactions & Calculations
  const selectedDayTransactions = useMemo(() => {
    if (!selectedDayKey) return []
    return txByDay.get(selectedDayKey) || []
  }, [selectedDayKey, txByDay])

  const filteredSelectedDayTxs = useMemo(() => {
    if (!daySearch) return selectedDayTransactions
    const q = daySearch.toLowerCase()
    return selectedDayTransactions.filter(
      (tx) =>
        tx.cleaned_payee.toLowerCase().includes(q) ||
        tx.raw_narration.toLowerCase().includes(q) ||
        tx.reference_number.toLowerCase().includes(q) ||
        (tx.category_name && tx.category_name.toLowerCase().includes(q)) ||
        (tx.account_name && tx.account_name.toLowerCase().includes(q))
    )
  }, [selectedDayTransactions, daySearch])

  const selectedDaySummary = useMemo(() => {
    if (!selectedDayKey) return { debit: 0, credit: 0, count: 0, sources: [] }
    let debit = 0
    let credit = 0
    const sourceMap = new Map<string, number>()

    selectedDayTransactions.forEach((tx) => {
      if (tx.is_transfer || tx.is_excluded) return
      if (tx.tx_type === 'DEBIT') {
        debit += tx.amount
        const name = tx.account_name || 'Bank Account'
        sourceMap.set(name, (sourceMap.get(name) || 0) + tx.amount)
      } else if (tx.tx_type === 'CREDIT') {
        credit += tx.amount
      }
    })

    return {
      debit,
      credit,
      count: selectedDayTransactions.length,
      sources: Array.from(sourceMap.entries()).map(([name, amount]) => ({
        name,
        amount,
      })),
    }
  }, [selectedDayKey, selectedDayTransactions])

  return (
    <div className="space-y-6">
      {/* Top Header & Month Navigation */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl flex items-center gap-2.5">
            <CalendarIcon className="h-7 w-7 text-primary" /> Spending Calendar
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            Day-by-day cash flow & expense tracking across your bank accounts & credit cards
          </p>
        </div>

        {/* Month Selector Controls */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handlePrevMonth}
            className="h-8 w-8 p-0"
            title="Previous Month"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>

          <div className="min-w-[160px] text-center font-bold text-sm tracking-tight text-foreground px-2">
            {MONTH_NAMES[month]} {year}
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleNextMonth}
            className="h-8 w-8 p-0"
            title="Next Month"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>

          <Button
            variant="secondary"
            size="sm"
            onClick={handleJumpToLatestData}
            className="h-8 text-xs font-semibold ml-1"
          >
            Active Data (Aug '26)
          </Button>
        </div>
      </div>

      {/* Month Key Financial Highlights Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="border-border/80 bg-card shadow-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Month Outflow (Spend)
              </span>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg border bg-rose-500/10 text-rose-400 border-rose-500/20">
                <TrendingDown className="h-4 w-4" />
              </div>
            </div>
            <div className="mt-3">
              <div className="text-2xl font-bold font-mono tracking-tight text-foreground tabular-nums">
                {formatINR(monthStats.totalOutflow)}
              </div>
              <p className="mt-1 text-xs text-muted-foreground font-medium">
                Total debit expenses this month
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card shadow-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Month Inflow (Credits)
              </span>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg border bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
                <TrendingUp className="h-4 w-4" />
              </div>
            </div>
            <div className="mt-3">
              <div className="text-2xl font-bold font-mono tracking-tight text-emerald-400 tabular-nums">
                {formatINR(monthStats.totalInflow)}
              </div>
              <p className="mt-1 text-xs text-muted-foreground font-medium">
                Salary, refunds & income
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card shadow-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Daily Average Outflow
              </span>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg border bg-primary/10 text-primary border-primary/20">
                <Sparkles className="h-4 w-4" />
              </div>
            </div>
            <div className="mt-3">
              <div className="text-2xl font-bold font-mono tracking-tight text-foreground tabular-nums">
                {formatINR(monthStats.dailyAverage)}
              </div>
              <p className="mt-1 text-xs text-muted-foreground font-medium">
                Average spend per day in {MONTH_NAMES[month].slice(0, 3)}
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-border/80 bg-card shadow-xs">
          <CardContent className="p-5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                Peak Spending Day
              </span>
              <div className="flex h-8 w-8 items-center justify-center rounded-lg border bg-amber-500/10 text-amber-400 border-amber-500/20">
                <ArrowUpRight className="h-4 w-4" />
              </div>
            </div>
            <div className="mt-3">
              <div className="text-2xl font-bold font-mono tracking-tight text-foreground tabular-nums">
                {monthStats.peakDayAmount > 0 ? formatINR(monthStats.peakDayAmount) : '₹0.00'}
              </div>
              <p className="mt-1 text-xs text-muted-foreground font-medium truncate">
                {monthStats.peakDayKey ? formatDate(monthStats.peakDayKey) : 'No spend recorded'}
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Monthly Spend by Account / Card Source Breakdown */}
      {monthStats.sourcesList.length > 0 && (
        <Card className="border-border/80 bg-card shadow-xs">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <CreditCard className="h-4 w-4 text-primary" /> Monthly Outflow by Account / Card Source
              </CardTitle>
              <Badge variant="secondary" className="font-mono text-xs">
                {monthStats.sourcesList.length} Active Sources
              </Badge>
            </div>
            <CardDescription className="text-xs">
              Distribution of your monthly spending across linked bank accounts and credit cards
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-0">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
              {monthStats.sourcesList.map((source) => {
                const percentage =
                  monthStats.totalOutflow > 0
                    ? (source.totalSpent / monthStats.totalOutflow) * 100
                    : 0
                const isCard = source.accountType === 'CREDIT_CARD'

                return (
                  <div
                    key={source.accountId}
                    className="rounded-lg border border-border/70 bg-muted/30 p-3.5 flex flex-col justify-between"
                  >
                    <div>
                      <div className="flex items-start justify-between gap-2">
                        <div className="flex items-center gap-1.5 min-w-0">
                          {isCard ? (
                            <CreditCard className="h-4 w-4 text-amber-400 shrink-0" />
                          ) : (
                            <Landmark className="h-4 w-4 text-sky-400 shrink-0" />
                          )}
                          <p className="text-xs font-semibold text-foreground truncate">
                            {source.bankName}
                          </p>
                        </div>
                        <Badge variant="outline" className="text-[10px] font-mono px-1 py-0 shrink-0">
                          {percentage.toFixed(0)}%
                        </Badge>
                      </div>
                      <p className="text-[11px] text-muted-foreground font-mono mt-0.5">
                        {source.variant || source.accountType} {source.mask ? `• ${source.mask}` : ''}
                      </p>
                    </div>

                    <div className="mt-3">
                      <div className="flex items-baseline justify-between">
                        <span className="text-sm font-bold font-mono text-foreground tabular-nums">
                          {formatINR(source.totalSpent)}
                        </span>
                        <span className="text-[10px] font-mono text-muted-foreground">
                          {source.count} {source.count === 1 ? 'tx' : 'txs'}
                        </span>
                      </div>
                      <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted overflow-hidden">
                        <div
                          className="h-full rounded-full bg-primary"
                          style={{ width: `${Math.min(100, Math.max(4, percentage))}%` }}
                        />
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Main Calendar Grid */}
      <Card className="border-border/80 bg-card shadow-xs overflow-hidden">
        <div className="overflow-x-auto w-full">
          <div className="min-w-[600px]">
            {/* Days of Week Header */}
            <div className="grid grid-cols-7 border-b bg-muted/50 text-center text-xs font-semibold text-muted-foreground py-2.5">
              {DAY_LABELS.map((dayLabel, idx) => (
                <div key={dayLabel} className={idx === 0 || idx === 6 ? 'text-primary/80' : ''}>
                  {dayLabel}
                </div>
              ))}
            </div>

            {/* Calendar Days Matrix */}
            <div className="grid grid-cols-7 auto-rows-fr gap-px bg-border/60">
              {calendarDays.map((dayItem) => {
                const hasSpend = dayItem.debitTotal > 0
                const hasCredit = dayItem.creditTotal > 0
                const isSelected = selectedDayKey === dayItem.dateKey

                // Calculate spend intensity color tint
                let intensityBg = 'bg-card'
                if (dayItem.isCurrentMonth && hasSpend) {
                  if (dayItem.debitTotal > 5000) {
                    intensityBg = 'bg-rose-500/15 hover:bg-rose-500/25'
                  } else if (dayItem.debitTotal > 1500) {
                    intensityBg = 'bg-amber-500/10 hover:bg-amber-500/20'
                  } else {
                    intensityBg = 'bg-primary/5 hover:bg-primary/15'
                  }
                } else if (dayItem.isCurrentMonth) {
                  intensityBg = 'bg-card hover:bg-muted/40'
                } else {
                  intensityBg = 'bg-muted/20 opacity-40 hover:opacity-75'
                }

                return (
                  <div
                    key={dayItem.dateKey}
                    onClick={() => {
                      if (dayItem.txCount > 0) {
                        setSelectedDayKey(dayItem.dateKey)
                        setDaySearch('')
                      }
                    }}
                    className={`min-h-[105px] sm:min-h-[120px] p-2 sm:p-2.5 transition-all flex flex-col justify-between relative cursor-pointer select-none ${intensityBg} ${
                      isSelected ? 'ring-2 ring-primary z-10' : ''
                    } ${dayItem.isToday ? 'border-t-2 border-t-primary' : ''}`}
                  >
                    {/* Day Number Header */}
                    <div className="flex items-center justify-between">
                      <span
                        className={`inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold font-mono ${
                          dayItem.isToday
                            ? 'bg-primary text-primary-foreground font-bold'
                            : dayItem.isCurrentMonth
                            ? 'text-foreground'
                            : 'text-muted-foreground'
                        }`}
                      >
                        {dayItem.dayNumber}
                      </span>

                      {dayItem.txCount > 0 && (
                        <Badge
                          variant="secondary"
                          className="px-1.5 py-0 text-[10px] font-mono h-4 shrink-0 font-medium"
                        >
                          {dayItem.txCount}
                        </Badge>
                      )}
                    </div>

                    {/* Day Financial Figures */}
                    <div className="mt-1.5 space-y-1">
                      {hasSpend ? (
                        <div>
                          <span className="text-[10px] uppercase font-semibold text-rose-400 block leading-tight">
                            Spent
                          </span>
                          <span className="text-xs sm:text-sm font-bold font-mono tracking-tight text-foreground tabular-nums block leading-tight">
                            {formatINR(dayItem.debitTotal)}
                          </span>
                        </div>
                      ) : hasCredit ? (
                        <div>
                          <span className="text-[10px] uppercase font-semibold text-emerald-400 block leading-tight">
                            Credit
                          </span>
                          <span className="text-xs font-bold font-mono text-emerald-400 tabular-nums block leading-tight">
                            +{formatINR(dayItem.creditTotal)}
                          </span>
                        </div>
                      ) : dayItem.isCurrentMonth ? (
                        <span className="text-[11px] text-muted-foreground/50 font-medium block">
                          -
                        </span>
                      ) : null}

                      {/* If both debit and credit on same day, show credit as subtext */}
                      {hasSpend && hasCredit && (
                        <span className="text-[10px] font-mono font-medium text-emerald-400 block">
                          +{formatINR(dayItem.creditTotal)}
                        </span>
                      )}
                    </div>

                    {/* Source indicators */}
                    {dayItem.sources.length > 0 && (
                      <div className="flex items-center gap-1 mt-1 overflow-hidden">
                        {dayItem.sources.slice(0, 2).map((src, i) => (
                          <span
                            key={i}
                            className="text-[9px] font-mono text-muted-foreground truncate max-w-[80px]"
                            title={src}
                          >
                            • {src.split(' ')[0]}
                          </span>
                        ))}
                        {dayItem.sources.length > 2 && (
                          <span className="text-[9px] font-mono text-muted-foreground">
                            +{dayItem.sources.length - 2}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      </Card>

      {/* DAY TRANSACTION INSPECTOR POPUP DIALOG */}
      <Dialog open={!!selectedDayKey} onOpenChange={(open) => !open && setSelectedDayKey(null)}>
        <DialogContent className="sm:max-w-4xl md:max-w-5xl w-[95vw] max-h-[90vh] flex flex-col p-6 overflow-hidden">
          <DialogHeader className="pb-3 border-b">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <DialogTitle className="text-lg font-bold flex items-center gap-2">
                  <CalendarIcon className="h-5 w-5 text-primary" />
                  {selectedDayKey ? formatDate(selectedDayKey) : 'Day Overview'}
                </DialogTitle>
                <DialogDescription className="text-xs mt-0.5">
                  Detailed transactions and payment sources for this specific day
                </DialogDescription>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="secondary" className="font-mono text-xs">
                  {selectedDaySummary.count} {selectedDaySummary.count === 1 ? 'Transaction' : 'Transactions'}
                </Badge>
              </div>
            </div>
          </DialogHeader>

          <div className="flex-1 overflow-y-auto space-y-4 py-2 pr-1">
            {/* Day Financial Summary Ribbon */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div className="rounded-lg bg-muted/40 border p-3">
                <span className="text-[11px] text-muted-foreground uppercase font-medium">Total Outflow</span>
                <p className="font-bold font-mono text-rose-400 text-base mt-0.5">
                  {formatINR(selectedDaySummary.debit)}
                </p>
              </div>
              <div className="rounded-lg bg-muted/40 border p-3">
                <span className="text-[11px] text-muted-foreground uppercase font-medium">Total Inflow</span>
                <p className="font-bold font-mono text-emerald-400 text-base mt-0.5">
                  {formatINR(selectedDaySummary.credit)}
                </p>
              </div>
              <div className="rounded-lg bg-muted/40 border p-3">
                <span className="text-[11px] text-muted-foreground uppercase font-medium">Net Day Flow</span>
                <p className="font-bold font-mono text-foreground text-base mt-0.5">
                  {formatINR(selectedDaySummary.credit - selectedDaySummary.debit)}
                </p>
              </div>
              <div className="rounded-lg bg-muted/40 border p-3">
                <span className="text-[11px] text-muted-foreground uppercase font-medium">Sources Active</span>
                <p className="font-bold font-mono text-primary text-base mt-0.5">
                  {selectedDaySummary.sources.length} Accounts/Cards
                </p>
              </div>
            </div>

            {/* Day Sources Distribution Chips */}
            {selectedDaySummary.sources.length > 0 && (
              <div className="flex items-center gap-2 flex-wrap text-xs bg-muted/20 border rounded-lg p-2.5">
                <span className="text-muted-foreground font-medium flex items-center gap-1">
                  <CreditCard className="h-3.5 w-3.5 text-primary" /> Sources:
                </span>
                {selectedDaySummary.sources.map((src, i) => (
                  <Badge key={i} variant="outline" className="font-mono text-[11px] py-0.5 px-2">
                    {src.name}: <strong className="text-foreground ml-1">{formatINR(src.amount)}</strong>
                  </Badge>
                ))}
              </div>
            )}

            {/* Search Filter */}
            <div className="flex items-center justify-between gap-3">
              <div className="relative flex-1 max-w-sm">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  placeholder="Search payee, narration, ref, mode..."
                  value={daySearch}
                  onChange={(e) => setDaySearch(e.target.value)}
                  className="pl-8 text-xs h-8"
                />
              </div>
              <span className="text-xs text-muted-foreground font-mono">
                Showing {filteredSelectedDayTxs.length} of {selectedDayTransactions.length} items
              </span>
            </div>

            {/* Day Transactions Table */}
            <div className="rounded-lg border overflow-hidden">
              <div className="max-h-[42vh] overflow-y-auto">
                <Table className="min-w-[700px]">
                  <TableHeader className="sticky top-0 bg-muted/80 backdrop-blur-xs z-10">
                    <TableRow className="hover:bg-transparent">
                      <TableHead className="text-[11px] font-semibold uppercase w-20">Type</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase text-right w-28">Amount</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase">Payee / Narration</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase w-32">Source Account</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase w-24">Mode</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase w-28">Category</TableHead>
                      <TableHead className="text-[11px] font-semibold uppercase w-28">Ref No</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredSelectedDayTxs.map((tx) => {
                      const isCredit = tx.tx_type === 'CREDIT'
                      return (
                        <TableRow key={tx.id} className="hover:bg-muted/40 text-xs">
                          <TableCell>
                            <Badge
                              variant={isCredit ? 'secondary' : 'outline'}
                              className={`text-[10px] font-semibold uppercase px-1.5 py-0 ${
                                isCredit
                                  ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20'
                                  : 'text-muted-foreground'
                              }`}
                            >
                              {tx.tx_type}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-right font-mono font-bold tabular-nums whitespace-nowrap">
                            <span className={isCredit ? 'text-emerald-400' : 'text-foreground'}>
                              {isCredit ? '+' : ''} {formatINR(tx.amount)}
                            </span>
                          </TableCell>
                          <TableCell className="max-w-xs">
                            <div className="font-medium text-foreground truncate">{tx.cleaned_payee}</div>
                            <div className="text-[10px] text-muted-foreground truncate font-mono mt-0.5">
                              {tx.raw_narration}
                            </div>
                          </TableCell>
                          <TableCell className="font-mono text-muted-foreground text-[11px]">
                            {tx.account_name || 'Bank Account'}
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline" className="text-[10px] font-mono">
                              {tx.payment_mode}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            {tx.category_name ? (
                              <Badge variant="secondary" className="text-[10px]">
                                {tx.category_name}
                              </Badge>
                            ) : (
                              <span className="text-muted-foreground text-[11px]">-</span>
                            )}
                          </TableCell>
                          <TableCell className="font-mono text-muted-foreground text-[11px] truncate max-w-[110px]">
                            {tx.reference_number || '-'}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>

          <DialogFooter className="border-t pt-3">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setSelectedDayKey(null)}
              className="text-xs"
            >
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
