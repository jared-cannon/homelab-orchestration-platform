import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Server, Router, HardDrive, Network, Activity, Clock, MoreVertical, Edit, Key, TestTube, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import type { Device, DeviceType, DeviceStatus } from '../api/generated-types'
import { DeviceStatusOnline, DeviceStatusOffline, DeviceStatusError, DeviceStatusUnknown } from '../api/generated-types'
import { Card } from './ui/card'
import { StatusBadge } from './ui/status-badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from './ui/dropdown-menu'
import { EditDeviceDialog } from './EditDeviceDialog'
import { UpdateCredentialsDialog } from './UpdateCredentialsDialog'
import { DeleteDeviceDialog } from './DeleteDeviceDialog'
import { DeviceResourceMetrics } from './DeviceResourceMetrics'
import { useTestConnection } from '../api/hooks'

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
      return 'Checking...'
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
  const navigate = useNavigate()
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [credentialsDialogOpen, setCredentialsDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const testConnection = useTestConnection()

  const handleClick = () => {
    navigate(`/devices/${device.id}`)
  }

  const handleTestConnection = async (e: React.MouseEvent) => {
    e.stopPropagation()

    try {
      const result = await testConnection.mutateAsync(device.id)

      if (result.success || result.ssh_connection) {
        const dockerInfo = result.docker_installed
          ? `✓ Docker ${result.docker_version} ${result.docker_running ? '(running)' : '(not running)'}`
          : '⚠️ Docker not installed'

        toast.success('Connection successful!', {
          description: `Connected to ${device.ip_address}. ${dockerInfo}`
        })
      }
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Connection failed'

      toast.error('Connection test failed', {
        description: message
      })
    }
  }

  return (
    <>
      <Card className="group cursor-pointer hover:border-primary/50 transition-all" onClick={handleClick}>
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

            <div className="flex items-center gap-2">
              <StatusBadge variant={getStatusVariant(device.status)}>
                <Activity className={`w-3 h-3 ${device.status === DeviceStatusUnknown ? 'animate-pulse' : ''}`} />
                {getStatusText(device.status)}
              </StatusBadge>

              <DropdownMenu>
                <DropdownMenuTrigger
                  onClick={(e) => e.stopPropagation()}
                  className="rounded-lg p-2 hover:bg-accent transition-colors"
                >
                  <MoreVertical className="w-4 h-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      handleTestConnection(e)
                    }}
                    disabled={testConnection.isPending}
                  >
                    <TestTube className="w-4 h-4 mr-2" />
                    {testConnection.isPending ? 'Testing...' : 'Test Connection'}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      setEditDialogOpen(true)
                    }}
                  >
                    <Edit className="w-4 h-4 mr-2" />
                    Edit Device
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      setCredentialsDialogOpen(true)
                    }}
                  >
                    <Key className="w-4 h-4 mr-2" />
                    Update Credentials
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={(e) => {
                      e.stopPropagation()
                      setDeleteDialogOpen(true)
                    }}
                    className="text-destructive focus:text-destructive"
                  >
                    <Trash2 className="w-4 h-4 mr-2" />
                    Delete Device
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
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

        {/* Resource Metrics */}
        {device.status === DeviceStatusOnline && (
          <div className="px-6 pt-4 pb-4 mt-4 border-t border-border/50">
            <DeviceResourceMetrics
              cpuUsagePercent={device.cpu_usage_percent}
              cpuCores={device.cpu_cores}
              totalRamMB={device.total_ram_mb}
              usedRamMB={device.used_ram_mb}
              totalStorageGB={device.total_storage_gb}
              usedStorageGB={device.used_storage_gb}
              resourcesUpdatedAt={device.resources_updated_at}
            />
          </div>
        )}

        {/* Loading state for unknown status */}
        {device.status === DeviceStatusUnknown && (
          <div className="px-6 pt-4 pb-4 mt-4 border-t border-border/50">
            <DeviceResourceMetrics
              isChecking={true}
            />
          </div>
        )}
      </Card>

      {/* Dialogs */}
      <EditDeviceDialog
        device={device}
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
      />
      <UpdateCredentialsDialog
        device={device}
        open={credentialsDialogOpen}
        onOpenChange={setCredentialsDialogOpen}
      />
      <DeleteDeviceDialog
        device={device}
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
      />
    </>
  )
}
