import { cn } from '../../lib/utils'

interface GuideSectionProps {
  title: string
  children: React.ReactNode
  className?: string
}

export function GuideSection({ title, children, className }: GuideSectionProps) {
  return (
    <div className={cn('space-y-3', className)}>
      <h4 className="font-semibold">{title}</h4>
      <div className="bg-muted/50 rounded-lg p-4 space-y-3">{children}</div>
    </div>
  )
}
