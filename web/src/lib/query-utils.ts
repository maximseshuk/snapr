import type { QueryKey, UseQueryResult } from '@tanstack/react-query'

import { useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

export const useInvalidateWithToast = (keys: QueryKey[]) => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return async (showToast = false) => {
    try {
      await Promise.all(keys.map((key) => queryClient.invalidateQueries({ queryKey: key })))
      if (showToast) toast.success(t('common.success'))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t('common.error'))
    }
  }
}

export const deriveLoadingState = (...queries: Pick<UseQueryResult, 'isLoading' | 'isFetching' | 'data'>[]) => {
  const isLoading = queries.some((q) => q.isLoading && !q.data)
  const isFetching = queries.some((q) => q.isFetching)
  return {
    loading: isLoading,
    refreshing: isFetching && !isLoading,
  }
}
