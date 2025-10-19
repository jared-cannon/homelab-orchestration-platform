import { useState } from 'react'
import { toast } from 'sonner'
import { Wifi, Server, CheckCircle2, Loader2 } from 'lucide-react'
import { useStartScan, useScanProgress, useCreateDevice } from '../api/hooks'
import type { DeviceCredentials } from '../api/types'
import { Button } from './ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog'
import { CredentialsForm } from './CredentialsForm'
import { Badge } from './ui/badge'

type WizardStep = 'scan' | 'credentials' | 'complete'

export function DeviceDiscoveryWizard() {
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState<WizardStep>('scan')
  const [scanId, setScanId] = useState<string | null>(null)
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set())
  const [showAlreadyAdded, setShowAlreadyAdded] = useState(true)
  const [credentials, setCredentials] = useState<DeviceCredentials>({
    type: 'auto',
    username: '',
    password: '',
    ssh_key: '',
    ssh_key_passwd: '',
  })

  const startScan = useStartScan()
  const { data: scanProgress } = useScanProgress(scanId || '')
  const createDevice = useCreateDevice()

  const handleStartScan = async () => {
    try {
      const result = await startScan.mutateAsync({})
      setScanId(result.scan_id)
      toast.success('Network scan started', {
        description: `Scanning ${result.cidr}`,
      })
    } catch (error) {
      const err = error as Error
      toast.error('Failed to start scan', {
        description: err.message,
      })
    }
  }

  const handleDeviceSelect = (ipAddress: string) => {
    const newSelected = new Set(selectedDevices)
    if (newSelected.has(ipAddress)) {
      newSelected.delete(ipAddress)
    } else {
      newSelected.add(ipAddress)
    }
    setSelectedDevices(newSelected)
  }

  const handleContinue = () => {
    if (selectedDevices.size === 0) {
      toast.error('No devices selected', {
        description: 'Please select at least one device to add',
      })
      return
    }
    setStep('credentials')
  }

  const handleAddDevices = async () => {
    if (!scanProgress) return

    // Filter out already-added devices
    const devicesToAdd = scanProgress.devices.filter(
      (device) =>
        selectedDevices.has(device.local_ip_address) &&
        !device.already_added &&
        device.status !== 'already_added'
    )

    let successCount = 0
    let errorCount = 0

    for (const device of devicesToAdd) {
      try {
        await createDevice.mutateAsync({
          name: device.hostname || `Device ${device.local_ip_address}`,
          type: device.type,
          ip_address: device.local_ip_address,
          mac_address: device.mac_address,
          credentials,
        })
        successCount++
      } catch (error) {
        errorCount++
      }
    }

    if (successCount > 0) {
      toast.success(`Added ${successCount} device(s)`)
    }
    if (errorCount > 0) {
      toast.error(`Failed to add ${errorCount} device(s)`)
    }

    setStep('complete')
  }

  const handleClose = () => {
    setOpen(false)
    // Reset state after dialog closes
    setTimeout(() => {
      setStep('scan')
      setScanId(null)
      setSelectedDevices(new Set())
      setShowAlreadyAdded(true)
      setCredentials({
        type: 'auto',
        username: '',
        password: '',
        ssh_key: '',
        ssh_key_passwd: '',
      })
    }, 300)
  }

  const renderScanStep = () => (
    <>
      <DialogHeader>
        <DialogTitle>Discover Devices</DialogTitle>
        <DialogDescription>
          Automatically discover SSH-enabled devices on your local network
        </DialogDescription>
      </DialogHeader>

      <div className="py-6">
        {!scanId ? (
          <div className="text-center space-y-4">
            <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
              <Wifi className="w-8 h-8 text-primary" />
            </div>
            <p className="text-muted-foreground">
              Click below to scan your network for devices
            </p>
            <Button
              onClick={handleStartScan}
              disabled={startScan.isPending}
              size="lg"
            >
              {startScan.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting scan...
                </>
              ) : (
                'Start Network Scan'
              )}
            </Button>
          </div>
        ) : scanProgress ? (
          <div className="space-y-4">
            {/* Progress bar */}
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <div className="flex items-center gap-2">
                  <span className="font-medium">
                    {scanProgress.phase === 'ping' && 'Discovering hosts...'}
                    {scanProgress.phase === 'ssh_scan' && 'Scanning for SSH...'}
                    {scanProgress.phase === 'credential_test' && 'Testing credentials...'}
                    {scanProgress.phase === 'completed' && 'Scan complete'}
                    {!scanProgress.phase && 'Scanning network...'}
                  </span>
                  {scanProgress.scan_rate && scanProgress.scan_rate > 0 && (
                    <span className="text-xs text-muted-foreground">
                      ({scanProgress.scan_rate.toFixed(1)} IPs/s)
                    </span>
                  )}
                </div>
                <span className="text-muted-foreground">
                  {scanProgress.scanned_hosts}/{scanProgress.total_hosts}
                </span>
              </div>
              <div className="h-2 bg-muted rounded-full overflow-hidden">
                <div
                  className="h-full bg-primary transition-all duration-300"
                  style={{
                    width: `${
                      (scanProgress.scanned_hosts / scanProgress.total_hosts) *
                      100
                    }%`,
                  }}
                />
              </div>
              {scanProgress.current_ip && (
                <p className="text-xs text-muted-foreground">
                  Currently scanning: {scanProgress.current_ip}
                </p>
              )}
            </div>

            {/* Found devices count */}
            <div className="text-center py-2">
              <p className="text-2xl font-bold text-primary">
                {scanProgress.discovered_count}
              </p>
              <p className="text-sm text-muted-foreground">
                {scanProgress.discovered_count === 1
                  ? 'device found'
                  : 'devices found'}
              </p>
            </div>

            {/* Device list */}
            {scanProgress.devices.length > 0 && (
              <div className="space-y-3">
                {/* Toggle for already-added devices */}
                {scanProgress.devices.some((d) => d.already_added) && (
                  <div className="flex items-center justify-between pb-2 border-b">
                    <p className="text-sm text-muted-foreground">
                      {scanProgress.devices.filter((d) => d.already_added).length} already added
                    </p>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowAlreadyAdded(!showAlreadyAdded)}
                      className="text-xs h-7"
                    >
                      {showAlreadyAdded ? 'Hide' : 'Show'} Added
                    </Button>
                  </div>
                )}

                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {scanProgress.devices
                    .filter((device) => showAlreadyAdded || !device.already_added)
                    .map((device) => {
                      const isAlreadyAdded = device.already_added || device.status === 'already_added'
                      return (
                        <div
                          key={device.local_ip_address}
                          onClick={() => !isAlreadyAdded && handleDeviceSelect(device.local_ip_address)}
                          className={`flex items-center gap-3 p-3 border rounded-lg transition-colors ${
                            isAlreadyAdded
                              ? 'opacity-50 cursor-not-allowed'
                              : 'cursor-pointer hover:bg-accent'
                          } ${
                            selectedDevices.has(device.local_ip_address)
                              ? 'border-primary bg-primary/5'
                              : 'border-border'
                          }`}
                        >
                          <div
                            className={`w-5 h-5 rounded border flex-shrink-0 flex items-center justify-center ${
                              selectedDevices.has(device.local_ip_address) && !isAlreadyAdded
                                ? 'bg-primary border-primary'
                                : 'border-muted-foreground'
                            }`}
                          >
                            {selectedDevices.has(device.local_ip_address) && !isAlreadyAdded && (
                              <CheckCircle2 className="w-3 h-3 text-primary-foreground" />
                            )}
                          </div>
                          <Server className="w-4 h-4 text-muted-foreground flex-shrink-0" />
                          <div className="flex-1 min-w-0">
                            <p className="font-medium truncate">
                              {device.hostname || 'Unknown'}
                            </p>
                            <p className="text-sm text-muted-foreground">
                              {device.local_ip_address}
                            </p>
                          </div>
                          <div className="flex items-center gap-2 flex-shrink-0">
                            {isAlreadyAdded && (
                              <Badge variant="secondary" className="text-xs">
                                Added
                              </Badge>
                            )}
                            {device.ssh_available && !isAlreadyAdded && (
                              <span className="text-xs bg-success/10 text-success px-2 py-1 rounded">
                                SSH
                              </span>
                            )}
                            {device.docker_detected && !isAlreadyAdded && (
                              <span className="text-xs bg-accent/10 text-accent px-2 py-1 rounded">
                                Docker
                              </span>
                            )}
                          </div>
                        </div>
                      )
                    })}
                </div>
              </div>
            )}
          </div>
        ) : null}
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={handleClose}>
          Cancel
        </Button>
        {scanProgress && scanProgress.devices.length > 0 && (
          <Button onClick={handleContinue} disabled={selectedDevices.size === 0}>
            Continue ({selectedDevices.size} selected)
          </Button>
        )}
      </DialogFooter>
    </>
  )

  const renderCredentialsStep = () => (
    <>
      <DialogHeader>
        <DialogTitle>Device Credentials</DialogTitle>
        <DialogDescription>
          Enter the credentials to connect to the selected devices
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-4 py-4">
        <CredentialsForm
          credentials={credentials}
          onChange={setCredentials}
          idPrefix="discovery"
        />
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={() => setStep('scan')}>
          Back
        </Button>
        <Button onClick={handleAddDevices} disabled={createDevice.isPending}>
          {createDevice.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Adding devices...
            </>
          ) : (
            `Add ${selectedDevices.size} device(s)`
          )}
        </Button>
      </DialogFooter>
    </>
  )

  const renderCompleteStep = () => (
    <>
      <DialogHeader>
        <DialogTitle>Devices Added</DialogTitle>
        <DialogDescription>
          Your devices have been successfully added to the platform
        </DialogDescription>
      </DialogHeader>

      <div className="py-6 text-center space-y-4">
        <div className="w-16 h-16 bg-success/10 rounded-full flex items-center justify-center mx-auto">
          <CheckCircle2 className="w-8 h-8 text-success" />
        </div>
        <p className="text-lg font-medium">All set!</p>
        <p className="text-muted-foreground">
          You can now manage your devices from the dashboard
        </p>
      </div>

      <DialogFooter>
        <Button onClick={handleClose} className="w-full">
          Done
        </Button>
      </DialogFooter>
    </>
  )

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Wifi className="mr-2 h-4 w-4" />
          Discover Devices
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[600px]">
        {step === 'scan' && renderScanStep()}
        {step === 'credentials' && renderCredentialsStep()}
        {step === 'complete' && renderCompleteStep()}
      </DialogContent>
    </Dialog>
  )
}
