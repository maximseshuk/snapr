import { IconBrandGithub, IconClockPlay, IconFileText, IconSparkles } from '@tabler/icons-react'
import * as React from 'react'
import { useTranslation } from 'react-i18next'

import { Logo } from '@/components/logo'
import { NavMain } from '@/components/nav-main'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { useLatestRelease } from '@/hooks/use-latest-release'
import { useSystem } from '@/hooks/use-system'
import { GITHUB_REPO_URL } from '@/lib/constants'

export const AppSidebar = ({ ...props }: React.ComponentProps<typeof Sidebar>) => {
  const { t } = useTranslation()

  const { data: systemData } = useSystem({ staleTime: 60_000 })

  const currentVersion = systemData?.version
  const { latestVersion, releaseUrl, hasUpdate } = useLatestRelease(currentVersion)

  const data = {
    navMain: [
      {
        title: t('header.jobs'),
        url: '/',
        icon: IconClockPlay,
        matchPaths: ['/', '/jobs'],
      },
      {
        title: t('system.title'),
        url: '/system',
        icon: IconFileText,
      },
    ],
  }

  return (
    <Sidebar variant="inset" collapsible="icon" {...props}>
      <SidebarHeader>
        <div className="flex items-center gap-2 p-1.5">
          <span className="inline-flex size-5 shrink-0">
            <Logo size={20} />
          </span>
          <span className="text-base font-semibold group-data-[collapsible=icon]:hidden">
            snapr
            {currentVersion && (
              <sup className="text-muted-foreground ml-1 text-[0.6rem] font-normal">
                v{currentVersion.replace(/^v/, '')}
              </sup>
            )}
          </span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={data.navMain} />
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu className="gap-2">
          {hasUpdate && releaseUrl && (
            <SidebarMenuItem>
              <SidebarMenuButton
                asChild
                size="lg"
                tooltip={`${t('sidebar.updateAvailable')} (${latestVersion})`}
                className="text-status-info-foreground! bg-status-info! hover:bg-status-info/80! hover:text-status-info-foreground! active:bg-status-info! active:text-status-info-foreground! border-status-info-border border group-data-[collapsible=icon]:justify-center! group-data-[collapsible=icon]:px-0!"
              >
                <a href={releaseUrl} target="_blank" rel="noopener noreferrer">
                  <IconSparkles />
                  <span className="flex flex-col gap-0.5 leading-tight group-data-[collapsible=icon]:hidden">
                    <span className="text-sm font-medium">{t('sidebar.updateAvailable')}</span>
                    <span className="text-status-info-foreground/75 text-xs">
                      {currentVersion} → {latestVersion}
                    </span>
                  </span>
                </a>
              </SidebarMenuButton>
            </SidebarMenuItem>
          )}
          <SidebarMenuItem>
            <SidebarMenuButton asChild>
              <a href={GITHUB_REPO_URL} target="_blank" rel="noopener noreferrer">
                <IconBrandGithub />
                <span>GitHub</span>
              </a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
