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
  const [formData, setFormData] = useState({
    name: device.name,
    type: device.type,
    mac_address: device.mac_address || '',
  })

  // Reset form when device changes
  useEffect(() => {
    setFormData({
      name: device.name,
      type: device.type,
      mac_address: device.mac_address || '',
    })
  }, [device])

  const updateDevice = useUpdateDevice()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Build update object with only changed fields
    const updates: { name?: string; type?: DeviceType; mac_address?: string } = {}

    if (formData.name !== device.name) {
      updates.name = formData.name
    }
    if (formData.type !== device.type) {
      updates.type = formData.type
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
              Update device information. IP address cannot be changed.
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
              <Label htmlFor="edit-ip">IP Address (Read-only)</Label>
              <Input
                id="edit-ip"
                value={device.ip_address}
                disabled
                className="bg-muted cursor-not-allowed"
              />
              <p className="text-xs text-muted-foreground">
                IP address cannot be changed after creation
              </p>
            </div>

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
