# Dependency Management & Auto-Provisioning

## Overview

Automatic detection and deployment of application dependencies to eliminate manual configuration. When deploying an app that requires Traefik, PostgreSQL, or Redis, the system automatically provisions these services.

## Problem Statement

**Traditional deployment workflow:**
```
User: "I want Immich"
  ↓
Manual Steps:
1. Read docs: "Immich requires Postgres and Redis"
2. Deploy Postgres manually
3. Create database manually
4. Generate random password
5. Deploy Redis manually
6. Configure environment variables manually
7. Finally deploy Immich
8. Troubleshoot networking issues
9. Give up or spend hours debugging

Total time: 2-4 hours
Success rate: ~60%
```

**Our automated workflow:**
```
User: "I want Immich"
  ↓
System: "Immich needs Traefik, Postgres, and Redis. Deploy them? [Yes]"
  ↓
User clicks "Yes"
  ↓
System deploys everything automatically
  ↓
2 minutes later: "Immich is ready at https://photos.homelab.local"

Total time: 2 minutes
Success rate: >95%
```

## Dependency Types

### 1. Infrastructure Dependencies

Core infrastructure required for apps to function:

#### Reverse Proxy (Traefik, Caddy, Nginx Proxy Manager)

**When Required:**
- App defines a domain (e.g., `domain: photos.homelab.local`)
- App requests HTTPS/SSL
- App needs automatic service discovery

**What We Do:**
```yaml
# App manifest indicates it needs a reverse proxy
dependencies:
  required:
    - type: reverse_proxy
      prefer: traefik
      alternatives: [caddy, nginx-proxy-manager]
```

**Auto-Provisioning:**
1. Check if Traefik is running on target device
2. If not, prompt: "Deploy Traefik reverse proxy? Required for HTTPS access."
3. Deploy Traefik with Let's Encrypt configured
4. Wait for Traefik to be healthy
5. Configure app to use Traefik (labels in docker-compose)

#### Databases (PostgreSQL, MySQL, MariaDB)

**When Required:**
- App manifest specifies `database.engine: postgres`
- App manifest has `database.auto_provision: true`

**What We Do:**
```yaml
# App manifest
database:
  engine: postgres
  auto_provision: true
  version: "15"
  env_prefix: "POSTGRES_"
```

**Auto-Provisioning Strategies:**

**Option A: Shared Instance (Default, Recommended)**
```
1. Check if shared Postgres instance exists on device
2. If yes:
   - Create new database in existing instance
   - Create dedicated user with secure password
   - Return connection credentials
3. If no:
   - Deploy shared Postgres container
   - Create database for this app
   - Create user
   - Return credentials
```

**Benefits:**
- Saves ~500MB-1GB RAM per database
- Easier backups (one instance to backup)
- Better resource utilization

**Option B: Dedicated Instance**
```
1. Deploy dedicated Postgres container for this app
2. Configure with app-specific tuning
3. Return credentials
```

**Use Cases for Dedicated:**
- High-load databases (>100 queries/sec)
- Apps requiring specific Postgres extensions
- Isolation requirements

#### Cache (Redis, Memcached)

**When Required:**
- App manifest specifies `cache.engine: redis`
- App manifest has `cache.auto_provision: true`

**Auto-Provisioning:**
```yaml
# App manifest
cache:
  engine: redis
  auto_provision: true
  version: "7"
```

**Strategy: Shared Redis Instance**
```
1. Check if shared Redis exists on device
2. If no: Deploy shared Redis container
3. Configure app with Redis connection + unique key prefix
4. Return credentials
```

**Key Prefix per App:**
```
App A: nextcloud:*
App B: immich:*
App C: laravel_app:*
```

Prevents key collisions while sharing single Redis instance.

### 2. Application Dependencies

Dependencies between applications:

#### Example: Nextcloud + Collabora

Collabora Online (office suite) runs as separate app but requires Nextcloud:

