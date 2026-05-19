import { IconArchive, IconCalendar, IconCircleDashed, IconClock, IconHistory, IconLoader2 } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import type { JobDetail, JobStatus } from '@/types/api'

import { statusPillVariants } from '@/components/status-variants'
import { Card, CardAction, CardContent, CardDescription, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { formatDate, formatRelativeTime } from '../utils/formatters'
import { describeCron } from '../utils/helpers'

interface JobStatusCardsProps {
  job: JobDetail
  status: JobStatus | null
  backupsCount: number
}

export const JobStatusCards = ({ job, status, backupsCount }: JobStatusCardsProps) => {
  const { t, i18n } = useTranslation()
  const isRunning = status?.status === 'running'

  return (
    <>
      <Card className="!py-2 md:hidden">
        <CardContent className="flex flex-col gap-3 px-4 py-2">
          <div className="grid grid-cols-2 grid-rows-[auto_auto] gap-x-3 gap-y-1">
            <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
              {isRunning ? (
                <IconLoader2 className="mt-0.5 size-3.5 shrink-0 animate-spin" />
              ) : (
                <IconCircleDashed className="mt-0.5 size-3.5 shrink-0" />
              )}
              <span>{t('jobDetails.status')}</span>
            </div>
            <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
              <IconArchive className="mt-0.5 size-3.5 shrink-0" />
              <span>{t('jobDetails.totalBackups')}</span>
            </div>
            {status ? (
              <div
                className={cn(
                  'inline-flex w-fit items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-semibold',
                  statusPillVariants({ tone: isRunning ? 'running' : 'idle' }),
                )}
              >
                {t(isRunning ? 'status.running' : 'status.idle')}
              </div>
            ) : (
              <Skeleton className="h-5 w-16" />
            )}
            <div className="text-base font-semibold">{backupsCount}</div>
          </div>
          <div className="grid grid-cols-2 grid-rows-[auto_auto] gap-x-3 gap-y-1">
            <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
              <IconHistory className="mt-0.5 size-3.5 shrink-0" />
              <span>{t('jobDetails.lastRun')}</span>
            </div>
            <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
              <IconClock className="mt-0.5 size-3.5 shrink-0" />
              <span>{t('jobDetails.nextRun')}</span>
            </div>
            <div className="truncate text-sm font-semibold lowercase">
              {status?.lastRun ? formatRelativeTime(status.lastRun, t) : '-'}
            </div>
            <div className="truncate text-sm font-semibold lowercase">
              {status?.nextRun ? formatRelativeTime(status.nextRun, t) : '-'}
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
              <IconCalendar className="mt-0.5 size-3.5 shrink-0" />
              <span>{t('jobDetails.schedule')}</span>
            </div>
            <div className="font-mono text-base font-semibold">{job.schedule}</div>
            <p className="text-muted-foreground text-xs">{describeCron(job.schedule, i18n.language)}</p>
          </div>
        </CardContent>
      </Card>

      <div className="hidden gap-4 md:grid md:grid-cols-2 lg:grid-cols-5">
        <Card>
          <CardHeader>
            <CardDescription>{t('jobDetails.status')}</CardDescription>
            <CardAction>
              {isRunning ? (
                <IconLoader2 className="text-status-running-foreground size-4 animate-spin" />
              ) : (
                <IconCircleDashed className="text-muted-foreground size-4" />
              )}
            </CardAction>
          </CardHeader>
          <CardContent>
            {status ? (
              <div
                className={cn(
                  'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-base font-semibold',
                  statusPillVariants({ tone: isRunning ? 'running' : 'idle' }),
                )}
              >
                {t(isRunning ? 'status.running' : 'status.idle')}
              </div>
            ) : (
              <Skeleton className="h-6 w-20" />
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('jobDetails.schedule')}</CardDescription>
            <CardAction>
              <IconCalendar className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="font-mono text-2xl leading-none font-semibold tracking-tight">{job.schedule}</div>
            <p className="text-muted-foreground mt-2 text-xs">{describeCron(job.schedule, i18n.language)}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('jobDetails.lastRun')}</CardDescription>
            <CardAction>
              <IconHistory className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="text-2xl leading-none font-semibold tracking-tight lowercase">
              {status?.lastRun ? formatRelativeTime(status.lastRun, t) : '-'}
            </div>
            <p className="text-muted-foreground mt-2 text-xs">{status?.lastRun ? formatDate(status.lastRun) : ''}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('jobDetails.nextRun')}</CardDescription>
            <CardAction>
              <IconClock className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="text-2xl leading-none font-semibold tracking-tight lowercase">
              {status?.nextRun ? formatRelativeTime(status.nextRun, t) : '-'}
            </div>
            <p className="text-muted-foreground mt-2 text-xs">{status?.nextRun ? formatDate(status.nextRun) : ''}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('jobDetails.totalBackups')}</CardDescription>
            <CardAction>
              <IconArchive className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="text-3xl leading-none font-semibold tracking-tight">{backupsCount}</div>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
