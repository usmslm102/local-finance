import { useState } from 'react'
import { Check, Copy, Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { mcpConfig, setupCommand, type MCPProvider, type SetupShell } from '@/lib/mcp-setup'

const providers = [
  { file: '~/.codex/config.toml (or CODEX_HOME/config.toml)', id: 'codex', label: 'Codex', title: 'Connect Codex', prerequisite: 'Install Codex CLI or the desktop app and Node.js 18+.', next: 'Restart Codex, then check LocalFinance in its MCP tools.', docs: 'https://learn.chatgpt.com/docs/extend/mcp?surface=cli' },
  { file: '~/.claude.json (or CLAUDE_CONFIG_DIR/.claude.json)', id: 'claude', label: 'Claude Code', title: 'Connect Claude Code', prerequisite: 'Install Claude Code and Node.js 18+.', next: 'Restart Claude Code and run /mcp to check the connection.', docs: 'https://code.claude.com/docs/en/mcp' },
  { file: '~/.config/opencode/opencode.json (or your custom OpenCode config)', id: 'ollama', label: 'Ollama', title: 'Connect Ollama through OpenCode', prerequisite: 'Install Ollama, OpenCode, and Node.js 18+. Have a local model with tool support ready.', next: 'After setup, run ollama launch opencode, choose a local model, and ask it to use LocalFinance.', docs: 'https://docs.ollama.com/integrations/opencode' },
] satisfies { file: string; id: MCPProvider; label: string; title: string; prerequisite: string; next: string; docs: string }[]

function Copyable({ value, preview, label, disabled = false }: { value: string; preview?: string; label: string; disabled?: boolean }) {
  const [copiedValue, setCopiedValue] = useState('')
  const [error, setError] = useState('')
  const [reveal, setReveal] = useState(false)
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedValue(value)
      setError('')
    } catch {
      setReveal(true)
      setError('Clipboard unavailable. Select and copy the text below.')
    }
  }
  return <div className="space-y-2">
    <div className="flex flex-wrap items-center justify-between gap-2">
      <Label>{label}</Label>
      <div className="flex gap-2">
        {preview && <Button variant="ghost" size="sm" disabled={disabled} onClick={() => setReveal(!reveal)} aria-pressed={reveal}>
          {reveal ? <EyeOff /> : <Eye />}{reveal ? 'Hide token' : 'Show token'}
        </Button>}
        <Button size="sm" disabled={disabled} onClick={copy}>
          {copiedValue === value ? <Check /> : <Copy />}{copiedValue === value ? 'Copied' : `Copy ${label.toLowerCase()}`}
        </Button>
      </div>
    </div>
    <Textarea aria-label={label} readOnly value={reveal ? value : preview ?? value} className="min-h-40 max-h-64 resize-y font-mono text-xs" onFocus={event => event.target.select()} />
    <p role="status" className="text-sm text-muted-foreground">{copiedValue === value ? 'Copied with your token. Run it on the computer running LocalFinance.' : preview && !reveal ? 'Token hidden in this preview. Copy includes your token automatically.' : ''}</p>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
  </div>
}

