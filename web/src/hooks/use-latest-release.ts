import { useQuery } from '@tanstack/react-query'

import { GITHUB_REPO, GITHUB_REPO_URL } from '@/lib/constants'
import { queryKeys } from '@/lib/query-keys'

interface GithubRelease {
  tag_name: string
  html_url: string
  name: string
  published_at: string
}

const normalize = (v: string) => v.replace(/^v/, '').trim()

export const useLatestRelease = (currentVersion?: string) => {
  const query = useQuery<GithubRelease>({
    queryKey: queryKeys.github.latestRelease(GITHUB_REPO),
    queryFn: async () => {
      const res = await fetch(`https://api.github.com/repos/${GITHUB_REPO}/releases/latest`)
      if (!res.ok) throw new Error(`GitHub ${res.status}`)
      return res.json()
    },
    enabled: !!currentVersion,
    staleTime: 60 * 60_000,
    gcTime: 24 * 60 * 60_000,
    retry: false,
  })

  const latest = query.data?.tag_name
  const isDev = currentVersion === 'dev'
  const hasUpdate = isDev || (!!currentVersion && !!latest && normalize(currentVersion) !== normalize(latest))

  return {
    latestVersion: latest ?? (isDev ? 'latest' : undefined),
    releaseUrl: query.data?.html_url ?? `${GITHUB_REPO_URL}/releases`,
    hasUpdate,
  }
}
