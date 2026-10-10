import assert from 'node:assert/strict'
import { readFile, mkdtemp, mkdir, writeFile, readdir, stat, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, dirname } from 'node:path'
import { spawnSync } from 'node:child_process'
import { test } from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/lib/mcp-setup.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { setupCommand } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const endpoint = 'http://127.0.0.1:18081/mcp'
// These must stay literal data through both shell layers.
const token = 'fixture-token\'"`$(touch SHOULD_NOT_EXIST)\\$;'
const hasPowerShell = spawnSync('pwsh', ['-NoProfile', '-Command', 'exit 0']).status === 0

for (const shell of ['shell', ...(hasPowerShell ? ['powershell'] : [])]) {
  for (const provider of ['codex', 'claude', 'ollama']) {
    test(`${shell}: ${provider} setup preserves settings, backs up and reruns safely`, async t => {
      const home = await mkdtemp(join(tmpdir(), 'lf-mcp-'))
      t.after(() => rm(home, { recursive: true, force: true }))
      const file = join(home, provider === 'codex' ? '.codex/config.toml' : provider === 'claude' ? '.claude.json' : '.config/opencode/opencode.json')
      await mkdir(dirname(file), { recursive: true })
      const original = provider === 'codex'
        ? 'model = "fixture"\n[mcp_servers.other]\nurl = "http://localhost:9000"\n[mcp_servers."localfinance"]\nurl = "http://old"\n[mcp_servers."localfinance".http_headers]\nAuthorization = "old"\n[features]\nother = true\n'
        : JSON.stringify({ model: 'fixture', [provider === 'claude' ? 'mcpServers' : 'mcp']: { other: { command: 'keep' }, localfinance: { url: 'http://old' } } })
      await writeFile(file, original)
      const bin = join(home, 'bin')
      await mkdir(bin)
      await writeFile(join(bin, 'ollama'), '#!/bin/sh\nprintf "%s\\n" "$*" > "$HOME/ollama-called"\n', { mode: 0o700 })
      const env = { ...process.env, HOME: home, USERPROFILE: home, PATH: `${bin}:${process.env.PATH}` }
      for (const name of ['CODEX_HOME', 'CLAUDE_CONFIG_DIR', 'XDG_CONFIG_HOME', 'OPENCODE_CONFIG', 'OPENCODE_CONFIG_CONTENT']) delete env[name]
      const command = setupCommand({ provider, shell, endpoint, token })
      const run = () => shell === 'shell' ? spawnSync('sh', ['-s'], { input: command, env, cwd: home, encoding: 'utf8' }) : spawnSync('pwsh', ['-NoProfile', '-NonInteractive', '-Command', command], { env, cwd: home, encoding: 'utf8' })
      const first = run()
      assert.equal(first.status, 0, first.stderr)
      const configured = await readFile(file, 'utf8')
      const second = run()
      assert.equal(second.status, 0, second.stderr)
      assert.equal(await readFile(file, 'utf8'), configured)
      assert.equal((await readdir(home)).includes('SHOULD_NOT_EXIST'), false)
      if (provider === 'codex') {
        assert.match(configured, /model = "fixture"/)
        assert.match(configured, /\[mcp_servers.other\]/)
        assert.match(configured, /\[features\]\nother = true/)
        assert.equal(configured.split('[mcp_servers.localfinance]').length, 2)
        assert.ok(configured.includes(JSON.stringify(`Bearer ${token}`)))
        assert.equal(configured.includes('http://old'), false)
      } else {
        const parsed = JSON.parse(configured)
        assert.equal(parsed.model, 'fixture')
        const servers = parsed[provider === 'claude' ? 'mcpServers' : 'mcp']
        assert.equal(servers.other.command, 'keep')
        assert.equal(servers.localfinance.url, endpoint)
        assert.equal(servers.localfinance.headers.Authorization, `Bearer ${token}`)
      }
      const backups = (await readdir(dirname(file))).filter(name => name.endsWith('.bak'))
      assert.equal(backups.length, 2)
      assert.ok((await Promise.all(backups.map(name => readFile(join(dirname(file), name), 'utf8')))).includes(original))
      assert.equal((await stat(file)).mode & 0o777, 0o600)
      assert.equal((await readdir(home)).includes('ollama-called'), false)
    })
  }
}

