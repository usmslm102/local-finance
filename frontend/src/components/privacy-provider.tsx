import React, { createContext, useContext, useEffect, useState, useCallback } from 'react'
import { playPrivacyToggleSound } from '@/lib/audio'

interface PrivacyContextType {
  isPrivacyMode: boolean
  togglePrivacyMode: () => void
  setPrivacyMode: (value: boolean) => void
  maskValue: (value: string | number) => string
}

const PrivacyContext = createContext<PrivacyContextType | undefined>(undefined)

const STORAGE_KEY = 'local-finance-privacy-mode'

export const PrivacyProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isPrivacyMode, setIsPrivacyMode] = useState<boolean>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      return stored === 'true'
    } catch {
      return false
    }
  })

  // Synchronize CSS class on body & document element
  useEffect(() => {
    if (typeof document !== 'undefined') {
      if (isPrivacyMode) {
        document.documentElement.classList.add('privacy-mode')
        document.body.classList.add('privacy-mode')
      } else {
        document.documentElement.classList.remove('privacy-mode')
        document.body.classList.remove('privacy-mode')
      }
    }
    try {
      localStorage.setItem(STORAGE_KEY, String(isPrivacyMode))
    } catch {
      // Ignore storage errors
    }
  }, [isPrivacyMode])

  const togglePrivacyMode = useCallback(() => {
    setIsPrivacyMode((prev) => {
      const next = !prev
      playPrivacyToggleSound(next)
      return next
    })
  }, [])

  const setPrivacyMode = useCallback((val: boolean) => {
    setIsPrivacyMode(val)
  }, [])

  const maskValue = useCallback(
    (value: string | number): string => {
      if (!isPrivacyMode) return String(value)
      return '••••••'
    },
    [isPrivacyMode]
  )

  // Global Keyboard Shortcuts: 'P' or '⌘.' / 'Ctrl+.'
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore if user is currently typing in an input, textarea, or contentEditable
      if (e.repeat || e.isComposing) return
      const target = e.target as HTMLElement
      const isInput =
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable ||
        !!target.closest('[role="combobox"], [role="listbox"], [role="dialog"]')

      // ⌘. or Ctrl+. works even inside inputs for quick panic toggle
      if ((e.metaKey || e.ctrlKey) && e.key === '.') {
        e.preventDefault()
        togglePrivacyMode()
        return
      }

      // 'P' or 'p' only triggers outside form controls
      if (!isInput && (e.key === 'p' || e.key === 'P') && !e.metaKey && !e.ctrlKey && !e.altKey) {
        e.preventDefault()
        togglePrivacyMode()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [togglePrivacyMode])

  return (
    <PrivacyContext.Provider
      value={{
        isPrivacyMode,
        togglePrivacyMode,
        setPrivacyMode,
        maskValue,
      }}
    >
      {children}
    </PrivacyContext.Provider>
  )
}

export function usePrivacy(): PrivacyContextType {
  const ctx = useContext(PrivacyContext)
  if (!ctx) {
    throw new Error('usePrivacy must be used within a PrivacyProvider')
  }
  return ctx
}
