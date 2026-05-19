import { useSettings } from '@/contexts/settings-context'
import { useLogStream } from '@/hooks/use-log-stream'
import { useLogLimits, useLogsAvailability } from '@/hooks/use-settings'
import { useSystem } from '@/hooks/use-system'
import { apiClient } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'
import { deriveLoadingState, useInvalidateWithToast } from '@/lib/query-utils'

const STATUS_POLL = 15_000

export const useSystemLogs = () => {
  const { settings } = useSettings()
  const { systemLogs: tail } = useLogLimits()
  const { system: enabled } = useLogsAvailability()
  const loadData = useInvalidateWithToast([queryKeys.system.status])

  const { entries, resetToken } = useLogStream({
    streamUrl: enabled ? apiClient.streamSystemLogsURL(0) : null,
    initialFetch: async () => {
      if (!enabled || tail <= 0) return null
      const res = await apiClient.getSystemLogs(tail)
      return res.logs
    },
    deps: [tail, enabled],
    maxEntries: tail > 0 ? tail : undefined,
  })

  const statusQuery = useSystem({ refetchInterval: STATUS_POLL })

  return {
    entries,
    resetToken,
    disabled: !enabled,
    systemData: statusQuery.data ?? null,
    ...deriveLoadingState(statusQuery),
    settings,
    loadData,
  }
}
