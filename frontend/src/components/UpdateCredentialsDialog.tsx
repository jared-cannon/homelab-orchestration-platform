import { useState } from 'react'
import { toast } from 'sonner'
import { useUpdateDeviceCredentials, useTestConnectionBeforeCreate } from '../api/hooks'
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
import { CredentialsForm } from './CredentialsForm'

interface UpdateCredentialsDialogProps {
  device: Device
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function UpdateCredentialsDialog({ device, open, onOpenChange }: UpdateCredentialsDialogProps) {
  const [formData, setFormData] = useState({
    type: 'auto' as 'auto' | 'password' | 'ssh_key',
    username: '',
    password: '',
    ssh_key: '',
    ssh_key_passwd: '',
  })

  const updateCredentials = useUpdateDeviceCredentials()
  const testConnection = useTestConnectionBeforeCreate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      await updateCredentials.mutateAsync({
        id: device.id,
        data: {
          credentials: formData,
        },
      })

      toast.success('Credentials updated!', {
        description: `Login information for ${device.name} has been updated`
      })

      // Reset form and close dialog
      setFormData({
        type: 'auto',
        username: '',
        password: '',
        ssh_key: '',
        ssh_key_passwd: '',
      })
      onOpenChange(false)
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Something went wrong'

      toast.error('Could not update credentials', {
        description: message
      })
    }
  }

  const handleTestConnection = async () => {
    // Validate required fields
    if (!formData.username) {
      toast.error('Missing username', {
        description: 'Please enter a username first'
      })
      return
    }

    if (formData.type === 'password' && !formData.password) {
      toast.error('Missing password', {
        description: 'Please enter a password to test the connection'
      })
      return
    }

    if (formData.type === 'ssh_key' && !formData.ssh_key) {
      toast.error('Missing security key', {
        description: 'Please paste your security key to test the connection'
      })
      return
    }

    try {
      const result = await testConnection.mutateAsync({
        ip_address: device.ip_address,
        credentials: formData,
      })

      // Success case
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

      // Provide helpful error messages based on the error
      if (message.includes('connection refused') || message.includes('no route to host')) {
        toast.error('Cannot reach device', {
          description: 'Make sure the device is powered on and connected to your network.'
        })
      } else if (message.includes('authentication failed') || message.includes('permission denied')) {
        toast.error('Login failed', {
          description: 'Check your username and password/security key are correct'
        })
      } else if (message.includes('timeout')) {
        toast.error('Connection timed out', {
          description: 'The device is taking too long to respond. Check your network connection.'
        })
      } else {
        toast.error('Connection failed', {
          description: message
        })
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[525px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Update Login Credentials</DialogTitle>
            <DialogDescription>
              Update the login information for {device.name} ({device.ip_address})
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <CredentialsForm
              credentials={formData}
              onChange={setFormData}
              idPrefix="update-cred"
            />

            <div className="pt-4 border-t">
              <Button
                type="button"
                variant="outline"
                onClick={handleTestConnection}
                disabled={testConnection.isPending}
                className="w-full"
              >
                {testConnection.isPending ? 'Testing...' : 'Test Connection'}
              </Button>
              <p className="text-xs text-muted-foreground mt-2 text-center">
                Test the new credentials before saving
              </p>
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
            <Button type="submit" disabled={updateCredentials.isPending}>
              {updateCredentials.isPending ? 'Updating...' : 'Update Credentials'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
