import React, { useState, useMemo, useEffect, useCallback } from 'react'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearch, useNavigate } from '@tanstack/react-router'
import { CATEGORY_TRANSFERS_ID, type Transaction, type TimeWindowPreset, type TransactionsSearchParams } from '@/types'
export type { TimeWindowPreset } from '@/types'
import { fetchAccounts, fetchCategories, fetchTransactions, createRule, reapplyRules, updateTransaction } from '@/lib/api'
import { formatDate } from '@/lib/utils'
import {
  ArrowDownLeft,
  ArrowUpRight,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Search,
  Zap,
  ArrowLeftRight,
  Info,
  Calendar as CalendarIcon,
  RotateCcw,
  SlidersHorizontal,
  Sparkles,
  CheckCircle2,
  Plus,
  X,
  Download,
  Building2,
  TrendingDown,
  TrendingUp,
  Wallet,
  Save,
  RefreshCw,
} from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Card } from '@/components/ui/card'
import { CopyButton } from '@/components/ui/copy-button'
import { MerchantAvatar } from '@/components/ui/merchant-avatar'
import { PrivacyAmount } from '@/components/ui/privacy-amount'


function formatISODate(d: Date): string {
  const yyyy = d.getFullYear()
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

function getTimeWindowDates(
  preset: TimeWindowPreset,
  customStart: string,
  customEnd: string
): { startDate?: string; endDate?: string } {
  const now = new Date()

  switch (preset) {
    case 'today': {
      const todayStr = formatISODate(now)
      return { startDate: todayStr, endDate: todayStr }
    }
    case 'yesterday': {
      const yesterday = new Date(now)
      yesterday.setDate(yesterday.getDate() - 1)
      const yesterdayStr = formatISODate(yesterday)
      return { startDate: yesterdayStr, endDate: yesterdayStr }
    }
    case 'this_week': {
      const day = now.getDay()
      const diff = now.getDate() - day + (day === 0 ? -6 : 1) // Monday start
      const monday = new Date(now)
      monday.setDate(diff)
      return { startDate: formatISODate(monday), endDate: formatISODate(now) }
    }
    case 'this_month': {
      const firstDay = new Date(now.getFullYear(), now.getMonth(), 1)
      const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0)
      return { startDate: formatISODate(firstDay), endDate: formatISODate(lastDay) }
    }
    case 'this_year': {
      const firstDay = new Date(now.getFullYear(), 0, 1)
      const lastDay = new Date(now.getFullYear(), 11, 31)
      return { startDate: formatISODate(firstDay), endDate: formatISODate(lastDay) }
    }
    case 'custom': {
      return {
        startDate: customStart || undefined,
        endDate: customEnd || undefined,
      }
    }
    case 'overall':
    default:
      return { startDate: undefined, endDate: undefined }
  }
}

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

const columnHelper = createColumnHelper<Transaction>()

const PAGE_SIZE_OPTIONS = [25, 50, 100, 250, 500, 1000]

