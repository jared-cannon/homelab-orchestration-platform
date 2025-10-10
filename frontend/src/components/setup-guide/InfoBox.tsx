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
    containerClass: 'bg-amber-500/10 border-amber-500/20',
    iconClass: 'text-amber-500',
    titleClass: 'text-amber-900 dark:text-amber-100',
    contentClass: 'text-amber-800 dark:text-amber-200',
  },
  info: {
    icon: Info,
    containerClass: 'bg-blue-500/10 border-blue-500/20',
    iconClass: 'text-blue-500',
    titleClass: 'text-blue-900 dark:text-blue-100',
    contentClass: 'text-blue-800 dark:text-blue-200',
  },
  success: {
    icon: CheckCircle2,
    containerClass: 'bg-success/10 border-success/20',
    iconClass: 'text-success',
    titleClass: 'text-foreground',
    contentClass: 'text-success',
  },
  tip: {
    icon: Info,
    containerClass: 'bg-primary/5 border-primary/20',
    iconClass: 'text-primary',
    titleClass: 'text-foreground',
    contentClass: 'text-muted-foreground',
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
