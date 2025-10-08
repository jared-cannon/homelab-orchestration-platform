import { Server, Router, HardDrive, Network, Activity, Clock } from 'lucide-react'
import type { Device, DeviceType, DeviceStatus } from '../api/generated-types'
import { DeviceStatusOnline, DeviceStatusOffline, DeviceStatusError, DeviceStatusUnknown } from '../api/generated-types'
import { Card } from './ui/card'
import { StatusBadge } from './ui/status-badge'

interface DeviceHealthCardProps {
  device: Device
}

function getDeviceIcon(type: DeviceType) {
  switch (type) {
    case 'router':
      return <Router className="size-5" />
    case 'server':
      return <Server className="size-5" />
    case 'nas':
      return <HardDrive className="size-5" />
    case 'switch':
      return <Network className="size-5" />
    default:
      return <Server className="size-5" />
  }
}

function getStatusVariant(status: DeviceStatus): 'success' | 'warning' | 'error' | 'default' {
  switch (status) {
    case DeviceStatusOnline:
      return 'success'
    case DeviceStatusOffline:
      return 'default'
    case DeviceStatusError:
      return 'error'
    case DeviceStatusUnknown:
    default:
      return 'warning'
  }
}

function getStatusText(status: DeviceStatus): string {
  switch (status) {
    case DeviceStatusOnline:
      return 'Online'
    case DeviceStatusOffline:
      return 'Offline'
    case DeviceStatusError:
      return 'Error'
    case DeviceStatusUnknown:
    default:
      return 'Unknown'
  }
}

function formatLastSeen(lastSeen?: string): string {
  if (!lastSeen) return 'Never'

  const date = new Date(lastSeen)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`

  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`

  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}d ago`
}

export function DeviceHealthCard({ device }: DeviceHealthCardProps) {
  return (
    <Card className="group cursor-pointer hover:border-primary/50 transition-all">
      <div className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-gradient-to-br from-primary/10 to-primary/5 p-3 group-hover:from-primary/20 group-hover:to-primary/10 transition-colors">
              {getDeviceIcon(device.type)}
            </div>
            <div>
              <h3 className="font-semibold text-base">{device.name}</h3>
              <p className="text-sm text-muted-foreground font-mono">{device.ip_address}</p>
            </div>
          </div>

          <StatusBadge variant={getStatusVariant(device.status)}>
            <Activity className="w-3 h-3" />
            {getStatusText(device.status)}
          </StatusBadge>
        </div>

        <div className="flex items-center justify-between pt-4 border-t border-border/50">
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Server className="w-4 h-4" />
            <span className="capitalize">{device.type}</span>
          </div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="w-4 h-4" />
            <span>{formatLastSeen(device.last_seen)}</span>
          </div>
        </div>
      </div>
    </Card>
  )
}
