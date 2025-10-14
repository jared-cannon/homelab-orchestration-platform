# App System Architecture

**Version:** 1.0
**Last Updated:** October 2025
**Status:** In Development

## Overview

Application marketplace and deployment system using standard Docker Compose format with intelligent orchestration enhancements. Combines user-facing features with programmatic deployment architecture.

---

## Architecture Decision

### Hybrid: Standard Compose + Intelligence Layer

**Rejected Approach: Go Templates in YAML**
```yaml
compose_template: |
  services:
    app:
      image: app:{{.Version}}
      environment:
        - DB_HOST={{.DatabaseHost}}
```

**Problems:**
- Non-standard format
- Cannot validate with docker-compose CLI
- Difficult community contributions
- Loses Docker ecosystem tooling
- Hard to test locally

**Adopted Approach:**
```
Standard docker-compose.yaml → Intelligence Enhancement → Programmatic Deployment
```

**Benefits:**
- Standard Docker Compose format
- Community can contribute easily
- Works with ecosystem tooling (validation, IDEs)
- Testable locally with `docker-compose up`
- Intelligence added during deployment, not in template

---

## Recipe Format

### Directory Structure

```
apps/vaultwarden/
├── manifest.yaml          # Platform metadata
├── docker-compose.yaml    # Standard Docker Compose
├── logo.png              # 512x512 PNG
├── screenshots/
└── README.md
```

### manifest.yaml Specification

Platform-specific metadata for intelligent orchestration.

```yaml
# Required fields
id: vaultwarden
name: Vaultwarden
version: 1.30.0
slug: vaultwarden
category: security
tagline: "Open source password manager"
description: "Self-hosted Bitwarden-compatible password manager"

# Branding
icon: logo.png
author: Vaultwarden Community
website: https://github.com/dani-garcia/vaultwarden
source_code: https://github.com/dani-garcia/vaultwarden

# Resource requirements (intelligent scheduler)
requirements:
  memory:
    minimum: 512MB
    recommended: 1GB
  storage:
    minimum: 1GB
    recommended: 5GB
    type: any  # ssd, hdd, any
  cpu:
    minimum_cores: 1
    recommended_cores: 1
  reliability: high  # high, medium, low
  always_on: true

# Database provisioning (optional)
database:
  engine: none  # postgres, mysql, mariadb, sqlite, none
  auto_provision: false

# Cache provisioning (optional)
cache:
  engine: none  # redis, memcached, none
  auto_provision: false

# Volume configuration
volumes:
  vaultwarden_data:
    description: User passwords and data
    size_estimate: 5GB
    backup_priority: high
    backup_frequency: daily

# Post-deployment automation
post_install:
  - type: message
    title: "Vaultwarden Installed"
    message: |
      Vaultwarden ready at https://vaultwarden.${DOMAIN}
      Create admin account by visiting URL.
      Disable signups after account creation.

# Health monitoring
health:
  endpoint: /alive
  interval: 30s
  timeout: 10s
  unhealthy_threshold: 3

# Update configuration
updates:
  strategy: manual  # automatic, manual, notify
  backup_before_update: true
  rollback_on_failure: true
```

### docker-compose.yaml Specification

Standard Docker Compose format. Environment variables substituted during deployment.

```yaml
version: '3.8'

services:
  vaultwarden:
    image: vaultwarden/server:1.30.0
    restart: unless-stopped

    # Variables injected by deployment service
    environment:
      - DOMAIN=https://${DOMAIN}
      - SIGNUPS_ALLOWED=${ALLOW_SIGNUPS}
      - ADMIN_TOKEN=${ADMIN_TOKEN}

    volumes:
      - vaultwarden-data:/data

    ports:
      - "${PORT}:80"

    deploy:
      replicas: 1
      resources:
        limits:
          memory: 1G
        reservations:
          memory: 512M
      # Placement constraints added programmatically

    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/alive"]
      interval: 30s
      timeout: 10s

volumes:
  vaultwarden-data:
    driver: local
```

---

## Deployment Flow

### User Workflow

```
1. Browse marketplace → Select app
2. System analyzes devices → Recommends optimal device
3. User confirms or overrides device selection
4. Configure app options (most: zero config)
5. System provisions dependencies (database, cache)
6. Deploy with real-time progress
7. Access app at generated URL
```

