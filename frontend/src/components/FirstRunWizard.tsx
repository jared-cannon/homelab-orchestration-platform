import { Sparkles, Network, Plus } from 'lucide-react'
import { AddDeviceDialog } from './AddDeviceDialog'
import { DeviceDiscoveryWizard } from './DeviceDiscoveryWizard'

export function FirstRunWizard() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted flex items-center justify-center px-4">
      <div className="max-w-3xl w-full">
        <div className="bg-card rounded-2xl border border-border shadow-xl p-8 sm:p-12">
          {/* Header */}
          <div className="text-center mb-10">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-primary/20 to-primary/5 mb-6">
              <Sparkles className="w-8 h-8 text-primary" />
            </div>
            <h1 className="text-4xl font-bold bg-gradient-to-r from-foreground to-foreground/70 bg-clip-text text-transparent mb-4">
              Welcome to Your Homelab
            </h1>
            <p className="text-lg text-muted-foreground mb-2">
              Let's get started by discovering devices on your network
            </p>
            <p className="text-sm text-muted-foreground/70">
              We'll automatically scan your network and help you add devices in seconds
            </p>
          </div>

          {/* Info Card */}
          <div className="bg-primary/5 border border-primary/20 rounded-xl p-6 mb-8">
            <div className="flex items-start gap-3 mb-3">
              <Network className="w-5 h-5 text-primary flex-shrink-0 mt-0.5" />
              <h3 className="font-semibold text-foreground">
                Two ways to add devices:
              </h3>
            </div>
            <ul className="space-y-3 text-sm text-muted-foreground ml-8">
              <li className="flex items-start">
                <span className="mr-2 text-primary">1.</span>
                <span><strong className="text-foreground">Automatic Discovery</strong> - Scan your network and we'll find devices for you</span>
              </li>
              <li className="flex items-start">
                <span className="mr-2 text-primary">2.</span>
                <span><strong className="text-foreground">Manual Entry</strong> - Add a specific device if you know its IP address</span>
              </li>
            </ul>
          </div>

          {/* Action Buttons */}
          <div className="space-y-4">
            {/* Primary: Auto Discovery */}
            <div className="p-6 rounded-xl border-2 border-primary/30 bg-gradient-to-br from-primary/5 to-primary/10 hover:border-primary/50 transition-all">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-foreground text-lg">Automatic Discovery</h3>
                  <p className="text-sm text-muted-foreground mt-1">Recommended for first-time setup</p>
                </div>
                <Sparkles className="w-6 h-6 text-primary" />
              </div>
              <DeviceDiscoveryWizard />
            </div>

            {/* Secondary: Manual Add */}
            <div className="p-6 rounded-xl border border-border bg-card hover:border-border/80 transition-all">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-foreground">Manual Entry</h3>
                  <p className="text-sm text-muted-foreground mt-1">Add a device by IP address</p>
                </div>
                <Plus className="w-5 h-5 text-muted-foreground" />
              </div>
              <AddDeviceDialog />
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="mt-8 text-center">
          <p className="text-sm text-muted-foreground">
            Once your devices are added, you can deploy apps like <strong>Nextcloud</strong>, <strong>Vaultwarden</strong>, and more with just a few clicks.
          </p>
        </div>
      </div>
    </div>
  )
}
