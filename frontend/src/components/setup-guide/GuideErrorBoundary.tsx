import React, { Component, ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'
import { Button } from '../ui/button'

interface GuideErrorBoundaryProps {
  children: ReactNode
}

interface GuideErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

export class GuideErrorBoundary extends Component<
  GuideErrorBoundaryProps,
  GuideErrorBoundaryState
> {
  constructor(props: GuideErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): GuideErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Guide rendering error:', error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center p-8 text-center space-y-4">
          <div className="w-16 h-16 rounded-full bg-destructive/10 flex items-center justify-center">
            <AlertTriangle className="w-8 h-8 text-destructive" />
          </div>
          <div className="space-y-2">
            <h3 className="text-lg font-semibold">Something went wrong</h3>
            <p className="text-sm text-muted-foreground max-w-md">
              We encountered an error while loading the guide. This is likely a temporary issue.
            </p>
            {this.state.error && (
              <details className="text-xs text-muted-foreground">
                <summary className="cursor-pointer hover:text-foreground">Error details</summary>
                <pre className="mt-2 p-2 bg-muted rounded text-left overflow-auto">
                  {this.state.error.message}
                </pre>
              </details>
            )}
          </div>
          <Button onClick={this.handleReset} variant="outline">
            Try Again
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}
