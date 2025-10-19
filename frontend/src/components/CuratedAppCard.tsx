import { Card } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import { Check, ExternalLink, Settings, Clock, Cpu } from 'lucide-react'
import type { CuratedRecipe, DeploymentInfo } from '../api/types'
import { cn } from '../lib/utils'

interface CuratedAppCardProps {
  recipe: CuratedRecipe
  deploymentInfo?: DeploymentInfo
  onDeploy: (recipe: CuratedRecipe) => void
  onCompare: (recipe: CuratedRecipe) => void
  onManage?: (recipe: CuratedRecipe) => void
  onOpenApp?: (url: string) => void
}

export function CuratedAppCard({
  recipe,
  deploymentInfo,
  onDeploy,
  onCompare,
  onManage,
  onOpenApp,
}: CuratedAppCardProps) {
  const isRunning = deploymentInfo?.status === 'running'

  const formatRAM = (mb?: number) => {
    if (!mb) return ''
    return mb >= 1024 ? `${(mb / 1024).toFixed(0)}GB` : `${mb}MB`
  }

  return (
    <Card
      className={cn(
        'group relative overflow-hidden transition-all duration-300 border-0',
        isRunning
          ? 'bg-gradient-to-br from-muted/40 to-muted/20 shadow-sm'
          : 'bg-card hover:shadow-xl hover:shadow-primary/5 shadow-md'
      )}
    >
      <div className="flex items-stretch relative">
        {/* Left: Icon with gradient background */}
        <div className="flex-shrink-0 px-8 py-6 flex items-center bg-gradient-to-br from-primary/5 to-primary/10">
          {recipe.icon_url ? (
            <img
              src={recipe.icon_url}
              alt={recipe.name}
              className="w-20 h-20 rounded-2xl object-cover ring-4 ring-background/50 shadow-lg"
            />
          ) : (
            <div className="w-20 h-20 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/30 flex items-center justify-center text-3xl shadow-lg">
              📦
            </div>
          )}
        </div>

        {/* Middle: Info */}
        <div className="flex-1 min-w-0 px-6 py-6">
          {/* Title and Status */}
          <div className="flex items-center gap-3 mb-3">
            <h3 className="font-bold text-xl tracking-tight">{recipe.name}</h3>
            {isRunning && (
              <Badge
                variant="success"
                className="flex items-center gap-1.5 px-2.5 py-0.5 bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/20"
              >
                <Check className="w-3.5 h-3.5" />
                <span className="text-xs font-medium">Running</span>
              </Badge>
            )}
          </div>

          {/* SaaS Replacements - Most Important */}
          {recipe.saas_replacements && recipe.saas_replacements.length > 0 && (
            <div className="flex items-center gap-2 mb-3">
              <span className="text-sm font-medium text-muted-foreground">Replaces</span>
              <div className="flex flex-wrap gap-2">
                {recipe.saas_replacements.map((replacement, idx) => (
                  <Badge
                    key={idx}
                    variant="secondary"
                    className="text-xs font-medium px-2.5 py-1 bg-primary/10 text-primary border-primary/20 hover:bg-primary/20 transition-colors"
                  >
                    {replacement.name}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {/* Tagline */}
          <p className="text-sm text-muted-foreground/80 leading-relaxed mb-4 line-clamp-2">
            {recipe.tagline}
          </p>

          {/* Running Info or Quick Specs */}
          {isRunning ? (
            <div className="flex items-center gap-3 text-sm">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-muted/50">
                <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                <span className="text-muted-foreground">
                  on <span className="font-medium text-foreground">{deploymentInfo.device_name}</span>
                </span>
              </div>
              {deploymentInfo.access_url && (
                <button
                  onClick={() => onOpenApp?.(deploymentInfo.access_url!)}
                  className="text-primary hover:text-primary/80 flex items-center gap-1.5 font-medium transition-colors"
                >
                  View App
                  <ExternalLink className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          ) : (
            <div className="flex items-center gap-4">
              {recipe.difficulty_level && (
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-muted/50">
                  <span className={cn(
                    "text-xs font-semibold uppercase tracking-wide",
                    recipe.difficulty_level === 'beginner' && 'text-green-600 dark:text-green-400',
                    recipe.difficulty_level === 'intermediate' && 'text-yellow-600 dark:text-yellow-400',
                    recipe.difficulty_level === 'advanced' && 'text-red-600 dark:text-red-400'
                  )}>
                    {recipe.difficulty_level}
                  </span>
                </div>
              )}
              {recipe.setup_time_minutes && (
                <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Clock className="w-4 h-4" />
                  <span className="font-medium">{recipe.setup_time_minutes} min</span>
                </div>
              )}
              {recipe.resources?.min_ram_mb && (
                <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                  <Cpu className="w-4 h-4" />
                  <span className="font-medium">{formatRAM(recipe.resources.min_ram_mb)}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Right: Actions */}
        <div className="flex-shrink-0 px-6 py-6 bg-gradient-to-bl from-muted/30 to-muted/10 flex flex-col items-center justify-center gap-3 min-w-[160px] border-l border-border/50">
          {/* Action Buttons */}
          <div className="flex flex-col gap-2.5 w-full">
            {isRunning ? (
              <>
                {deploymentInfo.access_url && (
                  <Button
                    onClick={() => onOpenApp?.(deploymentInfo.access_url!)}
                    className="w-full font-semibold shadow-sm hover:shadow-md transition-all"
                  >
                    <ExternalLink className="w-4 h-4 mr-2" />
                    Open App
                  </Button>
                )}
                <Button
                  onClick={() => onManage?.(recipe)}
                  variant="outline"
                  className="w-full font-medium hover:bg-muted/50 transition-all"
                >
                  <Settings className="w-4 h-4 mr-2" />
                  Manage
                </Button>
              </>
            ) : (
              <>
                <Button
                  onClick={() => onDeploy(recipe)}
                  className="w-full font-semibold shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
                >
                  Deploy Now
                </Button>
                <Button
                  onClick={() => onCompare(recipe)}
                  variant="outline"
                  className="w-full font-medium hover:bg-muted/50 transition-all"
                >
                  Compare →
                </Button>
              </>
            )}
          </div>
        </div>
      </div>
    </Card>
  )
}
