import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, ShieldCheck } from 'lucide-react'
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
import { MCPConnectionSetup } from './MCPConnectionSetup'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

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
  const [showAdvanced, setShowAdvanced] = useState(false)
  const save = useMutation({
    mutationFn: ({ enabled, port, allowCategorizationWrites }: { enabled: boolean; port: number; allowCategorizationWrites?: boolean }) =>
      updateMCPSettings(enabled, port, allowCategorizationWrites),
    onSuccess: (settings) => {
      client.setQueryData(['mcp-settings'], settings)
      setPortDraft(null)
    },
  })
  const rotate = useMutation({
    mutationFn: rotateMCPToken,
    onSuccess: (result) => {
      const firstSetup = !query.data?.has_token
      setToken(result.token)
      client.setQueryData(['mcp-settings'], result.settings)
      setShowRotate(false)
      if (firstSetup && !result.settings.enabled) save.mutate({ enabled: true, port: result.settings.port })
    },
  })
  const settings = query.data
  const busy = save.isPending || rotate.isPending
  const port = Number(portDraft ?? settings?.port ?? 8081)
  const validPort = Number.isInteger(port) && port >= 1024 && port <= 65535
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
                ? settings.allow_categorization_writes
                  ? 'Listening · category and rule writes'
                  : 'Listening · read-only'
                : settings.enabled
                  ? 'Unavailable'
                  : 'Disabled'}
            </Badge>
          </div>
          <CardDescription>
            Connect your AI tool in a few steps: enable access, choose a client,
            then copy and run its setup command.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <Alert>
            <ShieldCheck />
            <AlertTitle>Choose what you share</AlertTitle>
            <AlertDescription>
              Connected AI tools can read your financial data and may send it
              to their model provider. Access stays available while the app is
              locked. Disable MCP to stop access.
            </AlertDescription>
          </Alert>
          <div className="space-y-2">
            <p className="text-sm">
              One shared token grants finance reads to all connected AI tools,
              plus category and rule writes when allowed above.
            </p>
            <Button
              variant={settings.has_token ? "outline" : "default"}
              disabled={busy}
              onClick={() =>
                settings.has_token ? setShowRotate(true) : rotate.mutate()
              }
            >
              {rotate.isPending
                ? 'Creating…'
                : settings.has_token
                  ? 'Rotate access token'
                  : 'Create token & enable MCP'}
            </Button>
            {token && <p role="status" className="text-sm text-muted-foreground">Token ready. It’s included in the setup commands below.</p>}
          </div>
          <div className="flex items-center justify-between gap-4">
            <div>
              <Label htmlFor="mcp-enabled">Enable MCP</Label>
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
          <Button variant="ghost" size="sm" onClick={() => setShowAdvanced(!showAdvanced)} aria-expanded={showAdvanced} aria-controls="mcp-advanced">
            {showAdvanced ? 'Hide advanced settings' : 'Advanced settings & permissions'}
          </Button>
          {showAdvanced && <div id="mcp-advanced" className="space-y-5">
          <div className="flex items-center justify-between gap-4">
            <div>
              <Label htmlFor="mcp-categorization-writes">Allow category and rule writes</Label>
              <p className="text-sm text-muted-foreground">
                All clients sharing your token can create or update custom categories
                and categorization rules. Existing transactions stay unchanged;
                re-apply rules to the ledger through the app.
              </p>
            </div>
            <Switch
              id="mcp-categorization-writes"
              checked={settings.allow_categorization_writes}
              disabled={busy}
              onCheckedChange={(allowCategorizationWrites) =>
                save.mutate({ enabled: settings.enabled, port: settings.port, allowCategorizationWrites })
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
          </div>}
          {settings.error && (
            <Alert variant="destructive">
              <AlertCircle />
              <AlertTitle>MCP listener unavailable</AlertTitle>
              <AlertDescription>{settings.error}</AlertDescription>
            </Alert>
          )}
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
            Choose your client, copy one command, and run it on this computer.
            No configuration editing or separate token copying needed.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <MCPConnectionSetup endpoint={endpoint} token={token} setToken={setToken} listening={settings.listening} />
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
