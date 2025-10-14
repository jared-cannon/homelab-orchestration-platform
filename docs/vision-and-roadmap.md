# Vision & Roadmap

**Version:** 2.0
**Last Updated:** October 2025

## Vision

Build unified multi-node homelab orchestration with intelligent resource management. The "Ubiquiti UniFi Controller for homelabs."

## Problem

Multi-device homelab management requires per-device SSH access, manual resource allocation, and independent container management. This creates:
- Fragmented device administration
- Resource waste (container sprawl: 5 separate database containers consuming 5GB+ RAM)
- Manual deployment decisions (often suboptimal)
- Complex Docker/networking knowledge requirements
- No unified resource visibility

## Solution

### Unified Orchestration

Treat distributed homelab devices (Raspberry Pis, servers, NAS) as single unified system:
- Aggregate resource monitoring
- Automatic device selection
- Resource sharing (database pooling, shared caches)
- Cross-device migration

### Key Capabilities

**Intelligent Placement**

Device analysis: RAM, storage, CPU, current load. Automatic selection via scoring algorithm with manual override.

Example deployment:
```
Analyzing homelab...
Server-01: Score 72 (4GB RAM free, 85% loaded)
Server-02: Score 95 (8GB RAM free, 40% loaded, SSD) ← Selected
Pi-4:      Score 58 (2GB RAM free, SD card)

Deploying to Server-02 (optimal: 8GB RAM free, fast storage)
```

**Database Pooling**

Traditional: 5 apps × 5 database containers = 5GB RAM

Unified: 1 shared Postgres container = 1.5GB RAM (70% reduction)

Auto-provisioning:
1. Deploy NextCloud
2. Check for Postgres on target device
3. Deploy shared Postgres if absent
4. Create database `nextcloud_db`
5. Generate secure credentials
6. Inject into NextCloud configuration

**Infrastructure View**

Unified dashboard:
```
Your Homelab (3 devices online)
RAM:     ████████░░░░  28GB / 32GB
Storage: ████░░░░░░░░  800GB / 2TB
CPU:     ███████░░░░░  16 cores, 58% utilization

12 apps running
Savings from resource sharing: 2.8GB
```

**Automated Encrypted Backups**

- Client-side encryption (AES-256)
- Content-defined chunking deduplication
- Multi-destination support (S3-compatible + local NAS)
- Per-app scheduling with retention policies
- One-click restore

Storage savings: 85% reduction via deduplication (7 days × 200GB → 228GB)

## Competitive Differentiation

| Feature | CasaOS/Umbrel | Coolify | Portainer | Proxmox | K8s | This Platform |
|---------|---------------|---------|-----------|---------|-----|---------------|
| Multi-node orchestration | ❌ | ❌ | Manual | Manual | ✅ | ✅ |
| Intelligent placement | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| Database pooling | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ |
| Unified resource view | ❌ | ❌ | Per-device | Per-VM | ✅ | ✅ |
| Zero-config deployment | ✅ | Partial | ❌ | ❌ | ❌ | ✅ |
| Automated backups | ❌ | ❌ | ❌ | Manual | Manual | ✅ |
| Complexity level | Low | Medium | Medium | High | Very High | Low |

**Gap Analysis**

Before Ubiquiti: Complex enterprise gear (Cisco) OR consumer junk (Linksys)

Before This: Complex tools (Proxmox, K8s) OR simple single-node tools (CasaOS)

Positioning: Kubernetes-level intelligence with CasaOS-level simplicity.

## Target Users

**Primary Audience**
- Multi-device homelab operators (3+ devices)
- SSH-comfortable users
- Efficiency-focused (RAM optimization via pooling)
- Unified management preference over Kubernetes complexity

**Example Use Cases**
- Homelab enthusiast: 2 servers + 3 Raspberry Pis
- Self-hoster: 10+ services across multiple devices
- Small team: shared family/small business infrastructure

**Not For**
- Enterprise (use Kubernetes, Rancher, OpenShift)
- Single device (use CasaOS, Umbrel)
- Beginners (start with Synology)

## Technical Architecture

**Design Principles**

- Agentless: SSH + Docker API
- Intelligent scheduler: Resource scoring algorithm
- Database pooling: Shared instances with auto-provisioning
- Single binary: Go backend, embedded React frontend
- Multi-network aware: VLAN/subnet support
- Recipe-based: docker-compose.yaml + manifest.yaml (programmatically enhanced)

**Implementation Status**

- ✅ Device discovery and management
- ✅ Recipe marketplace (20+ apps)
- ✅ Single-device deployment
- 🚧 Intelligent resource scoring
- 🚧 Shared database infrastructure
- 🚧 Cross-device resource aggregation

