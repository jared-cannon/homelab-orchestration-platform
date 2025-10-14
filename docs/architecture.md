# Software Architecture

**Version:** 0.3 MVP (Unified Orchestration)
**Last Updated:** October 2025

## Version History
- **v0.3** (Oct 2025): **Unified orchestration focus** - Shift to multi-node intelligence, resource sharing, intelligent placement as MVP core features
- **v0.2** (Oct 2025): Architecture review, critical fixes, monorepo decision
- **v0.1** (Oct 2025): Initial design

## Critical Updates from v0.1

Design review identified and addressed critical issues:

1. **Redis Dependency Removed**: In-memory job queue for MVP (channels + goroutines), Redis migration path for scale

2. **Rollback Strategy Added**: Deployment state machine with automatic cleanup on failure

3. **Idempotency Required**: All operations check existing state before creating

4. **Credential Security Enhanced**: OS keychain integration (macOS Keychain, Linux Secret Service, Windows DPAPI)

5. **Docker Bootstrap**: Pre-flight checks with install instructions

6. **Network Discovery**: Manual IP entry as primary method, discovery as convenience feature

7. **OPNsense API**: REST API (not XML-RPC), AdGuard optional with graceful degradation

8. **Port Conflict Detection**: Pre-flight port availability validation

9. **Monorepo**: Embedded frontend architecture

1. Vision & Design Philosophy

## Core Design Goals

Simplify multi-node homelab orchestration while maintaining full control and transparency.

### Design Approach

**Target Gap:**
Existing solutions require either deep technical expertise (Proxmox, Kubernetes) or sacrifice multi-node capabilities (CasaOS, Coolify). This platform bridges that gap.

**Core Principles:**

1. **Unified Infrastructure Model**: Aggregate resources across devices
   - Present total available RAM, CPU cores, storage across all nodes
   - Abstract individual device management

2. **Intelligent Orchestration**: Automatic optimal placement
   - Score devices based on available resources, current load, reliability
   - Recommend best device for each deployment
   - Allow manual override

3. **Resource Sharing**: Reduce redundancy
   - Single shared database instance per device instead of per-container
   - Example: 5 apps sharing 1 Postgres (1.5GB) vs 5 separate instances (5GB)

4. **Infrastructure Abstraction**: Manage homelab, not devices
   - Default: System selects deployment target
   - Override: User can specify device manually

5. **Transparency**: Show all generated configurations
   - Display generated docker-compose.yml
   - Show SSH commands executed
   - Export configurations
   - Debug endpoints for troubleshooting

### Comparison with Existing Tools

| Feature | CasaOS / Coolify | Proxmox | This Platform |
|---------|------------------|---------|---------------|
| Multi-node | Single node | Yes | Unified orchestration |
| Intelligent placement | No | Manual | Automatic |
| Resource sharing | No | Manual | Automatic database pooling |
| Unified view | Per-device | Cluster view | Aggregate resources |
| Simplicity | Very simple | Complex | Simple with power user options |
| Target user | Beginners | Power users | Power users seeking simplicity |

## 1.1 Intelligent Orchestration

Multi-node orchestration with automatic device selection based on resource availability and current load.

### Intelligent Placement Algorithm

**Scoring Factors:**
- Available RAM (40% weight)
- Available Storage (30% weight)
- CPU capability (15% weight)
- Current load (10% weight)
- Reliability/uptime (5% weight)

System scores all devices, selects highest score, returns device + reasoning.

### Database Pooling

Deploy one shared database instance per device instead of per-application. System provisions new database within shared instance, returns credentials to application.

**Resource Savings:**
- Traditional: 5 apps × 1GB Postgres = 5GB RAM
- Pooled: 1 shared Postgres with 5 databases = 1.2GB RAM (60% reduction)

### Resource Aggregation

Dashboard aggregates metrics across all devices: total RAM, storage, CPU cores, online/offline device count, total deployed apps.

## 2. System Architecture
┌─────────────────────────────────────────────────────────┐
│                    User's Browser                       │
│                  (React + TypeScript)                   │
└───────────────────────┬─────────────────────────────────┘
                        │ HTTPS / WebSocket
                        ▼
