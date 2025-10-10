import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Activity, Server, Clock, RefreshCw, Edit, Trash2, Key, TestTube, AlertTriangle, WifiOff, ShieldAlert, HelpCircle } from 'lucide-react'
import { toast } from 'sonner'
import { Card } from '../components/ui/card'
import { StatusBadge } from '../components/ui/status-badge'
import { DeviceStatusOnline, DeviceStatusOffline, DeviceStatusError, DeviceStatusUnknown } from '../api/generated-types'
import { EditDeviceDialog } from '../components/EditDeviceDialog'
import { UpdateCredentialsDialog } from '../components/UpdateCredentialsDialog'
import { DeleteDeviceDialog } from '../components/DeleteDeviceDialog'
import { DeviceManagement } from '../components/DeviceManagement'
import { useDevice, useTestConnection } from '../api/hooks'

function getStatusVariant(status: string): 'success' | 'warning' | 'error' | 'default' {
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

function getStatusText(status: string): string {
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

export function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [credentialsDialogOpen, setCredentialsDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)

  // Use the centralized API client hook with auto-refresh every 10 seconds
  const { data: device, isLoading: loading, error, refetch, isRefetching } = useDevice(id || '', {
    refetchInterval: 10000, // Auto-refresh every 10 seconds
  })
  const testConnection = useTestConnection()

  const refreshDevice = async () => {
    await refetch()
  }

  const handleTestConnection = async () => {
    if (!id) return

    try {
      const result = await testConnection.mutateAsync(id)

      if (result.success || result.ssh_connection) {
        const dockerInfo = result.docker_installed
          ? `✓ Docker ${result.docker_version} ${result.docker_running ? '(running)' : '(not running)'}`
          : '⚠️ Docker not installed'

        toast.success('Connection successful!', {
          description: `Connected to ${device?.ip_address}. ${dockerInfo}`
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

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading device...</div>
      </div>
    )
  }

  if (error || !device) {
    // Determine error type and suggestions
    const errorMessage = error?.message || ''
    const isNotFound = errorMessage.includes('not found') || errorMessage.includes('404')
    const isAuthError = errorMessage.includes('auth') || errorMessage.includes('401') || errorMessage.includes('403')
    const isNetworkError = errorMessage.includes('fetch') || errorMessage.includes('network') || errorMessage.includes('Failed to fetch')

    let errorIcon = <AlertTriangle className="w-12 h-12 text-destructive" />
    let errorTitle = 'Unable to Load Device'
    let errorDescription = errorMessage || 'Device not found'
    let suggestions: { icon: React.ReactNode; text: string; action?: () => void }[] = []

    if (isNotFound) {
      errorIcon = <HelpCircle className="w-12 h-12 text-muted-foreground" />
      errorTitle = 'Device Not Found'
      errorDescription = 'This device may have been deleted or the ID is incorrect.'
      suggestions = [
        {
          icon: <ArrowLeft className="w-4 h-4" />,
          text: 'Return to dashboard to view all devices',
          action: () => navigate('/')
        }
      ]
    } else if (isAuthError) {
      errorIcon = <ShieldAlert className="w-12 h-12 text-destructive" />
      errorTitle = 'Authentication Error'
      errorDescription = 'Your session may have expired or you don\'t have permission to access this device.'
      suggestions = [
        {
          icon: <Key className="w-4 h-4" />,
          text: 'Try logging in again',
        },
        {
          icon: <ArrowLeft className="w-4 h-4" />,
          text: 'Return to dashboard',
          action: () => navigate('/')
        }
      ]
    } else if (isNetworkError) {
      errorIcon = <WifiOff className="w-12 h-12 text-destructive" />
      errorTitle = 'Connection Error'
      errorDescription = 'Unable to connect to the server. Please check your network connection.'
      suggestions = [
        {
          icon: <RefreshCw className="w-4 h-4" />,
          text: 'Retry loading the device',
          action: () => refetch()
        },
        {
          icon: <ArrowLeft className="w-4 h-4" />,
          text: 'Return to dashboard',
          action: () => navigate('/')
        }
      ]
    } else {
      suggestions = [
        {
          icon: <RefreshCw className="w-4 h-4" />,
          text: 'Try reloading the page',
          action: () => refetch()
        },
        {
          icon: <ArrowLeft className="w-4 h-4" />,
          text: 'Return to dashboard',
          action: () => navigate('/')
        }
      ]
    }

    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted p-8">
        <div className="max-w-4xl mx-auto">
          <button
            onClick={() => navigate('/')}
            className="flex items-center gap-2 text-muted-foreground hover:text-foreground mb-6 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Dashboard
          </button>

          <Card className="border-destructive/50">
            <div className="p-8">
              <div className="flex flex-col items-center text-center space-y-6">
                <div className="rounded-full bg-destructive/10 p-4">
                  {errorIcon}
                </div>

                <div className="space-y-2">
                  <h2 className="text-2xl font-bold">{errorTitle}</h2>
                  <p className="text-muted-foreground max-w-md">
                    {errorDescription}
                  </p>
                </div>

                {suggestions.length > 0 && (
                  <div className="w-full max-w-md space-y-3 pt-4">
                    <p className="text-sm font-medium text-muted-foreground">Suggested actions:</p>
                    <div className="space-y-2">
                      {suggestions.map((suggestion, index) => (
                        <button
                          key={index}
                          onClick={suggestion.action}
                          className="w-full flex items-center gap-3 px-4 py-3 bg-card border border-border rounded-lg hover:bg-accent transition-colors text-left"
                        >
                          {suggestion.icon}
                          <span>{suggestion.text}</span>
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          </Card>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <button
            onClick={() => navigate('/')}
            className="flex items-center gap-2 text-muted-foreground hover:text-foreground mb-6 transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            Back to Dashboard
          </button>

          <div className="flex items-start justify-between">
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                {device.name}
              </h1>
              <p className="mt-2 text-muted-foreground font-mono">{device.ip_address}</p>
            </div>

            <div className="flex items-center gap-3">
              <StatusBadge variant={getStatusVariant(device.status)}>
                <Activity className="w-3 h-3" />
                {getStatusText(device.status)}
              </StatusBadge>
              <button
                onClick={refreshDevice}
                disabled={isRefetching}
                className="p-2 rounded-lg bg-card border border-border hover:bg-accent transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${isRefetching ? 'animate-spin' : ''}`} />
              </button>
            </div>
          </div>
        </div>

        {/* Info Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
          {/* Device Info Card */}
          <Card>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Server className="w-5 h-5 text-primary" />
                </div>
                <h2 className="text-lg font-semibold">Device Information</h2>
              </div>

              <div className="space-y-3">
                <div>
                  <p className="text-sm text-muted-foreground">Type</p>
                  <p className="font-medium capitalize">{device.type}</p>
                </div>
                {device.mac_address && (
                  <div>
                    <p className="text-sm text-muted-foreground">MAC Address</p>
                    <p className="font-mono text-sm">{device.mac_address}</p>
                  </div>
                )}
                <div>
                  <p className="text-sm text-muted-foreground">Status</p>
                  <p className="font-medium capitalize">{getStatusText(device.status)}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Last Seen</p>
                  <p className="font-medium">{formatLastSeen(device.last_seen)}</p>
                </div>
              </div>
            </div>
          </Card>

          {/* Quick Actions Card */}
          <Card>
            <div className="p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="rounded-lg bg-primary/10 p-2">
                  <Clock className="w-5 h-5 text-primary" />
                </div>
                <h2 className="text-lg font-semibold">Quick Actions</h2>
              </div>

              <div className="space-y-2">
                <button
                  className="w-full px-4 py-3 bg-card border border-border rounded-lg hover:bg-accent transition-colors text-left disabled:opacity-50 disabled:cursor-not-allowed"
                  onClick={handleTestConnection}
                  disabled={testConnection.isPending}
                >
                  <div className="flex items-center gap-3">
                    <TestTube className="w-4 h-4" />
                    <span>{testConnection.isPending ? 'Testing...' : 'Test Connection'}</span>
                  </div>
                </button>
                <button
                  className="w-full px-4 py-3 bg-card border border-border rounded-lg hover:bg-accent transition-colors text-left"
                  onClick={() => setEditDialogOpen(true)}
                >
                  <div className="flex items-center gap-3">
                    <Edit className="w-4 h-4" />
                    <span>Edit Device Info</span>
                  </div>
                </button>
                <button
                  className="w-full px-4 py-3 bg-card border border-border rounded-lg hover:bg-accent transition-colors text-left"
                  onClick={() => setCredentialsDialogOpen(true)}
                >
                  <div className="flex items-center gap-3">
                    <Key className="w-4 h-4" />
                    <span>Update Credentials</span>
                  </div>
                </button>
                <button
                  className="w-full px-4 py-3 bg-card border border-destructive/50 rounded-lg hover:bg-destructive/10 transition-colors text-left text-destructive"
                  onClick={() => setDeleteDialogOpen(true)}
                >
                  <div className="flex items-center gap-3">
                    <Trash2 className="w-4 h-4" />
                    <span>Delete Device</span>
                  </div>
                </button>
              </div>
            </div>
          </Card>
        </div>

        {/* Device Management (Software, NFS, Volumes) */}
        <DeviceManagement deviceId={id!} />
      </div>

      {/* Dialogs */}
      {device && (
        <>
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
      )}
    </div>
  )
}
