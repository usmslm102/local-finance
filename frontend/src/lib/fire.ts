import type { FireAsset, FireBaseline, FireResult, FireScenario } from '@/types/fire'

export const fireStorageKey = 'localfinance-fire-scenario-v1'
export function fireDefaults(b: FireBaseline, fx = 0): FireScenario {
  const asset = (kind: FireAsset['kind'], name: string, currency: FireAsset['currency'], balance = 0, monthly = 0, growth = 6, access = 0, available = 100): FireAsset => ({kind, name, currency, balance, monthly, return_percent: growth, access_age: access, accessible_percent: available, monthly_increase: 0})
  return {
    age: 0, horizon: 95, monthly_expense: Math.round(b.monthly_expense), retirement_expense: Math.round(b.monthly_expense), expense_change_years: 2,
    inflation_percent: 6, withdrawal_percent: 3, contribution_growth_percent: 10,
    monthly_income: Math.round(b.monthly_salary), income_growth_percent: 10, invest_surplus: false,
    usd_inr: fx, fx_growth_percent: 0, rsu_unvested_usd: 0, rsu_annual_usd: 0, rsu_vest_years: 4, rsu_grant_growth_percent: 0, rsu_net_percent: 70,
    assets: [
      asset('MARKET', 'Indian stocks / mutual funds', 'INR', b.india_balance, Math.round(b.monthly_india), 8),
      asset('MARKET', 'US stocks', 'USD', b.usd_balance, Math.round(b.monthly_us_inr), 6),
      asset('EMPLOYER', 'Employer stock', 'USD'),
      asset('OTHER', 'Fixed deposits', 'INR', 0, Math.round(b.monthly_deposits), 5),
      asset('PF', 'PF', 'INR', 0, 0, 6, 60),
      asset('NPS', 'NPS', 'INR', 0, 0, 7, 60, 0),
      asset('OTHER', 'Other investments', 'INR', 0, Math.round(b.monthly_other), 5),
    ],
  }
}

export function loadFireScenario(raw: string | null): FireScenario | null {
  if (!raw) return null
  try {
    const value: unknown = JSON.parse(raw)
    if (!value || typeof value !== 'object' || !('assets' in value) || !Array.isArray(value.assets)) return null
    const s = value as FireScenario
    const keys = ['age','horizon','monthly_expense','retirement_expense','expense_change_years','inflation_percent','withdrawal_percent','contribution_growth_percent','monthly_income','income_growth_percent','usd_inr','fx_growth_percent','rsu_unvested_usd','rsu_annual_usd','rsu_vest_years','rsu_grant_growth_percent','rsu_net_percent'] as const
    if (s.assets.length < 1 || s.assets.length > 12 || keys.some(k => typeof s[k] !== 'number' || !Number.isFinite(s[k])) || typeof s.invest_surplus !== 'boolean') return null
    if (s.assets.some(a => !a || typeof a.name !== 'string' || !['INR', 'USD'].includes(a.currency) || !['MARKET','EMPLOYER','PF','NPS','OTHER'].includes(a.kind) || [a.balance,a.monthly,a.monthly_increase,a.return_percent,a.access_age,a.accessible_percent].some(v => typeof v !== 'number' || !Number.isFinite(v)))) return null
    return s
  } catch { return null }
}

export function fireAge(age: number) {
  const months = Math.round(age * 12)
  return `${Math.floor(months / 12)} years${months % 12 ? ` ${months % 12} months` : ''}`
}

export function fireChartPoints(result: FireResult, real: boolean) {
  return result.points.map(p => ({ ...p,
    portfolio: real ? p.today_value : p.portfolio,
    accessible: real ? (p.portfolio > 0 ? p.accessible * p.today_value / p.portfolio : 0) : p.accessible,
    // This is the initial retirement threshold, not a fresh target every year
    // after retirement. Ending it here keeps the withdrawal trajectory readable.
    target: p.retired && result.retirement_age != null && p.age > result.retirement_age ? null : real ? result.today_target : p.target,
  }))
}
