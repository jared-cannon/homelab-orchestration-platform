# Curated Marketplace: Escape SaaS, Self-Host Everything

## Philosophy

Traditional homelab app marketplaces overwhelm users with hundreds of apps organized by technical categories (databases, media servers, productivity). This creates decision paralysis and doesn't help users understand **why** they need these apps.

Our curated marketplace takes a different approach:

**Guide users from SaaS to self-hosted solutions.**

Instead of "here are 280 apps, good luck," we present curated journeys:
- "Want to leave Google? Here's how."
- "Replace Zapier with n8n in 5 minutes."
- "Own your photos with Immich instead of Google Photos."

## User Experience Flow

### Traditional Marketplace
```
User → Browse "Productivity" category → 47 apps → Confusion → Give up
```

### Curated Marketplace
```
User → "What do you want to replace?" → Google Drive → Nextcloud (with comparison) → Deploy
```

## Marketplace Structure

### Primary Navigation

**1. Escape SaaS**
Organized by what users want to replace:

- **Replace Google Workspace**
  - Nextcloud (Drive, Calendar, Contacts, Docs)
  - Collabora Online (Google Docs alternative)
  - OnlyOffice (Microsoft Office alternative)

- **Replace Google Photos**
  - Immich (Best Google Photos clone)
  - PhotoPrism (Advanced organization)
  - Nextcloud Memories (If you already have Nextcloud)

- **Replace Communication Tools**
  - Matrix/Synapse (Slack/Discord alternative)
  - Rocket.Chat (Team messaging)
  - Jitsi Meet (Zoom/Meet alternative)

- **Replace Automation/Integration**
  - n8n (Zapier/Make alternative)
  - Activepieces (Visual workflow automation)
  - Huginn (Lightweight automation)

- **Replace Password Managers**
  - Vaultwarden (Bitwarden compatible, self-hosted)
  - Passbolt (Team password management)

- **Replace Cloud Storage**
  - Nextcloud (Full suite)
  - Syncthing (Dropbox-style sync)
  - Seafile (High-performance file sync)

**2. Media & Entertainment**
- **Media Server**: Jellyfin, Plex, Emby
- **Media Management**: Sonarr, Radarr, Lidarr, Prowlarr
- **Music Streaming**: Navidrome, Airsonic
- **Photo Organization**: Immich, PhotoPrism
- **Ebook Management**: Calibre-Web, Kavita

**3. For Developers**
Laravel-focused infrastructure and tools:
- **App Servers**: Pre-configured Laravel environments
- **Databases**: Postgres, MySQL, Redis
- **Development Tools**: Gitea, GitLab, VSCode Server
- **CI/CD**: Drone CI, Woodpecker CI
- **Monitoring**: Uptime Kuma, Grafana, Prometheus

**4. Infrastructure & Management**
Core services that power your homelab:
- **Reverse Proxy**: Traefik, Caddy, Nginx Proxy Manager
- **DNS**: Pi-hole, AdGuard Home
- **VPN**: WireGuard, Tailscale
- **Monitoring**: Uptime Kuma, Netdata, Grafana
- **Container Management**: Portainer

**5. Browse All**
Traditional view with all apps, search, and filters

## SaaS Comparison Tables

Each "Escape SaaS" category includes comparison tables to help users decide:

### Example: Google Photos vs Immich

| Feature | Google Photos | Immich (Self-Hosted) |
|---------|---------------|----------------------|
| Storage | 15GB free, $2/mo for 100GB | Unlimited (your hardware) |
| Privacy | Google scans your photos | Your data, your server |
| Search | AI-powered face/object search | AI-powered (runs locally) |
| Mobile App | ✅ iOS, Android | ✅ iOS, Android |
| Sharing | Link sharing, albums | Link sharing, albums |
| Cost | $2-20/month | Hardware cost only |
| Setup Time | Instant | 5 minutes with our platform |

### Example: Zapier vs n8n

| Feature | Zapier | n8n (Self-Hosted) |
|---------|--------|-------------------|
| Workflows | 100 tasks/mo free, then $20+/mo | Unlimited |
| Integrations | 5,000+ | 400+ (growing) |
| Custom Code | JavaScript (paid plans) | JavaScript/Python (free) |
| Data Privacy | Zapier servers | Your server |
| Cost | $20-600/month | Hardware cost only |
| Self-Hosted | ❌ | ✅ |
| Setup Time | Instant | 5 minutes with our platform |

## Recipe Metadata Extensions

To support the curated marketplace, recipes now include additional metadata:

