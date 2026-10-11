import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { cn } from '@/lib/utils'
import React from 'react'

interface MerchantBrand {
  name: string
  icon: string
  bg: string
  textColor: string
  borderColor: string
}

export function getMerchantBrand(rawPayee: string): MerchantBrand {
  const p = (rawPayee || '').toUpperCase()

  if (p.includes('SWIGGY')) {
    return {
      name: 'Swiggy',
      icon: '🍔',
      bg: 'bg-warning/15',
      textColor: 'text-warning',
      borderColor: 'border-warning/30',
    }
  }
  if (p.includes('ZOMATO')) {
    return {
      name: 'Zomato',
      icon: '🍽️',
      bg: 'bg-destructive/15',
      textColor: 'text-destructive',
      borderColor: 'border-destructive/30',
    }
  }
  if (p.includes('BLINKIT') || p.includes('GROFERS')) {
    return {
      name: 'Blinkit',
      icon: '🛒',
      bg: 'bg-warning/15',
      textColor: 'text-warning',
      borderColor: 'border-warning/30',
    }
  }
  if (p.includes('ZEPTO')) {
    return {
      name: 'Zepto',
      icon: '⚡',
      bg: 'bg-info/15',
      textColor: 'text-info',
      borderColor: 'border-info/30',
    }
  }
  if (p.includes('AMAZON')) {
    return {
      name: 'Amazon',
      icon: '📦',
      bg: 'bg-warning/15',
      textColor: 'text-warning',
      borderColor: 'border-warning/30',
    }
  }
  if (p.includes('FLIPKART')) {
    return {
      name: 'Flipkart',
      icon: '🛍️',
      bg: 'bg-info/15',
      textColor: 'text-info',
      borderColor: 'border-info/30',
    }
  }
  if (p.includes('MYNTRA')) {
    return {
      name: 'Myntra',
      icon: '👗',
      bg: 'bg-pink-500/15',
      textColor: 'text-pink-400',
      borderColor: 'border-pink-500/30',
    }
  }
  if (p.includes('UBER')) {
    return {
      name: 'Uber',
      icon: '🚗',
      bg: 'bg-slate-500/15',
      textColor: 'text-slate-300',
      borderColor: 'border-slate-500/30',
    }
  }
  if (p.includes('OLA')) {
    return {
      name: 'Ola',
      icon: '🚕',
      bg: 'bg-lime-500/15',
      textColor: 'text-lime-400',
      borderColor: 'border-lime-500/30',
    }
  }
  if (p.includes('NETFLIX')) {
    return {
      name: 'Netflix',
      icon: '🎬',
      bg: 'bg-destructive/15',
      textColor: 'text-destructive',
      borderColor: 'border-destructive/30',
    }
  }
  if (p.includes('SPOTIFY')) {
    return {
      name: 'Spotify',
      icon: '🎵',
      bg: 'bg-success/15',
      textColor: 'text-success',
      borderColor: 'border-success/30',
    }
  }
  if (
    p.includes('SHELL') ||
    p.includes('PETROL') ||
    p.includes('HPCL') ||
    p.includes('BPCL') ||
    p.includes('IOCL') ||
    p.includes('FUEL')
  ) {
    return {
      name: 'Fuel',
      icon: '⛽',
      bg: 'bg-warning/15',
      textColor: 'text-warning',
      borderColor: 'border-warning/30',
    }
  }
  if (
    p.includes('MAKEMYTRIP') ||
    p.includes('INDIGO') ||
    p.includes('AIR INDIA') ||
    p.includes('FLIGHT') ||
    p.includes('HOTEL')
  ) {
    return {
      name: 'Travel',
      icon: '✈️',
      bg: 'bg-info/15',
      textColor: 'text-info',
      borderColor: 'border-info/30',
    }
  }
  if (p.includes('CRED') || p.includes('BILL PAYMENT')) {
    return {
      name: 'Card Bill',
      icon: '💳',
      bg: 'bg-info/15',
      textColor: 'text-info',
      borderColor: 'border-info/30',
    }
  }
  if (
    p.includes('ELECTRICITY') ||
    p.includes('BESCOM') ||
    p.includes('TATA POWER') ||
    p.includes('ADANI') ||
    p.includes('UTILITY') ||
    p.includes('WATER') ||
    p.includes('GAS')
  ) {
    return {
      name: 'Utility',
      icon: '⚡',
      bg: 'bg-warning/15',
      textColor: 'text-warning',
      borderColor: 'border-warning/30',
    }
  }
  if (p.includes('AIRTEL') || p.includes('JIO') || p.includes('VI ') || p.includes('VODAFONE')) {
    return {
      name: 'Telecom',
      icon: '📱',
      bg: 'bg-destructive/15',
      textColor: 'text-destructive',
      borderColor: 'border-destructive/30',
    }
  }
  if (p.includes('SALARY') || p.includes('PAYROLL') || p.includes('INTEREST')) {
    return {
      name: 'Income',
      icon: '💰',
      bg: 'bg-success/15',
      textColor: 'text-success',
      borderColor: 'border-success/30',
    }
  }
  if (p.includes('UPI')) {
    return {
      name: 'UPI',
      icon: '📱',
      bg: 'bg-success/15',
      textColor: 'text-success',
      borderColor: 'border-success/30',
    }
  }

  // Fallback: Use first letter initials
  const initials = rawPayee ? rawPayee.slice(0, 2).toUpperCase() : 'TX'
  return {
    name: rawPayee,
    icon: initials,
    bg: 'bg-muted/80',
    textColor: 'text-foreground font-mono text-[10px] font-bold',
    borderColor: 'border-border/80',
  }
}

interface MerchantAvatarProps {
  payee: string
  size?: 'xs' | 'sm' | 'md'
  className?: string
}

export const MerchantAvatar: React.FC<MerchantAvatarProps> = ({
  payee,
  size = 'sm',
  className = '',
}) => {
  const brand = getMerchantBrand(payee)

  const sizeClasses = {
    xs: 'size-5 text-[11px] rounded-md',
    sm: 'size-6 text-xs rounded-lg',
    md: 'size-8 text-sm rounded-xl',
  }[size]

  return (
    <Avatar className={cn('shrink-0', sizeClasses, className)} title={payee}>
      <AvatarFallback className={cn(brand.bg, brand.textColor)}>{brand.icon}</AvatarFallback>
    </Avatar>
  )
}
