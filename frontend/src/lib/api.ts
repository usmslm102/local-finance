import {
  Account,
  AnalyticsOverview,
  Category,
  CategorizationRule,
  ImportResult,
  ParserInfo,
  StatementImport,
  Transaction,
  TransactionListResponse,
} from '../types'

const BASE_URL = '/api'

export async function fetchInvestments(): Promise<import('../types/investments').InvestmentSnapshot[]> {
  const res = await fetchWithAuth(`${BASE_URL}/investments`)
  if (!res.ok) throw new Error('Unable to load investments')
  return res.json()
}

export async function fetchInvestmentFormats(): Promise<import('../types/investments').InvestmentFormats> {
  const res = await fetchWithAuth(`${BASE_URL}/investments/formats`)
  if (!res.ok) throw new Error('Unable to load supported investment formats')
  return res.json()
}

export async function previewInvestment(file: File): Promise<import('../types/investments').InvestmentSnapshot> {
  const body = new FormData()
  body.append('file', file)
  const res = await fetchWithAuth(`${BASE_URL}/investments/preview`, { method: 'POST', body })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Unable to preview investment statement' }))
    throw new Error(error.error)
  }
  return res.json()
}

export async function importInvestment(file: File): Promise<{ snapshot: import('../types/investments').InvestmentSnapshot; duplicate: boolean }> {
  const body = new FormData()
  body.append('file', file)
  const res = await fetchWithAuth(`${BASE_URL}/investments/import`, { method: 'POST', body })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: 'Unable to import investment statement' }))
    throw new Error(error.error)
  }
  return res.json()
}