### Enhanced manifest.yaml

```yaml
# Standard fields
name: Immich
category: media
tagline: "Self-hosted Google Photos alternative"

# NEW: SaaS Replacement Metadata
saas_replacements:
  - name: "Google Photos"
    comparison_url: "https://immich.app/docs/features/google-photos"
  - name: "iCloud Photos"

difficulty_level: "beginner"  # beginner, intermediate, advanced
setup_time_minutes: 5
popularity_score: 95  # 0-100, based on GitHub stars + community usage

# NEW: Feature Highlights (shown in comparison table)
feature_highlights:
  - "Unlimited storage (limited only by your hardware)"
  - "AI-powered face recognition and object detection"
  - "Native mobile apps (iOS, Android)"
  - "Automatic photo backup from mobile devices"
  - "Album sharing and collaboration"
  - "Timeline view with map integration"

# NEW: Prerequisites/Dependencies
dependencies:
  required:
    - traefik  # Auto-deploy if missing
  recommended:
    - postgres  # Auto-provision shared instance
    - redis     # Auto-provision shared instance

# Existing fields continue...
requirements:
  memory:
    minimum: 2GB
    recommended: 4GB
```

## Dependency Auto-Provisioning

### Problem

Traditional setup requires users to manually:
1. Deploy Traefik reverse proxy
2. Create database
3. Configure networking
4. Then finally deploy their app

**Users just want the app to work.**

### Solution: Smart Dependency Detection

When deploying an app, the system checks:

```
User clicks "Deploy Immich"
    ↓
Check: Is Traefik running on this device?
    ↓ No
Show prompt: "Immich needs Traefik for HTTPS. Deploy it automatically?"
    ↓ User clicks "Yes"
Deploy Traefik → Wait for healthy → Continue
    ↓
Check: Is Postgres available?
    ↓ No
Deploy shared Postgres instance → Create immich_db database
    ↓
Check: Is Redis available?
    ↓ No
Deploy shared Redis instance
    ↓
Deploy Immich with auto-generated config
    ↓
Success: "Immich is ready at https://photos.homelab.local"
```

### User Experience

**Before (Traditional):**
```
1. Read docs: "You need Traefik"
2. Google "how to install Traefik"
3. Deploy Traefik
4. Troubleshoot Traefik
5. Create Postgres database manually
6. Generate random password
7. Configure environment variables
8. Deploy app
9. Troubleshoot networking
```

**After (Our System):**
```
1. Click "Deploy Immich"
2. System: "This needs Traefik and Postgres. Deploy them? [Yes]"
3. Wait 2 minutes
4. App is running at https://photos.homelab.local
```

### Dependency Types

**Required Dependencies** (must be deployed)
- Reverse proxy (Traefik) if app defines domain/SSL
- Database if app requires one
- Cache if app requires Redis

**Recommended Dependencies** (optional but beneficial)
- Backup solution (improves reliability)
- Monitoring (Uptime Kuma)

**Conflicting Dependencies** (prevent deployment)
- Port conflicts
- Resource constraints

### Implementation

#### 1. Recipe Dependency Declaration

```yaml
# In manifest.yaml
dependencies:
  required:
    - type: reverse_proxy
      prefer: traefik
      alternatives: [caddy, nginx-proxy-manager]

    - type: database
      engine: postgres
      auto_provision: true
      shared: true  # Use shared instance

    - type: cache
      engine: redis
      auto_provision: true
      shared: true

  recommended:
    - type: backup
      for_volumes: [immich-data, immich-uploads]
```

#### 2. Pre-Deployment Validation

```go
type DependencyCheckResult struct {
    Satisfied    bool
    Missing      []Dependency
    NeedsPrompt  bool
    AutoDeploy   []Recipe
}

func (s *DeploymentService) CheckDependencies(recipe *Recipe, deviceID uuid.UUID) (*DependencyCheckResult, error) {
    result := &DependencyCheckResult{}

    for _, dep := range recipe.Dependencies.Required {
        exists := s.checkDependency(dep, deviceID)
        if !exists {
            result.Missing = append(result.Missing, dep)
            result.NeedsPrompt = true

            // Find recipe to auto-deploy
            depRecipe := s.findDependencyRecipe(dep)
            if depRecipe != nil {
                result.AutoDeploy = append(result.AutoDeploy, depRecipe)
            }
        }
    }

    result.Satisfied = len(result.Missing) == 0
    return result, nil
}
```

#### 3. Deployment Wizard Updates

**Step 1: App Selection** (existing)

**Step 2: Device Selection** (existing)

