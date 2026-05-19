import { Outlet, useMatches } from '@tanstack/react-router'
import { useState } from 'react'

import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { getItem, setItem, STORAGE_KEYS } from '@/lib/storage'
import { cn } from '@/lib/utils'

import { AppSidebar } from './app-sidebar'
import { Header } from './header'

const readSidebarOpen = () => getItem(STORAGE_KEYS.sidebarOpen) !== 'false'

export const AuthenticatedLayout = () => {
  const matches = useMatches()
  const backTo = matches
    .map((m) => (m.staticData as { backTo?: string } | undefined)?.backTo)
    .filter(Boolean)
    .pop()

  const [sidebarOpen, setSidebarOpen] = useState(readSidebarOpen)

  const handleOpenChange = (open: boolean) => {
    setSidebarOpen(open)
    setItem(STORAGE_KEYS.sidebarOpen, String(open))
  }

  return (
    <SidebarProvider open={sidebarOpen} onOpenChange={handleOpenChange}>
      <AppSidebar />
      <SidebarInset
        className={cn(
          '@container/content',
          'has-data-[layout=fixed]:h-svh',
          'peer-data-[variant=inset]:has-data-[layout=fixed]:h-[calc(100svh-(var(--spacing)*4))]',
        )}
      >
        <Header backTo={backTo} />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
