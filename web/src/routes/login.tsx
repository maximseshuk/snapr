import { createFileRoute, redirect, useRouter } from '@tanstack/react-router'

import { useAuth } from '@/contexts/auth-context'
import { Login } from '@/features/auth/components'

export const Route = createFileRoute('/login')({
  beforeLoad: ({ context }) => {
    if (!context.authEnabled || context.isAuthenticated) {
      throw redirect({
        to: '/',
      })
    }
  },
  component: LoginPage,
})

function LoginPage() {
  const router = useRouter()
  const { checkAuth } = useAuth()

  const handleSuccess = async () => {
    await checkAuth()
    await router.navigate({ to: '/' })
    await router.invalidate()
  }

  return <Login onLoginSuccess={handleSuccess} />
}
