import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

async function load(name) {
  const source = await readFile(new URL(`../src/lib/${name}.ts`, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
  })
  return import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
}
const { layoutCashFlow, flowPath, allocateFlowHeights } = await load('sankey-layout')
const { cashFlowNodeParams, contributingTransactions, cashFlowPeriodDates } =
  await load('cashflow-drilldown')
const node = (id, layer, value, type = 'CATEGORY') => ({
  id,
  name: id,
  layer,
  total_value: value,
  color_hex: '#008866',
  type,
})
const data = (nodes, links) => ({
  nodes,
  links,
  total_inflow: 0,
  total_outflow: 0,
  net_surplus: 0,
  savings_rate: 0,
})

test('dominant inflow and many small nodes fit within the canvas with readable labels', () => {
  const heights = allocateFlowHeights([1000000, ...Array(29).fill(1)], 30 * 64 + 100)
  assert.ok(heights.every((height) => height >= 64))
  assert.ok(Math.abs(heights.reduce((a, b) => a + b, 0) - 2020) < 0.001)
  const fixture = data(
    [
      node('income', 0, 1000000, 'INCOME_SOURCE'),
      ...Array.from({ length: 30 }, (_, index) =>
        node(`cat_${index}`, 3, index === 0 ? 999971 : 1)
      ),
    ],
    Array.from({ length: 30 }, (_, index) => ({
      source: 'income',
      target: `cat_${index}`,
      value: index === 0 ? 999971 : 1,
    }))
  )
  for (const expanded of [false, true]) {
    const layout = layoutCashFlow(fixture, expanded)
    for (const n of layout.nodes) assert.ok(n.y >= 0 && n.y + n.height <= layout.height)
    for (const layer of [0, 3]) {
      const column = layout.nodes.filter((n) => n.layer === layer)
      for (let index = 1; index < column.length; index++)
        assert.ok(column[index].y >= column[index - 1].y + column[index - 1].height)
    }
    for (const link of layout.links) {
      assert.ok(link.sourceY + link.sourceHeight <= link.source.y + link.source.height + 0.001)
      assert.ok(link.targetY + link.targetHeight <= link.target.y + link.target.height + 0.001)
      assert.ok(!/NaN|Infinity/.test(flowPath(link)))
    }
  }
})

test('compact groups preserve node and ribbon totals, keep savings visible, and expand to every node', () => {
  const nodes = [
    node('income', 0, 11000, 'INCOME_SOURCE'),
    ...Array.from({ length: 20 }, (_, index) => node(`cat_${index}`, 3, 500)),
    node('savings', 3, 1000, 'SURPLUS'),
  ]
  const fixture = data(
    nodes,
    nodes.slice(1).map((n) => ({ source: 'income', target: n.id, value: n.total_value }))
  )
  const compact = layoutCashFlow(fixture)
  assert.equal(compact.nodes.filter((n) => n.layer === 3).length, 7)
  assert.equal(
    compact.nodes.filter((n) => n.layer === 3).reduce((sum, n) => sum + n.total_value, 0),
    11000
  )
  assert.equal(
    compact.links.reduce((sum, link) => sum + link.value, 0),
    11000
  )
  assert.ok(compact.nodes.some((n) => n.id === 'savings' && !n.members))
  assert.equal(layoutCashFlow(fixture, true).nodes.length, nodes.length)
  assert.deepEqual(layoutCashFlow(data([...nodes].reverse(), fixture.links)), compact)
})

test('inconsistent routing totals and tiny ribbons cannot overflow their nodes', () => {
  const fixture = data(
    [
      node('bank', 1, 100, 'ACCOUNT'),
      ...Array.from({ length: 80 }, (_, i) => node(`card_${i}`, 2, 2, 'CHANNEL')),
    ],
    Array.from({ length: 80 }, (_, i) => ({ source: 'bank', target: `card_${i}`, value: 2 }))
  )
  const layout = layoutCashFlow(fixture, true)
  assert.ok(
    layout.links.every(
      (link) => link.sourceY + link.sourceHeight <= link.source.y + link.source.height + 0.0001
    )
  )
})