### Technical Process

```
1. User initiates deployment via UI
         ↓
2. Fetch manifest.yaml + docker-compose.yaml from repository
         ↓
3. Intelligence Layer Enhancement:
   - Device selection (resource scoring algorithm)
   - Database provisioning (if manifest.database.auto_provision)
   - Cache provisioning (if manifest.cache.auto_provision)
   - Secret generation (passwords, API keys)
   - Placement constraint injection
   - Environment variable substitution
         ↓
4. Programmatic deployment via Docker Swarm API
         ↓
5. Post-install hook execution
         ↓
6. Health monitoring activation
```

### State Machine

```
StatusValidating → StatusPreparing → StatusDeploying → StatusRunning
OR → StatusFailed → StatusRollingBack → StatusRolledBack
```

---

## Backend Services

### MarketplaceService

```go
// backend/internal/services/marketplace.go

type MarketplaceService struct {
    db              *gorm.DB
    recipeLoader    *RecipeLoader
    deviceService   *DeviceService
}

func (s *MarketplaceService) ListRecipes(category string) ([]Recipe, error)
func (s *MarketplaceService) GetRecipe(slug string) (*Recipe, error)
func (s *MarketplaceService) ValidateDeployment(recipeSlug string, deviceID uuid.UUID, config map[string]interface{}) (*ValidationResult, error)
func (s *MarketplaceService) LoadRecipesFromDisk() error
```

### DeploymentService

```go
// backend/internal/services/deployment_service.go

type DeploymentService struct {
    db                   *gorm.DB
    sshClient            *ssh.Client
    dockerClient         *docker.Client
    wsHub                *websocket.Hub
    appRegistry          *AppRegistry
    intelligentScheduler *IntelligentScheduler
    databasePool         *DatabasePool
    cachePool            *CachePool
    secretGenerator      *SecretGenerator
}

func (s *DeploymentService) Deploy(ctx context.Context, req DeployRequest) (*Deployment, error) {
    // 1. Fetch app definition from registry
    manifest, composeFile := s.appRegistry.FetchAppFiles(req.AppID)

    // 2. Intelligent device selection
    device, score := s.intelligentScheduler.SelectOptimalDevice(manifest.Requirements)

    // 3. Provision dependencies (database, cache)
    env := s.provisionDependencies(manifest, device)

    // 4. Enhance compose with intelligence (placement constraints, labels)
    enhancedCompose := s.enhanceCompose(composeFile, manifest, device, env)

    // 5. Deploy programmatically via Docker Swarm API
    services := s.deployToSwarm(req.AppID, enhancedCompose, device)

    // 6. Run post-install hooks
    s.runPostInstallHooks(manifest, services, env)

    // 7. Health check
    s.checkHealth(services, manifest.Health)

    // 8. WebSocket updates throughout
}
```

### Intelligence Enhancement

```go
func (d *DeploymentService) enhanceCompose(
    composeYAML string,
    manifest *Manifest,
    device *Device,
    env map[string]string,
) (string, error) {
    compose, err := parseComposeFile(composeYAML)
    if err != nil {
        return "", err
    }

    for serviceName, service := range compose.Services {
        // Add placement constraints
        if manifest.Requirements.Storage.Type == "ssd" {
            service.Deploy.Placement.Constraints = append(
                service.Deploy.Placement.Constraints,
                "node.labels.storage == ssd",
            )
        }

        // Add management labels
        service.Deploy.Labels["homelab.app"] = manifest.ID
        service.Deploy.Labels["homelab.version"] = manifest.Version
        service.Deploy.Labels["homelab.managed"] = "true"

        // Substitute environment variables
        for i, envVar := range service.Environment {
            service.Environment[i] = substituteEnv(envVar, env)
        }
    }

    // Add overlay network
    compose.Networks["homelab-overlay"] = NetworkConfig{
        External: true,
    }

    // Attach services to overlay network
    for _, service := range compose.Services {
        service.Networks = append(service.Networks, "homelab-overlay")
    }

    return marshalCompose(compose)
}
```

### AppRegistry Service

