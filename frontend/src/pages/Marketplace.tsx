import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useRecipes, useRecipeCategories } from '../api/hooks'
import { RecipeCard } from '../components/RecipeCard'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Search } from 'lucide-react'
import type { Recipe } from '../api/types'
import { DeploymentWizard } from '../components/DeploymentWizard'

export function MarketplacePage() {
  const navigate = useNavigate()
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [searchQuery, setSearchQuery] = useState('')
  const [deployingRecipe, setDeployingRecipe] = useState<Recipe | null>(null)

  const { data: recipes, isLoading, error } = useRecipes(selectedCategory)
  const { data: categories } = useRecipeCategories()

  // Filter recipes by search query
  const filteredRecipes = recipes?.filter((recipe) => {
    const query = searchQuery.toLowerCase()
    return (
      recipe.name.toLowerCase().includes(query) ||
      recipe.tagline.toLowerCase().includes(query) ||
      recipe.description.toLowerCase().includes(query)
    )
  })

  const handleDeploy = (recipe: Recipe) => {
    setDeployingRecipe(recipe)
  }

  const handleRecipeClick = (recipe: Recipe) => {
    navigate(`/marketplace/${recipe.slug}`)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading marketplace...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-600">
          Error loading marketplace: {(error as Error).message}
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent mb-2">
            Marketplace
          </h1>
          <p className="text-muted-foreground">
            Deploy applications to your homelab with one click
          </p>
        </div>

        {/* Search and Filters */}
        <div className="mb-6 space-y-4">
          {/* Search Bar */}
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
            <Input
              type="text"
              placeholder="Search apps..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>

          {/* Category Filters */}
          <div className="flex gap-2 flex-wrap">
            <Button
              variant={selectedCategory === '' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setSelectedCategory('')}
            >
              All
            </Button>
            {categories?.map((category) => (
              <Button
                key={category}
                variant={selectedCategory === category ? 'default' : 'outline'}
                size="sm"
                onClick={() => setSelectedCategory(category)}
                className="capitalize"
              >
                {category}
              </Button>
            ))}
          </div>
        </div>

        {/* Recipes Grid */}
        {filteredRecipes && filteredRecipes.length > 0 ? (
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {filteredRecipes.map((recipe) => (
              <RecipeCard
                key={recipe.slug}
                recipe={recipe}
                onDeploy={handleDeploy}
                onClick={handleRecipeClick}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-muted-foreground">
              {searchQuery
                ? `No apps found matching "${searchQuery}"`
                : 'No apps available in this category'}
            </p>
          </div>
        )}
      </div>

      {/* Deployment Wizard Modal */}
      {deployingRecipe && (
        <DeploymentWizard
          recipe={deployingRecipe}
          open={!!deployingRecipe}
          onOpenChange={(open) => !open && setDeployingRecipe(null)}
        />
      )}
    </div>
  )
}
