import { cva } from 'class-variance-authority'

export const statusPillVariants = cva('inline-flex items-center gap-1.5 rounded-sm border px-2.5 py-0.5', {
  variants: {
    tone: {
      running: 'bg-status-running text-status-running-foreground border-status-running-border',
      idle: 'bg-status-idle text-status-idle-foreground border-status-idle-border',
      success: 'bg-status-success text-status-success-foreground border-status-success-border',
      failed: 'bg-status-failed text-status-failed-foreground border-status-failed-border',
      info: 'bg-status-info text-status-info-foreground border-status-info-border',
    },
  },
  defaultVariants: { tone: 'idle' },
})
