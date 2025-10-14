import { Copy } from 'lucide-react'
import { Button } from '../ui/button'
import { toast } from 'sonner'
import { cn } from '../../lib/utils'

interface CommandBoxProps {
  command: string
  description?: string
  label?: string
  className?: string
}

export function CommandBox({
  command,
  description,
  label = 'command',
  className,
}: CommandBoxProps) {
  const handleCopy = () => {
    navigator.clipboard.writeText(command)
    toast.success(`Copied ${label} to clipboard`)
  }

  return (
    <div className={cn('space-y-2', className)}>
      <div
        className="flex items-center justify-between gap-3 bg-muted/30 border border-border/50 px-4 py-3 rounded-lg"
        role="region"
        aria-label={`Command: ${label}`}
      >
        <code className="font-mono text-xs sm:text-sm flex-1 text-foreground">{command}</code>
        <Button
          variant="ghost"
          size="sm"
          className="h-8 w-8 p-0 flex-shrink-0 hover:bg-background"
          onClick={handleCopy}
          aria-label={`Copy ${label} to clipboard`}
        >
          <Copy className="w-4 h-4" aria-hidden="true" />
        </Button>
      </div>
      {description && <p className="text-sm text-muted-foreground leading-relaxed">{description}</p>}
    </div>
  )
}
