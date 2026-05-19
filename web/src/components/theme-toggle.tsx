import { IconCheck, IconMoon, IconSun } from '@tabler/icons-react'
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu'
import { useTheme } from '@/contexts/theme-context'
import { cn } from '@/lib/utils'

const THEMES = ['light', 'dark', 'system'] as const

export const ThemeToggle = () => {
  const { theme, setTheme } = useTheme()
  const { t } = useTranslation()

  useEffect(() => {
    const themeColor = theme === 'dark' ? '#020817' : '#fff'
    const metaThemeColor = document.querySelector("meta[name='theme-color']")
    if (metaThemeColor) metaThemeColor.setAttribute('content', themeColor)
  }, [theme])

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="scale-95 rounded-full" aria-label={t('theme.toggle')}>
          <IconSun className="size-[1.2rem] scale-100 rotate-0 transition-all dark:scale-0 dark:-rotate-90" />
          <IconMoon className="absolute size-[1.2rem] scale-0 rotate-90 transition-all dark:scale-100 dark:rotate-0" />
          <span className="sr-only">{t('theme.toggle')}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {THEMES.map((value) => (
          <DropdownMenuItem key={value} onClick={() => setTheme(value)}>
            {t(`theme.${value}`)}
            <IconCheck size={14} className={cn('ms-auto', theme !== value && 'hidden')} />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
