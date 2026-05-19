export interface Job {
  name: string
  schedule: string
  sourcesCount: number
  storagesCount: number
  status: 'idle' | 'running'
  active: boolean
  lastRun?: string
  nextRun?: string
  lastResult?: {
    success: boolean
    duration?: string
    error?: string
  }
}

export interface JobDetail {
  name: string
  schedule: string
  sources?: Source[]
  storages?: Storage[]
  defaultStorage?: string
  compression?: string
  retention?: Retention
  hasBeforeScript?: boolean
  hasAfterScript?: boolean
  encryption?: Encryption
  notifiers?: Notifier[]
  split?: Split
}

export interface Split {
  chunkSize: string
}

export type SourceType = 'postgresql' | 'mysql' | 'mariadb' | 'mongodb' | 'redis' | 'sqlite' | 'local' | 'bunny' | 's3'

export interface Source {
  type: SourceType
  path?: string
  excludes?: string[]
  host?: string
  port?: number
  username?: string
  database?: string
  tables?: string[]
  excludeTables?: string[]
  allDatabases?: boolean
  hasUri?: boolean
  oplog?: boolean
  endpoint?: string
  zoneName?: string
  syncPath?: string
  extraParams?: Record<string, string>
}

export type StorageType = 's3' | 'local' | 'bunny' | 'sftp' | 'webdav'

export interface Storage {
  type: StorageType
  name?: string
  path?: string
  bucket?: string
  region?: string
  endpoint?: string
  storageClass?: string
  zoneName?: string
  pullZoneHostname?: string
  host?: string
  port?: number
  username?: string
  hasPrivateKey?: boolean
  hasKnownHosts?: boolean
  strictHostKey?: boolean
  hasUrl?: boolean
  urlHost?: string
}

export interface Retention {
  last: number
}

export interface Encryption {
  type?: string
  cipher?: string
}

export type NotifierType = 'webhook' | 'telegram' | 'email'

export interface Notifier {
  name?: string
  type: NotifierType
  onSuccess: boolean
  onFailure: boolean
  urlHost?: string
  chatId?: string
  from?: string
  to?: string[]
  smtpHost?: string
}

export interface JobStatus {
  name: string
  status: 'idle' | 'running'
  active: boolean
  lastRun?: string
  nextRun?: string
  lastResult?: {
    success: boolean
    duration?: string
    error?: string
  }
}

export interface System {
  uptime: number
  status: string
  environment: string
  jobsCount: number
  version?: string
}

export interface JobExecutionRequest {
  jobName: string
}

export interface JobExecutionResponse {
  job: string
  startedAt: string
}

export interface Backup {
  id: string
  jobName: string
  createdAt: string
  size: number
  path: string
  storageType: string
  isSplit?: boolean
  partsCount?: number
  fullDownloadSupported: boolean
}

// LogLine is a single rendered log entry. The backend serialises events as
// already-formatted ANSI strings, the viewer parses them client-side for
// colour highlights.
export type LogLine = string

export interface JobLogsResponse {
  job: string
  logs: LogLine[]
  status: 'idle' | 'running' | 'success' | 'failed'
  startTime?: string
  endTime?: string
  duration?: string
  error?: string
}

export interface SystemLogsResponse {
  logs: LogLine[]
}

export interface Settings {
  logLimits: {
    jobLogs: number
    systemLogs: number
  }
  logs: {
    system: boolean
    perJob: boolean
  }
  permissions?: {
    allowManualRun: boolean
    allowBackupDownload: boolean
    showConfig?: boolean
  }
}
