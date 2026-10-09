import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

// Use the existing TypeScript compiler with Node's test runner; no browser is needed.
const source = await readFile(new URL('../src/lib/api.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { fetchTransactions, fetchAllTransactions } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)

test('transaction requests and complete calendar datasets', async (t) => {
  t.mock.method(globalThis, 'fetch')
  for (const key of ['sessionStorage', 'localStorage']) {
    const original = Object.getOwnPropertyDescriptor(globalThis, key)
    Object.defineProperty(globalThis, key, { configurable: true, value: { getItem: () => null } })
    t.after(() => {
      if (original) Object.defineProperty(globalThis, key, original)
      else delete globalThis[key]
    })
  }

  await t.test('sends both amount bounds, including zero, with other filters', async () => {
    fetch.mock.mockImplementation(async (url) => {
      const params = new URL(url, 'http://localhost').searchParams
      assert.equal(params.get('min_amount'), '0')
      assert.equal(params.get('max_amount'), '200')
      assert.equal(params.get('account_id'), 'account-1')
      assert.equal(params.get('page'), '1')
      return Response.json({ items: [], total: 0, total_pages: 0, page_size: 50 })
    })
    await fetchTransactions({ min_amount: '0', max_amount: '200', account_id: 'account-1', page: 1 })
  })

  await t.test('loads rows beyond the first 5,000 with the same month filters', async () => {
    const requestedPages = []
    fetch.mock.mockImplementation(async (url) => {
      const params = new URL(url, 'http://localhost').searchParams
      const page = Number(params.get('page'))
      requestedPages.push(page)
      assert.equal(params.get('start_date'), '2026-08-01')
      assert.equal(params.get('end_date'), '2026-08-31')
      assert.equal(params.get('page_size'), '5000')
      return Response.json({
        items: page === 1 ? Array.from({ length: 5000 }, (_, id) => ({ id })) : [{ id: 5000 }],
        total: 5001, total_pages: 2, page_size: 5000,
      })
    })
    const result = await fetchAllTransactions({ start_date: '2026-08-01', end_date: '2026-08-31' })
    assert.deepEqual(requestedPages, [1, 2])
    assert.equal(result.length, 5001)
    assert.equal(result.at(-1).id, 5000)
  })

  await t.test('empty result stops after one request', async () => {
    let calls = 0
    fetch.mock.mockImplementation(async () => {
      calls += 1
      return Response.json({ items: [], total: 0, total_pages: 0, page_size: 5000 })
    })
    assert.deepEqual(await fetchAllTransactions(), [])
    assert.equal(calls, 1)
  })

  await t.test('a later page failure rejects the incomplete dataset', async () => {
    fetch.mock.mockImplementation(async (url) => {
      const page = new URL(url, 'http://localhost').searchParams.get('page')
      return page === '1'
        ? Response.json({ items: [{ id: 1 }], total: 5001, total_pages: 2, page_size: 5000 })
        : new Response('Error', { status: 500 })
    })
    await assert.rejects(fetchAllTransactions(), /Failed to fetch transactions/)
  })
})
