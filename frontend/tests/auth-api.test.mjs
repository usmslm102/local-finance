import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/lib/api.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { changeAuthPassword, fetchAccounts, previewInvestment, deleteInvestment } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)

test('password change replaces stored credentials before the next API call', async (t) => {
  for (const key of ['sessionStorage', 'localStorage']) {
    const previous = Object.getOwnPropertyDescriptor(globalThis, key)
    const values = new Map([['local_finance_token', 'old-token']])
    Object.defineProperty(globalThis, key, { configurable: true, value: {
      getItem: (key) => values.get(key) ?? null,
      setItem: (key, value) => values.set(key, value),
    } })
    t.after(() => previous ? Object.defineProperty(globalThis, key, previous) : delete globalThis[key])
  }
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    if (url.endsWith('/auth/change-password')) {
      assert.equal(options.headers.get('Authorization'), 'Bearer old-token')
      assert.equal(options.headers.get('X-LocalFinance-Request'), '1')
      return Response.json({ message: 'Updated', token: 'new-token' })
    }
    assert.equal(options.headers.get('Authorization'), 'Bearer new-token')
    assert.equal(options.headers.get('X-LocalFinance-Request'), '1')
    return Response.json([])
  })
  await changeAuthPassword('old-password', 'new-password')
  assert.equal(sessionStorage.getItem('local_finance_token'), 'new-token')
  assert.equal(localStorage.getItem('local_finance_token'), 'new-token')
  await fetchAccounts()
})

test('multipart uploads and bodyless mutations send the local request header', async (t) => {
  for (const key of ['sessionStorage', 'localStorage']) {
    const previous = Object.getOwnPropertyDescriptor(globalThis, key)
    Object.defineProperty(globalThis, key, { configurable: true, value: { getItem: () => null } })
    t.after(() => previous ? Object.defineProperty(globalThis, key, previous) : delete globalThis[key])
  }
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    assert.equal(options.headers.get('X-LocalFinance-Request'), '1')
    if (url.endsWith('/preview')) {
      assert.equal(options.method, 'POST')
      assert.ok(options.body instanceof FormData)
      assert.equal(options.headers.has('Content-Type'), false)
    } else {
      assert.equal(options.method, 'DELETE')
    }
    return Response.json({})
  })
  await previewInvestment(new Blob(['fictional statement']))
  await deleteInvestment('fictional-id')
})
