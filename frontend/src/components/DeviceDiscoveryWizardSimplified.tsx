import { useState } from 'react'
import { toast } from 'sonner'
import {
  Wifi,
  Server,
  CheckCircle2,
  XCircle,
  Loader2,
  ChevronDown,
  ChevronUp,
  Plus,
} from 'lucide-react'
import { useStartScan, useScanProgress, useCreateDevice } from '../api/hooks'
import type { DiscoveredDevice, DeviceCredentials } from '../api/types'
import { Button } from './ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog'
import { Input } from './ui/input'
import { Label } from './ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select'
import { Badge } from './ui/badge'

type DeviceWithCredentials = {
  device: DiscoveredDevice
  credentials: DeviceCredentials | null
  expanded: boolean
}

export function DeviceDiscoveryWizard() {
  const [open, setOpen] = useState(false)
  const [scanId, setScanId] = useState<string | null>(null)
  const [devices, setDevices] = useState<Map<string, DeviceWithCredentials>>(
    new Map()
  )
  const [showAlreadyAdded, setShowAlreadyAdded] = useState(false)

  const startScan = useStartScan()
  const { data: scanProgress } = useScanProgress(scanId || '')
  const createDevice = useCreateDevice()

  // Update devices map when scan progress changes
  useState(() => {
    if (scanProgress?.devices) {
      const newDevices = new Map(devices)
      scanProgress.devices.forEach((device) => {
        if (!newDevices.has(device.ip_address)) {
          newDevices.set(device.ip_address, {
            device,
            credentials:
              device.credential_status === 'working' || device.status === 'ready'
                ? null
                : {
                    type: 'password',
                    username: '',
                    password: '',
                    ssh_key: '',
                    ssh_key_passwd: '',
                  },
            expanded: false,
          })
        } else {
          // Update existing device data
          const existing = newDevices.get(device.ip_address)!
          newDevices.set(device.ip_address, {
            ...existing,
            device,
          })
        }
      })
      setDevices(newDevices)
    }
  })

  const handleStartScan = async () => {
    try {
      const result = await startScan.mutateAsync({})
      setScanId(result.scan_id)
      setDevices(new Map())
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

  const handleAddDevice = async (deviceInfo: DeviceWithCredentials) => {
    const { device, credentials } = deviceInfo

    try {
      await createDevice.mutateAsync({
        name: device.hostname || `Device ${device.ip_address}`,
        type: device.type,
        ip_address: device.ip_address,
        mac_address: device.mac_address,
        credentials: credentials || {
          type: 'password',
          username: '',
          password: '',
          ssh_key: '',
          ssh_key_passwd: '',
        },
      })
      toast.success(`Added ${device.hostname || device.ip_address}`)

      // Mark as added
      const newDevices = new Map(devices)
      const deviceData = newDevices.get(device.ip_address)!
      newDevices.set(device.ip_address, {
        ...deviceData,
        device: { ...device, already_added: true, status: 'already_added' },
      })
      setDevices(newDevices)
    } catch (error) {
      const err = error as Error
      toast.error(`Failed to add device`, {
        description: err.message,
      })
    }
  }

  const handleAddAllReady = async () => {
    const readyDevices = Array.from(devices.values()).filter(
      (d) =>
        (d.device.status === 'ready' || d.device.credential_status === 'working') &&
        !d.device.already_added
    )

    let successCount = 0
    for (const deviceInfo of readyDevices) {
      try {
        await handleAddDevice(deviceInfo)
        successCount++
      } catch {
        // Error already handled in handleAddDevice
      }
    }

    if (successCount > 0) {
      toast.success(`Added ${successCount} device(s)`)
    }
  }

  const toggleDeviceExpanded = (ipAddress: string) => {
    const newDevices = new Map(devices)
    const deviceData = newDevices.get(ipAddress)!
    newDevices.set(ipAddress, {
      ...deviceData,
      expanded: !deviceData.expanded,
    })
    setDevices(newDevices)
  }

  const updateCredentials = (
    ipAddress: string,
    credentials: DeviceCredentials
  ) => {
    const newDevices = new Map(devices)
    const deviceData = newDevices.get(ipAddress)!
    newDevices.set(ipAddress, {
      ...deviceData,
      credentials,
    })
    setDevices(newDevices)
  }

  const handleClose = () => {
    setOpen(false)
    setTimeout(() => {
      setScanId(null)
      setDevices(new Map())
    }, 300)
  }

  const getDeviceStatusBadge = (device: DiscoveredDevice) => {
    if (device.already_added || device.status === 'already_added') {
      return (
        <Badge variant="secondary" className="bg-muted text-muted-foreground">
          Already Added
        </Badge>
      )
    }
    if (device.status === 'ready' || device.credential_status === 'working') {
      return (
        <Badge variant="default" className="bg-emerald-500/10 text-emerald-700 dark:text-emerald-400">
          <CheckCircle2 className="w-3 h-3 mr-1" />
          Ready
        </Badge>
      )
    }
    if (device.status === 'needs_credentials' || device.credential_status === 'failed') {
      return (
        <Badge variant="secondary" className="bg-amber-500/10 text-amber-700 dark:text-amber-400">
          <XCircle className="w-3 h-3 mr-1" />
          Needs Credentials
        </Badge>
      )
    }
    return (
      <Badge variant="secondary">
        <Loader2 className="w-3 h-3 mr-1 animate-spin" />
        Checking...
      </Badge>
    )
  }

  const renderCredentialForm = (
    deviceInfo: DeviceWithCredentials,
    credentials: DeviceCredentials
  ) => (
    <div className="space-y-3 px-4 pb-3 pt-2 bg-muted/30 rounded-b-lg border-t">
      <div className="grid gap-2">
        <Label className="text-xs">Authentication Method</Label>
        <Select
          value={credentials.type}
          onValueChange={(value: 'password' | 'ssh_key') =>
            updateCredentials(deviceInfo.device.ip_address, {
              ...credentials,
              type: value,
            })
          }
        >
          <SelectTrigger className="h-8 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="password">Password</SelectItem>
            <SelectItem value="ssh_key">SSH Key</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="grid gap-2">
        <Label htmlFor={`username-${deviceInfo.device.ip_address}`} className="text-xs">
          Username
        </Label>
        <Input
          id={`username-${deviceInfo.device.ip_address}`}
          className="h-8 text-xs"
          value={credentials.username}
          onChange={(e) =>
            updateCredentials(deviceInfo.device.ip_address, {
              ...credentials,
              username: e.target.value,
            })
          }
          placeholder="root"
        />
      </div>

      {credentials.type === 'password' ? (
        <div className="grid gap-2">
          <Label htmlFor={`password-${deviceInfo.device.ip_address}`} className="text-xs">
            Password
          </Label>
          <Input
            id={`password-${deviceInfo.device.ip_address}`}
            className="h-8 text-xs"
            type="password"
            value={credentials.password}
            onChange={(e) =>
              updateCredentials(deviceInfo.device.ip_address, {
                ...credentials,
                password: e.target.value,
              })
            }
          />
        </div>
      ) : (
        <>
          <div className="grid gap-2">
            <Label htmlFor={`ssh-key-${deviceInfo.device.ip_address}`} className="text-xs">
              SSH Private Key
            </Label>
            <textarea
              id={`ssh-key-${deviceInfo.device.ip_address}`}
              value={credentials.ssh_key}
              onChange={(e) =>
                updateCredentials(deviceInfo.device.ip_address, {
                  ...credentials,
                  ssh_key: e.target.value,
                })
              }
              className="min-h-[80px] w-full rounded-md border border-input bg-background px-2 py-1.5 text-xs ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              placeholder="Paste private key"
            />
          </div>
          <div className="grid gap-2">
            <Label
              htmlFor={`key-passphrase-${deviceInfo.device.ip_address}`}
              className="text-xs"
            >
              Key Passphrase (Optional)
            </Label>
            <Input
              id={`key-passphrase-${deviceInfo.device.ip_address}`}
              className="h-8 text-xs"
              type="password"
              value={credentials.ssh_key_passwd}
              onChange={(e) =>
                updateCredentials(deviceInfo.device.ip_address, {
                  ...credentials,
                  ssh_key_passwd: e.target.value,
                })
              }
            />
          </div>
        </>
      )}

      <Button
        size="sm"
        onClick={() => handleAddDevice(deviceInfo)}
        disabled={
          createDevice.isPending ||
          !credentials.username ||
          (credentials.type === 'password' && !credentials.password) ||
          (credentials.type === 'ssh_key' && !credentials.ssh_key)
        }
        className="w-full h-8 text-xs"
      >
        {createDevice.isPending ? (
          <>
            <Loader2 className="mr-1 h-3 w-3 animate-spin" />
            Adding...
          </>
        ) : (
          <>
            <Plus className="mr-1 h-3 w-3" />
            Add Device
          </>
        )}
      </Button>
    </div>
  )

  const renderDeviceCard = (deviceInfo: DeviceWithCredentials) => {
    const { device, credentials, expanded } = deviceInfo
    const isAlreadyAdded = device.already_added || device.status === 'already_added'
    const isReady = device.status === 'ready' || device.credential_status === 'working'
    const needsCredentials =
      device.status === 'needs_credentials' || device.credential_status === 'failed'

    // Filter out already added devices unless showAlreadyAdded is true
    if (isAlreadyAdded && !showAlreadyAdded) {
      return null
    }

    return (
      <div
        key={device.ip_address}
        className={`border rounded-lg overflow-hidden transition-all ${
          isAlreadyAdded ? 'opacity-50' : ''
        }`}
      >
        <div className="flex items-center gap-3 p-3">
          <Server className="w-5 h-5 text-muted-foreground flex-shrink-0" />
          <div className="flex-1 min-w-0">
            <p className="font-medium truncate text-sm">
              {device.hostname || 'Unknown Device'}
            </p>
            <p className="text-xs text-muted-foreground">{device.ip_address}</p>
            {device.mac_address && (
              <p className="text-xs text-muted-foreground/70 font-mono">
                {device.mac_address}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            {device.services_detected &&
              device.services_detected.map((service) => (
                <span
                  key={service}
                  className="text-xs bg-indigo-500/10 text-indigo-700 dark:text-indigo-400 px-2 py-0.5 rounded capitalize"
                >
                  {service}
                </span>
              ))}
            {device.docker_detected && (
              <span className="text-xs bg-blue-500/10 text-blue-700 dark:text-blue-400 px-2 py-0.5 rounded">
                Docker
              </span>
            )}
            {device.os && (
              <span className="text-xs bg-slate-500/10 text-slate-700 dark:text-slate-400 px-2 py-0.5 rounded">
                {device.os}
              </span>
            )}
            {getDeviceStatusBadge(device)}
            {!isAlreadyAdded && isReady && (
              <Button
                size="sm"
                onClick={() => handleAddDevice(deviceInfo)}
                disabled={createDevice.isPending}
                className="h-7 text-xs"
              >
                <Plus className="mr-1 h-3 w-3" />
                Add
              </Button>
            )}
            {!isAlreadyAdded && needsCredentials && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => toggleDeviceExpanded(device.ip_address)}
                className="h-7 text-xs"
              >
                {expanded ? (
                  <ChevronUp className="h-3 w-3" />
                ) : (
                  <ChevronDown className="h-3 w-3" />
                )}
              </Button>
            )}
          </div>
        </div>
        {needsCredentials && expanded && credentials && renderCredentialForm(deviceInfo, credentials)}
      </div>
    )
  }

  const readyDevicesCount = Array.from(devices.values()).filter(
    (d) =>
      (d.device.status === 'ready' || d.device.credential_status === 'working') &&
      !d.device.already_added
  ).length

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Wifi className="mr-2 h-4 w-4" />
          Discover Devices
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>Discover Network Devices</DialogTitle>
          <DialogDescription>
            Automatically discover and add SSH-enabled devices on your network
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-hidden flex flex-col">
          {!scanId ? (
            <div className="text-center space-y-4 py-8">
              <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
                <Wifi className="w-8 h-8 text-primary" />
              </div>
              <p className="text-muted-foreground">
                Scan your network to find devices
              </p>
              <Button
                onClick={handleStartScan}
                disabled={startScan.isPending}
                size="lg"
              >
                {startScan.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Starting...
                  </>
                ) : (
                  'Start Network Scan'
                )}
              </Button>
            </div>
          ) : (
            <>
              {/* Progress */}
              {scanProgress && scanProgress.status === 'scanning' && (
                <div className="space-y-2 mb-4">
                  <div className="flex justify-between text-sm">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">
                        {scanProgress.phase === 'ping' && 'Discovering hosts...'}
                        {scanProgress.phase === 'ssh_scan' && 'Scanning for SSH...'}
                        {scanProgress.phase === 'credential_test' && 'Testing credentials...'}
                        {scanProgress.phase === 'completed' && 'Scan complete'}
                        {!scanProgress.phase && 'Scanning...'}
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
              )}

              {/* Stats and Actions */}
              {scanProgress && scanProgress.discovered_count > 0 && (
                <div className="flex items-center justify-between mb-4 pb-3 border-b">
                  <div className="flex items-center gap-4">
                    <div>
                      <p className="text-2xl font-bold text-primary">
                        {scanProgress.discovered_count}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        devices found
                      </p>
                    </div>
                    {readyDevicesCount > 0 && (
                      <div>
                        <p className="text-xl font-semibold text-emerald-600 dark:text-emerald-400">
                          {readyDevicesCount}
                        </p>
                        <p className="text-xs text-muted-foreground">ready</p>
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowAlreadyAdded(!showAlreadyAdded)}
                      className="text-xs"
                    >
                      {showAlreadyAdded ? 'Hide' : 'Show'} Added
                    </Button>
                    {readyDevicesCount > 0 && (
                      <Button
                        size="sm"
                        onClick={handleAddAllReady}
                        disabled={createDevice.isPending}
                      >
                        <Plus className="mr-1 h-3 w-3" />
                        Add All Ready ({readyDevicesCount})
                      </Button>
                    )}
                  </div>
                </div>
              )}

              {/* Device List */}
              <div className="space-y-2 overflow-y-auto flex-1">
                {Array.from(devices.values()).map((deviceInfo) =>
                  renderDeviceCard(deviceInfo)
                )}
                {devices.size === 0 && scanProgress?.status === 'completed' && (
                  <div className="text-center py-8 text-muted-foreground">
                    No devices found
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        <div className="flex justify-between pt-4 border-t">
          <Button variant="outline" onClick={handleClose}>
            Close
          </Button>
          {scanProgress?.status === 'completed' && devices.size > 0 && (
            <Button variant="outline" onClick={handleStartScan}>
              <Wifi className="mr-2 h-4 w-4" />
              Scan Again
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
