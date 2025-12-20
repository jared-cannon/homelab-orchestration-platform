import { Card } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import {
  ExternalLink,
  Settings,
  Trash2,
  AlertCircle,
  CheckCircle2,
  Clock,
  Server,
  Link as LinkIcon
} from 'lucide-react'
import type { Deployment } from '../api/types'
import { cn } from '../lib/utils'
import { format } from 'date-fns'

interface DeployedAppCardProps {
  deployment: Deployment & {
    hostname?: string
    url?: string
    use_traefik?: boolean
  }
  recipeIcon?: string
  recipeName?: string
  onOpenApp?: (url: string) => void
  onManage?: (deployment: Deployment) => void
  onDelete?: (deployment: Deployment) => void
}

export function DeployedAppCard({
  deployment,
  recipeIcon,
  recipeName,
  onOpenApp,
  onManage,
  onDelete,
}: DeployedAppCardProps) {
  const isRunning = deployment.status === 'running'
  const isFailed = deployment.status === 'failed'
  const isDeploying = ['preparing', 'deploying', 'configuring', 'health_check'].includes(deployment.status)

  // Get the display name (use recipeName if provided, otherwise use recipe_name from deployment)
  const displayName = recipeName || deployment.recipe_name || deployment.recipe_slug

  // Get access URL (prioritize new url field, fallback to domain-based URL)
  const accessUrl = deployment.url || (deployment.domain ? `http://${deployment.domain}` : undefined)

  // Format deployed date
  const deployedDate = deployment.deployed_at
    ? format(new Date(deployment.deployed_at), 'MMM d, yyyy h:mm a')
    : null

  // Get status badge variant and icon
  const getStatusBadge = () => {
    if (isRunning) {
      return {
        variant: 'success' as const,
        icon: <CheckCircle2 className="w-3.5 h-3.5" />,
        label: 'Running',
        className: 'bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/20'
      }
    }
    if (isFailed) {
      return {
        variant: 'destructive' as const,
        icon: <AlertCircle className="w-3.5 h-3.5" />,
        label: 'Failed',
        className: 'bg-red-500/10 text-red-700 dark:text-red-400 border-red-500/20'
      }
    }
    if (isDeploying) {
      return {
        variant: 'secondary' as const,
        icon: <Clock className="w-3.5 h-3.5 animate-spin" />,
        label: 'Deploying',
        className: 'bg-blue-500/10 text-blue-700 dark:text-blue-400 border-blue-500/20'
      }
    }
    return {
      variant: 'outline' as const,
      icon: null,
      label: deployment.status,
      className: ''
    }
  }

  const statusBadge = getStatusBadge()

  return (
    <Card
      className={cn(
        'group overflow-hidden transition-all duration-300',
        isRunning
          ? 'bg-gradient-to-br from-card to-muted/20 shadow-md hover:shadow-lg'
          : 'bg-card shadow-sm hover:shadow-md',
        isFailed && 'border-red-200 dark:border-red-900/30'
      )}
    >
      <div className="flex items-stretch">
        {/* Left: Icon */}
        <div className="flex-shrink-0 px-6 py-5 flex items-center bg-gradient-to-br from-primary/5 to-primary/10 border-r border-border/50">
          {recipeIcon ? (
            <img
              src={recipeIcon}
              alt={displayName}
              className="w-16 h-16 rounded-xl object-cover ring-2 ring-background/50 shadow-sm"
            />
          ) : (
            <div className="w-16 h-16 rounded-xl bg-gradient-to-br from-primary/20 to-primary/30 flex items-center justify-center text-2xl shadow-sm">
              📦
            </div>
          )}
        </div>

        {/* Middle: Info */}
        <div className="flex-1 min-w-0 px-6 py-5">
          {/* Title and Status */}
          <div className="flex items-center gap-3 mb-2">
            <h3 className="font-bold text-lg tracking-tight">{displayName}</h3>
            <Badge
              variant={statusBadge.variant}
              className={cn(
                'flex items-center gap-1.5 px-2.5 py-0.5',
                statusBadge.className
              )}
            >
              {statusBadge.icon}
              <span className="text-xs font-medium">{statusBadge.label}</span>
            </Badge>
          </div>

          {/* Recipe Slug (if different from name) */}
          {deployment.recipe_slug !== displayName && (
            <div className="text-sm text-muted-foreground mb-2">
              {deployment.recipe_slug}
            </div>
          )}

          {/* Deployment Info */}
          <div className="flex flex-col gap-2">
            {/* Device Info */}
            {deployment.device && (
              <div className="flex items-center gap-2 text-sm">
                <Server className="w-4 h-4 text-muted-foreground" />
                <span className="text-muted-foreground">
                  Deployed on <span className="font-medium text-foreground">{deployment.device.name}</span>
                </span>
              </div>
            )}

            {/* Access URL */}
            {isRunning && accessUrl && (
              <div className="flex items-center gap-2 text-sm">
                <LinkIcon className="w-4 h-4 text-muted-foreground" />
                <a
                  href={accessUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:text-primary/80 font-medium transition-colors flex items-center gap-1.5 group"
                  onClick={(e) => {
                    if (onOpenApp) {
                      e.preventDefault()
                      onOpenApp(accessUrl)
                    }
                  }}
                >
                  {deployment.hostname || accessUrl}
                  <ExternalLink className="w-3.5 h-3.5 opacity-0 group-hover:opacity-100 transition-opacity" />
                </a>
              </div>
            )}

            {/* Traefik Badge */}
            {deployment.use_traefik && (
              <div className="flex items-center gap-2">
                <Badge
                  variant="outline"
                  className="text-xs px-2 py-0.5 bg-purple-500/5 text-purple-700 dark:text-purple-400 border-purple-500/20"
                >
                  🔀 Traefik Routing
                </Badge>
              </div>
            )}

            {/* Deployed Date */}
            {deployedDate && (
              <div className="text-xs text-muted-foreground">
                Deployed {deployedDate}
              </div>
            )}

            {/* Error Message */}
            {isFailed && deployment.error_details && (
              <div className="mt-2 text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900/30 rounded-md px-3 py-2">
                <div className="flex items-start gap-2">
                  <AlertCircle className="w-4 h-4 flex-shrink-0 mt-0.5" />
                  <span className="text-xs leading-relaxed">{deployment.error_details}</span>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right: Actions */}
        <div className="flex-shrink-0 px-6 py-5 bg-gradient-to-bl from-muted/20 to-muted/10 flex flex-col items-center justify-center gap-2.5 min-w-[140px] border-l border-border/50">
          {isRunning && accessUrl && (
            <Button
              onClick={() => onOpenApp?.(accessUrl)}
              className="w-full font-semibold shadow-sm hover:shadow-md transition-all"
              size="sm"
            >
              <ExternalLink className="w-4 h-4 mr-2" />
              Open App
            </Button>
          )}

          {isRunning && (
            <Button
              onClick={() => onManage?.(deployment)}
              variant="outline"
              className="w-full font-medium hover:bg-muted/50 transition-all"
              size="sm"
            >
              <Settings className="w-4 h-4 mr-2" />
              Manage
            </Button>
          )}

          {(isFailed || isRunning) && (
            <Button
              onClick={() => onDelete?.(deployment)}
              variant="destructive"
              className="w-full font-medium transition-all"
              size="sm"
            >
              <Trash2 className="w-4 h-4 mr-2" />
              Delete
            </Button>
          )}

          {isDeploying && (
            <div className="text-xs text-center text-muted-foreground px-2">
              <Clock className="w-4 h-4 mx-auto mb-1 animate-spin" />
              Please wait...
            </div>
          )}
        </div>
      </div>
    </Card>
  )
}
