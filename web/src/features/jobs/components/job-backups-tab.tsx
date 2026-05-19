import { IconDownload } from '@tabler/icons-react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type SortingState,
  useReactTable,
} from '@tanstack/react-table'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { Backup } from '@/types/api'

import { DataTableColumnHeader, DataTablePagination, DataTableViewOptions } from '@/components/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useLocalStorage } from '@/hooks/use-local-storage'
import { partFilenames } from '@/lib/split-parts'
import { STORAGE_KEYS } from '@/lib/storage'

import { formatBytes, formatDate } from '../utils/formatters'

interface JobBackupsTabProps {
  backups: Backup[]
  onDownload: (backupId: string) => void
  onDownloadPart: (partFilename: string) => void
  canDownload?: boolean
}

export const JobBackupsTab = ({ backups, onDownload, onDownloadPart, canDownload = true }: JobBackupsTabProps) => {
  const { t } = useTranslation()
  const [backupsSorting, setBackupsSorting] = useState<SortingState>([{ id: 'createdAt', desc: true }])
  const [columnVisibility, setColumnVisibility] = useLocalStorage(STORAGE_KEYS.jobBackupsTableVisibility, {})

  const backupsColumns = useMemo<ColumnDef<Backup>[]>(
    () => [
      {
        accessorKey: 'createdAt',
        header: ({ column }) => <DataTableColumnHeader column={column} title={t('jobDetails.createdAt')} />,
        cell: ({ row }) => formatDate(row.original.createdAt),
        meta: { className: 'pl-4' },
      },
      {
        accessorKey: 'size',
        header: ({ column }) => <DataTableColumnHeader column={column} title={t('jobDetails.size')} />,
        cell: ({ row }) => {
          const { isSplit, partsCount } = row.original
          const splitBadge =
            isSplit && partsCount && partsCount > 0 ? (
              <Badge variant="secondary" className="ml-2">
                {t('jobDetails.partsCount', { count: partsCount })}
              </Badge>
            ) : null
          return (
            <span className="inline-flex items-center">
              {formatBytes(row.original.size)}
              {splitBadge}
            </span>
          )
        },
      },
      {
        accessorKey: 'storageType',
        header: ({ column }) => <DataTableColumnHeader column={column} title={t('jobDetails.storageType')} />,
        cell: ({ row }) => <Badge variant="outline">{row.original.storageType}</Badge>,
        enableSorting: false,
      },
      {
        accessorKey: 'path',
        header: ({ column }) => <DataTableColumnHeader column={column} title={t('jobDetails.path')} />,
        cell: ({ row }) => (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="block max-w-[280px] truncate font-mono text-xs">{row.original.path}</span>
              </TooltipTrigger>
              <TooltipContent className="max-w-md font-mono text-[11px] break-all">{row.original.path}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ),
        enableSorting: false,
      },
      {
        id: 'actions',
        header: ({ column }) => (
          <div className="flex justify-end">
            <DataTableColumnHeader column={column} title={t('jobDetails.actions')} />
          </div>
        ),
        cell: ({ row }) => {
          if (!canDownload) return null
          const backup = row.original
          const partCount = backup.partsCount ?? 0
          const hasParts = backup.isSplit && partCount > 0
          if (!hasParts) {
            return (
              <div className="flex justify-end">
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => onDownload(backup.id)}
                  aria-label={t('jobDetails.download')}
                >
                  <IconDownload className="size-4" />
                </Button>
              </div>
            )
          }
          const partNames = partFilenames(backup.id, partCount)
          return (
            <div className="flex justify-end">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="icon" aria-label={t('jobDetails.download')}>
                    <IconDownload className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="max-h-72 w-48 overflow-auto">
                  {backup.fullDownloadSupported && (
                    <>
                      <DropdownMenuItem onSelect={() => onDownload(backup.id)}>
                        {t('jobDetails.downloadFull')}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                    </>
                  )}
                  {partNames.map((filename, index) => (
                    <DropdownMenuItem key={filename} onSelect={() => onDownloadPart(filename)}>
                      {t('jobDetails.downloadPart', { n: index + 1 })}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          )
        },
        meta: { className: 'pr-4' },
        enableHiding: false,
        enableSorting: false,
      },
    ],
    [t, canDownload, onDownload, onDownloadPart],
  )

  const backupsTable = useReactTable({
    data: backups,
    columns: backupsColumns,
    state: {
      sorting: backupsSorting,
      columnVisibility,
    },
    onSortingChange: setBackupsSorting,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    autoResetPageIndex: false,
  })

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <DataTableViewOptions table={backupsTable} />
      </div>

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            {backupsTable.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} className={header.column.columnDef.meta?.className ?? ''}>
                    {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {backupsTable.getRowModel().rows?.length ? (
              backupsTable.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className={cell.column.columnDef.meta?.className ?? ''}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={backupsColumns.length} className="h-24 text-center">
                  {t('jobDetails.noBackups')}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <DataTablePagination table={backupsTable} />
    </div>
  )
}