**Step 3: Dependency Check** (NEW)
```
⚠️ This app needs additional infrastructure:

✅ Traefik Reverse Proxy
   • Provides HTTPS and domain routing
   • Will be deployed automatically to this device
   • Estimated time: 1 minute

✅ PostgreSQL Database (Shared Instance)
   • Used for app data storage
   • Will create database "immich_db" in shared Postgres
   • Estimated time: 30 seconds

[Deploy with dependencies] [Cancel]
```

**Step 4: Configuration** (existing)

**Step 5: Deploy with Progress** (enhanced)
```
Deploying Immich...

✅ Deployed Traefik (1/3) - 32 seconds
✅ Provisioned PostgreSQL database (2/3) - 18 seconds
⏳ Deploying Immich (3/3)...
```

## UI Design Mockup

### Marketplace Landing Page

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  🏠 Marketplace                                             │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  What do you want to self-host?                      │   │
│  │  ┌─────────────────────────────────────────────┐    │   │
│  │  │ 🔍 Search apps...                            │    │   │
│  │  └─────────────────────────────────────────────┘    │   │
│  │                                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐  │
│  │ Escape    │ │ Media &   │ │   For     │ │   Browse  │  │
│  │   SaaS    │ │  Entertainment │ Developers│ │    All    │  │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘  │
│                                                             │
│  Popular Replacements                                      │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  📸 Replace Google Photos → Immich                   │   │
│  │  "Self-host your photo library with AI-powered..."  │   │
│  │  ⭐ 95/100  ⏱️ 5 min setup  👤 Beginner             │   │
│  │                                    [Deploy Now →]   │   │
│  │                                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │                                                       │   │
│  │  ☁️ Replace Google Drive → Nextcloud                │   │
│  │  "Complete productivity suite with files, cal..."   │   │
│  │  ⭐ 92/100  ⏱️ 8 min setup  👤 Beginner             │   │
│  │                                    [Deploy Now →]   │   │
│  │                                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │                                                       │   │
│  │  🔗 Replace Zapier → n8n                            │   │
│  │  "Unlimited workflow automations, self-hosted..."   │   │
│  │  ⭐ 88/100  ⏱️ 5 min setup  👤 Intermediate         │   │
│  │                                    [Deploy Now →]   │   │
│  │                                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### App Detail Page (Enhanced)

```
┌─────────────────────────────────────────────────────────────┐
│  ← Back to Marketplace                                      │
│                                                             │
│  📸 Immich                                                  │
│  Self-hosted Google Photos alternative                     │
│                                                             │
│  ⭐ 95/100  ⏱️ 5 min  👤 Beginner  📦 v1.94.1             │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                       │   │
│  │  Replaces: Google Photos, iCloud Photos             │   │
│  │                                                       │   │
│  │  ✨ Unlimited storage (your hardware)                │   │
│  │  🤖 AI-powered face & object recognition            │   │
│  │  📱 Native mobile apps (iOS, Android)               │   │
│  │  🔄 Automatic photo backup                          │   │
│  │  📸 Album sharing & collaboration                   │   │
│  │  🗺️ Timeline view with map integration              │   │
│  │                                                       │   │
│  │                    [Deploy Immich]                   │   │
│  │                                                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─ Comparison with Google Photos ──────────────────┐      │
│  │                                                   │      │
│  │  Storage:  15GB free → Unlimited (your hardware) │      │
│  │  Privacy:  Google scans → Your server, your data │      │
│  │  Cost:     $2-20/mo → Hardware only              │      │
│  │  Search:   AI-powered → AI-powered (local)       │      │
│  │                                                   │      │
│  └───────────────────────────────────────────────────┘      │
│                                                             │
│  ┌─ What You'll Need ────────────────────────────────┐      │
│  │                                                   │      │
│  │  📊 Resources                                     │      │
│  │  • RAM: 2GB minimum, 4GB recommended             │      │
│  │  • Storage: 20GB+ for system, more for photos   │      │
│  │  • CPU: 2 cores recommended                      │      │
│  │                                                   │      │
│  │  🔧 Dependencies (Auto-deployed)                  │      │
│  │  ✅ Traefik - HTTPS reverse proxy                │      │
│  │  ✅ PostgreSQL - Database (shared instance)      │      │
│  │  ✅ Redis - Caching (shared instance)            │      │
│  │                                                   │      │
│  └───────────────────────────────────────────────────┘      │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Bundle Deployments

Allow users to deploy multiple related apps together:

### Example: Complete Google Replacement Bundle

```yaml
# bundles/escape-google.yaml
name: "Escape Google Workspace"
description: "Replace Google Drive, Photos, Calendar, and Docs"
tagline: "Own your data. All of it."

