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
        className="flex items-center justify-between gap-2 bg-muted px-3 py-2 rounded"
        role="region"
        aria-label={`Command: ${label}`}
      >
        <code className="font-mono text-xs sm:text-sm flex-1">{command}</code>
        <Button
          variant="ghost"
          size="sm"
          className="h-8 w-8 p-0 flex-shrink-0"
          onClick={handleCopy}
          aria-label={`Copy ${label} to clipboard`}
        >
          <Copy className="w-4 h-4" aria-hidden="true" />
        </Button>
      </div>
      {description && <p className="text-sm text-muted-foreground">{description}</p>}
    </div>
  )
}
