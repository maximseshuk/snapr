import type { ReactNode } from 'react'

import { IconAlertTriangle } from '@tabler/icons-react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'

interface StopJobDialogProps {
  trigger: ReactNode
  onConfirm: () => void
  disabled?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

export const StopJobDialog = ({ trigger, onConfirm, disabled, open, onOpenChange }: StopJobDialogProps) => {
  const { t } = useTranslation()

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogTrigger asChild disabled={disabled}>
        {trigger}
      </AlertDialogTrigger>
      <AlertDialogContent size="sm">
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-destructive/10 text-destructive dark:bg-destructive/20 dark:text-destructive">
            <IconAlertTriangle className="size-5" />
          </AlertDialogMedia>
          <AlertDialogTitle>{t('jobs.stopConfirm.title')}</AlertDialogTitle>
          <AlertDialogDescription>{t('jobs.stopConfirm.description')}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline">{t('common.cancel')}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={onConfirm}>
            {t('jobs.stopConfirm.action')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
