import { Link, useLocation } from '@tanstack/react-router'

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { checkIsActive, type NavItem } from '@/lib/navigation'

export const NavMain = ({ items }: { items: NavItem[] }) => {
  const { pathname } = useLocation()
  const { isMobile, setOpenMobile } = useSidebar()

  const handleNav = () => {
    if (isMobile) setOpenMobile(false)
  }

  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu className="gap-0.5">
          {items.map((item) => (
            <SidebarMenuItem key={item.title}>
              <SidebarMenuButton
                tooltip={item.title}
                asChild
                isActive={checkIsActive(pathname, item.url, item.matchPaths)}
              >
                <Link to={item.url} onClick={handleNav}>
                  {item.icon && <item.icon />}
                  <span>{item.title}</span>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
