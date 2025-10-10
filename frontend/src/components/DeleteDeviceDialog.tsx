import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { useDeleteDevice } from '../api/hooks'
import type { Device } from '../api/types'
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
import { AlertCircle } from 'lucide-react'

interface DeleteDeviceDialogProps {
  device: Device
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DeleteDeviceDialog({ device, open, onOpenChange }: DeleteDeviceDialogProps) {
  const [confirmationText, setConfirmationText] = useState('')
  const navigate = useNavigate()
  const deleteDevice = useDeleteDevice()

  const isConfirmed = confirmationText === device.name

  const handleDelete = async () => {
    if (!isConfirmed) {
      toast.error('Please type the device name to confirm')
      return
    }

    try {
      await deleteDevice.mutateAsync(device.id)

      toast.success('Device deleted', {
        description: `${device.name} has been removed from your homelab`
      })

      // Navigate back to dashboard
      navigate('/')
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Something went wrong'

      toast.error('Could not delete device', {
        description: message
      })
    }
  }

  const handleOpenChange = (newOpen: boolean) => {
    // Reset confirmation text when closing
    if (!newOpen) {
      setConfirmationText('')
    }
    onOpenChange(newOpen)
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[525px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertCircle className="w-5 h-5" />
            Delete Device
          </DialogTitle>
          <DialogDescription>
            This action cannot be undone. This will permanently delete the device and remove all associated data including stored credentials.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="rounded-lg bg-destructive/10 border border-destructive/20 p-4">
            <p className="text-sm font-medium mb-2">You are about to delete:</p>
            <div className="space-y-1 text-sm">
              <p><span className="text-muted-foreground">Name:</span> <span className="font-semibold">{device.name}</span></p>
              <p><span className="text-muted-foreground">IP:</span> <span className="font-mono">{device.ip_address}</span></p>
              <p><span className="text-muted-foreground">Type:</span> <span className="capitalize">{device.type}</span></p>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="confirm-name">
              Type <span className="font-mono font-semibold">{device.name}</span> to confirm
            </Label>
            <Input
              id="confirm-name"
              value={confirmationText}
              onChange={(e) => setConfirmationText(e.target.value)}
              placeholder={device.name}
              autoComplete="off"
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => handleOpenChange(false)}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={!isConfirmed || deleteDevice.isPending}
          >
            {deleteDevice.isPending ? 'Deleting...' : 'Delete Device'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
