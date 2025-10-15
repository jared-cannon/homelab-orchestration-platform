import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { BookOpen } from 'lucide-react'
import { useDevices, useAggregateResources } from '../api/hooks'
import { AddDeviceDialog } from '../components/AddDeviceDialog'
import { DeviceDiscoveryWizard } from '../components/DeviceDiscoveryWizard'
import { FirstRunWizard } from '../components/FirstRunWizard'
import { DeviceHealthCard } from '../components/DeviceHealthCard'
import { AggregateResourceCard } from '../components/AggregateResourceCard'
import { ServerSetupGuide } from '../components/ServerSetupGuide'
import { Button } from '../components/ui/button'

export function DevicesPage() {
  const { data: devices, isLoading, error } = useDevices()
  const { data: aggregateResources } = useAggregateResources()
  const [showSetupGuide, setShowSetupGuide] = useState(false)

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
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowSetupGuide(true)}
                className="text-muted-foreground hover:text-foreground"
              >
                <BookOpen className="mr-2 h-4 w-4" />
                Setup Guide
              </Button>
              <DeviceDiscoveryWizard />
              <AddDeviceDialog />
            </div>
          </div>

          {/* Homelab Overview */}
          <div className="mt-6 mb-6">
            <div className="bg-card border border-border rounded-lg p-5 shadow-sm mb-4">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-lg font-semibold">Your Homelab</h2>
                  <p className="text-sm text-muted-foreground mt-1">
                    {aggregateResources?.online_devices || 0} of {devices.length} devices online
                  </p>
                </div>
                <div className="text-right">
                  <div className="text-2xl font-bold text-primary">
                    {aggregateResources?.total_cpu_cores || 0}
                  </div>
                  <div className="text-xs text-muted-foreground">Total CPU Cores</div>
                </div>
              </div>
            </div>

            {/* Aggregate Resource Cards */}
            {aggregateResources && aggregateResources.total_devices > 0 && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <AggregateResourceCard
                  type="cpu"
                  used={aggregateResources.avg_cpu_usage_percent}
                  total={100}
                  unit="%"
                  percentage={aggregateResources.avg_cpu_usage_percent}
                  cores={aggregateResources.total_cpu_cores}
                  deviceCount={aggregateResources.online_devices}
                />
                <AggregateResourceCard
                  type="ram"
                  used={aggregateResources.used_ram_mb / 1024}
                  total={aggregateResources.total_ram_mb / 1024}
                  unit="GB"
                  percentage={aggregateResources.ram_usage_percent}
                />
                <AggregateResourceCard
                  type="storage"
                  used={aggregateResources.used_storage_gb}
                  total={aggregateResources.total_storage_gb}
                  unit="GB"
                  percentage={aggregateResources.storage_usage_percent}
                />
              </div>
            )}
          </div>
        </div>

        {/* Device Grid */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {devices.map((device) => (
            <DeviceHealthCard key={device.id} device={device} />
          ))}
        </div>
      </div>

      <ServerSetupGuide open={showSetupGuide} onOpenChange={setShowSetupGuide} />
    </div>
  )
}
