import { createContext, type ReactNode, useContext, useEffect, useState } from 'react'

import type { Settings } from '@/types/api'

import { apiClient } from '@/lib/api'

interface SettingsContextValue {
  settings: Settings | null
  loading: boolean
  refresh: () => Promise<void>
}

const SettingsContext = createContext<SettingsContextValue | undefined>(undefined)

export const SettingsProvider = ({ children }: { children: ReactNode }) => {
  const [settings, setSettings] = useState<Settings | null>(null)
  const [loading, setLoading] = useState(true)

  const loadSettings = async () => {
    try {
      const data = await apiClient.getSettings()
      setSettings(data)
    } catch (error) {
      console.error('Failed to load settings:', error)
    } finally {
      setLoading(false)
    }
  }

  const refresh = async () => {
    await loadSettings()
  }

  useEffect(() => {
    loadSettings()
    const onFocus = () => loadSettings()
    window.addEventListener('focus', onFocus)
    return () => window.removeEventListener('focus', onFocus)
  }, [])

  return <SettingsContext.Provider value={{ settings, loading, refresh }}>{children}</SettingsContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export const useSettings = () => {
  const context = useContext(SettingsContext)
  if (context === undefined) {
    throw new Error('useSettings must be used within a SettingsProvider')
  }
  return context
}
