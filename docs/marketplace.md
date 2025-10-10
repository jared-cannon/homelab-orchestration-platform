# Marketplace ("App Store") Feature Design Document

**Version:** 1.0
**Last Updated:** October 2025
**Status:** In Development

## Overview
Create an open-source application marketplace where users can browse and deploy pre-configured Docker Compose applications (recipes) to their homelab devices. Similar to Laravel Forge's one-click app deployments or Coolify's templates.

---

## Architecture Components

### 1. **Recipe Definition System** (YAML-based, similar to software-definitions/)

**Location**: `backend/marketplace-recipes/`

**Structure**:
```yaml
# backend/marketplace-recipes/vaultwarden.yaml
id: vaultwarden
name: Vaultwarden
slug: vaultwarden
category: security
tagline: "Open source password manager"
description: "Self-hosted Bitwarden-compatible password manager. Lightweight, secure, and feature-complete."

icon_url: "https://cdn.jsdelivr.net/gh/walkxcode/dashboard-icons/png/vaultwarden.png"

# Resource requirements
resources:
  min_ram_mb: 512
  min_storage_gb: 1
  recommended_ram_mb: 1024
  recommended_storage_gb: 5
  cpu_cores: 1

# Docker compose template
compose_template: |
  version: '3.8'
  services:
    vaultwarden:
      image: vaultwarden/server:{{.Version}}
      container_name: {{.ContainerName}}
      restart: unless-stopped
      environment:
        DOMAIN: "https://{{.Domain}}"
        SIGNUPS_ALLOWED: "{{.AllowSignups}}"
      volumes:
        - vaultwarden-data:/data
      ports:
        - "{{.InternalPort}}:80"

  volumes:
    vaultwarden-data:

# User-configurable options (shown in deployment wizard)
config_options:
  - name: domain
    label: "Domain"
    type: string
    default: "vault.home"
    required: true
    description: "Domain name for accessing Vaultwarden"

  - name: allow_signups
    label: "Allow new user registrations"
    type: boolean
    default: false
    description: "Allow anyone to create an account (disable after first user)"

  - name: version
    label: "Version"
    type: string
    default: "latest"
    description: "Vaultwarden version tag"

# Post-deployment instructions
post_deploy_instructions: |
  🎉 Vaultwarden is now running!

  **Next steps:**
  1. Visit https://{{.Domain}} to create your account
  2. Install browser extensions from https://bitwarden.com/download/
  3. Configure your server URL as https://{{.Domain}}
  4. IMPORTANT: Disable signups after creating your account (set SIGNUPS_ALLOWED=false)

# Health check
health_check:
  path: "/alive"
  expected_status: 200
  timeout_seconds: 60
```

### 2. **Backend Services**

#### **MarketplaceService** (`backend/internal/services/marketplace.go`)
```go
type MarketplaceService struct {
    db              *gorm.DB
    recipeLoader    *RecipeLoader
    deviceService   *DeviceService
}

// Core methods:
func (s *MarketplaceService) ListRecipes(category string) ([]Recipe, error)
func (s *MarketplaceService) GetRecipe(slug string) (*Recipe, error)
func (s *MarketplaceService) ValidateDeployment(recipeSlug string, deviceID uuid.UUID, config map[string]interface{}) (*ValidationResult, error)
func (s *MarketplaceService) LoadRecipesFromDisk() error
```

#### **DeploymentService** (`backend/internal/services/deployment.go`)
```go
type DeploymentService struct {
    db              *gorm.DB
    sshClient       *ssh.Client
    dockerClient    *docker.Client
    wsHub           *websocket.Hub
    marketplaceService *MarketplaceService
}

// Core deployment flow:
func (s *DeploymentService) Deploy(ctx context.Context, req DeployRequest) (*Deployment, error) {
    // 1. Validate (resources, ports, Docker)
    // 2. Render template
    // 3. Deploy via SSH + docker-compose
    // 4. Health check
    // 5. WebSocket updates throughout
}

// Deployment state machine (already exists in models/deployment.go)
// StatusValidating → StatusPreparing → StatusDeploying → StatusRunning
// OR → StatusFailed → StatusRollingBack → StatusRolledBack
```

