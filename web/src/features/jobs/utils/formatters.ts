import i18n from '@/i18n'

const locale = () => i18n.t('format.dateLocale')

export const formatDate = (dateString: string) =>
  new Date(dateString).toLocaleString(locale(), {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: i18n.t('format.hour12') === 'true',
  })

const UPTIME_UNITS = [
  { unit: 'day' as const, sec: 86400 },
  { unit: 'hour' as const, sec: 3600 },
  { unit: 'minute' as const, sec: 60 },
  { unit: 'second' as const, sec: 1 },
]

export const formatUptime = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds < 0) return '-'
  const fmt = (value: number, unit: 'day' | 'hour' | 'minute' | 'second') =>
    new Intl.NumberFormat(locale(), { style: 'unit', unit, unitDisplay: 'narrow' }).format(value)

  let remaining = Math.floor(seconds)
  const parts = UPTIME_UNITS.flatMap(({ unit, sec }) => {
    const value = Math.floor(remaining / sec)
    remaining %= sec
    return value > 0 ? [fmt(value, unit)] : []
  })
  return parts.length > 0 ? parts.join(' ') : fmt(0, 'second')
}

const BYTE_UNITS = ['byte', 'kilobyte', 'megabyte', 'gigabyte', 'terabyte', 'petabyte'] as const

export const formatBytes = (bytes: number) => {
  if (!Number.isFinite(bytes) || bytes < 0) return '-'
  let value = bytes
  let i = 0
  while (value >= 1000 && i < BYTE_UNITS.length - 1) {
    value /= 1000
    i++
  }
  return new Intl.NumberFormat(locale(), {
    style: 'unit',
    unit: BYTE_UNITS[i],
    unitDisplay: 'short',
    maximumFractionDigits: i === 0 ? 0 : value < 10 ? 2 : value < 100 ? 1 : 0,
  }).format(value)
}

export const formatRelativeTime = (dateString: string, t: (key: string) => string) => {
  const date = new Date(dateString)
  if (Number.isNaN(date.getTime())) return '-'
  const diffSec = (date.getTime() - Date.now()) / 1000
  const absSec = Math.abs(diffSec)

  if (absSec < 60) return t('common.now')

  const rtf = new Intl.RelativeTimeFormat(locale(), { style: 'short', numeric: 'auto' })
  const sign = Math.sign(diffSec)
  const [value, unit] =
    absSec < 3600
      ? [absSec / 60, 'minute' as const]
      : absSec < 86400
        ? [absSec / 3600, 'hour' as const]
        : [absSec / 86400, 'day' as const]

  return rtf.format(sign * Math.floor(value), unit)
}