export async function deleteInvestment(id: string): Promise<void> {
  const res = await fetchWithAuth(`${BASE_URL}/investments/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (!res.ok) throw new Error('Unable to delete investment snapshot')
}

export async function fetchMonthlyReview(month?: string): Promise<import('../types/monthly-review').MonthlyReviewData> {
  const query = new URLSearchParams(month ? { month } : {})
  const res = await fetchWithAuth(`${BASE_URL}/analytics/monthly-review?${query}`)
  if (!res.ok) throw new Error('Unable to load monthly review. Choose a past or current month, or try again.')
  return res.json()
}

export async function fetchMonthlyReviewEvidence(month: string, category: string, period: 'current' | 'previous', page: number): Promise<import('../types/monthly-review').ReviewEvidence> {
  const query = new URLSearchParams({ month, category, period, page: String(page) })
  const res = await fetchWithAuth(`${BASE_URL}/analytics/monthly-review/transactions?${query}`)
  if (!res.ok) throw new Error('Unable to load supporting transactions. Please try again.')
  return res.json()
}

export function getAuthToken(): string | null {
  return sessionStorage.getItem('local_finance_token') || localStorage.getItem('local_finance_token')
}

export function setAuthToken(token: string) {
  sessionStorage.setItem('local_finance_token', token)
  localStorage.setItem('local_finance_token', token)
}

export function clearAuthToken() {
  sessionStorage.removeItem('local_finance_token')
  localStorage.removeItem('local_finance_token')
}

async function fetchWithAuth(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const token = getAuthToken()
  const headers = new Headers(init?.headers)
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const res = await fetch(input, {
    ...init,
    headers,
  })

  if (res.status === 401) {
    try {
      const clone = res.clone()
      const data = await clone.json()
      if (data?.code === 'AUTH_REQUIRED') {
        window.dispatchEvent(new CustomEvent('auth:required'))
      }
    } catch {
      window.dispatchEvent(new CustomEvent('auth:required'))
    }
  }

  return res
}

export async function fetchAnalytics(): Promise<AnalyticsOverview> {
  const res = await fetchWithAuth(`${BASE_URL}/analytics/overview`)
  if (!res.ok) throw new Error('Failed to fetch analytics')
  return res.json()
}

export async function fetchAccounts(): Promise<Account[]> {
  const res = await fetchWithAuth(`${BASE_URL}/accounts`)
  if (!res.ok) throw new Error('Failed to fetch accounts')
  return res.json()
}

export async function updateAccount(
  id: string,
  req: import('../types').UpdateAccountRequest
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/accounts/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || 'Failed to update account')
  }
  return res.json()
}

export async function fetchCreditCardBills(accountId?: string): Promise<import('../types').CreditCardBill[]> {
  const query = accountId ? `?account_id=${encodeURIComponent(accountId)}` : ''
  const res = await fetchWithAuth(`${BASE_URL}/credit-cards/bills${query}`)
  if (!res.ok) throw new Error('Failed to fetch credit card bills')
  return res.json()
}

export interface TransactionParams {
  min_amount?: string
  max_amount?: string
  account_id?: string
  category_id?: string
  tx_type?: string
  search?: string
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export async function fetchTransactions(params: TransactionParams = {}): Promise<TransactionListResponse> {
  const query = new URLSearchParams()
  if (params.min_amount) query.append('min_amount', params.min_amount)
  if (params.max_amount) query.append('max_amount', params.max_amount)
  if (params.account_id) query.append('account_id', params.account_id)
  if (params.category_id) query.append('category_id', params.category_id)
  if (params.tx_type) query.append('tx_type', params.tx_type)
  if (params.search) query.append('search', params.search)
  if (params.start_date) query.append('start_date', params.start_date)
  if (params.end_date) query.append('end_date', params.end_date)
  if (params.page) query.append('page', params.page.toString())
  if (params.page_size) query.append('page_size', params.page_size.toString())

  const res = await fetchWithAuth(`${BASE_URL}/transactions?${query.toString()}`)
  if (!res.ok) throw new Error('Failed to fetch transactions')
  return res.json()
}

// For views that search or aggregate the whole matching set rather than one page.
export async function fetchAllTransactions(
  params: Omit<TransactionParams, 'page' | 'page_size'> = {}
): Promise<Transaction[]> {
  const items: Transaction[] = []
  let page = 1
  let totalPages = 1
  do {
    const result = await fetchTransactions({ ...params, page, page_size: 5000 })
    items.push(...result.items)
    totalPages = result.total_pages
    page += 1
  } while (page <= totalPages)
  return items
}

export interface UpdateTransactionPayload {
  category_id?: string | null
  notes?: string
  tags?: string
  is_manual_category?: boolean
}

export async function updateTransaction(
  id: string,
  payload: UpdateTransactionPayload
): Promise<Transaction> {
  const res = await fetchWithAuth(`${BASE_URL}/transactions/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to update transaction' }))
    throw new Error(errorData.error || 'Failed to update transaction')
  }
  return res.json()
}

export async function fetchCategories(): Promise<Category[]> {
  const res = await fetchWithAuth(`${BASE_URL}/categories`)
  if (!res.ok) throw new Error('Failed to fetch categories')
  return res.json()
}

export async function fetchRules(): Promise<CategorizationRule[]> {
  const res = await fetchWithAuth(`${BASE_URL}/rules`)
  if (!res.ok) throw new Error('Failed to fetch rules')
  return res.json()
}

export async function fetchStatements(): Promise<StatementImport[]> {
  const res = await fetchWithAuth(`${BASE_URL}/statements`)
  if (!res.ok) throw new Error('Failed to fetch statement history')
  return res.json()
}

export async function fetchParsers(): Promise<ParserInfo[]> {
  const res = await fetchWithAuth(`${BASE_URL}/parsers`)
  if (!res.ok) throw new Error('Failed to fetch parsers')
  return res.json()
}

export async function uploadStatement(formData: FormData): Promise<ImportResult> {
  const res = await fetchWithAuth(`${BASE_URL}/statements/upload`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Upload failed' }))
    throw new Error(errorData.error || 'Upload failed')
  }
  return res.json()
}

export async function previewStatement(formData: FormData): Promise<import('../types').StatementPreviewResult> {
  const res = await fetchWithAuth(`${BASE_URL}/statements/preview`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Preview failed' }))
    throw new Error(errorData.error || 'Preview failed')
  }
  return res.json()
}

