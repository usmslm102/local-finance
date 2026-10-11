import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
async function moduleUrl(path) {
  const source = await readFile(new URL(path, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.ES2022,
    },
  })
  return `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
}
const dependency = await moduleUrl('../src/lib/import-calendar.ts')
const raw = await readFile(
  new URL('../src/lib/calendar-preferences.ts', import.meta.url),
  'utf8',
)
const { outputText } = ts.transpileModule(
  raw.replace('@/lib/import-calendar', dependency),
  {
    compilerOptions: {
      target: ts.ScriptTarget.ES2022,
      module: ts.ModuleKind.ES2022,
    },
  },
)
const { parsePreferences, prunePreferences, withTracking } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)

test('preferences sanitize invalid shapes, inverted bounds and duplicated no-activity months', () => {
  for (const raw of [
    null,
    'not json',
    '[]',
    '{"sources":null}',
    '{"sources":[1]}',
  ])
    assert.deepEqual(parsePreferences(raw), { sources: {}, importMonths: {} })
  const result = parsePreferences(
    JSON.stringify({
      sources: {
        a: {
          from: '2026-03',
          until: '2026-01',
          noActivity: ['2026-02', '2026-02', 'invalid', null],
        },
        b: true,
      },
      importMonths: { valid: '2026-02', invalid: '2026-13', numeric: 202601 },
    }),
  )
  assert.deepEqual(result.sources.a, {
    from: '2026-03',
    until: undefined,
    noActivity: ['2026-02'],
  })
  assert.deepEqual(result.importMonths, { valid: '2026-02' })
  assert.equal(Object.hasOwn(result.sources, 'b'), false)
})
test('deleting a source or statement removes only its orphaned preferences', () => {
  const preferences = {
    sources: { active: { from: '2026-01' }, deleted: { from: '2020-01' } },
    importMonths: { kept: '2026-02', removed: '2026-03' },
  }
  assert.deepEqual(prunePreferences(preferences, ['active'], ['kept']), {
    sources: { active: { from: '2026-01' } },
    importMonths: { kept: '2026-02' },
  })
  assert.deepEqual(prunePreferences(preferences, [], []), {
    sources: {},
    importMonths: {},
  })
  assert.equal(Object.hasOwn(preferences.sources, 'deleted'), true)
})
test('updating tracking preserves unrelated manual assignments and no-activity confirmations', () => {
  const original = {
    sources: {
      a: { from: '2026-01', noActivity: ['2026-02'] },
      b: { from: '2025-01' },
    },
    importMonths: { file: '2026-03' },
  }
  const changed = withTracking(original, 'a', { until: '2026-06' })
  assert.deepEqual(changed.sources.a, {
    from: '2026-01',
    until: '2026-06',
    noActivity: ['2026-02'],
  })
  assert.deepEqual(changed.sources.b, original.sources.b)
  assert.deepEqual(changed.importMonths, original.importMonths)
  assert.equal(original.sources.a.until, undefined)
})
