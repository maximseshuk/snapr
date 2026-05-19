const isClient = typeof window !== 'undefined'

export const getItem = (key: string): string | undefined => {
  if (!isClient) return undefined
  try {
    return window.localStorage.getItem(key) ?? undefined
  } catch (error) {
    console.error(error)
    return undefined
  }
}

export const setItem = (key: string, value: string) => {
  if (!isClient) return
  try {
    window.localStorage.setItem(key, value)
  } catch (error) {
    console.error(error)
  }
}

export const STORAGE_KEYS = {
  theme: 'snapr.theme',
  language: 'snapr.language',
  sidebarOpen: 'snapr.sidebar.open',
  jobsTableVisibility: 'snapr.jobs.tableVisibility',
  jobBackupsTableVisibility: 'snapr.jobBackups.tableVisibility',
} as const
