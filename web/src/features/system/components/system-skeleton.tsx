import { Main } from '@/components/layout/main'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

const STATUS_CARD_COUNT = 3

export const SystemSkeleton = () => {
  return (
    <Main className="flex flex-1 flex-col gap-4">
      <div className="flex flex-col items-start gap-2 md:flex-row md:flex-wrap md:items-start md:justify-between">
        <div className="max-w-full min-w-0">
          <Skeleton className="h-7 w-48" />
        </div>
        <Skeleton className="size-9 shrink-0" />
      </div>

      <Card className="!py-2 md:hidden">
        <CardContent className="grid grid-cols-3 grid-rows-[auto_auto] gap-x-3 gap-y-1 px-4 py-2">
          {Array.from({ length: STATUS_CARD_COUNT }).map((_, index) => (
            <Skeleton key={`label-${index}`} className="h-3 w-16" />
          ))}
          {Array.from({ length: STATUS_CARD_COUNT }).map((_, index) => (
            <Skeleton key={`value-${index}`} className="h-5 w-12" />
          ))}
        </CardContent>
      </Card>

      <div className="hidden gap-4 md:grid md:grid-cols-3">
        {Array.from({ length: STATUS_CARD_COUNT }).map((_, index) => (
          <Card key={index}>
            <CardHeader>
              <Skeleton className="h-4 w-24" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-32" />
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="flex flex-1 flex-col gap-3 !py-4 md:gap-6 md:!py-6">
        <CardHeader className="px-4 md:px-6">
          <Skeleton className="h-5 w-32" />
        </CardHeader>
        <CardContent className="flex flex-1 flex-col px-4 md:px-6">
          <Skeleton className="h-96 w-full" />
        </CardContent>
      </Card>
    </Main>
  )
}
