import type { SankeyNode, Transaction } from '@/types'
import type { TransactionParams } from './api'

export function cashFlowPeriodDates(
  period: string,
  availableMonths: string[]
): Pick<TransactionParams, 'start_date' | 'end_date'> {
  let start = period,
    end = period
  if (period === 'ALL') return {}
  const financialYear = /^FY-(\d{4})/.exec(period)
  if (financialYear) {
    const year = Number(financialYear[1])
    return { start_date: `${year}-04-01`, end_date: `${year + 1}-03-31` }
  }
  if (period === 'LAST_3_MONTHS') {
    const months = [...availableMonths].sort().reverse()
    if (!months.length) return {}
    end = months[0]
    // The backend uses the three latest recorded months, including gaps.
    if (months.length < 3) return {}
    start = months[2]
  }
  if (!/^\d{4}-\d{2}$/.test(start) || !/^\d{4}-\d{2}$/.test(end)) return {}
  const [year, month] = end.split('-').map(Number)
  return { start_date: `${start}-01`, end_date: `${end}-${new Date(year, month, 0).getDate()}` }
}

export function incomeSource(tx: Transaction) {
  const narration = (tx.raw_narration ?? '').toUpperCase(),
    payee = (tx.cleaned_payee ?? '').toUpperCase()
  if (
    tx.payment_mode === 'SALARY' ||
    ['SALARY', 'PAYROLL', 'SAL/', 'MONTHLY SAL'].some((word) => narration.includes(word)) ||
    payee.includes('SALARY')
  )
    return 'src_salary'
  if (['REFUND', 'CASHBACK', 'REVERSAL'].some((word) => narration.includes(word)))
    return 'src_refund'
  if (
    ['INTEREST', 'DIVIDEND', 'ZERODHA', 'GROWW', 'MUTUAL FUND', 'FD ', 'TERM DEP'].some((word) =>
      narration.includes(word)
    )
  )
    return 'src_invest'
  return 'src_other'
}

export function cashFlowNodeParams(node: SankeyNode): TransactionParams {
  if (node.type === 'CATEGORY') return { tx_type: 'DEBIT', category_id: node.id.slice(4) }
  if (node.type === 'ACCOUNT') return { account_id: node.id.slice(4) }
  if (node.type === 'CHANNEL')
    return { account_id: node.id.replace(/^chan_card_/, ''), tx_type: 'DEBIT' }
  if (node.type === 'INCOME_SOURCE') return { tx_type: 'CREDIT' }
  return {}
}

export function contributingTransactions(node: SankeyNode, transactions: Transaction[]) {
  if (node.type === 'SURPLUS' || node.type === 'DEFICIT') return []
  const params = cashFlowNodeParams(node)
  return transactions.filter(
    (tx) =>
      !tx.is_transfer &&
      !tx.is_excluded &&
      tx.category_id !== 'cat_transfers' &&
      (!params.account_id ||
        (params.account_id === 'splitwise'
          ? tx.account_id === ''
          : tx.account_id === params.account_id)) &&
      (!params.category_id || tx.category_id === params.category_id) &&
      (!params.tx_type || tx.tx_type === params.tx_type) &&
      (node.type !== 'INCOME_SOURCE' || incomeSource(tx) === node.id)
  )
}