```yaml
# Collabora manifest
dependencies:
  required:
    - type: application
      name: nextcloud
      min_version: "25.0"
      message: "Collabora requires Nextcloud to be installed first"
```

**Auto-Detection:**
1. User tries to deploy Collabora
2. System checks: "Is Nextcloud installed?"
3. If no: Show message: "Install Nextcloud first, or deploy as bundle"
4. If yes: Configure Collabora to connect to Nextcloud

### 3. Network Dependencies

Apps that need to communicate:

```yaml
# Sonarr manifest (media management)
dependencies:
  recommended:
    - type: application
      name: transmission  # Torrent client
      purpose: "Download media automatically"
      auto_configure: true  # Automatically configure Sonarr to use Transmission
```

**Auto-Configuration:**
1. Deploy Sonarr
2. Detect Transmission is already deployed
3. Automatically configure Sonarr with Transmission's connection details
4. Test connection
5. Show success: "Sonarr configured to use Transmission for downloads"

## Dependency Resolution

### Dependency Graph

Build dependency graph before deployment:

```
User wants: Immich
    ↓
Dependency Analysis:
    ├─ Traefik (reverse proxy)
    │   └─ No dependencies
    ├─ PostgreSQL (database)
    │   └─ No dependencies
    └─ Redis (cache)
        └─ No dependencies

Deployment Order:
1. Traefik (1/4)
2. PostgreSQL (2/4) - can run in parallel with Traefik
3. Redis (3/4) - can run in parallel with Traefik
4. Immich (4/4) - waits for all dependencies
```

### Circular Dependency Detection

Prevent circular dependencies:

```yaml
# Invalid configuration
app_a:
  dependencies:
    - app_b

app_b:
  dependencies:
    - app_a
```

**Detection:**
```go
func (s *DependencyService) DetectCircularDependencies(recipe *Recipe) error {
    visited := make(map[string]bool)
    recStack := make(map[string]bool)

    if s.detectCycleUtil(recipe, visited, recStack) {
        return fmt.Errorf("circular dependency detected")
    }
    return nil
}
```

### Conflict Detection

Prevent conflicting dependencies:

```yaml
# Conflict: Both require exclusive port 80
app_a:
  dependencies:
    - traefik

app_b:
  dependencies:
    - nginx-proxy-manager  # Conflict! Both bind to port 80
```

**Resolution:**
```
⚠️ Conflict Detected

You're trying to deploy App B which requires Nginx Proxy Manager,
but Traefik is already running on port 80.

Options:
1. Use Traefik for App B (recommended)
2. Stop Traefik and deploy Nginx Proxy Manager
3. Configure Nginx Proxy Manager on different port

[Use Traefik] [Switch to Nginx] [Cancel]
```

## Implementation

### Data Model

```go
// Dependency represents a required or recommended dependency
type Dependency struct {
    Type         DependencyType   `yaml:"type" json:"type"`
    Name         string           `yaml:"name,omitempty" json:"name,omitempty"`
    MinVersion   string           `yaml:"min_version,omitempty" json:"min_version,omitempty"`
    Prefer       string           `yaml:"prefer,omitempty" json:"prefer,omitempty"`
    Alternatives []string         `yaml:"alternatives,omitempty" json:"alternatives,omitempty"`
    Purpose      string           `yaml:"purpose,omitempty" json:"purpose,omitempty"`
    AutoConfigure bool            `yaml:"auto_configure,omitempty" json:"auto_configure,omitempty"`
    Message      string           `yaml:"message,omitempty" json:"message,omitempty"`
}

type DependencyType string

const (
    DependencyTypeReverseProxy DependencyType = "reverse_proxy"
    DependencyTypeDatabase     DependencyType = "database"
    DependencyTypeCache        DependencyType = "cache"
    DependencyTypeApplication  DependencyType = "application"
    DependencyTypeInfrastructure DependencyType = "infrastructure"
)

// RecipeDependencies in manifest
type RecipeDependencies struct {
    Required    []Dependency `yaml:"required" json:"required"`
    Recommended []Dependency `yaml:"recommended" json:"recommended"`
}
```

