import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
const source = await readFile(
  new URL('../src/lib/import-calendar.ts', import.meta.url),
  'utf8',
)
const { outputText } = ts.transpileModule(source, {
  compilerOptions: {
    target: ts.ScriptTarget.ES2022,
    module: ts.ModuleKind.ES2022,
  },
})
const { sourceMonth, calendarSummary } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)
const bank = { id: 'bank', account_type: 'SAVINGS' }
const card = { id: 'card', account_type: 'CREDIT_CARD' }
const prefs = () => ({ sources: {}, importMonths: {} })
const statement = (id, account_id, start_date, end_date) => ({
  id,
  account_id,
  start_date,
  end_date,
  imported_at: '2026-10-01',
})
const state = (
  account,
  files,
  month,
  preferences = prefs(),
  bills = [],
  today = '2026-10-11',
) => sourceMonth(account, files, bills, preferences, month, today)

test('a cross-month credit-card cycle satisfies exactly its statement month', () => {
  const files = [statement('c', 'card', '2026-02-19', '2026-03-18')]
  assert.equal(state(card, files, '2026-03').status, 'imported')
  assert.equal(state(card, files, '2026-02').status, 'not-tracked')
  assert.equal(state(card, files, '2026-04').status, 'missing')
  const bills = [{ statement_import_id: 'c', statement_date: '2026-04-01' }]
  assert.equal(state(card, files, '2026-03', prefs(), bills).imports.length, 0)
  assert.equal(state(card, files, '2026-04', prefs(), bills).status, 'imported')
})
test('bank ranges satisfy multiple months and overlapping ranges merge without hiding gaps', () => {
  const files = [statement('a', 'bank', '2024-01-01', '2024-03-31')]
  for (const month of ['2024-01', '2024-02', '2024-03'])
    assert.equal(state(bank, files, month).status, 'imported')
  const partial = [
    statement('a', 'bank', '2024-02-01', '2024-02-15'),
    statement('b', 'bank', '2024-02-17', '2024-02-29'),
  ]
  assert.equal(state(bank, partial, '2024-02').status, 'partial')
  assert.equal(
    state(
      bank,
      [...partial, statement('c', 'bank', '2024-02-14', '2024-02-18')],
      '2024-02',
    ).status,
    'imported',
  )
})
test('missing periods remain visible between imports and other sources do not fill them', () => {
  const files = [
    statement('a', 'bank', '2026-01-01', '2026-01-31'),
    statement('b', 'bank', '2026-03-01', '2026-03-31'),
    statement('c', 'card', '2026-02-01', '2026-02-28'),
  ]
  assert.equal(state(bank, files, '2026-02').status, 'missing')
  assert.equal(state(bank, files, '2026-02').imports.length, 0)
})
test('tracking dates and no activity are explicit and imports override old no-activity flags', () => {
  const p = {
    sources: {
      bank: { from: '2026-02', until: '2026-07', noActivity: ['2026-03'] },
    },
    importMonths: {},
  }
  assert.equal(state(bank, [], '2026-01', p).status, 'not-tracked')
  assert.equal(state(bank, [], '2026-02', p).status, 'missing')
  assert.equal(state(bank, [], '2026-03', p).status, 'no-activity')
  assert.equal(state(bank, [], '2026-08', p).status, 'not-tracked')
  assert.equal(
    state(
      bank,
      [statement('a', 'bank', '2026-03-01', '2026-03-31')],
      '2026-03',
      p,
    ).status,
    'imported',
  )
})
test('invalid and inverted dates need confirmation; upload time does not establish a month', () => {
  const files = [
    statement('a', 'bank', '2026-02-30', '2026-03-01'),
    statement('b', 'bank', '2026-04-01', '2026-03-01'),
  ]
  const result = state(bank, files, '2026-03')
  assert.equal(result.status, 'unknown')
  assert.equal(result.unknown.length, 2)
  assert.equal(result.imports.length, 0)
  const p = prefs()
  p.importMonths.a = '2026-03'
  assert.equal(state(bank, files, '2026-03', p).status, 'imported')
  assert.equal(state(bank, files, '2026-04', p).status, 'unknown')
})
test('current/future months are not overdue; known closed card cycles can be missing', () => {
  const files = [statement('a', 'card', '2026-01-01', '2026-01-31')]
  assert.equal(state(card, files, '2026-10').status, 'pending')
  assert.equal(
    state({ ...card, billing_cycle_day: 5 }, files, '2026-10').status,
    'missing',
  )
  assert.equal(
    state({ ...card, billing_cycle_day: 15 }, files, '2026-10').status,
    'pending',
  )
  assert.equal(state(card, files, '2026-11').status, 'upcoming')
})
test('all-source summaries retain missing, partial and unknown sources alongside imports', () => {
  const summary = (...statuses) =>
    calendarSummary(statuses.map((status) => ({ status })))
  assert.equal(
    summary('imported', 'missing', 'partial'),
    '1 missing · 1 partial',
  )
  assert.equal(summary('imported', 'unknown'), 'Check dates')
  assert.equal(summary('imported', 'pending'), 'In progress')
  assert.equal(
    summary('imported', 'no-activity', 'not-tracked'),
    'All imported',
  )
  assert.equal(summary('upcoming', 'upcoming'), 'Upcoming')
})

test('undated files make unfilled tracked months uncertain without hiding known imports or tracking bounds', () => {
  const files = [
    statement('dated', 'bank', '2026-01-01', '2026-01-31'),
    statement('undated', 'bank'),
  ]
  assert.equal(state(bank, files, '2026-01').status, 'imported')
  assert.equal(state(bank, files, '2026-02').status, 'unknown')
  assert.equal(state(bank, files, '2025-12').status, 'not-tracked')
  const p = prefs()
  p.sources.bank = { until: '2026-05' }
  assert.equal(state(bank, files, '2026-06', p).status, 'not-tracked')
  p.importMonths.undated = '2026-03'
  assert.equal(state(bank, files, '2026-02', p).status, 'missing')
})
test('empty sources ask for a tracking start instead of dates that do not exist', () => {
  assert.equal(state(bank, [], '2026-02').status, 'unconfigured')
  assert.equal(
    calendarSummary([{ status: 'imported' }, { status: 'unconfigured' }]),
    'Set tracking start',
  )
})
test('card grace period is three calendar days and carries across month and year boundaries', () => {
  const files = [statement('dated', 'card', '2025-01-01', '2025-01-31')]
  const closing15 = { ...card, billing_cycle_day: 15 }
  assert.equal(
    state(closing15, files, '2026-10', prefs(), [], '2026-10-18').status,
    'pending',
  )
  assert.equal(
    state(closing15, files, '2026-10', prefs(), [], '2026-10-19').status,
    'missing',
  )
  const closing31 = { ...card, billing_cycle_day: 31 }
  assert.equal(
    state(closing31, files, '2026-02', prefs(), [], '2026-03-03').status,
    'pending',
  )
  assert.equal(
    state(closing31, files, '2026-02', prefs(), [], '2026-03-04').status,
    'missing',
  )
  assert.equal(
    state(closing31, files, '2025-12', prefs(), [], '2026-01-02').status,
    'pending',
  )
})
