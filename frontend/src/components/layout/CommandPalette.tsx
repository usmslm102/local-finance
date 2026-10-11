import React, { useEffect, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  LayoutDashboard,
  ReceiptText,
  Calendar,
  UploadCloud,
  Settings,
  Sparkles,
  CreditCard,
  Target,
  Store,
  ArrowLeftRight,
  Sun,
  Moon,
  Search,
  Layers,
  BookOpen,
  Trophy,
  Eye,
  EyeOff,
  Briefcase,
} from 'lucide-react'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { useTheme } from '@/components/theme-provider'
import { usePrivacy } from '@/components/privacy-provider'
import { Button } from '@/components/ui/button'

export const CommandPalette: React.FC = () => {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()
  const { isPrivacyMode, togglePrivacyMode } = usePrivacy()

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if ((e.key === 'k' && (e.metaKey || e.ctrlKey)) || e.key === '/') {
        if (
          (e.target instanceof HTMLElement && e.target.isContentEditable) ||
          e.target instanceof HTMLInputElement ||
          e.target instanceof HTMLTextAreaElement ||
          e.target instanceof HTMLSelectElement
        ) {
          return
        }

        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }

    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  const runCommand = (command: () => void) => {
    setOpen(false)
    command()
  }

  return (
    <>
      {/* Search trigger button for header */}
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
        aria-label="Search or jump to command"
        className="size-8 sm:w-44 md:w-56 justify-center sm:justify-between gap-2 px-2 sm:px-2.5 transition-all duration-150 active:scale-[0.98]"
      >
        <span className="flex items-center gap-1.5 truncate">
          <Search className="size-3.5 text-muted-foreground shrink-0" />
          <span className="truncate hidden sm:inline">Search or jump to...</span>
        </span>
        <kbd className="pointer-events-none hidden h-4.5 select-none items-center gap-0.5 rounded-sm border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100 sm:flex">
          <span className="text-xs">⌘</span>K
        </kbd>
      </Button>

      {/* Command Dialog */}
      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput placeholder="Type a command or search..." />
        <CommandList className="max-h-[320px]">
          <CommandEmpty>No results found.</CommandEmpty>

          <CommandGroup heading="Navigation">
            <CommandItem onSelect={() => runCommand(() => navigate({ to: '/planning' }))}>
              <Target className="size-4" />
              <span>Savings Forecast</span>
            </CommandItem>
            <CommandItem
              onSelect={() =>
                runCommand(() => navigate({ to: '/planning', search: { tab: 'goals' } }))
              }
            >
              <Target className="size-4" />
              <span>Savings Goals</span>
            </CommandItem>
            <CommandItem onSelect={() => runCommand(() => navigate({ to: '/data-health' }))}>
              <ReceiptText className="size-4" />
              <span>Data Health & Statement Coverage</span>
            </CommandItem>
            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <LayoutDashboard className="size-4 text-muted-foreground" />
              <span>Overview &amp; Dashboard</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/transactions' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <ReceiptText className="size-4 text-muted-foreground" />
              <span>Transactions Ledger</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/cashflow' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Layers className="size-4 text-muted-foreground" />
              <span>Cash Flow &amp; Sankey Diagram</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/salary' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Briefcase className="size-4 text-muted-foreground" />
              <span>Salary &amp; Income Insights</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/calendar' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Calendar className="size-4 text-muted-foreground" />
              <span>Spending Calendar</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/budget' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Target className="size-4 text-muted-foreground" />
              <span>Category Budgets &amp; Overspend</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/cards' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <CreditCard className="size-4 text-muted-foreground" />
              <span>Credit Cards &amp; Best-Card Engine</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/subscriptions' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Sparkles className="size-4 text-muted-foreground" />
              <span>Subscriptions &amp; Mandates</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/merchants' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Store className="size-4 text-muted-foreground" />
              <span>Merchant Intelligence</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/reconcile' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <ArrowLeftRight className="size-4 text-muted-foreground" />
              <span>Transfer Reconciler</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/wrapped' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Trophy className="size-4 text-warning" />
              <span>Year in Review &amp; LocalFinance Wrapped</span>
            </CommandItem>
          </CommandGroup>

          <CommandSeparator />

          <CommandGroup heading="Actions &amp; Data">
            <CommandItem
              onSelect={() => runCommand(() => togglePrivacyMode())}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              {isPrivacyMode ? (
                <Eye className="size-4 text-warning" />
              ) : (
                <EyeOff className="size-4 text-muted-foreground" />
              )}
              <span>
                {isPrivacyMode
                  ? 'Disable Discreet Mode (Reveal Figures)'
                  : 'Enable Discreet Mode (Mask Balances - Hotkey P)'}
              </span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/import' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <UploadCloud className="size-4 text-success" />
              <span>Import Bank Statement (PDF / CSV / Excel)</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/guide' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <BookOpen className="size-4 text-muted-foreground" />
              <span>User Guide &amp; Feature Walkthrough</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/whats-new' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Sparkles className="size-4 text-primary" />
              <span>What's New in LocalFinance (v1.2)</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => navigate({ to: '/settings' }))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              <Settings className="size-4 text-muted-foreground" />
              <span>Settings &amp; Auto-Categorization Rules</span>
            </CommandItem>

            <CommandItem
              onSelect={() => runCommand(() => setTheme(theme === 'dark' ? 'light' : 'dark'))}
              className="flex items-center gap-2.5 px-3 py-2 cursor-pointer"
            >
              {theme === 'dark' ? (
                <Sun className="size-4 text-warning" />
              ) : (
                <Moon className="size-4 text-info" />
              )}
              <span>Toggle Dark / Light Theme</span>
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </CommandDialog>
    </>
  )
}