export const TransactionTable: React.FC = () => {
  const searchParams = useSearch({ strict: false }) as TransactionsSearchParams
  const navigate = useNavigate()

  // 1. Primary filter & pagination values derived directly from URL search params
  const page = searchParams.page && searchParams.page > 0 ? searchParams.page : 1
  const pageSize =
    searchParams.pageSize && PAGE_SIZE_OPTIONS.includes(searchParams.pageSize)
      ? searchParams.pageSize
      : (() => {
          try {
            const saved = localStorage.getItem('lf_tx_page_size')
            if (saved) {
              const num = parseInt(saved, 10)
              if (PAGE_SIZE_OPTIONS.includes(num)) return num
            }
          } catch {}
          return 50
        })()

  const search = (searchParams.search || searchParams.q || '').trim()
  const accountId = searchParams.account || searchParams.account_id || 'ALL'
  const categoryId = searchParams.category || searchParams.category_id || 'ALL'
  const txType = (searchParams.type || searchParams.tx_type || 'ALL').toUpperCase()
  const timeWindow = (searchParams.timeWindow || searchParams.preset || 'overall') as TimeWindowPreset
  const customStartDate = searchParams.startDate || searchParams.start_date || ''
  const customEndDate = searchParams.endDate || searchParams.end_date || ''
  const minAmount = searchParams.minAmount || searchParams.min_amount || ''
  const maxAmount = searchParams.maxAmount || searchParams.max_amount || ''

  // 2. Responsive local states for text inputs
  const [searchInput, setSearchInput] = useState(search)
  const [minInput, setMinInput] = useState(minAmount)
  const [maxInput, setMaxInput] = useState(maxAmount)
  const [jumpPageInput, setJumpPageInput] = useState('')
  const [selectedTx, setSelectedTx] = useState<Transaction | null>(null)

  // Keep local inputs in sync when URL changes (e.g. browser back/forward or external navigation)
  useEffect(() => {
    setSearchInput(search)
  }, [search])

  useEffect(() => {
    setMinInput(minAmount)
  }, [minAmount])

  useEffect(() => {
    setMaxInput(maxAmount)
  }, [maxAmount])

  // Helper to update URL search parameters cleanly and bidirectionally
  const updateFilters = useCallback(
    (newParams: Partial<TransactionsSearchParams>, replace: boolean = true) => {
      const current: TransactionsSearchParams = {
        page: page > 1 ? page : undefined,
        pageSize: pageSize !== 50 ? pageSize : undefined,
        search: search || undefined,
        account: accountId !== 'ALL' ? accountId : undefined,
        category: categoryId !== 'ALL' ? categoryId : undefined,
        type: txType !== 'ALL' ? txType : undefined,
        timeWindow: timeWindow !== 'overall' ? timeWindow : undefined,
        startDate: customStartDate || undefined,
        endDate: customEndDate || undefined,
        minAmount: minAmount || undefined,
        maxAmount: maxAmount || undefined,
      }

      const merged = { ...current, ...newParams }

      const cleaned: Record<string, string | number | undefined> = {}
      if (merged.page && merged.page > 1) cleaned.page = merged.page
      if (merged.pageSize && merged.pageSize !== 50) cleaned.pageSize = merged.pageSize
      if (merged.search && merged.search.trim()) cleaned.search = merged.search.trim()
      if (merged.account && merged.account !== 'ALL') cleaned.account = merged.account
      if (merged.category && merged.category !== 'ALL') cleaned.category = merged.category
      if (merged.type && merged.type !== 'ALL') cleaned.type = merged.type
      if (merged.timeWindow && merged.timeWindow !== 'overall') cleaned.timeWindow = merged.timeWindow
      if (merged.startDate) cleaned.startDate = merged.startDate
      if (merged.endDate) cleaned.endDate = merged.endDate
      if (merged.minAmount) cleaned.minAmount = merged.minAmount
      if (merged.maxAmount) cleaned.maxAmount = merged.maxAmount

      navigate({
        to: '/transactions',
        search: cleaned,
        replace,
      })
    },
    [
      page,
      pageSize,
      search,
      accountId,
      categoryId,
      txType,
      timeWindow,
      customStartDate,
      customEndDate,
      minAmount,
      maxAmount,
      navigate,
    ]
  )

  // Debounced sync for search input
  useEffect(() => {
    if (searchInput.trim() === search) return
    const timer = setTimeout(() => {
      updateFilters({ search: searchInput.trim() || undefined, page: undefined }, true)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchInput, search, updateFilters])

  // Debounced sync for min/max amount inputs
  useEffect(() => {
    if (minInput === minAmount && maxInput === maxAmount) return
    const timer = setTimeout(() => {
      updateFilters(
        {
          minAmount: minInput || undefined,
          maxAmount: maxInput || undefined,
          page: undefined,
        },
        true
      )
    }, 400)
    return () => clearTimeout(timer)
  }, [minInput, maxInput, minAmount, maxAmount, updateFilters])

  const { startDate, endDate } = useMemo(
    () => getTimeWindowDates(timeWindow, customStartDate, customEndDate),
    [timeWindow, customStartDate, customEndDate]
  )

  const { data: accountsData } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const { data: categoriesData } = useQuery({
    queryKey: ['categories'],
    queryFn: fetchCategories,
  })

  const queryClient = useQueryClient()
  const [showQuickRuleModal, setShowQuickRuleModal] = useState(false)
  const [quickRulePattern, setQuickRulePattern] = useState('')
  const [quickRuleField, setQuickRuleField] = useState('cleaned_payee')
  const [quickRuleCategory, setQuickRuleCategory] = useState('')
  const [quickRuleSuccessMsg, setQuickRuleSuccessMsg] = useState('')

  const createRuleQuickMutation = useMutation({
    mutationFn: async () => {
      await createRule({
        match_field: quickRuleField,
        match_type: 'CONTAINS',
        match_pattern: quickRulePattern,
        target_category_id: quickRuleCategory,
        priority: 60,
        is_active: true,
      })
      const res = await reapplyRules()
      return res
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['rules'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      setQuickRuleSuccessMsg(`Rule created! Re-categorized ${data.updated_count} transactions.`)
      setTimeout(() => {
        setQuickRuleSuccessMsg('')
        setShowQuickRuleModal(false)
      }, 2000)
    },
  })

  const [updatingTxId, setUpdatingTxId] = useState<string | null>(null)
  const [rulePromptTx, setRulePromptTx] = useState<{ tx: Transaction; newCategoryName: string } | null>(null)
  const [editNotes, setEditNotes] = useState('')
  const [editTags, setEditTags] = useState('')
  const [isEditingDetails, setIsEditingDetails] = useState(false)

  useEffect(() => {
    if (selectedTx) {
      setEditNotes(selectedTx.notes || '')
      setEditTags(selectedTx.tags || '')
      setIsEditingDetails(false)
    }
  }, [selectedTx])

  const updateTxMutation = useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: import('@/lib/api').UpdateTransactionPayload }) =>
      updateTransaction(id, payload),
    onMutate: ({ id }) => {
      setUpdatingTxId(id)
    },
    onSuccess: (updatedTx, variables) => {
      setUpdatingTxId(null)
      queryClient.invalidateQueries({ queryKey: ['transactions'] })
      queryClient.invalidateQueries({ queryKey: ['analytics'] })
      queryClient.invalidateQueries({ queryKey: ['budgets'] })
      if (selectedTx && selectedTx.id === updatedTx.id) {
        setSelectedTx(updatedTx)
      }
      if (variables.payload.category_id && updatedTx.category_name && updatedTx.cleaned_payee) {
        setRulePromptTx({ tx: updatedTx, newCategoryName: updatedTx.category_name })
      }
    },
    onError: (err: any) => {
      setUpdatingTxId(null)
      console.error('Failed to update transaction:', err)
    },
  })

  const handleOpenQuickRule = (tx: Transaction, preferredCategoryId?: string) => {
    setQuickRulePattern(tx.cleaned_payee || tx.raw_narration)
    setQuickRuleField(tx.cleaned_payee ? 'cleaned_payee' : 'raw_narration')
    setQuickRuleCategory(preferredCategoryId || tx.category_id || categoriesData?.[0]?.id || '')
    setQuickRuleSuccessMsg('')
    setShowQuickRuleModal(true)
  }

  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'transactions',
      page,
      pageSize,
      search,
      accountId,
      categoryId,
      txType,
      startDate,
      endDate,
      minAmount,
      maxAmount,
    ],
    queryFn: () =>
      fetchTransactions({
        page,
        page_size: pageSize,
        search: search || undefined,
        account_id: accountId !== 'ALL' ? accountId : undefined,
        category_id: categoryId !== 'ALL' ? categoryId : undefined,
        tx_type: txType !== 'ALL' ? txType : undefined,
        start_date: startDate,
        end_date: endDate,
        min_amount: minAmount || undefined,
        max_amount: maxAmount || undefined,
      }),
  })

  const handlePageSizeChange = (val: string | null) => {
    if (!val) return
    const size = Number(val)
    try {
      localStorage.setItem('lf_tx_page_size', String(size))
    } catch {}
    updateFilters({ pageSize: size !== 50 ? size : undefined, page: undefined }, true)
  }

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.page_size)) : 1

  const handleJumpToPage = (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    const target = parseInt(jumpPageInput, 10)
    if (!isNaN(target) && target >= 1 && target <= totalPages) {
      updateFilters({ page: target > 1 ? target : undefined }, false)
      setJumpPageInput('')
    }
  }

  const isFiltered =
    search !== '' ||
    accountId !== 'ALL' ||
    categoryId !== 'ALL' ||
    txType !== 'ALL' ||
    timeWindow !== 'overall' ||
    customStartDate !== '' ||
    customEndDate !== '' ||
    minAmount !== '' ||
    maxAmount !== ''

  const handleResetFilters = () => {
    setSearchInput('')
    setMinInput('')
    setMaxInput('')
    navigate({
      to: '/transactions',
      search: pageSize !== 50 ? { pageSize } : {},
      replace: true,
    })
  }

  // The API applies every filter before counting and paginating transactions.
  const filteredItems = useMemo(() => data?.items || [], [data?.items])

  // Financial summary of visible/filtered transactions on the current page
  const pageStats = useMemo(() => {
    let debits = 0
    let credits = 0
    filteredItems.forEach((tx) => {
      if (tx.is_transfer || tx.is_excluded || tx.category_id === CATEGORY_TRANSFERS_ID) return
      if (tx.tx_type === 'DEBIT') debits += tx.amount
      else if (tx.tx_type === 'CREDIT') credits += tx.amount
    })
    return {
      debits,
      credits,
      net: credits - debits,
    }
  }, [filteredItems])

  // Client-side CSV Export of currently loaded/filtered ledger
  const handleExportCSV = () => {
    if (!filteredItems.length) return
    const headers = [
      'Date',
      'Payee',
      'Narration',
      'Type',
      'Amount',
      'Balance',
      'Payment Mode',
      'Category',
      'Account',
      'Ref / UTR',
      'UPI VPA',
      'Card Last 4',
    ]
    const rows = filteredItems.map((tx) => [
      tx.tx_date,
      `"${(tx.cleaned_payee || '').replace(/"/g, '""')}"`,
      `"${(tx.raw_narration || '').replace(/"/g, '""')}"`,
      tx.tx_type,
      tx.amount,
      tx.running_balance !== undefined && tx.running_balance !== null ? tx.running_balance : '',
      tx.payment_mode || '',
      `"${(tx.category_name || 'Uncategorized').replace(/"/g, '""')}"`,
      `"${(tx.account_name || '').replace(/"/g, '""')}"`,
      `"${(tx.reference_number || '').replace(/"/g, '""')}"`,
      `"${(tx.upi_vpa || '').replace(/"/g, '""')}"`,
      `"${(tx.card_last4 || '').replace(/"/g, '""')}"`,
    ])

    const csvContent = [headers.join(','), ...rows.map((r) => r.join(','))].join('\n')
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.setAttribute('href', url)
    link.setAttribute('download', `transactions_page${page}_${formatISODate(new Date())}.csv`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  const columns = useMemo(
    () => [
      columnHelper.accessor('tx_date', {
        header: 'Date',
        cell: (info) => (
          <span className="font-mono text-xs text-muted-foreground whitespace-nowrap tabular-nums">
            {formatDate(info.getValue())}
          </span>
        ),
      }),
      columnHelper.accessor('cleaned_payee', {
        header: 'Payee & Narration',
        cell: (info) => {
          const row = info.row.original
          const payeeText = row.cleaned_payee || row.raw_narration
          return (
            <div
              className="flex items-center gap-2.5 max-w-md cursor-pointer group"
              onClick={() => setSelectedTx(row)}
            >
              <MerchantAvatar payee={payeeText} size="sm" />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-1.5">
                  <p className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors truncate">
                    {payeeText}
                  </p>
                  {row.is_transfer && (
                    <Badge variant="outline" className="text-[9px] px-1 py-0 font-medium text-muted-foreground shrink-0">
                      <ArrowLeftRight className="h-2.5 w-2.5 mr-0.5" /> Transfer
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground truncate" title={row.raw_narration}>
                  {row.raw_narration}
                </p>
              </div>
            </div>
          )
        },
      }),
      columnHelper.accessor('payment_mode', {
        header: 'Mode',
        cell: (info) => {
          const mode = info.getValue() || 'OTHER'
          return (
            <Badge variant="secondary" className="text-[10px] font-mono font-medium px-1.5 py-0">
              {mode === 'UPI' && <Zap className="mr-1 h-2.5 w-2.5 text-primary" />}
              {mode}
            </Badge>
          )
        },
      }),
      columnHelper.accessor('category_name', {
        header: 'Category',
        cell: (info) => {
          const row = info.row.original
          const name = info.getValue() || 'Uncategorized'
          const color = row.category_color || '#94a3b8'
          const isUpdating = updatingTxId === row.id

          return (
            <div
              className="inline-block"
              onClick={(e) => e.stopPropagation()}
            >
              <Select
                value={row.category_id || 'UNSET'}
                onValueChange={(val) => {
                  const newCatId = val === 'UNSET' ? '' : val
                  updateTxMutation.mutate({
                    id: row.id,
                    payload: { category_id: newCatId },
                  })
                }}
                disabled={isUpdating}
              >
                <SelectTrigger
                  className="h-7 text-xs border border-border/50 bg-muted/40 hover:bg-muted/80 rounded-full px-2.5 py-0.5 w-auto max-w-[155px] font-medium shadow-none focus:ring-1 focus:ring-primary/40 gap-1.5 cursor-pointer"
                  title={row.is_manual_category ? 'Category manually set (Click to change)' : 'Click to change category'}
                >
                  {isUpdating ? (
                    <RefreshCw className="h-2.5 w-2.5 animate-spin text-primary shrink-0" />
                  ) : (
                    <span className="h-1.5 w-1.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
                  )}
                  <span className="truncate">{name}</span>
                  {row.is_manual_category && (
                    <span className="text-[10px] text-primary/80 font-bold shrink-0" title="Manually edited">✎</span>
                  )}
                </SelectTrigger>
                <SelectContent className="min-w-[190px]">
                  <SelectItem value="UNSET">
                    <span className="text-muted-foreground italic">Uncategorized</span>
                  </SelectItem>
                  {categoriesData?.map((cat) => (
                    <SelectItem key={cat.id} value={cat.id}>
                      <div className="flex items-center gap-2">
                        <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: cat.color_hex }} />
                        <span>{cat.name}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )
        },
      }),
      columnHelper.accessor('account_name', {
        header: 'Account',
        cell: (info) => (
          <span className="text-xs text-muted-foreground whitespace-nowrap truncate max-w-[130px] inline-block" title={info.getValue() || 'Primary Account'}>
            {info.getValue() || 'Primary Account'}
          </span>
        ),
      }),
      columnHelper.accessor('amount', {
        header: () => <div className="text-right">Amount</div>,
        cell: (info) => {
          const row = info.row.original
          const isCredit = row.tx_type === 'CREDIT'
          return (
            <div className="flex items-center justify-end space-x-1 font-mono font-bold text-sm tabular-nums">
              {isCredit ? (
                <ArrowDownLeft className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
              ) : (
                <ArrowUpRight className="h-3.5 w-3.5 text-rose-400 shrink-0" />
              )}
              <PrivacyAmount
                amount={row.amount}
                prefix={isCredit ? '+ ' : '- '}
                className={isCredit ? 'text-emerald-400' : 'text-rose-400'}
              />
            </div>
          )
        },
      }),
      columnHelper.accessor('running_balance', {
        header: () => <div className="text-right">Balance</div>,
        cell: (info) => {
          const bal = info.getValue()
          if (bal === undefined || bal === null) {
            return <span className="text-muted-foreground/40 text-right block font-mono text-xs">–</span>
          }
          return (
            <div className="text-right font-mono text-xs text-muted-foreground tabular-nums">
              <PrivacyAmount amount={bal} />
            </div>
          )
        },
      }),
      columnHelper.display({
        id: 'actions',
        header: () => <span className="sr-only">Details</span>,
        cell: (info) => (
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation()
              setSelectedTx(info.row.original)
            }}
            className="h-7 w-7 p-0 text-muted-foreground hover:text-foreground opacity-60 group-hover:opacity-100 transition-opacity"
            title="View transaction details"
          >
            <Info className="h-3.5 w-3.5" />
          </Button>
        ),
      }),
    ],
    [categoriesData, updatingTxId, updateTxMutation]
  )

  const table = useReactTable({
    data: filteredItems,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  const timeWindowPills: { id: TimeWindowPreset; label: string }[] = [
    { id: 'overall', label: 'All Time' },
    { id: 'today', label: 'Today' },
    { id: 'yesterday', label: 'Yesterday' },
    { id: 'this_week', label: 'This Week' },
    { id: 'this_month', label: 'This Month' },
    { id: 'this_year', label: 'This Year' },
    { id: 'custom', label: 'Custom Range' },
  ]

  const totalCount = data?.total || 0
  const startItem = totalCount === 0 ? 0 : (page - 1) * pageSize + 1
  const endItem = Math.min(page * pageSize, totalCount)
  const paginationPages = useMemo(() => getPaginationItems(page, totalPages), [page, totalPages])

  // Get active account and category objects for filter badges
  const activeAccount = useMemo(() => {
    if (accountId === 'ALL') return null
    return accountsData?.find((a) => a.id === accountId)
  }, [accountId, accountsData])

  const activeCategory = useMemo(() => {
    if (categoryId === 'ALL') return null
    return categoriesData?.find((c) => c.id === categoryId)
  }, [categoryId, categoriesData])

  return (
    <div className="space-y-4">
      {/* Search & Filter Controls Card */}
      <Card className="p-4 border-border/80 bg-card shadow-xs space-y-3.5">
        {/* Time Window Preset Filter Bar */}
        <div className="flex items-center justify-between gap-2 flex-wrap border-b pb-3">
          <div className="flex items-center gap-1.5 overflow-x-auto py-0.5">
            <span className="text-xs font-semibold text-muted-foreground mr-1 flex items-center gap-1 shrink-0">
              <CalendarIcon className="h-3.5 w-3.5 text-primary" /> Time Window:
            </span>
            {timeWindowPills.map((pill) => {
              const isActive = timeWindow === pill.id
              return (
                <button
                  key={pill.id}
                  onClick={() => {
                    updateFilters(
                      {
                        timeWindow: pill.id !== 'overall' ? pill.id : undefined,
                        startDate: pill.id === 'custom' ? customStartDate || undefined : undefined,
                        endDate: pill.id === 'custom' ? customEndDate || undefined : undefined,
                        page: undefined,
                      },
                      true
                    )
                  }}
                  className={`px-2.5 py-1 rounded-md text-xs font-medium transition-all shrink-0 cursor-pointer ${
                    isActive
                      ? 'bg-primary text-primary-foreground font-semibold shadow-xs'
                      : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                  }`}
                >
                  {pill.label}
                </button>
              )
            })}
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleExportCSV}
              disabled={filteredItems.length === 0}
              className="h-7 text-xs gap-1.5 px-2.5 cursor-pointer"
              title="Export visible transactions to CSV"
            >
              <Download className="h-3.5 w-3.5 text-muted-foreground" />
              Export CSV
            </Button>

            {isFiltered && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleResetFilters}
                className="h-7 text-xs text-muted-foreground hover:text-foreground gap-1 px-2 cursor-pointer"
              >
                <RotateCcw className="h-3 w-3" /> Reset Filters
              </Button>
            )}
          </div>
        </div>

        {/* Custom Date Range Picker inputs when Custom Range is active */}
        {timeWindow === 'custom' && (
          <div className="flex items-center gap-3 bg-muted/30 border rounded-lg p-3 text-xs">
            <span className="font-semibold text-foreground flex items-center gap-1">
              <SlidersHorizontal className="h-3.5 w-3.5 text-primary" /> Custom Range:
            </span>
            <div className="flex items-center gap-2 flex-wrap">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">From:</span>
                <Input
                  type="date"
                  value={customStartDate}
                  onChange={(e) => {
                    updateFilters({ startDate: e.target.value || undefined, page: undefined }, true)
                  }}
                  className="h-8 text-xs w-36"
                />
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">To:</span>
                <Input
                  type="date"
                  value={customEndDate}
                  onChange={(e) => {
                    updateFilters({ endDate: e.target.value || undefined, page: undefined }, true)
                  }}
                  className="h-8 text-xs w-36"
                />
              </div>
            </div>
          </div>
        )}

        {/* Search & Dropdown Filters Grid */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4 lg:grid-cols-5">
          {/* Search Input with Instant Clear Button */}
          <div className="relative md:col-span-2">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search narration, payee, UPI, ref#..."
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  updateFilters({ search: searchInput.trim() || undefined, page: undefined }, true)
                }
              }}
              className="pl-9 pr-8 text-xs"
            />
            {searchInput && (
              <button
                type="button"
                onClick={() => {
                  setSearchInput('')
                  updateFilters({ search: undefined, page: undefined }, true)
                }}
                className="absolute right-2.5 top-2.5 text-muted-foreground hover:text-foreground p-0.5 rounded-full cursor-pointer"
                title="Clear search"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>

          {/* Account Filter */}
          <div>
            <Select
              value={accountId}
              onValueChange={(val) => {
                updateFilters({ account: val && val !== 'ALL' ? val : undefined, page: undefined }, true)
              }}
            >
              <SelectTrigger className="text-xs">
                <SelectValue placeholder="All Accounts" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL">All Accounts & Cards</SelectItem>
                {accountsData?.map((a) => (
                  <SelectItem key={a.id} value={a.id}>
                    {a.bank_name} {a.account_number_mask ? `(${a.account_number_mask})` : ''}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Category Filter */}
          <div>
            <Select
              value={categoryId}
              onValueChange={(val) => {
                updateFilters({ category: val && val !== 'ALL' ? val : undefined, page: undefined }, true)
              }}
            >
              <SelectTrigger className="text-xs">
                <SelectValue placeholder="All Categories" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL">All Categories</SelectItem>
                {categoriesData?.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Type Filter */}
          <div>
            <Select
              value={txType}
              onValueChange={(val) => {
                updateFilters({ type: val && val !== 'ALL' ? val : undefined, page: undefined }, true)
              }}
            >
              <SelectTrigger className="text-xs">
                <SelectValue placeholder="All Types" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ALL">All Types</SelectItem>
                <SelectItem value="DEBIT">Debits (Spend)</SelectItem>
                <SelectItem value="CREDIT">Credits (Inflow)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Amount Range Filter Row */}
          <div className="md:col-span-4 lg:col-span-5 flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-border/50 text-xs">
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-muted-foreground text-[11px] font-semibold mr-1">
                Amount:
              </span>
              {[
                { label: 'All', min: '', max: '' },
                { label: '> ₹1k', min: '1000', max: '' },
                { label: '> ₹5k', min: '5000', max: '' },
                { label: '> ₹10k', min: '10000', max: '' },
                { label: 'Under ₹500', min: '', max: '500' },
              ].map((preset) => {
                const isActive = minAmount === preset.min && maxAmount === preset.max
                return (
                  <button
                    key={preset.label}
                    onClick={() => {
                      setMinInput(preset.min)
                      setMaxInput(preset.max)
                      updateFilters(
                        {
                          minAmount: preset.min || undefined,
                          maxAmount: preset.max || undefined,
                          page: undefined,
                        },
                        true
                      )
                    }}
                    className={`px-2 py-0.5 rounded-md border text-[10px] font-mono transition-colors cursor-pointer ${
                      isActive
                        ? 'bg-primary text-primary-foreground font-semibold'
                        : 'bg-muted/40 text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {preset.label}
                  </button>
                )
              })}
            </div>

            <div className="flex items-center gap-1.5">
              <Input
                type="number"
                placeholder="Min ₹"
                value={minInput}
                onChange={(e) => setMinInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    updateFilters({ minAmount: minInput || undefined, maxAmount: maxInput || undefined, page: undefined }, true)
                  }
                }}
                className="h-7 w-20 text-[11px] font-mono"
              />
              <span className="text-muted-foreground text-[10px]">–</span>
              <Input
                type="number"
                placeholder="Max ₹"
                value={maxInput}
                onChange={(e) => setMaxInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    updateFilters({ minAmount: minInput || undefined, maxAmount: maxInput || undefined, page: undefined }, true)
                  }
                }}
                className="h-7 w-20 text-[11px] font-mono"
              />
            </div>
          </div>
        </div>

        {/* Active Filters Pill Bar (when filters applied) */}
        {isFiltered && (
          <div className="flex items-center gap-1.5 flex-wrap pt-2 border-t border-border/40 text-[11px]">
            <span className="text-muted-foreground font-medium">Active filters:</span>
            {search && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Search: "{search}"
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() => {
                    setSearchInput('')
                    updateFilters({ search: undefined, page: undefined }, true)
                  }}
                />
              </Badge>
            )}
            {activeAccount && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Account: {activeAccount.bank_name} ({activeAccount.account_number_mask})
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() => updateFilters({ account: undefined, page: undefined }, true)}
                />
              </Badge>
            )}
            {activeCategory && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Category: {activeCategory.name}
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() => updateFilters({ category: undefined, page: undefined }, true)}
                />
              </Badge>
            )}
            {txType !== 'ALL' && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Type: {txType === 'DEBIT' ? 'Debits' : 'Credits'}
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() => updateFilters({ type: undefined, page: undefined }, true)}
                />
              </Badge>
            )}
            {timeWindow !== 'overall' && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Window: {timeWindow}
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() =>
                    updateFilters(
                      { timeWindow: undefined, startDate: undefined, endDate: undefined, page: undefined },
                      true
                    )
                  }
                />
              </Badge>
            )}
            {(minAmount || maxAmount) && (
              <Badge variant="secondary" className="gap-1 font-mono text-[10px] px-1.5 py-0.5">
                Amount: ₹{minAmount || '0'} – {maxAmount ? `₹${maxAmount}` : '∞'}
                <X
                  className="h-3 w-3 cursor-pointer hover:text-foreground"
                  onClick={() => {
                    setMinInput('')
                    setMaxInput('')
                    updateFilters({ minAmount: undefined, maxAmount: undefined, page: undefined }, true)
                  }}
                />
              </Badge>
            )}
          </div>
        )}
      </Card>

      {/* Financial KPIs Strip for Current View */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <Card className="p-3 border-border/70 bg-card/80 flex items-center justify-between">
          <div className="space-y-0.5">
            <p className="text-[11px] font-medium text-muted-foreground">Page Records</p>
            <p className="text-base font-bold font-mono text-foreground tabular-nums">
              {filteredItems.length}
              <span className="text-[11px] font-normal text-muted-foreground ml-1">/ {totalCount}</span>
            </p>
          </div>
          <div className="h-8 w-8 rounded-lg bg-muted/60 flex items-center justify-center text-muted-foreground">
            <Wallet className="h-4 w-4" />
          </div>
        </Card>

        <Card className="p-3 border-border/70 bg-card/80 flex items-center justify-between">
          <div className="space-y-0.5">
            <p className="text-[11px] font-medium text-rose-400">Page Debits (Spend)</p>
            <p className="text-base font-bold font-mono text-rose-400 tabular-nums">
              <PrivacyAmount amount={pageStats.debits} prefix="- " />
            </p>
          </div>
          <div className="h-8 w-8 rounded-lg bg-rose-500/10 flex items-center justify-center text-rose-400">
            <TrendingDown className="h-4 w-4" />
          </div>
        </Card>

        <Card className="p-3 border-border/70 bg-card/80 flex items-center justify-between">
          <div className="space-y-0.5">
            <p className="text-[11px] font-medium text-emerald-400">Page Credits (Inflow)</p>
            <p className="text-base font-bold font-mono text-emerald-400 tabular-nums">
              <PrivacyAmount amount={pageStats.credits} prefix="+ " />
            </p>
          </div>
          <div className="h-8 w-8 rounded-lg bg-emerald-500/10 flex items-center justify-center text-emerald-400">
            <TrendingUp className="h-4 w-4" />
          </div>
        </Card>

        <Card className="p-3 border-border/70 bg-card/80 flex items-center justify-between">
          <div className="space-y-0.5">
            <p className="text-[11px] font-medium text-muted-foreground">Page Net Cashflow</p>
            <p className={`text-base font-bold font-mono tabular-nums ${pageStats.net >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}>
              <PrivacyAmount amount={Math.abs(pageStats.net)} prefix={pageStats.net >= 0 ? '+ ' : '- '} />
            </p>
          </div>
          <div className="h-8 w-8 rounded-lg bg-muted/60 flex items-center justify-center text-muted-foreground">
            <Building2 className="h-4 w-4" />
          </div>
        </Card>
      </div>

      {/* Top Pagination & Quick Navigation Bar */}
      <div className="flex items-center justify-between gap-3 flex-wrap px-1 text-xs">
        <div className="text-muted-foreground font-mono flex items-center gap-1.5">
          <span>Showing</span>
          <span className="font-semibold text-foreground">{startItem}–{endItem}</span>
          <span>of</span>
          <span className="font-semibold text-foreground">{totalCount}</span>
          <span>transactions</span>
          {isFetching && <span className="inline-block h-2 w-2 rounded-full bg-primary animate-ping ml-1" title="Updating..." />}
        </div>

        <div className="flex items-center gap-2 flex-wrap">
          {/* Rows per page selector */}
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground text-[11px]">Rows:</span>
            <Select value={String(pageSize)} onValueChange={handlePageSizeChange}>
              <SelectTrigger className="h-7 w-[95px] text-xs font-mono">
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="end">
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <SelectItem key={size} value={String(size)} className="font-mono text-xs">
                    {size} / page
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Quick Page Jump Input */}
          <form onSubmit={handleJumpToPage} className="flex items-center gap-1">
            <Input
              type="number"
              min={1}
              max={totalPages}
              placeholder={`Page 1–${totalPages}`}
              value={jumpPageInput}
              onChange={(e) => setJumpPageInput(e.target.value)}
              className="h-7 w-24 text-[11px] font-mono"
            />
            <Button type="submit" variant="secondary" size="sm" className="h-7 px-2 text-xs cursor-pointer">
              Go
            </Button>
          </form>
        </div>
      </div>

      {/* Transaction Table Card */}
      <Card className="overflow-hidden border-border/80 bg-card shadow-xs">
        <div className="overflow-x-auto max-h-[70vh]">
          <Table>
            <TableHeader className="sticky top-0 bg-card/95 backdrop-blur-xs z-10 border-b shadow-xs">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id} className="hover:bg-transparent">
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id} className="text-xs font-semibold text-muted-foreground uppercase tracking-wider bg-muted/40 py-3">
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-44 text-center">
                    <div className="flex flex-col items-center justify-center space-y-2 text-muted-foreground">
                      <div className="h-6 w-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                      <p className="text-xs">Loading transactions ledger...</p>
                    </div>
                  </TableCell>
                </TableRow>
              ) : table.getRowModel().rows.length > 0 ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    className="hover:bg-muted/60 transition-colors group cursor-pointer"
                    onClick={() => setSelectedTx(row.original)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className="py-2.5">
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-44 text-center">
                    <div className="flex flex-col items-center justify-center space-y-2 text-muted-foreground">
                      <Info className="h-8 w-8 text-muted-foreground/60" />
                      <p className="text-sm font-semibold text-foreground">No matching transactions found</p>
                      <p className="text-xs max-w-sm text-muted-foreground">
                        Try adjusting your search terms, time window, or filter criteria.
                      </p>
                      {isFiltered && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={handleResetFilters}
                          className="mt-2 text-xs gap-1.5 cursor-pointer"
                        >
                          <RotateCcw className="h-3 w-3" /> Clear all filters
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        {/* Enhanced Bottom Pagination Footer */}
        <div className="flex flex-col sm:flex-row items-center justify-between gap-3 border-t border-border/60 bg-muted/20 px-4 py-3">
          {/* Range and Info */}
          <div className="text-xs text-muted-foreground font-mono">
            Showing <span className="font-semibold text-foreground">{startItem}–{endItem}</span> of{' '}
            <span className="font-semibold text-foreground">{totalCount}</span> transactions
            {startDate && endDate && (
              <span className="ml-2 text-[11px] text-muted-foreground hidden lg:inline">
                ({formatDate(startDate)} – {formatDate(endDate)})
              </span>
            )}
          </div>

          {/* Full Navigation Buttons: First, Prev, Dynamic Page Pills, Next, Last */}
          <div className="flex items-center gap-1.5 flex-wrap justify-center">
            {/* First Page Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => updateFilters({ page: undefined }, false)}
              disabled={page <= 1}
              className="h-8 w-8 p-0 cursor-pointer"
              title="First Page"
            >
              <ChevronsLeft className="h-3.5 w-3.5" />
              <span className="sr-only">First Page</span>
            </Button>

            {/* Previous Page Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => updateFilters({ page: Math.max(1, page - 1) > 1 ? page - 1 : undefined }, false)}
              disabled={page <= 1}
              className="h-8 gap-1 px-2.5 text-xs cursor-pointer"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              <span className="hidden sm:inline">Prev</span>
            </Button>

            {/* Direct Page Number Pills */}
            <div className="flex items-center gap-1">
              {paginationPages.map((item, idx) => {
                if (item === 'ellipsis') {
                  return (
                    <span key={`ellipsis-${idx}`} className="px-1.5 text-xs text-muted-foreground select-none">
                      …
                    </span>
                  )
                }
                const isCurrent = item === page
                return (
                  <button
                    key={`page-${item}`}
                    onClick={() => updateFilters({ page: (item as number) > 1 ? (item as number) : undefined }, false)}
                    className={`h-8 min-w-[32px] px-2 rounded-md text-xs font-mono transition-all cursor-pointer ${
                      isCurrent
                        ? 'bg-primary text-primary-foreground font-bold shadow-xs'
                        : 'bg-muted/40 hover:bg-muted text-muted-foreground hover:text-foreground border border-transparent hover:border-border'
                    }`}
                  >
                    {item}
                  </button>
                )
              })}
            </div>

            {/* Next Page Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => updateFilters({ page: Math.min(totalPages, page + 1) }, false)}
              disabled={page >= totalPages}
              className="h-8 gap-1 px-2.5 text-xs cursor-pointer"
            >
              <span className="hidden sm:inline">Next</span>
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>

            {/* Last Page Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => updateFilters({ page: totalPages }, false)}
              disabled={page >= totalPages}
              className="h-8 w-8 p-0 cursor-pointer"
              title={`Last Page (${totalPages})`}
            >
              <ChevronsRight className="h-3.5 w-3.5" />
              <span className="sr-only">Last Page</span>
            </Button>
          </div>

          {/* Jump to page form & Rows Selector */}
          <div className="flex items-center gap-2">
            <form onSubmit={handleJumpToPage} className="flex items-center gap-1">
              <span className="text-muted-foreground text-[11px] hidden md:inline">Go to:</span>
              <Input
                type="number"
                min={1}
                max={totalPages}
                placeholder={String(page)}
                value={jumpPageInput}
                onChange={(e) => setJumpPageInput(e.target.value)}
                className="h-8 w-16 text-xs font-mono text-center"
              />
              <Button type="submit" variant="secondary" size="sm" className="h-8 px-2.5 text-xs cursor-pointer">
                Go
              </Button>
            </form>
          </div>
        </div>
      </Card>

      {/* Transaction Details Dialog */}
      <Dialog open={!!selectedTx} onOpenChange={(open) => !open && setSelectedTx(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <Info className="h-4 w-4 text-primary" /> Transaction Details
            </DialogTitle>
            <DialogDescription className="text-xs">
              Complete raw metadata and parsed financial fields
            </DialogDescription>
          </DialogHeader>
          {selectedTx && (
            <div className="space-y-3 pt-2 text-xs">
              <div className="rounded-lg border p-3 bg-muted/30 space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-muted-foreground">Amount:</span>
                  <span className="font-bold font-mono text-base text-foreground">
                    <PrivacyAmount
                      amount={selectedTx.amount}
                      prefix={selectedTx.tx_type === 'CREDIT' ? '+ ' : '- '}
                      className={selectedTx.tx_type === 'CREDIT' ? 'text-emerald-400' : 'text-rose-400'}
                    />
                  </span>
                </div>
                {selectedTx.running_balance !== undefined && selectedTx.running_balance !== null && (
                  <div className="flex justify-between items-center">
                    <span className="text-muted-foreground">Running Balance:</span>
                    <span className="font-mono text-xs font-semibold text-foreground">
                      <PrivacyAmount amount={selectedTx.running_balance} />
                    </span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Date:</span>
                  <span className="font-mono text-foreground">{formatDate(selectedTx.tx_date)}</span>
                </div>
                {selectedTx.value_date && selectedTx.value_date !== selectedTx.tx_date && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Value Date:</span>
                    <span className="font-mono text-foreground">{formatDate(selectedTx.value_date)}</span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Type:</span>
                  <Badge variant={selectedTx.tx_type === 'CREDIT' ? 'default' : 'secondary'} className="text-[10px]">
                    {selectedTx.tx_type}
                  </Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Payment Mode:</span>
                  <span className="font-medium text-foreground">{selectedTx.payment_mode}</span>
                </div>
                {selectedTx.account_name && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Account:</span>
                    <span className="font-medium text-foreground">{selectedTx.account_name}</span>
                  </div>
                )}
                <div className="flex justify-between items-center py-0.5">
                  <span className="text-muted-foreground flex items-center gap-1.5">
                    Category:
                    {selectedTx.is_manual_category && (
                      <Badge variant="outline" className="text-[10px] px-1.5 py-0 text-primary border-primary/40 font-normal">
                        Manual
                      </Badge>
                    )}
                  </span>
                  <Select
                    value={selectedTx.category_id || 'UNSET'}
                    onValueChange={(val) => {
                      const newCatId = val === 'UNSET' ? '' : val
                      updateTxMutation.mutate({
                        id: selectedTx.id,
                        payload: { category_id: newCatId },
                      })
                    }}
                    disabled={updateTxMutation.isPending && updatingTxId === selectedTx.id}
                  >
                    <SelectTrigger className="h-8 text-xs w-48 sm:w-56 cursor-pointer">
                      <SelectValue placeholder="Select Category" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="UNSET">
                        <span className="text-muted-foreground italic">Uncategorized</span>
                      </SelectItem>
                      {categoriesData?.map((cat) => (
                        <SelectItem key={cat.id} value={cat.id}>
                          <div className="flex items-center gap-2">
                            <span className="h-2 w-2 rounded-full shrink-0" style={{ backgroundColor: cat.color_hex }} />
                            <span>{cat.name}</span>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                {selectedTx.cleaned_payee && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Cleaned Payee:</span>
                    <span className="font-semibold text-foreground">{selectedTx.cleaned_payee}</span>
                  </div>
                )}
                {selectedTx.upi_vpa && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">UPI VPA:</span>
                    <span className="font-mono text-foreground">{selectedTx.upi_vpa}</span>
                  </div>
                )}
                {selectedTx.card_last4 && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Card Swiped:</span>
                    <span className="font-mono text-foreground">•••• {selectedTx.card_last4}</span>
                  </div>
                )}
                {selectedTx.reference_number && (
                  <div className="flex justify-between items-center">
                    <span className="text-muted-foreground">Ref / UTR:</span>
                    <div className="flex items-center gap-1.5">
                      <span className="font-mono text-foreground">{selectedTx.reference_number}</span>
                      <CopyButton value={selectedTx.reference_number} label="Copy Ref / UTR" />
                    </div>
                  </div>
                )}
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                    Raw Statement Narration
                  </p>
                  <CopyButton value={selectedTx.raw_narration} label="Copy Narration" />
                </div>
                <div className="p-2.5 rounded-md bg-muted font-mono text-[11px] text-foreground break-all border">
                  {selectedTx.raw_narration}
                </div>
              </div>

              {/* Notes & Tags Editor */}
              <div className="space-y-2 pt-2 border-t">
                <div className="flex items-center justify-between">
                  <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                    Notes & Tags
                  </p>
                  {isEditingDetails && (
                    <Button
                      size="sm"
                      variant="secondary"
                      className="h-6 text-[11px] px-2.5 gap-1 font-semibold cursor-pointer"
                      disabled={updateTxMutation.isPending}
                      onClick={() => {
                        updateTxMutation.mutate({
                          id: selectedTx.id,
                          payload: { notes: editNotes, tags: editTags },
                        })
                        setIsEditingDetails(false)
                      }}
                    >
                      <Save className="h-3 w-3" /> Save Notes & Tags
                    </Button>
                  )}
                </div>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                  <div>
                    <label className="text-[10px] text-muted-foreground block mb-0.5 font-medium">Custom Notes</label>
                    <Input
                      placeholder="Add private note..."
                      className="h-8 text-xs bg-background"
                      value={isEditingDetails ? editNotes : selectedTx.notes || ''}
                      onChange={(e) => {
                        setIsEditingDetails(true)
                        setEditNotes(e.target.value)
                      }}
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-muted-foreground block mb-0.5 font-medium">Tags (comma-separated)</label>
                    <Input
                      placeholder="e.g. tax, business, personal"
                      className="h-8 text-xs bg-background"
                      value={isEditingDetails ? editTags : selectedTx.tags || ''}
                      onChange={(e) => {
                        setIsEditingDetails(true)
                        setEditTags(e.target.value)
                      }}
                    />
                  </div>
                </div>
              </div>

              {/* Quick Action: Create Auto-Categorization Rule */}
              <div className="pt-2 border-t flex justify-end">
                <Button
                  size="sm"
                  onClick={() => handleOpenQuickRule(selectedTx)}
                  className="text-xs font-semibold gap-1.5 w-full sm:w-auto cursor-pointer"
                >
                  <Sparkles className="h-3.5 w-3.5 text-primary-foreground" /> Create Auto-Rule from Payee
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Quick Auto-Rule Creation Dialog */}
      <Dialog open={showQuickRuleModal} onOpenChange={setShowQuickRuleModal}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold flex items-center gap-2">
              <Sparkles className="h-4 w-4 text-primary" /> Create Auto-Categorization Rule
            </DialogTitle>
            <DialogDescription className="text-xs">
              Future and existing transactions matching this pattern will be categorized automatically
            </DialogDescription>
          </DialogHeader>

          {quickRuleSuccessMsg ? (
            <div className="py-6 flex flex-col items-center justify-center space-y-2 text-center">
              <CheckCircle2 className="h-8 w-8 text-emerald-400" />
              <p className="text-xs font-semibold text-foreground">{quickRuleSuccessMsg}</p>
            </div>
          ) : (
            <div className="space-y-3 py-2 text-xs">
              <div className="space-y-1">
                <label className="font-semibold text-foreground">Target Field</label>
                <Select value={quickRuleField} onValueChange={(val) => setQuickRuleField(val || 'cleaned_payee')}>
                  <SelectTrigger className="text-xs h-8.5">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="cleaned_payee">Cleaned Payee Name</SelectItem>
                    <SelectItem value="raw_narration">Raw Statement Narration</SelectItem>
                    <SelectItem value="upi_vpa">UPI VPA</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1">
                <label className="font-semibold text-foreground">Pattern / Match Keyword *</label>
                <Input
                  value={quickRulePattern}
                  onChange={(e) => setQuickRulePattern(e.target.value)}
                  className="text-xs font-mono h-8.5"
                  placeholder="e.g. SWIGGY"
                />
              </div>

              <div className="space-y-1">
                <label className="font-semibold text-foreground">Assign Category *</label>
                <Select value={quickRuleCategory} onValueChange={(val) => setQuickRuleCategory(val || '')}>
                  <SelectTrigger className="text-xs h-8.5">
                    <SelectValue placeholder="Select Category" />
                  </SelectTrigger>
                  <SelectContent>
                    {categoriesData?.map((c) => (
                      <SelectItem key={c.id} value={c.id}>
                        {c.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <DialogFooter className="gap-2 pt-3">
                <Button variant="outline" size="sm" onClick={() => setShowQuickRuleModal(false)} className="cursor-pointer">
                  Cancel
                </Button>
                <Button
                  size="sm"
                  onClick={() => createRuleQuickMutation.mutate()}
                  disabled={createRuleQuickMutation.isPending || !quickRulePattern.trim() || !quickRuleCategory}
                  className="font-semibold gap-1.5 cursor-pointer"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {createRuleQuickMutation.isPending ? 'Saving & Applying...' : 'Create & Apply Rule'}
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Smart Rule Suggestion Floating Prompt */}
      {rulePromptTx && (
        <div className="fixed bottom-6 right-6 z-50 max-w-md rounded-xl border border-primary/30 bg-card p-4 shadow-xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-4">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-2 text-primary font-semibold text-xs">
              <Sparkles className="h-4 w-4" />
              <span>Create Auto-Rule for Future Transactions?</span>
            </div>
            <button
              onClick={() => setRulePromptTx(null)}
              className="text-muted-foreground hover:text-foreground text-xs cursor-pointer"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
          <p className="text-xs text-muted-foreground mt-1.5 leading-relaxed">
            You categorized this transaction as <strong className="text-foreground">{rulePromptTx.newCategoryName}</strong>. Would you like to automatically assign this category to all future transactions from <strong className="text-foreground">{rulePromptTx.tx.cleaned_payee}</strong>?
          </p>
          <div className="mt-3 flex items-center justify-end gap-2">
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs cursor-pointer"
              onClick={() => setRulePromptTx(null)}
            >
              Dismiss
            </Button>
            <Button
              size="sm"
              className="h-7 text-xs gap-1.5 font-semibold cursor-pointer"
              onClick={() => {
                handleOpenQuickRule(rulePromptTx.tx, rulePromptTx.tx.category_id)
                setRulePromptTx(null)
              }}
            >
              <Sparkles className="h-3 w-3" />
              Create Rule
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
