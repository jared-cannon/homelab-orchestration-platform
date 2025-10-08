import { useEffect } from 'react'
import { toast } from 'sonner'
import { useDevices } from '../api/hooks'
import { AddDeviceDialog } from '../components/AddDeviceDialog'
import { DeviceDiscoveryWizard } from '../components/DeviceDiscoveryWizard'
import { FirstRunWizard } from '../components/FirstRunWizard'
import { DeviceHealthCard } from '../components/DeviceHealthCard'

export function DevicesPage() {
  const { data: devices, isLoading, error } = useDevices()

  // Show toast notification when device loading fails
  useEffect(() => {
    if (error) {
      toast.error('Could not load devices', {
        description: (error as Error).message || 'Please check your connection and try again'
      })
    }
  }, [error])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading devices...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-600">
          Error loading devices: {(error as Error).message}
        </div>
      </div>
    )
  }

  // Show first-run wizard if no devices
  if (!devices || devices.length === 0) {
    return <FirstRunWizard />
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
                Dashboard
              </h1>
              <p className="mt-2 text-muted-foreground">
                Monitor and manage your homelab devices
              </p>
            </div>
            <div className="flex gap-3">
              <DeviceDiscoveryWizard />
              <AddDeviceDialog />
            </div>
          </div>

          {/* Stats Summary */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-6">
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                  <span className="text-lg font-bold text-primary">{devices.length}</span>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Total Devices</p>
                  <p className="text-xs text-muted-foreground/70">Across your network</p>
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-success/10 flex items-center justify-center">
                  <span className="text-lg font-bold text-success">
                    {devices.filter(d => d.status === 'online').length}
                  </span>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Online</p>
                  <p className="text-xs text-muted-foreground/70">Active connections</p>
                </div>
              </div>
            </div>

            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-lg bg-muted flex items-center justify-center">
                  <span className="text-lg font-bold text-muted-foreground">
                    {devices.filter(d => d.status === 'offline').length}
                  </span>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Offline</p>
                  <p className="text-xs text-muted-foreground/70">Unreachable</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Device Grid */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {devices.map((device) => (
            <DeviceHealthCard key={device.id} device={device} />
          ))}
        </div>
      </div>
    </div>
  )
}