Reference: [architecture.md](architecture.md), [intelligent-orchestration.md](intelligent-orchestration.md)

## Current Status

### ✅ Phase 0: Foundation (Completed)
- Device scanning and discovery
- Multi-device dashboard with real-time status
- SSH credential management and testing
- Device resource monitoring (CPU, RAM, disk, Docker)
- WebSocket real-time updates
- Marketplace recipe system (YAML-based)
- Marketplace UI with search/filtering
- Recipe detail pages
- Deployment wizard (basic device selection, validation)
- Single-device Docker Compose deployment

### 🚧 Phase 1: Multi-Node Intelligence (In Progress)

**Priority 1: Intelligent Placement**
- [ ] Resource scoring algorithm (RAM 40%, storage 30%, CPU 15%, load 10%, uptime 5%)
- [ ] `IntelligentScheduler.SelectOptimalDevice(app)` returns best device + reasoning
- [ ] Automatic device selection with manual override
- [ ] "Analyzing your homelab..." → "Recommended: Server-02 (8GB RAM free, 40% load)"

**Priority 2: Resource Aggregation**
- [ ] Cross-device resource monitoring service
- [ ] `GetAggregateResources()` returns total RAM/CPU/storage across all devices
- [ ] Poll all devices every 30 seconds
- [ ] Unified dashboard: "Your homelab: 24GB RAM (60% used across 3 devices)"
- [ ] Total apps deployed, devices online/offline
- [ ] Highlight overloaded devices

**Priority 3: Database Pooling Foundation**
- [ ] Detect if Postgres/MySQL already running on device
- [ ] Deploy shared database instance if needed
- [ ] API for provisioning database/user in shared instance
- [ ] Automatic database provisioning from recipes

---

## Roadmap

## Phase 2: Intelligent Deployment UX (Q1 2026)

**Goal:** Surface multi-node intelligence through UI

### Smart Deployment Wizard

**Step 1: App Selection**
- Browse marketplace
- View app requirements

**Step 2: Intelligent Device Selection (auto-selected)**
```
🔍 Analyzing your homelab...

✅ Recommended: Server-02 (Score: 95/100)
   • 8GB RAM available (app needs 2GB)
   • Fast SSD storage
   • Current load: 40% (plenty of headroom)
   • Uptime: 99.8%

Override? [Deploy to a different device ▼]
   • Server-01: Score 72 (4GB RAM, but 85% loaded)
   • Pi-4: Score 58 (2GB RAM, SD card storage)
```

**Step 3: Resource Preview**
- "NextCloud will use existing Postgres instance → Saves 1GB RAM"
- OR "No Postgres found → Will deploy shared instance (1.2GB RAM)"

**Step 4: Configuration**
- Most apps: Zero config
- Advanced users: Show config options

**Step 5: Real-time Deployment**
- WebSocket progress updates
- "Deploying shared Postgres... ✓"
- "Creating database nextcloud_db... ✓"
- "Starting NextCloud... ✓"

### Unified Dashboard

**Aggregate Resource View**
```
Your Homelab (3 devices online)
┌────────────────────────────────────┐
│ RAM:     ████████░░░░  24GB / 32GB │
│ Storage: ████░░░░░░░░  800GB / 2TB │
│ CPU:     ███████░░░░░  58% avg     │
└────────────────────────────────────┘

12 apps running
Saved 2.8GB RAM by sharing databases
```

**Device Health Cards**
- Green: < 70% utilized
- Yellow: 70-90% utilized
- Red: > 90% utilized
- Suggested actions for overloaded devices

**Quick Actions**
- Deploy new app
- View all deployments
- Migrate overloaded apps

### Device Comparison View

- Side-by-side device comparison
- Real-time resource availability
- Show scores for current app
- Visual indicators: ✅ Recommended, ⚠️ Acceptable, ❌ Insufficient
- Click any device to override automatic selection

---

## Phase 3: Database Pooling & Resource Optimization (Q2 2026)

**Goal:** Prove 60% RAM savings value proposition

### Shared Database Infrastructure

**Problem:** Container sprawl (5 apps = 5 Postgres = 5GB RAM)

**Solution:** One shared Postgres instance per device

**Features:**
- Automatic shared instance deployment
- Recipe-based (postgres-shared.yaml)
- Health monitoring
- Automatic restart on crash

**Database Provisioning API**
```go
DatabasePool.ProvisionDatabase(appName, deviceID)
→ Creates database: appname_db
→ Creates user: appname_user
→ Generates secure password
→ Returns connection credentials
```

**Recipe Integration**
- Apps declare: `requires_database: postgres`
- System checks: "Is Postgres running on target device?"
- If no: Deploy shared instance
- If yes: Provision new database in existing instance
- Inject credentials into docker-compose template