test('invalid JSON and custom OpenCode configurations are left untouched', async t => {
  const home = await mkdtemp(join(tmpdir(), 'lf-mcp-refuse-'))
  t.after(() => rm(home, { recursive: true, force: true }))
  const file = join(home, '.config/opencode/opencode.json')
  await mkdir(dirname(file), { recursive: true })
  await writeFile(file, '{ broken JSON')
  const env = { ...process.env, HOME: home, XDG_CONFIG_HOME: join(home, '.config') }
  const script = setupCommand({ provider: 'ollama', shell: 'shell', endpoint, token })
  const bin = join(home, 'bin')
  await mkdir(bin)
  await writeFile(join(bin, 'ollama'), '#!/bin/sh\ntouch "$HOME/UNEXPECTED_LAUNCH"\n', { mode: 0o700 })
  env.PATH = `${bin}:${env.PATH}`
  const result = spawnSync('sh', ['-s'], { input: script, env, encoding: 'utf8' })
  assert.notEqual(result.status, 0)
  assert.equal(await readFile(file, 'utf8'), '{ broken JSON')
  assert.equal((await readdir(home)).includes('UNEXPECTED_LAUNCH'), false)
  await writeFile(file, '{}')
  await writeFile(join(dirname(file), 'opencode.jsonc'), '// existing custom config')
  const custom = spawnSync('sh', ['-s'], { input: script, env, encoding: 'utf8' })
  assert.notEqual(custom.status, 0)
  assert.match(custom.stderr, /manual configuration/)
  assert.equal(await readFile(file, 'utf8'), '{}')
})

test('advanced Codex TOML stops without overwriting the file', async t => {
  const home = await mkdtemp(join(tmpdir(), 'lf-mcp-toml-'))
  t.after(() => rm(home, { recursive: true, force: true }))
  const directory = join(home, '.codex')
  await mkdir(directory)
  const file = join(directory, 'config.toml')
  for (const original of ['instructions = """\nkeep\n"""', 'mcp_servers.localfinance.url = "http://old"', '[mcp_servers]\nlocalfinance = { url = "http://old" }']) {
    await writeFile(file, original)
    const result = spawnSync('sh', ['-s'], { input: setupCommand({ provider: 'codex', shell: 'shell', endpoint, token }), env: { ...process.env, HOME: home, CODEX_HOME: directory }, encoding: 'utf8' })
    assert.notEqual(result.status, 0)
    assert.match(result.stderr, /manual configuration/)
    assert.equal(await readFile(file, 'utf8'), original)
    assert.deepEqual(await readdir(directory), ['config.toml'])
  }
})

for (const shell of ['shell', ...(hasPowerShell ? ['powershell'] : [])]) {
  test(`${shell}: outdated Node stops before touching configuration`, async t => {
    const home = await mkdtemp(join(tmpdir(), 'lf-mcp-old-node-'))
    t.after(() => rm(home, { recursive: true, force: true }))
    const preload = join(home, 'old-node.cjs')
    await writeFile(preload, "Object.defineProperty(process.versions, 'node', { value: '16.20.2' });")
    const directory = join(home, '.codex')
    const env = { ...process.env, HOME: home, CODEX_HOME: directory, NODE_OPTIONS: `--require=${preload}` }
    const command = setupCommand({ provider: 'codex', shell, endpoint, token })
    const result = shell === 'shell'
      ? spawnSync('sh', ['-s'], { input: command, env, encoding: 'utf8' })
      : spawnSync('pwsh', ['-NoProfile', '-NonInteractive', '-Command', command], { env, encoding: 'utf8' })
    assert.notEqual(result.status, 0)
    assert.match(result.stderr, /Node.js 18\+ is required/)
    assert.equal((await readdir(home)).includes('.codex'), false)
  })
}

test('missing clients produce useful warnings while allowing configuration ahead of installation', async t => {
  const home = await mkdtemp(join(tmpdir(), 'lf-mcp-client-check-'))
  t.after(() => rm(home, { recursive: true, force: true }))
  const bin = join(home, 'bin')
  await mkdir(bin)
  const { symlink } = await import('node:fs/promises')
  await symlink(process.execPath, join(bin, 'node'))
  const env = { ...process.env, HOME: home, CODEX_HOME: join(home, '.codex'), CLAUDE_CONFIG_DIR: home, XDG_CONFIG_HOME: join(home, '.config'), PATH: bin }
  delete env.OPENCODE_CONFIG
  delete env.OPENCODE_CONFIG_CONTENT
  for (const provider of ['codex', 'claude', 'ollama']) {
    const result = spawnSync('/bin/sh', ['-s'], { input: setupCommand({ provider, shell: 'shell', endpoint, token }), env, encoding: 'utf8' })
    assert.equal(result.status, 0, result.stderr)
    assert.match(result.stdout, /LocalFinance configured/)
    if (provider === 'codex') assert.match(result.stderr, /Codex desktop app/)
    if (provider === 'claude') assert.match(result.stderr, /claude was not found on PATH/)
    if (provider === 'ollama') {
      assert.match(result.stderr, /ollama was not found on PATH/)
      assert.match(result.stderr, /opencode was not found on PATH/)
    }
  }
})
