import { MonthlyReview } from '@/components/cashflow/MonthlyReview'
import {
  createRouter,
  createRoute,
  createRootRoute,
  Outlet,
  useRouterState,
  Link,
} from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { AppSidebar } from '@/components/layout/AppSidebar'
import { CommandPalette } from '@/components/layout/CommandPalette'
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { OverviewCard } from '@/components/dashboard/OverviewCard'
import { SpendingChart } from '@/components/dashboard/SpendingChart'
import { CashFlowChart } from '@/components/dashboard/CashFlowChart'
import { TopPayeesList } from '@/components/dashboard/TopPayeesList'
import { AccountCard } from '@/components/dashboard/AccountCard'
import { CreditCardBillsCard } from '@/components/dashboard/CreditCardBillsCard'
import { TransactionTable } from '@/components/transactions/TransactionTable'
import { StatementUploader } from '@/components/import/StatementUploader'
import { CalendarView } from '@/components/calendar/CalendarView'
import { SettingsView } from '@/components/settings/SettingsView'
import { SubscriptionsView } from '@/components/subscriptions/SubscriptionsView'
import { CardsView } from '@/components/cards/CardsView'
import { BudgetView } from '@/components/budget/BudgetView'
import { ReconcileView } from '@/components/reconcile/ReconcileView'
import { MerchantsView } from '@/components/merchants/MerchantsView'
import { CashFlowView } from '@/components/cashflow/CashFlowView'
import { GuideView } from '@/components/guide/GuideView'
import { WhatsNewView } from '@/components/whatsnew/WhatsNewView'
import { WrappedView } from '@/components/wrapped/WrappedView'
import { SalaryView } from '@/components/salary/SalaryView'
import { PrivacyProvider } from '@/components/privacy-provider'
import { PrivacyToggle } from '@/components/layout/PrivacyToggle'
import { UpdateIndicator } from '@/components/updates/UpdateIndicator'
import { NetWorthRunwayCard } from '@/components/dashboard/NetWorthRunwayCard'
import type {
  TransactionsSearchParams,
  TimeWindowPreset,
  MerchantsSearchParams,
  BudgetSearchParams,
  CashFlowSearchParams,
  SubscriptionsSearchParams,
  SettingsSearchParams,
  SalarySearchParams,
} from '@/types'
import { fetchAccounts, fetchAnalytics, fetchCreditCardBills } from '@/lib/api'
import { formatINR } from '@/lib/utils'
import {
  Wallet,
  TrendingDown,
  TrendingUp,
  Landmark,
  UploadCloud,
  Lock,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { TooltipProvider } from '@/components/ui/tooltip'
import { ThemeProvider } from '@/components/theme-provider'
import { AuthProvider, useAuth } from '@/components/auth/AuthProvider'
import { LockScreen } from '@/components/auth/LockScreen'

// Root Layout Component Content
const RootLayoutContent: React.FC = () => {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname
  const { isLocked, authStatus, lockApp } = useAuth()

  const getPageTitle = (path: string) => {
    switch (path) {
      case '/': return 'Dashboard'
      case '/transactions': return 'Transactions Ledger'
      case '/cashflow': return 'Cash Flow & Sankey'
      case '/salary': return 'Salary & Income Insights'
      case '/calendar': return 'Spending Calendar'
      case '/budget': return 'Category Budgets'
      case '/cards': return 'Credit Cards & Rewards'
      case '/subscriptions': return 'Subscriptions & Mandates'
      case '/merchants': return 'Merchant Intelligence'
      case '/reconcile': return 'Transfer Reconciler'
      case '/import': return 'Import Statements'
      case '/guide': return 'User Guide & Walkthrough'
      case '/whats-new': return "What's New in LocalFinance"
      case '/wrapped': return 'LocalFinance Wrapped'
      case '/settings': return 'Settings'
      case '/categories': return 'Categorization Rules'
      default: return 'Finance Intelligence'
    }
  }

  return (
    <TooltipProvider>
      <SidebarProvider defaultOpen={true}>
        <div className="flex min-h-screen w-full bg-background text-foreground antialiased">
          <AppSidebar />
          <SidebarInset className="flex flex-col flex-1 min-w-0">
            {/* Top Bar Header */}
            <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center justify-between gap-2 border-b bg-background/95 backdrop-blur-md px-3 sm:px-6">
              <div className="flex items-center gap-2 sm:gap-3 min-w-0">
                <SidebarTrigger className="-ml-1 shrink-0" />
                <Separator orientation="vertical" className="h-4 shrink-0 hidden xs:block" />
                <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground truncate max-w-[140px] sm:max-w-none">
                  {getPageTitle(currentPath)}
                </span>
              </div>

              <div className="flex items-center gap-2 sm:gap-2.5 shrink-0">
                {authStatus?.auth_enabled && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => lockApp()}
                    className="h-8 gap-1.5 text-xs text-muted-foreground hover:text-foreground px-2"
                    title="Lock Application"
                  >
                    <Lock className="h-3.5 w-3.5 text-primary" />
                    <span className="hidden sm:inline font-medium">Lock</span>
                  </Button>
                )}
                <UpdateIndicator />
                <PrivacyToggle />
                <CommandPalette />
                {currentPath !== '/import' && (
                  <Link to="/import">
                    <Button size="sm" variant="default" className="h-8 gap-1.5 text-xs font-medium shadow-xs transition-all active:scale-[0.98] px-2.5 sm:px-3">
                      <UploadCloud className="h-3.5 w-3.5 shrink-0" />
                      <span className="hidden xs:inline">Import</span>
                    </Button>
                  </Link>
                )}
              </div>
            </header>

            {/* Main Workspace Area */}
            <main className="flex-1 w-full max-w-7xl mx-auto px-3 sm:px-6 lg:px-8 py-4 sm:py-6">
              {isLocked ? <LockScreen /> : <Outlet />}
            </main>

            {/* Minimal Footer */}
            <footer className="border-t bg-card/40 py-4 text-center text-xs text-muted-foreground">
              <p>LocalFinance • 100% Offline Local Personal Finance Intelligence • Indian Ecosystem</p>
            </footer>
          </SidebarInset>
        </div>
      </SidebarProvider>
    </TooltipProvider>
  )
}

