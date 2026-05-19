import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import type { Job } from '@/types/api'

import { apiClient } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'

export const useJobActions = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const setJobStatus = (jobName: string, status: Job['status']) => {
    queryClient.setQueryData<Job[]>(queryKeys.jobs.all, (prev) =>
      prev?.map((job) => (job.name === jobName ? { ...job, status } : job)),
    )
  }

  const runMutation = useMutation({
    mutationFn: (jobName: string) => apiClient.runJob(jobName),
    onMutate: (jobName) => setJobStatus(jobName, 'running'),
    onSuccess: () => {
      toast.success(t('success.jobStarted'))
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all })
    },
    onError: (error, jobName) => {
      toast.error(error instanceof Error ? error.message : t('common.error'))
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all })
      setJobStatus(jobName, 'idle')
    },
  })

  const cancelMutation = useMutation({
    mutationFn: (jobName: string) => apiClient.cancelJob(jobName),
    onSuccess: (_data, jobName) => {
      toast.success(t('success.jobCancelled'))
      setJobStatus(jobName, 'idle')
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs.all })
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : t('common.error')),
  })

  return {
    cancellingJob: cancelMutation.isPending ? cancelMutation.variables : null,
    handleRunJob: (jobName: string) => runMutation.mutate(jobName),
    handleCancelJob: (jobName: string) => cancelMutation.mutate(jobName),
  }
}
