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
  Download,
  CheckCircle2,
  AlertCircle,
  Loader2,
  RefreshCw,
  ExternalLink,
  ShieldCheck,
  ArrowUpCircle,
} from 'lucide-react'
import { applySystemUpdate, fetchSystemVersion } from '@/lib/api'
import type { SystemVersionInfo, ApplyUpdateResponse } from '@/types'

interface UpdateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  versionInfo: SystemVersionInfo | null
  onUpdateSuccess?: () => void
}

type UpdateStage = 'idle' | 'updating' | 'restarting' | 'done' | 'restart_timeout' | 'error'

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
  const [pollAttempt, setPollAttempt] = useState<number>(0)

  // Reset state when dialog opens
  useEffect(() => {
    if (open) {
      setStage('idle')
      setErrorMsg('')
      setUpdateResult(null)
      setRestartSeconds(10)
      setPollAttempt(0)
    }
  }, [open])

  // Polling loop once restarting has begun
  useEffect(() => {
    if (stage !== 'restarting') return

    let intervalId: any
    let count = 0
    const MAX_POLLS = 20 // 20 polls * 1.5s = 30 seconds max before timeout

    intervalId = setInterval(async () => {
      count++
      setPollAttempt(count)
      setRestartSeconds((prev) => Math.max(0, prev - 1))

      try {
        // Poll local server strictly offline (no GitHub network request needed during restart)
        const check = await fetchSystemVersion(false, true)
        const expectedVersion = updateResult?.new_version || versionInfo?.latest_version
        // Strictly verify that the server has restarted and reports the newly installed version!
        if (expectedVersion && check.current_version === expectedVersion) {
          clearInterval(intervalId)
          setStage('done')
          setTimeout(() => {
            window.location.reload()
          }, 1500)
          return
        }
      } catch {
        // Expected temporary connection failure while old process exits and new process binds port
      }

      if (count >= MAX_POLLS) {
        clearInterval(intervalId)
        setStage('restart_timeout')
      }
    }, 1500)

    return () => clearInterval(intervalId)
  }, [stage, updateResult?.new_version, versionInfo?.latest_version])

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

  const handleRetryRestartPoll = () => {
    setStage('restarting')
    setRestartSeconds(10)
    setPollAttempt(0)
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
              <DialogTitle className="text-base font-semibold">
                Software Update Available
              </DialogTitle>
              <DialogDescription className="text-xs">
                {versionInfo.release_name || `Release ${versionInfo.latest_version}`} is ready to install
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        {/* Version Details */}
        <div className="space-y-4 my-2 overflow-y-auto pr-1 text-sm">
          <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-3">
            <div className="space-y-0.5">
              <p className="text-[11px] font-medium text-muted-foreground">Current Installed</p>
              <p className="font-mono text-xs font-semibold">{versionInfo.current_version}</p>
            </div>
            <div className="text-muted-foreground font-mono">→</div>
            <div className="space-y-0.5 text-right">
              <p className="text-[11px] font-medium text-muted-foreground">Latest Available</p>
              <Badge variant="default" className="font-mono text-xs bg-primary text-primary-foreground">
                {versionInfo.latest_version}
              </Badge>
            </div>
          </div>

          {/* Release Highlights / Changelog */}
          {versionInfo.release_notes ? (
            <div className="space-y-1.5">
              <p className="text-xs font-semibold text-foreground">What's New in this Release</p>
              <div className="rounded-lg border bg-muted/20 p-3 text-xs text-muted-foreground whitespace-pre-wrap max-h-40 overflow-y-auto font-sans leading-relaxed selection:bg-primary/20">
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
                <p className="text-sm font-semibold">Downloading &amp; Applying Update...</p>
                <p className="text-xs text-muted-foreground">
                  Validating SHA256 checksum against release manifest and updating binary in place.
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
                  Waiting for updated server to respond ({restartSeconds}s, attempt {pollAttempt}/20)...
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
                <p className="text-xs text-muted-foreground">Server upgraded to {versionInfo.latest_version}. Reloading application...</p>
              </div>
            </div>
          )}

          {stage === 'restart_timeout' && (
            <Alert className="border-amber-500/30 bg-amber-500/10 text-amber-900 dark:text-amber-200">
              <AlertCircle className="h-4 w-4 text-amber-600 dark:text-amber-400" />
              <AlertTitle className="text-xs font-semibold">Reconnection Taking Longer than Expected</AlertTitle>
              <AlertDescription className="text-xs mt-1 space-y-2">
                <p>
                  The binary update was applied, but the web interface hasn't reconnected after 30 seconds.
                  If the process did not restart automatically, please start it from your terminal or launcher.
                </p>
                <div className="flex items-center gap-2 pt-1">
                  <Button size="sm" variant="outline" onClick={handleRetryRestartPoll} className="h-7 text-xs">
                    Retry Connection
                  </Button>
                  <Button size="sm" variant="secondary" onClick={() => window.location.reload()} className="h-7 text-xs">
                    Reload Page
                  </Button>
                </div>
              </AlertDescription>
            </Alert>
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
