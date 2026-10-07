export type AccountType = 'SAVINGS' | 'CURRENT' | 'CREDIT_CARD' | 'WALLET'

export type TxType = 'DEBIT' | 'CREDIT'

export type PaymentMode =
  | 'UPI'
  | 'CARD_POS'
  | 'CARD_ONLINE'
  | 'NET_BANKING'
  | 'NEFT'
  | 'IMPS'
  | 'RTGS'
  | 'ATM'
  | 'CHARGES'
  | 'INTEREST'
  | 'SALARY'
  | 'OTHER'

export interface Account {
  id: string
  bank_name: string
  account_type: AccountType
  account_number?: string
  account_number_mask: string
  currency: string
  opening_balance: number
  current_balance: number
  credit_limit?: number
  billing_cycle_day?: number
  nickname?: string
  customer_id?: string
  ifsc_code?: string
  branch_name?: string
  account_holder_name?: string
  card_network?: string
  card_variant?: string
  created_at: string
  updated_at: string
}

export interface UpdateAccountRequest {
  bank_name?: string
  account_type?: AccountType
  account_number?: string
  account_number_mask?: string
  nickname?: string
  account_holder_name?: string
}

export interface StatementImport {
  id: string
  account_id: string
  bank_name?: string
  filename: string
  file_hash: string
  statement_format: string
  parser_used: string
  start_date?: string
  end_date?: string
  total_transactions: number
  opening_balance?: number
  closing_balance?: number
  total_debits?: number
  total_credits?: number
  imported_at: string
}

export interface Category {
  id: string
  name: string
  parent_id?: string
  color_hex: string
  icon: string
  is_system: boolean
}

export interface CategorizationRule {
  id: string
  priority: number
  match_field: string
  match_type: string
  match_pattern: string
  exclude_pattern?: string
  tx_type?: 'ALL' | 'DEBIT' | 'CREDIT'
  target_category_id: string
  target_category?: string
  assign_tags?: string
  is_active: boolean
}

export interface Transaction {
  id: string
  account_id: string
  account_name?: string
  statement_import_id?: string
  tx_hash: string
  tx_date: string
  value_date?: string
  raw_narration: string
  cleaned_payee: string
  payment_mode: PaymentMode
  reference_number: string
  tx_type: TxType
  amount: number
  running_balance?: number
  category_id?: string
  category_name?: string
  category_color?: string
  category_icon?: string
  upi_vpa?: string
  card_last4?: string
  merchant_category?: string
  cashback_amount?: number
  reward_points_earned?: number
  is_transfer?: boolean
  is_excluded?: boolean
  transfer_peer_id?: string
  transfer_match_reason?: string
  net_amount?: number
  original_currency?: string
  original_amount?: number
  is_recurring: boolean
  is_manual_category?: boolean
  notes: string
  tags: string
  created_at: string
}

export interface CreditCardBill {
  id: string
  account_id: string
  bank_name?: string
  statement_import_id?: string
  statement_date: string
  payment_due_date: string
  total_due_amount: number
  minimum_due_amount?: number
  reward_points_earned: number
  reward_points_balance: number
  cashback_earned?: number
  cashback_credited?: number
  finance_charges?: number
  credit_limit?: number
  available_credit_limit?: number
  payment_status: 'UNPAID' | 'PAID' | 'PARTIALLY_PAID'
  created_at: string
}

export interface ImportResult {
  statement_import_id: string
  account_id: string
  bank_name: string
  account_type: string
  parser_used: string
  confidence: number
  total_parsed: number
  inserted_count: number
  duplicate_count: number
  start_date: string
  end_date: string
  warnings?: string[]
}

export interface PreviewTransactionItem {
  date: string
  value_date?: string
  raw_narration: string
  cleaned_payee: string
  payment_mode: PaymentMode
  reference_number: string
  tx_type: TxType
  amount: number
  running_balance?: number
  category_id?: string
  category_name?: string
  category_color?: string
  category_icon?: string
  upi_vpa?: string
  is_transfer: boolean
  is_duplicate: boolean
}

