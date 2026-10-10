import { usePrivacy } from '@/components/privacy-provider'
import React from 'react'
import { LucideIcon } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'
import { cn, formatINR } from '@/lib/utils'
import { AnimatedNumber } from '@/components/ui/animated-number'

interface OverviewCardProps {
  title: string
  amount: string
  numericValue?: number
  subtitle?: string
  icon: LucideIcon
  variant?: 'default' | 'income' | 'expense' | 'savings'
}

export const OverviewCard: React.FC<OverviewCardProps> = ({
  title,
  amount,
  numericValue,
  subtitle,
  icon: Icon,
  variant = 'default',
}) => {
  const { isPrivacyMode } = usePrivacy()
  const variantStyles = {
    default: {
      iconBg: 'bg-primary/10 text-primary border-primary/20',
      textAccent: 'text-foreground',
    },
    income: {
      iconBg: 'bg-success/10 text-success border-success/20',
      textAccent: 'text-success ',
    },
    expense: {
      iconBg: 'bg-destructive/10 text-destructive border-destructive/20',
      textAccent: 'text-foreground',
    },
    savings: {
      iconBg: 'bg-info/10 text-info border-info/20',
      textAccent: 'text-info ',
    },
  }

  const { iconBg } = variantStyles[variant]

  return (
    <Card className="overflow-hidden transition-all duration-200">
      <CardContent className="p-5">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
            {title}
          </span>
          <div
            className={cn(
              'flex size-8 items-center justify-center rounded-lg border transition-transform duration-200 group-hover:scale-105',
              iconBg
            )}
          >
            <Icon className="size-4" />
          </div>
        </div>
        <div className="mt-3">
          <div className="text-2xl font-bold font-mono tracking-tight text-foreground tabular-nums privacy-blur">
            {numericValue !== undefined ? (
              <AnimatedNumber
                value={numericValue}
                formatFn={(value) => formatINR(value, isPrivacyMode)}
              />
            ) : isPrivacyMode ? (
              '₹••••••'
            ) : (
              amount
            )}
          </div>
          {subtitle && (
            <p className="mt-1 text-xs text-muted-foreground flex items-center gap-1.5 font-medium">
              {subtitle}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
