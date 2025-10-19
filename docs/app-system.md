# App System Architecture

## Overview

Application marketplace and deployment system using standard Docker Compose format with intelligent orchestration enhancements.

---

## Architecture Decision

### Hybrid: Standard Compose + Intelligence Layer

**Rejected Approach: Go Templates in YAML**

Problems:
- Non-standard format incompatible with Docker ecosystem
- Cannot validate with docker-compose CLI
- Difficult community contributions
- Loses Docker tooling support
- Hard to test locally

**Adopted Approach:**
```
Standard docker-compose.yaml → Intelligence Enhancement → Programmatic Deployment
```

Benefits:
- Standard Docker Compose format
- Community contributions enabled
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

# Dependency Auto-Provisioning (NEW)
dependencies:
  required:
    - type: reverse_proxy
      prefer: traefik
      alternatives: [caddy, nginx-proxy-manager]
  recommended:
    - type: backup
      for_volumes: [vaultwarden_data]

# Post-deployment automation
post_install:
  - type: message
    title: "Vaultwarden Installed"
    message: |
      Vaultwarden ready at https://vaultwarden.${DOMAIN}
      Create admin account by visiting URL.

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

Standard Docker Compose format with environment variable substitution during deployment.

```yaml
version: '3.8'

services:
  vaultwarden:
    image: vaultwarden/server:1.30.0
    restart: unless-stopped

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

    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/alive"]
      interval: 30s
      timeout: 10s

volumes:
  vaultwarden-data:
    driver: local
```

---

## Dependency Auto-Provisioning

### Overview

Automatically detect and deploy infrastructure dependencies to simplify deployment. When deploying an app that requires Traefik, PostgreSQL, or Redis, the system automatically provisions these services.

See [dependency-management.md](dependency-management.md) for detailed documentation.

### Dependency Types

**Infrastructure Dependencies:**
- **Reverse Proxy**: Traefik, Caddy, Nginx Proxy Manager (for HTTPS and domain routing)
- **Database**: PostgreSQL, MySQL, MariaDB (shared or dedicated instances)
- **Cache**: Redis, Memcached (shared instances with key prefixing)

**Application Dependencies:**
- Other apps that this app requires (e.g., Collabora requires Nextcloud)
- Auto-configuration when possible

### manifest.yaml Dependency Declaration

```yaml
# Enhanced manifest with dependencies
dependencies:
  required:
    - type: reverse_proxy
      prefer: traefik
      alternatives: [caddy, nginx-proxy-manager]

    - type: database
      engine: postgres
      auto_provision: true
      shared: true  # Use shared instance (default)

    - type: cache
      engine: redis
      auto_provision: true
      shared: true

  recommended:
    - type: backup
      for_volumes: [app-data, app-uploads]
      message: "Recommended: Enable automated backups for data protection"
```

### Deployment with Dependencies

**Enhanced User Workflow:**
```
1. Browse marketplace → Select app
2. System analyzes devices → Recommends optimal device
3. System checks dependencies:
   - Missing Traefik? → Prompt to auto-deploy
   - Missing Postgres? → Auto-provision database in shared instance
   - Missing Redis? → Auto-provision cache in shared instance
4. User confirms deployment with dependencies
5. System provisions dependencies first (in correct order)
6. Deploy app with auto-generated credentials
7. Access app at generated URL
```

**Benefits:**
- **Zero Configuration**: Users don't need to manually provision databases, caches, or reverse proxies
- **Resource Efficiency**: Shared instances reduce RAM usage by 60-70%
- **Automatic Networking**: Apps automatically configured to connect to dependencies
- **Error Prevention**: Ensures all requirements are satisfied before deployment

### Resource Savings via Shared Infrastructure

**Traditional Approach (without sharing):**
```
App A: Nginx + PHP + Postgres + Redis = 1.5GB RAM
App B: Nginx + PHP + Postgres + Redis = 1.5GB RAM
App C: Nginx + PHP + Postgres + Redis = 1.5GB RAM
Total: 4.5GB RAM
```

