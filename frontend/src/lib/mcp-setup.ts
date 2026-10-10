export type MCPProvider = 'codex' | 'claude' | 'ollama'
export type SetupShell = 'shell' | 'powershell'

interface MCPSetupOptions {
  provider: MCPProvider
  endpoint: string
  token: string
}

const providerSettings = {
  codex: { format: 'toml', rootKey: 'mcp_servers', transport: '', clients: ['codex'] },
  claude: { format: 'json', rootKey: 'mcpServers', transport: 'http', clients: ['claude'] },
  ollama: { format: 'json', rootKey: 'mcp', transport: 'remote', clients: ['opencode'] },
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

// Quote user values for the selected shell; never treat them as shell code.
function shellQuote(value: string, shell: SetupShell) {
  return `'${value.replaceAll("'", shell === 'powershell' ? "''" : "'\"'\"'")}'`
}

export function setupCommand({ provider, shell, endpoint, token }: MCPSetupOptions & { shell: SetupShell }) {
  const quote = (value: string) => shellQuote(value, shell)
  const url = quote(endpoint)
  const header = quote(`Authorization: Bearer ${token}`)
  const clients = providerSettings[provider].clients
  const codexHeaders = `[mcp_servers.localfinance.http_headers]\nAuthorization = ${JSON.stringify(`Bearer ${token}`)}`
  let command: string
  if (provider === 'claude') {
    // Replace only this user-scoped connection, so token rotation can be rerun.
    command = shell === 'powershell'
      ? `  & claude mcp remove --scope user localfinance 2>$null | Out-Null\n  & claude mcp add --transport http --scope user localfinance ${url} --header ${header}`
      : `claude mcp remove --scope user localfinance >/dev/null 2>&1 || true\nclaude mcp add --transport http --scope user localfinance ${url} --header ${header}`
  } else if (provider === 'ollama') {
    command = `${shell === 'powershell' ? '  & ' : ''}opencode mcp add localfinance --url ${url} --header ${quote(`Authorization=Bearer ${token}`)}`
  } else {
    command = `${shell === 'powershell' ? '  & ' : ''}codex mcp add localfinance --url ${url}`
  }
  if (shell === 'powershell') {
    const checks = clients.map(client => `  if (-not (Get-Command ${client} -ErrorAction SilentlyContinue)) { throw '${client} is not installed or not on PATH. Install it and run setup again.' }`).join('\n')
    // The native add command replaces this server, including its old headers.
    // Restrict access before writing the token, on Windows and POSIX hosts.
    const saveHeader = provider === 'codex' ? `\n  $codexDirectory = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME '.codex' }\n  $configFile = Join-Path $codexDirectory 'config.toml'\n  if ($env:OS -eq 'Windows_NT') {\n    $owner = [System.Security.Principal.WindowsIdentity]::GetCurrent().User\n    $acl = [System.Security.AccessControl.FileSecurity]::new()\n    $acl.SetOwner($owner)\n    $acl.SetAccessRuleProtection($true, $false)\n    $acl.AddAccessRule([System.Security.AccessControl.FileSystemAccessRule]::new($owner, 'FullControl', 'Allow'))\n    Set-Acl -LiteralPath $configFile -AclObject $acl\n  } else {\n    & chmod 600 $configFile\n    if ($LASTEXITCODE -ne 0) { throw 'Could not protect the Codex configuration file.' }\n  }\n  Add-Content -LiteralPath $configFile -Encoding utf8 -Value @'\n\n${codexHeaders}\n'@` : ''
    return `& {\n  $ErrorActionPreference = 'Stop'\n  $PSNativeCommandUseErrorActionPreference = $false\n${checks}\n${command}\n  if ($LASTEXITCODE -ne 0) { throw 'MCP setup failed. Check the client error above; update the client if it does not support these options.' }${saveHeader}\n  Write-Output 'LocalFinance configured. Restart your AI client.'\n}`
  }
  const checks = clients.map(client => `command -v ${client} >/dev/null 2>&1 || { echo '${client} is not installed or not on PATH. Install it and run setup again.' >&2; exit 1; }`).join('\n')
  const saveHeader = provider === 'codex' ? `\nconfig_file="\${CODEX_HOME:-$HOME/.codex}/config.toml"\nchmod 600 "$config_file"\ncat >> "$config_file" <<'LOCALFINANCE_HEADERS'\n\n${codexHeaders}\nLOCALFINANCE_HEADERS` : ''
  return `(\nset -e\n${checks}\n${command}${saveHeader}\necho 'LocalFinance configured. Restart your AI client.'\n)`
}
