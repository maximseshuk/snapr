import type { Icon } from '@tabler/icons-react'

export interface NavItem {
  title: string
  url: string
  icon?: Icon
  matchPaths?: string[]
  items?: { title: string; url: string }[]
}

export const checkIsActive = (currentPath: string, itemUrl: string, matchPaths?: string[]) => {
  const cleanPath = currentPath.split('?')[0]
  const cleanItemUrl = itemUrl.split('?')[0]

  if (cleanPath === cleanItemUrl) return true

  if (matchPaths) {
    return matchPaths.some((p) => {
      if (cleanPath === p) return true
      if (p === '/') return false
      return cleanPath.startsWith(p.endsWith('/') ? p : p + '/')
    })
  }

  return false
}
