import { cn } from '@/lib/utils'
import { buttonVariants } from '@/components/ui/button'
import React from 'react'
import { Link } from '@tanstack/react-router'
import {
  Sparkles,
  Layers,
  ArrowLeftRight,
  CreditCard,
  UploadCloud,
  CheckCircle2,
  ArrowRight,
  ShieldCheck,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface ReleaseNote {
  version: string
  title: string
  date: string
  isLatest?: boolean
  badge: string
  icon: React.ComponentType<{ className?: string }>
  iconColor: string
  summary: string
  features: {
    title: string
    description: string
    tag: string
  }[]
  primaryCta?: {
    text: string
    url: string
  }
  secondaryCta?: {
    text: string
    url: string
  }
}

const releases: ReleaseNote[] = [
  {
    version: 'v1.2',
    title: 'Cash Flow Sankey & MoM Anomaly Intelligence',
    date: 'Latest Release',
    isLatest: true,
    badge: 'Current Version',
    icon: Layers,
    iconColor: '#10B981',
    summary:
      'Added interactive multi-tier cash flow topology diagrams, automatic month-over-month spending surge detection, and 6-month category trajectory sparklines.',
    features: [
      {
        title: 'Multi-Tier Interactive Sankey Topology',
        description:
          'Visualizes full money flow routing from Income Sources → Holding Bank Accounts → Credit Cards & Direct Channels → Categories + Net Savings Surplus with hover highlights and tooltips.',
        tag: 'Visuals',
      },
      {
        title: 'MoM Anomaly & Shift Sniffer',
        description:
          'Automated local heuristic engine flagging category surges (≥30% increase with >₹1,000 delta), significant spending drops, new category outflows, and high-savings milestones.',
        tag: 'Intelligence',
      },
      {
        title: '6-Month Category Trajectory Sparklines',
        description:
          'Comparative analysis table comparing current month spend against previous month and 3-month rolling averages with mini SVG sparklines.',
        tag: 'Analytics',
      },
      {
        title: '1-Click Flow Drilldown Dialog',
        description:
          'Click any node or link in the Sankey diagram to inspect the exact individual ledger transactions contributing to that flow.',
        tag: 'UX',
      },
    ],
    primaryCta: {
      text: 'Try Cash Flow Sankey',
      url: '/cashflow',
    },
  },
  {
    version: 'v1.1',
    title: 'Smart Transfer Reconciler & Merchant Intelligence',
    date: 'Major Update',
    badge: 'Reconciliation & Profiles',
    icon: ArrowLeftRight,
    iconColor: '#14B8A6',
    summary:
      'Introduced intelligent cross-account credit card bill payment pairing to eliminate double-counted expenses, UPI Lite auto-exclusion, and merchant lifetime profiles.',
    features: [
      {
        title: 'Cross-Account Transfer Auto-Pairing',
        description:
          'Heuristic engine matches bank account debits (Cred, BBPS, NetBanking, IMPS) with credit card credits within 4 days, preventing double-counted monthly outflow totals.',
        tag: 'Reconciliation',
      },
      {
        title: 'UPI Lite & Wallet Load Auto-Exclusion',
        description:
          'Automatically flags and excludes intermediate wallet loads and UPI Lite top-ups from expense totals with 1-click inclusion toggles.',
        tag: 'Accuracy',
      },
      {
        title: 'Dedicated Merchant Profiles',
        description:
          'Inspect lifetime spending, transaction counts, average order values (AOV), and payment method shares for merchants like Swiggy, Amazon, Blinkit, and Uber.',
        tag: 'Profiles',
      },
      {
        title: '1-Click Auto-Rule Builder',
        description:
          'Tap "Create Auto-Rule from Payee" in the transaction modal to instantly prefill regex patterns and map payees to categories.',
        tag: 'Automation',
      },
    ],
    primaryCta: {
      text: 'Open Transfer Reconciler',
      url: '/reconcile',
    },
    secondaryCta: {
      text: 'Inspect Merchants',
      url: '/merchants',
    },
  },
  {
    version: 'v1.0',
    title: 'Card Reward Optimizer & Category Budgeting',
    date: 'Milestone Release',
    badge: 'Rewards & Budgets',
    icon: CreditCard,
    iconColor: '#EC4899',
    summary:
      'Launched the interactive Best-Card recommendation engine, 50-day interest-free grace period visualizer, category budgeting pacing bars, and recurring subscriptions tracker.',
    features: [
      {
        title: 'Interactive Best-Card Recommender',
        description:
          'Search any merchant (Swiggy, Amazon, Flipkart, Blinkit, Petrol) to identify which card in your portfolio yields the highest reward rate and estimated cashbacks.',
        tag: 'Cards',
      },
      {
        title: '50-Day Interest-Free Runway Optimizer',
        description:
          'Visual timeline calculating exact remaining interest-free runways across cards based on statement generation and due dates to maximize cash float.',
        tag: 'Optimizer',
      },
      {
        title: 'Category Monthly Budget Limits & Pacing',
        description:
          'Set custom monthly targets per category with dynamic Green → Amber → Red pacing bars and live Daily Recommended Allowance calculations.',
        tag: 'Budgeting',
      },
      {
        title: 'Subscriptions & Mandates Auto-Detection',
        description:
          'Multi-tier sniffer capturing UPI AutoPay mandates, OTT subscriptions, telecom bills, and loan EMIs with renewal countdowns and monthly burn rates.',
        tag: 'Recurring',
      },
    ],
    primaryCta: {
      text: 'Optimize Card Rewards',
      url: '/cards',
    },
    secondaryCta: {
      text: 'Manage Budgets',
      url: '/budget',
    },
  },
  {
    version: 'v0.9',
    title: 'Core Ingestion & Narration Intelligence Engine',
    date: 'Foundation',
    badge: 'Offline Core',
    icon: UploadCloud,
    iconColor: '#3B82F6',
    summary:
      'Built the modular bank adapter architecture, Indian UPI regex engine, password-decrypted PDF parser, and pure-Go SQLite embedded migrations.',
    features: [
      {
        title: 'Extensible Parser Registry',
        description:
          'Modular statement parsers for HDFC Savings (PDF/Excel/CSV), HDFC CC (PDF/CSV), ICICI Amazon Pay CC (PDF), and Axis Flipkart CC (PDF) with confidence sniffing.',
        tag: 'Parsers',
      },
      {
        title: 'Indian UPI & POS Narration Cleaner',
        description:
          'Resolves noisy Indian banking narrations (`UPI/DR/...`, VPA handles `@okhdfcbank`, `@icici`, `@paytm`), merchant terminal dumps, and 12-digit UTR identifiers.',
        tag: 'Cleaners',
      },
      {
        title: 'Deterministic SHA-256 Deduplication',
        description:
          'Idempotent upserts prevent duplicate records when re-uploading overlapping statement cycles while strictly preserving user categorization edits.',
        tag: 'Integrity',
      },
      {
        title: 'Zero CGO Pure-Go SQLite Engine',
        description:
          'Single self-contained binary distribution embedding SQLite and production React SPA with automatic Goose database schema migrations.',
        tag: 'Architecture',
      },
    ],
    primaryCta: {
      text: 'Import Statements',
      url: '/import',
    },
  },
]

export const WhatsNewView: React.FC = () => {
  return (
    <div className="flex flex-col gap-8 max-w-5xl mx-auto">
      {/* Header */}
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Sparkles className="size-4" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            What's New in LocalFinance
          </h1>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
          <span>Release changelog and feature evolution</span>
          <span>•</span>
          <Badge variant="outline" className="text-success">
            <ShieldCheck className="size-3 mr-1" />
            100% Offline Single Binary
          </Badge>
          <Badge variant="outline">v1.2 Active</Badge>
        </div>
      </div>

      {/* Release Timeline */}
      <div className="flex flex-col gap-6">
        {releases.map((rel) => {
          const Icon = rel.icon

          return (
            <div
              key={rel.version}
              className={cn(
                'relative rounded-xl border p-5 sm:p-6 shadow-xs transition-all',
                rel.isLatest
                  ? 'bg-card/90 border-primary/40 ring-1 ring-primary/20 shadow-md'
                  : 'bg-card/60 backdrop-blur-xs border-border'
              )}
            >
              {/* Version Header Ribbon */}
              <div className="flex flex-wrap items-start justify-between gap-3 border-b border-border/50 pb-4">
                <div className="flex min-w-0 items-start gap-3">
                  <div
                    className="flex size-9 shrink-0 items-center justify-center rounded-xl text-white shadow-xs"
                    style={{ backgroundColor: rel.iconColor }}
                  >
                    <Icon className="size-5" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-sm font-bold text-foreground">
                        {rel.version}
                      </span>
                      <span className="text-sm font-bold text-foreground break-words">
                        — {rel.title}
                      </span>
                    </div>
                    <span className="text-[11px] text-muted-foreground">{rel.date}</span>
                  </div>
                </div>

                <Badge
                  variant={rel.isLatest ? 'default' : 'outline'}
                  className={cn(
                    'text-[10px] font-bold px-2 py-0.5',
                    rel.isLatest
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted/60 text-muted-foreground'
                  )}
                >
                  {rel.badge}
                </Badge>
              </div>

              {/* Release Summary */}
              <p className="text-xs text-muted-foreground leading-relaxed my-4">{rel.summary}</p>

              {/* Itemized Feature Cards */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3 mb-5">
                {rel.features.map((feat, fIdx) => (
                  <div
                    key={fIdx}
                    className="rounded-lg border bg-background/50 p-3.5 flex flex-col gap-1.5"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <h4 className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                        <CheckCircle2 className="size-3.5 text-success shrink-0" />
                        <span>{feat.title}</span>
                      </h4>
                      <Badge variant="secondary" className="px-1.5 py-0">
                        {feat.tag}
                      </Badge>
                    </div>
                    <p className="text-[11px] text-muted-foreground leading-relaxed pl-5">
                      {feat.description}
                    </p>
                  </div>
                ))}
              </div>

              {/* Action Buttons */}
              <div className="flex flex-wrap items-center gap-3 pt-3 border-t border-border/50">
                {rel.primaryCta && (
                  <Link
                    to={rel.primaryCta.url}
                    className={buttonVariants({
                      size: 'sm',
                      className: 'gap-1.5 text-xs font-semibold shadow-xs',
                    })}
                  >
                    <span>{rel.primaryCta.text}</span>
                    <ArrowRight className="size-3.5" />
                  </Link>
                )}

                {rel.secondaryCta && (
                  <Link
                    to={rel.secondaryCta.url}
                    className={buttonVariants({
                      variant: 'outline',
                      size: 'sm',
                      className: 'gap-1.5 text-xs font-medium',
                    })}
                  >
                    <span>{rel.secondaryCta.text}</span>
                  </Link>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
