import { lazy } from 'react'

export const TransactionTable = lazy(() =>
  import('@/components/transactions/TransactionTable').then((module) => ({
    default: module.TransactionTable,
  }))
)
export const StatementImportView = lazy(() =>
  import('@/components/import/StatementImportView').then((module) => ({
    default: module.StatementImportView,
  }))
)
export const CalendarView = lazy(() =>
  import('@/components/calendar/CalendarView').then((module) => ({ default: module.CalendarView }))
)
export const SettingsView = lazy(() =>
  import('@/components/settings/SettingsView').then((module) => ({ default: module.SettingsView }))
)
export const SubscriptionsView = lazy(() =>
  import('@/components/subscriptions/SubscriptionsView').then((module) => ({
    default: module.SubscriptionsView,
  }))
)
export const CardsView = lazy(() =>
  import('@/components/cards/CardsView').then((module) => ({ default: module.CardsView }))
)
export const BudgetView = lazy(() =>
  import('@/components/budget/BudgetView').then((module) => ({ default: module.BudgetView }))
)
export const ReconcileView = lazy(() =>
  import('@/components/reconcile/ReconcileView').then((module) => ({
    default: module.ReconcileView,
  }))
)
export const MerchantsView = lazy(() =>
  import('@/components/merchants/MerchantsView').then((module) => ({
    default: module.MerchantsView,
  }))
)
export const CashFlowView = lazy(() =>
  import('@/components/cashflow/CashFlowView').then((module) => ({ default: module.CashFlowView }))
)
export const GuideView = lazy(() =>
  import('@/components/guide/GuideView').then((module) => ({ default: module.GuideView }))
)
export const WhatsNewView = lazy(() =>
  import('@/components/whatsnew/WhatsNewView').then((module) => ({ default: module.WhatsNewView }))
)
export const WrappedView = lazy(() =>
  import('@/components/wrapped/WrappedView').then((module) => ({ default: module.WrappedView }))
)
export const SalaryView = lazy(() =>
  import('@/components/salary/SalaryView').then((module) => ({ default: module.SalaryView }))
)
export const InvestmentsView = lazy(() =>
  import('@/components/investments/InvestmentsView').then((module) => ({
    default: module.InvestmentsView,
  }))
)

export const PlanningView = lazy(() =>
  import('@/components/planning/PlanningView').then((module) => ({ default: module.PlanningView }))
)
export const SplitwiseView = lazy(() =>
  import('@/components/splitwise/SplitwiseView').then((module) => ({ default: module.SplitwiseView }))
)
export const DataHealthView = lazy(() =>
  import('@/components/data-health/DataHealthView').then((module) => ({
    default: module.DataHealthView,
  }))
)