export async function previewBatchStatements(formData: FormData): Promise<import('../types').StatementPreviewResult[]> {
  const res = await fetchWithAuth(`${BASE_URL}/statements/preview-batch`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Batch preview failed' }))
    throw new Error(errorData.error || 'Batch preview failed')
  }
  return res.json()
}

export async function uploadBatchStatements(formData: FormData): Promise<import('../types').BatchImportResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/statements/upload-batch`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Batch upload failed' }))
    throw new Error(errorData.error || 'Batch upload failed')
  }
  return res.json()
}

export async function resetDatabase(): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/database/reset`, {
    method: 'POST',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to reset database' }))
    throw new Error(errorData.error || 'Failed to reset database')
  }
  return res.json()
}

export async function fetchDatabaseInfo(): Promise<import('../types').DatabaseInfo> {
  const res = await fetchWithAuth(`${BASE_URL}/database/info`)
  if (!res.ok) throw new Error('Failed to fetch database info')
  return res.json()
}

export async function restoreDatabase(file: File): Promise<{ message: string; info: import('../types').DatabaseInfo }> {
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetchWithAuth(`${BASE_URL}/database/restore`, {
    method: 'POST',
    body: formData,
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to restore database' }))
    throw new Error(errorData.error || 'Failed to restore database')
  }
  return res.json()
}

export async function fetchSubscriptions(): Promise<import('../types').SubscriptionsSummary> {
  const res = await fetchWithAuth(`${BASE_URL}/subscriptions`)
  if (!res.ok) throw new Error('Failed to fetch subscriptions')
  return res.json()
}

export async function scanSubscriptions(): Promise<import('../types').SubscriptionsSummary> {
  const res = await fetchWithAuth(`${BASE_URL}/subscriptions/scan`, {
    method: 'POST',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to scan subscriptions' }))
    throw new Error(errorData.error || 'Failed to scan subscriptions')
  }
  return res.json()
}

export async function createSubscription(data: import('../types').UpsertSubscriptionRequest): Promise<import('../types').Subscription> {
  const res = await fetchWithAuth(`${BASE_URL}/subscriptions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to create subscription' }))
    throw new Error(errorData.error || 'Failed to create subscription')
  }
  return res.json()
}

export async function updateSubscription(id: string, data: Partial<import('../types').UpsertSubscriptionRequest>): Promise<import('../types').Subscription> {
  const res = await fetchWithAuth(`${BASE_URL}/subscriptions/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to update subscription' }))
    throw new Error(errorData.error || 'Failed to update subscription')
  }
  return res.json()
}

export async function deleteSubscription(id: string): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/subscriptions/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to delete subscription' }))
    throw new Error(errorData.error || 'Failed to delete subscription')
  }
  return res.json()
}

export async function createRule(data: Partial<import('../types').CategorizationRule>): Promise<import('../types').CategorizationRule> {
  const res = await fetchWithAuth(`${BASE_URL}/rules`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to create rule' }))
    throw new Error(errorData.error || 'Failed to create rule')
  }
  return res.json()
}

export async function updateRule(id: string, data: Partial<import('../types').CategorizationRule>): Promise<import('../types').CategorizationRule> {
  const res = await fetchWithAuth(`${BASE_URL}/rules/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to update rule' }))
    throw new Error(errorData.error || 'Failed to update rule')
  }
  return res.json()
}

export async function deleteRule(id: string): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/rules/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to delete rule' }))
    throw new Error(errorData.error || 'Failed to delete rule')
  }
  return res.json()
}

export async function reapplyRules(): Promise<{ updated_count: number; message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/rules/reapply`, {
    method: 'POST',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to reapply rules' }))
    throw new Error(errorData.error || 'Failed to reapply rules')
  }
  return res.json()
}

export async function createCategory(data: Partial<import('../types').Category>): Promise<import('../types').Category> {
  const res = await fetchWithAuth(`${BASE_URL}/categories`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to create category' }))
    throw new Error(errorData.error || 'Failed to create category')
  }
  return res.json()
}

export async function deleteCategory(id: string): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/categories/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to delete category' }))
    throw new Error(errorData.error || 'Failed to delete category')
  }
  return res.json()
}

