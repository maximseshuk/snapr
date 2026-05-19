import { createFileRoute } from '@tanstack/react-router'

import { JobsList } from '@/features/jobs/components'

export const Route = createFileRoute('/_authenticated/')({
  component: JobsList,
})
