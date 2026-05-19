import { IconEye, IconLoader, IconPlayerPlay, IconPlayerStop } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { StopJobDialog } from './stop-job-dialog'

interface JobRowActionsProps {
  jobName: string
  isRunning: boolean
  isCancelling: boolean
  canRunJobs: boolean
  onRun: (name: string) => void
  onCancel: (name: string) => void
}

export const JobRowActions = ({
  jobName,
  isRunning,
  isCancelling,
  canRunJobs,
  onRun,
  onCancel,
}: JobRowActionsProps) => {
  const { t } = useTranslation()
  const [stopOpen, setStopOpen] = useState(false)

  return (
    <div className="flex items-center justify-end gap-1">
      <Button variant="outline" size="icon" className="h-8 w-8" asChild aria-label={t('jobs.viewDetails')}>
        <Link to={`/jobs/$jobName`} params={{ jobName }}>
          <IconEye className="size-4" />
        </Link>
      </Button>
      {canRunJobs && isRunning && (
        <StopJobDialog
          open={stopOpen}
          onOpenChange={setStopOpen}
          onConfirm={() => onCancel(jobName)}
          disabled={isCancelling}
          trigger={
            <Button
              variant="outline"
              size="icon"
              className="border-destructive/30 text-destructive hover:bg-destructive hover:border-destructive h-8 w-8 hover:text-white"
              disabled={isCancelling}
              aria-label={t('jobs.cancel')}
            >
              {isCancelling ? <IconLoader className="size-4 animate-spin" /> : <IconPlayerStop className="size-4" />}
            </Button>
          }
        />
      )}
      {canRunJobs && !isRunning && (
        <Button
          variant="outline"
          size="icon"
          className="h-8 w-8"
          onClick={() => onRun(jobName)}
          aria-label={t('jobs.runNow')}
        >
          <IconPlayerPlay className="size-4" />
        </Button>
      )}
    </div>
  )
}