```go
// backend/internal/services/app_registry.go

type AppRegistry struct {
    db         *gorm.DB
    httpClient *http.Client
    repoURL    string  // GitHub repository URL
}

const DefaultRepoURL = "https://raw.githubusercontent.com/username/homelab-apps/main"

func (r *AppRegistry) SyncRegistry() error
func (r *AppRegistry) FetchAppFiles(appID string) (*Manifest, string, error)
func (r *AppRegistry) CheckForUpdates() ([]AppUpdate, error)
```

**Migration Path:**
1. Phase 1: Load from local `marketplace-recipes/` (current)
2. Phase 2: Support both local and GitHub registry
3. Phase 3: Deprecate local, use GitHub only

---

## API Endpoints

```
# Marketplace
GET    /api/v1/marketplace/recipes              # List all recipes
GET    /api/v1/marketplace/recipes/:slug        # Get recipe details
POST   /api/v1/marketplace/recipes/:slug/validate  # Validate deployment config

# Deployments
GET    /api/v1/deployments                      # List user's deployments
POST   /api/v1/deployments                      # Create new deployment
GET    /api/v1/deployments/:id                  # Deployment details
POST   /api/v1/deployments/:id/start            # Start containers
POST   /api/v1/deployments/:id/stop             # Stop containers
POST   /api/v1/deployments/:id/restart          # Restart
DELETE /api/v1/deployments/:id                  # Remove deployment
GET    /api/v1/deployments/:id/logs             # Container logs (stream)
GET    /api/v1/deployments/:id/troubleshoot     # Debug info
```

---

## Frontend Components

### Pages

**MarketplacePage** (`/marketplace`)
- Browse recipes grid
- Search and filtering
- Category navigation

**RecipeDetailPage** (`/marketplace/:slug`)
- Recipe details
- Deploy button
- Requirements display

**DeploymentsPage** (`/deployments`)
- User's deployed apps
- Status indicators
- Quick actions

**DeploymentDetailPage** (`/deployments/:id`)
- Deployment status
- Container logs
- Controls (start/stop/restart)

### Key Components

**RecipeCard**
```tsx
<RecipeCard recipe={recipe}>
  - Icon
  - Name + tagline
  - Category badge
  - Resource requirements (RAM, Storage)
  - "Deploy" button
</RecipeCard>
```

**DeploymentWizard**
```tsx
<DeploymentWizard recipe={recipe}>
  Step 1: Select Device → AUTOMATIC with manual override
    - System analyzes devices, scores them
    - Auto-selects highest-scoring device
    - Shows score and reasoning: "Server-02 (Score: 95) - 8GB RAM free, 40% load"
    - User can override: "Deploy to a different device ▼"

  Step 2: Resource Preview → Shows database sharing savings
    - "NextCloud will use existing Postgres → Saves 1GB RAM"
    - OR "No Postgres found → Will deploy shared instance (1.2GB RAM)"

  Step 3: Configure Options (from recipe.config_options)
    - Most apps: Zero config needed
    - Advanced users: Show optional config

  Step 4: Deploy (real-time progress via WebSocket)
    - Shows: "Deploying shared Postgres...", "Creating database nextcloud_db...", etc.

  Step 5: Success (post-deploy instructions + resource savings)
    - "Saved 1GB RAM by using shared database"
</DeploymentWizard>
```

**DeploymentCard**
```tsx
<DeploymentCard deployment={deployment}>
  - Status badge (running/stopped/failed)
  - App name + domain
  - Quick actions: Start/Stop/Restart/Delete
  - "View Logs" button
  - Link to app URL
</DeploymentCard>
```

---

## App Repository Structure

### GitHub Repository

```
homelab-apps/ (GitHub repository)
├── apps/
│   ├── nextcloud/
│   │   ├── manifest.yaml
│   │   ├── docker-compose.yaml
│   │   ├── logo.png
│   │   ├── screenshots/
│   │   └── README.md
│   ├── vaultwarden/
│   └── ...
├── index.json                     # App catalog
├── categories/
│   ├── productivity.yaml
│   ├── media.yaml
│   └── security.yaml
├── templates/
│   └── app-template/
├── scripts/
│   ├── validate-app.sh
│   ├── test-deploy.sh
│   └── generate-index.sh
└── .github/
    └── workflows/
        ├── validate-pr.yml
        └── publish.yml
```

### Catalog Index Format