#### **RecipeLoader** (`backend/internal/services/recipe_loader.go`)
```go
// Loads and validates YAML recipes from marketplace-recipes/
type RecipeLoader struct {
    recipesPath string
    cache       map[string]*Recipe
}

func (r *RecipeLoader) LoadAll() (map[string]*Recipe, error)
func (r *RecipeLoader) Validate(recipe *Recipe) error
```

### 3. **Database Models** (Already exist, minor enhancements)

**Application** model is already defined - will be seeded from YAML recipes:
```go
// backend/internal/models/application.go
type Application struct {
    ID              uuid.UUID
    Name            string
    Slug            string // "vaultwarden"
    Category        string // "security", "media", "productivity"
    Description     string
    IconURL         string
    DockerImage     string
    RequiredRAM     int64  // bytes
    RequiredStorage int64  // bytes
    ConfigTemplate  string // docker-compose template
    SetupSteps      []byte // JSON - post-deploy instructions
}
```

**Deployment** model already exists - tracks deployed apps.

### 4. **API Endpoints**

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
GET    /api/v1/deployments/:id/debug            # Debug info (compose, SSH commands)
```

### 5. **Frontend Components**

#### **Pages**:
- **MarketplacePage** (`/marketplace`) - Browse recipes grid
- **RecipeDetailPage** (`/marketplace/:slug`) - Recipe details + deploy button
- **DeploymentsPage** (`/deployments`) - User's deployed apps
- **DeploymentDetailPage** (`/deployments/:id`) - Deployment status, logs, controls

#### **Key Components**:

**RecipeCard** - App card in marketplace grid:
```tsx
<RecipeCard recipe={recipe}>
  - Icon
  - Name + tagline
  - Category badge
  - Resource requirements (RAM, Storage)
  - "Deploy" button
</RecipeCard>
```

**DeploymentWizard** - Multi-step modal:
```tsx
<DeploymentWizard recipe={recipe}>
  Step 1: Select Device (dropdown)
  Step 2: Configure Options (from recipe.config_options)
  Step 3: Resource Check (✓ RAM available, ✓ Storage available, ✓ Port free)
  Step 4: Deploy (real-time progress via WebSocket)
  Step 5: Success (post-deploy instructions)
</DeploymentWizard>
```

**DeploymentCard** - Deployed app card:
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

## UI/UX Design (Laravel Forge/Herd Style)

### Marketplace Page
```
┌─────────────────────────────────────────────────────────────┐
│ 🏪 Marketplace                                   [Search...] │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  [All] [Security] [Media] [Productivity] [Monitoring]        │
│                                                               │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                │
│  │ 🔐 Vault  │  │ 📸 Immich │  │ 🎬 Jellyfin│                │
│  │ warden    │  │           │  │            │                │
│  │ Password  │  │ Photo lib │  │ Media srv  │                │
│  │           │  │           │  │            │                │
│  │ 512MB RAM │  │ 2GB RAM   │  │ 1GB RAM    │                │
│  │ 1GB Disk  │  │ 50GB Disk │  │ 10GB Disk  │                │
│  │           │  │           │  │            │                │
│  │ [Deploy]  │  │ [Deploy]  │  │ [Deploy]   │                │
│  └───────────┘  └───────────┘  └───────────┘                │
└─────────────────────────────────────────────────────────────┘
```

### Deployment Wizard Modal
```
┌─────────────────────────────────────────┐
│ Deploy Vaultwarden                    × │
├─────────────────────────────────────────┤
│                                         │
│ ● Select Device ○ Configure ○ Deploy   │
│                                         │
│ Choose where to deploy:                │
│ ┌─────────────────────────────────┐   │
│ │ 🖥️  Main Server (192.168.1.100) │   │
│ │ ✓ 4GB RAM available             │   │
│ │ ✓ 50GB storage available        │   │
│ │ ✓ Docker installed              │   │
│ └─────────────────────────────────┘   │
│                                         │
│ [Back]                     [Continue] ─>│
└─────────────────────────────────────────┘
```

### Deployments Page
```
┌─────────────────────────────────────────────────────────────┐
│ 📦 My Apps                                                   │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 🔐 Vaultwarden                       [●] Running    │    │
│  │ https://vault.home                                  │    │
│  │ Main Server • Deployed 2 days ago                   │    │
│  │                                                       │    │
│  │ [Open App] [Restart] [Stop] [Logs] [Delete]         │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ 📸 Immich                            [○] Stopped    │    │
│  │ https://photos.home                                 │    │
│  │ Main Server • Deployed 1 week ago                   │    │
│  │                                                       │    │
│  │ [Start] [Logs] [Delete]                             │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Foundation (Week 1) ✅ COMPLETED
1. ✅ Create recipe YAML schema
2. ✅ Implement RecipeLoader service
3. ✅ Create marketplace database migrations (using existing Application/Deployment models)
4. ✅ Add API endpoints for recipes
5. ⏳ Frontend: Marketplace page with static grid (NEXT)

