import { IconRefresh } from '@tabler/icons-react'
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { DataTablePagination, DataTableToolbar } from '@/components/data-table'
import { Main } from '@/components/layout/main'
import { PageHeader } from '@/components/page-header'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useLocalStorage } from '@/hooks/use-local-storage'
import { usePermissions } from '@/hooks/use-settings'
import { STORAGE_KEYS } from '@/lib/storage'
import { cn } from '@/lib/utils'

import { useJobActions } from '../hooks/use-job-actions'
import { useJobs } from '../hooks/use-jobs'
import { JobsListSkeleton } from './jobs-list-skeleton'
import { getJobsTableColumns } from './jobs-table-columns'

declare module '@tanstack/react-table' {
  interface ColumnMeta<TData, TValue> {
    className?: string
  }
}

export const JobsList = () => {
  const { t, i18n } = useTranslation()
  const { canRunJobs } = usePermissions()
  const { jobs, loading, refreshing, loadData } = useJobs()
  const { cancellingJob, handleRunJob, handleCancelJob } = useJobActions()

  const [globalFilter, setGlobalFilter] = useState('')
  const [sorting, setSorting] = useState<SortingState>([])
  const [columnVisibility, setColumnVisibility] = useLocalStorage(STORAGE_KEYS.jobsTableVisibility, {})

  const columns = useMemo(
    () =>
      getJobsTableColumns({
        t,
        locale: i18n.language,
        handleRunJob,
        handleCancelJob,
        cancellingJob,
        canRunJobs,
      }),
    [t, i18n.language, handleRunJob, handleCancelJob, cancellingJob, canRunJobs],
  )

  const table = useReactTable({
    data: jobs,
    columns,
    state: {
      globalFilter,
      sorting,
      columnVisibility,
    },
    onGlobalFilterChange: setGlobalFilter,
    onSortingChange: setSorting,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    autoResetPageIndex: false,
    globalFilterFn: (row, _columnId, filterValue) => {
      const name = String(row.getValue('name')).toLowerCase()
      const schedule = String(row.getValue('schedule')).toLowerCase()
      const searchValue = String(filterValue).toLowerCase()
      return name.includes(searchValue) || schedule.includes(searchValue)
    },
  })

  if (loading) {
    return <JobsListSkeleton />
  }

  return (
    <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
      <PageHeader
        title={t('jobs.title')}
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

      <div className={cn('max-sm:has-[div[role="toolbar"]]:mb-16', 'flex flex-1 flex-col gap-4')}>
        <DataTableToolbar table={table} searchPlaceholder={t('jobs.search')} />

        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="sticky top-0 z-10">
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead
                      key={header.id}
                      className={cn(
                        'bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted',
                        header.column.columnDef.meta?.className ?? '',
                      )}
                    >
                      {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows?.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow key={row.id}>
                    {row.getVisibleCells().map((cell) => (
                      <TableCell
                        key={cell.id}
                        className={cn(
                          'bg-background group-hover/row:bg-muted group-data-[state=selected]/row:bg-muted',
                          cell.column.columnDef.meta?.className ?? '',
                        )}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center">
                    {t('jobs.noJobs')}
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>

        <DataTablePagination table={table} className="mt-auto" />
      </div>
    </Main>
  )
}
