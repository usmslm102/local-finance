import React, { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Sparkles,
  ArrowUpCircle,
  ExternalLink,
  ShieldCheck,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Download,
  Database,
  RefreshCw,
} from 'lucide-react'
import type { SystemVersionInfo, ApplyUpdateResponse } from '@/types'
import { applySystemUpdate, fetchSystemVersion } from '@/lib/api'

interface UpdateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  versionInfo: SystemVersionInfo | null
  onUpdateSuccess?: () => void
}

type UpdateStage = 'idle' | 'updating' | 'restarting' | 'done' | 'error'

export const UpdateDialog: React.FC<UpdateDialogProps> = ({
  open,
  onOpenChange,
  versionInfo,
  onUpdateSuccess,
}) => {
  const [stage, setStage] = useState<UpdateStage>('idle')
  const [errorMsg, setErrorMsg] = useState<string>('')
  const [updateResult, setUpdateResult] = useState<ApplyUpdateResponse | null>(null)
  const [restartSeconds, setRestartSeconds] = useState<number>(10)

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setStage('idle')
      setErrorMsg('')
      setUpdateResult(null)
      setRestartSeconds(10)
    }
  }, [open])

  // Polling loop once restarting has begun
  useEffect(() => {
    if (stage !== 'restarting') return

    let intervalId: any
    let pollCount = 0

    intervalId = setInterval(async () => {
      pollCount++
      setRestartSeconds((prev) => Math.max(0, prev - 1))

      try {
        const check = await fetchSystemVersion(true)
        // If the server is back up and version updated, finish!
        if (check.current_version === versionInfo?.latest_version || pollCount >= 8) {
          clearInterval(intervalId)
          setStage('done')
          setTimeout(() => {
            window.location.reload()
          }, 1500)
        }
      } catch {
        // Server is restarting, expected to fail momentarily
      }
    }, 1500)

    return () => clearInterval(intervalId)
  }, [stage, versionInfo?.latest_version])

  const handleApplyUpdate = async () => {
    setStage('updating')
    setErrorMsg('')

    try {
      const res = await applySystemUpdate()
      setUpdateResult(res)
      setStage('restarting')
      onUpdateSuccess?.()
    } catch (err: any) {
      setStage('error')
      setErrorMsg(err.message || 'Failed to apply update.')
    }
  }

  if (!versionInfo) return null

  return (
    <Dialog open={open} onOpenChange={stage === 'updating' || stage === 'restarting' ? undefined : onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[90vh] flex flex-col p-6">
        <DialogHeader>
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Sparkles className="h-5 w-5" />
            </div>
            <div>
              <DialogTitle className="text-lg font-bold flex items-center gap-2">
                New Release Available
                <Badge variant="default" className="text-xs px-2 py-0.5">
                  {versionInfo.latest_version}
                </Badge>
              </DialogTitle>
              <DialogDescription className="text-xs text-muted-foreground mt-0.5">
                Current installed version: <span className="font-mono font-medium text-foreground">{versionInfo.current_version}</span>
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {/* Content area */}
        <div className="space-y-4 my-2 overflow-y-auto max-h-[50vh] pr-1 text-sm">
          {/* Release Notes Card */}
          {versionInfo.release_notes ? (
            <div className="rounded-lg border bg-muted/30 p-3.5 space-y-2">
              <div className="flex items-center justify-between text-xs font-semibold text-muted-foreground">
                <span>Release Highlights ({versionInfo.release_name || versionInfo.latest_version})</span>
                {versionInfo.published_at && (
                  <span className="font-normal">{new Date(versionInfo.published_at).toLocaleDateString()}</span>
                )}
              </div>
              <div className="text-xs font-mono whitespace-pre-wrap text-foreground/90 max-h-48 overflow-y-auto leading-relaxed bg-background/60 p-2.5 rounded border">
                {versionInfo.release_notes}
              </div>
            </div>
          ) : (
            <div className="text-xs text-muted-foreground italic">
              No release notes provided for this version.
            </div>
          )}

          {/* Safe Backup Notice */}
          <div className="flex items-start gap-2.5 rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-3 text-xs text-emerald-800 dark:text-emerald-300">
            <ShieldCheck className="h-4 w-4 shrink-0 mt-0.5 text-emerald-600 dark:text-emerald-400" />
            <div>
              <p className="font-medium text-emerald-950 dark:text-emerald-200">
                Safe 100% Offline SQLite Backup
              </p>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                Before updating, an automatic snapshot of your entire database is created in <span className="font-mono text-foreground">~/.localfinance/backups/</span>.
              </p>
            </div>
          </div>

          {/* Status Message during updates */}
          {stage === 'updating' && (
            <div className="flex flex-col items-center justify-center p-6 space-y-3 bg-muted/40 rounded-xl border border-primary/20">
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
              <div className="text-center space-y-1">
                <p className="text-sm font-semibold">Downloading & Applying Update...</p>
                <p className="text-xs text-muted-foreground">
                  Validating SHA256 checksum and updating binary in place.
                </p>
              </div>
            </div>
          )}

          {stage === 'restarting' && (
            <div className="flex flex-col items-center justify-center p-6 space-y-3 bg-primary/5 rounded-xl border border-primary/20">
              <RefreshCw className="h-8 w-8 animate-spin text-primary" />
              <div className="text-center space-y-1">
                <p className="text-sm font-semibold text-primary">Restarting LocalFinance Server</p>
                <p className="text-xs text-muted-foreground">
                  The application will automatically refresh in {restartSeconds}s once the server comes back online.
                </p>
                {updateResult?.backup_path && (
                  <p className="text-[11px] text-muted-foreground mt-2 font-mono truncate max-w-sm">
                    Backup saved: {updateResult.backup_path}
                  </p>
                )}
              </div>
            </div>
          )}

          {stage === 'done' && (
            <div className="flex flex-col items-center justify-center p-6 space-y-3 bg-emerald-500/10 rounded-xl border border-emerald-500/30">
              <CheckCircle2 className="h-8 w-8 text-emerald-500" />
              <div className="text-center space-y-1">
                <p className="text-sm font-semibold text-emerald-700 dark:text-emerald-400">Update Installed Successfully!</p>
                <p className="text-xs text-muted-foreground">Reloading application...</p>
              </div>
            </div>
          )}

          {stage === 'error' && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Update Failed</AlertTitle>
              <AlertDescription className="text-xs mt-1">
                {errorMsg}
                <div className="mt-2">
                  <a
                    href={versionInfo.release_url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 font-semibold underline hover:text-foreground"
                  >
                    Download release manually from GitHub <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
              </AlertDescription>
            </Alert>
          )}

          {/* If auto-update is not possible (e.g. running from DMG or no permissions) */}
          {!versionInfo.can_auto_update && stage === 'idle' && (
            <Alert className="border-amber-500/30 bg-amber-500/10 text-amber-900 dark:text-amber-200">
              <AlertCircle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
              <AlertTitle className="text-xs font-semibold">Manual Update Required</AlertTitle>
              <AlertDescription className="text-xs mt-1">
                {versionInfo.auto_update_error ||
                  'The application cannot be replaced automatically in its current location.'}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter className="mt-2 sm:justify-between items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.open(versionInfo.release_url, '_blank')}
            className="text-xs text-muted-foreground hover:text-foreground h-9 gap-1.5"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            Release on GitHub
          </Button>

          <div className="flex items-center gap-2">
            {stage === 'idle' && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onOpenChange(false)}
                  className="text-xs h-9"
                >
                  Later
                </Button>

                {versionInfo.can_auto_update ? (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={handleApplyUpdate}
                    className="text-xs font-semibold h-9 gap-1.5 bg-primary text-primary-foreground shadow-sm hover:bg-primary/90"
                  >
                    <ArrowUpCircle className="h-4 w-4" />
                    Update &amp; Restart
                  </Button>
                ) : (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => window.open(versionInfo.asset_url || versionInfo.release_url, '_blank')}
                    className="text-xs font-semibold h-9 gap-1.5"
                  >
                    <Download className="h-4 w-4" />
                    Download Release
                  </Button>
                )}
              </>
            )}

            {stage === 'error' && (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onOpenChange(false)}
                  className="text-xs h-9"
                >
                  Close
                </Button>
                {versionInfo.can_auto_update && (
                  <Button
                    variant="default"
                    size="sm"
                    onClick={handleApplyUpdate}
                    className="text-xs h-9 gap-1.5"
                  >
                    <RefreshCw className="h-3.5 w-3.5" />
                    Retry Update
                  </Button>
                )}
              </>
            )}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
