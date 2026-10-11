import React, { useEffect, useState, useRef } from 'react'

interface AnimatedNumberProps {
  value: number
  duration?: number
  formatFn?: (val: number) => string
  className?: string
}

export const AnimatedNumber: React.FC<AnimatedNumberProps> = ({
  value,
  duration = 600,
  formatFn,
  className = '',
}) => {
  const [displayValue, setDisplayValue] = useState(0)
  const prevValueRef = useRef(0)
  const isFirstMount = useRef(true)
  const animationRef = useRef<number | null>(null)

  useEffect(() => {
    const startValue = isFirstMount.current ? 0 : prevValueRef.current
    const endValue = value
    isFirstMount.current = false
    prevValueRef.current = value

    if (
      startValue === endValue ||
      duration <= 0 ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      setDisplayValue(endValue)
      return
    }

    const startTime = performance.now()

    const animate = (currentTime: number) => {
      const elapsed = currentTime - startTime
      const progress = Math.min(elapsed / duration, 1)

      // Smooth ease-out cubic curve
      const easeOut = 1 - Math.pow(1 - progress, 3)
      const current = startValue + (endValue - startValue) * easeOut

      setDisplayValue(current)

      if (progress < 1) {
        animationRef.current = requestAnimationFrame(animate)
      } else {
        setDisplayValue(endValue)
      }
    }

    animationRef.current = requestAnimationFrame(animate)

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current)
      }
    }
  }, [value, duration])

  const formatted = formatFn ? formatFn(displayValue) : displayValue.toFixed(2)

  return <span className={`font-mono tabular-nums tracking-tight ${className}`}>{formatted}</span>
}
