import { useSettings } from '@/contexts/settings-context'

export const usePermissions = () => {
  const { settings } = useSettings()
  return {
    canRunJobs: settings?.permissions?.allowManualRun ?? true,
    canDownloadBackups: settings?.permissions?.allowBackupDownload ?? true,
    canViewConfig: settings?.permissions?.showConfig ?? true,
  }
}

export const useLogLimits = () => {
  const { settings } = useSettings()
  return {
    jobLogs: settings?.logLimits?.jobLogs ?? 0,
    systemLogs: settings?.logLimits?.systemLogs ?? 0,
  }
}

export const useLogsAvailability = () => {
  const { settings } = useSettings()
  return {
    system: settings?.logs?.system ?? true,
    perJob: settings?.logs?.perJob ?? true,
  }
}
