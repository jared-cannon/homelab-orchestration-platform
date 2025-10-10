import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Package, HardDrive, Database, Plus, Trash2, Loader2, RefreshCw, Info } from 'lucide-react'
import { toast } from 'sonner'
import { Card } from './ui/card'
import { apiClient } from '../api/client'
import type {
  InstallSoftwareRequest,
  SoftwareType,
  CreateVolumeRequest,
  VolumeType,
} from '../api/types'

interface DeviceManagementProps {
  deviceId: string
}

type Tab = 'software' | 'nfs' | 'volumes'

export function DeviceManagement({ deviceId }: DeviceManagementProps) {
  const [activeTab, setActiveTab] = useState<Tab>('software')

  return (
    <Card>
      <div className="p-6">
        <div className="flex items-center gap-3 mb-6">
          <div className="rounded-lg bg-primary/10 p-2">
            <Package className="w-5 h-5 text-primary" />
          </div>
          <h2 className="text-lg font-semibold">Device Management</h2>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6 border-b border-border">
          <button
            onClick={() => setActiveTab('software')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'software'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <div className="flex items-center gap-2">
              <Package className="w-4 h-4" />
              Software
            </div>
          </button>
          <button
            onClick={() => setActiveTab('nfs')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'nfs'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <div className="flex items-center gap-2">
              <HardDrive className="w-4 h-4" />
              NFS Storage
            </div>
          </button>
          <button
            onClick={() => setActiveTab('volumes')}
            className={`px-4 py-2 font-medium transition-colors ${
              activeTab === 'volumes'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            <div className="flex items-center gap-2">
              <Database className="w-4 h-4" />
              Docker Volumes
            </div>
          </button>
        </div>

        {/* Tab Content */}
        {activeTab === 'software' && <SoftwareTab deviceId={deviceId} />}
        {activeTab === 'nfs' && <NFSTab deviceId={deviceId} />}
        {activeTab === 'volumes' && <VolumesTab deviceId={deviceId} />}
      </div>
    </Card>
  )
}

// Software Tab
function SoftwareTab({ deviceId }: { deviceId: string }) {
  const queryClient = useQueryClient()
  const [installing, setInstalling] = useState(false)

  const { data: software = [], isLoading } = useQuery({
    queryKey: ['software', deviceId],
    queryFn: () => apiClient.listInstalledSoftware(deviceId),
  })

  const detectMutation = useMutation({
    mutationFn: () => apiClient.detectInstalledSoftware(deviceId),
    onSuccess: (detected) => {
      queryClient.invalidateQueries({ queryKey: ['software', deviceId] })
      if (detected.length > 0) {
        toast.success(`Detected ${detected.length} installed package(s)`)
      } else {
        toast.info('No software detected')
      }
    },
    onError: (error: Error) => {
      toast.error('Detection failed', { description: error.message })
    },
  })

  const handleRefresh = () => {
    detectMutation.mutate()
  }

  const installMutation = useMutation({
    mutationFn: (data: InstallSoftwareRequest) =>
      apiClient.installSoftware(deviceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['software', deviceId] })
      toast.success('Software installed successfully')
      setInstalling(false)
    },
    onError: (error: Error) => {
      toast.error('Installation failed', { description: error.message })
      setInstalling(false)
    },
  })

  const handleInstall = async (softwareType: SoftwareType) => {
    setInstalling(true)
    installMutation.mutate({
      software_type: softwareType,
      add_user_to_group: softwareType === 'docker',
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const availableSoftware: { name: SoftwareType; label: string }[] = [
    { name: 'docker', label: 'Docker Engine' },
    { name: 'nfs-server', label: 'NFS Server' },
    { name: 'nfs-client', label: 'NFS Client' },
  ]

  const installedNames = software.map((s) => s.name)
  const available = availableSoftware.filter(
    (s) => !installedNames.includes(s.name)
  )

  return (
    <div className="space-y-4">
      {/* Installed Software */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-muted-foreground">Installed Software</h3>
          <div className="flex gap-2">
            <button
              onClick={() => toast.info('Update checking coming soon', {
                description: 'This feature will check for available package updates'
              })}
              className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
            >
              Check for Updates
            </button>
            <button
              onClick={handleRefresh}
              disabled={detectMutation.isPending}
              className="p-1 rounded hover:bg-accent transition-colors disabled:opacity-50"
              title="Scan device for installed software"
            >
              <RefreshCw className={`w-4 h-4 ${detectMutation.isPending ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {software.length === 0 ? (
          <div className="text-center py-6">
            <p className="text-sm text-muted-foreground mb-2">No software detected</p>
            <p className="text-xs text-muted-foreground">
              Click the refresh icon to scan for installed packages
            </p>
          </div>
        ) : (
          <>
            <div className="space-y-2 mb-3">
              {software.map((s) => (
                <div
                  key={s.id}
                  className="flex items-center justify-between p-3 bg-card border border-border rounded-lg"
                >
                  <div>
                    <p className="font-medium capitalize">{s.name.replace('-', ' ')}</p>
                    <p className="text-sm text-muted-foreground">{s.version}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-1 text-xs rounded-full bg-green-500/10 text-green-500">
                      Installed
                    </span>
                  </div>
                </div>
              ))}
            </div>
            <div className="flex items-start gap-2 p-3 bg-muted/50 rounded-lg">
              <Info className="w-4 h-4 text-muted-foreground mt-0.5 flex-shrink-0" />
              <p className="text-xs text-muted-foreground">
                Uninstallation should be done manually via SSH to prevent accidental service disruption
              </p>
            </div>
          </>
        )}
      </div>

      {/* Available Software */}
      {available.length > 0 && (
        <div>
          <h3 className="text-sm font-medium text-muted-foreground mb-3">Available to Install</h3>
          <div className="space-y-2">
            {available.map((s) => (
              <button
                key={s.name}
                onClick={() => handleInstall(s.name)}
                disabled={installing}
                className="w-full flex items-center justify-between p-3 bg-card border border-border rounded-lg hover:bg-accent transition-colors disabled:opacity-50"
              >
                <span className="font-medium">{s.label}</span>
                <Plus className="w-4 h-4" />
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// NFS Tab
function NFSTab({ deviceId }: { deviceId: string }) {
  const queryClient = useQueryClient()
  const [showExportForm, setShowExportForm] = useState(false)
  const [showMountForm, setShowMountForm] = useState(false)

  const { data: exports = [], isLoading: exportsLoading } = useQuery({
    queryKey: ['nfs-exports', deviceId],
    queryFn: () => apiClient.listNFSExports(deviceId),
  })

  const { data: mounts = [], isLoading: mountsLoading } = useQuery({
    queryKey: ['nfs-mounts', deviceId],
    queryFn: () => apiClient.listNFSMounts(deviceId),
  })

  const setupServerMutation = useMutation({
    mutationFn: (exportPath: string) =>
      apiClient.setupNFSServer(deviceId, { export_path: exportPath }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nfs-exports', deviceId] })
      toast.success('NFS server setup successfully')
      setShowExportForm(false)
    },
    onError: (error: Error) => {
      toast.error('Setup failed', { description: error.message })
    },
  })

  const removeExportMutation = useMutation({
    mutationFn: (exportId: string) => apiClient.removeNFSExport(deviceId, exportId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nfs-exports', deviceId] })
      toast.success('Export removed')
    },
  })

  const mountMutation = useMutation({
    mutationFn: (data: { serverIp: string; remotePath: string; localPath: string }) =>
      apiClient.mountNFSShare(deviceId, {
        server_ip: data.serverIp,
        remote_path: data.remotePath,
        local_path: data.localPath,
        permanent: true,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nfs-mounts', deviceId] })
      toast.success('NFS share mounted')
      setShowMountForm(false)
    },
    onError: (error: Error) => {
      toast.error('Mount failed', { description: error.message })
    },
  })

  const unmountMutation = useMutation({
    mutationFn: (mountId: string) => apiClient.unmountNFSShare(deviceId, mountId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['nfs-mounts', deviceId] })
      toast.success('Share unmounted')
    },
  })

  return (
    <div className="space-y-6">
      {/* NFS Exports (Server) */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium">NFS Exports (Server)</h3>
          <button
            onClick={() => setShowExportForm(!showExportForm)}
            className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
          >
            {showExportForm ? 'Cancel' : 'Add Export'}
          </button>
        </div>

        {showExportForm && (
          <form
            onSubmit={(e) => {
              e.preventDefault()
              const form = e.target as HTMLFormElement
              const path = (form.elements.namedItem('path') as HTMLInputElement).value
              setupServerMutation.mutate(path)
            }}
            className="mb-4 p-4 bg-card border border-border rounded-lg space-y-3"
          >
            <input
              name="path"
              placeholder="/srv/nfs/shared"
              required
              className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <button
              type="submit"
              disabled={setupServerMutation.isPending}
              className="w-full px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
            >
              {setupServerMutation.isPending ? 'Setting up...' : 'Setup NFS Export'}
            </button>
          </form>
        )}

        {exportsLoading ? (
          <div className="flex justify-center py-4">
            <Loader2 className="w-5 h-5 animate-spin" />
          </div>
        ) : exports.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4">No NFS exports configured</p>
        ) : (
          <div className="space-y-2">
            {exports.map((exp) => (
              <div
                key={exp.id}
                className="flex items-center justify-between p-3 bg-card border border-border rounded-lg"
              >
                <div>
                  <p className="font-medium font-mono text-sm">{exp.path}</p>
                  <p className="text-xs text-muted-foreground">
                    Clients: {exp.client_cidr} | Options: {exp.options}
                  </p>
                </div>
                <button
                  onClick={() => removeExportMutation.mutate(exp.id)}
                  className="p-2 text-destructive hover:bg-destructive/10 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* NFS Mounts (Client) */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium">NFS Mounts (Client)</h3>
          <button
            onClick={() => setShowMountForm(!showMountForm)}
            className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
          >
            {showMountForm ? 'Cancel' : 'Mount Share'}
          </button>
        </div>

        {showMountForm && (
          <form
            onSubmit={(e) => {
              e.preventDefault()
              const form = e.target as HTMLFormElement
              mountMutation.mutate({
                serverIp: (form.elements.namedItem('serverIp') as HTMLInputElement).value,
                remotePath: (form.elements.namedItem('remotePath') as HTMLInputElement)
                  .value,
                localPath: (form.elements.namedItem('localPath') as HTMLInputElement).value,
              })
            }}
            className="mb-4 p-4 bg-card border border-border rounded-lg space-y-3"
          >
            <input
              name="serverIp"
              placeholder="NFS Server IP (e.g., 192.168.1.100)"
              required
              className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <input
              name="remotePath"
              placeholder="Remote Path (e.g., /srv/nfs/shared)"
              required
              className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <input
              name="localPath"
              placeholder="Local Mount Point (e.g., /mnt/nfs/shared)"
              required
              className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
            />
            <button
              type="submit"
              disabled={mountMutation.isPending}
              className="w-full px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
            >
              {mountMutation.isPending ? 'Mounting...' : 'Mount NFS Share'}
            </button>
          </form>
        )}

        {mountsLoading ? (
          <div className="flex justify-center py-4">
            <Loader2 className="w-5 h-5 animate-spin" />
          </div>
        ) : mounts.length === 0 ? (
          <p className="text-sm text-muted-foreground py-4">No NFS mounts configured</p>
        ) : (
          <div className="space-y-2">
            {mounts.map((mount) => (
              <div
                key={mount.id}
                className="flex items-center justify-between p-3 bg-card border border-border rounded-lg"
              >
                <div>
                  <p className="font-medium font-mono text-sm">
                    {mount.server_ip}:{mount.remote_path}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    Mounted at: {mount.local_path} | Permanent: {mount.permanent ? 'Yes' : 'No'}
                  </p>
                </div>
                <button
                  onClick={() => unmountMutation.mutate(mount.id)}
                  className="p-2 text-destructive hover:bg-destructive/10 rounded"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// Volumes Tab
function VolumesTab({ deviceId }: { deviceId: string }) {
  const queryClient = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [volumeType, setVolumeType] = useState<VolumeType>('local')

  const { data: volumes = [], isLoading, refetch } = useQuery({
    queryKey: ['volumes', deviceId],
    queryFn: () => apiClient.listVolumes(deviceId),
  })

  const createMutation = useMutation({
    mutationFn: (data: CreateVolumeRequest) => apiClient.createVolume(deviceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes', deviceId] })
      toast.success('Volume created successfully')
      setShowForm(false)
    },
    onError: (error: Error) => {
      toast.error('Volume creation failed', { description: error.message })
    },
  })

  const removeMutation = useMutation({
    mutationFn: (name: string) => apiClient.removeVolume(deviceId, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes', deviceId] })
      toast.success('Volume removed')
    },
    onError: (error: Error) => {
      toast.error('Remove failed', { description: error.message })
    },
  })

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.target as HTMLFormElement
    const name = (form.elements.namedItem('name') as HTMLInputElement).value

    if (volumeType === 'local') {
      createMutation.mutate({ name, type: 'local' })
    } else {
      const serverIp = (form.elements.namedItem('serverIp') as HTMLInputElement).value
      const nfsPath = (form.elements.namedItem('nfsPath') as HTMLInputElement).value
      createMutation.mutate({
        name,
        type: 'nfs',
        nfs_server_ip: serverIp,
        nfs_path: nfsPath,
      })
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">Docker Volumes</h3>
          <button
            onClick={() => refetch()}
            className="p-1 rounded hover:bg-accent transition-colors"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="px-3 py-1 text-sm bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
        >
          {showForm ? 'Cancel' : 'Create Volume'}
        </button>
      </div>

      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="p-4 bg-card border border-border rounded-lg space-y-3"
        >
          <div>
            <label className="block text-sm font-medium mb-2">Volume Type</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setVolumeType('local')}
                className={`px-4 py-2 rounded border ${
                  volumeType === 'local'
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'border-border hover:bg-accent'
                }`}
              >
                Local
              </button>
              <button
                type="button"
                onClick={() => setVolumeType('nfs')}
                className={`px-4 py-2 rounded border ${
                  volumeType === 'nfs'
                    ? 'bg-primary text-primary-foreground border-primary'
                    : 'border-border hover:bg-accent'
                }`}
              >
                NFS
              </button>
            </div>
          </div>

          <input
            name="name"
            placeholder="Volume name"
            required
            className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
          />

          {volumeType === 'nfs' && (
            <>
              <input
                name="serverIp"
                placeholder="NFS Server IP"
                required
                className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <input
                name="nfsPath"
                placeholder="NFS Path (e.g., /srv/nfs/shared)"
                required
                className="w-full px-3 py-2 bg-background border border-border rounded focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </>
          )}

          <button
            type="submit"
            disabled={createMutation.isPending}
            className="w-full px-4 py-2 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50"
          >
            {createMutation.isPending ? 'Creating...' : 'Create Volume'}
          </button>
        </form>
      )}

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
        </div>
      ) : volumes.length === 0 ? (
        <p className="text-sm text-muted-foreground py-4">No volumes created</p>
      ) : (
        <div className="space-y-2">
          {volumes.map((vol) => (
            <div
              key={vol.id}
              className="flex items-center justify-between p-3 bg-card border border-border rounded-lg"
            >
              <div>
                <div className="flex items-center gap-2">
                  <p className="font-medium font-mono text-sm">{vol.name}</p>
                  <span
                    className={`px-2 py-0.5 text-xs rounded ${
                      vol.type === 'nfs'
                        ? 'bg-blue-500/10 text-blue-500'
                        : 'bg-gray-500/10 text-gray-500'
                    }`}
                  >
                    {vol.type}
                  </span>
                  {vol.in_use && (
                    <span className="px-2 py-0.5 text-xs rounded bg-green-500/10 text-green-500">
                      In Use
                    </span>
                  )}
                </div>
                {vol.type === 'nfs' && vol.nfs_server_ip && (
                  <p className="text-xs text-muted-foreground mt-1">
                    {vol.nfs_server_ip}:{vol.nfs_path}
                  </p>
                )}
              </div>
              <button
                onClick={() => removeMutation.mutate(vol.name)}
                disabled={removeMutation.isPending}
                className="p-2 text-destructive hover:bg-destructive/10 rounded disabled:opacity-50"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
