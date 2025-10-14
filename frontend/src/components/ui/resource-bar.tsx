import { cn } from '../../lib/utils'

interface ResourceBarProps {
  label: string
  used: number
  total: number
  unit: string
  color?: 'cpu' | 'ram' | 'storage' | 'primary' | 'secondary'
  showPercentage?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function ResourceBar({
  label,
  used,
  total,
  unit,
  color = 'primary',
  showPercentage = true,
  size = 'md',
  className
}: ResourceBarProps) {
  const percentage = total > 0 ? (used / total) * 100 : 0

  // Format numbers based on unit
  const formatValue = (value: number): string => {
    // For GB, show 1 decimal place for values < 10, none for >= 10
    if (unit === 'GB') {
      if (value < 10) {
        return value.toFixed(1)
      }
      return Math.round(value).toLocaleString()
    }
    // For other units, use toLocaleString for comma separation
    return value.toLocaleString()
  }

  // Determine warning level based on usage
  const getStatusColor = () => {
    if (percentage >= 90) return 'bg-destructive'
    if (percentage >= 80) return 'bg-warning'

    switch (color) {
      case 'cpu':
        return 'bg-[hsl(var(--cpu-color))]'
      case 'ram':
        return 'bg-[hsl(var(--ram-color))]'
      case 'storage':
        return 'bg-[hsl(var(--storage-color))]'
      case 'secondary':
        return 'bg-secondary'
      default:
        return 'bg-primary'
    }
  }

  const sizeClasses = {
    sm: 'h-1.5',
    md: 'h-2',
    lg: 'h-3'
  }

  const textSizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base'
  }

  return (
    <div className={cn('space-y-1', className)}>
      <div className="flex items-center justify-between">
        <span className={cn('font-medium text-foreground', textSizeClasses[size])}>
          {label}
        </span>
        <span className={cn('text-muted-foreground', textSizeClasses[size])}>
          {formatValue(used)}{unit} / {formatValue(total)}{unit}
          {showPercentage && (
            <span className="ml-1.5 text-foreground font-medium">
              ({percentage.toFixed(0)}%)
            </span>
          )}
        </span>
      </div>
      <div className={cn('w-full bg-muted rounded-full overflow-hidden', sizeClasses[size])}>
        <div
          className={cn(
            'h-full transition-all duration-500 ease-out rounded-full',
            getStatusColor()
          )}
          style={{ width: `${Math.min(percentage, 100)}%` }}
        />
      </div>
    </div>
  )
}
