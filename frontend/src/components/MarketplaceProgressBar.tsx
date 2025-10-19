import { Progress } from './ui/progress'

interface MarketplaceProgressBarProps {
  deployed: number
  total: number
  percentage: number
}

export function MarketplaceProgressBar({ deployed, total, percentage }: MarketplaceProgressBarProps) {
  const getMessage = () => {
    if (percentage === 0) return "Start your self-hosting journey"
    if (percentage < 25) return "Great start! Keep going"
    if (percentage < 50) return "You're building momentum"
    if (percentage < 75) return "More than halfway there!"
    if (percentage < 100) return "Almost completely self-hosted!"
    return "Amazing! You've escaped all SaaS services!"
  }

  const getEmoji = () => {
    if (percentage === 0) return "🚀"
    if (percentage < 50) return "💪"
    if (percentage < 100) return "🎯"
    return "🎉"
  }

  return (
    <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-primary/5 via-primary/10 to-primary/5 p-6 border border-primary/10 shadow-lg shadow-primary/5">
      {/* Background Pattern */}
      <div className="absolute inset-0 bg-grid-white/5 [mask-image:radial-gradient(white,transparent_70%)]" />

      <div className="relative space-y-4">
        {/* Stats */}
        <div className="flex items-center justify-between">
          <div className="flex items-baseline gap-3">
            <span className="text-4xl font-black bg-gradient-to-r from-primary to-primary/60 bg-clip-text text-transparent">
              {deployed}
            </span>
            <span className="text-lg font-medium text-muted-foreground">
              of {total} services replaced
            </span>
          </div>
          <div className="text-4xl">{getEmoji()}</div>
        </div>

        {/* Progress Bar */}
        <div className="space-y-2">
          <Progress value={percentage} className="h-4 bg-muted/50" />
          <div className="flex items-center justify-between text-sm">
            <p className="font-medium text-muted-foreground">{getMessage()}</p>
            <span className="font-bold text-primary">{percentage}%</span>
          </div>
        </div>
      </div>
    </div>
  )
}
