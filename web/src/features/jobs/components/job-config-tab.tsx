import { useTranslation } from 'react-i18next'

import type { JobDetail } from '@/types/api'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'

import { getNotifierIcon, getSourceIcon, getStorageIcon, translateFieldName } from '../utils/helpers'

type Translate = (key: string, options?: { count: number }) => string

interface JobConfigTabProps {
  job: JobDetail
}

const isPresentValue = (value: unknown) =>
  value !== undefined && value !== '' && value !== null && (!Array.isArray(value) || value.length > 0)

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

// Render a key/value record (e.g. pg_dump extraParams) as readable flags:
// an empty value is a bare flag (--clean), otherwise --key=value.
const formatFlag = (key: string, value: unknown) =>
  value === '' || value === null || value === undefined ? `--${key}` : `--${key}=${String(value)}`

const formatValue = (value: unknown) => {
  if (Array.isArray(value)) return value.join(', ')
  if (typeof value === 'object' && value !== null) return JSON.stringify(value)
  return String(value)
}

const FieldRow = ({ label, children }: { label: string; children: React.ReactNode }) => {
  return (
    <div className="grid grid-cols-3 gap-4 text-sm">
      <div className="text-muted-foreground font-medium">{label}</div>
      <div className="col-span-2 font-mono break-all">{children}</div>
    </div>
  )
}

const FieldRows = ({ data, excludeKeys, t }: { data: object; excludeKeys: string[]; t: Translate }) => {
  return (
    <div className="space-y-2">
      {Object.entries(data)
        .filter(([key, value]) => !excludeKeys.includes(key) && isPresentValue(value))
        .map(([key, value]) => (
          <FieldRow key={key} label={translateFieldName(key, t)}>
            {isPlainObject(value) ? (
              <div className="flex flex-wrap gap-1.5">
                {Object.entries(value).map(([flagKey, flagValue]) => (
                  <Badge key={flagKey} variant="secondary" className="font-mono font-normal">
                    {formatFlag(flagKey, flagValue)}
                  </Badge>
                ))}
              </div>
            ) : (
              formatValue(value)
            )}
          </FieldRow>
        ))}
    </div>
  )
}

export const JobConfigTab = ({ job }: JobConfigTabProps) => {
  const { t } = useTranslation()
  const sources = job.sources ?? []
  const storages = job.storages ?? []
  const notifiers = job.notifiers ?? []

  return (
    <Card className="!py-4">
      <CardContent>
        <div className="space-y-6">
          <section>
            <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.sources')}</h3>
            <div className="space-y-3">
              {sources.map((source, idx) => {
                const SourceIcon = getSourceIcon(source.type)
                return (
                  <div key={idx} className="ring-border rounded-lg p-4 ring-1">
                    <div className="mb-3 flex items-center gap-2">
                      <SourceIcon className="text-muted-foreground size-4" />
                      <Badge variant="outline">{source.type}</Badge>
                    </div>
                    <FieldRows data={source} excludeKeys={['type']} t={t} />
                  </div>
                )
              })}
            </div>
          </section>

          <section>
            <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.storages')}</h3>
            <div className="space-y-3">
              {storages.map((storage, idx) => {
                const StorageIcon = getStorageIcon(storage.type)
                const storageName = storage.name || (storage.type === 'local' ? storage.path : storage.bucket)
                const isDefault = job.defaultStorage === storageName || (!job.defaultStorage && idx === 0)
                return (
                  <div key={idx} className="ring-border rounded-lg p-4 ring-1">
                    <div className="mb-3 flex items-center gap-2">
                      <StorageIcon className="text-muted-foreground size-4" />
                      <Badge variant="outline">{storage.type}</Badge>
                      {isDefault && <Badge>{t('jobDetails.default')}</Badge>}
                      {storage.name && <span className="text-muted-foreground font-mono text-xs">{storage.name}</span>}
                    </div>
                    <FieldRows data={storage} excludeKeys={['type', 'name']} t={t} />
                  </div>
                )
              })}
            </div>
          </section>

          {notifiers.length > 0 && (
            <section>
              <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.notifiers')}</h3>
              <div className="space-y-3">
                {notifiers.map((notifier, idx) => {
                  const NotifierIcon = getNotifierIcon(notifier.type)
                  return (
                    <div key={idx} className="ring-border rounded-lg p-4 ring-1">
                      <div className="mb-3 flex items-center gap-2">
                        <NotifierIcon className="text-muted-foreground size-4" />
                        <Badge variant="outline">{notifier.type}</Badge>
                        {notifier.name && (
                          <span className="text-muted-foreground font-mono text-xs">{notifier.name}</span>
                        )}
                      </div>
                      <FieldRows data={notifier} excludeKeys={['type', 'name']} t={t} />
                    </div>
                  )
                })}
              </div>
            </section>
          )}

          {job.encryption && (
            <section>
              <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.encryption')}</h3>
              <div className="ring-border rounded-lg p-4 ring-1">
                <FieldRows data={job.encryption} excludeKeys={[]} t={t} />
              </div>
            </section>
          )}

          {job.split && (
            <section>
              <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.split')}</h3>
              <div className="ring-border rounded-lg p-4 ring-1">
                <FieldRow label={t('jobDetails.splitChunkSize')}>{job.split.chunkSize}</FieldRow>
              </div>
            </section>
          )}

          <section>
            <h3 className="mb-3 text-sm font-semibold">{t('jobDetails.settings')}</h3>
            <div className="ring-border space-y-2 rounded-lg p-4 ring-1">
              <FieldRow label={t('jobDetails.compression')}>{job.compression || t('jobDetails.none')}</FieldRow>
              {job.retention && (
                <FieldRow label={t('jobDetails.retention')}>
                  {t('jobDetails.retentionValue', { count: job.retention.last })}
                </FieldRow>
              )}
              {job.hasBeforeScript && (
                <FieldRow label={t('jobDetails.beforeScript')}>{t('jobDetails.configured')}</FieldRow>
              )}
              {job.hasAfterScript && (
                <FieldRow label={t('jobDetails.afterScript')}>{t('jobDetails.configured')}</FieldRow>
              )}
            </div>
          </section>
        </div>
      </CardContent>
    </Card>
  )
}