export function MCPConnectionSetup({ endpoint, token, setToken, listening }: { endpoint: string; token: string; setToken: (value: string) => void; listening: boolean }) {
  const [shell, setShell] = useState<SetupShell>(() => /Win/i.test(navigator.platform) ? 'powershell' : 'shell')
  const [manualProvider, setManualProvider] = useState<MCPProvider | null>(null)
  const ready = Boolean(token.trim())
  return <div className="space-y-5">
    <div className="space-y-2">
      <Label htmlFor="mcp-setup-token">Access token</Label>
      <Input id="mcp-setup-token" type="password" autoComplete="off" placeholder="Paste your saved token" value={token} onChange={event => setToken(event.target.value.trim())} />
      <p className="text-sm text-muted-foreground">New tokens are filled in automatically. To reconnect, paste your saved token. If you lost it, rotate it above and reconnect every client.</p>
    </div>
    <Tabs defaultValue="codex" className="gap-5 min-w-0">
      <TabsList aria-label="AI client" className="w-full h-auto! flex-wrap justify-start">
        {providers.map(provider => <TabsTrigger key={provider.id} value={provider.id} className="min-h-9 px-3">{provider.label}</TabsTrigger>)}
        <TabsTrigger value="other" className="min-h-9 px-3">Other clients</TabsTrigger>
      </TabsList>
      {providers.map(provider => <TabsContent key={provider.id} value={provider.id} className="space-y-4 min-w-0">
        <div className="space-y-1">
          <h3 className="font-medium">{provider.title}</h3>
          <p className="text-sm text-muted-foreground">{provider.prerequisite}</p>
        </div>
        <Tabs value={shell} onValueChange={value => setShell(value as SetupShell)} className="gap-3">
          <TabsList aria-label={`${provider.label} command format`} className="max-w-full h-auto! flex-wrap justify-start">
            <TabsTrigger value="shell" className="min-h-8">Shell · macOS / Linux</TabsTrigger>
            <TabsTrigger value="powershell" className="min-h-8">PowerShell · Windows</TabsTrigger>
          </TabsList>
          {(['shell', 'powershell'] as const).map(format => <TabsContent key={format} value={format}>
            <Copyable label="Command" disabled={!ready} value={ready ? setupCommand({ provider: provider.id, shell: format, endpoint, token: token.trim() }) : 'Create or paste a token to prepare your setup command.'} preview={ready ? setupCommand({ provider: provider.id, shell: format, endpoint, token: 'TOKEN_HIDDEN_IN_PREVIEW' }) : undefined} />
          </TabsContent>)}
        </Tabs>
        <p className="text-sm text-muted-foreground">Run the command in your terminal. It saves a private user configuration and backs up existing settings. It replaces only the LocalFinance connection.</p>
        <p className="text-sm">{provider.next}</p>
        <Button variant="ghost" size="sm" onClick={() => setManualProvider(manualProvider === provider.id ? null : provider.id)} aria-expanded={manualProvider === provider.id}>Manual configuration</Button>
        {manualProvider === provider.id && <div className="space-y-3">
          <p className="text-sm text-muted-foreground">Merge this entry into {provider.file}. Preserve your other settings.</p>
          <Copyable label="Configuration" disabled={!ready} value={mcpConfig({ provider: provider.id, endpoint, token: token.trim() })} preview={mcpConfig({ provider: provider.id, endpoint, token: 'TOKEN_HIDDEN_IN_PREVIEW' })} />
        </div>}
        <a href={provider.docs} target="_blank" rel="noreferrer" className="block w-fit text-sm text-muted-foreground underline underline-offset-4">{provider.label} setup documentation</a>
      </TabsContent>)}
      <TabsContent value="other" className="space-y-4">
        <p className="text-sm text-muted-foreground">Use a client with Streamable HTTP support on this computer. Cloud-only clients cannot reach this local address. Each client has its own configuration format.</p>
        <Copyable label="Connection details" disabled={!ready} value={`URL: ${endpoint}\nTransport: Streamable HTTP\nAuthorization: Bearer ${token.trim()}`} preview={`URL: ${endpoint}\nTransport: Streamable HTTP\nAuthorization: Bearer TOKEN_HIDDEN_IN_PREVIEW`} />
      </TabsContent>
    </Tabs>
    {!listening && <p role="status" className="text-sm text-muted-foreground">You can copy and run setup now. Enable MCP above when you’re ready to connect. If the listener is unavailable, choose another port and retry.</p>}
    <p className="text-sm text-muted-foreground">Keep LocalFinance running. Commands contain your access token; keep them private. The token stays in memory here and is cleared when you leave this settings tab.</p>
  </div>
}