```json
{
  "version": "1.0",
  "updated": "2025-10-13T10:30:00Z",
  "apps": [
    {
      "id": "nextcloud",
      "name": "NextCloud",
      "description": "Self-hosted file sync and share",
      "category": "productivity",
      "version": "28.0.1",
      "icon_url": "https://raw.githubusercontent.com/.../apps/nextcloud/logo.png",
      "manifest_url": "https://raw.githubusercontent.com/.../apps/nextcloud/manifest.yaml",
      "compose_url": "https://raw.githubusercontent.com/.../apps/nextcloud/docker-compose.yaml",
      "requirements": {
        "memory_mb": 2048,
        "storage_gb": 50
      },
      "last_updated": "2025-10-10T14:20:00Z"
    }
  ]
}
```

### Registry Synchronization

```go
func (r *AppRegistry) SyncRegistry() error {
    indexURL := fmt.Sprintf("%s/index.json", r.repoURL)
    resp, err := r.httpClient.Get(indexURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var index AppIndex
    if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
        return err
    }

    for _, app := range index.Apps {
        existingApp, _ := r.db.GetApp(app.ID)

        if existingApp == nil {
            r.db.Create(&app)
        } else if app.Version != existingApp.Version {
            r.db.Update(&app)
        }
    }

    return nil
}

func (r *AppRegistry) FetchAppFiles(appID string) (*Manifest, string, error) {
    app, err := r.db.GetApp(appID)
    if err != nil {
        return nil, "", err
    }

    manifestResp, err := r.httpClient.Get(app.ManifestURL)
    if err != nil {
        return nil, "", err
    }
    defer manifestResp.Body.Close()

    var manifest Manifest
    if err := yaml.NewDecoder(manifestResp.Body).Decode(&manifest); err != nil {
        return nil, "", err
    }

    composeResp, err := r.httpClient.Get(app.ComposeURL)
    if err != nil {
        return nil, "", err
    }
    defer composeResp.Body.Close()

    composeBytes, err := io.ReadAll(composeResp.Body)
    if err != nil {
        return nil, "", err
    }

    return &manifest, string(composeBytes), nil
}
```

---

## Update Mechanism

### Version Detection

```go
type UpdateChecker struct {
    registry *AppRegistry
    db       *gorm.DB
}

func (u *UpdateChecker) CheckForUpdates() ([]AppUpdate, error) {
    if err := u.registry.SyncRegistry(); err != nil {
        return nil, err
    }

    var installed []Deployment
    u.db.Find(&installed)

    updates := []AppUpdate{}
    for _, deployment := range installed {
        latestApp, _ := u.registry.GetApp(deployment.AppID)
        if latestApp == nil {
            continue
        }

        if semver.Compare(latestApp.Version, deployment.Version) > 0 {
            updates = append(updates, AppUpdate{
                AppID:          deployment.AppID,
                CurrentVersion: deployment.Version,
                LatestVersion:  latestApp.Version,
            })
        }
    }

    return updates, nil
}
```

### Rolling Update

```go
func (u *UpdateManager) UpdateApp(deploymentID uuid.UUID) error {
    deployment, err := u.db.GetDeployment(deploymentID)
    if err != nil {
        return err
    }

    manifest, compose, err := u.registry.FetchAppFiles(deployment.AppID)
    if err != nil {
        return err
    }

    // Create backup
    if err := u.backup.Create(deployment); err != nil {
        return err
    }

    // Deploy new version
    newDeployment, err := u.deployer.DeployApp(deployment.AppID, deployment.UserConfig)
    if err != nil {
        u.deployer.Rollback(deployment)
        return err
    }

    // Health check with timeout
    healthy, err := u.healthChecker.Check(newDeployment, 60*time.Second)
    if err != nil || !healthy {
        u.deployer.Rollback(newDeployment)
        return fmt.Errorf("health check failed, rolled back")
    }

    // Remove old deployment
    u.deployer.Remove(deployment)

    return nil
}
```

---

## Community Contribution

### Adding New Apps

**1. Repository Fork**

**2. App Scaffolding**
```bash
$ homelab-cli create-app

App name: Monica
Category: productivity
Docker image: monica:latest
Requires database: yes
Database engine: mysql

Generated: apps/monica/
```

**3. File Customization**

Edit `manifest.yaml` with requirements, `docker-compose.yaml` if needed.

