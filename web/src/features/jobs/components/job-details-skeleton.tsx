import { Main } from '@/components/layout/main'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

const STATUS_CARD_COUNT = 5

export const JobDetailsSkeleton = () => {
  return (
    <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
      <div className="flex flex-col items-start gap-2 md:flex-row md:flex-wrap md:items-start md:justify-between">
        <div className="max-w-full min-w-0">
          <Skeleton className="h-7 w-72" />
        </div>
        <div className="flex shrink-0 flex-wrap gap-2">
          <Skeleton className="size-9" />
          <Skeleton className="h-9 w-28" />
        </div>
      </div>

      <Card className="!py-2 md:hidden">
        <CardContent className="flex flex-col gap-3 px-4 py-2">
          <div className="grid grid-cols-2 grid-rows-[auto_auto] gap-x-3 gap-y-1">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-12" />
          </div>
          <div className="grid grid-cols-2 grid-rows-[auto_auto] gap-x-3 gap-y-1">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-20" />
          </div>
          <div className="flex flex-col gap-1">
            <Skeleton className="h-3 w-16" />
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-3 w-40" />
          </div>
        </CardContent>
      </Card>

      <div className="hidden gap-4 md:grid md:grid-cols-2 lg:grid-cols-5">
        {Array.from({ length: STATUS_CARD_COUNT }).map((_, index) => (
          <Card key={index}>
            <CardHeader>
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent className="space-y-2">
              <Skeleton className="h-7 w-32" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="flex gap-2">
        <Skeleton className="h-9 w-24" />
        <Skeleton className="h-9 w-28" />
        <Skeleton className="h-9 w-32" />
      </div>

      <Card className="flex flex-1 flex-col">
        <CardHeader>
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    </Main>
  )
}
