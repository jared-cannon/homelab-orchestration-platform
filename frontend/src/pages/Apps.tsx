import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  useDeployments,
  useDeleteDeployment,
  useCancelDeployment,
  useRestartDeployment,
  useStopDeployment,
  useStartDeployment,
  useDeploymentAccessURLs,
  useTroubleshootDeployment,
  useCleanupDeployments,
} from '../api/hooks'
import type { Deployment, DeploymentStatus } from '../api/types'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '../components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../components/ui/dropdown-menu'
import {
  Loader2,
  MoreVertical,
  Trash2,
  ExternalLink,
  CheckCircle,
  XCircle,
  Clock,
  AlertCircle,
  RefreshCw,
  FileText,
  StopCircle,
  Play,
  Square,
  RotateCw,
  Wrench,
  Server,
  ChevronDown,
  ChevronRight,
  Package,
  Activity,
  HardDrive,
  Cpu,
} from 'lucide-react'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../components/ui/dialog'
import { LogViewer } from '../components/LogViewer'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../components/ui/collapsible'

type FilterStatus = DeploymentStatus | 'all' | 'deploying_composite'

const IN_PROGRESS_STATUSES: DeploymentStatus[] = ['validating', 'preparing', 'deploying', 'configuring', 'health_check']

const STATUS_LABELS: Record<FilterStatus, string> = {
  all: 'apps',
  running: 'running apps',
  deploying_composite: 'deploying apps',
  failed: 'failed apps',
  validating: 'validating apps',
  preparing: 'preparing apps',
  deploying: 'deploying apps',
  configuring: 'configuring apps',
  health_check: 'apps in health check',
  stopped: 'stopped apps',
  rolling_back: 'apps rolling back',
  rolled_back: 'rolled back apps',
}

// Grouped app interface
interface GroupedApp {
  recipe_slug: string
  recipe_name: string
  deployments: Deployment[]
  statusCounts: {
    running: number
    failed: number
    deploying: number
    stopped: number
    rolling_back: number
    total: number
  }
  devices: Array<{ id: string; name: string }>
}