const RootComponent: React.FC = () => {
  return (
    <ThemeProvider defaultTheme="dark" storageKey="local-finance-theme">
      <PrivacyProvider>
        <AuthProvider>
          <RootLayoutContent />
        </AuthProvider>
      </PrivacyProvider>
    </ThemeProvider>
  )
}

const rootRoute = createRootRoute({
  component: RootComponent,
})

// Overview Dashboard Page
const DashboardPage: React.FC = () => {
  const { data: analytics, isLoading } = useQuery({
    queryKey: ['analytics'],
    queryFn: fetchAnalytics,
  })

  const { data: accounts } = useQuery({
    queryKey: ['accounts'],
    queryFn: fetchAccounts,
  })

  const { data: ccBills } = useQuery({
    queryKey: ['credit-card-bills'],
    queryFn: () => fetchCreditCardBills(),
  })

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground text-sm">
        Loading financial analytics...
      </div>
    )
  }

  const totalIncome = analytics?.total_income || 0
  const totalExpense = analytics?.total_expense || 0

  return (
    <div className="space-y-8">
      {/* Header section */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            Financial Intelligence
          </h1>
          <p className="text-xs text-muted-foreground mt-1">
            Offline spending intelligence across your Indian bank accounts & credit cards
          </p>
        </div>
        <Link to="/import">
          <Button size="sm" className="gap-2 font-semibold shadow-xs">
            <UploadCloud className="h-4 w-4" />
            <span>Import Statement</span>
          </Button>
        </Link>
      </div>

      {/* First-time Welcome Banner when no transactions exist */}
      {(!analytics || analytics.total_transactions === 0) && (
        <Card className="border-primary/30 bg-gradient-to-br from-primary/10 via-card to-card p-6 sm:p-8 shadow-xs space-y-4">
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="space-y-1.5 max-w-xl">
              <Badge variant="outline" className="text-[10px] font-mono border-primary/30 text-primary">
                100% Offline • Zero Cloud • Local Storage
              </Badge>
              <h2 className="text-xl sm:text-2xl font-bold tracking-tight text-foreground">
                Welcome to LocalFinance! 🚀
              </h2>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Your privacy-first financial intelligence engine is ready. Import your bank or credit card statements (PDF, Excel, CSV) to unlock automated Indian UPI cleaning, cash flow Sankey visualization, 50-day float optimizer, and Year-in-Review Wrapped.
              </p>
            </div>

            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2.5 shrink-0 w-full sm:w-auto">
              <Link to="/import">
                <Button size="sm" className="w-full gap-2 text-xs font-semibold shadow-xs">
                  <UploadCloud className="h-4 w-4" />
                  <span>Import Statement</span>
                </Button>
              </Link>
              <Link to="/guide">
                <Button size="sm" variant="outline" className="w-full text-xs font-semibold">
                  <span>View User Guide</span>
                </Button>
              </Link>
            </div>
          </div>
        </Card>
      )}

      <MonthlyReview compact />

      {/* KPI Overview Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <OverviewCard
          title="Total Inflow"
          amount={formatINR(totalIncome)}
          numericValue={totalIncome}
          subtitle="Salary, refunds & credits"
          icon={TrendingUp}
          variant="income"
        />
        <OverviewCard
          title="Total Outflow"
          amount={formatINR(totalExpense)}
          numericValue={totalExpense}
          subtitle="Debits & card expenses"
          icon={TrendingDown}
          variant="expense"
        />
        <OverviewCard
          title="Bank Liquidity"
          amount={formatINR(analytics?.total_bank_liquidity || 0)}
          numericValue={analytics?.total_bank_liquidity || 0}
          subtitle="Savings & Current accounts"
          icon={Landmark}
          variant="savings"
        />
        <OverviewCard
          title="Credit Card Dues"
          amount={formatINR(analytics?.total_credit_due || 0)}
          numericValue={analytics?.total_credit_due || 0}
          subtitle={
            analytics?.total_credit_limit
              ? `${(analytics.credit_utilization_rate || 0).toFixed(1)}% of ${formatINR(analytics.total_credit_limit)} limit`
              : 'Active card dues'
          }
          icon={Wallet}
          variant={analytics?.credit_utilization_rate && analytics.credit_utilization_rate > 30 ? 'expense' : 'default'}
        />
      </div>

      {/* Liquid Net Worth & Emergency Runway Gauge */}
      {analytics && (
        <NetWorthRunwayCard
          totalLiquidity={analytics.total_bank_liquidity || 0}
          totalCreditDue={analytics.total_credit_due || 0}
          monthlyExpenses={analytics.monthly_trends?.map((m) => m.expense) || []}
        />
      )}

      {/* Credit Card Dues & Billing Statements */}
      {ccBills && ccBills.length > 0 && (
        <CreditCardBillsCard bills={ccBills} />
      )}

      {/* Accounts List / Cards */}
      {accounts && accounts.length > 0 && (
        <div className="space-y-3">
          <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
            <Landmark className="h-3.5 w-3.5 text-primary" /> Linked Accounts & Cards ({accounts.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {accounts.map((acc) => (
              <AccountCard key={acc.id} account={acc} />
            ))}
          </div>
        </div>
      )}

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        <div className="lg:col-span-6">
          <SpendingChart data={analytics?.category_breakdown || []} />
        </div>
        <div className="lg:col-span-6">
          <CashFlowChart data={analytics?.monthly_trends || []} />
        </div>
      </div>

      {/* Top Payees */}
      <TopPayeesList data={analytics?.top_payees || []} />
    </div>
  )
}

// Routes
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardPage,
})

const transactionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/transactions',
  validateSearch: (search: Record<string, unknown>): TransactionsSearchParams => {
    const pageNum = search.page ? Number(search.page) : undefined
    const pageSizeNum = search.pageSize
      ? Number(search.pageSize)
      : search.page_size
        ? Number(search.page_size)
        : undefined
    const searchStr = (search.search as string) || (search.q as string) || undefined
    const accountStr = (search.account as string) || (search.account_id as string) || undefined
    const categoryStr = (search.category as string) || (search.category_id as string) || undefined
    const typeStr = (search.type as string) || (search.tx_type as string) || undefined
    const timeWindowStr = (search.timeWindow as TimeWindowPreset) || (search.preset as TimeWindowPreset) || undefined
    const startStr = (search.startDate as string) || (search.start_date as string) || undefined
    const endStr = (search.endDate as string) || (search.end_date as string) || undefined
    const minAmtStr = search.minAmount ? String(search.minAmount) : search.min_amount ? String(search.min_amount) : undefined
    const maxAmtStr = search.maxAmount ? String(search.maxAmount) : search.max_amount ? String(search.max_amount) : undefined

    return {
      page: pageNum && !isNaN(pageNum) && pageNum > 0 ? pageNum : undefined,
      pageSize: pageSizeNum && !isNaN(pageSizeNum) && pageSizeNum > 0 ? pageSizeNum : undefined,
      search: searchStr?.trim() || undefined,
      account: accountStr && accountStr !== 'ALL' ? accountStr : undefined,
      category: categoryStr && categoryStr !== 'ALL' ? categoryStr : undefined,
      type: typeStr && typeStr !== 'ALL' ? typeStr.toUpperCase() : undefined,
      timeWindow: timeWindowStr && timeWindowStr !== 'overall' ? timeWindowStr : undefined,
      startDate: startStr || undefined,
      endDate: endStr || undefined,
      minAmount: minAmtStr || undefined,
      maxAmount: maxAmtStr || undefined,
    }
  },
  component: () => (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Transactions Ledger</h1>
        <p className="text-xs text-muted-foreground mt-1">
          Detailed transaction ledger with UPI extraction, payee resolution, and custom filtering
        </p>
      </div>
      <TransactionTable />
    </div>
  ),
})

const calendarRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/calendar',
  component: () => <CalendarView />,
})

const importRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/import',
  component: () => (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground">Statement Import</h1>
        <p className="text-xs text-muted-foreground mt-1">
          Upload bank and credit card files with auto-detection and smart duplicate fingerprinting
        </p>
      </div>
      <StatementUploader />
    </div>
  ),
})

const subscriptionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/subscriptions',
  validateSearch: (search: Record<string, unknown>): SubscriptionsSearchParams => {
    const statusStr = (search.status as string) || undefined
    const categoryStr =
      (search.category as string) ||
      (search.service as string) ||
      (search.type as string) ||
      undefined
    const searchStr = (search.search as string) || (search.q as string) || undefined
    return {
      status: statusStr && statusStr !== 'ALL' ? statusStr : undefined,
      category: categoryStr && categoryStr !== 'ALL' ? categoryStr : undefined,
      search: searchStr?.trim() || undefined,
    }
  },
  component: () => <SubscriptionsView />,
})

const cardsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/cards',
  component: () => <CardsView />,
})

const budgetRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/budget',
  validateSearch: (search: Record<string, unknown>): BudgetSearchParams => {
    const monthStr = (search.month as string) || undefined
    const tabStr = (search.tab as BudgetSearchParams['tab']) || undefined
    const searchStr = (search.search as string) || (search.q as string) || undefined
    return {
      month: monthStr || undefined,
      tab: tabStr && tabStr !== 'ALL' ? tabStr : undefined,
      search: searchStr?.trim() || undefined,
    }
  },
  component: () => <BudgetView />,
})

const reconcileRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reconcile',
  component: () => <ReconcileView />,
})

const merchantsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/merchants',
  validateSearch: (search: Record<string, unknown>): MerchantsSearchParams => {
    const searchStr = (search.search as string) || (search.q as string) || undefined
    const categoryStr = (search.category as string) || undefined
    const sortStr =
      (search.sortBy as string) ||
      (search.sort as string) ||
      (search.sort_by as string) ||
      undefined
    const payeeStr = (search.payee as string) || (search.merchant as string) || undefined
    return {
      search: searchStr?.trim() || undefined,
      category: categoryStr && categoryStr !== 'ALL' ? categoryStr : undefined,
      sortBy: sortStr && sortStr !== 'total_spend' ? sortStr : undefined,
      payee: payeeStr || undefined,
    }
  },
  component: () => <MerchantsView />,
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  validateSearch: (search: Record<string, unknown>): SettingsSearchParams => {
    const tabStr = (search.tab as SettingsSearchParams['tab']) || undefined
    return {
      tab: tabStr && tabStr !== 'general' ? tabStr : undefined,
    }
  },
  component: () => <SettingsView />,
})

const categoriesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/categories',
  validateSearch: (search: Record<string, unknown>): SettingsSearchParams => {
    const tabStr = (search.tab as SettingsSearchParams['tab']) || undefined
    return {
      tab: tabStr || undefined,
    }
  },
  component: () => <SettingsView initialTab="rules" />,
})

const cashflowRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/cashflow',
  validateSearch: (search: Record<string, unknown>): CashFlowSearchParams => {
    const periodStr = (search.month as string) || (search.period as string) || undefined
    return {
      month: periodStr || undefined,
    }
  },
  component: () => <CashFlowView />,
})

const guideRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/guide',
  component: () => <GuideView />,
})

const whatsnewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/whats-new',
  component: () => <WhatsNewView />,
})

const wrappedRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/wrapped',
  component: () => <WrappedView />,
})

const salaryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/salary',
  validateSearch: (search: Record<string, unknown>): SalarySearchParams => {
    const pageNum = search.page !== undefined && search.page !== '' ? Number(search.page) : undefined
    const rawPageSize = search.pageSize !== undefined ? search.pageSize : search.page_size
    const pageSizeNum =
      rawPageSize !== undefined && rawPageSize !== '' ? Number(rawPageSize) : undefined
    const tabStr = (search.tab as string) || (search.view as string) || undefined
    const searchStr = (search.search as string) || (search.q as string) || undefined
    const companyStr = (search.company as string) || (search.employer as string) || undefined

    return {
      tab: tabStr === 'monthly' ? 'monthly' : tabStr === 'yearly' ? 'yearly' : undefined,
      company: companyStr && companyStr !== 'ALL' ? companyStr : undefined,
      search: searchStr?.trim() || undefined,
      page: pageNum && !isNaN(pageNum) && pageNum > 0 ? pageNum : undefined,
      pageSize: pageSizeNum !== undefined && !isNaN(pageSizeNum) && pageSizeNum >= 0 ? pageSizeNum : undefined,
    }
  },
  component: () => <SalaryView />,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  transactionsRoute,
  cashflowRoute,
  salaryRoute,
  merchantsRoute,
  cardsRoute,
  budgetRoute,
  reconcileRoute,
  calendarRoute,
  subscriptionsRoute,
  importRoute,
  guideRoute,
  whatsnewRoute,
  wrappedRoute,
  settingsRoute,
  categoriesRoute,
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