**Resource Savings Dashboard**
```
💡 Resource Optimization

Database Sharing Savings:
─────────────────────────────────
Before: 5 apps × 5 Postgres = 5GB RAM
After:  5 apps → 1 Postgres = 1.5GB RAM

You're saving 3.5GB RAM (70%)
─────────────────────────────────

Apps using shared Postgres on Server-02:
• NextCloud (nextcloud_db)
• Monica (monica_db)
• Paperless (paperless_db)
• n8n (n8n_db)
• Immich (immich_db)
```

### Redis/Cache Pooling

- Same concept as database pooling
- Shared Redis instance per device
- Apps use shared cache
- Further RAM savings

### Resource Optimization Recommendations

- Analyze current deployments
- Suggest optimizations:
  - "3 apps using separate Postgres instances → Migrate to shared instance? (Save 2GB)"
  - "Vaultwarden has 3GB RAM allocated but only uses 500MB → Reduce allocation?"

---

## Phase 4: Advanced Multi-Device Features (Q3 2026)

### One-Click App Migration

**Problem:** Moving app between devices requires manual backup/restore

**Migration Wizard**
```
Step 1: Select app to migrate
Step 2: Select destination device
Step 3: Validation (does destination have resources?)
Step 4: Execute migration:
  - Export volumes from source
  - Transfer data via SSH/SCP
  - Deploy to destination with same config
  - Health check on destination
  - Optionally remove from source
```

**Smart Migration Recommendations**
- "Jellyfin using 95% of Pi-4 RAM → Migrate to NAS?"
- "Database on Pi with limited storage → Migrate to NAS?"

**Bulk Migration**
- "Migrate ALL apps from Pi-Zero to Pi-4"
- Useful when upgrading/replacing hardware

### Cross-Device Deployments

**Problem:** Some apps need multiple components on different devices

**Multi-Device Recipes**
```yaml
# recipes/nextcloud-distributed.yaml
name: Nextcloud (Distributed)
components:
  - name: database
    device_criteria:
      min_ram_gb: 2
      min_storage_gb: 50
      prefer: "storage-optimized"  # Deploy to NAS

  - name: web
    device_criteria:
      min_ram_gb: 1
      prefer: "always-on"  # Deploy to Pi

networking:
  auto_link: true  # Automatically configure DB connection
```

**Automatic Cross-Device Networking**
- Configure web app with database IP automatically
- Support WireGuard/Tailscale for secure inter-device communication
- Handle service discovery across devices

### Device Groups & Batch Operations

**Problem:** Performing actions across multiple devices is tedious

**Device Group Management**
- Create groups: "Pi Cluster", "Storage Nodes", "Edge Devices"
- Group-level operations:
  - "Deploy monitoring to ALL devices in group"
  - "Restart Docker on ALL Pi devices"
  - "Update Traefik on ALL devices"

**Batch Deployment**
- "Deploy Uptime Kuma to all devices for monitoring"
- "Deploy Traefik to all edge devices"
- Visual progress: "Deployed 3/5 devices, Pi-Zero failed"

**Group Dashboards**
- Aggregate stats for device group
- Compare performance across group members

---

## Phase 5: Intelligence & Automation (Q4 2026)

### Automated Load Balancing

**Problem:** Apps deployed once and never moved, even if device becomes overloaded

**Auto-Scaling Across Devices**
- Detect device overload (> 90% RAM)
- Suggest migrating least-critical app to another device
- Optionally auto-migrate with user approval

**Round-Robin Deployment**
- "Deploy 10 instances of this app"
- System distributes across all capable devices automatically

### Predictive Recommendations

**Usage Pattern Analysis**
- "Jellyfin CPU spikes every evening 7-10pm"
- "Database RAM usage growing 5% per month → will exceed capacity in 6 months"

**Proactive Suggestions**
- "Pi-4 temperature consistently > 70°C → Consider adding cooling or migrating apps"
- "NAS storage 80% full → Suggest cleanup or expansion"

### Disaster Recovery

**Automatic Backups**
- Schedule volume backups for all deployments
- Cross-device backup (backup Pi apps to NAS)

**One-Click Restore**
- "Pi-Zero died → Restore all 3 apps to Pi-4"

---

## Phase 6: Community & Ecosystem (2027+)

### Device Templates

**Pre-configured device profiles**
- "Raspberry Pi 4B - Media Server"
- "Synology NAS - Storage + Services"
- "Mini PC - Compute Node"

**One-click setup**
- "Make this Pi a media server" → auto-installs Jellyfin, Sonarr, Radarr, Traefik

### Homelab Sharing & Templates