export function AppsPage() {
  const navigate = useNavigate()
  const [selectedStatus, setSelectedStatus] = useState<FilterStatus>('all')
  const [deploymentToDelete, setDeploymentToDelete] = useState<Deployment | null>(null)
  const [logsDialogOpen, setLogsDialogOpen] = useState(false)
  const [selectedLogs, setSelectedLogs] = useState({ name: '', logs: '' })
  const [troubleshootDialogOpen, setTroubleshootDialogOpen] = useState(false)
  const [selectedDeploymentId, setSelectedDeploymentId] = useState<string | null>(null)
  const [operationStates, setOperationStates] = useState<Record<string, boolean>>({})
  const [expandedApps, setExpandedApps] = useState<Set<string>>(new Set())
  const [failedDialogOpen, setFailedDialogOpen] = useState(false)

  const { data: deployments, isLoading, error, refetch } = useDeployments()
  const deleteDeployment = useDeleteDeployment()
  const cancelDeployment = useCancelDeployment()
  const restartDeployment = useRestartDeployment()
  const stopDeployment = useStopDeployment()
  const startDeployment = useStartDeployment()
  const cleanupDeployments = useCleanupDeployments()

  // Separate active and failed deployments
  const { activeDeployments, failedDeployments } = useMemo(() => {
    if (!deployments || !Array.isArray(deployments)) {
      return { activeDeployments: [], failedDeployments: [] }
    }

    const active: Deployment[] = []
    const failed: Deployment[] = []

    deployments.forEach((deployment) => {
      // Defensive: Ensure deployment has required fields
      if (!deployment || !deployment.status) return

      if (deployment.status === 'failed' || deployment.status === 'rolled_back') {
        failed.push(deployment)
      } else {
        active.push(deployment)
      }
    })

    return { activeDeployments: active, failedDeployments: failed }
  }, [deployments])

  // Group active deployments by app (recipe)
  const groupedApps = useMemo(() => {
    if (!activeDeployments || !Array.isArray(activeDeployments)) return []

    const groups = new Map<string, GroupedApp>()

    activeDeployments.forEach((deployment) => {
      // Defensive: Ensure deployment has required fields
      if (!deployment || !deployment.recipe_slug) return

      const key = deployment.recipe_slug
      if (!groups.has(key)) {
        groups.set(key, {
          recipe_slug: deployment.recipe_slug,
          recipe_name: deployment.recipe_name,
          deployments: [],
          statusCounts: {
            running: 0,
            failed: 0,
            deploying: 0,
            stopped: 0,
            rolling_back: 0,
            total: 0,
          },
          devices: [],
        })
      }

      const group = groups.get(key)!
      group.deployments.push(deployment)
      group.statusCounts.total++

      // Count statuses comprehensively
      // Note: failed and rolled_back deployments are already filtered out in activeDeployments
      if (deployment.status === 'running') {
        group.statusCounts.running++
      } else if (IN_PROGRESS_STATUSES.includes(deployment.status)) {
        group.statusCounts.deploying++
      } else if (deployment.status === 'stopped') {
        group.statusCounts.stopped++
      } else if (deployment.status === 'rolling_back') {
        group.statusCounts.rolling_back++
      }
      // Any other status (edge cases) will just contribute to total count

      // Track unique devices - defensive null check
      const deviceExists = group.devices.some(d => d.id === deployment.device_id)
      if (!deviceExists && deployment.device_id) {
        group.devices.push({
          id: deployment.device_id,
          name: deployment.device?.name || `Device ${deployment.device_id.slice(0, 8)}`,
        })
      }
    })

    return Array.from(groups.values())
  }, [activeDeployments])

  // Filter apps based on selected status
  const filteredApps = useMemo(() => {
    if (selectedStatus === 'all') return groupedApps

    return groupedApps.filter((app) => {
      if (selectedStatus === 'deploying_composite') {
        return app.statusCounts.deploying > 0
      }
      return app.deployments.some(d => d.status === selectedStatus)
    })
  }, [groupedApps, selectedStatus])

  const handleDelete = async () => {
    if (!deploymentToDelete) return

    try {
      await deleteDeployment.mutateAsync(deploymentToDelete.id)
      toast.success('Deployment deleted successfully')
      setDeploymentToDelete(null)
      // Close failed dialog if it was open (deletion from failed deployments list)
      setFailedDialogOpen(false)
    } catch (error) {
      toast.error('Failed to delete deployment', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
      setDeploymentToDelete(null)
    }
  }

  const handleCancel = async (deployment: Deployment) => {
    setOperationStates(prev => ({ ...prev, [`cancel-${deployment.id}`]: true }))
    try {
      await cancelDeployment.mutateAsync(deployment.id)
      toast.success('Deployment cancelled successfully')
    } catch (error) {
      toast.error('Failed to cancel deployment', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setOperationStates(prev => ({ ...prev, [`cancel-${deployment.id}`]: false }))
    }
  }

  const handleRestart = async (deployment: Deployment) => {
    setOperationStates(prev => ({ ...prev, [`restart-${deployment.id}`]: true }))
    try {
      await restartDeployment.mutateAsync(deployment.id)
      toast.success(`${deployment.recipe_name} restarted successfully`)
    } catch (error) {
      toast.error('Failed to restart app', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setOperationStates(prev => ({ ...prev, [`restart-${deployment.id}`]: false }))
    }
  }

  const handleStop = async (deployment: Deployment) => {
    setOperationStates(prev => ({ ...prev, [`stop-${deployment.id}`]: true }))
    try {
      await stopDeployment.mutateAsync(deployment.id)
      toast.success(`${deployment.recipe_name} stopped successfully`)
    } catch (error) {
      toast.error('Failed to stop app', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setOperationStates(prev => ({ ...prev, [`stop-${deployment.id}`]: false }))
    }
  }

  const handleStart = async (deployment: Deployment) => {
    setOperationStates(prev => ({ ...prev, [`start-${deployment.id}`]: true }))
    try {
      await startDeployment.mutateAsync(deployment.id)
      toast.success(`${deployment.recipe_name} started successfully`)
    } catch (error) {
      toast.error('Failed to start app', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    } finally {
      setOperationStates(prev => ({ ...prev, [`start-${deployment.id}`]: false }))
    }
  }

  const handleTroubleshoot = (deploymentId: string) => {
    setSelectedDeploymentId(deploymentId)
    setTroubleshootDialogOpen(true)
  }

  const toggleAppExpansion = (recipeSlug: string) => {
    setExpandedApps(prev => {
      const newSet = new Set(prev)
      if (newSet.has(recipeSlug)) {
        newSet.delete(recipeSlug)
      } else {
        newSet.add(recipeSlug)
      }
      return newSet
    })
  }

  const isCancellable = (status: DeploymentStatus) => {
    return IN_PROGRESS_STATUSES.includes(status)
  }

  const canRestart = (status: DeploymentStatus) => {
    return status === 'running' || status === 'stopped'
  }

  const canStop = (status: DeploymentStatus) => {
    return status === 'running'
  }

  const canStart = (status: DeploymentStatus) => {
    return status === 'stopped'
  }

  // Component for deployment actions
  const DeploymentActions = ({ deployment }: { deployment: Deployment }) => {
    const { data: accessURLs } = useDeploymentAccessURLs(deployment.id, {
      enabled: deployment.status === 'running'
    })
    const hasMultipleURLs = accessURLs && accessURLs.length > 1
    const primaryURL = accessURLs && accessURLs.length > 0 ? accessURLs[0] : null

    return (
      <>
        {/* Open App Button */}
        {deployment.status === 'running' && primaryURL && (
          hasMultipleURLs ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button size="sm" variant="default" className="gap-1.5" aria-label="Open application">
                  <ExternalLink className="w-3.5 h-3.5" />
                  Open
                  <ChevronDown className="w-3 h-3" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                {accessURLs?.map((urlInfo, idx) => (
                  <DropdownMenuItem
                    key={idx}
                    onClick={() => {
                      const newWindow = window.open(urlInfo.url, '_blank')
                      if (!newWindow) {
                        toast.error('Popup blocked - please allow popups for this site')
                      }
                    }}
                  >
                    <ExternalLink className="w-4 h-4 mr-2" />
                    {urlInfo.description}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          ) : (
            <Button
              size="sm"
              variant="default"
              onClick={() => {
                const newWindow = window.open(primaryURL.url, '_blank')
                if (!newWindow) {
                  toast.error('Popup blocked - please allow popups for this site')
                }
              }}
              className="gap-1.5"
              aria-label="Open application"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              Open
            </Button>
          )
        )}

        {/* Restart Button */}
        {canRestart(deployment.status) && (
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleRestart(deployment)}
            disabled={operationStates[`restart-${deployment.id}`]}
            title="Restart app"
            aria-label="Restart application"
          >
            {operationStates[`restart-${deployment.id}`] ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <RotateCw className="w-3.5 h-3.5" />
            )}
          </Button>
        )}

        {/* Stop/Start Button */}
        {canStop(deployment.status) ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleStop(deployment)}
            disabled={operationStates[`stop-${deployment.id}`]}
            title="Stop app"
            aria-label="Stop application"
          >
            {operationStates[`stop-${deployment.id}`] ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Square className="w-3.5 h-3.5" />
            )}
          </Button>
        ) : canStart(deployment.status) ? (
          <Button
            size="sm"
            variant="outline"
            onClick={() => handleStart(deployment)}
            disabled={operationStates[`start-${deployment.id}`]}
            title="Start app"
            aria-label="Start application"
          >
            {operationStates[`start-${deployment.id}`] ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Play className="w-3.5 h-3.5" />
            )}
          </Button>
        ) : null}

        {/* Troubleshoot Button */}
        <Button
          size="sm"
          variant="outline"
          onClick={() => handleTroubleshoot(deployment.id)}
          title="Troubleshoot"
          aria-label="Troubleshoot deployment"
        >
          <Wrench className="w-3.5 h-3.5" />
        </Button>

        {/* More Actions Dropdown */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" aria-label="More actions">
              <MoreVertical className="w-4 h-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {deployment.deployment_logs && (
              <DropdownMenuItem
                onClick={() => {
                  setSelectedLogs({
                    name: deployment.recipe_name,
                    logs: deployment.deployment_logs || '',
                  })
                  setLogsDialogOpen(true)
                }}
              >
                <FileText className="w-4 h-4 mr-2" />
                View Logs
              </DropdownMenuItem>
            )}
            {isCancellable(deployment.status) && (
              <DropdownMenuItem
                onClick={() => handleCancel(deployment)}
                disabled={operationStates[`cancel-${deployment.id}`]}
                className="text-orange-600"
              >
                {operationStates[`cancel-${deployment.id}`] ? (
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                ) : (
                  <StopCircle className="w-4 h-4 mr-2" />
                )}
                Cancel Deployment
              </DropdownMenuItem>
            )}
            <DropdownMenuItem
              onClick={() => setDeploymentToDelete(deployment)}
              className="text-red-600"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </>
    )
  }

  const getStatusBadge = (status: DeploymentStatus) => {
    const config: Record<DeploymentStatus, { label: string; variant: 'default' | 'secondary' | 'success' | 'warning' | 'danger'; icon: React.ReactNode }> = {
      validating: { label: 'Validating', variant: 'secondary', icon: <Loader2 className="w-3 h-3 animate-spin" /> },
      preparing: { label: 'Preparing', variant: 'secondary', icon: <Loader2 className="w-3 h-3 animate-spin" /> },
      deploying: { label: 'Deploying', variant: 'default', icon: <Loader2 className="w-3 h-3 animate-spin" /> },
      configuring: { label: 'Configuring', variant: 'default', icon: <Loader2 className="w-3 h-3 animate-spin" /> },
      health_check: { label: 'Health Check', variant: 'default', icon: <Clock className="w-3 h-3" /> },
      running: { label: 'Running', variant: 'success', icon: <CheckCircle className="w-3 h-3" /> },
      stopped: { label: 'Stopped', variant: 'warning', icon: <AlertCircle className="w-3 h-3" /> },
      failed: { label: 'Failed', variant: 'danger', icon: <XCircle className="w-3 h-3" /> },
      rolling_back: { label: 'Rolling Back', variant: 'danger', icon: <Loader2 className="w-3 h-3 animate-spin" /> },
      rolled_back: { label: 'Rolled Back', variant: 'warning', icon: <AlertCircle className="w-3 h-3" /> },
    }

    const { label, variant, icon } = config[status] || { label: status, variant: 'secondary' as const, icon: null }

    return (
      <Badge variant={variant} className="flex items-center gap-1.5 w-fit">
        {icon}
        <span>{label}</span>
      </Badge>
    )
  }

  const handleCleanupFailed = async () => {
    try {
      const result = await cleanupDeployments.mutateAsync('failed')
      toast.success(`Successfully deleted ${result.deleted_count} failed deployment${result.deleted_count === 1 ? '' : 's'}`)
      setFailedDialogOpen(false)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Unknown error'
      // Check if this is a partial failure (some deleted, some failed)
      if (errorMsg.includes('deleted') && errorMsg.includes('errors:')) {
        toast.warning('Some deployments could not be deleted', {
          description: errorMsg,
          duration: 8000, // Longer duration for detailed message
        })
        setFailedDialogOpen(false) // Close dialog on partial success
      } else {
        toast.error('Failed to cleanup deployments', {
          description: errorMsg,
        })
      }
    }
  }

  const getStatusCounts = () => {
    if (!deployments) return { all: 0, running: 0, deploying: 0, failed: 0 }

    return {
      all: groupedApps.length, // Count unique apps, not deployments
      running: groupedApps.filter(app => app.statusCounts.running > 0).length,
      deploying: groupedApps.filter(app => app.statusCounts.deploying > 0).length,
      failed: groupedApps.filter(app => app.statusCounts.failed > 0).length,
    }
  }

  const counts = getStatusCounts()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Loader2 className="w-8 h-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <XCircle className="w-12 h-12 text-red-600 mx-auto mb-4" />
          <h2 className="text-xl font-semibold mb-2">Error loading apps</h2>
          <p className="text-muted-foreground">{(error as Error).message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8 flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent">
              Apps
            </h1>
            <p className="mt-2 text-muted-foreground">
              Single pane of glass for your entire homelab ecosystem
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate('/marketplace')}
              className="flex items-center gap-2"
            >
              <Package className="w-4 h-4" />
              Browse Apps
            </Button>
            <button
              onClick={() => refetch()}
              className="p-2 rounded-lg bg-card border border-border hover:bg-accent transition-colors"
              title="Refresh apps"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Resource Overview - Placeholder for future metrics */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                <Server className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Devices</p>
                <p className="text-xl font-bold">{deployments ? new Set(deployments.map(d => d.device_id)).size : 0}</p>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center">
                <Cpu className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">CPU Usage</p>
                <p className="text-xl font-bold">-</p>
                <p className="text-xs text-muted-foreground">Coming soon</p>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-orange-100 dark:bg-orange-900/30 flex items-center justify-center">
                <Activity className="w-5 h-5 text-orange-600 dark:text-orange-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Memory</p>
                <p className="text-xl font-bold">-</p>
                <p className="text-xs text-muted-foreground">Coming soon</p>
              </div>
            </div>
          </div>

          <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                <HardDrive className="w-5 h-5 text-green-600 dark:text-green-400" />
              </div>
              <div>
                <p className="text-sm font-medium text-muted-foreground">Storage</p>
                <p className="text-xl font-bold">-</p>
                <p className="text-xs text-muted-foreground">Coming soon</p>
              </div>
            </div>
          </div>
        </div>

        {/* Stats Summary */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mb-6">
          <button
            onClick={() => setSelectedStatus('all')}
            className={`bg-card border rounded-lg p-4 shadow-sm text-left transition-colors ${
              selectedStatus === 'all' ? 'border-primary ring-2 ring-primary/20' : 'border-border hover:border-primary/50'
            }`}
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                <span className="text-lg font-bold text-primary">{counts.all}</span>
              </div>
              <div>
                <p className="text-sm font-medium">Total Apps</p>
                <p className="text-xs text-muted-foreground">Across all devices</p>
              </div>
            </div>
          </button>

          <button
            onClick={() => setSelectedStatus('running')}
            className={`bg-card border rounded-lg p-4 shadow-sm text-left transition-colors ${
              selectedStatus === 'running' ? 'border-green-500 ring-2 ring-green-500/20' : 'border-border hover:border-green-500/50'
            }`}
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-green-100 flex items-center justify-center">
                <span className="text-lg font-bold text-green-700">{counts.running}</span>
              </div>
              <div>
                <p className="text-sm font-medium">Running</p>
                <p className="text-xs text-muted-foreground">Active apps</p>
              </div>
            </div>
          </button>

          <button
            onClick={() => setSelectedStatus('deploying_composite')}
            className={`bg-card border rounded-lg p-4 shadow-sm text-left transition-colors ${
              selectedStatus === 'deploying_composite' ? 'border-blue-500 ring-2 ring-blue-500/20' : 'border-border hover:border-blue-500/50'
            }`}
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-blue-100 flex items-center justify-center">
                <span className="text-lg font-bold text-blue-700">{counts.deploying}</span>
              </div>
              <div>
                <p className="text-sm font-medium">Deploying</p>
                <p className="text-xs text-muted-foreground">Installing now</p>
              </div>
            </div>
          </button>

          <button
            onClick={() => setSelectedStatus('failed')}
            className={`bg-card border rounded-lg p-4 shadow-sm text-left transition-colors ${
              selectedStatus === 'failed' ? 'border-red-500 ring-2 ring-red-500/20' : 'border-border hover:border-red-500/50'
            }`}
          >
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-red-100 flex items-center justify-center">
                <span className="text-lg font-bold text-red-700">{counts.failed}</span>
              </div>
              <div>
                <p className="text-sm font-medium">Failed</p>
                <p className="text-xs text-muted-foreground">Needs attention</p>
              </div>
            </div>
          </button>
        </div>

        {/* Failed Deployments Banner */}
        {failedDeployments.length > 0 && (
          <div className="mb-6 p-4 bg-orange-50 dark:bg-orange-950/20 border border-orange-200 dark:border-orange-800 rounded-lg">
            <div className="flex items-start gap-3">
              <AlertCircle className="w-5 h-5 text-orange-600 dark:text-orange-400 mt-0.5 flex-shrink-0" />
              <div className="flex-1">
                <h4 className="font-semibold text-orange-900 dark:text-orange-100 mb-1">
                  {failedDeployments.length} failed deployment{failedDeployments.length === 1 ? '' : 's'} hidden from view
                </h4>
                <p className="text-sm text-orange-700 dark:text-orange-300">
                  These deployments failed during installation and are not part of your active apps.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setFailedDialogOpen(true)}
                className="border-orange-300 dark:border-orange-700 hover:bg-orange-100 dark:hover:bg-orange-900/30"
              >
                View & Clean Up
              </Button>
            </div>
          </div>
        )}

        {/* Apps List */}
        {!filteredApps || filteredApps.length === 0 ? (
          <div className="bg-card border border-border rounded-lg shadow-sm text-center py-12">
            <Package className="w-16 h-16 mx-auto mb-4 text-muted-foreground" />
            <p className="text-muted-foreground mb-2">
              {selectedStatus === 'all'
                ? 'No apps running yet. Deploy an app from the marketplace to get started!'
                : `No ${STATUS_LABELS[selectedStatus]} found.`
              }
            </p>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => navigate('/marketplace')}
            >
              Browse Marketplace
            </Button>
          </div>
        ) : (
          <div className="space-y-4">
            {filteredApps.map((app) => {
              // Defensive: Ensure app has required fields
              if (!app || !app.recipe_slug) return null

              return (
              <Collapsible
                key={app.recipe_slug}
                open={expandedApps.has(app.recipe_slug)}
                onOpenChange={() => toggleAppExpansion(app.recipe_slug)}
              >
                <div className="bg-card border border-border rounded-lg shadow-sm overflow-hidden">
                  {/* App Header */}
                  <CollapsibleTrigger asChild>
                    <div className="p-4 hover:bg-accent/50 cursor-pointer transition-colors">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-4 flex-1">
                          <div className="flex items-center gap-2">
                            {expandedApps.has(app.recipe_slug) ? (
                              <ChevronDown className="w-5 h-5 text-muted-foreground" />
                            ) : (
                              <ChevronRight className="w-5 h-5 text-muted-foreground" />
                            )}
                            <Package className="w-8 h-8 text-primary" />
                          </div>

                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-1">
                              <h3 className="text-lg font-semibold">{app.recipe_name}</h3>
                              <Badge variant="secondary">{app.statusCounts.total} {app.statusCounts.total === 1 ? 'deployment' : 'deployments'}</Badge>
                            </div>

                            <div className="flex items-center gap-4 text-sm text-muted-foreground">
                              <div className="flex items-center gap-2">
                                <Server className="w-4 h-4" />
                                <span>{app.devices.length} {app.devices.length === 1 ? 'device' : 'devices'}</span>
                              </div>
                              {app.statusCounts.running > 0 && (
                                <Badge variant="success" className="text-xs">
                                  {app.statusCounts.running} running
                                </Badge>
                              )}
                              {app.statusCounts.deploying > 0 && (
                                <Badge variant="default" className="text-xs">
                                  {app.statusCounts.deploying} deploying
                                </Badge>
                              )}
                              {app.statusCounts.stopped > 0 && (
                                <Badge variant="warning" className="text-xs">
                                  {app.statusCounts.stopped} stopped
                                </Badge>
                              )}
                              {app.statusCounts.rolling_back > 0 && (
                                <Badge variant="danger" className="text-xs">
                                  {app.statusCounts.rolling_back} rolling back
                                </Badge>
                              )}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={(e) => {
                              e.stopPropagation()
                              navigate(`/marketplace?app=${app.recipe_slug}`)
                            }}
                          >
                            Deploy to new device
                          </Button>
                        </div>
                      </div>
                    </div>
                  </CollapsibleTrigger>

                  {/* Deployments Table */}
                  <CollapsibleContent>
                    <div className="border-t border-border">
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Device</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Deployed</TableHead>
                            <TableHead className="text-right">Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {app.deployments.map((deployment) => {
                            // Defensive: Ensure deployment has required fields
                            if (!deployment || !deployment.id) return null

                            return (
                            <TableRow key={deployment.id}>
                              <TableCell>
                                <div>
                                  <Badge
                                    variant="secondary"
                                    className="text-xs flex items-center gap-1 w-fit cursor-pointer hover:bg-secondary/80"
                                    onClick={() => navigate(`/devices/${deployment.device_id}`)}
                                  >
                                    <Server className="w-3 h-3" />
                                    {deployment.device?.name || `Device ${deployment.device_id.slice(0, 8)}`}
                                  </Badge>
                                  {deployment.error_details && (
                                    <div className="text-xs text-red-600 mt-1 flex items-start gap-1">
                                      <AlertCircle className="w-3 h-3 mt-0.5 flex-shrink-0" />
                                      <span>{deployment.error_details}</span>
                                    </div>
                                  )}
                                </div>
                              </TableCell>
                              <TableCell>{getStatusBadge(deployment.status)}</TableCell>
                              <TableCell className="text-sm text-muted-foreground">
                                {deployment.deployed_at
                                  ? new Date(deployment.deployed_at).toLocaleString()
                                  : 'Pending'
                                }
                              </TableCell>
                              <TableCell className="text-right">
                                <div className="flex items-center justify-end gap-1">
                                  <DeploymentActions deployment={deployment} />
                                </div>
                              </TableCell>
                            </TableRow>
                            )
                          })}
                        </TableBody>
                      </Table>
                    </div>
                  </CollapsibleContent>
                </div>
              </Collapsible>
              )
            })}
          </div>
        )}
      </div>

      {/* Delete Confirmation Dialog */}
      <Dialog open={!!deploymentToDelete} onOpenChange={() => setDeploymentToDelete(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Deployment</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete the deployment of{' '}
              <span className="font-semibold">{deploymentToDelete?.recipe_name}</span>
              {' '}on device{' '}
              <span className="font-semibold">
                {deploymentToDelete?.device?.name || `Device ${deploymentToDelete?.device_id?.slice(0, 8)}`}
              </span>?
              This will stop and remove all containers. Volumes will be preserved.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeploymentToDelete(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteDeployment.isPending}
            >
              {deleteDeployment.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  Deleting...
                </>
              ) : (
                'Delete'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Deployment Logs Dialog */}
      <Dialog open={logsDialogOpen} onOpenChange={setLogsDialogOpen}>
        <DialogContent className="max-w-4xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Deployment Logs - {selectedLogs.name}</DialogTitle>
            <DialogDescription>
              Complete deployment logs for this application
            </DialogDescription>
          </DialogHeader>
          <div className="mt-4">
            <LogViewer logs={selectedLogs.logs || ''} className="max-h-[60vh]" />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Troubleshoot Dialog */}
      <TroubleshootDialog
        open={troubleshootDialogOpen}
        onOpenChange={setTroubleshootDialogOpen}
        deploymentId={selectedDeploymentId}
      />

      {/* Failed Deployments Dialog */}
      <Dialog
        open={failedDialogOpen}
        onOpenChange={(open) => {
          // Prevent closing dialog while cleanup is in progress
          if (!cleanupDeployments.isPending && !deleteDeployment.isPending) {
            setFailedDialogOpen(open)
          }
        }}
      >
        <DialogContent className="max-w-4xl max-h-[80vh]">
          <DialogHeader>
            <DialogTitle>Failed Deployments</DialogTitle>
            <DialogDescription>
              These deployments failed during installation. You can safely delete them to clean up your workspace.
            </DialogDescription>
          </DialogHeader>

          <div className="mt-4">
            {failedDeployments.length === 0 ? (
              <div className="text-center py-8">
                <CheckCircle className="w-12 h-12 text-green-600 mx-auto mb-3" />
                <p className="text-sm text-muted-foreground">No failed deployments</p>
              </div>
            ) : (
              <div className="space-y-3 max-h-96 overflow-y-auto">
                {failedDeployments.map((deployment) => {
                  // Defensive: Ensure deployment has required fields
                  if (!deployment || !deployment.id) return null

                  return (
                  <div
                    key={deployment.id}
                    className="p-4 bg-card border border-border rounded-lg"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          <h4 className="font-medium">{deployment.recipe_name}</h4>
                          <Badge variant="danger" className="text-xs">
                            Failed
                          </Badge>
                        </div>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                          <Server className="w-3 h-3" />
                          <span>{deployment.device?.name || `Device ${deployment.device_id.slice(0, 8)}`}</span>
                        </div>
                        {deployment.error_details && (
                          <div className="text-xs text-red-600 mt-2 flex items-start gap-1">
                            <AlertCircle className="w-3 h-3 mt-0.5 flex-shrink-0" />
                            <span>{deployment.error_details}</span>
                          </div>
                        )}
                        {deployment.deployed_at && (
                          <p className="text-xs text-muted-foreground mt-1">
                            Failed: {new Date(deployment.deployed_at).toLocaleString()}
                          </p>
                        )}
                      </div>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                          // Set deployment to delete but keep dialog open until deletion completes
                          setDeploymentToDelete(deployment)
                        }}
                        disabled={deleteDeployment.isPending}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50"
                      >
                        {deleteDeployment.isPending && deploymentToDelete?.id === deployment.id ? (
                          <Loader2 className="w-4 h-4 animate-spin" />
                        ) : (
                          <Trash2 className="w-4 h-4" />
                        )}
                      </Button>
                    </div>
                  </div>
                  )
                })}
              </div>
            )}
          </div>

          <DialogFooter className="flex items-center justify-between">
            <Button
              variant="outline"
              onClick={() => setFailedDialogOpen(false)}
              disabled={cleanupDeployments.isPending || deleteDeployment.isPending}
            >
              Close
            </Button>
            {failedDeployments.length > 0 && (
              <Button
                variant="destructive"
                onClick={handleCleanupFailed}
                disabled={cleanupDeployments.isPending || deleteDeployment.isPending}
              >
                {cleanupDeployments.isPending ? (
                  <>
                    <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 className="w-4 h-4 mr-2" />
                    Delete All Failed
                  </>
                )}
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// Troubleshoot Dialog Component
function TroubleshootDialog({
  open,
  onOpenChange,
  deploymentId,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  deploymentId: string | null
}) {
  const { data: troubleshootData, isLoading } = useTroubleshootDeployment(deploymentId || '', {
    enabled: open && !!deploymentId
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Troubleshoot Deployment</DialogTitle>
          <DialogDescription>
            Diagnostic information to help identify and resolve issues
          </DialogDescription>
        </DialogHeader>
        <div className="mt-4 space-y-4">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="w-8 h-8 animate-spin text-primary" />
            </div>
          ) : troubleshootData ? (
            <>
              {/* Firewall Status */}
              {troubleshootData.firewall_status && (
                <div className="border rounded-lg p-4">
                  <h3 className="font-semibold mb-2 flex items-center gap-2">
                    <Wrench className="w-4 h-4" />
                    Firewall Status
                  </h3>
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Enabled:</span>
                      <Badge variant={troubleshootData.firewall_status.enabled ? 'success' : 'secondary'}>
                        {troubleshootData.firewall_status.enabled ? 'Yes' : 'No'}
                      </Badge>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Type:</span>
                      <span>{troubleshootData.firewall_status.type || 'N/A'}</span>
                    </div>
                    {troubleshootData.firewall_status.open_ports && (
                      <div>
                        <span className="text-muted-foreground">Open Ports:</span>
                        <div className="mt-1 flex flex-wrap gap-1">
                          {troubleshootData.firewall_status.open_ports.map((port: number) => (
                            <Badge key={port} variant="secondary">{port}</Badge>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Required Ports */}
              {troubleshootData.required_ports && troubleshootData.required_ports.length > 0 && (
                <div className="border rounded-lg p-4">
                  <h3 className="font-semibold mb-2">Required Ports</h3>
                  <div className="flex flex-wrap gap-2">
                    {troubleshootData.required_ports.map((portSpec: any, idx: number) => (
                      <Badge key={idx} variant="secondary">
                        {portSpec.port}/{portSpec.protocol}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Container Status */}
              {troubleshootData.container_status && (
                <div className="border rounded-lg p-4">
                  <h3 className="font-semibold mb-2">Container Status</h3>
                  <pre className="text-xs bg-muted p-3 rounded overflow-x-auto">
                    {JSON.stringify(troubleshootData.container_status, null, 2)}
                  </pre>
                </div>
              )}

              {/* Recent Logs */}
              {troubleshootData.recent_logs && (
                <div className="border rounded-lg p-4">
                  <h3 className="font-semibold mb-2">Recent Logs</h3>
                  <LogViewer logs={troubleshootData.recent_logs} className="max-h-60" />
                </div>
              )}

              {/* Recommendations */}
              {troubleshootData.recommendations && troubleshootData.recommendations.length > 0 && (
                <div className="border rounded-lg p-4 bg-blue-50 dark:bg-blue-950/20">
                  <h3 className="font-semibold mb-2 flex items-center gap-2">
                    <AlertCircle className="w-4 h-4 text-blue-600" />
                    Recommendations
                  </h3>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    {troubleshootData.recommendations.map((rec: string, idx: number) => (
                      <li key={idx}>{rec}</li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          ) : (
            <p className="text-muted-foreground text-center py-8">No troubleshooting data available</p>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
