import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'

const compile = source => `data:text/javascript;base64,${Buffer.from(ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
}).outputText).toString('base64')}`
const apiURL = compile(await readFile(new URL('../src/lib/api.ts', import.meta.url), 'utf8'))
const desktopSource = (await readFile(new URL('../src/lib/desktop.ts', import.meta.url), 'utf8'))
  .replace("from './api'", `from '${apiURL}'`)
const { installDesktopActions } = await import(compile(desktopSource))

test('desktop preserves browser behavior and sends native actions through UI auth', async t => {
  const values = {
    localStorage: { getItem: () => 'existing-session' },
    sessionStorage: { getItem: () => null },
    window: { open: () => 'browser-window', location: new URL('http://127.0.0.1:43210'), alert: () => {}, dispatchEvent: () => {} },
    document: { addEventListener: () => {} },
    Element: class { constructor(anchor) { this.anchor = anchor } closest() { return this.anchor } },
  }
  for (const [key, value] of Object.entries(values)) {
    const original = Object.getOwnPropertyDescriptor(globalThis, key)
    Object.defineProperty(globalThis, key, { configurable: true, value })
    t.after(() => { if (original) Object.defineProperty(globalThis, key, original); else delete globalThis[key] })
  }
  t.mock.method(globalThis, 'fetch')
  t.mock.method(document, 'addEventListener')
  const browserOpen = window.open
  fetch.mock.mockImplementation(async () => Response.json({ enabled: false }))
  await installDesktopActions()
  assert.equal(window.open, browserOpen)
  assert.equal(document.addEventListener.mock.callCount(), 0)

  const requests = []
  fetch.mock.mockImplementation(async (url, options) => {
    requests.push({ url, options })
    assert.equal(options.headers.get('X-LocalFinance-Request'), '1')
    assert.equal(options.headers.get('Authorization'), 'Bearer existing-session')
    if (url === '/api/desktop') return Response.json({ enabled: true })
    if (url.endsWith('/api/database/backup')) {
      return new Response('sqlite-snapshot', { headers: { 'Content-Disposition': 'attachment; filename="ledger.db"' } })
    }
    return new Response(null, { status: 204 })
  })
  await installDesktopActions()
  assert.equal(window.open('https://example.com/releases'), null)
  await new Promise(setImmediate)
  const open = requests.find(request => request.url === '/api/desktop/open-url')
  assert.deepEqual(JSON.parse(open.options.body), { url: 'https://example.com/releases' })

  const listener = document.addEventListener.mock.calls[0].arguments[1]
  let prevented = false
  const anchor = { href: 'http://127.0.0.1:43210/api/database/backup', download: '', hasAttribute: name => name === 'download' }
  listener({ target: new Element(anchor), preventDefault() { prevented = true } })
  await new Promise(setImmediate)
  assert.equal(prevented, true)
  const save = requests.find(request => request.url.startsWith('/api/desktop/save'))
  assert.equal(save.url, '/api/desktop/save?name=ledger.db')
  assert.equal(await save.options.body.text(), 'sqlite-snapshot')

  const count = requests.length
  const external = { ...anchor, href: 'https://example.com/download.csv' }
  listener({ target: new Element(external), preventDefault() {} })
  await new Promise(setImmediate)
  assert.equal(requests.length, count, 'must not send UI credentials to external downloads')
})