**4. Local Validation**
```bash
$ homelab-cli validate monica
✓ manifest.yaml valid
✓ docker-compose.yaml valid
✓ All files present

$ docker-compose -f apps/monica/docker-compose.yaml config
# Verify output
```

**5. Pull Request Submission**

CI pipeline validates:
- YAML syntax
- Schema compliance
- Docker image security scan
- Test deployment

**6. Merge and Publish**

App becomes available in marketplace after approval.

### GitHub Actions Validation

```yaml
# .github/workflows/validate-app.yml
name: Validate App
on:
  pull_request:
    paths:
      - 'apps/**'

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Identify changed apps
        id: changed
        run: |
          APPS=$(git diff --name-only ${{ github.event.pull_request.base.sha }} | grep '^apps/' | cut -d/ -f2 | sort -u)
          echo "apps=$APPS" >> $GITHUB_OUTPUT

      - name: Validate manifests
        run: |
          for app in ${{ steps.changed.outputs.apps }}; do
            node scripts/validate-manifest.js apps/$app/manifest.yaml
          done

      - name: Validate compose files
        run: |
          for app in ${{ steps.changed.outputs.apps }}; do
            docker-compose -f apps/$app/docker-compose.yaml config
          done

      - name: Security scan
        run: |
          for app in ${{ steps.changed.outputs.apps }}; do
            IMAGE=$(grep 'image:' apps/$app/docker-compose.yaml | head -1 | awk '{print $2}')
            trivy image --severity HIGH,CRITICAL $IMAGE
          done
```

---

## Implementation Plan

### Phase 1: Foundation ✅ COMPLETED
1. ✅ Recipe YAML schema (old format with templates)
2. ✅ RecipeLoader service (local file loading)
3. ✅ Marketplace database migrations
4. ✅ API endpoints for recipes
5. ✅ Frontend: Marketplace page with grid

**Deliverable:** Backend serves recipes via API (old format)

### Phase 2: Hybrid Architecture Migration (Current)
1. **Define new format** ✅
   - Separate manifest.yaml + docker-compose.yaml
   - Standard Docker Compose (no Go templates)

2. **AppRegistry Service** (In Progress)
   - Fetch apps from GitHub repository
   - Parse manifest.yaml + docker-compose.yaml
   - Cache locally in database
   - Update checking mechanism

3. **Programmatic Deployment** (In Progress)
   - Parse standard docker-compose.yaml
   - Enhance with intelligent placement constraints
   - Substitute environment variables
   - Deploy via Docker Swarm API

4. **DeploymentService Refactor** (Pending)
   - Remove template rendering
   - Add compose enhancement logic
   - Add dependency provisioning (database, cache)
   - Integrate with IntelligentScheduler

**Deliverable:** Deploy apps from GitHub repository with intelligent orchestration

### Phase 3: Management & Monitoring (Week 3)
1. Deployments list page
2. Start/Stop/Restart/Delete actions
3. Container logs streaming (WebSocket)
4. Health checks
5. Post-deployment instructions display

**Deliverable:** Full lifecycle management

### Phase 4: Additional Recipes (Week 4)
1. Add 5-10 curated recipes:
   - Vaultwarden (password manager)
   - Uptime Kuma (monitoring)
   - Jellyfin (media server)
   - Immich (photo library)
   - Nextcloud (file sync)
   - Paperless-ngx (document management)
   - Homepage (dashboard)
2. Recipe validation tests
3. Documentation for custom recipes

**Deliverable:** Production-ready marketplace

---

## Technical Benefits

### Advantages

| Aspect | Template Approach | Standard Compose Approach |
|--------|-------------------|---------------------------|
| Format | Custom Go template | Standard Docker Compose |
| Validation | Custom parser | `docker-compose config` |
| Testing | Complex | `docker-compose up` |
| Community | High barrier | Low barrier |
| Tooling | None | Full ecosystem |
| Intelligence | In template | In deployment layer |

### Architecture Flow

```
GitHub Repository (Standard Compose Files)
            ↓
     Registry Service (Sync)
            ↓
    Intelligence Layer (Enhancement)
            ↓
   Docker Swarm API (Deployment)
            ↓
     Running Services (Optimized)
```

The intelligence layer enhances standard compose files during deployment rather than embedding logic in templates. This maintains compatibility with Docker ecosystem while adding intelligent orchestration features.
