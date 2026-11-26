import type { FireResult, FireScenario } from '@/types/fire'

export function fireNeedsFX(s: FireScenario) {
  const firstMarket = s.assets.find(a => !['PF', 'NPS', 'EMPLOYER'].includes(a.kind))
  return s.rsu_unvested_usd > 0 || s.rsu_annual_usd > 0 ||
    s.assets.some(a => a.currency === 'USD' && (a.balance > 0 || a.monthly > 0 || a.monthly_increase > 0)) ||
    (s.invest_surplus && firstMarket?.currency === 'USD')
}

export function fireInputNeeds(s: FireScenario) {
  const needs: string[] = []
  if (s.age < 18) needs.push('your age')
  if (!(s.monthly_expense > 0)) needs.push('current living costs')
  if (!(s.retirement_expense > 0)) needs.push('retirement living costs')
  if (fireNeedsFX(s) && !(s.usd_inr > 0)) needs.push('an INR per USD rate')
  return needs
}

export function fireStartingValues(s: FireScenario) {
  if (fireNeedsFX(s) && !(s.usd_inr > 0)) return null
  let total = 0, accessible = 0
  for (const a of s.assets) {
    const value = a.balance * (a.currency === 'USD' ? s.usd_inr : 1)
    if (!Number.isFinite(value)) return null
    total += value
    if (a.access_age <= s.age) accessible += value * a.accessible_percent / 100
  }
  return { total, accessible }
}

// Compose a single engine point in one money basis. Do not combine current
// assets, future assets and today's spending to imply a readiness probability.
export function fireAssetBreakdown(result: FireResult, real: boolean) {
  const point = result.at_retirement ?? result.points[0]
  if (!point) return null
  const factor = real && point.portfolio > 0 ? point.today_value / point.portfolio : 1
  const total = real ? point.today_value : point.portfolio
  const accessible = point.accessible * factor
  return { total, accessible, locked: total - accessible, atRetirement: !!result.at_retirement }
}

export function fireTargetProgress(accessible: number, target: number) {
  if (!Number.isFinite(accessible) || !Number.isFinite(target) || target <= 0) return {meter: 0, label: 'Unavailable'}
  const ratio = Math.max(0, accessible / target * 100)
  return {meter: Math.min(100, ratio), label: Number.isFinite(ratio) ? `${Math.floor(ratio).toLocaleString('en-IN')}%` : '100%+'}
}
