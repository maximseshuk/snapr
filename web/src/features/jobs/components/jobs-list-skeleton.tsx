import { Main } from '@/components/layout/main'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { cn } from '@/lib/utils'

const COLUMN_COUNT = 6
const ROW_COUNT = 5

export const JobsListSkeleton = () => {
  return (
    <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
      <div className="flex flex-col items-start gap-2 md:flex-row md:flex-wrap md:items-start md:justify-between">
        <div className="max-w-full min-w-0">
          <Skeleton className="h-7 w-48" />
        </div>
        <Skeleton className="size-9 shrink-0" />
      </div>

      <div className={cn('max-sm:has-[div[role="toolbar"]]:mb-16', 'flex flex-1 flex-col gap-4')}>
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-[150px] lg:w-[250px]" />
          <Skeleton className="h-8 w-24" />
        </div>

        <div className="overflow-hidden rounded-md border">
          <Table>
            <TableHeader className="sticky top-0 z-10">
              <TableRow>
                {Array.from({ length: COLUMN_COUNT }).map((_, index) => (
                  <TableHead key={index} className="bg-background">
                    <Skeleton className="h-4 w-24" />
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {Array.from({ length: ROW_COUNT }).map((_, rowIndex) => (
                <TableRow key={rowIndex}>
                  {Array.from({ length: COLUMN_COUNT }).map((_, colIndex) => (
                    <TableCell key={colIndex} className="bg-background">
                      <Skeleton className="h-4 w-full max-w-[180px]" />
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>

        <div className="mt-auto flex items-center justify-between">
          <Skeleton className="h-8 w-[130px]" />
          <div className="flex items-center gap-2">
            <Skeleton className="size-8" />
            <Skeleton className="size-8" />
            <Skeleton className="size-8" />
            <Skeleton className="size-8" />
            <Skeleton className="size-8" />
          </div>
        </div>
      </div>
    </Main>
  )
}
