import {
  IconBrandTelegram,
  IconCloud,
  IconCloudUpload,
  IconDatabase,
  IconFile,
  IconFolder,
  IconMail,
  IconServer,
  IconWebhook,
} from '@tabler/icons-react'
import 'cronstrue/locales/ru'
import cronstrue from 'cronstrue'

export const describeCron = (cronExpression: string, locale: string) => {
  try {
    return cronstrue.toString(cronExpression, { use24HourTimeFormat: true, locale })
  } catch {
    return cronExpression
  }
}

export const translateFieldName = (fieldName: string, t: (key: string) => string): string => {
  const words = fieldName.replace(/([A-Z])/g, ' $1').trim()
  const translationKey = `config.${fieldName}`
  const translated = t(translationKey)
  return translated !== translationKey ? translated : words.charAt(0).toUpperCase() + words.slice(1)
}

const STORAGE_ICONS: Record<string, typeof IconCloud> = {
  s3: IconCloud,
  local: IconFolder,
  bunny: IconCloudUpload,
  sftp: IconServer,
  webdav: IconCloudUpload,
}

export const getStorageIcon = (type: string) => STORAGE_ICONS[type.toLowerCase()] || IconServer

const SOURCE_ICONS: Record<string, typeof IconCloud> = {
  postgresql: IconDatabase,
  mysql: IconDatabase,
  mariadb: IconDatabase,
  mongodb: IconDatabase,
  redis: IconDatabase,
  sqlite: IconFile,
  local: IconFolder,
  bunny: IconCloudUpload,
  s3: IconCloud,
}

export const getSourceIcon = (type: string) => SOURCE_ICONS[type.toLowerCase()] || IconServer

const NOTIFIER_ICONS: Record<string, typeof IconCloud> = {
  webhook: IconWebhook,
  telegram: IconBrandTelegram,
  email: IconMail,
}

export const getNotifierIcon = (type: string) => NOTIFIER_ICONS[type.toLowerCase()] || IconServer
