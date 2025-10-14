import { Cpu, Database, HardDrive } from 'lucide-react'
import { Card } from './ui/card'
import { cn } from '../lib/utils'

interface AggregateResourceCardProps {
  type: 'cpu' | 'ram' | 'storage'
  used: number
  total: number
  unit: string
  percentage: number
  cores?: number
  className?: string
}

const typeConfig = {
  cpu: {
    icon: Cpu,
    label: 'CPU Usage',
    color: 'hsl(var(--cpu-color))',
    bgColor: 'bg-[hsl(var(--cpu-color)/0.1)]'
  },
  ram: {
    icon: Database,
    label: 'Memory',
    color: 'hsl(var(--ram-color))',
    bgColor: 'bg-[hsl(var(--ram-color)/0.1)]'
  },
  storage: {
    icon: HardDrive,
    label: 'Storage',
    color: 'hsl(var(--storage-color))',
    bgColor: 'bg-[hsl(var(--storage-color)/0.1)]'
  }
}

export function AggregateResourceCard({
  type,
  used,
  total,
  unit,
  percentage,
  cores,
  className
}: AggregateResourceCardProps) {
  const config = typeConfig[type]
  const Icon = config.icon

  // Determine status color based on percentage
  const getStatusClass = () => {
    if (percentage >= 90) return 'text-destructive'
    if (percentage >= 80) return 'text-warning'
    return 'text-foreground'
  }

  const getBarColor = () => {
    if (percentage >= 90) return 'bg-destructive'
    if (percentage >= 80) return 'bg-warning'
    return `bg-[${config.color}]`
  }

  return (
    <Card className={cn('p-6', className)}>
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={cn('p-2.5 rounded-lg', config.bgColor)}
            style={{ backgroundColor: `${config.color}15` }}
          >
            <Icon className="w-5 h-5" style={{ color: config.color }} />
          </div>
          <div>
            <h3 className="text-sm font-medium text-muted-foreground">{config.label}</h3>
            {type === 'cpu' && cores && (
              <p className="text-xs text-muted-foreground mt-0.5">{cores} cores total</p>
            )}
          </div>
        </div>
        <div className={cn('text-2xl font-bold tabular-nums', getStatusClass())}>
          {percentage.toFixed(0)}%
        </div>
      </div>

      {/* Progress bar */}
      <div className="w-full h-3 bg-muted rounded-full overflow-hidden mb-3">
        <div
          className={cn(
            'h-full transition-all duration-500 ease-out',
            getBarColor()
          )}
          style={{
            width: `${Math.min(percentage, 100)}%`,
            backgroundColor: config.color
          }}
        />
      </div>

      {/* Usage details */}
      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">
          {used.toLocaleString()}{unit} used
        </span>
        <span className="text-muted-foreground">
          {total.toLocaleString()}{unit} total
        </span>
      </div>
    </Card>
  )
}
