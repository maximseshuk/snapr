import type { ReactNode } from 'react'

import { createContext, useContext, useEffect, useState } from 'react'

import { apiClient } from '@/lib/api'

interface AuthContextType {
  isAuthenticated: boolean
  authEnabled: boolean
  isLoading: boolean
  checkAuth: () => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [authEnabled, setAuthEnabled] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  const checkAuth = async () => {
    try {
      const response = await apiClient.checkAuth()
      setIsAuthenticated(response.authenticated)
      setAuthEnabled(response.authEnabled)
    } catch (error) {
      console.error('Auth check failed:', error)
      setIsAuthenticated(false)
      setAuthEnabled(true)
    } finally {
      setIsLoading(false)
    }
  }

  const logout = async () => {
    try {
      await apiClient.logout()
      setIsAuthenticated(false)
    } catch (error) {
      console.error('Logout failed:', error)
    }
  }

  useEffect(() => {
    checkAuth()
  }, [])

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        authEnabled,
        isLoading,
        checkAuth,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export const useAuth = () => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
