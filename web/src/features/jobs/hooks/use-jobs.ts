import { useQuery } from '@tanstack/react-query'

import { apiClient } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'
import { deriveLoadingState, useInvalidateWithToast } from '@/lib/query-utils'

const POLL_RUNNING = 3_000

export const useJobs = () => {
  const loadData = useInvalidateWithToast([queryKeys.jobs.all])

  const query = useQuery({
    queryKey: queryKeys.jobs.all,
    queryFn: () => apiClient.getJobs(),
    refetchInterval: (q) => (q.state.data?.some((job) => job.status === 'running') ? POLL_RUNNING : false),
  })

  return {
    jobs: query.data ?? [],
    ...deriveLoadingState(query),
    loadData,
  }
}
