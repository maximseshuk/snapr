import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import type { JobStatus } from '@/types/api'

import { useSettings } from '@/contexts/settings-context'
import { apiClient } from '@/lib/api'
import { queryKeys } from '@/lib/query-keys'
import { deriveLoadingState, useInvalidateWithToast } from '@/lib/query-utils'

import { useJobLogs } from './use-job-logs'

const STATUS_POLL_RUNNING = 3_000

export const useJobDetails = (jobName: string | undefined) => {
  const { t } = useTranslation()
  const { settings } = useSettings()
  const queryClient = useQueryClient()

  const enabled = !!jobName
  const decodedJobName = jobName ?? ''
  const keys = queryKeys.job(decodedJobName)
  const loadData = useInvalidateWithToast([keys.all])

  const configQuery = useQuery({
    queryKey: keys.config,
    queryFn: () => apiClient.getJobConfig(decodedJobName),
    enabled,
  })

  const statusQuery = useQuery({
    queryKey: keys.status,
    queryFn: () => apiClient.getJobStatus(decodedJobName),
    enabled,
    refetchInterval: (query) => (query.state.data?.status === 'running' ? STATUS_POLL_RUNNING : false),
  })

  const backupsQuery = useQuery({
    queryKey: keys.backups,
    queryFn: () => apiClient.getJobBackups(decodedJobName),
    enabled,
  })

  const jobLogs = useJobLogs(enabled ? decodedJobName : undefined)

  const prevStatusRef = useRef<JobStatus['status'] | undefined>(undefined)
  useEffect(() => {
    const current = statusQuery.data?.status
    if (prevStatusRef.current === 'running' && current === 'idle') {
      queryClient.invalidateQueries({ queryKey: keys.backups })
    }
    prevStatusRef.current = current
  }, [statusQuery.data?.status, keys.backups, queryClient])

  const runMutation = useMutation({
    mutationFn: () => apiClient.runJob(decodedJobName),
    onSuccess: () => {
      toast.success(t('success.jobStarted'))
      queryClient.setQueryData<JobStatus>(keys.status, (prev) =>
        prev ? { ...prev, status: 'running', active: true } : prev,
      )
      queryClient.invalidateQueries({ queryKey: keys.status })
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : t('common.error')),
  })

  const cancelMutation = useMutation({
    mutationFn: () => apiClient.cancelJob(decodedJobName),
    onSuccess: () => {
      toast.success(t('success.jobCancelled'))
      queryClient.setQueryData<JobStatus>(keys.status, (prev) =>
        prev ? { ...prev, status: 'idle', active: false } : prev,
      )
      queryClient.invalidateQueries({ queryKey: keys.status })
    },
    onError: (error) => toast.error(error instanceof Error ? error.message : t('common.error')),
  })

  const handleDownloadBackup = (backupId: string) => {
    if (!decodedJobName) return
    apiClient.downloadBackup(decodedJobName, backupId)
  }

  const handleDownloadBackupPart = (partFilename: string) => {
    if (!decodedJobName) return
    apiClient.downloadBackupPart(decodedJobName, partFilename)
  }

  return {
    job: configQuery.data ?? null,
    status: statusQuery.data ?? null,
    backups: backupsQuery.data ?? [],
    logs: jobLogs,
    ...deriveLoadingState(configQuery, statusQuery, backupsQuery),
    settings,
    loadData,
    handleRunJob: () => runMutation.mutate(),
    handleCancelJob: () => cancelMutation.mutate(),
    isCancelling: cancelMutation.isPending,
    isStarting: runMutation.isPending,
    handleDownloadBackup,
    handleDownloadBackupPart,
  }
}
