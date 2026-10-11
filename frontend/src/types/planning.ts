export interface SavingsGoal {
  id: string
  name: string
  target_amount: number
  saved_amount: number
  monthly_contribution: number
  target_date: string
  created_at: string
  updated_at: string
}
export type GoalInput = Pick<
  SavingsGoal,
  'name' | 'target_amount' | 'saved_amount' | 'monthly_contribution' | 'target_date'
>
export interface DataHealthAccount {
  account_id: string
  latest_transaction: string
  transaction_count: number
  review_count: number
}
