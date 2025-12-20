import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { Search, Filter, Package, PlusCircle } from 'lucide-react'
import { useDeployments, useDeleteDeployment } from '../api/hooks'
import { DeployedAppCard } from '../components/DeployedAppCard'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Badge } from '../components/ui/badge'
import type { Deployment, DeploymentStatus } from '../api/types'

export function MyAppsPage() {
  const navigate = useNavigate()
  const { data: deployments, isLoading } = useDeployments()
  const deleteDeploymentMutation = useDeleteDeployment()

  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<DeploymentStatus | 'all'>('all')

  // Filter and search deployments
  const filteredDeployments = useMemo(() => {
    if (!deployments) return []

    let filtered = deployments

    // Filter by status
    if (statusFilter !== 'all') {
      filtered = filtered.filter(d => d.status === statusFilter)
    }

    // Search by name or slug
    if (searchQuery) {
      const query = searchQuery.toLowerCase()
      filtered = filtered.filter(d =>
        d.recipe_name?.toLowerCase().includes(query) ||
        d.recipe_slug?.toLowerCase().includes(query)
      )
    }

    // Sort by deployed_at (most recent first)
    return filtered.sort((a, b) => {
      if (!a.deployed_at) return 1
      if (!b.deployed_at) return -1
      return new Date(b.deployed_at).getTime() - new Date(a.deployed_at).getTime()
    })
  }, [deployments, statusFilter, searchQuery])

  // Count deployments by status
  const stats = useMemo(() => {
    if (!deployments) return { total: 0, running: 0, failed: 0, deploying: 0 }

    return {
      total: deployments.length,
      running: deployments.filter(d => d.status === 'running').length,
      failed: deployments.filter(d => d.status === 'failed').length,
      deploying: deployments.filter(d =>
        ['preparing', 'deploying', 'configuring', 'health_check'].includes(d.status)
      ).length,
    }
  }, [deployments])

  const handleDeleteDeployment = async (deployment: Deployment) => {
    if (!confirm(`Are you sure you want to delete ${deployment.recipe_name || deployment.recipe_slug}? This action cannot be undone.`)) {
      return
    }

    try {
      await deleteDeploymentMutation.mutateAsync(deployment.id)
      toast.success('Deployment deleted successfully')
    } catch (error) {
      toast.error('Failed to delete deployment', {
        description: (error as Error).message
      })
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading your apps...</div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex justify-between items-start mb-6">
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent flex items-center gap-3">
                <Package className="w-8 h-8 text-primary" />
                My Apps
              </h1>
              <p className="mt-2 text-muted-foreground">
                Manage all your deployed applications
              </p>
            </div>
            <Button
              onClick={() => navigate('/apps')}
              className="font-semibold shadow-lg shadow-primary/20"
            >
              <PlusCircle className="w-4 h-4 mr-2" />
              Deploy New App
            </Button>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="text-2xl font-bold text-foreground">{stats.total}</div>
              <div className="text-sm text-muted-foreground">Total Apps</div>
            </div>
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="text-2xl font-bold text-green-600 dark:text-green-400">{stats.running}</div>
              <div className="text-sm text-muted-foreground">Running</div>
            </div>
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{stats.deploying}</div>
              <div className="text-sm text-muted-foreground">Deploying</div>
            </div>
            <div className="bg-card border border-border rounded-lg p-4 shadow-sm">
              <div className="text-2xl font-bold text-red-600 dark:text-red-400">{stats.failed}</div>
              <div className="text-sm text-muted-foreground">Failed</div>
            </div>
          </div>

          {/* Search and Filter */}
          <div className="flex flex-col sm:flex-row gap-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
              <Input
                type="text"
                placeholder="Search apps by name..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <div className="flex gap-2 items-center">
              <Filter className="w-4 h-4 text-muted-foreground" />
              <div className="flex gap-2 flex-wrap">
                <Badge
                  variant={statusFilter === 'all' ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setStatusFilter('all')}
                >
                  All
                </Badge>
                <Badge
                  variant={statusFilter === 'running' ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setStatusFilter('running')}
                >
                  Running
                </Badge>
                <Badge
                  variant={statusFilter === 'deploying' ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setStatusFilter('deploying')}
                >
                  Deploying
                </Badge>
                <Badge
                  variant={statusFilter === 'failed' ? 'default' : 'outline'}
                  className="cursor-pointer"
                  onClick={() => setStatusFilter('failed')}
                >
                  Failed
                </Badge>
              </div>
            </div>
          </div>
        </div>

        {/* Apps Grid */}
        {filteredDeployments.length === 0 ? (
          <div className="text-center py-16">
            <Package className="w-16 h-16 text-muted-foreground mx-auto mb-4 opacity-50" />
            <h3 className="text-xl font-semibold text-muted-foreground mb-2">
              {searchQuery || statusFilter !== 'all' ? 'No apps found' : 'No apps deployed yet'}
            </h3>
            <p className="text-sm text-muted-foreground mb-6">
              {searchQuery || statusFilter !== 'all'
                ? 'Try adjusting your search or filter'
                : 'Deploy your first app from the marketplace'}
            </p>
            {!searchQuery && statusFilter === 'all' && (
              <Button onClick={() => navigate('/apps')} variant="outline">
                <PlusCircle className="w-4 h-4 mr-2" />
                Browse Apps
              </Button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4">
            {filteredDeployments.map((deployment) => (
              <DeployedAppCard
                key={deployment.id}
                deployment={deployment}
                onOpenApp={(url) => window.open(url, '_blank')}
                onManage={(dep) => navigate(`/deployments/${dep.id}`)}
                onDelete={handleDeleteDeployment}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
