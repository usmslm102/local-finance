import React, { useState, useEffect, useRef } from 'react'
import { loginAuth } from '@/lib/api'
import { useAuth } from './AuthProvider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Lock, Unlock, Eye, EyeOff, ShieldCheck, AlertCircle, RefreshCw, KeyRound } from 'lucide-react'
import { playSuccessChime, playSoftClick } from '@/lib/audio'

export const LockScreen: React.FC = () => {
  const { unlock } = useAuth()
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    // Focus password field on mount
    inputRef.current?.focus()
  }, [])

  const handleUnlock = async (e?: React.FormEvent) => {
    if (e) e.preventDefault()
    if (!password.trim() || isLoading) return

    setIsLoading(true)
    setError(null)
    playSoftClick(500)

    try {
      await loginAuth(password)
      playSuccessChime()
      unlock()
    } catch (err: any) {
      setError(err.message || 'Incorrect master password')
      setIsLoading(false)
      inputRef.current?.select()
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-xl px-4 select-none">
      {/* Background ambient lighting */}
      <div className="absolute inset-0 bg-radial from-primary/10 via-background/40 to-background pointer-events-none" />

      <Card className="relative w-full max-w-md border-border/70 bg-card/95 shadow-2xl backdrop-blur-2xl overflow-hidden animate-in fade-in-50 zoom-in-95 duration-200">
        <div className="h-1.5 w-full bg-gradient-to-r from-primary/60 via-primary to-primary/60" />

        <CardContent className="p-6 sm:p-8 space-y-6 text-center">
          {/* Lock Icon Emblem */}
          <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary/10 border border-primary/20 text-primary shadow-inner">
            <Lock className="h-8 w-8 stroke-[2.2]" />
          </div>

          <div className="space-y-1.5">
            <div className="inline-flex items-center gap-1.5">
              <Badge variant="outline" className="text-[10px] font-mono border-primary/30 text-primary uppercase tracking-wider">
                <ShieldCheck className="h-3 w-3 mr-1 inline" /> 100% Offline App Lock
              </Badge>
            </div>
            <h2 className="text-xl sm:text-2xl font-bold tracking-tight text-foreground">
              LocalFinance is Locked
            </h2>
            <p className="text-xs text-muted-foreground max-w-xs mx-auto">
              Enter your master password or PIN to access your local financial ledger
            </p>
          </div>

          {/* Form */}
          <form onSubmit={handleUnlock} className="space-y-4 text-left">
            <div className="space-y-2">
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-muted-foreground">
                  <KeyRound className="h-4 w-4" />
                </div>
                <Input
                  ref={inputRef}
                  type={showPassword ? 'text' : 'password'}
                  placeholder="Master password or PIN..."
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value)
                    if (error) setError(null)
                  }}
                  disabled={isLoading}
                  className="pl-9 pr-10 text-sm h-11 bg-muted/40 font-mono tracking-wider focus-visible:ring-primary"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  tabIndex={-1}
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute inset-y-0 right-0 pr-3 flex items-center text-muted-foreground hover:text-foreground transition-colors"
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>

              {error && (
                <div className="flex items-center gap-1.5 text-xs text-destructive font-medium pt-1 animate-in fade-in slide-in-from-top-1">
                  <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                  <span>{error}</span>
                </div>
              )}
            </div>

            <Button
              type="submit"
              disabled={isLoading || !password.trim()}
              className="w-full h-10 font-semibold gap-2 shadow-sm text-sm"
            >
              {isLoading ? (
                <>
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  <span>Verifying...</span>
                </>
              ) : (
                <>
                  <Unlock className="h-4 w-4" />
                  <span>Unlock Ledger</span>
                </>
              )}
            </Button>
          </form>

          {/* Security Guarantee Details */}
          <div className="pt-2 border-t border-border/50 text-[11px] text-muted-foreground/80 space-y-1">
            <p>
              🔒 <strong>Local-only verification</strong> via bcrypt hash on your SQLite database.
            </p>
            <p className="text-[10px] text-muted-foreground/60">
              Your password never leaves this machine.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
