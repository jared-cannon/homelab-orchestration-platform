import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { useUpdateDevice } from '../api/hooks'
import type { Device, DeviceType } from '../api/types'
import { Button } from './ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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

interface EditDeviceDialogProps {
  device: Device
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function EditDeviceDialog({ device, open, onOpenChange }: EditDeviceDialogProps) {
  const [showTailscaleSection, setShowTailscaleSection] = useState(!!device.tailscale_address)
  const [formData, setFormData] = useState({
    name: device.name,
    type: device.type,
    local_ip_address: device.local_ip_address,
    tailscale_address: device.tailscale_address || '',
    primary_connection: device.primary_connection || ('local' as 'local' | 'tailscale'),
    mac_address: device.mac_address || '',
  })

  // Reset form when device changes
  useEffect(() => {
    setFormData({
      name: device.name,
      type: device.type,
      local_ip_address: device.local_ip_address,
      tailscale_address: device.tailscale_address || '',
      primary_connection: device.primary_connection || ('local' as 'local' | 'tailscale'),
      mac_address: device.mac_address || '',
    })
    setShowTailscaleSection(!!device.tailscale_address)
  }, [device])

  const updateDevice = useUpdateDevice()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Build update object with only changed fields
    const updates: {
      name?: string
      type?: DeviceType
      local_ip_address?: string
      tailscale_address?: string
      primary_connection?: 'local' | 'tailscale'
      mac_address?: string
    } = {}

    if (formData.name !== device.name) {
      updates.name = formData.name
    }
    if (formData.type !== device.type) {
      updates.type = formData.type
    }
    if (formData.local_ip_address !== device.local_ip_address) {
      updates.local_ip_address = formData.local_ip_address
    }
    if (formData.tailscale_address !== (device.tailscale_address || '')) {
      updates.tailscale_address = formData.tailscale_address || undefined
    }
    if (formData.primary_connection !== device.primary_connection) {
      updates.primary_connection = formData.primary_connection
    }
    if (formData.mac_address !== (device.mac_address || '')) {
      updates.mac_address = formData.mac_address || undefined
    }

    // If no changes, just close dialog
    if (Object.keys(updates).length === 0) {
      toast.info('No changes made')
      onOpenChange(false)
      return
    }

    try {
      await updateDevice.mutateAsync({
        id: device.id,
        data: updates,
      })

      toast.success('Device updated!', {
        description: `${formData.name} has been updated`
      })

      onOpenChange(false)
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Something went wrong'

      toast.error('Could not update device', {
        description: message
      })
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[525px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Device</DialogTitle>
            <DialogDescription>
              Update device information including name, type, IP address, and MAC address.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="edit-name">Device Name</Label>
              <Input
                id="edit-name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="My Server"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="edit-type">Device Type</Label>
              <Select
                value={formData.type}
                onValueChange={(value: DeviceType) =>
                  setFormData({ ...formData, type: value })
                }
              >
                <SelectTrigger id="edit-type">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="router">Router</SelectItem>
                  <SelectItem value="server">Server</SelectItem>
                  <SelectItem value="nas">NAS</SelectItem>
                  <SelectItem value="switch">Switch</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="edit-local-ip">Local IP Address</Label>
              <Input
                id="edit-local-ip"
                value={formData.local_ip_address}
                onChange={(e) =>
                  setFormData({ ...formData, local_ip_address: e.target.value })
                }
                placeholder="192.168.1.100"
                required
              />
            </div>

            {!showTailscaleSection && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setShowTailscaleSection(true)}
                className="w-full"
              >
                + Add Tailscale Address (Optional)
              </Button>
            )}

            {showTailscaleSection && (
              <div className="grid gap-3 p-4 border rounded-md bg-muted/30">
                <div className="flex items-center justify-between">
                  <Label className="text-sm font-medium">Tailscale Connection</Label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setShowTailscaleSection(false)
                      setFormData({
                        ...formData,
                        tailscale_address: '',
                        primary_connection: 'local',
                      })
                    }}
                    className="h-6 px-2"
                  >
                    Remove
                  </Button>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="edit-tailscale-ip">Tailscale IP or Hostname</Label>
                  <Input
                    id="edit-tailscale-ip"
                    value={formData.tailscale_address}
                    onChange={(e) =>
                      setFormData({ ...formData, tailscale_address: e.target.value })
                    }
                    placeholder="100.64.1.5 or machine.wolf-bear.ts.net"
                  />
                  <p className="text-xs text-muted-foreground">
                    For remote access via Tailscale VPN
                  </p>
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="edit-primary-connection">Preferred Connection</Label>
                  <Select
                    value={formData.primary_connection}
                    onValueChange={(value: 'local' | 'tailscale') =>
                      setFormData({ ...formData, primary_connection: value })
                    }
                  >
                    <SelectTrigger id="edit-primary-connection">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="local">Local IP (Primary)</SelectItem>
                      <SelectItem value="tailscale">Tailscale (Primary)</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">
                    System will try primary first, then fallback to the other
                  </p>
                </div>
              </div>
            )}

            <div className="grid gap-2">
              <Label htmlFor="edit-mac">MAC Address (Optional)</Label>
              <Input
                id="edit-mac"
                value={formData.mac_address}
                onChange={(e) =>
                  setFormData({ ...formData, mac_address: e.target.value })
                }
                placeholder="00:11:22:33:44:55"
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={updateDevice.isPending}>
              {updateDevice.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