**Deliverable**: Backend ready, can serve recipes via API

### Phase 2: Single App Deployment (Week 2)
1. Implement DeploymentService with state machine
2. Template rendering (Go templates for docker-compose)
3. WebSocket progress updates
4. DeploymentWizard component (device selection, config)
5. Resource validation (RAM, storage, ports)

**Deliverable**: Can deploy Vaultwarden to a device

### Phase 3: Management & Monitoring (Week 3)
1. Deployments list page
2. Start/Stop/Restart/Delete actions
3. Container logs streaming (WebSocket)
4. Health checks
5. Post-deployment instructions display

**Deliverable**: Full lifecycle management of deployed apps

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
3. Documentation for adding custom recipes

**Deliverable**: Production-ready marketplace with core apps

---

## Extension Points for Community Recipes

### User-Contributed Recipes
1. Users can add recipes to `~/.homelab/custom-recipes/` directory
2. Custom recipes loaded alongside built-in ones
3. Recipe validation on load (schema check)
4. UI indicates custom vs official recipes

### Future: Recipe Repository
- GitHub repo for community recipes
- PR-based review process
- Automated testing (validate YAML, test deployments)
- Recipe ratings/reviews in UI

---

## Key Design Decisions

### 1. **YAML over Database**
- Recipes are version-controlled YAML files
- Easier for contributors to add recipes (PR a YAML file)
- Can distribute recipes via Git
- Database stores *deployments*, not recipes

### 2. **Docker Compose, Not Kubernetes**
- MVP focuses on single-server deployments
- Docker Compose is familiar to homelab users
- Can add K8s support later via different recipe format

### 3. **Template Engine: Go text/template**
- Native Go support
- Simple variable substitution
- Familiar to developers

### 4. **Resource Checking Before Deploy**
- Prevents failed deployments
- Clear feedback: "Need 2GB RAM, only 1GB available"
- Checks: RAM, disk, port conflicts, Docker installed

### 5. **Real-Time Progress**
- WebSocket events for deployment status
- User sees: "Pulling image... 45%", "Starting container..."
- Not just spinners - actual progress

---

## Success Metrics

### Technical
- Recipe load time < 100ms
- Deploy Vaultwarden in < 2 minutes
- Resource check < 500ms

### User Experience
- 90% of users can deploy first app without docs
- < 5% deployment failures (with proper rollback)
- Post-deployment success rate > 95%

---

## Files to Create/Modify

### Backend
**New**:
- `backend/marketplace-recipes/*.yaml` (10 recipes)
- `backend/internal/services/marketplace.go`
- `backend/internal/services/deployment.go`
- `backend/internal/services/recipe_loader.go`
- `backend/internal/api/marketplace.go`
- `backend/internal/api/deployments.go`

**Modified**:
- `backend/cmd/server/main.go` (register routes)
- `backend/internal/models/application.go` (minor tweaks)

### Frontend
**New**:
- `frontend/src/pages/Marketplace.tsx`
- `frontend/src/pages/RecipeDetail.tsx`
- `frontend/src/pages/Deployments.tsx`
- `frontend/src/pages/DeploymentDetail.tsx`
- `frontend/src/components/RecipeCard.tsx`
- `frontend/src/components/DeploymentWizard.tsx`
- `frontend/src/components/DeploymentCard.tsx`
- `frontend/src/api/marketplace.ts` (API client)
- `frontend/src/api/deployments.ts` (API client)

**Modified**:
- `frontend/src/App.tsx` (add routes)
- `frontend/src/components/AuthLayout.tsx` (add nav links)

### Documentation
**New**:
- `docs/marketplace.md` (this document)
- `docs/adding-recipes.md` (contributor guide)

---

## Next Steps
1. ✅ Create this design document
2. ✅ Get user approval on design
3. Begin Phase 1 implementation (recipe system)
