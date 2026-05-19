import type { LogLine } from '@/types/api'

import { useLogStream } from '@/hooks/use-log-stream'
import { useLogLimits, useLogsAvailability } from '@/hooks/use-settings'
import { apiClient } from '@/lib/api'

interface UseJobLogsResult {
  entries: LogLine[]
  resetToken: number
  disabled: boolean
}

export const useJobLogs = (jobName: string | undefined): UseJobLogsResult => {
  const { jobLogs: tail } = useLogLimits()
  const { perJob: enabled } = useLogsAvailability()

  const { entries, resetToken } = useLogStream({
    streamUrl: jobName && enabled ? apiClient.streamJobLogsURL(jobName, 0) : null,
    initialFetch: async () => {
      if (!jobName || !enabled || tail <= 0) return null
      const res = await apiClient.getJobLogs(jobName, tail)
      return res.logs
    },
    deps: [jobName, tail, enabled],
    maxEntries: tail > 0 ? tail : undefined,
  })

  return { entries, resetToken, disabled: !enabled }
}
