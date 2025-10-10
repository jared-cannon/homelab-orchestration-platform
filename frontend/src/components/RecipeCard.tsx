import { Card } from './ui/card'
import { Button } from './ui/button'
import { Badge } from './ui/badge'
import type { Recipe } from '../api/types'
import { Database, Cpu } from 'lucide-react'

interface RecipeCardProps {
  recipe: Recipe
  onDeploy?: (recipe: Recipe) => void
  onClick?: (recipe: Recipe) => void
}

export function RecipeCard({ recipe, onDeploy, onClick }: RecipeCardProps) {
  const handleDeploy = (e: React.MouseEvent) => {
    e.stopPropagation() // Prevent card click when deploy button is clicked
    onDeploy?.(recipe)
  }

  const handleCardClick = () => {
    onClick?.(recipe)
  }

  return (
    <Card
      className="overflow-hidden hover:shadow-lg transition-shadow duration-200 cursor-pointer"
      onClick={handleCardClick}
    >
      <div className="p-6 space-y-4">
        {/* Icon and Title */}
        <div className="flex items-start gap-4">
          {recipe.icon_url ? (
            <img
              src={recipe.icon_url}
              alt={`${recipe.name} icon`}
              className="w-16 h-16 rounded-lg object-cover flex-shrink-0"
              onError={(e) => {
                // Fallback if image fails to load
                e.currentTarget.style.display = 'none'
              }}
            />
          ) : (
            <div className="w-16 h-16 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
              <span className="text-2xl font-bold text-primary">
                {recipe.name.charAt(0)}
              </span>
            </div>
          )}

          <div className="flex-1 min-w-0">
            <h3 className="font-semibold text-lg truncate">{recipe.name}</h3>
            <p className="text-sm text-muted-foreground line-clamp-1">
              {recipe.tagline}
            </p>
          </div>
        </div>

        {/* Category Badge */}
        <div>
          <Badge variant="secondary" className="capitalize">
            {recipe.category}
          </Badge>
        </div>

        {/* Resource Requirements */}
        <div className="space-y-2 text-sm">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Cpu className="w-4 h-4" />
            <span>
              {recipe.resources.min_ram_mb >= 1024
                ? `${(recipe.resources.min_ram_mb / 1024).toFixed(1)} GB RAM`
                : `${recipe.resources.min_ram_mb} MB RAM`}
            </span>
          </div>
          <div className="flex items-center gap-2 text-muted-foreground">
            <Database className="w-4 h-4" />
            <span>{recipe.resources.min_storage_gb} GB Storage</span>
          </div>
        </div>

        {/* Deploy Button */}
        <Button onClick={handleDeploy} className="w-full">
          Deploy
        </Button>
      </div>
    </Card>
  )
}