### Service Layer

```go
type DependencyService struct {
    db                *gorm.DB
    recipeLoader      *RecipeLoader
    deploymentService *DeploymentService
    deviceService     *DeviceService
}

// CheckDependencies analyzes what's needed for deployment
func (s *DependencyService) CheckDependencies(
    recipe *Recipe,
    deviceID uuid.UUID,
) (*DependencyCheckResult, error) {
    result := &DependencyCheckResult{
        Satisfied: true,
        Missing:   []MissingDependency{},
        ToProvision: []ProvisionPlan{},
    }

    // Check required dependencies
    for _, dep := range recipe.Dependencies.Required {
        satisfied, err := s.checkDependency(dep, deviceID)
        if err != nil {
            return nil, err
        }

        if !satisfied {
            result.Satisfied = false
            result.Missing = append(result.Missing, MissingDependency{
                Dependency: dep,
                Critical:   true,
            })

            // Create provision plan
            plan := s.createProvisionPlan(dep, deviceID)
            result.ToProvision = append(result.ToProvision, plan)
        }
    }

    // Check recommended dependencies
    for _, dep := range recipe.Dependencies.Recommended {
        satisfied, _ := s.checkDependency(dep, deviceID)
        if !satisfied {
            result.Missing = append(result.Missing, MissingDependency{
                Dependency: dep,
                Critical:   false,
            })
        }
    }

    return result, nil
}

// checkDependency verifies if a dependency is satisfied
func (s *DependencyService) checkDependency(
    dep Dependency,
    deviceID uuid.UUID,
) (bool, error) {
    switch dep.Type {
    case DependencyTypeReverseProxy:
        return s.checkReverseProxy(dep, deviceID)

    case DependencyTypeDatabase:
        return s.checkDatabase(dep, deviceID)

    case DependencyTypeCache:
        return s.checkCache(dep, deviceID)

    case DependencyTypeApplication:
        return s.checkApplication(dep, deviceID)

    default:
        return false, fmt.Errorf("unknown dependency type: %s", dep.Type)
    }
}

// checkReverseProxy checks if reverse proxy is running
func (s *DependencyService) checkReverseProxy(
    dep Dependency,
    deviceID uuid.UUID,
) (bool, error) {
    // Check if Traefik (or preferred proxy) is deployed
    deployments, err := s.deploymentService.ListDeploymentsByDevice(deviceID)
    if err != nil {
        return false, err
    }

    for _, deployment := range deployments {
        if deployment.Recipe.Slug == dep.Prefer ||
           contains(dep.Alternatives, deployment.Recipe.Slug) {
            if deployment.Status == DeploymentStatusRunning {
                return true, nil
            }
        }
    }

    return false, nil
}

// ProvisionDependencies automatically provisions missing dependencies
func (s *DependencyService) ProvisionDependencies(
    result *DependencyCheckResult,
    deviceID uuid.UUID,
    userID uuid.UUID,
) error {
    for i, plan := range result.ToProvision {
        // Broadcast progress
        s.broadcastProgress(userID, i+1, len(result.ToProvision), plan.Name)

        switch plan.Type {
        case DependencyTypeReverseProxy:
            err := s.provisionReverseProxy(plan, deviceID)
            if err != nil {
                return fmt.Errorf("failed to provision reverse proxy: %w", err)
            }

        case DependencyTypeDatabase:
            err := s.provisionDatabase(plan, deviceID)
            if err != nil {
                return fmt.Errorf("failed to provision database: %w", err)
            }

        case DependencyTypeCache:
            err := s.provisionCache(plan, deviceID)
            if err != nil {
                return fmt.Errorf("failed to provision cache: %w", err)
            }
        }

        // Wait for dependency to be healthy
        err := s.waitForHealthy(plan, deviceID, 60*time.Second)
        if err != nil {
            return fmt.Errorf("dependency %s did not become healthy: %w", plan.Name, err)
        }
    }

    return nil
}

// provisionDatabase provisions database (shared or dedicated)
func (s *DependencyService) provisionDatabase(
    plan ProvisionPlan,
    deviceID uuid.UUID,
) error {
    if plan.UseSharedInstance {
        // Use or create shared Postgres instance
        return s.provisionSharedDatabase(plan, deviceID)
    } else {
        // Deploy dedicated database container
        return s.provisionDedicatedDatabase(plan, deviceID)
    }
}

// provisionSharedDatabase creates database in shared instance
func (s *DependencyService) provisionSharedDatabase(
    plan ProvisionPlan,
    deviceID uuid.UUID,
) error {
    // Check if shared Postgres exists
    exists := s.sharedPostgresExists(deviceID)

    if !exists {
        // Deploy shared Postgres container
        err := s.deploySharedPostgres(deviceID, plan.Version)
        if err != nil {
            return err
        }

        // Wait for Postgres to be ready
        time.Sleep(10 * time.Second)
    }

    // Create database in shared instance
    dbName := fmt.Sprintf("%s_db", plan.AppName)
    dbUser := fmt.Sprintf("%s_user", plan.AppName)
    dbPassword := generateSecurePassword(32)

    err := s.createDatabaseInSharedInstance(deviceID, dbName, dbUser, dbPassword)
    if err != nil {
        return err
    }

    // Store credentials for later use
    plan.Credentials = map[string]string{
        "DB_HOST":     "shared-postgres",
        "DB_PORT":     "5432",
        "DB_DATABASE": dbName,
        "DB_USERNAME": dbUser,
        "DB_PASSWORD": dbPassword,
    }

    return nil
}
```

