import { useState } from 'react'
import { useCuratedMarketplace } from '../api/hooks'
import { MarketplaceProgressBar } from '../components/MarketplaceProgressBar'
import { CuratedAppCard } from '../components/CuratedAppCard'
import { SaaSComparisonModal } from '../components/SaaSComparisonModal'
import { DeploymentWizard } from '../components/DeploymentWizard'
import { Badge } from '../components/ui/badge'
import type { CuratedRecipe } from '../api/types'

type FilterType = 'all' | 'not_installed' | 'running'
type DifficultyFilter = 'all' | 'beginner' | 'intermediate' | 'advanced'
type SortType = 'not_installed_first' | 'popularity' | 'setup_time'

export function AppsPage() {
  const { data, isLoading, error } = useCuratedMarketplace()

  const [filterType, setFilterType] = useState<FilterType>('all')
  const [difficultyFilter, setDifficultyFilter] = useState<DifficultyFilter>('all')
  const [sortType, setSortType] = useState<SortType>('not_installed_first')

  const [selectedRecipeForComparison, setSelectedRecipeForComparison] = useState<CuratedRecipe | null>(null)
  const [selectedRecipeForDeployment, setSelectedRecipeForDeployment] = useState<CuratedRecipe | null>(null)

  const handleDeploy = (recipe: CuratedRecipe) => {
    // DeploymentWizard now handles device selection and dependency checking
    setSelectedRecipeForDeployment(recipe)
  }

  const handleCompare = (recipe: CuratedRecipe) => {
    setSelectedRecipeForComparison(recipe)
  }

  const handleManage = (recipe: CuratedRecipe) => {
    // Navigate to deployment management
    console.log('Manage:', recipe)
  }

  const handleOpenApp = (url: string) => {
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading apps...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-600">
          Error loading apps: {(error as Error).message}
        </div>
      </div>
    )
  }

  if (!data) {
    return null
  }

  // Filter and sort recipes
  let filteredRecipes = data.recipes.filter((recipe) => {
    // Filter by installation status
    const isRunning = data.user_deployments[recipe.slug]?.status === 'running'
    if (filterType === 'not_installed' && isRunning) return false
    if (filterType === 'running' && !isRunning) return false

    // Filter by difficulty
    if (difficultyFilter !== 'all' && recipe.difficulty_level !== difficultyFilter) {
      return false
    }

    return true
  })

  // Sort recipes
  filteredRecipes = [...filteredRecipes].sort((a, b) => {
    if (sortType === 'not_installed_first') {
      const aRunning = data.user_deployments[a.slug]?.status === 'running'
      const bRunning = data.user_deployments[b.slug]?.status === 'running'
      if (aRunning !== bRunning) return aRunning ? 1 : -1
      // Secondary sort by name alphabetically
      return a.name.localeCompare(b.name)
    } else if (sortType === 'popularity') {
      // Sort alphabetically by name for now (can be replaced with actual popularity metrics later)
      return a.name.localeCompare(b.name)
    } else if (sortType === 'setup_time') {
      return (a.setup_time_minutes || 999) - (b.setup_time_minutes || 999)
    }
    return 0
  })

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-muted/5 to-muted/10">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        {/* Header - Centered content within aligned container */}
        <div className="mb-10 max-w-5xl mx-auto">
          <div className="text-center max-w-3xl mx-auto">
            <h1 className="text-5xl font-black tracking-tight mb-4 bg-gradient-to-r from-foreground via-foreground to-foreground/60 bg-clip-text text-transparent">
              Escape SaaS, Self-Host Everything
            </h1>
            <p className="text-lg text-muted-foreground/90 leading-relaxed">
              Replace expensive subscriptions with open-source alternatives you control
            </p>
          </div>
        </div>

        {/* Progress Bar - Aligned with cards */}
        <div className="mb-8 max-w-5xl mx-auto">
          <MarketplaceProgressBar
            deployed={data.stats.deployed}
            total={data.stats.total_curated}
            percentage={data.stats.percentage}
          />
        </div>

        {/* Modern Filters - Aligned with cards */}
        <div className="mb-8 max-w-5xl mx-auto">
          <div className="flex items-center justify-between gap-4 flex-wrap bg-card/50 backdrop-blur-sm rounded-2xl p-4 border border-border/30 shadow-sm">
            {/* Status Filter - Modern Segmented Control */}
            <div className="inline-flex rounded-xl bg-muted/50 p-1 gap-1">
              <button
                onClick={() => setFilterType('all')}
                className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all ${
                  filterType === 'all'
                    ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                    : 'text-muted-foreground hover:text-foreground hover:bg-background/50'
                }`}
              >
                All Apps
              </button>
              <button
                onClick={() => setFilterType('not_installed')}
                className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all ${
                  filterType === 'not_installed'
                    ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                    : 'text-muted-foreground hover:text-foreground hover:bg-background/50'
                }`}
              >
                Not Installed
              </button>
              <button
                onClick={() => setFilterType('running')}
                className={`px-4 py-2 text-sm font-semibold rounded-lg transition-all ${
                  filterType === 'running'
                    ? 'bg-primary text-primary-foreground shadow-md shadow-primary/20'
                    : 'text-muted-foreground hover:text-foreground hover:bg-background/50'
                }`}
              >
                Running
              </button>
            </div>

            {/* Right Side Filters */}
            <div className="flex items-center gap-4">
              <span className="text-sm font-medium text-muted-foreground/80">
                {filteredRecipes.length} {filteredRecipes.length === 1 ? 'app' : 'apps'}
              </span>

              {/* Difficulty Filter */}
              {difficultyFilter !== 'all' && (
                <Badge variant="secondary" className="gap-1.5 px-3 py-1">
                  <span className="capitalize">{difficultyFilter}</span>
                  <button
                    onClick={() => setDifficultyFilter('all')}
                    className="hover:text-foreground ml-1"
                  >
                    ×
                  </button>
                </Badge>
              )}

              {/* Custom Sort Dropdown */}
              <div className="relative">
                <select
                  value={sortType}
                  onChange={(e) => setSortType(e.target.value as SortType)}
                  className="appearance-none text-sm font-medium border-0 rounded-lg px-4 py-2 pr-10 bg-muted/50 hover:bg-muted transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/50"
                >
                  <option value="not_installed_first">Not Installed First</option>
                  <option value="popularity">Most Popular</option>
                  <option value="setup_time">Quickest Setup</option>
                </select>
                <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center">
                  <svg className="h-4 w-4 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Single Column Feed */}
        {filteredRecipes.length > 0 ? (
          <div className="max-w-5xl mx-auto space-y-5">
            {filteredRecipes.map((recipe) => (
              <CuratedAppCard
                key={recipe.slug}
                recipe={recipe}
                deploymentInfo={data.user_deployments[recipe.slug]}
                onDeploy={handleDeploy}
                onCompare={handleCompare}
                onManage={handleManage}
                onOpenApp={handleOpenApp}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-20 max-w-5xl mx-auto">
            <div className="text-6xl mb-4">🔍</div>
            <p className="text-lg font-medium text-muted-foreground/80">
              No apps match the selected filters
            </p>
          </div>
        )}
      </div>

      {/* SaaS Comparison Modal */}
      <SaaSComparisonModal
        open={!!selectedRecipeForComparison}
        onOpenChange={(open) => !open && setSelectedRecipeForComparison(null)}
        recipe={selectedRecipeForComparison}
        onDeploy={() => {
          if (selectedRecipeForComparison) {
            setSelectedRecipeForDeployment(selectedRecipeForComparison)
            setSelectedRecipeForComparison(null)
          }
        }}
      />

      {/* Deployment Wizard (now includes dependency checking) */}
      {selectedRecipeForDeployment && (
        <DeploymentWizard
          recipe={selectedRecipeForDeployment}
          open={!!selectedRecipeForDeployment}
          onOpenChange={(open) => !open && setSelectedRecipeForDeployment(null)}
        />
      )}
    </div>
  )
}
