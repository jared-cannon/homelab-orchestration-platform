import { useState, useEffect } from 'react'
import { toast } from 'sonner'
import { useCreateDevice, useTestConnectionBeforeCreate } from '../api/hooks'
import type { DeviceType } from '../api/types'
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
import { Input } from './ui/input'
import { Label } from './ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select'

export function AddDeviceDialog() {
  const [open, setOpen] = useState(false)
  const [showTailscaleSection, setShowTailscaleSection] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    type: 'server' as DeviceType,
    local_ip_address: '',
    tailscale_address: '',
    primary_connection: 'local' as 'local' | 'tailscale',
    mac_address: '',
    credentials: {
      type: 'auto' as 'auto' | 'password' | 'ssh_key' | 'tailscale',
      username: '',
      password: '',
      ssh_key: '',
      ssh_key_passwd: '',
    },
  })

  const createDevice = useCreateDevice()
  const testConnection = useTestConnectionBeforeCreate()

  // Auto-expand Tailscale section when Tailscale auth is selected
  useEffect(() => {
    if (formData.credentials.type === 'tailscale') {
      setShowTailscaleSection(true)
    }
  }, [formData.credentials.type])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    // Validation: if using Tailscale auth, require Tailscale address
    if (formData.credentials.type === 'tailscale' && !formData.tailscale_address) {
      toast.error('Tailscale address required', {
        description: 'When using Tailscale SSH, please provide a Tailscale address'
      })
      return
    }

    try {
      await createDevice.mutateAsync({
        name: formData.name,
        type: formData.type,
        local_ip_address: formData.local_ip_address,
        tailscale_address: formData.tailscale_address || undefined,
        primary_connection: formData.primary_connection,
        mac_address: formData.mac_address || undefined,
        credentials: formData.credentials,
      })

      toast.success('Device added!', {
        description: `${formData.name} is ready to use`
      })

      // Reset form and close dialog
      setFormData({
        name: '',
        type: 'server',
        local_ip_address: '',
        tailscale_address: '',
        primary_connection: 'local',
        mac_address: '',
        credentials: {
          type: 'auto',
          username: '',
          password: '',
          ssh_key: '',
          ssh_key_passwd: '',
        },
      })
      setShowTailscaleSection(false)
      setOpen(false)
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Something went wrong'

      // Provide helpful error messages
      if (message.includes('invalid IP')) {
        toast.error('Invalid IP address', {
          description: 'Please check the IP address and try again. Example: 192.168.1.100'
        })
      } else if (message.includes('already exists')) {
        toast.error('Device already exists', {
          description: 'A device with this IP address is already added'
        })
      } else {
        toast.error('Could not add device', {
          description: message
        })
      }
    }
  }

  const handleTestConnection = async () => {
    // Validate required fields
    if (!formData.local_ip_address || !formData.credentials.username) {
      toast.error('Missing information', {
        description: 'Please enter an IP address and username first'
      })
      return
    }

    if (formData.credentials.type === 'password' && !formData.credentials.password) {
      toast.error('Missing password', {
        description: 'Please enter a password to test the connection'
      })
      return
    }

    if (formData.credentials.type === 'ssh_key' && !formData.credentials.ssh_key) {
      toast.error('Missing security key', {
        description: 'Please paste your security key to test the connection'
      })
      return
    }

    // For auto type, no additional validation needed - will use SSH agent/default keys

    // Use primary connection for testing
    const testAddress = formData.primary_connection === 'tailscale' && formData.tailscale_address
      ? formData.tailscale_address
      : formData.local_ip_address

    try {
      const result = await testConnection.mutateAsync({
        ip_address: testAddress,
        credentials: formData.credentials,
      })

      // Success case
      if (result.success || result.ssh_connection) {
        const dockerInfo = result.docker_installed
          ? `✓ Docker ${result.docker_version} ${result.docker_running ? '(running)' : '(not running)'}`
          : '⚠️ Docker not installed'

        toast.success('Connection successful!', {
          description: `Connected to ${testAddress}. ${dockerInfo}`
        })
      }
    } catch (error) {
      const err = error as Error
      const message = err.message || 'Connection failed'

      // Provide helpful error messages based on the error
      if (message.includes('connection refused') || message.includes('no route to host')) {
        toast.error('Cannot reach device', {
          description: 'Make sure the device is powered on and connected to your network. Check the IP address.'
        })
      } else if (message.includes('authentication failed') || message.includes('permission denied')) {
        toast.error('Login failed', {
          description: 'Check your username and password/security key are correct'
        })
      } else if (message.includes('timeout')) {
        toast.error('Connection timed out', {
          description: 'The device is taking too long to respond. Check your network connection.'
        })
      } else if (message.includes('invalid IP')) {
        toast.error('Invalid IP address', {
          description: 'Please check the IP address format. Example: 192.168.1.100'
        })
      } else {
        toast.error('Connection failed', {
          description: message
        })
      }
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>Add Device</Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-[525px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Add New Device</DialogTitle>
            <DialogDescription>
              Add a new device to your homelab. We'll need a few details to connect to it.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Device Name</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                placeholder="My Server"
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="type">Device Type</Label>
              <Select
                value={formData.type}
                onValueChange={(value: DeviceType) =>
                  setFormData({ ...formData, type: value })
                }
              >
                <SelectTrigger id="type">
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
              <Label htmlFor="local-ip">Local IP Address</Label>
              <Input
                id="local-ip"
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
                  <Label htmlFor="tailscale-ip">Tailscale IP or Hostname</Label>
                  <Input
                    id="tailscale-ip"
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
                  <Label htmlFor="primary-connection">Preferred Connection</Label>
                  <Select
                    value={formData.primary_connection}
                    onValueChange={(value: 'local' | 'tailscale') =>
                      setFormData({ ...formData, primary_connection: value })
                    }
                  >
                    <SelectTrigger id="primary-connection">
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
              <Label htmlFor="mac">MAC Address (Optional)</Label>
              <Input
                id="mac"
                value={formData.mac_address}
                onChange={(e) =>
                  setFormData({ ...formData, mac_address: e.target.value })
                }
                placeholder="00:11:22:33:44:55"
              />
            </div>

            <div className="border-t pt-4">
              <h4 className="text-sm font-medium mb-3">Device Login</h4>

              <div className="grid gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="auth-type">Login Method</Label>
                  <Select
                    value={formData.credentials.type}
                    onValueChange={(value: 'auto' | 'password' | 'ssh_key' | 'tailscale') =>
                      setFormData({
                        ...formData,
                        credentials: { ...formData.credentials, type: value },
                      })
                    }
                  >
                    <SelectTrigger id="auth-type">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="auto">Use My SSH Key (Recommended)</SelectItem>
                      <SelectItem value="password">Password</SelectItem>
                      <SelectItem value="ssh_key">Security Key</SelectItem>
                      <SelectItem value="tailscale">Tailscale SSH</SelectItem>
                    </SelectContent>
                  </Select>
                  {formData.credentials.type === 'auto' && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Will use your default SSH key or SSH agent - no credentials stored
                    </p>
                  )}
                  {formData.credentials.type === 'tailscale' && (
                    <p className="text-xs text-muted-foreground mt-1">
                      Uses Tailscale's built-in SSH - requires device to be on your Tailnet
                    </p>
                  )}
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="username">Username</Label>
                  <Input
                    id="username"
                    value={formData.credentials.username}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        credentials: {
                          ...formData.credentials,
                          username: e.target.value,
                        },
                      })
                    }
                    placeholder="root"
                    required
                  />
                </div>

                {formData.credentials.type === 'password' && (
                  <div className="grid gap-2">
                    <Label htmlFor="password">Password</Label>
                    <Input
                      id="password"
                      type="password"
                      value={formData.credentials.password}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          credentials: {
                            ...formData.credentials,
                            password: e.target.value,
                          },
                        })
                      }
                      required
                    />
                  </div>
                )}

                {formData.credentials.type === 'ssh_key' && (
                  <>
                    <div className="grid gap-2">
                      <Label htmlFor="ssh-key">Security Key</Label>
                      <textarea
                        id="ssh-key"
                        value={formData.credentials.ssh_key}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            credentials: {
                              ...formData.credentials,
                              ssh_key: e.target.value,
                            },
                          })
                        }
                        className="min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                        placeholder="Paste your private key here"
                        required
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label htmlFor="key-passphrase">
                        Key Password (Optional)
                      </Label>
                      <Input
                        id="key-passphrase"
                        type="password"
                        value={formData.credentials.ssh_key_passwd}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            credentials: {
                              ...formData.credentials,
                              ssh_key_passwd: e.target.value,
                            },
                          })
                        }
                      />
                    </div>
                  </>
                )}
              </div>

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
                  Test the connection before adding the device
                </p>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createDevice.isPending}>
              {createDevice.isPending ? 'Adding...' : 'Add Device'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
