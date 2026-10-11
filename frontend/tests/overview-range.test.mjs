import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/lib/overview-range.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { overviewRange } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('rolling overview ranges cross years and end on local today', () => {
  const now = new Date(2026, 9, 11)
  for (const [period, start] of [['1', '2026-09-11'], ['3', '2026-07-11'], ['6', '2026-04-11'], ['12', '2025-10-11']]) {
    assert.deepEqual(overviewRange(period, now), { start_date: start, end_date: '2026-10-11' })
  }
  assert.deepEqual(overviewRange('all', now), {})
  assert.deepEqual(overviewRange('custom', now), {})
})

test('month end and leap year ranges clamp to valid calendar dates', () => {
  assert.deepEqual(overviewRange('1', new Date(2024, 2, 31)), { start_date: '2024-02-29', end_date: '2024-03-31' })
  assert.deepEqual(overviewRange('12', new Date(2024, 1, 29)), { start_date: '2023-02-28', end_date: '2024-02-29' })
  assert.deepEqual(overviewRange('3', new Date(2026, 0, 31)), { start_date: '2025-10-31', end_date: '2026-01-31' })
})
