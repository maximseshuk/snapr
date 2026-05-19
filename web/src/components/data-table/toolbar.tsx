import { IconX } from '@tabler/icons-react'
import { type Table } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { DataTableViewOptions } from './view-options'

interface DataTableToolbarProps<TData> {
  table: Table<TData>
  searchPlaceholder?: string
}

export const DataTableToolbar = <TData,>({ table, searchPlaceholder = 'Search...' }: DataTableToolbarProps<TData>) => {
  const { t } = useTranslation()
  const isFiltered = table.getState().globalFilter

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2">
        <Input
          placeholder={searchPlaceholder}
          value={(table.getState().globalFilter ?? '') as string}
          onChange={(event) => table.setGlobalFilter(event.target.value)}
          className="h-8 w-[150px] lg:w-[250px]"
        />
        {isFiltered && (
          <Button variant="ghost" onClick={() => table.setGlobalFilter('')} className="h-8 px-2 lg:px-3">
            {t('table.reset')}
            <IconX className="ml-2 size-4" />
          </Button>
        )}
      </div>
      <DataTableViewOptions table={table} />
    </div>
  )
}