test('empty, zero and invalid flows have finite layouts', () => {
  for (const fixture of [data([], []), data([node('zero', 0, 0), node('nan', 1, NaN)], [])]) {
    const layout = layoutCashFlow(fixture)
    assert.equal(layout.nodes.length, 0)
    assert.ok(Number.isFinite(layout.height))
  }
})

test('drilldowns retain the full category ID and the selected account or card', () => {
  assert.deepEqual(cashFlowNodeParams(node('cat_cat_food', 3, 10)), {
    category_id: 'cat_food',
    tx_type: 'DEBIT',
  })
  assert.deepEqual(cashFlowNodeParams(node('acc_test-account', 1, 10, 'ACCOUNT')), {
    account_id: 'test-account',
  })
  assert.deepEqual(cashFlowNodeParams(node('chan_card_test-card', 2, 10, 'CHANNEL')), {
    account_id: 'test-card',
    tx_type: 'DEBIT',
  })
})

test('income sources and financial exclusions match the overview rules', () => {
  const transactions = [
    { id: 'salary', tx_type: 'CREDIT', raw_narration: 'Payroll', account_id: 'a' },
    { id: 'refund', tx_type: 'CREDIT', raw_narration: 'Refund', account_id: 'a' },
    { id: 'interest', tx_type: 'CREDIT', raw_narration: 'Interest credit', account_id: 'a' },
    { id: 'other', tx_type: 'CREDIT', raw_narration: 'UPI credit', account_id: 'a' },
    { id: 'transfer', tx_type: 'CREDIT', raw_narration: 'Payroll', is_transfer: true },
    { id: 'excluded', tx_type: 'CREDIT', raw_narration: 'Payroll', is_excluded: true },
    {
      id: 'category-transfer',
      tx_type: 'CREDIT',
      raw_narration: 'Payroll',
      category_id: 'cat_transfers',
    },
    { id: 'debit', tx_type: 'DEBIT', raw_narration: 'Payroll' },
  ]
  for (const [source, id] of [
    ['salary', 'salary'],
    ['refund', 'refund'],
    ['invest', 'interest'],
    ['other', 'other'],
  ]) {
    assert.deepEqual(
      contributingTransactions(node(`src_${source}`, 0, 1, 'INCOME_SOURCE'), transactions).map(
        (tx) => tx.id
      ),
      [id]
    )
  }
  assert.deepEqual(contributingTransactions(node('surplus', 3, 1, 'SURPLUS'), transactions), [])
})

test('month and rolling drilldowns use real month ends, including leap years and dataset gaps', () => {
  assert.deepEqual(cashFlowPeriodDates('2024-02', []), {
    start_date: '2024-02-01',
    end_date: '2024-02-29',
  })
  assert.deepEqual(cashFlowPeriodDates('2025-02', []), {
    start_date: '2025-02-01',
    end_date: '2025-02-28',
  })
  assert.deepEqual(cashFlowPeriodDates('LAST_3_MONTHS', ['2026-10', '2026-08', '2026-04']), {
    start_date: '2026-04-01',
    end_date: '2026-10-31',
  })
  assert.deepEqual(cashFlowPeriodDates('LAST_3_MONTHS', ['2026-10']), {})
  assert.deepEqual(cashFlowPeriodDates('ALL', []), {})
  assert.deepEqual(cashFlowPeriodDates('FY-2025-26', []), {
    start_date: '2025-04-01',
    end_date: '2026-03-31',
  })
})

test('all four layers fit a narrow desktop without shrinking node text', () => {
  const fixture = data(
    [
      node('income', 0, 100, 'INCOME_SOURCE'),
      node('bank', 1, 100, 'ACCOUNT'),
      node('card', 2, 80, 'CHANNEL'),
      node('category', 3, 80),
    ],
    []
  )
  const layout = layoutCashFlow(fixture, false, 872)
  assert.equal(layout.width, 872)
  for (const node of layout.nodes) {
    assert.ok(node.width >= 150)
    assert.ok(node.x + node.width <= layout.width)
  }
})
