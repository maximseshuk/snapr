import { IconCircleDashed, IconLoader2 } from '@tabler/icons-react'
import { Link } from '@tanstack/react-router'
import { type ColumnDef } from '@tanstack/react-table'

import type { Job } from '@/types/api'

import { DataTableColumnHeader } from '@/components/data-table'
import { statusPillVariants } from '@/components/status-variants'
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { formatDate } from '../utils/formatters'
import { describeCron } from '../utils/helpers'
import { JobRowActions } from './job-row-actions'

interface GetColumnsParams {
  t: (key: string) => string
  locale: string
  handleRunJob: (jobName: string) => void
  handleCancelJob: (jobName: string) => void
  cancellingJob: string | null
  canRunJobs: boolean
}

const getLastResultBadge = (result: Job['lastResult'] | undefined, t: (key: string) => string) => {
  if (!result) {
    return <span className="text-muted-foreground text-sm">-</span>
  }

  if (result.success) {
    return (
      <Badge variant="outline" className={statusPillVariants({ tone: 'success' })}>
        {t('status.success')}
      </Badge>
    )
  }

  return (
    <Badge variant="outline" className={statusPillVariants({ tone: 'failed' })}>
      {t('status.failed')}
    </Badge>
  )
}

export const getJobsTableColumns = ({
  t,
  locale,
  handleRunJob,
  handleCancelJob,
  cancellingJob,
  canRunJobs,
}: GetColumnsParams): ColumnDef<Job>[] => [
  {
    accessorKey: 'name',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.name')} />,
    cell: ({ row }) => (
      <Link
        to={`/jobs/$jobName`}
        params={{ jobName: row.original.name }}
        className="block max-w-[200px] truncate font-medium hover:underline"
        title={row.original.name}
      >
        {row.original.name}
      </Link>
    ),
    meta: {
      className: cn(
        'max-lg:drop-shadow-[0_1px_2px_rgb(0_0_0_/_0.1)] max-lg:dark:drop-shadow-[0_1px_2px_rgb(255_255_255_/_0.1)]',
        'pl-4 max-lg:sticky max-lg:start-0 max-lg:z-10',
      ),
    },
    enableHiding: false,
  },
  {
    accessorKey: 'schedule',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.schedule')} />,
    cell: ({ row }) => (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="text-muted-foreground cursor-help font-mono text-sm">{row.original.schedule}</span>
        </TooltipTrigger>
        <TooltipContent>{describeCron(row.original.schedule, locale)}</TooltipContent>
      </Tooltip>
    ),
    enableSorting: false,
  },
  {
    accessorKey: 'status',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.status')} />,
    cell: ({ row }) => (
      <Badge
        variant="outline"
        className={statusPillVariants({ tone: row.original.status === 'running' ? 'running' : 'idle' })}
      >
        {row.original.status === 'running' ? (
          <IconLoader2 className="size-4 animate-spin" />
        ) : (
          <IconCircleDashed className="size-4" />
        )}
        {row.original.status === 'running' ? t('status.running') : t('status.idle')}
      </Badge>
    ),
    meta: { className: 'w-[140px] min-w-[140px]' },
    size: 140,
  },
  {
    accessorKey: 'nextRun',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.nextRun')} />,
    cell: ({ row }) =>
      row.original.nextRun ? (
        <span className="text-muted-foreground text-sm">{formatDate(row.original.nextRun)}</span>
      ) : (
        <span className="text-muted-foreground text-sm">-</span>
      ),
  },
  {
    accessorKey: 'lastRun',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.lastRun')} />,
    cell: ({ row }) =>
      row.original.lastRun ? (
        <span className="text-muted-foreground text-sm">{formatDate(row.original.lastRun)}</span>
      ) : (
        <span className="text-muted-foreground text-sm">-</span>
      ),
  },
  {
    accessorKey: 'lastResult',
    header: ({ column }) => <DataTableColumnHeader column={column} title={t('column.lastResult')} />,
    cell: ({ row }) => getLastResultBadge(row.original.lastResult, t),
    meta: { className: 'w-[140px] min-w-[140px]' },
    size: 140,
  },
  {
    id: 'actions',
    header: ({ column }) => (
      <div className="flex justify-end">
        <DataTableColumnHeader column={column} title={t('column.actions')} />
      </div>
    ),
    cell: ({ row }) => (
      <JobRowActions
        jobName={row.original.name}
        isRunning={row.original.status === 'running'}
        isCancelling={cancellingJob === row.original.name}
        canRunJobs={canRunJobs}
        onRun={handleRunJob}
        onCancel={handleCancelJob}
      />
    ),
    meta: {
      className: 'pr-4',
    },
    enableHiding: false,
    enableSorting: false,
  },
]
