import { cn } from '@/lib/utils'

interface LogoProps extends React.SVGProps<SVGSVGElement> {
  size?: number
}

export const Logo = ({ size = 64, className, ...props }: LogoProps) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={cn('text-foreground', className)}
      {...props}
    >
      <rect x="10" y="10" width="34" height="34" rx="6" stroke="currentColor" strokeWidth="3" opacity="0.35" />
      <rect x="16" y="16" width="34" height="34" rx="6" stroke="currentColor" strokeWidth="3" opacity="0.6" />
      <rect x="22" y="22" width="34" height="34" rx="6" fill="var(--primary)" stroke="currentColor" strokeWidth="3" />
    </svg>
  )
}
