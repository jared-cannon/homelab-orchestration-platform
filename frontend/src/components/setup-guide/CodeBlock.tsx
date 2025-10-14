import { Copy } from 'lucide-react'
import { Button } from '../ui/button'
import { toast } from 'sonner'
import { cn } from '../../lib/utils'

interface CodeBlockProps {
  code: string
  language?: 'bash' | 'powershell' | 'text'
  showCopy?: boolean
  copyLabel?: string
  className?: string
}

export function CodeBlock({
  code,
  language: _language = 'bash', // Keep for future syntax highlighting
  showCopy = true,
  copyLabel = 'command',
  className,
}: CodeBlockProps) {
  const handleCopy = () => {
    navigator.clipboard.writeText(code)
    toast.success(`Copied ${copyLabel} to clipboard`)
  }

  return (
    <div
      className={cn(
        'flex items-center justify-between gap-3 bg-muted/30 border border-border/50 rounded-lg p-4',
        className
      )}
      role="region"
      aria-label={`Code block: ${copyLabel}`}
    >
      <code className="font-mono text-xs sm:text-sm flex-1 break-all text-foreground">{code}</code>
      {showCopy && (
        <Button
          variant="ghost"
          size="sm"
          className="h-8 w-8 p-0 flex-shrink-0 hover:bg-background"
          onClick={handleCopy}
          aria-label={`Copy ${copyLabel} to clipboard`}
        >
          <Copy className="w-4 h-4" aria-hidden="true" />
        </Button>
      )}
    </div>
  )
}
