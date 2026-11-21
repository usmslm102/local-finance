import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/lib/statement-form.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, { compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 } })
const { statementForm } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)

test('mixed-account staging preserves each reviewed account and parser when defaults change', () => {
  const file = new File(['statement'], 'statement.csv')
  const defaults = { accountId: 'account-a', parserId: 'parser-a' }
  const first = { ...defaults, password: 'fixture' }
  defaults.accountId = 'account-b'; defaults.parserId = 'parser-b'
  const second = { ...defaults }
  defaults.accountId = 'account-c'
  for (const settings of [first, second]) {
    const preview = statementForm(file, settings)
    const commit = statementForm(file, settings)
    for (const key of ['account_id', 'parser_id', 'password']) assert.equal(commit.get(key), preview.get(key))
  }
  assert.equal(statementForm(file, first).get('account_id'), 'account-a')
  assert.equal(statementForm(file, second).get('account_id'), 'account-b')
})

test('automatic detection sends no manual overrides or empty password', () => {
  const form = statementForm(new File([''], 'empty.csv'), { accountId: 'AUTO', parserId: 'AUTO' })
  assert.equal(form.get('account_id'), null)
  assert.equal(form.get('parser_id'), null)
  assert.equal(form.get('password'), null)
})
