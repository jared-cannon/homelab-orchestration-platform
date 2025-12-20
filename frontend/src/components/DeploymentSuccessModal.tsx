import { useNavigate } from 'react-router-dom'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog'
import { Button } from './ui/button'
import { CheckCircle, ExternalLink, Package } from 'lucide-react'

interface DeploymentSuccessModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  deployment: {
    id: string
    recipe_name: string
    recipe_slug: string
    url?: string
    hostname?: string
    use_traefik?: boolean
  }
  recipeIcon?: string
  postDeployInstructions?: string
}

export function DeploymentSuccessModal({
  open,
  onOpenChange,
  deployment,
  recipeIcon,
  postDeployInstructions,
}: DeploymentSuccessModalProps) {
  const navigate = useNavigate()

  const accessUrl = deployment.url
  const displayName = deployment.recipe_name || deployment.recipe_slug

  const handleOpenApp = () => {
    if (accessUrl) {
      window.open(accessUrl, '_blank')
    }
  }

  const handleGoToMyApps = () => {
    onOpenChange(false)
    navigate('/my-apps')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <div className="flex items-center justify-center mb-4">
            <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/20 flex items-center justify-center">
              <CheckCircle className="w-10 h-10 text-green-600 dark:text-green-400" />
            </div>
          </div>
          <DialogTitle className="text-center text-2xl">
            Deployment Successful!
          </DialogTitle>
          <DialogDescription className="text-center">
            {displayName} is now running and ready to use
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* App Icon and Name */}
          <div className="flex items-center justify-center gap-3 p-4 bg-muted/50 rounded-lg border border-border">
            {recipeIcon ? (
              <img
                src={recipeIcon}
                alt={displayName}
                className="w-12 h-12 rounded-lg object-cover shadow-sm"
              />
            ) : (
              <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-primary/20 to-primary/30 flex items-center justify-center text-xl">
                📦
              </div>
            )}
            <div className="flex-1">
              <h3 className="font-semibold text-lg">{displayName}</h3>
              {deployment.hostname && (
                <p className="text-sm text-muted-foreground">
                  {deployment.hostname}
                </p>
              )}
            </div>
          </div>

          {/* Access URL */}
          {accessUrl && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-muted-foreground">
                Access URL
              </label>
              <div className="flex items-center gap-2">
                <div className="flex-1 px-3 py-2 bg-card border border-border rounded-lg font-mono text-sm truncate">
                  {accessUrl}
                </div>
                <Button
                  onClick={handleOpenApp}
                  size="sm"
                  className="flex-shrink-0"
                >
                  <ExternalLink className="w-4 h-4 mr-1.5" />
                  Open
                </Button>
              </div>
              {deployment.use_traefik && (
                <p className="text-xs text-muted-foreground">
                  🔒 Traefik will automatically provide HTTPS with Let's Encrypt
                </p>
              )}
            </div>
          )}

          {/* Post-Deploy Instructions */}
          {postDeployInstructions && (
            <div className="p-4 bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900/30 rounded-lg">
              <h4 className="font-medium text-blue-900 dark:text-blue-400 mb-2 flex items-center gap-2">
                📋 Next Steps
              </h4>
              <div className="text-sm text-blue-800 dark:text-blue-300 whitespace-pre-wrap">
                {postDeployInstructions}
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="flex-col sm:flex-row gap-2">
          <Button
            variant="outline"
            onClick={handleGoToMyApps}
            className="w-full sm:w-auto"
          >
            <Package className="w-4 h-4 mr-2" />
            View My Apps
          </Button>
          {accessUrl ? (
            <Button
              onClick={handleOpenApp}
              className="w-full sm:w-auto"
            >
              <ExternalLink className="w-4 h-4 mr-2" />
              Open {displayName}
            </Button>
          ) : (
            <Button
              onClick={() => onOpenChange(false)}
              className="w-full sm:w-auto"
            >
              Close
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
