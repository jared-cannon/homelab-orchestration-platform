import { BookOpen } from 'lucide-react'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from './ui/dialog'
import { GuideRenderer } from './setup-guide/GuideRenderer'
import { GuideErrorBoundary } from './setup-guide/GuideErrorBoundary'
import { serverSetupGuide } from './setup-guide/server-setup-config'

interface ServerSetupGuideProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ServerSetupGuide({ open, onOpenChange }: ServerSetupGuideProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-w-[95vw] sm:max-w-2xl lg:max-w-4xl max-h-[90vh] overflow-y-auto"
        aria-describedby="server-setup-guide-description"
      >
        <DialogHeader>
          <DialogTitle className="text-2xl flex items-center gap-2">
            <BookOpen className="w-6 h-6 text-primary" aria-hidden="true" />
            {serverSetupGuide.title}
          </DialogTitle>
          <p id="server-setup-guide-description" className="text-muted-foreground mt-2">
            {serverSetupGuide.description}
          </p>
        </DialogHeader>

        <div className="mt-4">
          <GuideErrorBoundary>
            <GuideRenderer config={serverSetupGuide} />
          </GuideErrorBoundary>
        </div>
      </DialogContent>
    </Dialog>
  )
}