apps:
  - nextcloud        # Google Drive, Calendar, Contacts
  - immich           # Google Photos
  - collabora        # Google Docs (runs in Nextcloud)
  - vaultwarden      # Password manager

estimated_time: "15 minutes"
total_resources:
  ram: "6GB minimum, 10GB recommended"
  storage: "50GB minimum, 200GB+ recommended"
```

**UI:**
```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  Bundle: Escape Google Workspace                       │
│                                                         │
│  Replace all Google services in one click:             │
│  ✅ Google Drive → Nextcloud                          │
│  ✅ Google Photos → Immich                            │
│  ✅ Google Docs → Collabora Online                    │
│  ✅ Google Passwords → Vaultwarden                    │
│                                                         │
│  Total setup time: ~15 minutes                         │
│  Total resources: 10GB RAM, 200GB storage             │
│                                                         │
│  [Deploy Full Bundle] [Customize Apps]                │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## Recipe Quality Score

Calculated automatically based on:

```go
type QualityScore struct {
    GitHubStars      int     // 0-30 points (normalized)
    LastUpdated      int     // 0-20 points (recency)
    DockerPullCount  int     // 0-15 points (popularity)
    DeploymentSuccess float64 // 0-15 points (our success rate)
    UserRating       float64 // 0-10 points (user feedback)
    Completeness     int     // 0-10 points (metadata quality)

    Total int // Sum (0-100)
}

func CalculateQualityScore(recipe *Recipe) int {
    score := 0

    // GitHub stars (capped at 30k = max points)
    stars := min(recipe.Metadata.GitHubStars, 30000)
    score += (stars * 30) / 30000

    // Recency (last 6 months = max points)
    daysSinceUpdate := time.Since(recipe.Metadata.LastUpdated).Hours() / 24
    if daysSinceUpdate < 180 {
        score += int((180 - daysSinceUpdate) * 20 / 180)
    }

    // Success rate (our deployment tracking)
    score += int(recipe.Metadata.SuccessRate * 15)

    // User rating (if we add reviews)
    score += int(recipe.UserRating * 2) // 5-star rating * 2 = 10 points

    // Metadata completeness
    if recipe.Description != "" && recipe.IconURL != "" && len(recipe.FeatureHighlights) >= 3 {
        score += 10
    }

    return min(score, 100)
}
```

## Search & Filtering

Enhanced search with multiple strategies:

### Search Modes

1. **SaaS Replacement Search**
   - Query: "google photos" → Show Immich, PhotoPrism
   - Query: "zapier" → Show n8n, Activepieces
   - Query: "dropbox" → Show Nextcloud, Syncthing

2. **Feature Search**
   - Query: "password manager" → Vaultwarden, Passbolt
   - Query: "photo backup" → Immich, PhotoPrism
   - Query: "automation" → n8n, Huginn

3. **Traditional Name Search**
   - Query: "nextcloud" → Nextcloud
   - Query: "jellyfin" → Jellyfin

### Filters

- **Difficulty**: Beginner, Intermediate, Advanced
- **Setup Time**: < 5 min, 5-15 min, > 15 min
- **Resources**: Low (< 1GB RAM), Medium (1-4GB), High (> 4GB)
- **Category**: All existing categories
- **Has Mobile App**: Yes/No
- **Quality Score**: > 80, > 60, All

## Migration from Traditional Structure

Existing recipes continue to work. New metadata is optional:

```yaml
# Existing recipe (backward compatible)
name: Uptime Kuma
category: monitoring
# ... existing fields work fine

# Enhanced recipe (new metadata)
name: Immich
category: media
saas_replacements:
  - Google Photos
difficulty_level: beginner
setup_time_minutes: 5
# ... new fields add more value
```

## Success Metrics

Track effectiveness of curated marketplace:

1. **Deployment Success Rate**: % of deployments that complete successfully
2. **Time to First App**: How quickly new users deploy their first app
3. **Dependency Auto-Deploy Success**: % of automated dependency deployments that work
4. **Bundle Adoption**: % of users deploying bundles vs individual apps
5. **Search Effectiveness**: % of searches that lead to deployments

**Target Metrics:**
- Deployment success rate: > 95%
- Time to first app: < 10 minutes
- Dependency auto-deploy success: > 90%

---

**Version:** 1.0
**Last Updated:** October 2025
**Status:** Implementation in Progress
