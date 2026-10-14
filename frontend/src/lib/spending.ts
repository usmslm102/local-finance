import type { CategorySpend, SankeyNode } from '@/types'

export function cashFlowCategories(nodes: SankeyNode[], total: number): CategorySpend[] {
  return nodes
    .filter((node) => node.type === 'CATEGORY' && node.total_value > 0)
    .map((node) => ({
      category_id: node.id.slice(4),
      category_name: node.name,
      color_hex: node.color_hex,
      icon: node.icon ?? '',
      total_amount: node.total_value,
      tx_count: node.tx_count ?? 0,
      percentage: total > 0 ? (node.total_value / total) * 100 : 0,
    }))
    .sort((a, b) => b.total_amount - a.total_amount)
}

export function validSpendingMonth(value: unknown): value is string {
  return typeof value === 'string' && /^(?!0000)\d{4}-(0[1-9]|1[0-2])$/.test(value)
}

export function spendingMonthLabel(month?: string): string {
  if (!validSpendingMonth(month)) return 'All time'
  const date = new Date(`${month}-01T12:00:00`)
  return date.toLocaleDateString('en-IN', { month: 'long', year: 'numeric' })
}
