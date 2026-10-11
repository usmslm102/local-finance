import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/lib/planning.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { statementCoverage, forecastBaseline, forecastScenario, goalProgress, goalReservations } =
  await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const statement = (start_date, end_date, account_id = 'a') => ({ account_id, start_date, end_date })

test('coverage merges overlaps, retains gaps and clips to elapsed days', () => {
  const result = statementCoverage(
    [
      statement('2024-02-01', '2024-02-04'),
      statement('2024-02-03', '2024-02-06'),
      statement('2024-02-10', '2024-03-03'),
      statement('2024-02-01', '2024-02-29', 'b'),
    ],
    'a',
    '2024-02',
    '2024-02-14'
  )
  assert.equal(result.covered, 11)
  assert.equal(result.total, 14)
  assert.deepEqual(result.gaps, [{ start: '2024-02-07', end: '2024-02-09' }])
})
test('coverage ignores invalid/inverted ranges and counts leap days without guessing from activity', () => {
  const result = statementCoverage(
    [statement('2024-02-30', '2024-03-03'), statement('2024-02-20', '2024-02-01')],
    'a',
    '2024-02',
    '2024-03-02'
  )
  assert.equal(result.covered, 0)
  assert.equal(result.total, 29)
  assert.deepEqual(result.gaps, [{ start: '2024-02-01', end: '2024-02-29' }])
  assert.equal(statementCoverage([], 'a', '2024-04', '2024-03-02').total, 0)
})
test('baseline excludes current/future months and exposes gaps rather than selecting older observations', () => {
  const result = forecastBaseline(
    [
      { month: '2026-10', income: 100000, expense: 10 },
      { month: '2026-08', income: 100, expense: 50 },
      { month: '2026-07', income: 200, expense: 100 },
      { month: '2026-04', income: 900000, expense: 1 },
    ],
    '2026-10-11'
  )
  assert.equal(result.count, 2)
  assert.equal(result.income, 150)
  assert.equal(result.expense, 75)
  assert.deepEqual(
    result.months.map((m) => m.month),
    ['2026-07', '2026-08', '2026-09']
  )
  assert.equal(result.months[2].recorded, undefined)
})
test('forecast reserves a one-off only once and keeps goals separate from money held', () => {
  const rows = forecastScenario(
    { opening: 1000, income: 200, expense: 100, oneOff: 500, goalContribution: 50, months: 3 },
    '2026-11-30'
  )
  assert.deepEqual(
    rows.map((r) => r.balance),
    [1000, 600, 700, 800]
  )
  assert.deepEqual(
    rows.map((r) => r.available),
    [1000, 550, 600, 650]
  )
  assert.deepEqual(
    rows.map((r) => r.month),
    ['2026-11', '2026-12', '2027-01', '2027-02']
  )
})
test('forecast uses cents and allows negative opening values and shortfalls', () => {
  const rows = forecastScenario(
    { opening: -0.01, income: 0.2, expense: 0.1, oneOff: 0, goalContribution: 0, months: 3 },
    '2026-01-31'
  )
  assert.equal(rows[3].balance, 0.29)
  assert.equal(rows[0].available, -0.01)
})
const goal = {
  target_amount: 1000,
  saved_amount: 400,
  monthly_contribution: 100,
  target_date: '2026-12-01',
}
test('goal pacing includes this month, handles past deadlines and already reached targets', () => {
  const p = goalProgress(goal, '2026-10-11')
  assert.equal(p.requiredMonthly, 200)
  assert.equal(p.monthsLeft, 3)
  assert.equal(p.onTrack, false)
  assert.equal(goalProgress(goal, '2027-01-01').overdue, true)
  assert.equal(goalProgress({ ...goal, saved_amount: 1200 }, '2027-01-01').completed, true)
  assert.equal(goalProgress({ ...goal, saved_amount: 1200 }, '2027-01-01').percent, 100)
})
test('goal reservations stop at the remaining target and exclude completed goals', () => {
  const reserves = goalReservations(
    [
      goal,
      { ...goal, saved_amount: 1000 },
      { ...goal, saved_amount: 900, monthly_contribution: 200 },
    ],
    8
  )
  assert.deepEqual(reserves, [0, 200, 300, 400, 500, 600, 700, 700, 700])
  const rows = forecastScenario(
    {
      opening: 1000,
      income: 0,
      expense: 0,
      oneOff: 0,
      goalContribution: 0,
      goalReservations: reserves,
      months: 8,
    },
    '2026-10-11'
  )
  assert.equal(rows[8].available, 300)
})
