import { useEffect, useRef } from 'react'
import { CheckCircle, AlertTriangle, XCircle, Info } from 'lucide-react'

interface LogViewerProps {
  logs: string
  className?: string
  autoScroll?: boolean
}

interface ParsedLogLine {
  timestamp: string
  icon: React.ReactNode
  iconColor: string
  message: string
  type: 'success' | 'info' | 'warning' | 'error'
}

/**
 * LogViewer - Laravel-inspired log display component
 * Features:
 * - Clean, modern design with light background
 * - Color-coded log levels with icons
 * - Monospace font for readability
 * - Auto-scroll to bottom
 * - Brand colors (indigo accents)
 */
export function LogViewer({ logs, className = '', autoScroll = true }: LogViewerProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when logs update
  useEffect(() => {
    if (autoScroll && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  // Parse log line to extract timestamp, icon, and styling
  const parseLogLine = (line: string): ParsedLogLine | null => {
    if (!line.trim()) return null

    // Extract timestamp: [2024-01-01 12:00:00]
    const timestampMatch = line.match(/^\[([^\]]+)\]/)
    const timestamp = timestampMatch ? timestampMatch[1] : ''
    const message = timestampMatch ? line.substring(timestampMatch[0].length).trim() : line

    // Determine log type based on prefix icons
    let type: ParsedLogLine['type'] = 'info'
    let icon: React.ReactNode = <Info className="w-4 h-4" />
    let iconColor = 'text-blue-600'

    if (message.startsWith('✓')) {
      type = 'success'
      icon = <CheckCircle className="w-4 h-4" />
      iconColor = 'text-green-600'
    } else if (message.startsWith('❌')) {
      type = 'error'
      icon = <XCircle className="w-4 h-4" />
      iconColor = 'text-red-600'
    } else if (message.startsWith('⚠️')) {
      type = 'warning'
      icon = <AlertTriangle className="w-4 h-4" />
      iconColor = 'text-orange-600'
    } else if (message.startsWith('▶')) {
      type = 'info'
      icon = <Info className="w-4 h-4" />
      iconColor = 'text-blue-600'
    }

    return { timestamp, icon, iconColor, message, type }
  }

  const logLines = logs.split('\n').filter((line) => line.trim() !== '')

  if (logLines.length === 0) {
    return (
      <div className={`bg-muted/30 border border-border rounded-lg p-4 text-center text-sm text-muted-foreground ${className}`}>
        No logs available
      </div>
    )
  }

  return (
    <div
      ref={scrollRef}
      className={`bg-muted/30 border border-border rounded-lg overflow-auto max-h-[400px] ${className}`}
    >
      <div className="p-3 space-y-1">
        {logLines.map((line, index) => {
          const parsed = parseLogLine(line)
          if (!parsed) return null

          return (
            <div
              key={`${index}-${parsed.timestamp}`}
              className="flex items-start gap-3 px-2 py-1.5 rounded hover:bg-muted/50 transition-colors"
            >
              {/* Icon */}
              <div className={`mt-0.5 flex-shrink-0 ${parsed.iconColor}`}>
                {parsed.icon}
              </div>

              {/* Timestamp */}
              {parsed.timestamp && (
                <div className="text-xs text-muted-foreground font-mono w-[140px] flex-shrink-0">
                  {parsed.timestamp}
                </div>
              )}

              {/* Message */}
              <div className="flex-1 text-sm font-mono text-foreground break-words">
                {parsed.message}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
