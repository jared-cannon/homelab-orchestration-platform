import { cn } from '../../lib/utils'

type StatusVariant = 'success' | 'warning' | 'error' | 'info' | 'default'

interface StatusBadgeProps {
  variant?: StatusVariant
  children: React.ReactNode
  className?: string
}

const variantStyles: Record<StatusVariant, string> = {
  success: 'bg-success/10 text-success border-success/20',
  warning: 'bg-yellow-500/10 text-yellow-700 border-yellow-500/20',
  error: 'bg-destructive/10 text-destructive border-destructive/20',
  info: 'bg-accent/10 text-accent border-accent/20',
  default: 'bg-muted text-muted-foreground border-border',
}

export function StatusBadge({
  variant = 'default',
  children,
  className,
}: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-medium border transition-colors',
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}
