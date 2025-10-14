import { cn } from '../../lib/utils'

interface GuideSectionProps {
  title: string
  children: React.ReactNode
  className?: string
}

export function GuideSection({ title, children, className }: GuideSectionProps) {
  return (
    <div className={cn('space-y-4', className)}>
      <h4 className="font-semibold text-base">{title}</h4>
      <div className="bg-muted/30 rounded-lg p-5 space-y-4 border border-border/50">{children}</div>
    </div>
  )
}
