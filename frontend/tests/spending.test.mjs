import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/lib/spending.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { validSpendingMonth, spendingMonthLabel, cashFlowCategories } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)

test('cash-flow category insights use the exact Sankey amounts and purchase counts', () => {
  const nodes = [
    { id: 'acc_bank', type: 'ACCOUNT', total_value: 900 },
    { id: 'cat_cat_food', type: 'CATEGORY', name: 'Food', total_value: 80, tx_count: 2, color_hex: '#123456' },
    { id: 'cat_uncat', type: 'CATEGORY', name: 'Uncategorized', total_value: 20, tx_count: 1 },
    { id: 'dest_surplus', type: 'SURPLUS', total_value: 800 },
  ]
  const categories = cashFlowCategories(nodes, 100)
  assert.deepEqual(categories.map((item) => [item.category_id, item.total_amount, item.tx_count, item.percentage]), [
    ['cat_food', 80, 2, 80], ['uncat', 20, 1, 20],
  ])
  assert.deepEqual(cashFlowCategories([], 0), [])
})

test('all-time and malformed URL months never produce invalid date requests', () => {
  for (const value of [undefined, null, '', 202608, '2026-00', '2026-13', '0000-01', '2026-8', '2026-08-01', '2026-08&category=food']) {
    assert.equal(validSpendingMonth(value), false)
    assert.equal(spendingMonthLabel(value), 'All time')
  }
})
