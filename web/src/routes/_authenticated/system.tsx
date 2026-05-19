import { createFileRoute } from '@tanstack/react-router'

import { System } from '@/features/system/components'

export const Route = createFileRoute('/_authenticated/system')({
  component: System,
})
