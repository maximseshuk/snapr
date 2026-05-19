import { cn } from '@/lib/utils'

interface PageHeaderProps {
  title: string
  actions?: React.ReactNode
  className?: string
}

export const PageHeader = ({ title, actions, className }: PageHeaderProps) => {
  return (
    <div
      className={cn(
        'flex flex-col items-start gap-2 md:flex-row md:flex-wrap md:items-start md:justify-between',
        className,
      )}
    >
      <div className="max-w-full min-w-0">
        <h2 className="truncate text-2xl font-bold tracking-tight">{title}</h2>
      </div>
      {actions && <div className="flex shrink-0 flex-wrap gap-2">{actions}</div>}
    </div>
  )
}
