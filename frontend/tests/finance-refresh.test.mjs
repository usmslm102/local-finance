import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
import { QueryClient, QueryObserver, MutationCache } from '@tanstack/react-query'
const source = await readFile(new URL('../src/lib/finance-refresh.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } })
const { refreshActiveFinance } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const tick = () => new Promise((resolve) => setImmediate(resolve))

test('initial in-flight reports cannot overwrite a revision refresh with pre-write data', async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  t.after(() => client.clear())
  let releaseOld
  let reads = 0
  const observer = new QueryObserver(client, { queryKey: ['overview'], queryFn: () => ++reads === 1 ? new Promise((resolve) => { releaseOld = resolve }) : Promise.resolve(2) })
  const unsubscribe = observer.subscribe(() => {})
  t.after(unsubscribe)
  await tick()
  await refreshActiveFinance(client)
  releaseOld(1)
  await tick()
  assert.equal(client.getQueryData(['overview']), 2)
  assert.equal(reads, 2)
})

test('a cancelled freshness effect does not start replacement requests', async (t) => {
  const client = new QueryClient()
  t.after(() => client.clear())
  let reads = 0
  const observer = new QueryObserver(client, { queryKey: ['overview'], queryFn: async () => ++reads })
  const unsubscribe = observer.subscribe(() => {})
  t.after(unsubscribe)
  await tick()
  await refreshActiveFinance(client, () => false)
  assert.equal(reads, 1)
})

test('read-only recommendations do not recursively invalidate themselves; mutations refresh once', async (t) => {
  const client = new QueryClient({ mutationCache: new MutationCache({ onSuccess: () => refreshActiveFinance(client) }) })
  t.after(() => client.clear())
  let reads = 0
  const observer = new QueryObserver(client, { queryKey: ['recommendation'], queryFn: async () => ++reads })
  const unsubscribe = observer.subscribe(() => {})
  t.after(unsubscribe)
  await tick()
  await client.invalidateQueries()
  await tick()
  assert.equal(reads, 2)
  await client.getMutationCache().build(client, { mutationFn: async () => true }).execute()
  await tick()
  assert.equal(reads, 3)
})

test('a successful local save cannot be overwritten by the inspector initial GET', async (t) => {
  const client = new QueryClient({ mutationCache: new MutationCache({ onSuccess: () => refreshActiveFinance(client) }) })
  t.after(() => client.clear())
  let releaseOld
  let reads = 0
  const key = ['transaction', 'fixture']
  const observer = new QueryObserver(client, { queryKey: key, queryFn: () => ++reads === 1 ? new Promise((resolve) => { releaseOld = resolve }) : Promise.resolve({ notes: 'saved' }) })
  const unsubscribe = observer.subscribe(() => {})
  t.after(unsubscribe)
  await tick()
  await client.getMutationCache().build(client, { mutationFn: async () => ({ notes: 'saved' }), onSuccess: (result) => client.setQueryData(key, result) }).execute()
  releaseOld({ notes: 'before' })
  await tick()
  assert.deepEqual(client.getQueryData(key), { notes: 'saved' })
  assert.equal(reads, 2)
})
