import React, { useState, useRef } from 'react'
import { loginAuth } from '@/lib/api'
import { useAuth } from './AuthProvider'
import { Button } from '@/components/ui/button'
import { Field, FieldLabel, FieldError, FieldGroup } from '@/components/ui/field'
import {
  InputGroup,
  InputGroupInput,
  InputGroupAddon,
  InputGroupButton,
} from '@/components/ui/input-group'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Lock, Unlock, Eye, EyeOff, RefreshCw, KeyRound } from 'lucide-react'
import { playSuccessChime } from '@/lib/audio'

export const LockScreen: React.FC = () => {
  const { unlock } = useAuth()
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const handleUnlock = async (event: React.FormEvent) => {
    event.preventDefault()
    if (!password.trim() || isLoading) return
    setIsLoading(true)
    setError(null)
    try {
      await loginAuth(password)
      playSuccessChime()
      unlock()
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Unable to unlock. Try again.')
      setIsLoading(false)
      inputRef.current?.select()
    }
  }
  return (
    <Dialog
      open
      onOpenChange={() => {
        /* The lock stays open until authentication succeeds. */
      }}
    >
      <DialogContent showCloseButton={false} className="max-w-md">
        <DialogHeader>
          <Lock className="mb-2 size-8 text-primary" />
          <DialogTitle>Unlock LocalFinance</DialogTitle>
          <DialogDescription>
            Enter your master password or PIN to open your financial ledger.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleUnlock}>
          <FieldGroup>
            <Field data-invalid={!!error}>
              <FieldLabel htmlFor="unlock-password">Master password or PIN</FieldLabel>
              <InputGroup>
                <InputGroupAddon>
                  <KeyRound />
                </InputGroupAddon>
                <InputGroupInput
                  id="unlock-password"
                  ref={inputRef}
                  type={showPassword ? 'text' : 'password'}
                  autoFocus
                  autoComplete="current-password"
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value)
                    setError(null)
                  }}
                  disabled={isLoading}
                  aria-invalid={!!error}
                  aria-describedby={error ? 'unlock-error' : undefined}
                />
                <InputGroupAddon align="inline-end">
                  <InputGroupButton
                    size="icon-sm"
                    aria-label={showPassword ? 'Hide password' : 'Show password'}
                    aria-pressed={showPassword}
                    onClick={() => setShowPassword(!showPassword)}
                  >
                    {showPassword ? <EyeOff /> : <Eye />}
                  </InputGroupButton>
                </InputGroupAddon>
              </InputGroup>
              {error && <FieldError id="unlock-error">{error}</FieldError>}
            </Field>
            <Button type="submit" disabled={isLoading || !password.trim()}>
              {isLoading ? (
                <RefreshCw data-icon="inline-start" className="animate-spin" />
              ) : (
                <Unlock data-icon="inline-start" />
              )}
              {isLoading ? 'Verifying…' : 'Unlock ledger'}
            </Button>
          </FieldGroup>
        </form>
        <p className="text-xs text-muted-foreground">Your password is verified on this machine.</p>
      </DialogContent>
    </Dialog>
  )
}
