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
    containerClass: 'bg-amber-500/5 dark:bg-amber-400/5 border-amber-200/40 dark:border-amber-400/20',
    iconClass: 'text-amber-600 dark:text-amber-500',
    titleClass: 'text-foreground',
    contentClass: 'text-muted-foreground',
  },
  info: {
    icon: Info,
    containerClass: 'bg-primary/5 dark:bg-primary/5 border-primary/20 dark:border-primary/20',
    iconClass: 'text-primary dark:text-primary',
    titleClass: 'text-foreground',
    contentClass: 'text-muted-foreground',
  },
  success: {
    icon: CheckCircle2,
    containerClass: 'bg-emerald-500/5 dark:bg-emerald-400/5 border-emerald-200/40 dark:border-emerald-400/20',
    iconClass: 'text-emerald-600 dark:text-emerald-500',
    titleClass: 'text-foreground',
    contentClass: 'text-muted-foreground',
  },
  tip: {
    icon: Info,
    containerClass: 'bg-violet-500/5 dark:bg-violet-400/5 border-violet-200/40 dark:border-violet-400/20',
    iconClass: 'text-violet-600 dark:text-violet-500',
    titleClass: 'text-foreground',
    contentClass: 'text-muted-foreground',
  },
}

export function InfoBox({ variant, title, children, className }: InfoBoxProps) {
  const config = variantConfig[variant]
  const Icon = config.icon

  return (
    <div className={cn('border rounded-lg p-4 flex gap-3', config.containerClass, className)}>
      <Icon className={cn('w-5 h-5 flex-shrink-0 mt-0.5', config.iconClass)} />
      <div className="flex-1 space-y-2">
        {title && <p className={cn('text-sm font-semibold', config.titleClass)}>{title}</p>}
        <div className={cn('text-sm leading-relaxed', config.contentClass)}>{children}</div>
      </div>
    </div>
  )
}