### Deployment Integration

```go
// Enhanced deployment flow with dependency provisioning
func (s *DeploymentService) Deploy(
    ctx context.Context,
    req DeployRequest,
) (*Deployment, error) {
    // Get recipe
    recipe, err := s.recipeLoader.GetRecipe(req.RecipeSlug)
    if err != nil {
        return nil, err
    }

    // Check dependencies
    depResult, err := s.dependencyService.CheckDependencies(recipe, req.DeviceID)
    if err != nil {
        return nil, err
    }

    // If dependencies missing, provision them (if auto-provision enabled)
    if !depResult.Satisfied && req.AutoProvisionDependencies {
        s.broadcastMessage(req.UserID, "Provisioning dependencies...")

        err = s.dependencyService.ProvisionDependencies(depResult, req.DeviceID, req.UserID)
        if err != nil {
            return nil, fmt.Errorf("dependency provisioning failed: %w", err)
        }

        s.broadcastMessage(req.UserID, "✅ Dependencies provisioned")
    }

    // Continue with normal deployment
    return s.deployApp(ctx, recipe, req)
}
```

## User Experience

### Deployment Wizard with Dependency Detection

**Step 1: App Selection**
```
Selected: Immich
Category: Media
Replaces: Google Photos
```

**Step 2: Device Selection**
```
Recommended: homelab-server-02
  8GB RAM available
  200GB storage
  Current load: 35%
```

**Step 3: Dependency Check** (NEW)

```
┌────────────────────────────────────────────────────┐
│  Dependencies Required                            │
│                                                    │
│  Immich needs the following to run:               │
│                                                    │
│  ✅ Traefik Reverse Proxy                        │
│     Provides HTTPS and domain routing             │
│     Status: ⚠️ Not installed                     │
│     Will deploy automatically                     │
│     Estimated time: 1 minute                      │
│                                                    │
│  ✅ PostgreSQL Database                          │
│     Stores photo metadata and user data           │
│     Status: ⚠️ Not installed                     │
│     Will create database in shared Postgres       │
│     Estimated time: 30 seconds                    │
│                                                    │
│  ✅ Redis Cache                                  │
│     Improves performance                          │
│     Status: ⚠️ Not installed                     │
│     Will use shared Redis instance                │
│     Estimated time: 20 seconds                    │
│                                                    │
│  Total setup time: ~2 minutes                     │
│                                                    │
│  [ Deploy with dependencies ]  [Cancel]           │
│                                                    │
│  ℹ️ This will automatically set up everything    │
│     needed for Immich to run.                     │
└────────────────────────────────────────────────────┘
```

