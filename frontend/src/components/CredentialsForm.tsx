import { Input } from './ui/input'
import { Label } from './ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from './ui/select'
import type { DeviceCredentials } from '../api/types'

interface CredentialsFormProps {
  credentials: DeviceCredentials
  onChange: (credentials: DeviceCredentials) => void
  idPrefix?: string // For unique IDs when multiple forms on same page
  showLabels?: boolean // Whether to show field labels
}

export function CredentialsForm({
  credentials,
  onChange,
  idPrefix = 'cred',
  showLabels = true,
}: CredentialsFormProps) {
  return (
    <>
      <div className="grid gap-2">
        {showLabels && <Label htmlFor={`${idPrefix}-auth-type`}>Login Method</Label>}
        <Select
          value={credentials.type}
          onValueChange={(value: 'auto' | 'password' | 'ssh_key') =>
            onChange({
              ...credentials,
              type: value,
            })
          }
        >
          <SelectTrigger id={`${idPrefix}-auth-type`}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="auto">Use My SSH Key (Recommended)</SelectItem>
            <SelectItem value="password">Password</SelectItem>
            <SelectItem value="ssh_key">Security Key</SelectItem>
          </SelectContent>
        </Select>
        {credentials.type === 'auto' && (
          <p className="text-xs text-muted-foreground mt-1">
            Will use your default SSH key or SSH agent - no credentials stored
          </p>
        )}
      </div>

      <div className="grid gap-2">
        {showLabels && <Label htmlFor={`${idPrefix}-username`}>Username</Label>}
        <Input
          id={`${idPrefix}-username`}
          value={credentials.username}
          onChange={(e) =>
            onChange({
              ...credentials,
              username: e.target.value,
            })
          }
          placeholder="root"
          required
        />
      </div>

      {credentials.type === 'password' && (
        <div className="grid gap-2">
          {showLabels && <Label htmlFor={`${idPrefix}-password`}>Password</Label>}
          <Input
            id={`${idPrefix}-password`}
            type="password"
            value={credentials.password}
            onChange={(e) =>
              onChange({
                ...credentials,
                password: e.target.value,
              })
            }
            required
          />
        </div>
      )}

      {credentials.type === 'ssh_key' && (
        <>
          <div className="grid gap-2">
            {showLabels && <Label htmlFor={`${idPrefix}-ssh-key`}>Security Key</Label>}
            <textarea
              id={`${idPrefix}-ssh-key`}
              value={credentials.ssh_key}
              onChange={(e) =>
                onChange({
                  ...credentials,
                  ssh_key: e.target.value,
                })
              }
              className="min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              placeholder="Paste your private key here"
              required
            />
          </div>
          <div className="grid gap-2">
            {showLabels && (
              <Label htmlFor={`${idPrefix}-key-passphrase`}>
                Key Password (Optional)
              </Label>
            )}
            <Input
              id={`${idPrefix}-key-passphrase`}
              type="password"
              value={credentials.ssh_key_passwd}
              onChange={(e) =>
                onChange({
                  ...credentials,
                  ssh_key_passwd: e.target.value,
                })
              }
            />
          </div>
        </>
      )}
    </>
  )
}