**Our Approach (with shared infrastructure):**
```
Shared Traefik: 100MB RAM (all apps)
Shared Postgres: 800MB RAM (3 databases)
Shared Redis: 250MB RAM (3 key prefixes)
Apps (PHP only): 3 × 400MB = 1.2GB RAM
Total: 2.35GB RAM

Savings: 2.15GB RAM (48% reduction)
```

---

## Deployment Flow

### User Workflow

```
1. Browse marketplace → Select app
2. System analyzes devices → Recommends optimal device
3. System checks dependencies → Prompts for auto-provisioning
4. User confirms or overrides device selection
5. Configure app options (most: zero config)
6. System provisions dependencies (Traefik, database, cache)
7. Deploy with real-time progress
8. Access app at generated URL
```

### Technical Process

```
1. Fetch manifest.yaml + docker-compose.yaml from repository
         ↓
2. Dependency Analysis:
   - Check required dependencies (reverse proxy, database, cache)
   - Check if dependencies exist on target device
   - Build dependency provision plan
         ↓
3. Dependency Provisioning (if needed):
   - Deploy Traefik (if missing and required)
   - Provision database in shared instance (if needed)
   - Provision cache in shared instance (if needed)
   - Wait for dependencies to be healthy
         ↓
4. Intelligence Layer Enhancement:
   - Device selection (resource scoring algorithm)
   - Secret generation (passwords, API keys, DB credentials)
   - Placement constraint injection
   - Environment variable substitution (including dependency credentials)
         ↓
5. Programmatic deployment via Docker Swarm API
         ↓
6. Post-install hook execution
         ↓
7. Health monitoring activation
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

func (s *DeploymentService) Deploy(ctx context.Context, req DeployRequest) (*Deployment, error)
```

**Deployment Process:**
1. Fetch app definition from registry
2. Intelligent device selection via scoring algorithm
3. Provision dependencies (database, cache)
4. Enhance compose file (placement constraints, labels, env vars)
5. Deploy programmatically via Docker Swarm API
6. Run post-install hooks
7. Activate health monitoring
8. Send WebSocket updates throughout

### AppRegistry Service

```go
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
└── scripts/
    ├── validate-app.sh
    ├── test-deploy.sh
    └── generate-index.sh
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

**SyncRegistry:**
- Fetch index.json from GitHub repository
- Parse app catalog
- Update local database with new apps and versions

**FetchAppFiles:**
- Retrieve manifest.yaml and docker-compose.yaml from GitHub
- Cache locally for deployment
- Validate schemas

---

## Update Mechanism

### Version Detection

**CheckForUpdates:**
1. Sync registry from GitHub
2. Compare installed app versions with latest versions
3. Use semantic versioning for comparison
4. Return list of available updates

### Rolling Update

**UpdateApp:**
1. Create backup of current deployment
2. Deploy new version with enhanced compose
3. Health check with timeout (60s)
4. If health check fails: Rollback to previous version
5. If successful: Remove old deployment

---

## Community Contribution

### Adding New Apps

**Process:**
1. Fork GitHub repository
2. Create app directory with manifest.yaml + docker-compose.yaml
3. Validate locally:
   - `docker-compose config` - validate syntax
   - Schema validation for manifest.yaml
   - Test deployment
4. Submit pull request
5. CI pipeline validates (YAML syntax, schema, security scan)
6. After approval, app appears in marketplace

### Validation Requirements

**CI Pipeline Checks:**
- YAML syntax validation
- Schema compliance (manifest.yaml)
- Docker Compose validation
- Docker image security scan (Trivy)
- Test deployment

---

## Architecture Flow

```
GitHub Repository (Standard Compose Files)
            ↓
     Registry Service (Sync)
            ↓
    Intelligence Layer (Enhancement)
    - Device selection
    - Dependency provisioning
    - Placement constraints
    - Environment substitution
            ↓
   Docker Swarm API (Deployment)
            ↓
     Running Services (Optimized)
```

The intelligence layer enhances standard compose files during deployment rather than embedding logic in templates. This maintains compatibility with Docker ecosystem while adding intelligent orchestration features.

---

**Version:** 1.0
**Last Updated:** October 2025
**Status:** In Development
