import React from 'react'
import { formatINR } from '@/lib/utils'
import { usePrivacy } from '@/components/privacy-provider'

interface PrivacyAmountProps extends React.HTMLAttributes<HTMLSpanElement> {
  amount: number
  prefix?: string
  showBlurOnHover?: boolean
}

export const PrivacyAmount: React.FC<PrivacyAmountProps> = ({
  amount,
  prefix,
  className = '',
  showBlurOnHover: _showBlurOnHover,
  ...props
}) => {
  const { isPrivacyMode } = usePrivacy()
  const formatted = formatINR(amount)

  if (!isPrivacyMode) {
    return (
      <span className={className} {...props}>
        {prefix}
        {formatted}
      </span>
    )
  }

  return (
    <span className={className} title="Discreet Mode active" {...props}>
      {prefix}
      {formatINR(amount, true)}
    </span>
  )
}
