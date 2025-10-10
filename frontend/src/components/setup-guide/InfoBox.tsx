import { AlertTriangle, Info, CheckCircle2, LucideIcon } from 'lucide-react'
import { cn } from '../../lib/utils'

type InfoBoxVariant = 'warning' | 'info' | 'success' | 'tip'

interface InfoBoxProps {
  variant: InfoBoxVariant
  title?: string
  children: React.ReactNode
  className?: string
}

const variantConfig: Record<
  InfoBoxVariant,
  {
    icon: LucideIcon
    containerClass: string
    iconClass: string
    titleClass: string
    contentClass: string
  }
> = {
  warning: {
    icon: AlertTriangle,
    containerClass: 'bg-amber-50 dark:bg-amber-950/50 border-amber-200 dark:border-amber-800',
    iconClass: 'text-amber-600 dark:text-amber-400',
    titleClass: 'text-amber-950 dark:text-amber-50',
    contentClass: 'text-amber-900 dark:text-amber-100',
  },
  info: {
    icon: Info,
    containerClass: 'bg-blue-50 dark:bg-blue-950/50 border-blue-200 dark:border-blue-800',
    iconClass: 'text-blue-600 dark:text-blue-400',
    titleClass: 'text-blue-950 dark:text-blue-50',
    contentClass: 'text-blue-900 dark:text-blue-100',
  },
  success: {
    icon: CheckCircle2,
    containerClass: 'bg-emerald-50 dark:bg-emerald-950/50 border-emerald-200 dark:border-emerald-800',
    iconClass: 'text-emerald-600 dark:text-emerald-400',
    titleClass: 'text-emerald-950 dark:text-emerald-50',
    contentClass: 'text-emerald-900 dark:text-emerald-100',
  },
  tip: {
    icon: Info,
    containerClass: 'bg-violet-50 dark:bg-violet-950/50 border-violet-200 dark:border-violet-800',
    iconClass: 'text-violet-600 dark:text-violet-400',
    titleClass: 'text-violet-950 dark:text-violet-50',
    contentClass: 'text-violet-900 dark:text-violet-100',
  },
}

export function InfoBox({ variant, title, children, className }: InfoBoxProps) {
  const config = variantConfig[variant]
  const Icon = config.icon

  return (
    <div className={cn('border rounded-lg p-3 flex gap-2', config.containerClass, className)}>
      <Icon className={cn('w-4 h-4 flex-shrink-0 mt-0.5', config.iconClass)} />
      <div className="flex-1 space-y-1">
        {title && <p className={cn('text-sm font-medium', config.titleClass)}>{title}</p>}
        <div className={cn('text-sm', config.contentClass)}>{children}</div>
      </div>
    </div>
  )
}
