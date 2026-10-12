import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const compile = source => `data:text/javascript;base64,${Buffer.from(ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
}).outputText).toString('base64')}`
const react = compile(`
export const useCallback = fn => fn;
export const useSyncExternalStore = () => globalThis.updateHarness.autoCheck;
export const useState = initial => {
  const h = globalThis.updateHarness;
  const slot = h.cursor++;
  if (!(slot in h.states)) h.states[slot] = initial;
  return [h.states[slot], value => h.states[slot] = value];
};`)
const query = compile(`
export const useQuery = () => ({ data: globalThis.updateHarness.data, error: globalThis.updateHarness.error });
export const useQueryClient = () => ({ setQueryData: (_, data) => globalThis.updateHarness.data = data });`)
const api = compile(`export const fetchSystemVersion = async () => globalThis.updateHarness.response;`)
const source = (await readFile(new URL('../src/hooks/use-system-update.ts', import.meta.url), 'utf8'))
  .replace("from 'react'", `from '${react}'`)
  .replace("from '@tanstack/react-query'", `from '${query}'`)
  .replace("from '@/lib/api'", `from '${api}'`)
const { useSystemUpdate } = await import(compile(source))
const desktop = {
  current_version: 'v1.13.0', latest_version: 'v1.13.0', update_available: false,
  can_auto_update: false, auto_update_error: 'Download the desktop installer.',
  checked_at: '2026-10-12T06:00:00Z',
}
function setup(t, data, autoCheck = true, error = null) {
  globalThis.updateHarness = { data, response: data, autoCheck, error, states: [], cursor: 0 }
  t.after(() => { delete globalThis.updateHarness })
  return () => {
    updateHarness.cursor = 0
    return useSystemUpdate()
  }
}

test('desktop installer guidance is not a failed manual release check', async t => {
  const render = setup(t, desktop)
  await render().checkNow()
  assert.equal(render().checkError, null)
  assert.equal(render().versionInfo.checked_at, desktop.checked_at)
})
test('desktop offline metadata does not report an update failure', t => {
  const render = setup(t, { ...desktop, latest_version: '', checked_at: '' }, false)
  assert.equal(render().checkError, null)
  assert.equal(render().autoCheckEnabled, false)
})
test('successful portable check ignores executable installation restrictions', async t => {
  const render = setup(t, { ...desktop, auto_update_error: 'Executable is not writable' })
  await render().checkNow()
  assert.equal(render().checkError, null)
})
test('available desktop updates still open the installer dialog', async t => {
  const render = setup(t, { ...desktop, latest_version: 'v1.14.0', update_available: true })
  await render().checkNow()
  assert.equal(render().dialogOpen, true)
  assert.equal(render().checkError, null)
})
test('actual API release-check failures are visible for background and manual checks', async t => {
  const render = setup(t, { ...desktop, checked_at: '', update_check_error: 'GitHub API unavailable' })
  assert.equal(render().checkError, 'GitHub API unavailable')
  await render().checkNow()
  assert.equal(render().checkError, 'GitHub API unavailable')
})
test('background transport failures are visible', t => {
  const render = setup(t, undefined, true, new Error('Local server unavailable'))
  assert.equal(render().checkError, 'Local server unavailable')
})

test('a successful retry clears a previous release-check error', async t => {
  const render = setup(t, { ...desktop, update_check_error: 'Temporary failure' })
  await render().checkNow()
  assert.equal(render().checkError, 'Temporary failure')
  updateHarness.response = desktop
  await render().checkNow()
  assert.equal(render().checkError, null)
})