┌─────────────────────────────────────────────────────────┐
│              Control Plane (Go Backend)                 │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐ │
│  │   REST API  │  │  WebSocket   │  │  Job Queue    │ │
│  │   (Fiber)   │  │  (events)    │  │  (Asynq)      │ │
│  └─────────────┘  └──────────────┘  └───────────────┘ │
│  ┌─────────────────────────────────────────────────┐   │
│  │            Orchestration Engine                 │   │
│  │  - Device Manager                               │   │
│  │  - App Deployer (docker-compose generator)     │   │
│  │  - Network Manager (OPNsense API client)       │   │
│  │  - Backup Manager                               │   │
│  └─────────────────────────────────────────────────┘   │
│  ┌─────────────┐  ┌──────────────┐                    │
│  │  SQLite DB  │  │  Vault (sops)│ (credentials)      │
│  └─────────────┘  └──────────────┘                    │
└───────────────────────┬─────────────────────────────────┘
                        │ SSH / API calls
            ┌───────────┴──────────┬──────────────┐
            ▼                      ▼              ▼
    ┌──────────────┐      ┌──────────────┐   ┌─────────┐
    │  OPNsense    │      │ Docker Host  │   │ Switch  │
    │  (XML-RPC)   │      │ (SSH/API)    │   │ (SNMP)  │
    └──────────────┘      └──────────────┘   └─────────┘
Architecture Decisions
Single-Node Installation (MVP)

Control plane runs on user's primary server
Web UI accessible at http://homelab.local or configured domain
No clustering/HA in MVP (add later)

Agentless Design

No software to install on managed devices
Uses existing APIs (OPNsense XML-RPC, Docker API over SSH)
Reduces complexity and attack surface

Event-Driven for Real-Time Updates

WebSocket connection for live status updates
No polling for stats (push model)
Job queue for long-running tasks (deployments, backups)


## 3. Technology Stack

**Backend: Go**
- Single binary distribution
- Strong concurrency (goroutines)
- Low memory footprint (~20-50MB)
- Cross-compilation for ARM/x86
- Frameworks: Fiber (web), GORM (ORM), Viper (config), Zap (logging)

**Frontend: React + TypeScript**
- Vite (build tool)
- TanStack Query (server state)
- Tailwind CSS + shadcn/ui (styling/components)
- Recharts (dashboards)

**Database: SQLite (MVP) → PostgreSQL (scale)**
- Zero config, single file, easy backups
- GORM abstraction enables future Postgres migration

**Job Queue: In-Memory (MVP) → Asynq (scale)**
- Go channels + worker pool
- Zero dependencies maintains single binary
- Future: Redis + Asynq for persistence/multi-node

## 4. Repository Structure