- Export homelab configuration
- Share recipe combinations: "My Perfect Homelab Stack"
- Import someone else's setup

### Advanced Networking

- Built-in VPN mesh (WireGuard/Tailscale integration)
- Network performance monitoring between devices
- Bandwidth usage tracking per app

---

## Success Metrics

### MVP Success Criteria

**Phase 1: Multi-Node Intelligence**
- Deploy app without manual device selection
- Aggregate resource display
- Resource scoring with reasoning

**Phase 2: Intelligent Deployment UX**
- Deployment wizard shows recommended device with score
- Unified resource dashboard
- Manual override capability

**Phase 3: Database Pooling**
- 3+ apps share single Postgres instance
- Dashboard displays RAM savings
- Zero manual database configuration

**Phase 4: Automated Encrypted Backups**
- S3-compatible destination configuration
- Default policy auto-applies to apps
- Daily encrypted backups with deduplication
- Snapshot browsing and file restoration
- Storage savings metrics

**Phase 5: Cross-Device Migration**
- One-click app migration between devices
- Automatic migration suggestions for overloaded devices
- Automatic data transfer

### User Experience Goals

**Target Workflow**
1. 3 devices connected
2. Dashboard: "24GB RAM across 3 devices"
3. Click "Deploy NextCloud"
4. System: "Recommended: Server-02 (8GB free)"
5. Click "Deploy" (zero configuration)
6. 2 minutes: "NextCloud ready at http://nextcloud.home"
7. Deploy 2 more apps
8. Dashboard: "Saved 2GB RAM via shared Postgres"

### Technical Metrics

- Install time: < 5 minutes
- First app deployed: < 10 minutes
- Dashboard load time: < 2 seconds
- Memory usage: < 100MB (backend)
- API response time: < 200ms (p95)
- Device discovery success rate: > 95%
- Deployment success rate: > 90%
- Average deployment time: < 2 minutes
- Cross-device migration success: > 85%

### User Adoption

- Target: 1,000 active homelabs by end of 2026
- Key metric: Average devices per user (goal: 3+)
- NPS score: > 50

### Community

- Custom recipes submitted: 50+
- GitHub stars: 1,000+
- Active contributors: 20+

---

## Scope Boundaries

### Excluded from MVP

- VM management (use Proxmox)
- Kubernetes support (use Rancher)
- Hardware vendor model
- App store marketplace (use Docker images)
- Mobile app (web UI sufficient)
- Cloud hosting (self-hosted only)

### Deferred Post-MVP

- Cross-subnet device discovery (Phase 1: manual IP)
- Multi-user RBAC (Phase 1: single admin)
- Advanced monitoring/alerting (Phase 1: basic health checks)

---

## Go-To-Market Strategy

### Phase 1: MVP Launch (Month 1-6)

**Goal:** Prove concept with early adopters

**Target:** r/selfhosted community (100k+ multi-device users)

**Approach:**
1. r/selfhosted post
2. Demo video: 5 apps across 3 devices, RAM savings
3. GitHub release with install script
4. Discord community

**Success Metric:** 100 active installs within 3 months

### Phase 2: Community Growth (Month 7-12)

**Goal:** Build momentum, refine based on feedback

**Activities:**
- YouTube tutorials
- Self-hosting forum posts
- Popular app integrations (Home Assistant, Plex)
- Community recipe contributions

**Success Metric:** 1,000 active installs, 500 GitHub stars

### Phase 3: Ecosystem Expansion (Year 2+)

**Opportunities:**
- Hardware partnerships (pre-configured nodes)
- Managed hosting option
- Enterprise tier (RBAC, advanced features)

---

## Market Opportunity

**Gap**
- 100k+ r/selfhosted users with multiple devices
- No unified multi-device homelab management
- Container sprawl wastes RAM
- Manual device selection causes poor utilization

**Value Proposition**
- Intelligent multi-node orchestration
- Automatic database pooling (60-70% RAM savings)
- Automated encrypted backups (85% storage savings)
- Unified infrastructure view
- Zero-configuration deployment

**Positioning**
- "Ubiquiti UniFi Controller for homelabs"
- Power user segment (3+ devices)
- Community-driven efficient homelab management
- Future: Hardware ecosystem (like UniFi Dream Machine)

---

## Key Design Principles

1. **Unified Infrastructure Model**: "Homelab has 24GB RAM" not "Server-01 has 8GB, Server-02 has 16GB"
2. **Intelligence Over Configuration**: Smart defaults, manual override optional
3. **Efficiency Through Sharing**: Database pooling, shared Redis, unified reverse proxy
4. **Transparency + Control**: Show generated docker-compose.yml, explain decisions
5. **Progressive Disclosure**: Beginner → click deploy; Advanced → full control
