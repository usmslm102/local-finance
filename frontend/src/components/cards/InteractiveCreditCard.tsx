import { cn } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'
import React, { useRef, useState } from 'react'
import { Wifi } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { formatINR } from '@/lib/utils'

interface CreditCardProps {
  cardName: string
  bankName: string
  network: string
  last4: string
  cardholderName?: string
  variant?: string
  totalDue?: number
  dueDate?: string
  creditLimit?: number
  colorScheme?: 'indigo' | 'slate' | 'emerald' | 'amber' | 'rose'
  compact?: boolean
}

export const InteractiveCreditCard: React.FC<CreditCardProps> = ({
  cardName,
  bankName,
  network,
  last4,
  cardholderName = 'CARDHOLDER',
  variant,
  totalDue,
  dueDate,
  creditLimit,
  colorScheme = 'indigo',
  compact = false,
}) => {
  const { isPrivacyMode } = usePrivacy()
  const cardRef = useRef<HTMLDivElement>(null)
  const [rotation, setRotation] = useState({ x: 0, y: 0 })
  const [glare, setGlare] = useState({ x: 50, y: 50, opacity: 0 })
  const [isHovered, setIsHovered] = useState(false)

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    const rect = cardRef.current.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    const centerX = rect.width / 2
    const centerY = rect.height / 2

    // Max 8 degrees rotation for tactile 3D perspective
    const rotateY = ((x - centerX) / centerX) * 8
    const rotateX = -((y - centerY) / centerY) * 8

    setRotation({ x: rotateX, y: rotateY })
    setGlare({
      x: (x / rect.width) * 100,
      y: (y / rect.height) * 100,
      opacity: 0.28,
    })
  }

  const handleMouseEnter = () => {
    if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches) setIsHovered(true)
  }

  const handleMouseLeave = () => {
    setIsHovered(false)
    setRotation({ x: 0, y: 0 })
    setGlare((prev) => ({ ...prev, opacity: 0 }))
  }

  // Theme gradients based on card variant
  const getGradients = () => {
    switch (colorScheme) {
      case 'emerald':
        return 'from-slate-900 via-slate-800 to-slate-950 border-border'
      case 'amber':
        return 'from-stone-900 via-stone-800 to-stone-950 border-border'
      case 'rose':
        return 'from-slate-900 via-slate-800 to-slate-950 border-border'
      case 'slate':
        return 'from-slate-900 via-neutral-900 to-zinc-950 border-slate-600/50 shadow-black/70'
      default:
        return 'from-slate-950 via-slate-800 to-slate-900 border-border'
    }
  }

  return (
    <div style={{ perspective: '1200px' }} className="w-full select-none">
      <div
        ref={cardRef}
        onMouseMove={handleMouseMove}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        style={{
          transform: `rotateX(${rotation.x}deg) rotateY(${rotation.y}deg)`,
          transformStyle: 'preserve-3d',
          transition: isHovered
            ? 'transform 0.04s ease-out'
            : 'transform 0.5s cubic-bezier(0.16, 1, 0.3, 1)',
        }}
        className={cn(
          'relative aspect-[1.586/1] w-full',
          compact ? 'max-w-[280px] p-3.5 rounded-xl' : 'max-w-sm p-5 rounded-2xl',
          'bg-gradient-to-br',
          getGradients(),
          'border text-white shadow-xl transition-shadow duration-300 hover:shadow-2xl overflow-hidden'
        )}
      >
        {/* Ambient Glare Light Foil */}
        <div
          className="pointer-events-none absolute inset-0 transition-opacity duration-200"
          style={{
            opacity: glare.opacity,
            background: `radial-gradient(circle at ${glare.x}% ${glare.y}%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0.1) 40%, rgba(255,255,255,0) 70%)`,
          }}
        />

        {/* Subtle holographic background texture pattern */}
        <div className="pointer-events-none absolute inset-0 opacity-[0.03] bg-[radial-gradient(#fff_1px,transparent_1px)] [background-size:12px_12px]" />

        {/* Top Row: Bank Name & Contactless WiFi */}
        <div className="relative z-10 flex items-center justify-between">
          <div className="flex flex-col">
            <span
              className={cn(
                compact ? 'text-[11px]' : 'text-xs',
                'font-bold tracking-wider uppercase text-white/90'
              )}
            >
              {bankName}
            </span>
            <span
              className={cn(
                compact ? 'text-[9px]' : 'text-[10px]',
                'font-medium text-white/60 tracking-wide'
              )}
            >
              {variant || cardName}
            </span>
          </div>
          <div className="flex items-center gap-1.5">
            <Wifi className={cn(compact ? 'size-3.5' : 'size-4', 'text-white/70 rotate-90')} />
            <Badge variant="outline" className="border-white/20 px-1.5 py-0">
              {network || 'VISA'}
            </Badge>
          </div>
        </div>

        {/* EMV Chip & Contactless */}
        <div
          className={cn('relative z-10', compact ? 'my-2.5' : 'my-4', 'flex items-center gap-3')}
        >
          {/* Gold EMV Chip */}
          <div
            className={cn(
              'relative',
              compact ? 'h-5 w-7 rounded-xs' : 'h-7 w-9 rounded-md',
              'bg-gradient-to-br from-warning via-warning to-warning p-0.5 shadow-inner'
            )}
          >
            <div className="h-full w-full rounded-xs border border-warning/40 bg-warning/60 grid grid-cols-2 gap-0.5">
              <div className="border-r border-b border-warning/30" />
              <div className="border-b border-warning/30" />
              <div className="border-r border-warning/30" />
              <div />
            </div>
          </div>
        </div>

        {/* Masked Card Number */}
        <div
          className={cn(
            'relative z-10 font-mono tracking-widest',
            compact ? 'text-xs' : 'text-sm',
            'text-white/90 font-medium'
          )}
        >
          •••• •••• •••• {last4.replace(/[^0-9]/g, '') || '••••'}
        </div>

        {/* Bottom Row: Cardholder & Limits */}
        <div
          className={cn(
            'relative z-10',
            compact ? 'mt-2' : 'mt-3',
            'flex items-end justify-between'
          )}
        >
          <div className="flex flex-col">
            <span className="text-[7px] uppercase tracking-widest text-white/50">Cardholder</span>
            <span
              className={cn(
                'text-[10px] font-semibold tracking-wider text-white/90 uppercase truncate',
                compact ? 'max-w-[110px]' : 'max-w-[150px]'
              )}
            >
              {cardholderName}
            </span>
          </div>

          {dueDate && totalDue !== undefined ? (
            <div className="flex flex-col text-right">
              <span className="text-[7px] uppercase tracking-widest text-white/50">
                Due: {dueDate}
              </span>
              <span
                className={cn(
                  compact ? 'text-[11px]' : 'text-xs',
                  'font-bold font-mono text-white'
                )}
              >
                {formatINR(totalDue, isPrivacyMode)}
              </span>
            </div>
          ) : creditLimit ? (
            <div className="flex flex-col text-right">
              <span className="text-[7px] uppercase tracking-widest text-white/50">Limit</span>
              <span
                className={cn(
                  compact ? 'text-[11px]' : 'text-xs',
                  'font-bold font-mono text-white/80'
                )}
              >
                {formatINR(creditLimit, isPrivacyMode)}
              </span>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}