**Monorepo Benefits:**
- Atomic API changes (frontend + backend in one commit)
- Type generation (Go → TypeScript)
- Embedded frontend (single binary via //go:embed)
- Simplified versioning

**Key Directories:**
```
backend/
  cmd/server/          # Entry point
  internal/            # Private code (api, services, models)
  pkg/                 # Reusable packages
  templates/           # App deployment templates

frontend/
  src/                 # React application
  public/              # Static assets

docs/                  # Documentation
scripts/               # Build and install scripts
```

**Build Process:**
- `make dev`: Run Go + Vite concurrently
- `make build`: Build frontend, embed in Go binary
- `make types`: Generate TypeScript from Go models (tygo)

## 5. Data Models
go// Core entities

type Device struct {
    ID           uuid.UUID      `gorm:"primaryKey"`
    Name         string         `gorm:"not null"`
    Type         DeviceType     `gorm:"not null"` // router, server, switch, nas
    IPAddress    string         `gorm:"not null"`
    MACAddress   string         
    Status       DeviceStatus   // online, offline, error
    Credentials  string         `gorm:"type:text"` // encrypted JSON
    Metadata     datatypes.JSON // device-specific config
    LastSeen     *time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type Application struct {
    ID              uuid.UUID `gorm:"primaryKey"`
    Name            string    // "Immich"
    Slug            string    // "immich"
    Category        string    // "photos"
    Description     string
    IconURL         string
    DockerImage     string    // "ghcr.io/immich-app/immich-server"
    RequiredRAM     int64     // bytes
    RequiredStorage int64     // bytes
    ConfigTemplate  string    `gorm:"type:text"` // docker-compose template
    SetupSteps      datatypes.JSON // post-deploy instructions
}

type Deployment struct {
    ID             uuid.UUID `gorm:"primaryKey"`
    ApplicationID  uuid.UUID
    Application    Application `gorm:"foreignKey:ApplicationID"`
    DeviceID       uuid.UUID
    Device         Device `gorm:"foreignKey:DeviceID"`
    Status         DeploymentStatus // deploying, running, stopped, error
    Config         datatypes.JSON   // user-provided config
    Domain         string           // "immich.home"
    InternalPort   int
    ExternalPort   int
    ContainerID    string           // Docker container ID
    DeployedAt     *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type NetworkConfig struct {
    ID            uuid.UUID `gorm:"primaryKey"`
    DeviceID      uuid.UUID
    Device        Device `gorm:"foreignKey:DeviceID"`
    VLANs         datatypes.JSON
    FirewallRules datatypes.JSON
    DNSRewrites   datatypes.JSON
    UpdatedAt     time.Time
}

type BackupJob struct {
    ID          uuid.UUID `gorm:"primaryKey"`
    Type        BackupType // config, volumes, full
    Status      JobStatus  // pending, running, completed, failed
    TargetPath  string
    FileSize    int64
    StartedAt   *time.Time
    CompletedAt *time.Time
    Error       string
    CreatedAt   time.Time
}

7. API Design
REST API (Fiber)
Base URL: http://homelab.local:8080/api/v1

Authentication: JWT tokens (future: support OAuth/OIDC)

Endpoints:

# Devices
GET    /devices                 # List all devices
POST   /devices                 # Add device
GET    /devices/:id             # Device details
PATCH  /devices/:id             # Update device
DELETE /devices/:id             # Remove device
GET    /devices/:id/stats       # Real-time stats
POST   /devices/discover        # Scan network

# Applications
GET    /applications            # App catalog
GET    /applications/:slug      # App details
POST   /applications/:slug/deploy  # Deploy app

# Deployments
GET    /deployments             # List deployments
GET    /deployments/:id         # Deployment details
POST   /deployments/:id/start   # Start containers
POST   /deployments/:id/stop    # Stop containers
POST   /deployments/:id/restart # Restart
DELETE /deployments/:id         # Remove deployment
GET    /deployments/:id/logs    # Container logs

# Network
GET    /network/status          # Overall network health
GET    /network/dns             # DNS rewrites
POST   /network/dns             # Add DNS rewrite
PATCH  /network/dns/:id         # Update rewrite
DELETE /network/dns/:id         # Delete rewrite
GET    /network/firewall        # Firewall rules summary

# Backups
GET    /backups                 # List backups
POST   /backups                 # Create backup
POST   /backups/:id/restore     # Restore backup
DELETE /backups/:id             # Delete backup

# System
GET    /system/health           # API health check
GET    /system/version          # Software version
POST   /system/update           # Update system
WebSocket API
javascript// Connection: ws://homelab.local:8080/ws

// Client subscribes to events
{
  "action": "subscribe",
  "channels": ["devices", "deployments", "logs"]
}

// Server pushes updates
{
  "channel": "devices",
  "event": "status_change",
  "data": {
    "device_id": "...",
    "status": "offline"
  }
}

{
  "channel": "deployments",
  "event": "progress",
  "data": {
    "deployment_id": "...",
    "step": "pulling_image",
    "progress": 45
  }
}

## 8. Deployment Principles

**Idempotency**: All operations check existing state before creating resources. Re-running deployment is safe.

**Atomic Rollback**: LIFO cleanup on failure. RollbackManager tracks actions, executes in reverse order.

**State Machine**:
```
validating → preparing → deploying → configuring → health_check → running
                                ↓ failure
                           rolling_back → rolled_back
```

**Deployment Flow:**
1. Pre-flight validation (resources, ports, Docker)
2. Infrastructure prep (reverse proxy if needed)
3. Generate docker-compose from template
4. Deploy via SSH (docker-compose up)
5. Configure networking (proxy rules, DNS)
6. Health check
7. Mark running or rollback

**Transparency**: Store generated compose, SSH commands, and rollback log for debugging.

## 9. Design Decisions Log

This section tracks key architectural decisions made during development.

### DD-001: Monorepo vs Multi-repo (2025-10-07)
**Decision**: Use monorepo structure
**Rationale**:
- Atomic changes across API contracts
- Type generation from Go → TypeScript
- Single binary distribution via embed
- Early stage velocity (both evolve together)

**Alternatives Considered**: Separate repos for frontend/backend
**Trade-offs**: Slightly larger clone size, need combined .gitignore
**Status**: Implemented

### DD-002: Job Queue - In-Memory vs Redis (2025-10-07)
**Decision**: Use in-memory queue for MVP, migrate to Asynq+Redis later
**Rationale**:
- Asynq requires Redis, breaks "single binary" promise
- In-memory sufficient for single control plane
- Go channels + worker pool simple to implement
- Abstract interface makes swapping backends easy

**Alternatives Considered**: Asynq from start, RabbitMQ, Temporal
**Trade-offs**: No job persistence, lost on restart (acceptable for MVP)
**Migration Path**: Add Redis when users need multi-node or persistence
**Status**: Implemented

### DD-003: Credential Storage - OS Keychain vs File Encryption (2025-10-07)
**Decision**: Use OS keychain (macOS Keychain, Linux Secret Service, Windows DPAPI)
**Rationale**:
- More secure than file-based master key
- Integrates with OS security model
- No single point of failure file

**Alternatives Considered**: mozilla/sops, HashiCorp Vault, encrypted file
**Trade-offs**: Platform-specific code, fallback to AES if keychain unavailable
**Status**: Planned

### DD-004: OPNsense API - REST vs XML-RPC (2025-10-07)
**Decision**: Use OPNsense REST API, not XML-RPC
**Rationale**:
- XML-RPC is legacy, being phased out
- REST API has better documentation
- More maintainable

**Alternatives Considered**: XML-RPC (original plan), screen scraping
**Trade-offs**: Requires newer OPNsense version (20.7+)
**Status**: Planned

### DD-005: Network Discovery - Manual First vs Auto-discovery (2025-10-07)
**Decision**: Manual IP entry as primary, mDNS/ARP as optional enhancement
**Rationale**:
- mDNS doesn't work across VLANs (common in homelabs)
- ARP scanning limited to same subnet
- Manual entry more reliable
- Discovery as convenience feature, not core

**Alternatives Considered**: mDNS/ARP as primary, requiring flat network
**Trade-offs**: Less "magical" but more reliable
**Status**: Planned

### DD-006: Deployment Rollback Strategy (2025-10-07)
**Decision**: Implement LIFO rollback manager with defer pattern
**Rationale**:
- Ensures cleanup on any failure
- Reverse order rollback (LIFO) naturally undoes dependencies
- Go's defer pattern makes it simple

**Implementation**: RollbackManager with action stack
**Status**: Planned

### DD-007: App Deployment Order (2025-10-07)
**Decision**: Start with simple apps (Vaultwarden, Uptime Kuma), add complex ones later
**Rationale**:
- Immich has complex setup (multiple containers, database, ML)
- Vaultwarden is single container, simple
- Learn from simple deployments first

**Original Plan**: Immich first
**Revised Plan**: Vaultwarden → Uptime Kuma → Jellyfin → Immich
**Status**: Planned

---

## 10. Development Phases

### Phase 0: Foundation [COMPLETE]
Monorepo structure, tooling, can run Hello World. Backend + frontend with hot-reload.

### Phase 1: Device Management [IN PROGRESS]
Device CRUD, SSH validation, Docker checks, basic dashboard, health monitoring.

### Phase 2: Intelligent Orchestration
IntelligentScheduler (resource scoring), resource aggregation service, database pooling, smart device recommendations.

### Phase 3: Deployment Engine
App repository, deployment pipeline with rollback, reverse proxy automation, real-time progress via WebSocket.

### Phase 4: Database Pooling
Shared Postgres/MySQL instances, database provisioning API, resource savings dashboard.

### Phase 5: Polish & Distribution
App migration, load rebalancing, OS keychain integration, binary releases, installation scripts, documentation.

### Phase 6: Automated Backups
restic integration, S3/NFS destinations, backup policies, scheduled backups, one-click restore.

### Post-MVP
Multi-user support, remote access (Tailscale), managed switch integration, Kubernetes support, monitoring/alerting.

## 11. Security

**Credential Storage:**
- OS keychain (macOS Keychain, Linux Secret Service, Windows DPAPI)
- AES-256-GCM fallback if keychain unavailable
- No plaintext credentials on disk

**API Security:**
- JWT authentication
- Rate limiting (100 req/min per IP)
- HTTPS required in production
- CORS restricted to same-origin

**Device Access:**
- SSH keys preferred over passwords
- Credentials never logged
- Audit log of configuration changes

## 12. Success Metrics

**Technical:**
- Install time: < 5 minutes
- First app deployed: < 10 minutes
- Memory usage: < 100MB
- API response time: < 200ms (p95)

**User Experience:**
- 80% deploy first app without docs
- 50% deploy 3+ apps in first week
- < 5% encounter errors during setup

## Summary

**Core Principles:**
1. Hide complexity without removing capability
2. Fail gracefully, rollback atomically
3. Idempotent by default
4. Manual first, automation second
5. Start simple, grow complex

**Design Priorities:**
- Simplicity: Single binary, SQLite, Docker-only
- Reliability: Rollback on failure, idempotent operations
- Security: OS keychain, encrypted credentials, HTTPS
- Transparency: Show generated configs, SSH commands
- Scalability: Can grow to Postgres, Redis, K8s later
