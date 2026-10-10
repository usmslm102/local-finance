import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/lib/api.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } })
const { splitwiseRequest, splitwiseUpload } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('Splitwise sends protected uploads and handles confirmation responses', async t => {
  for (const key of ['sessionStorage', 'localStorage']) {
    const original = Object.getOwnPropertyDescriptor(globalThis, key)
    Object.defineProperty(globalThis, key, { configurable: true, value: { getItem: () => 'example-token' } })
    t.after(() => { if (original) Object.defineProperty(globalThis, key, original); else delete globalThis[key] })
  }
  t.mock.method(globalThis, 'fetch', async (url, init) => {
    assert.equal(url, '/api/splitwise/preview')
    assert.equal(init.headers.get('X-LocalFinance-Request'), '1')
    assert.equal(init.headers.get('Authorization'), 'Bearer example-token')
    assert.equal(init.headers.has('Content-Type'), false)
    assert.equal(init.body.get('person'), 'Sanjay')
    assert.equal(init.body.get('group'), 'Example')
    return Response.json([{ id: 'fictional-entry' }])
  })
  assert.deepEqual(await splitwiseUpload('preview', new File(['fictional'], 'fictional.csv'), 'Example', 'Sanjay'), [{ id: 'fictional-entry' }])
  fetch.mock.mockImplementation(async (url, init) => {
    assert.equal(url, '/api/splitwise/example/confirm')
    assert.equal(init.method, 'POST')
    assert.deepEqual(JSON.parse(init.body), { share_cents: 5000 })
    return new Response(null, { status: 204 })
  })
  assert.equal(await splitwiseRequest('/example/confirm', { share_cents: 5000 }), undefined)
  fetch.mock.mockImplementation(async () => Response.json({ error: 'invalid share' }, { status: 400 }))
  await assert.rejects(splitwiseRequest('/example/confirm', { share_cents: -1 }), /invalid share/)
})
