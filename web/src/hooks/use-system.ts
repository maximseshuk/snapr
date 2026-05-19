import { useQuery } from '@tanstack/react-query'

import { apiClient } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'

interface UseSystemOptions {
  staleTime?: number
  refetchInterval?: number | false
}

export const useSystem = (options?: UseSystemOptions) =>
  useQuery({
    queryKey: queryKeys.system.status,
    queryFn: () => apiClient.getSystem(),
    staleTime: options?.staleTime ?? 30_000,
    refetchInterval: options?.refetchInterval,
  })
