import { cn } from '@/lib/utils'
import React from 'react'
import { usePrivacy } from '@/components/privacy-provider'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

export const PrivacyToggle: React.FC = () => {
  const { isPrivacyMode, togglePrivacyMode } = usePrivacy()

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant={isPrivacyMode ? 'secondary' : 'ghost'}
            size="sm"
            onClick={togglePrivacyMode}
            className={cn(
              'h-8 px-2 sm:px-2.5 gap-1.5 text-xs transition-all',
              isPrivacyMode
                ? 'bg-warning/15 text-warning border border-warning/30 hover:bg-warning/25'
                : 'text-muted-foreground hover:text-foreground'
            )}
            aria-label={isPrivacyMode ? 'Disable Discreet Mode' : 'Enable Discreet Mode'}
          >
            {isPrivacyMode ? (
              <EyeOff className="size-3.5 text-warning shrink-0" />
            ) : (
              <Eye className="size-3.5 shrink-0" />
            )}
            <span className="hidden md:inline font-medium">
              {isPrivacyMode ? 'Discreet' : 'Public'}
            </span>
            <kbd className="hidden lg:inline-flex h-4 select-none items-center gap-0.5 rounded border border-border/80 bg-muted/60 px-1 font-mono text-[9px] font-medium text-muted-foreground">
              P
            </kbd>
          </Button>
        }
      />
      <TooltipContent side="bottom" align="end" className="text-xs">
        <p className="font-semibold">
          {isPrivacyMode ? 'Discreet Mode Active' : 'Enable Discreet Mode'}
        </p>
        <p className="text-[11px] text-muted-foreground">
          {isPrivacyMode
            ? 'Balances are blurred/masked. Press P or ⌘. to restore.'
            : 'Mask balances & numbers in public spaces (Hotkey: P)'}
        </p>
      </TooltipContent>
    </Tooltip>
  )
}
