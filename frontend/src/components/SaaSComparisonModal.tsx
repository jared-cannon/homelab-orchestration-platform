import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from './ui/dialog'
import { Button } from './ui/button'
import { Check, X, Info } from 'lucide-react'
import type { CuratedRecipe } from '../api/types'

interface SaaSComparisonModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  recipe: CuratedRecipe | null
  onDeploy: () => void
}

// Comparison data for popular SaaS services
const getComparisonData = (saasName: string) => {
  const comparisons: Record<string, any> = {
    'Google Photos': {
      features: [
        { name: 'Storage', saas: '15GB free, $2/mo for 100GB', selfHosted: 'Unlimited (your hardware)' },
        { name: 'Privacy', saas: 'Google scans your photos', selfHosted: 'Your data, your server' },
        { name: 'AI Search', saas: true, selfHosted: true, note: '(runs locally)' },
        { name: 'Mobile Apps', saas: true, selfHosted: true },
        { name: 'Sharing', saas: 'Link sharing, albums', selfHosted: 'Link sharing, albums' },
        { name: 'Monthly Cost', saas: '$2-20/month', selfHosted: '$0' },
        { name: 'Setup Time', saas: 'Instant', selfHosted: '5 minutes' },
      ],
    },
    'iCloud Photos': {
      features: [
        { name: 'Storage', saas: '5GB free, $1/mo for 50GB', selfHosted: 'Unlimited (your hardware)' },
        { name: 'Privacy', saas: 'Apple scans your photos', selfHosted: 'Your data, your server' },
        { name: 'Face Recognition', saas: true, selfHosted: true },
        { name: 'Cross-Platform', saas: 'iOS, macOS, Web', selfHosted: 'All platforms' },
        { name: 'Monthly Cost', saas: '$1-10/month', selfHosted: '$0' },
      ],
    },
    'Zapier': {
      features: [
        { name: 'Workflows', saas: '100 tasks/mo free, then $20+/mo', selfHosted: 'Unlimited' },
        { name: 'Integrations', saas: '5,000+', selfHosted: '400+ (growing)' },
        { name: 'Custom Code', saas: 'JavaScript (paid plans)', selfHosted: 'JavaScript/Python (free)' },
        { name: 'Data Privacy', saas: 'Zapier servers', selfHosted: 'Your server' },
        { name: 'Monthly Cost', saas: '$20-600/month', selfHosted: '$0' },
        { name: 'Self-Hosted', saas: false, selfHosted: true },
      ],
    },
    'Google Drive': {
      features: [
        { name: 'Storage', saas: '15GB free, $2/mo for 100GB', selfHosted: 'Unlimited (your hardware)' },
        { name: 'Privacy', saas: 'Google has access', selfHosted: 'Your data, your server' },
        { name: 'Collaboration', saas: true, selfHosted: true },
        { name: 'Office Suite', saas: 'Google Docs', selfHosted: 'OnlyOffice/Collabora' },
        { name: 'Calendar & Contacts', saas: true, selfHosted: true },
        { name: 'Monthly Cost', saas: '$2-20/month', selfHosted: '$0' },
      ],
    },
    'Dropbox': {
      features: [
        { name: 'Storage', saas: '2GB free, $12/mo for 2TB', selfHosted: 'Unlimited (your hardware)' },
        { name: 'Privacy', saas: 'Dropbox has access', selfHosted: 'Your data, your server' },
        { name: 'Sync Speed', saas: 'Fast', selfHosted: 'Network dependent' },
        { name: 'File Versioning', saas: true, selfHosted: true },
        { name: 'Monthly Cost', saas: '$12-20/month', selfHosted: '$0' },
      ],
    },
  }

  return comparisons[saasName] || {
    features: [
      { name: 'Monthly Cost', saas: '$$$', selfHosted: '$0' },
      { name: 'Privacy', saas: 'Their servers', selfHosted: 'Your server' },
      { name: 'Control', saas: 'Limited', selfHosted: 'Full control' },
    ],
  }
}

export function SaaSComparisonModal({
  open,
  onOpenChange,
  recipe,
  onDeploy,
}: SaaSComparisonModalProps) {
  if (!recipe || !recipe.saas_replacements || recipe.saas_replacements.length === 0) {
    return null
  }

  const primaryReplacement = recipe.saas_replacements[0]
  const comparisonData = getComparisonData(primaryReplacement.name)

  const renderValue = (value: any) => {
    if (typeof value === 'boolean') {
      return value ? (
        <Check className="w-5 h-5 text-green-600 dark:text-green-400" />
      ) : (
        <X className="w-5 h-5 text-muted-foreground/50" />
      )
    }
    return <span className="text-sm">{value}</span>
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[800px] max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold">
            Comparing {primaryReplacement.name} vs {recipe.name}
          </DialogTitle>
          <DialogDescription className="text-sm">
            Understanding the differences between cloud and self-hosted solutions
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {/* Modern Table */}
          <div className="border border-border rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="bg-muted/50">
                  <th className="text-left py-3 px-4 font-semibold text-sm w-1/3">Feature</th>
                  <th className="text-left py-3 px-4 font-semibold text-sm w-1/3">{primaryReplacement.name}</th>
                  <th className="text-left py-3 px-4 font-semibold text-sm w-1/3 bg-muted/80">{recipe.name}</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {comparisonData.features.map((feature: any, index: number) => (
                  <tr key={index} className="hover:bg-muted/30 transition-colors">
                    <td className="py-3 px-4 font-medium text-sm text-muted-foreground">
                      {feature.name}
                    </td>
                    <td className="py-3 px-4 text-sm">
                      {renderValue(feature.saas)}
                    </td>
                    <td className="py-3 px-4 text-sm bg-muted/30">
                      <div className="flex items-center gap-2">
                        {renderValue(feature.selfHosted)}
                        {feature.note && (
                          <span className="text-xs text-muted-foreground">{feature.note}</span>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Simple Info Box */}
          <div className="mt-6 p-4 rounded-lg bg-muted/50 border border-border">
            <div className="flex items-start gap-3">
              <Info className="w-5 h-5 text-muted-foreground mt-0.5 flex-shrink-0" />
              <div className="space-y-2 text-sm text-muted-foreground">
                <p>
                  <strong className="text-foreground">Self-hosting</strong> means running the software on your own hardware.
                  This gives you complete control and privacy, but requires setup and maintenance.
                </p>
                <p>
                  <strong className="text-foreground">Cloud services</strong> are managed by third parties with monthly fees.
                  They're convenient but your data lives on someone else's servers.
                </p>
              </div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button
            onClick={() => {
              onDeploy()
              onOpenChange(false)
            }}
          >
            Deploy {recipe.name}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
