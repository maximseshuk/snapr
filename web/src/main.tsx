import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { createRouter, RouterProvider } from '@tanstack/react-router'
import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import './index.css'
import './i18n'
import { Spinner } from '@/components/ui/spinner'
import { AuthProvider, useAuth } from '@/contexts/auth-context'
import { SettingsProvider } from '@/contexts/settings-context'
import { ThemeProvider } from '@/contexts/theme-context'

import { routeTree } from './routeTree.gen'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

const router = createRouter({
  routeTree,
  context: {
    isAuthenticated: false,
    authEnabled: false,
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const AppContent = () => {
  const { isAuthenticated, authEnabled, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="bg-background flex min-h-screen items-center justify-center">
        <Spinner className="size-8" />
      </div>
    )
  }

  return (
    <RouterProvider
      router={router}
      context={{
        isAuthenticated,
        authEnabled,
      }}
    />
  )
}

export const App = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <SettingsProvider>
          <AppContent />
        </SettingsProvider>
      </AuthProvider>
      {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
    </QueryClientProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider defaultTheme="dark">
      <App />
    </ThemeProvider>
  </StrictMode>,
)