export interface StatementPreviewResult {
  file_id: string
  filename: string
  file_size: number
  file_hash: string
  parser_id: string
  parser_name: string
  confidence: number
  bank_name: string
  account_type: string
  account_number?: string
  account_number_mask: string
  account_holder_name?: string
  start_date: string
  end_date: string
  total_transactions: number
  opening_balance: number
  closing_balance: number
  total_debits: number
  total_credits: number
  total_due_amount: number
  minimum_due_amount: number
  payment_due_date: string
  credit_limit: number
  available_credit_limit: number
  reward_points_balance: number
  cashback_earned: number
  card_network: string
  card_variant: string
  transactions: PreviewTransactionItem[]
  existing_txs_count: number
  new_txs_count: number
  warnings?: string[]
  error?: string
  requires_password?: boolean
}

export interface BatchImportResponse {
  results: ImportResult[]
  errors: string[]
  success: number
  failed: number
}

export interface CategorySpend {
  category_id: string
  category_name: string
  color_hex: string
  icon: string
  total_amount: number
  tx_count: number
  percentage: number
}

export interface MonthlyCashFlow {
  month: string
  income: number
  expense: number
  net: number
}

export interface PayeeSpend {
  payee: string
  payment_mode: string
  total_spent: number
  tx_count: number
}

export interface AnalyticsOverview {
  total_income: number
  total_expense: number
  net_savings: number
  savings_rate: number
  total_bank_liquidity: number
  total_credit_due: number
  total_credit_limit: number
  credit_utilization_rate: number
  total_cashback_earned: number
  total_reward_points: number
  total_accounts: number
  total_transactions: number
  category_breakdown: CategorySpend[]
  monthly_trends: MonthlyCashFlow[]
  top_payees: PayeeSpend[]
  upcoming_bills?: CreditCardBill[]
}

