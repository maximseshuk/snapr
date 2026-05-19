import { createFileRoute, redirect } from '@tanstack/react-router'

import { AuthenticatedLayout } from '@/components/layout/authenticated-layout'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ context }) => {
    if (context.authEnabled && !context.isAuthenticated) {
      throw redirect({
        to: '/login',
      })
    }
  },
  component: AuthenticatedLayout,
})
