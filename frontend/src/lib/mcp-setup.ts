export type MCPProvider = 'codex' | 'claude' | 'ollama'
export type SetupShell = 'shell' | 'powershell'

interface MCPSetupOptions {
  provider: MCPProvider
  endpoint: string
  token: string
}

const providerSettings = {
  codex: { format: 'toml', file: 'config.toml', directoryEnv: 'CODEX_HOME', defaultDirectory: ['.codex'], subdirectory: [], rootKey: 'mcp_servers', transport: '', clients: ['codex'] },
  claude: { format: 'json', file: '.claude.json', directoryEnv: 'CLAUDE_CONFIG_DIR', defaultDirectory: [], subdirectory: [], rootKey: 'mcpServers', transport: 'http', clients: ['claude'] },
  ollama: { format: 'json', file: 'opencode.json', directoryEnv: 'XDG_CONFIG_HOME', defaultDirectory: ['.config'], subdirectory: ['opencode'], rootKey: 'mcp', transport: 'remote', clients: ['ollama', 'opencode'] },
} as const

export function mcpConfig({ provider, endpoint, token }: MCPSetupOptions) {
  const settings = providerSettings[provider]
  const headers = { Authorization: `Bearer ${token}` }
  if (settings.format === 'toml') {
    return `[mcp_servers.localfinance]\nurl = ${JSON.stringify(endpoint)}\nhttp_headers = { Authorization = ${JSON.stringify(headers.Authorization)} }\n`
  }
  const connection = { type: settings.transport, url: endpoint, headers }
  return JSON.stringify({
    [settings.rootKey]: {
      localfinance: settings.transport === 'remote' ? { ...connection, enabled: true, oauth: false } : connection,
    },
  }, null, 2)
}

// The same local installer runs in either shell. Values are serialized as data,
// never interpolated into shell expressions. No packages or scripts are fetched.
export function setupCommand({ shell, ...options }: MCPSetupOptions & { shell: SetupShell }) {
  const { provider } = options
  const settings = providerSettings[provider]
  const config = mcpConfig(options)
  const installer = `if (Number(process.versions.node.split('.')[0]) < 18) {
  console.error('Setup stopped: Node.js 18+ is required. Upgrade Node.js and run this command again.');
  process.exit(1);
}
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const provider = ${JSON.stringify(provider)};
const settings = ${JSON.stringify(settings)};
const config = ${JSON.stringify(config)};
const home = os.homedir();
const directory = path.join(process.env[settings.directoryEnv] || path.join(home, ...settings.defaultDirectory), ...settings.subdirectory);
const file = path.join(directory, settings.file);
let tempFilePath;
try {
  if (provider === 'ollama' && (process.env.OPENCODE_CONFIG || process.env.OPENCODE_CONFIG_CONTENT || fs.existsSync(path.join(directory, 'opencode.jsonc')))) throw new Error('Custom OpenCode configuration detected. Merge the manual configuration in Settings into that file instead.');
  if (fs.existsSync(file) && fs.lstatSync(file).isSymbolicLink()) throw new Error('Configuration is a symbolic link. Use the manual configuration in Settings.');
  const before = fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
  let after;
  if (settings.format === 'toml') {
    // Refuse advanced TOML forms rather than risk changing unrelated settings.
    const mcpTableKey = ${JSON.stringify('(?:mcp_servers|"mcp_servers"|\'mcp_servers\')')};
    const inlineMCP = new RegExp('^\\\\s*' + mcpTableKey + '(?:\\\\s*\\\\.[^=\\\\n]+)?\\\\s*=', 'm');
    const parentMCP = new RegExp('^\\\\s*\\\\[\\\\s*' + mcpTableKey + '\\\\s*\\\\]', 'm');
    const localFinanceTable = new RegExp('^\\\\s*\\\\[\\\\s*' + mcpTableKey + ${JSON.stringify(String.raw`\s*\.\s*(?:localfinance|"localfinance"|'localfinance')(?:\s*\.|\s*\])`)});
    if (before.includes('"""') || before.includes("'''")) throw new Error('Multiline TOML detected. Use the manual configuration in Settings.');
    if (inlineMCP.test(before)) throw new Error('Inline MCP table detected. Use the manual configuration in Settings.');
    if (parentMCP.test(before)) throw new Error('Parent MCP table detected. Use the manual configuration in Settings.');
    const lines = before.split(/\\r?\\n/);
    let skip = false;
    const kept = [];
    for (const line of lines) {
      if (/^\\s*\\[/.test(line)) {
        skip = localFinanceTable.test(line);
      }
      if (!skip) kept.push(line);
    }
    after = kept.join('\\n').trimEnd() + '\\n\\n' + config;
  } else {
    const data = before.trim() ? JSON.parse(before.replace(/^\\uFEFF/, '')) : {};
    const isPlainObject = value => value !== null && typeof value === 'object' && !Array.isArray(value);
    if (!isPlainObject(data)) throw new Error('Expected an object in the client configuration.');
    const key = settings.rootKey;
    if (data[key] !== undefined && !isPlainObject(data[key])) throw new Error('Expected an object for ' + key + '.');
    data[key] = { ...data[key], ...JSON.parse(config)[key] };
    after = JSON.stringify(data, null, 2) + '\\n';
  }
  fs.mkdirSync(directory, { recursive: true });
  if (before) {
    const backup = file + '.localfinance-' + Date.now() + '.bak';
    fs.writeFileSync(backup, before, { flag: 'wx', mode: 0o600 });
    console.log('Backup: ' + backup);
  }
  tempFilePath = file + '.localfinance-' + process.pid + '.tmp';
  fs.writeFileSync(tempFilePath, after, { flag: 'wx', mode: 0o600 });
  fs.renameSync(tempFilePath, file);
  tempFilePath = undefined;
  console.log('LocalFinance configured. Restart your AI client.');
} catch (error) {
  if (tempFilePath && fs.existsSync(tempFilePath)) fs.unlinkSync(tempFilePath);
  console.error('Setup stopped: ' + error.message);
  process.exitCode = 1;
}`
  // Client checks warn rather than block: desktop Codex need not expose a CLI,
  // and users can prepare configuration before installing their chosen client.
  const missingClientMessage = (client: string) => client === 'codex'
    ? 'Codex CLI was not found on PATH. Use the installed Codex desktop app, or install the CLI before connecting.'
    : `${client} was not found on PATH. Install it before connecting.`
  if (shell === 'powershell') {
    const clientChecks = settings.clients.map(client =>
      `  if (-not (Get-Command ${client} -ErrorAction SilentlyContinue)) { Write-Warning '${missingClientMessage(client)}' }`,
    ).join('\n')
    return `& {\n  if (-not (Get-Command node -ErrorAction SilentlyContinue)) { throw 'Install Node.js 18+ first, then run this command again.' }\n${clientChecks}\n@'\n${installer}\n'@ | node\n  if ($LASTEXITCODE -ne 0) { throw 'Configuration was not updated. See the error above.' }\n}`
  }
  const clientChecks = settings.clients.map(client =>
    `command -v ${client} >/dev/null 2>&1 || echo '${missingClientMessage(client)}' >&2`,
  ).join('\n')
  return `(\ncommand -v node >/dev/null 2>&1 || { echo 'Install Node.js 18+ first, then run this command again.' >&2; exit 1; }\n${clientChecks}\nnode <<'LOCALFINANCE_SETUP'\n${installer}\nLOCALFINANCE_SETUP\n[ $? -eq 0 ] || exit 1\n)`
}
