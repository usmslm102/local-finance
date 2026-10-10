export interface SplitwiseEntry {
  id: string
  group: string
  person: string
  date: string
  description: string
  category: string
  cost_cents: number
  net_cents: number
  share_cents: number
  kind: 'EXPENSE' | 'PAYMENT'
  status: 'REVIEW' | 'CONFIRMED' | 'IGNORED'
  transaction_id?: string
  category_id?: string
  members: string[]
}

export interface SplitwiseMember {
  group: string
  name: string
  person: string
  aliases: string[]
  pattern: string
  suggested_pattern: string
  matched_payments: number
}
