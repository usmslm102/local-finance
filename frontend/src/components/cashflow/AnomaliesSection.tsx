import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import React from 'react'
import { MoMAnomaly } from '@/types'
import { formatINR } from '@/lib/utils'
import {
  TrendingUp,
  TrendingDown,
  Sparkles,
  AlertTriangle,
  CheckCircle2,
  HelpCircle,
  PlusCircle,
  PiggyBank,
} from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface AnomaliesSectionProps {
  anomalies: MoMAnomaly[]
  previousMonth: string
  selectedMonth: string
}

export const AnomaliesSection: React.FC<AnomaliesSectionProps> = ({ anomalies, previousMonth }) => {
  const { isPrivacyMode } = usePrivacy()
  if (!anomalies || anomalies.length === 0) {
    return (
      <div className="flex items-center gap-3 rounded-xl border border-success/20 bg-success/5 p-4 text-xs text-foreground">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-success/10 text-success">
          <CheckCircle2 className="size-4" />
        </div>
        <div>
          <h4 className="font-semibold text-success">Stable Spending Velocity</h4>
          <p className="text-muted-foreground mt-0.5">
            No unusual spending spikes or anomalous shifts detected compared to{' '}
            {previousMonth || 'last month'}.
          </p>
        </div>
      </div>
    )
  }

  const getSeverityStyle = (severity: string, type: string) => {
    switch (severity) {
      case 'warning':
        return {
          border: 'border-warning/30 ',
          bg: 'bg-warning/5 ',
          iconBg: 'bg-warning/15 text-warning ',
          badge: 'bg-warning/15 text-warning border-warning/30',
          icon: TrendingUp,
        }
      case 'danger':
        return {
          border: 'border-destructive/30 ',
          bg: 'bg-destructive/5 ',
          iconBg: 'bg-destructive/15 text-destructive ',
          badge: 'bg-destructive/15 text-destructive border-destructive/30',
          icon: AlertTriangle,
        }
      case 'success':
        return {
          border: 'border-success/30 ',
          bg: 'bg-success/5 ',
          iconBg: 'bg-success/15 text-success ',
          badge: 'bg-success/15 text-success border-success/30',
          icon: type === 'SAVINGS_MILESTONE' ? PiggyBank : TrendingDown,
        }
      case 'info':
      default:
        return {
          border: 'border-info/30 ',
          bg: 'bg-info/5 ',
          iconBg: 'bg-info/15 text-info ',
          badge: 'bg-info/15 text-info border-info/30',
          icon: type === 'NEW_SPEND' ? PlusCircle : Sparkles,
        }
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground flex items-center gap-2">
          <Sparkles className="size-3.5 text-primary" />
          MoM Spending Shifts &amp; Anomalies ({anomalies.length})
        </h3>
        <span className="text-[11px] text-muted-foreground">Compared to {previousMonth}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {anomalies.map((a) => {
          const style = getSeverityStyle(a.severity, a.type)
          const Icon = style.icon || HelpCircle

          return (
            <div
              key={a.id}
              className={cn(
                'flex flex-col justify-between rounded-xl border p-3.5 transition-all hover:shadow-xs',
                style.border,
                style.bg
              )}
            >
              <div>
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <div
                      className={cn(
                        'flex size-7 shrink-0 items-center justify-center rounded-lg',
                        style.iconBg
                      )}
                    >
                      <Icon className="size-4" />
                    </div>
                    <span className="text-xs font-bold text-foreground">{a.title}</span>
                  </div>

                  {a.percentage_change !== 0 && (
                    <Badge
                      variant="outline"
                      className={cn('text-[10px] font-bold px-1.5 py-0.5 shrink-0', style.badge)}
                    >
                      {a.percentage_change > 0 ? '+' : ''}
                      {a.percentage_change.toFixed(0)}%
                    </Badge>
                  )}
                </div>

                <p className="text-[11px] text-muted-foreground mt-2 leading-relaxed">
                  {isPrivacyMode
                    ? a.description.replace(/₹[\d,]+(?:\.\d+)?/g, '₹••••••')
                    : a.description}
                </p>
              </div>

              <div className="flex items-center justify-between border-t border-border/40 pt-2.5 mt-3 text-[11px]">
                {a.top_contributor ? (
                  <span className="text-muted-foreground truncate font-medium">
                    {a.top_contributor}
                  </span>
                ) : (
                  <span className="text-muted-foreground">
                    Current:{' '}
                    <strong className="text-foreground">
                      {formatINR(a.current_amount, isPrivacyMode)}
                    </strong>
                  </span>
                )}

                {a.delta_amount !== 0 && (
                  <span
                    className={cn(
                      'font-semibold shrink-0',
                      a.delta_amount > 0 ? 'text-warning ' : 'text-success '
                    )}
                  >
                    {a.delta_amount > 0 ? '+' : ''}
                    {formatINR(a.delta_amount, isPrivacyMode)}
                  </span>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
