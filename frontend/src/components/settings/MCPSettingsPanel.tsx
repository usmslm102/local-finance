import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, Check, Copy, ShieldCheck } from 'lucide-react'
import { fetchMCPSettings, rotateMCPToken, updateMCPSettings } from '@/lib/api'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

function ConfigSnippet({ title, value }: { title: string; value: string }) {
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      setError('')
    } catch {
      setError(
        'Clipboard unavailable. Select and copy the configuration below.',
      )
    }
  }
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-3">
        <Label>{title}</Label>
        <Button size="sm" variant="outline" onClick={copy}>
          {copied ? <Check /> : <Copy />}
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <Textarea
        aria-label={`${title} configuration`}
        readOnly
        value={value}
        className="font-mono text-xs min-h-24"
      />
      {error && (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      )}
    </div>
  )
}

export function MCPSettingsPanel() {
  const client = useQueryClient()
  const query = useQuery({
    queryKey: ['mcp-settings'],
    queryFn: fetchMCPSettings,
    refetchInterval: 10000,
  })
  const [portDraft, setPortDraft] = useState<string | null>(null)
  const [token, setToken] = useState('')
  const [showRotate, setShowRotate] = useState(false)
  const [copied, setCopied] = useState(false)
  const [copyError, setCopyError] = useState('')
  const save = useMutation({
    mutationFn: ({ enabled, port }: { enabled: boolean; port: number }) =>
      updateMCPSettings(enabled, port),
    onSuccess: (settings) => {
      client.setQueryData(['mcp-settings'], settings)
      setPortDraft(null)
    },
  })
  const rotate = useMutation({
    mutationFn: rotateMCPToken,
    onSuccess: (result) => {
      setToken(result.token)
      setCopied(false)
      setCopyError('')
      client.setQueryData(['mcp-settings'], result.settings)
      setShowRotate(false)
    },
  })
  const settings = query.data
  const busy = save.isPending || rotate.isPending
  const port = Number(portDraft ?? settings?.port ?? 8081)
  const validPort = Number.isInteger(port) && port >= 1024 && port <= 65535
  const copyToken = async () => {
    try {
      await navigator.clipboard.writeText(token)
      setCopied(true)
      setCopyError('')
    } catch {
      setCopyError('Clipboard unavailable. Select and copy the token field.')
    }
  }
  if (query.isPending)
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Loading MCP settings…
      </p>
    )
  if (!settings)
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <AlertTitle>Unable to load MCP settings</AlertTitle>
        <AlertDescription>
          <p>{query.error?.message}</p>
          <Button variant="outline" onClick={() => query.refetch()}>
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    )
  const endpoint = settings.endpoint
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <CardTitle className="text-base">AI access through MCP</CardTitle>
            <Badge variant="outline">
              {settings.listening
                ? 'Listening · read-only'
                : settings.enabled
                  ? 'Unavailable'
                  : 'Disabled'}
            </Badge>
          </div>
          <CardDescription>
            Connect Codex, Claude Code, or another AI tool to your local finance
            data.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <Alert>
            <ShieldCheck />
            <AlertTitle>Choose what you share</AlertTitle>
            <AlertDescription>
              LocalFinance serves data only on this computer. Connected AI tools
              can read transactions, notes, payees, balances, and reports, and
              may send them to their model provider. MCP stays available while
              the app is locked. Disable MCP to stop access.
            </AlertDescription>
          </Alert>
          <div className="flex items-center justify-between gap-4">
            <div>
              <Label htmlFor="mcp-enabled">Enable read-only MCP</Label>
              <p className="text-sm text-muted-foreground">
                {settings.has_token
                  ? 'LocalFinance must remain running.'
                  : 'Create a token first. LocalFinance must remain running.'}
              </p>
            </div>
            <Switch
              id="mcp-enabled"
              checked={settings.enabled}
              disabled={busy || (!settings.has_token && !settings.enabled)}
              onCheckedChange={(enabled) =>
                save.mutate({ enabled, port: settings.port })
              }
            />
          </div>
          <div className="flex items-end gap-3 flex-wrap">
            <div className="space-y-2">
              <Label htmlFor="mcp-port">Local MCP port</Label>
              <Input
                id="mcp-port"
                type="number"
                min={1024}
                max={65535}
                value={portDraft ?? String(settings.port)}
                onChange={(event) => setPortDraft(event.target.value)}
                className="w-36"
                aria-describedby="mcp-port-help"
              />
            </div>
            <Button
              variant="outline"
              disabled={busy || !validPort}
              onClick={() => save.mutate({ enabled: settings.enabled, port })}
            >
              {save.isPending ? 'Saving…' : 'Save / retry listener'}
            </Button>
          </div>
          <p id="mcp-port-help" className="text-sm text-muted-foreground">
            Use a port from 1024 to 65535. Changing it requires updating your AI
            tool configuration.
          </p>
          {settings.error && (
            <Alert variant="destructive">
              <AlertCircle />
              <AlertTitle>MCP listener unavailable</AlertTitle>
              <AlertDescription>{settings.error}</AlertDescription>
            </Alert>
          )}
          <div className="border-t pt-4 space-y-2">
            <p className="text-sm">
              One shared token grants read-only access to all connected AI
              tools.
            </p>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() =>
                settings.has_token ? setShowRotate(true) : rotate.mutate()
              }
            >
              {rotate.isPending
                ? 'Creating…'
                : settings.has_token
                  ? 'Rotate access token'
                  : 'Create access token'}
            </Button>
            {token && (
              <div className="space-y-3 rounded-lg border p-4">
                <Label htmlFor="mcp-new-token">
                  New token — save it before leaving this tab
                </Label>
                <Input
                  id="mcp-new-token"
                  readOnly
                  value={token}
                  className="font-mono"
                  onFocus={(event) => event.target.select()}
                />
                <div className="flex gap-2 flex-wrap">
                  <Button variant="outline" onClick={copyToken}>
                    {copied ? <Check /> : <Copy />}
                    {copied ? 'Token copied' : 'Copy token'}
                  </Button>
                  <Button variant="ghost" onClick={() => setToken('')}>
                    Dismiss token
                  </Button>
                </div>
                <p className="text-sm text-muted-foreground">
                  Shown once. LocalFinance stores only its hash. Do not commit
                  the token to a repository.
                </p>
                {copyError && (
                  <p role="alert" className="text-sm text-destructive">
                    {copyError}
                  </p>
                )}
              </div>
            )}
          </div>
          {(save.error || rotate.error || query.error) && (
            <p role="alert" className="text-sm text-destructive">
              {save.error?.message ||
                rotate.error?.message ||
                query.error?.message}
            </p>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Connect your AI tool</CardTitle>
          <CardDescription>
            Replace YOUR_TOKEN with the token you saved. Use private user
            configuration.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <ConfigSnippet
            title="Codex · config.toml"
            value={`[mcp_servers.localfinance]\nurl = "${endpoint}"\nbearer_token_env_var = "LOCALFINANCE_MCP_TOKEN"`}
          />
          <p className="text-sm text-muted-foreground">
            Set LOCALFINANCE_MCP_TOKEN in the environment that launches Codex.
            If your desktop client does not inherit it, replace
            bearer_token_env_var with the static header below in private
            configuration.
          </p>
          <ConfigSnippet
            title="Codex · alternative static header"
            value={`[mcp_servers.localfinance]\nurl = "${endpoint}"\nhttp_headers = { Authorization = "Bearer YOUR_TOKEN" }`}
          />
          <ConfigSnippet
            title="Claude Code · private user configuration"
            value={JSON.stringify(
              {
                mcpServers: {
                  localfinance: {
                    type: 'http',
                    url: endpoint,
                    headers: { Authorization: 'Bearer YOUR_TOKEN' },
                  },
                },
              },
              null,
              2,
            )}
          />
          <p className="text-sm text-muted-foreground">
            Add this entry to your private Claude Code configuration, or use
            claude mcp add --transport http --scope user localfinance with this
            URL and the Authorization header.
          </p>
          <ConfigSnippet
            title="Other clients · Streamable HTTP"
            value={`URL: ${endpoint}\nTransport: Streamable HTTP\nAuthorization: Bearer YOUR_TOKEN`}
          />
          <p className="text-sm text-muted-foreground">
            Verify tool discovery in your AI client. Cloud-only clients cannot
            reach this computer’s loopback address. Rotating the token requires
            updating every client.
          </p>
        </CardContent>
      </Card>
      <Dialog open={showRotate} onOpenChange={setShowRotate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Replace the shared token?</DialogTitle>
            <DialogDescription>
              Every existing AI connection will lose access immediately. Save
              the new token and update each client configuration.
            </DialogDescription>
          </DialogHeader>
          {rotate.error && (
            <p role="alert" className="text-sm text-destructive">
              {rotate.error.message}
            </p>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowRotate(false)}>
              Cancel
            </Button>
            <Button disabled={rotate.isPending} onClick={() => rotate.mutate()}>
              {rotate.isPending ? 'Rotating…' : 'Rotate token'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
