import { IconBriefcase, IconClock, IconRefresh } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout/main'
import { LogViewer } from '@/components/log-viewer'
import { PageHeader } from '@/components/page-header'
import { statusPillVariants } from '@/components/status-variants'
import { Button } from '@/components/ui/button'
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatUptime } from '@/features/jobs/utils/formatters'
import { cn } from '@/lib/utils'

import { useSystemLogs } from '../hooks/use-system-logs'
import { SystemSkeleton } from './system-skeleton'

export const System = () => {
  const { t } = useTranslation()
  const { entries, resetToken, disabled, systemData, loading, refreshing, settings, loadData } = useSystemLogs()

  const isOk = systemData?.status === 'ok'
  const statusPill = statusPillVariants({ tone: isOk ? 'success' : 'failed' })

  if (loading) {
    return <SystemSkeleton />
  }

  return (
    <Main className="flex flex-1 flex-col gap-4">
      <PageHeader
        title={t('system.title')}
        actions={
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
        }
      />

      <Card className="!py-2 md:hidden">
        <CardContent className="grid grid-cols-3 grid-rows-[auto_auto] gap-x-3 gap-y-1 px-4 py-2">
          <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
            <span
              className={cn(
                'mt-1 inline-block size-2 shrink-0 animate-pulse rounded-full',
                isOk ? 'bg-emerald-500' : 'bg-rose-500',
              )}
            />
            <span>{t('system.status')}</span>
          </div>
          <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
            <IconClock className="mt-0.5 size-3.5 shrink-0" />
            <span>{t('system.uptime')}</span>
          </div>
          <div className="text-muted-foreground flex items-start gap-1.5 text-xs">
            <IconBriefcase className="mt-0.5 size-3.5 shrink-0" />
            <span>{t('system.totalJobs')}</span>
          </div>
          <div
            className={cn(
              'inline-flex w-fit items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-semibold',
              statusPill,
            )}
          >
            {isOk ? t('status.running') : t('status.failed')}
          </div>
          <div className="truncate text-sm font-semibold">
            {systemData?.uptime ? formatUptime(systemData.uptime) : 'N/A'}
          </div>
          <div className="text-sm font-semibold">{systemData?.jobsCount || 0}</div>
        </CardContent>
      </Card>

      <div className="hidden gap-4 md:grid md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardDescription>{t('system.status')}</CardDescription>
            <CardAction>
              <span
                className={cn(
                  'inline-block size-2.5 animate-pulse rounded-full',
                  isOk ? 'bg-emerald-500' : 'bg-rose-500',
                )}
              />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div
              className={cn(
                'inline-flex items-center gap-2 rounded-full border px-3 py-1 text-base font-semibold',
                statusPill,
              )}
            >
              {isOk ? t('status.running') : t('status.failed')}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('system.uptime')}</CardDescription>
            <CardAction>
              <IconClock className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="text-3xl leading-none font-semibold tracking-tight">
              {systemData?.uptime ? formatUptime(systemData.uptime) : 'N/A'}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardDescription>{t('system.totalJobs')}</CardDescription>
            <CardAction>
              <IconBriefcase className="text-muted-foreground size-4" />
            </CardAction>
          </CardHeader>
          <CardContent>
            <div className="text-3xl leading-none font-semibold tracking-tight">{systemData?.jobsCount || 0}</div>
          </CardContent>
        </Card>
      </div>

      <Card className="flex min-h-[400px] flex-1 flex-col gap-3 overflow-hidden !py-4">
        <CardHeader>
          <CardTitle>{t('system.logs')}</CardTitle>
        </CardHeader>
        <CardContent className="flex min-h-0 min-w-0 flex-1 flex-col">
          <LogViewer
            entries={entries}
            resetKey={resetToken}
            emptyMessage={disabled ? t('logs.disabledSystem') : t('logs.empty')}
            autoScrollLabel={t('logs.autoScroll')}
            searchPlaceholder={t('logs.searchPlaceholder')}
            downloadFileName="system.log"
            fullscreenLabel={t('logs.fullscreen')}
            exitFullscreenLabel={t('logs.exitFullscreen')}
            downloadLabel={t('logs.download')}
            wrapLabel={t('logs.wrap')}
            unwrapLabel={t('logs.unwrap')}
            jumpToTopLabel={t('logs.jumpToTop')}
            jumpToBottomLabel={t('logs.jumpToBottom')}
            toolbar={
              !disabled &&
              settings && (
                <div className="text-muted-foreground text-sm">
                  {t('logs.showingLastLines', { count: settings.logLimits.systemLogs })}
                </div>
              )
            }
          />
        </CardContent>
      </Card>
    </Main>
  )
}
