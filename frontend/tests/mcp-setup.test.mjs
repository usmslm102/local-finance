import assert from 'node:assert/strict'
import { readFile, mkdtemp, mkdir, writeFile, readdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { spawnSync } from 'node:child_process'
import { test } from 'node:test'
import ts from 'typescript'

const source = await readFile(new URL('../src/lib/mcp-setup.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022 },
})
const { setupCommand } = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const endpoint = 'http://127.0.0.1:18081/mcp'
const token = 'fixture-token\'"`$(touch SHOULD_NOT_EXIST)\\$;'
const powerShell = spawnSync('which', ['pwsh'], { encoding: 'utf8' }).stdout.trim()
const shells = ['shell', ...(powerShell ? ['powershell'] : [])]
const quote = value => `'${value.replaceAll("'", "'\"'\"'")}'`

async function fixture(t) {
  const home = await mkdtemp(join(tmpdir(), 'lf-native setup-'))
  t.after(() => rm(home, { recursive: true, force: true }))
  const bin = join(home, 'bin')
  await mkdir(bin)
  const env = { ...process.env, HOME: home, USERPROFILE: home, CODEX_HOME: join(home, '.codex'), CLAUDE_CONFIG_DIR: home, XDG_CONFIG_HOME: join(home, '.config'), XDG_DATA_HOME: join(home, '.data'), XDG_CACHE_HOME: join(home, '.cache'), XDG_STATE_HOME: join(home, '.state'), LF_TEST_HOME: home }
  for (const key of ['NODE_OPTIONS', 'OPENCODE_CONFIG', 'OPENCODE_CONFIG_CONTENT', 'OPENCODE_CONFIG_DIR']) delete env[key]
  const run = (provider, shell) => spawnSync(shell === 'shell' ? '/bin/sh' : powerShell, shell === 'shell' ? ['-s'] : ['-NoProfile', '-NonInteractive', '-Command', setupCommand({ provider, shell, endpoint, token })], {
    input: shell === 'shell' ? setupCommand({ provider, shell, endpoint, token }) : undefined,
    env, cwd: home, encoding: 'utf8', timeout: 15000,
  })
  return { home, bin, env, run }
}

async function mockClients(f, clients) {
  const script = join(f.home, 'client.cjs')
  await writeFile(script, `const fs = require('fs'); const path = require('path');
const [client, ...args] = process.argv.slice(2);
const home = process.env.LF_TEST_HOME;
const log = path.join(home, 'calls.json');
const calls = fs.existsSync(log) ? JSON.parse(fs.readFileSync(log)) : [];
calls.push({ client, args }); fs.writeFileSync(log, JSON.stringify(calls));
if (process.env.LF_CLIENT_FAIL && args[1] !== 'remove') process.exit(2);
if (client === 'codex' && args[1] === 'add') {
  fs.mkdirSync(process.env.CODEX_HOME, { recursive: true });
  fs.writeFileSync(path.join(process.env.CODEX_HOME, 'config.toml'), '[mcp_servers.localfinance]\\nurl = "' + args[4] + '"\\n');
}
`)
  for (const client of clients) {
    await writeFile(join(f.bin, client), `#!/bin/sh\nexec ${quote(process.execPath)} ${quote(script)} ${client} "$@"\n`, { mode: 0o700 })
  }
  f.env.PATH = f.bin // The generated command cannot depend on node or other clients.
  // Codex's small header append uses standard POSIX utilities.
  const { symlink } = await import('node:fs/promises')
  await symlink('/bin/cat', join(f.bin, 'cat'))
  await symlink('/bin/chmod', join(f.bin, 'chmod'))
}

for (const shell of shells) {
  for (const provider of ['codex', 'claude', 'ollama']) {
    test(`${shell}: ${provider} configures through its CLI and quotes the token literally`, async t => {
      const f = await fixture(t)
      await mockClients(f, ['codex', 'claude', 'ollama', 'opencode'])
      const result = f.run(provider, shell)
      assert.equal(result.status, 0, result.stderr)
      const calls = JSON.parse(await readFile(join(f.home, 'calls.json'), 'utf8'))
      if (provider === 'codex') {
        assert.deepEqual(calls, [{ client: 'codex', args: ['mcp', 'add', 'localfinance', '--url', endpoint] }])
        const config = await readFile(join(f.env.CODEX_HOME, 'config.toml'), 'utf8')
        assert.ok(config.includes(`Authorization = ${JSON.stringify(`Bearer ${token}`)}`))
      } else if (provider === 'claude') {
        assert.deepEqual(calls, [
          { client: 'claude', args: ['mcp', 'remove', '--scope', 'user', 'localfinance'] },
          { client: 'claude', args: ['mcp', 'add', '--transport', 'http', '--scope', 'user', 'localfinance', endpoint, '--header', `Authorization: Bearer ${token}`] },
        ])
      } else {
        assert.deepEqual(calls, [{ client: 'opencode', args: ['mcp', 'add', 'localfinance', '--url', endpoint, '--header', `Authorization=Bearer ${token}`] }])
      }
      assert.equal((await readdir(f.home)).includes('SHOULD_NOT_EXIST'), false)
      assert.match(result.stdout, /LocalFinance configured/)
    })

    test(`${shell}: ${provider} stops before configuration when its client is missing`, async t => {
      const f = await fixture(t)
      await mockClients(f, provider === 'ollama' ? ['ollama'] : [])
      const result = f.run(provider, shell)
      assert.notEqual(result.status, 0)
      assert.match(result.stderr, /not installed or not on PATH/)
      assert.equal((await readdir(f.home)).includes('calls.json'), false)
    })

    test(`${shell}: ${provider} reports a failed native client command`, async t => {
      const f = await fixture(t)
      await mockClients(f, ['codex', 'claude', 'ollama', 'opencode'])
      f.env.LF_CLIENT_FAIL = '1'
      const result = f.run(provider, shell)
      assert.notEqual(result.status, 0)
      assert.doesNotMatch(result.stdout, /LocalFinance configured/)
      assert.equal((await readdir(f.home)).includes('.codex'), false)
    })
  }
}

// Exercise installed vendor CLIs against isolated user configuration as well.
for (const shell of shells) {
  for (const provider of ['codex', 'claude', 'ollama']) {
    const client = provider === 'ollama' ? 'opencode' : provider
    const installed = spawnSync('which', [client], { encoding: 'utf8' }).status === 0
    test(`${shell}: installed ${client} preserves other settings and supports repeated setup`, { skip: !installed }, async t => {
      const f = await fixture(t)
      if (provider === 'ollama') {
        await writeFile(join(f.bin, 'ollama'), '#!/bin/sh\nexit 0\n', { mode: 0o700 })
        f.env.PATH = `${f.bin}:${process.env.PATH}`
      }
      const directory = provider === 'codex' ? f.env.CODEX_HOME : provider === 'claude' ? f.home : join(f.env.XDG_CONFIG_HOME, 'opencode')
      await mkdir(directory, { recursive: true })
      const file = join(directory, provider === 'codex' ? 'config.toml' : provider === 'claude' ? '.claude.json' : 'opencode.json')
      const rootKey = provider === 'claude' ? 'mcpServers' : 'mcp'
      const original = provider === 'codex'
        ? 'model = "fixture"\n[mcp_servers.other]\nurl = "http://127.0.0.1:19000/mcp"\n'
        : JSON.stringify({ [rootKey]: { other: { type: provider === 'claude' ? 'http' : 'remote', url: 'http://127.0.0.1:19000/mcp' } } })
      await writeFile(file, original)
      for (let i = 0; i < 2; i++) {
        const result = f.run(provider, shell)
        assert.equal(result.status, 0, result.stderr)
      }
      const config = await readFile(file, 'utf8')
      if (provider === 'codex') {
        assert.match(config, /model = "fixture"/)
        assert.match(config, /\[mcp_servers.other\]/)
        assert.equal(config.split('[mcp_servers.localfinance.http_headers]').length, 2)
        assert.ok(config.includes(JSON.stringify(`Bearer ${token}`)))
      } else {
        const servers = JSON.parse(config)[rootKey]
        assert.equal(servers.other.url, 'http://127.0.0.1:19000/mcp')
        assert.equal(servers.localfinance.url, endpoint)
        assert.equal(servers.localfinance.headers.Authorization, `Bearer ${token}`)
      }
      assert.equal((await readdir(f.home)).includes('SHOULD_NOT_EXIST'), false)
    })
  }
}
