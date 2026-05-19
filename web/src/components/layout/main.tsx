import { cn } from '@/lib/utils'

type MainProps = React.HTMLAttributes<HTMLElement>

export const Main = ({ className, ...props }: MainProps) => {
  return <main className={cn('px-4 pt-2 pb-6', className)} {...props} />
}