export interface TransactionListResponse {
  items: Transaction[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface ParserInfo {
  id: string
  name: string
  supported_types: string[]
}

export interface DatabaseInfo {
  path: string
  file_size: number
  wal_mode: boolean
  last_modified: string
  total_accounts: number
  total_transactions: number
  total_statements: number
  total_rules: number
}

export type SubscriptionFrequency = 'MONTHLY' | 'QUARTERLY' | 'YEARLY' | 'WEEKLY'
export type SubscriptionStatus = 'ACTIVE' | 'PAUSED' | 'CANCELLED'

export interface Subscription {
  id: string
  name: string
  merchant_pattern: string
  category_id?: string
  category_name?: string
  category_color?: string
  category_icon?: string
  account_id?: string
  account_name?: string
  frequency: SubscriptionFrequency
  expected_amount: number
  currency: string
  billing_day: number
  next_due_date?: string
  last_paid_date?: string
  last_paid_amount?: number
  status: SubscriptionStatus
  is_auto_detected: boolean
  notes?: string
  created_at: string
  updated_at: string
}

export interface SubscriptionsSummary {
  total_active: number
  monthly_burn_rate: number
  annual_projected: number
  upcoming_in_30_days: number
  subscriptions: Subscription[]
}

export interface UpsertSubscriptionRequest {
  name: string
  merchant_pattern: string
  category_id?: string
  account_id?: string
  frequency: SubscriptionFrequency
  expected_amount: number
  billing_day?: number
  next_due_date?: string
  status?: SubscriptionStatus
  notes?: string
}

export interface CardRewardRule {
  id: string
  account_id: string
  merchant_pattern: string
  category_name: string
  reward_percentage: number
  reward_description: string
  max_cap_per_month?: number
  min_spend_per_txn?: number
  created_at: string
  updated_at: string
}

export interface CardDetails extends Account {
  annual_fee: number
  fee_waiver_threshold: number
  billing_day: number
  payment_due_days: number
  card_color: string
  reward_type: string
  base_reward_rate: number
  total_spend_this_year: number
  fee_waiver_progress_pct: number
  fee_waiver_remaining: number
  next_statement_date: string
  next_payment_due_date: string
  days_until_statement: number
  interest_free_days_remaining: number
  total_cashback_earned: number
  total_reward_points: number
  total_due_amount: number
  available_credit_limit: number
  reward_rules: CardRewardRule[]
}

export interface CardRecommendationRequest {
  merchant: string
  category?: string
  amount: number
}

export interface CardRecommendation {
  account_id: string
  bank_name: string
  card_variant: string
  card_network: string
  card_color: string
  reward_rate: number
  reward_description: string
  estimated_reward: number
  reward_type: string
  interest_free_days: number
  rank: number
  notes: string
}

export interface CardPortfolioOverview {
  cards: CardDetails[]
  best_card_to_swipe_today?: CardDetails
  total_credit_limit: number
  total_outstanding: number
  overall_utilization_rate: number
  total_annual_fee_liability: number
  total_fee_savings_projected: number
  total_cashback_earned: number
  total_reward_points: number
}

export interface UpdateCardMetadataRequest {
  card_variant?: string
  card_network?: string
  card_color?: string
  account_holder_name?: string
  credit_limit?: number
  annual_fee?: number
  fee_waiver_threshold?: number
  billing_day?: number
  payment_due_days?: number
  base_reward_rate?: number
  reward_type?: string
}

export interface CategoryBudget {
  id: string
  category_id: string
  category_name: string
  category_color: string
  category_icon: string
  monthly_limit: number
  month: string
  actual_spend: number
  spent_percentage: number
  remaining_amount: number
  status: 'SAFE' | 'WARNING' | 'EXCEEDED'
  days_elapsed_in_month: number
  days_remaining_in_month: number
  total_days_in_month: number
  daily_recommended_allowance: number
  projected_month_end_spend: number
  pacing_status: 'ON_TRACK' | 'PACING_HIGH' | 'PACING_EXCEEDED'
}

export interface BudgetSummary {
  month: string
  total_budget: number
  total_spent: number
  total_remaining: number
  overall_spent_percentage: number
  total_days_in_month: number
  days_elapsed_in_month: number
  days_remaining_in_month: number
  overall_daily_recommended_allowance: number
  categories_with_budgets: number
  overspent_categories_count: number
  warning_categories_count: number
  budgets: CategoryBudget[]
}

export interface UpsertCategoryBudgetRequest {
  category_id: string
  monthly_limit: number
  month?: string
}

// ==========================================
// TRANSFER RECONCILIATION TYPES
// ==========================================

export interface TransferPair {
  id: string
  debit_tx: Transaction
  credit_tx: Transaction
  match_confidence: number
  match_reason: string
  is_confirmed: boolean
  date_difference_days: number
  amount_difference: number
}

export interface ReconciliationSummary {
  total_paired_transfers: number
  total_paired_amount: number
  pending_candidates_count: number
  double_count_prevented_amount: number
  wallet_excluded_count: number
  wallet_excluded_amount: number
  pairs: TransferPair[]
  candidates: TransferPair[]
  wallet_transactions: Transaction[]
}

export interface LinkTransferPairRequest {
  debit_tx_id: string
  credit_tx_id: string
  match_reason?: string
}

export interface UnlinkTransferPairRequest {
  tx_id: string
}

// ==========================================
// MERCHANT INTELLIGENCE TYPES
// ==========================================

export interface MonthlySpendDataPoint {
  month: string
  spend_amount: number
  tx_count: number
}

export interface PaymentSourceShare {
  account_name: string
  spend_amount: number
  tx_count: number
  share_pct: number
}

export interface MerchantProfile {
  cleaned_payee: string
  category_name: string
  category_color: string
  category_icon: string
  total_spend: number
  total_credits: number
  net_spend: number
  total_tx_count: number
  debit_tx_count: number
  credit_tx_count: number
  average_order_value: number
  first_tx_date: string
  last_tx_date: string
  days_since_last_tx: number
  preferred_payment_mode: string
  preferred_payment_source: string
  payment_sources: PaymentSourceShare[]
  monthly_spend_history: MonthlySpendDataPoint[]
  recent_transactions: Transaction[]
}

export interface MerchantSummaryItem {
  cleaned_payee: string
  category_name: string
  category_color: string
  category_icon: string
  total_spend: number
  total_credits: number
  tx_count: number
  average_order_value: number
  first_tx_date: string
  last_tx_date: string
  spend_share_pct: number
  primary_source: string
}

export interface MerchantListResponse {
  total_merchants: number
  total_spend: number
  top_category: string
  average_order_value: number
  merchants: MerchantSummaryItem[]
}

// ==========================================
// CASH FLOW SANKEY & MOM INTELLIGENCE TYPES
// ==========================================

export type SankeyNodeType =
  | 'INCOME_SOURCE'
  | 'ACCOUNT'
  | 'CHANNEL'
  | 'CATEGORY'
  | 'SURPLUS'
  | 'DEFICIT'

export interface SankeyNode {
  id: string
  name: string
  type: SankeyNodeType
  color_hex: string
  icon?: string
  total_value: number
  layer: number
  tx_count?: number
}

export interface SankeyLink {
  source: string
  target: string
  value: number
  color_hex?: string
}

export interface SankeyData {
  nodes: SankeyNode[]
  links: SankeyLink[]
  total_inflow: number
  total_outflow: number
  net_surplus: number
  savings_rate: number
}

export type AnomalySeverity = 'warning' | 'info' | 'success' | 'danger'

export interface MoMAnomaly {
  id: string
  type: string
  severity: AnomalySeverity
  title: string
  description: string
  category_name?: string
  category_color?: string
  current_amount: number
  previous_amount: number
  delta_amount: number
  percentage_change: number
  top_contributor?: string
}

export interface MonthlyCategoryDataPoint {
  month: string
  amount: number
}

export interface CategoryComparisonItem {
  category_id: string
  category_name: string
  category_color: string
  category_icon: string
  current_spend: number
  previous_spend: number
  three_month_avg: number
  delta_amount: number
  percentage_change: number
  trend: 'UP' | 'DOWN' | 'FLAT'
  history: MonthlyCategoryDataPoint[]
  top_payees?: PayeeSpend[]
}

export interface CashFlowIntelligenceSummary {
  total_inflow: number
  total_outflow: number
  net_surplus: number
  savings_rate: number
  fixed_needs_spend: number
  discretionary_spend: number
  credit_card_share_pct: number
  direct_bank_share_pct: number
  top_spending_category: string
  top_spending_amount: number
}

export interface CashFlowIntelligenceResponse {
  period: string
  selected_month: string
  previous_month: string
  available_months: string[]
  summary: CashFlowIntelligenceSummary
  sankey: SankeyData
  anomalies: MoMAnomaly[]
  category_comparisons: CategoryComparisonItem[]
}

export interface WrappedMerchantHighlight {
  payee: string
  total_spent: number
  order_count: number
  payment_mode: string
}

export interface WrappedStory {
  year: string
  available_years: string[]
  total_income: number
  total_expense: number
  net_savings: number
  savings_rate: number
  total_transactions: number
  persona_title: string
  persona_badge: string
  persona_description: string
  crown_merchant?: WrappedMerchantHighlight
  top_merchants: WrappedMerchantHighlight[]
  biggest_purchase?: Transaction
  busiest_day?: string
  busiest_day_spend?: number
  busiest_day_tx_count?: number
  total_cashback: number
  total_reward_points: number
  top_categories: CategorySpend[]
  upi_tx_count: number
  card_tx_count: number
  total_card_spend: number
  total_upi_spend: number
}

export interface AuthStatusResponse {
  auth_enabled: boolean
  is_authenticated: boolean
  auto_lock_minutes: number
  file_permissions?: string
}

export interface AuthLoginResponse {
  message: string
  token: string
  expires_at: string
}

export interface SalaryYearSummary {
  year: number
  total_earned: number
  monthly_average: number
  paycheck_count: number
  yoy_growth_pct: number
}

export interface SalaryMonthlyDataPoint {
  month: string
  amount: number
  employer: string
  is_hike: boolean
  hike_pct?: number
  is_bonus: boolean
}

export interface EmployerSummary {
  employer_name: string
  total_earned: number
  first_paycheck: string
  last_paycheck: string
  paycheck_count: number
  monthly_average: number
}

export interface SalaryInsightsResponse {
  lifetime_earned: number
  total_paychecks: number
  first_salary_amount: number
  first_salary_date: string
  latest_salary_amount: number
  latest_salary_date: string
  latest_employer: string
  average_monthly: number
  peak_salary_amount: number
  peak_salary_date: string
  peak_employer: string
  overall_growth_pct: number
  current_year_earned: number
  current_fy_earned: number
  yearly_progress: SalaryYearSummary[]
  monthly_history: SalaryMonthlyDataPoint[]
  employers: EmployerSummary[]
  recent_paychecks: Transaction[]
}

export type TimeWindowPreset =
  | 'overall'
  | 'today'
  | 'yesterday'
  | 'this_week'
  | 'this_month'
  | 'this_year'
  | 'custom'

export interface TransactionsSearchParams {
  page?: number
  pageSize?: number
  search?: string
  q?: string
  account?: string
  account_id?: string
  category?: string
  category_id?: string
  type?: string
  tx_type?: string
  timeWindow?: TimeWindowPreset
  preset?: TimeWindowPreset
  startDate?: string
  start_date?: string
  endDate?: string
  end_date?: string
  minAmount?: string
  min_amount?: string
  maxAmount?: string
  max_amount?: string
}

export interface MerchantsSearchParams {
  search?: string
  q?: string
  category?: string
  sortBy?: string
  sort?: string
  sort_by?: string
  payee?: string
  merchant?: string
}

export interface BudgetSearchParams {
  month?: string
  tab?: 'ALL' | 'BUDGETED' | 'WARNING' | 'UNBUDGETED'
  search?: string
  q?: string
}

export interface CashFlowSearchParams {
  month?: string
  period?: string
}

export interface SubscriptionsSearchParams {
  status?: string
  category?: string
  service?: string
  type?: string
  search?: string
  q?: string
}

export interface SettingsSearchParams {
  tab?: 'general' | 'accounts' | 'categories' | 'rules' | 'security' | 'database' | 'sound' | 'mcp'
}

export interface CalendarSearchParams {
  month?: string
  date?: string
  day?: string
  search?: string
  q?: string
}

export interface SalarySearchParams {
  tab?: 'yearly' | 'monthly'
  view?: 'yearly' | 'monthly'
  company?: string
  employer?: string
  search?: string
  q?: string
  page?: number
  pageSize?: number
  page_size?: number
}

export interface ReconcileSearchParams {
  tab?: 'CANDIDATES' | 'PAIRED' | 'WALLET'
}


export interface SystemVersionInfo {
  current_version: string
  latest_version: string
  update_available: boolean
  can_auto_update: boolean
  auto_update_error?: string
  release_name: string
  release_notes: string
  release_url: string
  published_at: string
  asset_url?: string
  asset_name?: string
  asset_size?: number
  checksum_url?: string
  checked_at: string
}

export interface ApplyUpdateResponse {
  previous_version: string
  new_version: string
  backup_path: string
  message: string
}