// -------------------------------------------------------------
// Credit Card Rewards, Optimization & Best-Card Engine
// -------------------------------------------------------------

export async function fetchCardPortfolioOverview(): Promise<import('../types').CardPortfolioOverview> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/portfolio`)
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to fetch card portfolio' }))
    throw new Error(errorData.error || 'Failed to fetch card portfolio')
  }
  return res.json()
}

export async function recommendBestCards(
  data: import('../types').CardRecommendationRequest
): Promise<{
  merchant: string
  category?: string
  spend_amount: number
  recommendations: import('../types').CardRecommendation[]
}> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/recommend`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to recommend cards' }))
    throw new Error(errorData.error || 'Failed to recommend cards')
  }
  return res.json()
}

export async function updateCardMetadata(
  id: string,
  data: import('../types').UpdateCardMetadataRequest
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to update card' }))
    throw new Error(errorData.error || 'Failed to update card')
  }
  return res.json()
}

export async function fetchCardRewardRules(accountId?: string): Promise<import('../types').CardRewardRule[]> {
  const url = accountId ? `${BASE_URL}/cards/rules?account_id=${accountId}` : `${BASE_URL}/cards/rules`
  const res = await fetchWithAuth(url)
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to fetch reward rules' }))
    throw new Error(errorData.error || 'Failed to fetch reward rules')
  }
  return res.json()
}

export async function createCardRewardRule(
  data: Partial<import('../types').CardRewardRule>
): Promise<import('../types').CardRewardRule> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/rules`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to create reward rule' }))
    throw new Error(errorData.error || 'Failed to create reward rule')
  }
  return res.json()
}

export async function updateCardRewardRule(
  id: string,
  data: Partial<import('../types').CardRewardRule>
): Promise<import('../types').CardRewardRule> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/rules/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to update reward rule' }))
    throw new Error(errorData.error || 'Failed to update reward rule')
  }
  return res.json()
}

export async function deleteCardRewardRule(id: string): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/cards/rules/${id}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to delete reward rule' }))
    throw new Error(errorData.error || 'Failed to delete reward rule')
  }
  return res.json()
}

// ==========================================
// CATEGORY BUDGETING API
// ==========================================

export async function fetchBudgetSummary(month?: string): Promise<import('../types').BudgetSummary> {
  const url = month ? `${BASE_URL}/budgets?month=${encodeURIComponent(month)}` : `${BASE_URL}/budgets`
  const res = await fetchWithAuth(url)
  if (!res.ok) {
    throw new Error('Failed to fetch budget summary')
  }
  return res.json()
}

export async function upsertCategoryBudget(
  data: import('../types').UpsertCategoryBudgetRequest
): Promise<import('../types').BudgetSummary> {
  const res = await fetchWithAuth(`${BASE_URL}/budgets`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to save category budget' }))
    throw new Error(errorData.error || 'Failed to save category budget')
  }
  return res.json()
}

export async function deleteCategoryBudget(
  categoryId: string,
  month?: string
): Promise<{ message: string }> {
  const url = month
    ? `${BASE_URL}/budgets/${categoryId}?month=${encodeURIComponent(month)}`
    : `${BASE_URL}/budgets/${categoryId}`
  const res = await fetchWithAuth(url, {
    method: 'DELETE',
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to remove category budget' }))
    throw new Error(errorData.error || 'Failed to remove category budget')
  }
  return res.json()
}

// ==========================================
// RECONCILIATION API
// ==========================================

export async function fetchReconciliationSummary(): Promise<import('../types').ReconciliationSummary> {
  const res = await fetchWithAuth(`${BASE_URL}/reconciliation/summary`)
  if (!res.ok) {
    throw new Error('Failed to fetch reconciliation summary')
  }
  return res.json()
}

export async function scanReconciliation(): Promise<{
  message: string
  auto_linked_count: number
  wallet_excluded_count: number
}> {
  const res = await fetchWithAuth(`${BASE_URL}/reconciliation/scan`, {
    method: 'POST',
  })
  if (!res.ok) {
    throw new Error('Failed to scan reconciliation')
  }
  return res.json()
}

export async function linkTransferPair(
  data: import('../types').LinkTransferPairRequest
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/reconciliation/link`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to link transfer pair' }))
    throw new Error(errorData.error || 'Failed to link transfer pair')
  }
  return res.json()
}

