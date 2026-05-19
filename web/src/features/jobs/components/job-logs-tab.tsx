import { useTranslation } from 'react-i18next'

import type { LogLine, Settings } from '@/types/api'

import { LogViewer } from '@/components/log-viewer'

interface JobLogsTabProps {
  jobName: string
  entries: LogLine[]
  disabled: boolean
  settings: Settings | null
}

export const JobLogsTab = ({ jobName, entries, disabled, settings }: JobLogsTabProps) => {
  const { t } = useTranslation()

  return (
    <LogViewer
      entries={entries}
      resetKey={jobName}
      emptyMessage={disabled ? t('logs.disabledPerJob') : t('logs.empty')}
      autoScrollLabel={t('logs.autoScroll')}
      searchPlaceholder={t('logs.searchPlaceholder')}
      downloadFileName={`${jobName}.log`}
      fullscreenLabel={t('logs.fullscreen')}
      exitFullscreenLabel={t('logs.exitFullscreen')}
      downloadLabel={t('logs.download')}
      wrapLabel={t('logs.wrap')}
      unwrapLabel={t('logs.unwrap')}
      jumpToTopLabel={t('logs.jumpToTop')}
      jumpToBottomLabel={t('logs.jumpToBottom')}
      className="flex h-full min-h-[400px] flex-1 flex-col"
      toolbar={
        !disabled &&
        settings && (
          <div className="text-muted-foreground text-sm">
            {t('logs.showingLastLines', { count: settings.logLimits.jobLogs })}
          </div>
        )
      }
    />
  )
}
