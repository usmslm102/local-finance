import React, { useState, useEffect, useRef } from 'react'
import { Check, Copy, AlertCircle } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface CopyButtonProps {
  value: string
  label?: string
  className?: string
  size?: 'xs' | 'sm' | 'default'
}

export const CopyButton: React.FC<CopyButtonProps> = ({
  value,
  label = 'Copy to clipboard',
  className,
  size = 'xs',
}) => {
  const [status, setStatus] = useState<'idle' | 'copied' | 'error'>('idle')
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current)
    },
    []
  )
  const handleCopy = async (event: React.MouseEvent) => {
    event.stopPropagation()
    if (timer.current) clearTimeout(timer.current)
    try {
      await navigator.clipboard.writeText(value)
      setStatus('copied')
    } catch {
      setStatus('error')
    }
    timer.current = setTimeout(() => setStatus('idle'), 2500)
  }
  const message =
    status === 'copied'
      ? 'Copied'
      : status === 'error'
        ? 'Copy failed. Check clipboard permissions and try again.'
        : label
  return (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size={size === 'default' ? 'icon' : 'icon-sm'}
              onClick={handleCopy}
              aria-label={message}
              className={cn('shrink-0', className)}
            >
              {status === 'copied' ? <Check /> : status === 'error' ? <AlertCircle /> : <Copy />}
            </Button>
          }
        />
        <TooltipContent>{message}</TooltipContent>
      </Tooltip>
      <span className="sr-only" role="status">
        {status === 'idle' ? '' : message}
      </span>
    </>
  )
}