export async function unlinkTransferPair(
  txId: string
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/reconciliation/unlink`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tx_id: txId }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to unlink transfer pair' }))
    throw new Error(errorData.error || 'Failed to unlink transfer pair')
  }
  return res.json()
}

export async function toggleExcludeTransaction(
  txId: string
): Promise<{ message: string; is_excluded: boolean }> {
  const res = await fetchWithAuth(`${BASE_URL}/reconciliation/toggle-exclude/${txId}`, {
    method: 'POST',
  })
  if (!res.ok) {
    throw new Error('Failed to toggle transaction exclusion')
  }
  return res.json()
}

// ==========================================
// MERCHANT INTELLIGENCE API
// ==========================================

export async function fetchMerchants(params?: {
  search?: string
  category?: string
  sort_by?: string
}): Promise<import('../types').MerchantListResponse> {
  const query = new URLSearchParams()
  if (params?.search) query.set('search', params.search)
  if (params?.category) query.set('category', params.category)
  if (params?.sort_by) query.set('sort_by', params.sort_by)

  const res = await fetchWithAuth(`${BASE_URL}/merchants?${query.toString()}`)
  if (!res.ok) {
    throw new Error('Failed to fetch merchants')
  }
  return res.json()
}

export async function fetchMerchantProfile(
  payeeName: string
): Promise<import('../types').MerchantProfile> {
  const res = await fetchWithAuth(`${BASE_URL}/merchants/${encodeURIComponent(payeeName)}`)
  if (!res.ok) {
    throw new Error('Failed to fetch merchant profile')
  }
  return res.json()
}

// ==========================================
// SAMPLE STATEMENT API
// ==========================================

export interface SampleStatementItem {
  category: string
  filename: string
  title: string
  bank: string
  type: string
  format: string
  path: string
}

export async function fetchSampleStatements(): Promise<SampleStatementItem[]> {
  const res = await fetchWithAuth(`${BASE_URL}/samples`)
  if (!res.ok) {
    throw new Error('Failed to fetch sample statements')
  }
  return res.json()
}

export async function loadSampleFile(relPath: string, filename: string): Promise<File> {
  const res = await fetchWithAuth(`${BASE_URL}/samples/download?path=${encodeURIComponent(relPath)}`)
  if (!res.ok) {
    throw new Error(`Failed to download sample statement: ${relPath}`)
  }
  const blob = await res.blob()
  const mimeType = filename.endsWith('.pdf')
    ? 'application/pdf'
    : filename.endsWith('.csv')
    ? 'text/csv'
    : 'application/octet-stream'
  return new File([blob], filename, { type: mimeType })
}

// ==========================================
// CASH FLOW SANKEY & MOM INTELLIGENCE API
// ==========================================

export async function fetchCashFlowIntelligence(
  period?: string
): Promise<import('../types').CashFlowIntelligenceResponse> {
  const query = period ? `?period=${encodeURIComponent(period)}` : ''
  const res = await fetchWithAuth(`${BASE_URL}/analytics/cashflow${query}`)
  if (!res.ok) {
    throw new Error('Failed to fetch cash flow intelligence')
  }
  return res.json()
}

// ==========================================
// SALARY & INCOME INSIGHTS API
// ==========================================

export async function fetchSalaryInsights(): Promise<import('../types').SalaryInsightsResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/analytics/salary`)
  if (!res.ok) {
    throw new Error('Failed to fetch salary insights')
  }
  return res.json()
}

// ==========================================
// FINANCIAL WRAPPED & YEAR IN REVIEW API
// ==========================================

