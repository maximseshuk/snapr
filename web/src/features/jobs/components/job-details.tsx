import { IconLoader, IconPlayerPlay, IconPlayerStop, IconRefresh } from '@tabler/icons-react'
import { useParams } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { usePermissions } from '@/hooks/use-settings'

import { useJobDetails } from '../hooks/use-job-details'
import { JobBackupsTab } from './job-backups-tab'
import { JobConfigTab } from './job-config-tab'
import { JobDetailsSkeleton } from './job-details-skeleton'
import { JobLogsTab } from './job-logs-tab'
import { JobStatusCards } from './job-status-cards'
import { StopJobDialog } from './stop-job-dialog'

export const JobDetails = () => {
  const { t } = useTranslation()
  const { jobName } = useParams({ from: '/_authenticated/jobs/$jobName' })
  const { canRunJobs, canDownloadBackups, canViewConfig } = usePermissions()
  const {
    job,
    status,
    backups,
    logs,
    loading,
    refreshing,
    settings,
    loadData,
    handleRunJob,
    handleCancelJob,
    handleDownloadBackup,
    handleDownloadBackupPart,
    isCancelling,
    isStarting,
  } = useJobDetails(jobName)

  if (loading) {
    return <JobDetailsSkeleton />
  }

  if (!job) {
    return (
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">{t('jobDetails.notFound')}</h2>
          <p className="text-muted-foreground">{t('jobDetails.notFoundDescription')}</p>
        </div>
      </Main>
    )
  }

  return (
    <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
      <PageHeader
        title={`${t('jobDetails.heading')}: ${job.name}`}
        actions={
          <>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => loadData(true)}
                  disabled={refreshing}
                  aria-label={t('common.refresh')}
                >
                  <IconRefresh className={refreshing ? 'animate-spin' : ''} />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{t('common.refresh')}</TooltipContent>
            </Tooltip>
            {canRunJobs &&
              (status?.status === 'running' ? (
                <StopJobDialog
                  onConfirm={handleCancelJob}
                  disabled={isCancelling}
                  trigger={
                    <Button variant="outline" disabled={isCancelling}>
                      {isCancelling ? <IconLoader className="animate-spin" /> : <IconPlayerStop />}
                      {t('jobDetails.stopJob')}
                    </Button>
                  }
                />
              ) : (
                <Button onClick={handleRunJob} disabled={isStarting}>
                  {isStarting ? <IconLoader className="animate-spin" /> : <IconPlayerPlay />}
                  {t('jobDetails.runJob')}
                </Button>
              ))}
          </>
        }
      />

      <JobStatusCards job={job} status={status} backupsCount={backups.length} />

      <Tabs defaultValue="logs" className="flex-1">
        <TabsList>
          <TabsTrigger value="logs">{t('jobDetails.tabs.logs')}</TabsTrigger>
          <TabsTrigger value="backups">
            {t('jobDetails.tabs.backups')} ({backups.length})
          </TabsTrigger>
          {canViewConfig && <TabsTrigger value="config">{t('jobDetails.tabs.configuration')}</TabsTrigger>}
        </TabsList>

        <TabsContent value="logs" className="min-w-0 flex-1">
          <JobLogsTab jobName={job.name} entries={logs.entries} disabled={logs.disabled} settings={settings} />
        </TabsContent>

        <TabsContent value="backups">
          <JobBackupsTab
            backups={backups}
            onDownload={handleDownloadBackup}
            onDownloadPart={handleDownloadBackupPart}
            canDownload={canDownloadBackups}
          />
        </TabsContent>

        {canViewConfig && (
          <TabsContent value="config">
            <JobConfigTab job={job} />
          </TabsContent>
        )}
      </Tabs>
    </Main>
  )
}
