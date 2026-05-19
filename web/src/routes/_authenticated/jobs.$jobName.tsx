import { createFileRoute } from '@tanstack/react-router'

import { JobDetails } from '@/features/jobs/components'

export const Route = createFileRoute('/_authenticated/jobs/$jobName')({
  component: JobDetails,
  staticData: { backTo: '/' },
})
