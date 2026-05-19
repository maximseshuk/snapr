import { IconArrowLeft, IconLogout } from '@tabler/icons-react'
import { Link, useRouter } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { LanguageSelector } from '@/components/language-selector'
import { ThemeToggle } from '@/components/theme-toggle'
import { Button } from '@/components/ui/button'
import { SidebarTrigger, useSidebar } from '@/components/ui/sidebar'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { useAuth } from '@/contexts/auth-context'
import { cn } from '@/lib/utils'

type HeaderProps = React.HTMLAttributes<HTMLElement> & {
  backTo?: string
  ref?: React.Ref<HTMLElement>
}

const SCROLL_SHADOW_THRESHOLD = 10

export const Header = ({ className, backTo, ...props }: HeaderProps) => {
  const { isAuthenticated, authEnabled, logout } = useAuth()
  const { state, isMobile, openMobile } = useSidebar()
  const router = useRouter()
  const { t } = useTranslation()
  const sidebarOpen = isMobile ? openMobile : state === 'expanded'
  const [isScrolled, setIsScrolled] = useState(false)

  useEffect(() => {
    const onScroll = () => {
      const top = document.body.scrollTop || document.documentElement.scrollTop
      setIsScrolled(top > SCROLL_SHADOW_THRESHOLD)
    }
    onScroll()
    document.addEventListener('scroll', onScroll, { passive: true })
    return () => document.removeEventListener('scroll', onScroll)
  }, [])

  const handleLogout = async () => {
    await logout()
    toast.success(t('success.logout'))
    await router.navigate({ to: '/login' })
    await router.invalidate()
  }

  return (
    <header
      className={cn(
        'bg-background/80 supports-[backdrop-filter]:bg-background/60 sticky top-0 z-30 h-16 rounded-t-xl backdrop-blur',
        isScrolled &&
          'after:pointer-events-none after:absolute after:inset-x-0 after:-bottom-1 after:h-1 after:bg-gradient-to-b after:from-black/10 after:to-transparent after:content-[""]',
        className,
      )}
      {...props}
    >
      <div className="flex h-full items-center gap-3 p-4 sm:gap-4">
        <SidebarTrigger
          variant="outline"
          className="max-md:scale-125"
          aria-label={sidebarOpen ? t('sidebar.close') : t('sidebar.open')}
        />
        {backTo && (
          <Button variant="ghost" size="icon" className="size-7 max-md:scale-125" asChild aria-label={t('common.back')}>
            <Link to={backTo}>
              <IconArrowLeft />
            </Link>
          </Button>
        )}
        <div className="flex-1" />
        <div className="flex items-center gap-3">
          <LanguageSelector />
          <ThemeToggle />
          {authEnabled && isAuthenticated && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handleLogout}
                  aria-label={t('header.logout')}
                  className="scale-95 rounded-full"
                >
                  <IconLogout className="size-[1.2rem]" />
                  <span className="sr-only">{t('header.logout')}</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t('header.logout')}</TooltipContent>
            </Tooltip>
          )}
        </div>
      </div>
    </header>
  )
}
