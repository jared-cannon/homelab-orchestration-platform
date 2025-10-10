import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useRecipe } from '../api/hooks'
import { Button } from '../components/ui/button'
import { Badge } from '../components/ui/badge'
import { Card } from '../components/ui/card'
import { ArrowLeft, Cpu, Database, HardDrive, CheckCircle } from 'lucide-react'
import { DeploymentWizard } from '../components/DeploymentWizard'

export function RecipeDetailPage() {
  const { slug } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const [showWizard, setShowWizard] = useState(false)

  const { data: recipe, isLoading, error } = useRecipe(slug!)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading recipe...</div>
      </div>
    )
  }

  if (error || !recipe) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <p className="text-red-600 mb-4">
            Error loading recipe: {(error as Error)?.message || 'Recipe not found'}
          </p>
          <Button onClick={() => navigate('/marketplace')}>
            Back to Marketplace
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Back Button */}
        <Button
          variant="ghost"
          onClick={() => navigate('/marketplace')}
          className="mb-6"
        >
          <ArrowLeft className="w-4 h-4 mr-2" />
          Back to Marketplace
        </Button>

        {/* Header */}
        <div className="mb-8">
          <div className="flex items-start gap-6 mb-4">
            {recipe.icon_url ? (
              <img
                src={recipe.icon_url}
                alt={`${recipe.name} icon`}
                className="w-24 h-24 rounded-lg object-cover flex-shrink-0"
                onError={(e) => {
                  e.currentTarget.style.display = 'none'
                }}
              />
            ) : (
              <div className="w-24 h-24 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                <span className="text-4xl font-bold text-primary">
                  {recipe.name.charAt(0)}
                </span>
              </div>
            )}

            <div className="flex-1">
              <h1 className="text-3xl font-bold mb-2">{recipe.name}</h1>
              <p className="text-xl text-muted-foreground mb-3">
                {recipe.tagline}
              </p>
              <Badge variant="secondary" className="capitalize">
                {recipe.category}
              </Badge>
            </div>
          </div>

          <Button
            size="lg"
            onClick={() => setShowWizard(true)}
            className="w-full sm:w-auto"
          >
            Deploy {recipe.name}
          </Button>
        </div>

        {/* Description */}
        <Card className="p-6 mb-6">
          <h2 className="text-xl font-semibold mb-3">About</h2>
          <p className="text-muted-foreground whitespace-pre-line">
            {recipe.description}
          </p>
        </Card>

        {/* Resource Requirements */}
        <Card className="p-6 mb-6">
          <h2 className="text-xl font-semibold mb-4">Resource Requirements</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="flex items-start gap-3">
              <Cpu className="w-5 h-5 text-primary mt-0.5" />
              <div>
                <div className="font-medium">RAM</div>
                <div className="text-sm text-muted-foreground">
                  Minimum:{' '}
                  {recipe.resources.min_ram_mb >= 1024
                    ? `${(recipe.resources.min_ram_mb / 1024).toFixed(1)} GB`
                    : `${recipe.resources.min_ram_mb} MB`}
                </div>
                <div className="text-sm text-muted-foreground">
                  Recommended:{' '}
                  {recipe.resources.recommended_ram_mb >= 1024
                    ? `${(recipe.resources.recommended_ram_mb / 1024).toFixed(1)} GB`
                    : `${recipe.resources.recommended_ram_mb} MB`}
                </div>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <HardDrive className="w-5 h-5 text-primary mt-0.5" />
              <div>
                <div className="font-medium">Storage</div>
                <div className="text-sm text-muted-foreground">
                  Minimum: {recipe.resources.min_storage_gb} GB
                </div>
                <div className="text-sm text-muted-foreground">
                  Recommended: {recipe.resources.recommended_storage_gb} GB
                </div>
              </div>
            </div>

            <div className="flex items-start gap-3">
              <Database className="w-5 h-5 text-primary mt-0.5" />
              <div>
                <div className="font-medium">CPU Cores</div>
                <div className="text-sm text-muted-foreground">
                  {recipe.resources.cpu_cores} {recipe.resources.cpu_cores === 1 ? 'core' : 'cores'}
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* Configuration Options */}
        {recipe.config_options && recipe.config_options.length > 0 && (
          <Card className="p-6 mb-6">
            <h2 className="text-xl font-semibold mb-4">Configuration Options</h2>
            <div className="space-y-3">
              {recipe.config_options.map((option) => (
                <div key={option.name} className="border-l-2 border-primary/20 pl-4">
                  <div className="font-medium">
                    {option.label}
                    {option.required && (
                      <span className="text-red-500 ml-1">*</span>
                    )}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {option.description}
                  </div>
                  <div className="text-xs text-muted-foreground mt-1">
                    Type: {option.type} • Default: {String(option.default)}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        )}

        {/* Post-Deploy Instructions */}
        {recipe.post_deploy_instructions && (
          <Card className="p-6">
            <h2 className="text-xl font-semibold mb-3 flex items-center gap-2">
              <CheckCircle className="w-5 h-5 text-green-500" />
              After Deployment
            </h2>
            <div className="text-sm text-muted-foreground whitespace-pre-line">
              {recipe.post_deploy_instructions}
            </div>
          </Card>
        )}
      </div>

      {/* Deployment Wizard Modal */}
      {showWizard && (
        <DeploymentWizard
          recipe={recipe}
          open={showWizard}
          onOpenChange={setShowWizard}
        />
      )}
    </div>
  )
}
