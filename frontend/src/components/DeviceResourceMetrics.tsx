import { AlertCircle } from 'lucide-react'
import { ResourceBar } from './ui/resource-bar'

interface DeviceResourceMetricsProps {
  cpuUsagePercent?: number
  cpuCores?: number
  totalRamMB?: number
  usedRamMB?: number
  totalStorageGB?: number
  usedStorageGB?: number
  resourcesUpdatedAt?: string
  className?: string
}

function formatTimeSince(timestamp?: string): string {
  if (!timestamp) return 'never'

  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`

  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`

  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}d ago`
}

function isStaleData(timestamp?: string): boolean {
  if (!timestamp) return true

  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  // Consider data stale if older than 2 minutes (should poll every 30s)
  return diffMins >= 2
}

export function DeviceResourceMetrics({
  cpuUsagePercent,
  cpuCores,
  totalRamMB,
  usedRamMB,
  totalStorageGB,
  usedStorageGB,
  resourcesUpdatedAt,
  className
}: DeviceResourceMetricsProps) {
  // Check if we have any resource data
  const hasData = cpuUsagePercent !== undefined || totalRamMB !== undefined || totalStorageGB !== undefined

  if (!hasData) {
    return (
      <div className={className}>
        <p className="text-sm text-muted-foreground">No resource data available</p>
      </div>
    )
  }

  const stale = isStaleData(resourcesUpdatedAt)
  const timeSince = formatTimeSince(resourcesUpdatedAt)

  return (
    <div className={className}>
      {/* Stale data warning */}
      {stale && (
        <div className="flex items-center gap-2 mb-3 text-xs text-yellow-600 dark:text-yellow-500">
          <AlertCircle className="w-3 h-3" />
          <span>Data may be outdated (last updated {timeSince})</span>
        </div>
      )}

      <div className="space-y-3">
        {/* CPU */}
        {cpuUsagePercent !== undefined && cpuCores !== undefined && (
          <ResourceBar
            label="CPU"
            used={cpuUsagePercent}
            total={100}
            unit="%"
            color="cpu"
            showPercentage={false}
            size="sm"
          />
        )}

        {/* RAM */}
        {totalRamMB !== undefined && usedRamMB !== undefined && (
          <ResourceBar
            label="RAM"
            used={usedRamMB / 1024}
            total={totalRamMB / 1024}
            unit="GB"
            color="ram"
            showPercentage={false}
            size="sm"
          />
        )}

        {/* Storage */}
        {totalStorageGB !== undefined && usedStorageGB !== undefined && (
          <ResourceBar
            label="Disk"
            used={usedStorageGB}
            total={totalStorageGB}
            unit="GB"
            color="storage"
            showPercentage={false}
            size="sm"
          />
        )}
      </div>

      {/* Update timestamp */}
      {!stale && resourcesUpdatedAt && (
        <p className="text-xs text-muted-foreground mt-2">
          Updated {timeSince}
        </p>
      )}
    </div>
  )
}
