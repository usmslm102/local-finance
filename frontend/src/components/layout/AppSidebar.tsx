import React from 'react'
import { useSystemUpdate } from '@/hooks/use-system-update'
import { UpdateDialog } from '@/components/updates/UpdateDialog'
import { ArrowUpCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Link, useRouterState } from '@tanstack/react-router'
import {
  LayoutDashboard,
  ReceiptText,
  Calendar,
  UploadCloud,
  Settings,
  ShieldCheck,
  Sparkles,
  CreditCard,
  Target,
  Store,
  ArrowLeftRight,
  Database,
  Layers,
  BookOpen,
  Trophy,
  Briefcase,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { ModeToggle } from '@/components/mode-toggle'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from '@/components/ui/sidebar'

interface NavItem {
  title: string
  url: string
  icon: React.ComponentType<{ className?: string }>
  badge?: string
}

const mainNavItems: NavItem[] = [
  {
    title: 'Overview',
    url: '/',
    icon: LayoutDashboard,
  },
  {
    title: 'Transactions',
    url: '/transactions',
    icon: ReceiptText,
  },
  {
    title: 'Calendar',
    url: '/calendar',
    icon: Calendar,
  },
]

const intelligenceNavItems: NavItem[] = [
  {
    title: 'Cash Flow & Sankey',
    url: '/cashflow',
    icon: Layers,
  },
  {
    title: 'Salary & Income',
    url: '/salary',
    icon: Briefcase,
  },
  {
    title: 'Budgets & Overspend',
    url: '/budget',
    icon: Target,
  },
  {
    title: 'Cards & Rewards',
    url: '/cards',
    icon: CreditCard,
  },
  {
    title: 'Subscriptions',
    url: '/subscriptions',
    icon: Sparkles,
  },
  {
    title: 'Merchant Intelligence',
    url: '/merchants',
    icon: Store,
  },
  {
    title: 'Transfer Reconciler',
    url: '/reconcile',
    icon: ArrowLeftRight,
  },
  {
    title: 'Year in Review',
    url: '/wrapped',
    icon: Trophy,
    badge: 'Wrapped',
  },
]

const systemNavItems: NavItem[] = [
  {
    title: 'Import Statements',
    url: '/import',
    icon: UploadCloud,
  },
  {
    title: 'Settings & Rules',
    url: '/settings',
    icon: Settings,
  },
]

const helpNavItems: NavItem[] = [
  {
    title: 'User Guide',
    url: '/guide',
    icon: BookOpen,
  },
  {
    title: "What's New",
    url: '/whats-new',
    icon: Sparkles,
    badge: 'v1.2',
  },
]

export const AppSidebar: React.FC = () => {
  const routerState = useRouterState()
  const currentPath = routerState.location.pathname
  const { versionInfo, dialogOpen, setDialogOpen, refetch } = useSystemUpdate()

  return (
    <Sidebar collapsible="icon" className="border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
      {/* Sidebar Header: Brand - strictly h-14 with bottom border aligning with main header */}
      <SidebarHeader className="h-14 border-b border-sidebar-border px-3.5 flex justify-center group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:justify-center">
        <Link
          to="/"
          className="flex items-center gap-3 rounded-lg p-1 transition-colors hover:bg-sidebar-accent group-data-[collapsible=icon]:p-0 group-data-[collapsible=icon]:justify-center"
        >
          <img
            src="/favicon.svg"
            alt="LocalFinance Logo"
            className="h-8 w-8 shrink-0 rounded-lg shadow-xs select-none"
          />
          <div className="flex flex-col overflow-hidden group-data-[collapsible=icon]:hidden">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-bold tracking-tight text-foreground truncate">
                LocalFinance
              </span>
              <Badge
                variant="outline"
                className="px-1 py-0 text-[9px] font-semibold uppercase tracking-wider text-emerald-600 dark:text-emerald-400 border-emerald-500/30"
              >
                Offline
              </Badge>
            </div>
            <p className="text-[10px] text-muted-foreground flex items-center gap-1 font-medium truncate">
              <ShieldCheck className="h-3 w-3 text-emerald-500 shrink-0" />
              100% Local SQLite
            </p>
          </div>
        </Link>
      </SidebarHeader>

      {/* Sidebar Navigation Groups */}
      <SidebarContent className="px-2 py-3 space-y-4">
        {/* Main Ledger */}
        <SidebarGroup className="p-0">
          <SidebarGroupLabel className="px-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground group-data-[collapsible=icon]:hidden">
            Main Ledger
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {mainNavItems.map((item) => {
                const Icon = item.icon
                const isActive = currentPath === item.url
                return (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      render={<Link to={item.url} />}
                      isActive={isActive}
                      tooltip={item.title}
                      className={`h-9 gap-3 rounded-md px-2.5 text-xs font-medium transition-colors ${
                        isActive
                          ? 'bg-sidebar-accent text-sidebar-accent-foreground font-semibold shadow-xs'
                          : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground'
                      }`}
                    >
                      <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                      <span className="truncate">{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Financial Intelligence */}
        <SidebarGroup className="p-0">
          <SidebarGroupLabel className="px-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground group-data-[collapsible=icon]:hidden">
            Intelligence
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {intelligenceNavItems.map((item) => {
                const Icon = item.icon
                const isActive = currentPath === item.url
                return (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      render={<Link to={item.url} />}
                      isActive={isActive}
                      tooltip={item.title}
                      className={`h-9 gap-3 rounded-md px-2.5 text-xs font-medium transition-colors ${
                        isActive
                          ? 'bg-sidebar-accent text-sidebar-accent-foreground font-semibold shadow-xs'
                          : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground'
                      }`}
                    >
                      <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                      <span className="truncate">{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Data & Settings */}
        <SidebarGroup className="p-0">
          <SidebarGroupLabel className="px-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground group-data-[collapsible=icon]:hidden">
            Data &amp; Tools
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {systemNavItems.map((item) => {
                const Icon = item.icon
                const isActive = currentPath === item.url
                return (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      render={<Link to={item.url} />}
                      isActive={isActive}
                      tooltip={item.title}
                      className={`h-9 gap-3 rounded-md px-2.5 text-xs font-medium transition-colors ${
                        isActive
                          ? 'bg-sidebar-accent text-sidebar-accent-foreground font-semibold shadow-xs'
                          : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground'
                      }`}
                    >
                      <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                      <span className="truncate">{item.title}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        {/* Help & Resources */}
        <SidebarGroup className="p-0">
          <SidebarGroupLabel className="px-2 text-[10px] font-bold uppercase tracking-wider text-muted-foreground group-data-[collapsible=icon]:hidden">
            Help &amp; Updates
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {versionInfo?.update_available && (
                <SidebarMenuItem>
                  <SidebarMenuButton
                    onClick={() => setDialogOpen(true)}
                    tooltip={`Update Available: ${versionInfo.latest_version}`}
                    className="h-9 gap-3 rounded-md px-2.5 text-xs font-semibold transition-colors bg-primary/10 text-primary hover:bg-primary/20"
                  >
                    <ArrowUpCircle className="h-4 w-4 shrink-0 text-primary" />
                    <span className="truncate">Update to {versionInfo.latest_version}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              )}
              {helpNavItems.map((item) => {
                const Icon = item.icon
                const isActive = currentPath === item.url
                return (
                  <SidebarMenuItem key={item.url}>
                    <SidebarMenuButton
                      render={<Link to={item.url} />}
                      isActive={isActive}
                      tooltip={item.title}
                      className={`h-9 gap-3 rounded-md px-2.5 text-xs font-medium transition-colors ${
                        isActive
                          ? 'bg-sidebar-accent text-sidebar-accent-foreground font-semibold shadow-xs'
                          : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-sidebar-foreground'
                      }`}
                    >
                      <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary' : 'text-muted-foreground'}`} />
                      <span className="truncate">{item.title}</span>
                      {item.badge && (
                        <span className="ml-auto rounded-sm bg-primary/10 px-1.5 py-0 text-[9px] font-semibold text-primary group-data-[collapsible=icon]:hidden">
                          {item.badge}
                        </span>
                      )}
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )
              })}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      {/* Sidebar Footer: System Status & Dark Mode Toggle */}
      <SidebarFooter className="border-t border-sidebar-border p-3 group-data-[collapsible=icon]:p-2 group-data-[collapsible=icon]:justify-center">
        <div className="flex items-center justify-between gap-2 group-data-[collapsible=icon]:justify-center">
          <div className="flex items-center gap-2 overflow-hidden group-data-[collapsible=icon]:hidden">
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-emerald-500/10 text-emerald-500">
              <Database className="h-3.5 w-3.5" />
            </div>
            <div className="flex flex-col overflow-hidden">
              <span className="text-xs font-semibold text-foreground truncate">
                local_finance.db
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setDialogOpen(true)}
                className="h-auto p-0 text-[10px] text-muted-foreground hover:text-primary hover:bg-transparent transition-colors justify-start font-normal gap-1 cursor-pointer"
              >
                <span>{versionInfo?.current_version || 'v1.2.0'}</span>
                {versionInfo?.update_available && (
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-primary animate-pulse" />
                )}
                <span>• Local Only</span>
              </Button>
            </div>
          </div>
          <ModeToggle />
        </div>
      </SidebarFooter>

      <UpdateDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        versionInfo={versionInfo || null}
        onUpdateSuccess={() => refetch()}
      />

      {/* Drag & Hover Rail for Collapsing */}
      <SidebarRail />
    </Sidebar>
  )
}