**Step 4: Configuration** (existing)

**Step 5: Deployment with Dependency Progress**

```
┌────────────────────────────────────────────────────┐
│  Deploying Immich...                              │
│                                                    │
│  ✅ Deployed Traefik (1/4) - 55 seconds           │
│     https://traefik.homelab.local/dashboard       │
│                                                    │
│  ✅ Provisioned PostgreSQL (2/4) - 22 seconds     │
│     Created database: immich_db                   │
│     Created user: immich_user                     │
│                                                    │
│  ✅ Provisioned Redis (3/4) - 15 seconds          │
│     Using shared Redis instance                   │
│     Key prefix: immich:*                          │
│                                                    │
│  ⏳ Deploying Immich (4/4)...                     │
│     Pulling image: ghcr.io/immich-app/immich-server│
│                                                    │
│  Progress: ███████████████████░ 85%               │
└────────────────────────────────────────────────────┘
```

**Step 6: Success**

```
┌────────────────────────────────────────────────────┐
│  ✅ Immich Successfully Deployed!                 │
│                                                    │
│  Your photo library is ready to use.              │
│                                                    │
│  Access:                                           │
│  🌐 https://photos.homelab.local                  │
│                                                    │
│  Mobile Apps:                                      │
│  📱 iOS: Download from App Store                  │
│  🤖 Android: Download from Play Store             │
│                                                    │
│  Server URL for mobile app:                       │
│  https://photos.homelab.local                     │
│                                                    │
│  Dependencies Deployed:                           │
│  • Traefik (reverse proxy)                        │
│  • PostgreSQL (shared instance, database: immich_db)│
│  • Redis (shared instance)                        │
│                                                    │
│  [Open Immich] [View Details] [Close]             │
└────────────────────────────────────────────────────┘
```

## Resource Savings Dashboard

Show users the benefits of shared infrastructure:

```
┌────────────────────────────────────────────────────┐
│  Resource Optimization Report                     │
│                                                    │
│  💰 You're saving resources by sharing infrastructure│
│                                                    │
│  Database Pooling:                                 │
│  ┌────────────────────────────────────────────┐   │
│  │ Apps using shared PostgreSQL (3):          │   │
│  │ • Nextcloud (nextcloud_db)                 │   │
│  │ • Immich (immich_db)                       │   │
│  │ • Vaultwarden (vaultwarden_db)             │   │
│  │                                            │   │
│  │ Traditional: 3 × 600MB = 1.8GB RAM        │   │
│  │ Shared: 1 × 800MB = 800MB RAM             │   │
│  │ Savings: 1GB RAM (56%)                     │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Cache Pooling:                                    │
│  ┌────────────────────────────────────────────┐   │
│  │ Apps using shared Redis (4):               │   │
│  │ • Nextcloud, Immich, Laravel App, n8n     │   │
│  │                                            │   │
│  │ Traditional: 4 × 200MB = 800MB RAM        │   │
│  │ Shared: 1 × 250MB = 250MB RAM             │   │
│  │ Savings: 550MB RAM (69%)                   │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Reverse Proxy Consolidation:                     │
│  ┌────────────────────────────────────────────┐   │
│  │ Apps using shared Traefik (6):             │   │
│  │ All apps get automatic HTTPS               │   │
│  │                                            │   │
│  │ Traditional: 6 × nginx = 6 × 50MB = 300MB │   │
│  │ Shared Traefik: 100MB RAM                  │   │
│  │ Savings: 200MB RAM (67%)                   │   │
│  └────────────────────────────────────────────┘   │
│                                                    │
│  Total Savings: 1.75GB RAM (63%)                  │
│                                                    │
└────────────────────────────────────────────────────┘
```

---

**Version:** 1.0
**Last Updated:** October 2025
**Status:** Design Complete