export async function fetchWrappedStory(
  year?: string
): Promise<import('../types').WrappedStory> {
  const query = year ? `?year=${encodeURIComponent(year)}` : ''
  const res = await fetchWithAuth(`${BASE_URL}/analytics/wrapped${query}`)
  if (!res.ok) {
    throw new Error('Failed to fetch financial wrapped story')
  }
  return res.json()
}

// ==========================================
// LOCAL AUTH & SECURITY SETTINGS API
// ==========================================

export async function fetchAuthStatus(): Promise<import('../types').AuthStatusResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/status`)
  if (!res.ok) {
    throw new Error('Failed to fetch authentication status')
  }
  return res.json()
}

export async function setupAuth(
  password: string,
  autoLockMinutes: number = 60
): Promise<import('../types').AuthLoginResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/setup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password, auto_lock_minutes: autoLockMinutes }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Setup failed' }))
    throw new Error(errorData.error || 'Failed to setup master password')
  }
  const data: import('../types').AuthLoginResponse = await res.json()
  if (data.token) {
    setAuthToken(data.token)
  }
  return data
}

export async function loginAuth(
  password: string
): Promise<import('../types').AuthLoginResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Login failed' }))
    throw new Error(errorData.error || 'Invalid master password')
  }
  const data: import('../types').AuthLoginResponse = await res.json()
  if (data.token) {
    setAuthToken(data.token)
  }
  return data
}

export async function logoutAuth(): Promise<void> {
  try {
    await fetchWithAuth(`${BASE_URL}/auth/logout`, {
      method: 'POST',
    })
  } finally {
    clearAuthToken()
  }
}

export async function changeAuthPassword(
  currentPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/change-password`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Password change failed' }))
    throw new Error(errorData.error || 'Failed to change password')
  }
  return res.json()
}

export async function disableAuth(password: string): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/disable`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Failed to disable security' }))
    throw new Error(errorData.error || 'Failed to disable security')
  }
  clearAuthToken()
  return res.json()
}

export async function updateSecuritySettings(
  autoLockMinutes: number
): Promise<{ message: string }> {
  const res = await fetchWithAuth(`${BASE_URL}/auth/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ auto_lock_minutes: autoLockMinutes }),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: 'Update settings failed' }))
    throw new Error(errorData.error || 'Failed to update security settings')
  }
  return res.json()
}




// ==========================================
// SYSTEM VERSION & AUTO-UPDATER API
// ==========================================

export async function fetchSystemVersion(refresh = false, offline = false): Promise<import("../types").SystemVersionInfo> {
  const params = new URLSearchParams()
  if (refresh) params.set("refresh", "true")
  if (offline) params.set("offline", "true")
  const query = params.toString() ? `?${params.toString()}` : ""
  const res = await fetchWithAuth(`${BASE_URL}/system/version${query}`)
  if (!res.ok) {
    throw new Error("Failed to fetch system version")
  }
  return res.json()
}

export async function applySystemUpdate(): Promise<import("../types").ApplyUpdateResponse> {
  const res = await fetchWithAuth(`${BASE_URL}/system/update`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  })
  if (!res.ok) {
    const errorData = await res.json().catch(() => ({ error: "Update failed" }))
    throw new Error(errorData.error || "Failed to apply update")
  }
  return res.json()
}

export interface MCPSettings {
  enabled: boolean
  port: number
  has_token: boolean
  listening: boolean
  endpoint: string
  error?: string
}

async function mcpRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetchWithAuth(`${BASE_URL}/mcp/${path}`, init)
  const data = await res.json()
  if (!res.ok) throw new Error(data.error || 'Unable to update MCP access')
  return data
}

export function fetchMCPSettings(): Promise<MCPSettings> {
  return mcpRequest('settings')
}
export function updateMCPSettings(enabled: boolean, port: number): Promise<MCPSettings> {
  return mcpRequest('settings', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ enabled, port }) })
}
export function rotateMCPToken(): Promise<{ token: string; settings: MCPSettings }> {
  return mcpRequest('token/rotate', { method: 'POST' })
}
