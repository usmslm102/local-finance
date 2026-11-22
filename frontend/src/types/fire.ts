export interface FireAsset {
  kind: 'MARKET' | 'EMPLOYER' | 'PF' | 'NPS' | 'OTHER'
  name: string
  currency: 'INR' | 'USD'
  balance: number
  monthly: number
  monthly_increase: number
  return_percent: number
  access_age: number
  accessible_percent: number
}
export interface FireScenario {
  age: number
  horizon: number
  monthly_expense: number
  retirement_expense: number
  expense_change_years: number
  inflation_percent: number
  withdrawal_percent: number
  contribution_growth_percent: number
  monthly_income: number
  income_growth_percent: number
  invest_surplus: boolean
  usd_inr: number
  fx_growth_percent: number
  rsu_unvested_usd: number
  rsu_annual_usd: number
  rsu_vest_years: number
  rsu_grant_growth_percent: number
  rsu_net_percent: number
  assets: FireAsset[]
}
export interface FireMonth {
  month: string
  observed: boolean
  expense: number
  salary: number
  india: number
  us: number
  deposits: number
  other: number
}
export interface FireBaseline {
  months: FireMonth[]
  observed_months: number
  monthly_expense: number
  monthly_salary: number
  monthly_india: number
  monthly_us_inr: number
  monthly_deposits: number
  monthly_other: number
  india_balance: number
  usd_balance: number
  portfolios: { provider: string; as_of: string; currency: string; value: number | null }[]
  warnings: string[]
}
export interface FirePoint {
  age: number
  portfolio: number
  accessible: number
  today_value: number
  target: number
  invested: number
  monthly_investment: number
  monthly_expense: number
  retired: boolean
}
export interface FireResult {
  retirement_age: number | null
  today_target: number
  at_retirement: FirePoint | null
  points: FirePoint[]
  warnings: string[]
}
