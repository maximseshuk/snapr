import type {
  Backup,
  Job,
  JobDetail,
  JobExecutionResponse,
  JobLogsResponse,
  JobStatus,
  Settings,
  System,
  SystemLogsResponse,
} from '@/types/api'

import i18n from '@/i18n'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api/v1'

class ApiClient {
  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Accept-Language': i18n.language,
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new Error(errorData.error || `HTTP ${response.status}`)
    }

    return response.json() as Promise<T>
  }

  async getSystem(): Promise<System> {
    return this.request<System>('/status')
  }

  async getSettings(): Promise<Settings> {
    return this.request<Settings>('/settings')
  }

  async getJobs(): Promise<Job[]> {
    const response = await this.request<{ jobs: Job[] }>('/jobs')
    return response.jobs ?? []
  }

  async getJobStatus(jobName: string): Promise<JobStatus> {
    return this.request<JobStatus>(`/jobs/${encodeURIComponent(jobName)}/status`)
  }

  async runJob(jobName: string): Promise<JobExecutionResponse> {
    return this.request<JobExecutionResponse>(`/jobs/${encodeURIComponent(jobName)}/run`, { method: 'POST' })
  }

  async cancelJob(jobName: string): Promise<{ job: string; cancelledAt: string }> {
    return this.request<{ job: string; cancelledAt: string }>(`/jobs/${encodeURIComponent(jobName)}/cancel`, {
      method: 'POST',
    })
  }

  async getJobConfig(jobName: string): Promise<JobDetail> {
    const response = await this.request<{ config: JobDetail }>(`/jobs/${encodeURIComponent(jobName)}/config`)
    return response.config
  }

  async getJobBackups(jobName: string): Promise<Backup[]> {
    try {
      const response = await this.request<{ backups: Backup[] }>(`/jobs/${encodeURIComponent(jobName)}/backups`)
      return response.backups ?? []
    } catch {
      return []
    }
  }

  async getJobLogs(jobName: string, tail: number): Promise<JobLogsResponse> {
    try {
      return await this.request<JobLogsResponse>(`/jobs/${encodeURIComponent(jobName)}/logs?tail=${tail}`)
    } catch {
      return { job: jobName, logs: [], status: 'idle' }
    }
  }

  async getSystemLogs(tail: number): Promise<SystemLogsResponse> {
    try {
      return await this.request<SystemLogsResponse>(`/logs/system?tail=${tail}`)
    } catch {
      return { logs: [] }
    }
  }

  // SSE URL for EventSource; tail=0 = no backfill (use REST when history needed)
  streamSystemLogsURL(tail: number): string {
    return `${API_BASE_URL}/logs/system/stream?tail=${tail}`
  }

  streamJobLogsURL(jobName: string, tail: number): string {
    return `${API_BASE_URL}/jobs/${encodeURIComponent(jobName)}/logs/stream?tail=${tail}`
  }

  downloadBackup(jobName: string, backupId: string): void {
    const url = `${API_BASE_URL}/jobs/${encodeURIComponent(jobName)}/backups/${encodeURIComponent(backupId)}/download`
    window.open(url, '_blank')
  }

  downloadBackupPart(jobName: string, partFilename: string): void {
    const url = `${API_BASE_URL}/jobs/${encodeURIComponent(jobName)}/backups/${encodeURIComponent(partFilename)}/download`
    window.open(url, '_blank')
  }

  async login(username: string, password: string): Promise<void> {
    await this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    })
  }

  async logout(): Promise<void> {
    await this.request('/auth/logout', { method: 'POST' })
  }

  async checkAuth(): Promise<{ authenticated: boolean; authEnabled: boolean }> {
    return this.request<{ authenticated: boolean; authEnabled: boolean }>('/auth/check')
  }
}

export const apiClient = new ApiClient()
