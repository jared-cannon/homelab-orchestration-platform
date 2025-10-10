import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useDeployments, useDeleteDeployment, useCancelDeployment } from '../api/hooks'
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

type FilterStatus = DeploymentStatus | 'all' | 'deploying_composite'

const IN_PROGRESS_STATUSES: DeploymentStatus[] = ['validating', 'preparing', 'deploying', 'configuring', 'health_check']

const STATUS_LABELS: Record<FilterStatus, string> = {
  all: 'deployments',
  running: 'running deployments',
  deploying_composite: 'in-progress deployments',
  failed: 'failed deployments',
  validating: 'validating deployments',
  preparing: 'preparing deployments',
  deploying: 'deploying deployments',
  configuring: 'configuring deployments',
  health_check: 'deployments in health check',
  stopped: 'stopped deployments',
  rolling_back: 'rolling back deployments',
  rolled_back: 'rolled back deployments',
}

export function DeploymentsPage() {
  const navigate = useNavigate()
  const [selectedStatus, setSelectedStatus] = useState<FilterStatus>('all')
  const [deploymentToDelete, setDeploymentToDelete] = useState<Deployment | null>(null)
  const [logsDialogOpen, setLogsDialogOpen] = useState(false)
  const [selectedLogs, setSelectedLogs] = useState({ name: '', logs: '' })

  const { data: deployments, isLoading, error, refetch } = useDeployments()
  const deleteDeployment = useDeleteDeployment()
  const cancelDeployment = useCancelDeployment()

  const filteredDeployments = deployments?.filter((d) => {
    if (selectedStatus === 'all') return true
    if (selectedStatus === 'deploying_composite') return IN_PROGRESS_STATUSES.includes(d.status)
    return d.status === selectedStatus
  })

  const handleDelete = async () => {
    if (!deploymentToDelete) return

    try {
      await deleteDeployment.mutateAsync(deploymentToDelete.id)
      toast.success('Deployment deleted successfully')
      setDeploymentToDelete(null)
    } catch (error) {
      toast.error('Failed to delete deployment', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  const handleCancel = async (deployment: Deployment) => {
    try {
      await cancelDeployment.mutateAsync(deployment.id)
      toast.success('Deployment cancelled successfully')
    } catch (error) {
      toast.error('Failed to cancel deployment', {
        description: error instanceof Error ? error.message : 'Unknown error',
      })
    }
  }

  const isCancellable = (status: DeploymentStatus) => {
    return IN_PROGRESS_STATUSES.includes(status)
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

  const getStatusCounts = () => {
    if (!deployments) return { all: 0, running: 0, deploying: 0, failed: 0 }

    return {
      all: deployments.length,
      running: deployments.filter(d => d.status === 'running').length,
      deploying: deployments.filter(d => IN_PROGRESS_STATUSES.includes(d.status)).length,
      failed: deployments.filter(d => d.status === 'failed').length,
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
          <h2 className="text-xl font-semibold mb-2">Error loading deployments</h2>
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
              Deployments
            </h1>
            <p className="mt-2 text-muted-foreground">
              View and manage all application deployments
            </p>
          </div>
          <button
            onClick={() => refetch()}
            className="p-2 rounded-lg bg-card border border-border hover:bg-accent transition-colors"
            title="Refresh deployments"
          >
            <RefreshCw className="w-4 h-4" />
          </button>
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
                <p className="text-sm font-medium">Total</p>
                <p className="text-xs text-muted-foreground">All deployments</p>
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
                <p className="text-xs text-muted-foreground">Active deployments</p>
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
                <p className="text-xs text-muted-foreground">In progress</p>
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

        {/* Deployments Table */}
        <div className="bg-card border border-border rounded-lg shadow-sm">
          {!filteredDeployments || filteredDeployments.length === 0 ? (
            <div className="text-center py-12">
              <p className="text-muted-foreground">
                {selectedStatus === 'all'
                  ? 'No deployments yet. Deploy an app from the marketplace to get started!'
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
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Application</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Deployed</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredDeployments.map((deployment) => (
                  <TableRow key={deployment.id}>
                    <TableCell className="font-medium">
                      <div>
                        <div>{deployment.recipe_name}</div>
                        <div className="text-xs text-muted-foreground">{deployment.recipe_slug}</div>
                        {deployment.error_details && (
                          <div className="text-xs text-red-600 mt-1 flex items-start gap-1">
                            <AlertCircle className="w-3 h-3 mt-0.5 flex-shrink-0" />
                            <span>{deployment.error_details}</span>
                          </div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <button
                        onClick={() => navigate(`/devices/${deployment.device_id}`)}
                        className="text-primary hover:underline flex items-center gap-1"
                      >
                        {deployment.device?.name || `Device ${deployment.device_id.slice(0, 8)}`}
                        <ExternalLink className="w-3 h-3" />
                      </button>
                    </TableCell>
                    <TableCell>{getStatusBadge(deployment.status)}</TableCell>
                    <TableCell>
                      {deployment.deployed_at
                        ? new Date(deployment.deployed_at).toLocaleString()
                        : 'Pending'
                      }
                    </TableCell>
                    <TableCell className="text-right">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <MoreVertical className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {deployment.deployment_logs && (
                            <DropdownMenuItem
                              onClick={() => {
                                setSelectedLogs({
                                  name: deployment.recipe_name,
                                  logs: deployment.deployment_logs,
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
                              disabled={cancelDeployment.isPending}
                              className="text-orange-600"
                            >
                              {cancelDeployment.isPending ? (
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
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </div>
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
          <div className="mt-4 max-h-[60vh] overflow-auto">
            <pre className="text-xs font-mono bg-muted p-4 rounded-lg whitespace-pre-wrap break-words">
              {selectedLogs.logs || 'No logs available'}
            </pre>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setLogsDialogOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
