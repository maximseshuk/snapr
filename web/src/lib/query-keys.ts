export const queryKeys = {
  jobs: {
    all: ['jobs'] as const,
  },
  job: (name: string) => ({
    all: ['job', name] as const,
    config: ['job', name, 'config'] as const,
    status: ['job', name, 'status'] as const,
    backups: ['job', name, 'backups'] as const,
    logs: ['job', name, 'logs'] as const,
  }),
  system: {
    status: ['system', 'status'] as const,
    logs: ['system', 'logs'] as const,
  },
  github: {
    latestRelease: (repo: string) => ['github', 'release', repo] as const,
  },
} as const
