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

## The "Ubiquiti for Homelabs" Vision

Ubiquiti succeeded by making enterprise networking accessible to prosumers. We're doing the same for homelab orchestration.

### What Ubiquiti Did for Networking

**Before Ubiquiti:**
- Complex enterprise gear (Cisco, Juniper) OR consumer junk (Linksys)
- Nothing in between

**Ubiquiti's Innovation:**
- Enterprise features with consumer-friendly UI
- Unified management (UniFi Controller = single pane of glass)
- Beautiful hardware + beautiful software
- Accessible pricing

### What We're Doing for Homelabs

**Before This Tool:**
```
Power User: Proxmox + Kubernetes + manual Docker
            (Complex but powerful)
              ↓ [Big Gap] ↓
Beginner:   Synology apps, Raspberry Pi projects
            (Simple but limited)
```

**With This Tool:**
```
┌───────────────────────────────────────────────────┐
│ Anyone can run a production-grade homelab         │
│ - Simple as CasaOS                                │
│ - Powerful as Proxmox                             │
│ - Intelligent like Kubernetes                     │
│ - Unified like UniFi Controller                   │
└───────────────────────────────────────────────────┘
```

### Core Principles

1. **Unified Infrastructure Model**: Your homelab is ONE system with distributed resources
   - Not "3 Raspberry Pis and 2 servers"
   - But "My homelab has 24GB RAM, 12 CPU cores, 2TB storage"

2. **Intelligent Orchestration**: System decides optimal placement
   - User: "Deploy NextCloud"
   - System: "Analyzing... Server-02 has 8GB RAM free and SSD storage → deploying there"

3. **Resource Sharing**: Eliminate container sprawl
   - Instead of: 5 apps × 5 separate Postgres containers = 5GB RAM
   - We do: 5 apps → 1 shared Postgres instance = 1.5GB RAM (saves 3.5GB)

4. **Infrastructure Abstraction**: User doesn't manage devices, they manage their homelab
   - Not: "Deploy to which device?"
   - But: "Deploy" (system handles placement)

5. **Complexity Hidden, Not Removed**: Power users retain full control
   - Show generated docker-compose.yml
   - Export configs
   - Override intelligent placement
   - 90% zero-config, 10% full control

### What Makes Us Different

| Feature | CasaOS / Coolify | Proxmox | Us |
|---------|------------------|---------|-----|
| **Multi-node** | ❌ Single node | ✅ Yes | ✅ **Unified orchestration** |
| **Intelligent placement** | ❌ No | ❌ Manual | ✅ **Automatic** |
| **Resource sharing** | ❌ No | ⚠️ Manual | ✅ **Automatic database pooling** |
| **Unified view** | ❌ Per-device | ⚠️ Cluster view | ✅ **Aggregate resources** |
| **Simplicity** | ✅ Very simple | ❌ Complex | ✅ **Simple + powerful** |
| **Target user** | Beginners | Power users | **Power users who want simplicity** |

## 1.1 Core Differentiator: Intelligent Orchestration

This is what sets us apart from every other homelab tool. Not just multi-node management, but **intelligent multi-node orchestration**.

### Intelligent Placement Algorithm

When user clicks "Deploy NextCloud", the system:

```go
type IntelligentScheduler struct {
    deviceService   *DeviceService
    resourceMonitor *ResourceMonitor
}

func (s *IntelligentScheduler) SelectOptimalDevice(app *Recipe) (*Device, *PlacementScore, error) {
    // 1. Get all available devices
    devices := s.deviceService.ListDevices()

    // 2. Score each device
    scores := []PlacementScore{}
    for _, device := range devices {
        score := s.calculatePlacementScore(app, device)
        scores = append(scores, score)
    }

    // 3. Sort by score (highest = best)
    sort.Sort(sort.Reverse(scores))

    // 4. Return best device with reasoning
    return scores[0].Device, &scores[0], nil
}

func (s *IntelligentScheduler) calculatePlacementScore(app *Recipe, device *Device) PlacementScore {
    score := PlacementScore{Device: device}

    // Factor 1: Available RAM (40% weight)
    ramAvailable := device.TotalRAM - device.UsedRAM
    if ramAvailable >= app.RecommendedRAM {
        score.RAMScore = 100
    } else if ramAvailable >= app.MinRAM {
        score.RAMScore = 50
    } else {
        score.RAMScore = 0 // Disqualify
    }

    // Factor 2: Available Storage (30% weight)
    storageAvailable := device.TotalStorage - device.UsedStorage
    if storageAvailable >= app.RecommendedStorage {
        score.StorageScore = 100
    } else if storageAvailable >= app.MinStorage {
        score.StorageScore = 50
    } else {
        score.StorageScore = 0 // Disqualify
    }

    // Factor 3: CPU Capability (15% weight)
    // Prefer more powerful CPUs for resource-intensive apps
    score.CPUScore = (device.CPUCores / app.RecommendedCPUCores) * 100

    // Factor 4: Current Load (10% weight)
    // Prefer devices with lower current load
    loadPercentage := (device.UsedRAM / device.TotalRAM) * 100
    score.LoadScore = 100 - loadPercentage

    // Factor 5: Reliability (5% weight)
    // Prefer devices with high uptime
    score.ReliabilityScore = device.UptimePercentage

    // Calculate weighted total
    score.TotalScore = (score.RAMScore * 0.40) +
                       (score.StorageScore * 0.30) +
                       (score.CPUScore * 0.15) +
                       (score.LoadScore * 0.10) +
                       (score.ReliabilityScore * 0.05)

    // Generate reasoning
    score.Reasoning = fmt.Sprintf(
        "Selected %s: %dGB RAM available, %dGB storage, %.0f%% current load",
        device.Name, ramAvailable/1024/1024/1024, storageAvailable/1024/1024/1024, loadPercentage,
    )

    return score
}
```

**User Experience:**
```
User: "Deploy Vaultwarden"

System analyzes:
✓ Server-01: Score 85 (4GB RAM free, but 80% loaded)
✓ Server-02: Score 95 (8GB RAM free, only 40% loaded) ← Selected
✓ Pi-4:      Score 60 (2GB RAM free, sufficient but lower reliability)

System: "Deploying to Server-02 (best match: 8GB RAM available, 40% current load)"
```

### Database Pooling Architecture

Instead of each app deploying its own database container, we intelligently share database instances:

```go
type DatabasePool struct {
    postgresInstances map[uuid.UUID]*DatabaseInstance // deviceID -> instance
    mysqlInstances    map[uuid.UUID]*DatabaseInstance
}

func (p *DatabasePool) GetOrCreatePostgresInstance(deviceID uuid.UUID) (*DatabaseInstance, error) {
    // Check if Postgres already running on this device
    if instance, exists := p.postgresInstances[deviceID]; exists {
        return instance, nil
    }

    // Deploy shared Postgres instance
    instance := &DatabaseInstance{
        DeviceID:     deviceID,
        Type:         "postgres",
        ContainerID:  generateContainerID(),
        Databases:    []string{},
    }

    // Deploy using Docker Compose
    if err := p.deployPostgresContainer(deviceID); err != nil {
        return nil, err
    }

    p.postgresInstances[deviceID] = instance
    return instance, nil
}

func (p *DatabasePool) ProvisionDatabase(app *Deployment, dbType string) (*DatabaseCredentials, error) {
    // 1. Get or create shared database instance
    instance, err := p.GetOrCreatePostgresInstance(app.DeviceID)
    if err != nil {
        return nil, err
    }

    // 2. Create new database for this app
    dbName := fmt.Sprintf("%s_db", app.RecipeSlug)
    dbUser := fmt.Sprintf("%s_user", app.RecipeSlug)
    dbPassword := generateSecurePassword()

    // 3. Execute SQL commands
    commands := []string{
        fmt.Sprintf("CREATE DATABASE %s;", dbName),
        fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", dbUser, dbPassword),
        fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, dbUser),
    }

    for _, cmd := range commands {
        if err := instance.ExecuteSQL(cmd); err != nil {
            return nil, err
        }
    }

    // 4. Return credentials for app
    return &DatabaseCredentials{
        Host:     instance.InternalIP,
        Port:     5432,
        Database: dbName,
        Username: dbUser,
        Password: dbPassword,
    }, nil
}
```

**Result:**
```
Before (Traditional Docker Compose):
- NextCloud → Postgres container (1GB RAM)
- Monica → Postgres container (1GB RAM)
- Paperless → Postgres container (1GB RAM)
Total: 3GB RAM

After (Our System):
- Shared Postgres container (1.2GB RAM)
  ├── nextcloud_db
  ├── monica_db
  └── paperless_db
Total: 1.2GB RAM (saves 1.8GB = 60% reduction)
```

### Cross-Device Resource Aggregation

Unified dashboard shows total resources across all devices:

```go
type AggregateResources struct {
    TotalRAM       uint64 // Sum of all device RAM
    UsedRAM        uint64 // Sum of all used RAM
    TotalStorage   uint64 // Sum of all device storage
    UsedStorage    uint64 // Sum of all used storage
    TotalCPUCores  int    // Sum of all CPU cores
    AvgCPUUsage    float64 // Average CPU usage across devices

    Devices        []Device
    OnlineDevices  int
    OfflineDevices int
    TotalApps      int
}

func (s *ResourceMonitor) GetAggregateResources() (*AggregateResources, error) {
    devices := s.deviceService.ListDevices()
    agg := &AggregateResources{}

    for _, device := range devices {
        if device.Status == DeviceStatusOnline {
            agg.TotalRAM += device.TotalRAM
            agg.UsedRAM += device.UsedRAM
            agg.TotalStorage += device.TotalStorage
            agg.UsedStorage += device.UsedStorage
            agg.TotalCPUCores += device.CPUCores
            agg.AvgCPUUsage += device.CPUUsage
            agg.OnlineDevices++
        } else {
            agg.OfflineDevices++
        }
    }

    agg.AvgCPUUsage /= float64(agg.OnlineDevices)
    agg.Devices = devices

    return agg, nil
}
```

**Dashboard Display:**
```
┌─────────────────────────────────────────────────┐
│  Homelab Overview                               │
├─────────────────────────────────────────────────┤
│  Total Resources Across 3 Devices:              │
│  ┌──────────────────────────────────────────┐   │
│  │ RAM:     ████████░░░░  24GB / 32GB       │   │
│  │ Storage: ████░░░░░░░░  500GB / 2TB       │   │
│  │ CPU:     ███████░░░░░  12 cores, 65% avg │   │
│  └──────────────────────────────────────────┘   │
│                                                  │
│  12 apps running • 3 devices online              │
└─────────────────────────────────────────────────┘
```

2. Functional Requirements (MVP)
2.1 Device Management

 Auto-discover devices on local network (mDNS, ARP scanning)
 Manual device addition (IP, credentials, device type)
 Support device types: OPNsense router, Linux server (Docker host)
 Health monitoring (ping, service status)
 Credential storage (encrypted)

2.2 Network Control (OPNsense Integration)

 Read-only dashboard: WAN status, firewall activity, DHCP leases
 DNS management via AdGuard Home plugin

Add/edit DNS rewrites
View blocked queries
Manage blocklists


 Basic VLAN visualization (future: creation/editing)

2.3 Application Deployment

 Curated app catalog (10 apps for MVP):

Immich (photos)
Vaultwarden (passwords)
Nextcloud (files/calendar)
Jellyfin (media)
Home Assistant (smart home)
Uptime Kuma (monitoring)
AdGuard Home (already on router, but optional server instance)
NGINX Proxy Manager (reverse proxy)
Portainer (Docker GUI backup)
Paperless-ngx (documents)


 One-click deployment flow:

Select app from catalog
Choose target server
Configure basic settings (ports, domain name)
Deploy (docker-compose + reverse proxy + DNS)
Show success with direct link


 Automatic infrastructure setup:

Deploy NGINX Proxy Manager if not present
Generate SSL certs (self-signed or Let's Encrypt)
Create DNS rewrites in AdGuard
Open firewall ports (optional, ask user)



2.4 Unified Dashboard

 System health overview (all nodes)
 Network statistics (bandwidth, active devices)
 Service status cards (green/yellow/red)
 Resource usage graphs (CPU, RAM, disk per node)
 Quick actions (restart service, view logs, update)

2.5 Backup & Configuration

 Export configuration as JSON
 Backup docker volumes on schedule
 One-click restore to previous state
 Configuration versioning (git-like)


3. Non-Functional Requirements

Performance: Dashboard loads in < 2s, actions complete in < 10s
Reliability: System continues working if one node goes down
Security: All credentials encrypted at rest, HTTPS only, RBAC for multi-user
Usability: Non-technical users can deploy first app in < 10 minutes (**"Mom Test"**)
Portability: Runs on any Linux x86_64/ARM64, minimal dependencies
Observability: All actions logged, errors surfaced clearly
Simplicity: **"If a non-technical user can't go from zero to running Nextcloud in < 10 minutes, we've failed"**


3.5 Ubiquiti Simplicity Philosophy & UX Principles

## Core Principle: The "Mom Test"
> "If your mom can't deploy Nextcloud without calling you, the UX has failed."

Ubiquiti succeeded by making enterprise networking simple enough for prosumers. We aim to do the same for homelab orchestration. This isn't about dumbing down features — it's about **progressive disclosure** and **sensible defaults**.

### Current Simplicity Scorecard

**✅ What's Simple:**
- Single binary installation (no docker, no k8s to install first)
- Zero-config database (SQLite auto-creates)
- Secure credential storage (OS keychain with encrypted fallback)
- Clean architecture (models → services → API is obvious)
- Standard tech stack (GORM, Fiber, TanStack Query)

**⚠️ Partially Simple:**
- Device addition requires SSH knowledge
- No discovery, must know IP address
- Error messages need more context

**❌ Not Simple Enough:**
- **No first-run wizard** - User sees empty device list, no guidance
- **No device discovery** - Ubiquiti finds devices automatically
- **No deployment engine yet** - Can't prove simplicity
- **No real-time feedback** - Just spinners, no progress
- **No self-healing** - Errors are dead ends
- **Jargon everywhere** - "SSH credentials", "pre-flight validation"

### Gap vs Ubiquiti UX

| Feature | Ubiquiti | Us (Current) | Us (Goal) |
|---------|----------|--------------|-----------|
| First Run | Setup wizard | Empty device list | Friendly wizard |
| Device Discovery | Automatic (mDNS) | Manual IP entry | Scan + manual fallback |
| Error Handling | "Fix this" button | Error toast | Fix suggestion + action |
| Progress | Real-time % | Spinner only | % complete + ETA |
| Default Config | Works out of box | User configures | Smart defaults + override |
| Language | Plain English | Technical jargon | "Mom-friendly" |

### UX Principles

#### 1. Progressive Disclosure
**Principle:** Start simple, reveal complexity on demand.

**Implementation:**
- **Basic Mode (default):** Hides SSH keys, Docker commands, YAML
- **Advanced Mode (opt-in):** Shows "What I Did" debug view, SSH commands
- **Expert Mode:** Full control, raw compose editing, manual overrides

**Example:**
```
Basic:  [Device Name] [IP Address] [Username] [Password] → Deploy
Advanced: + Show SSH Key Auth | Show Generated Docker Compose
Expert: Edit Raw YAML | Override Port Mappings | Custom Networks
```

#### 2. Clear, Actionable Errors
**Principle:** Every error must explain why AND how to fix.

**Bad:**
```
❌ Error: Connection refused (errno 111)
```

**Good:**
```
❌ Can't connect to device
💡 Possible fixes:
   • Is the device powered on?
   • Check IP address is correct (currently: 192.168.1.100)
   • Ensure SSH is enabled on port 22
   [Test Connection Again]  [Change IP Address]
```

#### 3. Real-Time Feedback
**Principle:** User should always know what's happening and how long it will take.

**Implementation:**
- WebSocket progress updates during deployment
- Progress bar with current step: "Pulling Docker image... 45% (2min remaining)"
- Not just "Deploying..." with spinner

#### 4. Self-Healing & Rollback
**Principle:** System should recover automatically or guide user to recovery.

**Implementation:**
- Auto-retry on transient failures (3 attempts)
- Automatic rollback on deployment failure (LIFO cleanup)
- Clear recovery instructions: "Deployment failed. Rolled back to previous state. Want to try again?"

#### 5. Language Simplification
**Principle:** No jargon unless user opts into "Advanced Mode".

**Translations:**
| Technical Term | Mom-Friendly Term |
|----------------|-------------------|
| SSH credentials | Device login |
| Pre-flight validation | Checking requirements |
| Container orchestration | App management |
| Deployment status: validating | Making sure device is ready |
| Port mapping | Which port to use |
| docker-compose | (hide entirely in Basic mode) |

#### 6. First-Run Wizard
**Principle:** New users see a clear path to success, not a blank page.

**Flow:**
```
Step 1: Welcome
  "Welcome to Homelab Orchestration! Let's add your first device."
  [Big friendly "Get Started" button]

Step 2: Find Your Device
  Option A: "Scan my network" → Shows discovered devices
  Option B: "I know the IP address" → Manual entry

Step 3: Test Connection
  Auto-test SSH on port 22
  ✓ Connected! / ❌ Can't connect (show fixes)

Step 4: Check Requirements
  ✓ Docker is installed and running
  or
  ❌ Docker not found
     [Install Docker for me] [Skip for now]

Step 5: Success
  "Your device is ready! Want to deploy your first app?"
  Card: Nextcloud - Your personal cloud (Recommended for beginners)
  [Browse All Apps]
```

### Mom Test Success Criteria

Before marking Phase 1 complete, we must pass ALL of these:

- [ ] **First-run wizard guides through device setup** (no blank pages)
- [ ] **Device can be added without knowing "SSH" exists** (just "device login")
- [ ] **Connection test shows clear results** (✓ Connected or ❌ with fix steps)
- [ ] **All errors have actionable fix suggestions** (not just "errno X")
- [ ] **No jargon in basic mode** (SSH, Docker, compose hidden)
- [ ] **Can add device in < 3 minutes** (including finding IP)
- [ ] **Real-time progress for long operations** (not just spinners)
- [ ] **Clear next steps after every action** ("Device ready! Deploy an app?")

### Testing the "Mom Test"
1. Have a non-technical user attempt full flow
2. Note every question they ask → UX failed there
3. Note every error they can't fix → needs better error message
4. If they give up → critical UX blocker
5. If they succeed without help → UX passed!

### Phase Priorities with Simplicity Focus

**Phase 1 (Current):**
- ✅ Device CRUD, SSH validation, Docker checks
- ✅ Comprehensive tests (62.7% API coverage, real integrations)
  - Device service tests (in-memory SQLite)
  - Credential service tests (file backend)
  - API integration tests (real HTTP handlers)
  - `make test` command with coverage reporting
- ✅ **COMPLETED: First-run wizard UI**
  - Welcoming wizard component shown when no devices exist
  - Clear instructions on what's needed (IP, credentials)
  - Help link for finding device IP address
- ✅ **COMPLETED: Improved all error messages**
  - Toast notifications with sonner
  - Actionable error messages (why + how to fix)
  - Context-specific guidance for common issues
- ✅ **COMPLETED: Connection test feedback**
  - Test connection before adding device
  - Real-time validation of SSH credentials
  - Docker detection and version reporting
  - Clear error messages for common issues (wrong password, network unreachable, etc.)
- ✅ **COMPLETED: Simplified UI language (no jargon)**
  - "SSH" → "Device Login"
  - "SSH Key" → "Security Key"
  - "Authentication" → "Login Method"
  - All technical terms replaced with user-friendly language

**Phase 2:**
- Deployment engine with real-time progress
- One-click app deployment (Nextcloud, Vaultwarden)
- Automatic rollback on failure
- "What I Did" transparency view
- Network discovery (mDNS scan)

**Deliverable:** By Phase 1 end, a non-technical user can add a device via wizard, understand validation results, and know what to do next. **If it fails, they know why and how to fix it without Googling.**


4. System Architecture
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


5. Technology Stack Decision
Backend: Go
Why Go over Python/Node?
CriteriaGoPythonNode/TSDistribution✅ Single binary, no runtime❌ Requires Python + deps⚠️ Can bundle, but largePerformance✅ Fast, low memory⚠️ Slower, GIL issues⚠️ Fast for I/O, but heavierConcurrency✅ Goroutines (perfect for multi-device)❌ Threading is painful✅ Async/await is goodType Safety✅ Compile-time checks❌ Runtime errors✅ With TypeScriptEcosystem✅ Great for systems/networking✅ Best for integrations/ML✅ Huge npm ecosystemLearning Curve⚠️ Medium (new syntax)✅ Easy✅ Familiar to web devsDeployment✅ Copy binary, done❌ venv/pip headaches⚠️ node_modules dramaCommunity✅ Strong in DevOps/infra✅ Strong in data/ML✅ Strong in web
Decision: Go for these reasons:

Distribution simplicity - Like Herd (native), we want: curl -sSL install.sh | bash → done. Go compiles to a single binary.
System-level operations - We're managing SSH connections, spawning processes, managing files. Go excels here (Docker, Kubernetes, Tailscale all use Go).
Concurrency - Managing 5+ devices simultaneously? Goroutines make this trivial. Python's asyncio is messier, Node is better but still more complex.
Professional feel - Go signals "production-ready infrastructure tool" vs. Python's "script that grew up" feel.
Memory footprint - Go service uses ~20-50MB RAM vs. Node's ~100-200MB. Matters on small servers.
Cross-compilation - GOOS=linux GOARCH=arm64 go build → ARM binary for free. Python packaging for ARM is painful.

Tradeoffs we're accepting:

Fewer contributors know Go (but it's easy to learn)
No Ansible integration (we'll use SSH + templating instead)
Smaller library ecosystem (but Docker, SSH, HTTP clients are excellent)

Go Framework Stack
go// Core frameworks
- fiber/v2          // Web framework (Express-like, faster than Gin)
- gorm             // ORM for SQLite
- viper            // Configuration management
- zap              // Structured logging
- validator/v10    // Request validation

// Job Processing (MVP - in-memory)
- channels + goroutines  // Simple queue for MVP (no Redis dependency)
// Future: Migrate to Asynq + Redis when scaling beyond single node

// Infrastructure clients
- ssh              // stdlib SSH client (crypto/ssh)
- docker/client    // Official Docker SDK
- websocket        // gorilla/websocket for live updates

// Security
- keyring          // OS keychain integration (99designs/keyring)
- crypto/bcrypt    // Password hashing
- crypto/aes       // Fallback encryption if keychain unavailable
Frontend: React + TypeScript
javascript// Core stack
- React 18          // UI framework
- TypeScript        // Type safety
- Vite              // Build tool (fast!)
- TanStack Query    // Server state management
- Zustand           // Client state (lighter than Redux)
- Tailwind CSS      // Styling
- shadcn/ui         // Component library (customizable)
- Recharts          // Dashboards/graphs
- react-router      // Routing
Why this frontend stack?

Vite - Instant HMR, feels native (like Herd's desktop app responsiveness)
shadcn/ui - Beautiful components, fully customizable (not a black box like MUI)
TanStack Query - Perfect for our API-heavy app, handles caching/invalidation
Tailwind - Rapid styling, consistent design system

Database: SQLite (MVP) → PostgreSQL (Scale)
SQLite for MVP because:

Zero configuration
Single file, easy backups
Sufficient for < 100 devices
Go's database/sql works great with it

Migration path:

GORM abstracts DB, switching to Postgres is config change
When users hit scale, offer Postgres backend

Job Queue: In-Memory (MVP) → Asynq (Scale)
For long-running tasks:

App deployments (docker-compose up takes time)
Backups
Network scans
Updates

Why In-Memory for MVP?

✅ Zero dependencies (no Redis to install/manage)
✅ Keeps "single binary" promise intact
✅ Sufficient for single control plane node
✅ Simple implementation with Go channels + worker pool

Migration Path to Asynq:

When users need multi-node control plane or job persistence
Redis becomes optional dependency (can run alongside)
Abstract job interface makes swapping backends trivial
go
type JobQueue interface {
    Enqueue(task Task) error
    Process(handler TaskHandler) error
}

MVP Implementation:
go
// Simple channel-based queue
type InMemoryQueue struct {
    tasks chan Task
    workers int
}

// Workers process tasks concurrently
func (q *InMemoryQueue) Start() {
    for i := 0; i < q.workers; i++ {
        go q.worker()
    }
}


6. Repository Structure (Monorepo)

Decision: Single repository with backend and frontend

Why Monorepo?

✅ Atomic changes across API contracts (frontend + backend in one commit)
✅ Shared type generation (Go structs → TypeScript interfaces)
✅ True single binary (embed React build with //go:embed)
✅ Simplified versioning (one version, one release)
✅ Better DX (one make dev command)
✅ Early stage velocity (both evolve together)

Directory Structure:
bash
homelab-orchestration-platform/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # Entry point
│   ├── internal/                    # Private application code
│   │   ├── api/                     # HTTP handlers (Fiber routes)
│   │   ├── services/                # Business logic
│   │   │   ├── deployment/          # App deployment orchestration
│   │   │   ├── device/              # Device management
│   │   │   ├── network/             # OPNsense, DNS, firewall
│   │   │   └── backup/              # Backup operations
│   │   ├── models/                  # GORM database models
│   │   ├── queue/                   # Job queue implementation
│   │   ├── docker/                  # Docker client wrapper
│   │   ├── ssh/                     # SSH client utilities
│   │   ├── opnsense/                # OPNsense REST API client
│   │   └── websocket/               # WebSocket hub
│   ├── pkg/                         # Reusable packages (can be imported externally)
│   │   ├── templates/               # docker-compose template engine
│   │   └── validation/              # Shared validators
│   ├── migrations/                  # SQL migration files
│   ├── templates/                   # App deployment templates
│   │   ├── vaultwarden.yml.tmpl
│   │   ├── uptime-kuma.yml.tmpl
│   │   └── immich.yml.tmpl
│   ├── web/                         # Embedded frontend (from build)
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── api/                     # Generated TypeScript client
│   │   │   ├── client.ts            # API client
│   │   │   └── types.ts             # Auto-generated from Go
│   │   ├── components/
│   │   │   ├── ui/                  # shadcn components
│   │   │   ├── devices/
│   │   │   ├── deployments/
│   │   │   └── network/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Devices.tsx
│   │   │   ├── AppCatalog.tsx
│   │   │   └── Settings.tsx
│   │   ├── lib/
│   │   │   ├── websocket.ts         # WebSocket client
│   │   │   └── query-client.ts      # TanStack Query setup
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── public/
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── docs/
│   ├── architecture.md              # This file (living document)
│   ├── api.md                       # API documentation
│   └── deployment-guide.md
│
├── deployments/
│   ├── docker/
│   │   └── Dockerfile               # Multi-stage build
│   └── systemd/
│       └── homelab.service
│
├── scripts/
│   ├── install.sh                   # curl | bash installer
│   ├── dev-setup.sh                 # Developer onboarding
│   └── generate-types.sh            # Go → TypeScript type gen
│
├── Makefile                         # Unified build commands
├── .gitignore                       # Combined Go + Node
└── README.md

Build System:
make
# Key Makefile targets

.PHONY: dev
dev:  ## Run development servers
	@goreman start  # Runs Go + Vite concurrently

.PHONY: build
build: frontend-build embed-build  ## Build single binary with embedded frontend

frontend-build:
	cd frontend && npm run build
	# Outputs to frontend/dist/

embed-build:
	# Copy frontend/dist/ to backend/web/
	cp -r frontend/dist/* backend/web/
	# Build Go binary (embeds backend/web/)
	cd backend && go build -o ../bin/homelab cmd/server/main.go

.PHONY: docker
docker:  ## Build Docker image
	docker build -f deployments/docker/Dockerfile -t homelab:latest .

.PHONY: types
types:  ## Generate TypeScript types from Go structs
	cd backend && tygo generate

Frontend Embedding (Go):
go
// backend/cmd/server/main.go
package main

import (
    "embed"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/filesystem"
)

//go:embed web/*
var embedFrontend embed.FS

func main() {
    app := fiber.New()

    // API routes
    api := app.Group("/api/v1")
    // ... register API handlers

    // Serve embedded React app
    app.Use("/", filesystem.New(filesystem.Config{
        Root: http.FS(embedFrontend),
        PathPrefix: "web",
        Index: "index.html",
    }))

    app.Listen(":8080")
}

Type Generation:
bash
# Using tygo (Go → TypeScript)
# backend/tygo.yaml
packages:
  - path: "internal/models"
    type_mappings:
      time.Time: "string"
      uuid.UUID: "string"

# Generates frontend/src/api/types.ts
# Run: make types

Development Workflow:

Development: make dev (Go runs on :8080, Vite on :5173 with proxy)
Type changes: make types (regenerate TS types from Go models)
Build: make build (single binary with embedded frontend)
Deploy: make docker (containerized for easy distribution)

This structure provides:
- Clean separation of concerns
- Easy navigation (backend vs frontend clear)
- Single source of truth for types
- Simple build process
- Production = one binary or one Docker image

7. Data Models
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

8. Application Deployment Pipeline
This is the core "magic" - how one-click deployment works.

Key Principles:
1. **Idempotent**: Re-running deployment doesn't create duplicates
2. **Atomic Rollback**: Failure at any step cleans up everything
3. **Observable**: User sees every step in real-time via WebSocket
4. **Debuggable**: "Show me what you did" reveals all generated configs

Deployment State Machine:
go
type DeploymentStatus string

const (
    StatusValidating    DeploymentStatus = "validating"
    StatusPreparing     DeploymentStatus = "preparing"
    StatusDeploying     DeploymentStatus = "deploying"
    StatusConfiguring   DeploymentStatus = "configuring"
    StatusHealthCheck   DeploymentStatus = "health_check"
    StatusRunning       DeploymentStatus = "running"
    StatusFailed        DeploymentStatus = "failed"
    StatusRollingBack   DeploymentStatus = "rolling_back"
    StatusRolledBack    DeploymentStatus = "rolled_back"
)

Core Deployment Flow with Rollback:
go
func (s *DeploymentService) DeployApplication(
    ctx context.Context,
    appSlug string,
    deviceID uuid.UUID,
    config DeploymentConfig,
) (*Deployment, error) {

    // Track rollback actions
    rollback := NewRollbackManager()
    defer func() {
        if r := recover(); r != nil {
            rollback.Execute()
            panic(r)
        }
    }()

    // 1. Pre-flight Validation (IDEMPOTENT)
    s.UpdateStatus(deployment, StatusValidating)

    app := s.GetApplication(appSlug)
    device := s.GetDevice(deviceID)

    // Check if already deployed (idempotency)
    if existing := s.FindDeployment(appSlug, deviceID); existing != nil {
        if existing.Status == StatusRunning {
            return existing, nil // Already deployed, return existing
        }
        // Cleanup failed previous attempt
        s.Cleanup(existing)
    }

    // Validate resources
    if err := s.ValidateResources(device, app); err != nil {
        return nil, fmt.Errorf("insufficient resources: %w", err)
    }

    // Check port availability
    if err := s.CheckPortAvailable(device, config.InternalPort); err != nil {
        return nil, fmt.Errorf("port conflict: %w", err)
    }

    // Validate Docker installed
    if err := s.CheckDockerInstalled(device); err != nil {
        return nil, fmt.Errorf("docker not found: %w\n\nInstall Docker:\n  curl -fsSL https://get.docker.com | sh", err)
    }

    // 2. Prepare Infrastructure (IDEMPOTENT)
    s.UpdateStatus(deployment, StatusPreparing)

    if !s.HasReverseProxy(device) {
        if err := s.DeployNginxProxyManager(device); err != nil {
            return nil, fmt.Errorf("failed to deploy reverse proxy: %w", err)
        }
        rollback.Add(func() {
            s.RemoveNginxProxyManager(device)
        })
    }

    // 3. Generate Config (DETERMINISTIC)
    compose := s.RenderTemplate(app.ConfigTemplate, config)

    // 4. Deploy to Device (IDEMPOTENT - docker-compose up is idempotent)
    s.UpdateStatus(deployment, StatusDeploying)

    deployPath := fmt.Sprintf("/opt/homelab/%s", appSlug)
    if err := s.SSHClient.Exec(device, fmt.Sprintf("mkdir -p %s", deployPath)); err != nil {
        return nil, fmt.Errorf("failed to create deploy directory: %w", err)
    }

    composePath := fmt.Sprintf("%s/docker-compose.yml", deployPath)
    if err := s.SSHClient.CopyFile(device, composePath, compose); err != nil {
        return nil, fmt.Errorf("failed to copy compose file: %w", err)
    }

    // Store compose for debugging
    deployment.GeneratedCompose = compose

    if err := s.SSHClient.Exec(device, fmt.Sprintf("cd %s && docker-compose up -d", deployPath)); err != nil {
        rollback.Execute()
        return nil, fmt.Errorf("docker-compose failed: %w", err)
    }

    rollback.Add(func() {
        s.SSHClient.Exec(device, fmt.Sprintf("cd %s && docker-compose down -v", deployPath))
    })

    // 5. Configure Networking (IDEMPOTENT)
    s.UpdateStatus(deployment, StatusConfiguring)

    // Add reverse proxy rule (check if exists first)
    if !s.HasProxyRule(device, config.Domain) {
        if err := s.AddReverseProxy(device, config.Domain, config.InternalPort); err != nil {
            rollback.Execute()
            return nil, fmt.Errorf("failed to configure reverse proxy: %w", err)
        }
        rollback.Add(func() {
            s.RemoveReverseProxy(device, config.Domain)
        })
    }

    // Add DNS rewrite (check if exists first)
    if router := s.GetRouter(); router != nil {
        if !s.HasDNSRewrite(router, config.Domain) {
            if err := s.AddDNSRewrite(router, config.Domain, device.IP); err != nil {
                // Non-fatal: DNS might be managed manually
                log.Warn("Failed to add DNS rewrite (continuing anyway): %v", err)
                s.NotifyUser(fmt.Sprintf("⚠️ Couldn't auto-configure DNS. Manually add: %s → %s", config.Domain, device.IP))
            } else {
                rollback.Add(func() {
                    s.RemoveDNSRewrite(router, config.Domain)
                })
            }
        }
    }

    // 6. Health Check
    s.UpdateStatus(deployment, StatusHealthCheck)

    if err := s.WaitForHealthy(device, config.InternalPort, 60*time.Second); err != nil {
        rollback.Execute()
        return nil, fmt.Errorf("health check failed: %w", err)
    }

    // 7. Finalize
    deployment.Status = StatusRunning
    deployment.DeployedAt = time.Now()
    s.DB.Save(deployment)

    // 8. Notify Success
    s.NotifyClients("deployment_complete", deployment)

    log.Info("Successfully deployed %s to %s at https://%s", appSlug, device.Name, config.Domain)

    return deployment, nil
}

Rollback Manager:
go
type RollbackManager struct {
    actions []func()
    mu      sync.Mutex
}

func (r *RollbackManager) Add(action func()) {
    r.mu.Lock()
    defer r.mu.Unlock()
    // Add to front (LIFO - reverse order)
    r.actions = append([]func(){action}, r.actions...)
}

func (r *RollbackManager) Execute() {
    r.mu.Lock()
    defer r.mu.Unlock()

    log.Info("Executing rollback (%d actions)", len(r.actions))

    for _, action := range r.actions {
        func() {
            defer func() {
                if err := recover(); err != nil {
                    log.Error("Rollback action failed: %v", err)
                }
            }()
            action()
        }()
    }
}

Idempotency Checks:
go
// Each operation checks existing state first
func (s *DeploymentService) AddReverseProxy(device *Device, domain string, port int) error {
    // Check if rule exists
    rules := s.NPMClient.GetProxyRules(device)
    for _, rule := range rules {
        if rule.Domain == domain {
            log.Info("Proxy rule for %s already exists, skipping", domain)
            return nil // Idempotent
        }
    }

    // Create rule
    return s.NPMClient.CreateProxyRule(device, domain, port)
}

func (s *DeploymentService) AddDNSRewrite(router *Device, domain string, ip string) error {
    // Check if rewrite exists
    rewrites := s.AdGuardClient.GetRewrites(router)
    for _, rw := range rewrites {
        if rw.Domain == domain {
            log.Info("DNS rewrite for %s already exists, skipping", domain)
            return nil // Idempotent
        }
    }

    // Create rewrite
    return s.AdGuardClient.CreateRewrite(router, domain, ip)
}

Debugging & Transparency:
Every deployment record stores:
go
type Deployment struct {
    // ... other fields

    GeneratedCompose string         `gorm:"type:text"` // Show user what we created
    SSHCommands      []string        `gorm:"type:json"` // Every SSH command executed
    RollbackLog      []string        `gorm:"type:json"` // What was rolled back
    ErrorDetails     string          `gorm:"type:text"` // Full error for debugging
}

// API endpoint for transparency
// GET /api/v1/deployments/:id/debug
{
    "docker_compose": "...",
    "ssh_commands": [
        "mkdir -p /opt/homelab/vaultwarden",
        "cd /opt/homelab/vaultwarden && docker-compose up -d"
    ],
    "proxy_config": { ... },
    "dns_config": { ... }
}
Docker Compose Template Example
yaml# templates/immich.yml.tmpl
version: '3.8'

services:
  immich-server:
    image: ghcr.io/immich-app/immich-server:${VERSION}
    container_name: immich-server
    restart: unless-stopped
    volumes:
      - ${UPLOAD_PATH}:/usr/src/app/upload
      - /etc/localtime:/etc/localtime:ro
    environment:
      DB_HOSTNAME: immich-postgres
      DB_USERNAME: ${DB_USER}
      DB_PASSWORD: ${DB_PASSWORD}
      DB_DATABASE_NAME: ${DB_NAME}
      REDIS_HOSTNAME: immich-redis
    depends_on:
      - immich-postgres
      - immich-redis
    ports:
      - "${INTERNAL_PORT}:3001"

  immich-postgres:
    image: tensorchord/pgvecto-rs:pg14-v0.2.0
    container_name: immich-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - immich-pgdata:/var/lib/postgresql/data

  immich-redis:
    image: redis:7-alpine
    container_name: immich-redis
    restart: unless-stopped

volumes:
  immich-pgdata:

9. Design Decisions Log

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
**Status**: ✅ Implemented

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
**Status**: ✅ Implemented

### DD-003: Credential Storage - OS Keychain vs File Encryption (2025-10-07)
**Decision**: Use OS keychain (macOS Keychain, Linux Secret Service, Windows DPAPI)
**Rationale**:
- More secure than file-based master key
- Integrates with OS security model
- No single point of failure file

**Alternatives Considered**: mozilla/sops, HashiCorp Vault, encrypted file
**Trade-offs**: Platform-specific code, fallback to AES if keychain unavailable
**Status**: 🔄 Planned

### DD-004: OPNsense API - REST vs XML-RPC (2025-10-07)
**Decision**: Use OPNsense REST API, not XML-RPC
**Rationale**:
- XML-RPC is legacy, being phased out
- REST API has better documentation
- More maintainable

**Alternatives Considered**: XML-RPC (original plan), screen scraping
**Trade-offs**: Requires newer OPNsense version (20.7+)
**Status**: 🔄 Planned

### DD-005: Network Discovery - Manual First vs Auto-discovery (2025-10-07)
**Decision**: Manual IP entry as primary, mDNS/ARP as optional enhancement
**Rationale**:
- mDNS doesn't work across VLANs (common in homelabs)
- ARP scanning limited to same subnet
- Manual entry more reliable
- Discovery as convenience feature, not core

**Alternatives Considered**: mDNS/ARP as primary, requiring flat network
**Trade-offs**: Less "magical" but more reliable
**Status**: 🔄 Planned

### DD-006: Deployment Rollback Strategy (2025-10-07)
**Decision**: Implement LIFO rollback manager with defer pattern
**Rationale**:
- Ensures cleanup on any failure
- Reverse order rollback (LIFO) naturally undoes dependencies
- Go's defer pattern makes it simple

**Implementation**: RollbackManager with action stack
**Status**: 🔄 Planned

### DD-007: App Deployment Order (2025-10-07)
**Decision**: Start with simple apps (Vaultwarden, Uptime Kuma), add complex ones later
**Rationale**:
- Immich has complex setup (multiple containers, database, ML)
- Vaultwarden is single container, simple
- Learn from simple deployments first

**Original Plan**: Immich first
**Revised Plan**: Vaultwarden → Uptime Kuma → Jellyfin → Immich
**Status**: 🔄 Planned

---

## 10. Development Phases (Revised)

### Phase 0: Foundation Setup (Week 1) ✅ COMPLETE

🎯 **Goal**: Monorepo structure, tooling, can run "Hello World"

- [x] Initialize monorepo structure (backend/ and frontend/ directories)
- [x] Set up Go modules (go.mod) and Vite (package.json)
- [x] Create Makefile with dev, build, test, types targets
- [x] Configure Procfile/goreman for concurrent dev servers
- [x] Set up basic Fiber server with health check endpoint
- [x] Set up React with Vite, basic routing, API integration
- [x] Configure Vite proxy for backend API calls
- [x] Write tests for health check endpoint (passing)
- [x] Create combined .gitignore for Go + Node
- [x] Create README.md with quickstart instructions
- [x] Test: `make dev` runs both servers successfully
- [ ] Configure //go:embed for frontend embedding (deferred to build phase)
- [ ] Test: `make build` creates single binary (deferred to Phase 2)
- [ ] Set up tygo for Go → TypeScript type generation (deferred to Phase 1)

**Deliverable**: ✅ **COMPLETE** - Developers can clone, run `make dev`, see:
- Backend at http://localhost:8080/api/v1/health
- Frontend at http://localhost:5173
- Full hot-reload for both frontend and backend

---

### Phase 1: Multi-Node Intelligence (Weeks 2-6) 🚧 IN PROGRESS
🎯 **Goal**: **Core differentiator features** - Intelligent placement, resource aggregation, multi-node orchestration

**Priority: Build what makes us DIFFERENT first**

**Intelligent Orchestration (NEW - MVP CORE):**
- [ ] **IntelligentScheduler service** - Resource scoring algorithm
  - Score devices based on: RAM, storage, CPU, current load, uptime
  - `SelectOptimalDevice(app)` returns best device + reasoning
  - Override mechanism for manual device selection
- [ ] **Resource aggregation service** - Cross-device monitoring
  - `GetAggregateResources()` returns total RAM/CPU/storage across all devices
  - Real-time resource usage tracking per device
  - Device health scoring
- [ ] **Database pooling service** (simplified for MVP)
  - Detect if Postgres/MySQL already deployed on device
  - Deploy shared database instance if needed
  - API for provisioning new database/user in shared instance
  - Future: Automatic database provisioning from recipes

**Multi-Device Management (ENHANCED):**
- [x] Device CRUD API endpoints
- [x] SSH client wrapper with connection pooling
- [x] Docker installation checker (pre-flight validation)
- [x] Device health monitoring (ping, Docker API status)
- [ ] **Multi-device dashboard** - Show aggregate resources
- [ ] **Device resource polling** - Update RAM/CPU/storage every 30s
- [ ] **Smart device recommendations** - "Server-02 recommended for this app"

**Network Integration** (deferred to Phase 2/3):
- [ ] OPNsense REST API client (not XML-RPC)
- [ ] Read-only network stats (WAN status, DHCP leases)
- [ ] AdGuard Home detection (optional, graceful degradation)
- [x] Manual device addition as primary flow

**Frontend**:
- [x] Tailwind CSS v4 setup
- [x] shadcn/ui component library setup
- [x] TypeScript types for API models (tygo configured)
- [x] API client with TanStack Query hooks
- [x] Device list page with status indicators
- [x] Device add form with dialog (manual IP entry, SSH credentials)
- [x] Basic dashboard with device health cards (DeviceHealthCard component, grid layout)
- [x] WebSocket connection infrastructure (hub complete, client pending)
- [x] Error toast notifications (device loading errors, add/delete operations)

**Deliverable**: ✅ Can add device (router or server) by IP, see status, validate Docker installed
**Test Coverage**: 26 frontend tests (25 passing, 1 skipped) across 3 test files

---

### Phase 2: Intelligent Deployment UX (Weeks 7-9)
🎯 **Goal**: **Surface the intelligence** - UI that showcases multi-node orchestration

**App Repository Architecture (NEW):**
- [ ] Implement AppRegistry service - Fetch apps from GitHub repository
- [ ] Standard docker-compose.yaml format (no Go templates)
- [ ] Separate manifest.yaml for platform metadata
- [ ] Programmatic deployment via Docker Swarm API
- [ ] Compose enhancement with intelligent placement constraints
- [ ] See [docs/app-repository.md](app-repository.md) for complete specification

**Smart Deployment Wizard (NEW FOCUS):**
- [ ] **Automatic device selection** - System picks best device by default
  - "Analyzing your homelab..." loading state
  - "Recommended: Server-02 (8GB RAM available, 40% load)" with score explanation
  - Allow manual override: "Or deploy to a different device"
- [ ] **Resource availability preview** - Show what's available across ALL devices
  - Before deployment: "Your homelab has 16GB RAM free across 3 devices"
  - After selection: "Server-02 selected: 8GB RAM free, 500GB storage"
- [ ] **Database sharing indicators** - Show when apps will share resources
  - "NextCloud will use existing Postgres instance (saves 1GB RAM)"
  - "No Postgres found, deploying shared instance (1.2GB RAM)"

**Frontend Enhancements:**
- [ ] **Unified dashboard** - Aggregate resource view
  - Total RAM/CPU/storage bars
  - Online devices count
  - Total apps deployed
  - Quick actions for common tasks
- [ ] **Deployment wizard updates**
  - Step 1: App selection (existing)
  - Step 2: **NEW** - Smart device recommendation (auto-selected)
  - Step 3: Configuration (simplified, most apps zero-config)
  - Step 4: Real-time deployment progress
- [ ] **Device comparison view** - Show all devices with resource scores
  - Side-by-side device cards
  - Highlight recommended device
  - Show reasoning for each score

**Deliverable**: ✅ User clicks "Deploy NextCloud" → System automatically picks Server-02 → Shows "Recommended because 8GB RAM free" → One-click deploy

---

### Phase 3: Database Pooling & Resource Optimization (Weeks 10-12)
🎯 **Goal**: **Prove the value prop** - Show concrete RAM savings from resource sharing

**Database Pooling (CORE FEATURE):**
- [ ] **Shared Postgres service** - Deploy one instance per device
  - Detect if Postgres already running
  - Auto-deploy if needed (recipe-based)
  - Health monitoring
- [ ] **Database provisioning API**
  - `CreateDatabase(appName)` creates new DB + user in shared instance
  - Execute SQL via Docker exec
  - Return connection credentials
- [ ] **Recipe integration** - Apps request database instead of deploying own
  - Recipe declares `requires_database: postgres`
  - System provisions from shared instance
  - Inject credentials into docker-compose template
- [ ] **Resource savings dashboard**
  - "You're saving 2.8GB RAM by sharing databases"
  - Show: 5 apps sharing 1 Postgres vs 5 separate instances
  - Visual: Before/after RAM usage comparison

**Deliverable**: ✅ Deploy 3 apps (NextCloud, Monica, Paperless) → All share one Postgres instance → Show "Saved 2GB RAM"

---

### Phase 4: Cross-Device Migration & Polish (Weeks 13-16)
🎯 **Goal**: Complete the multi-node story - apps can move between devices

**Cross-Device Features:**
- [ ] **App migration** - Move deployment from one device to another
  - Export volumes/data from source device
  - Transfer via SCP to destination
  - Redeploy on destination with same config
  - Health check and cutover
- [ ] **Load rebalancing recommendations**
  - Detect overloaded devices (>90% RAM)
  - Suggest migrations: "Server-01 is full, migrate Vaultwarden to Pi-4?"
  - One-click migration execution
- [ ] **Reverse proxy automation** (moved from Phase 3)
  - Traefik auto-deployment (simpler than NGINX Proxy Manager)
  - Automatic routing rules
  - SSL certificates (Let's Encrypt or self-signed)

**Reliability**:
- [ ] Comprehensive error handling throughout
- [ ] Retry logic for network operations
- [ ] Graceful degradation (missing AdGuard, etc.)
- [ ] User-facing error messages (no stack traces)

**Backup & Config** (moved to Phase 5):
- [ ] See Phase 5 for comprehensive backup architecture

**Security**:
- [ ] OS keychain integration (99designs/keyring)
- [ ] Fallback to AES encryption if keychain unavailable
- [ ] Audit log of all configuration changes
- [ ] HTTPS with self-signed cert (auto-generated)

**Observability**:
- [ ] Structured logging (zap)
- [ ] Basic metrics endpoint (Prometheus format)
- [ ] Health check dashboard

**Distribution**:
- [ ] Multi-stage Dockerfile
- [ ] curl | bash install script
- [ ] systemd service file
- [ ] Binary releases for Linux (amd64, arm64)
- [ ] Docker image on DockerHub

**Documentation**:
- [ ] Updated architecture.md with completed sections
- [ ] API documentation
- [ ] User quickstart guide
- [ ] Demo video (5 min walkthrough)

**Deliverable**: ✅ Public beta on GitHub, ready for first 100 users

---

### Phase 5: Automated Encrypted Backups (Weeks 17-20)
🎯 **Goal**: Data protection without manual backup scripts - automated, encrypted backups to S3 or local NAS

**Backup Infrastructure:**
- [ ] **BackupService with restic integration**
  - Client-side AES-256 encryption
  - Deduplication with content-defined chunking
  - Repository management (init, check, unlock)
- [ ] **Backup destinations**
  - S3-compatible cloud storage (Backblaze B2, Wasabi, AWS S3, MinIO)
  - Local NAS or server storage (NFS)
  - Credentials encrypted with OS keychain
- [ ] **Backup policies**
  - Default policy: Daily at 2 AM, retain 7 daily, 4 weekly, 6 monthly
  - Per-app custom policies (hourly for critical apps, weekly for media)
  - Automatic retention enforcement (forget + prune)

**Automated Operations:**
- [ ] **Scheduled backups**
  - Cron-based scheduler checks every minute
  - Execute due backups in background
  - Track backup status per app
- [ ] **Backup scope**
  - Docker volumes (app data)
  - Docker Compose files (app configs)
  - Platform database (deployments, devices)
  - Shared database pools (PostgreSQL, MySQL)

**Restore Operations:**
- [ ] **Snapshot browsing**
  - List all snapshots for app or system
  - Show snapshot metadata (size, files changed, duration)
  - Filter by date range or app
- [ ] **One-click restore**
  - Full app state restore
  - Selective file restore
  - Restore to different device

**Dashboard Integration:**
- [ ] **Backup settings page**
  - Add/manage backup destinations
  - Create/edit backup policies
  - Test S3 connectivity
- [ ] **Per-app backup tab**
  - Enable/disable backups
  - Choose backup policy
  - View backup history
  - Trigger immediate backup
  - Browse and restore snapshots
- [ ] **Backup monitoring**
  - Dashboard widget: "Last backup: 2 hours ago"
  - Storage usage tracking
  - Show deduplication savings: "Saved 85% storage costs"
  - Backup health alerts (failed backups, low storage)

**Technical Implementation:**
- [ ] Database models: BackupDestination, BackupPolicy, BackupConfiguration, BackupSnapshot
- [ ] restic wrapper service with environment management
- [ ] Repository password encryption (OS keychain)
- [ ] Multi-destination redundancy (backup to both cloud AND local)
- [ ] Bandwidth limiting for large backups
- [ ] Pre/post backup hooks (database dumps)

**See [docs/backup-architecture.md](backup-architecture.md) for complete technical specification**

**Deliverable**: ✅ Configure Backblaze B2 destination → Default policy applies to all apps → Daily encrypted backups → Browse snapshots → One-click restore → Dashboard shows "85% storage savings from deduplication"

---

### Post-MVP (Future)
Ideas not in MVP scope:
- Multi-user support (RBAC, family accounts)
- Remote access (Tailscale/Cloudflare Tunnel integration)
- Managed switch integration (VLAN creation via UI)
- Kubernetes support
- Mobile app (React Native)
- Monitoring/alerting (Prometheus + Alertmanager)
- Community app store
- Terraform/Pulumi export

## 11. Deployment Strategy
For End Users
Option 1: Docker (Recommended)
bashdocker run -d \
  --name homelab-control \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v homelab-data:/data \
  yourusername/homelab-control:latest
Option 2: Binary Install
bashcurl -sSL https://install.homelab.sh | bash
# Downloads binary to /usr/local/bin/homelab
# Creates systemd service
# Starts web UI at http://localhost:8080
Option 3: From Source
bashgit clone https://github.com/you/homelab-control
cd homelab-control
make install
For Developers
bash# Backend
cd backend
go run cmd/server/main.go

# Frontend
cd frontend  
npm install
npm run dev

# Access: http://localhost:5173 (Vite) → API at :8080

## 12. Security Considerations
Credential Storage:

OS keychain integration (99designs/keyring)
  - macOS: Keychain
  - Linux: Secret Service (gnome-keyring, kwallet)
  - Windows: DPAPI
Fallback to AES-256-GCM if keychain unavailable
No plaintext credentials on disk

API Security:

JWT authentication
Rate limiting (100 req/min per IP)
HTTPS required in production (self-signed cert auto-generated)
CORS restricted to same-origin

Device Access:

SSH keys preferred over passwords
Credentials never logged
Audit log of all configuration changes

Updates:

Code signing for binaries
Version pinning for Docker images
Optional auto-updates (off by default)


## 13. Success Metrics (MVP)
Technical:

 Install time: < 5 minutes
 First app deployed: < 10 minutes
 Dashboard load time: < 2 seconds
 Memory usage: < 100MB (backend)
 API response time: < 200ms (p95)

User Experience:

 80% of users deploy first app without docs
 50% deploy 3+ apps in first week
 < 5% encounter errors during setup
 NPS score > 40


## 14. Open Questions / Future Scope
Not in MVP, but thinking about:

Multi-user support (family members with different permissions)
Remote access (managed Tailscale/Cloudflare Tunnel)
Managed switch integration (VLAN assignment via UI)
Kubernetes support (for advanced users)
Mobile app (React Native, reuse frontend code)
Monitoring/alerting (Prometheus + Alertmanager integration)
App store (community plugins)
Terraform/Pulumi export (IaC for advanced users)


## 15. Getting Started (Developer Onboarding)
Prerequisites:

Go 1.21+
Node 18+
Docker
Make

First Contribution:
bashgit clone https://github.com/you/homelab-control
cd homelab-control
make dev-setup     # Installs deps, creates .env
make dev           # Starts backend + frontend
make test          # Run tests
Contributing:

Check CONTRIBUTING.md
Join Discord/Slack for questions
All PRs require tests + docs updates


---

## Summary

This architecture balances:
✅ **Simplicity** (single binary, SQLite, Docker-only, no Redis for MVP)
✅ **Reliability** (rollback on failure, idempotent operations, pre-flight checks)
✅ **Security** (OS keychain integration, encrypted credentials, HTTPS)
✅ **Transparency** (show generated configs, SSH commands, debug endpoints)
✅ **Power** (full access to underlying tools, escape hatches everywhere)
✅ **Scalability** (can grow to Postgres, Redis, K8s later)
✅ **Developer Experience** (Go + TS, monorepo, type generation, modern stack)

### Key Architectural Principles

1. **Hide complexity without removing capability**
   Every "magic" action has a "show me what you did" button that displays generated docker-compose, SSH commands, API calls, etc.

2. **Fail gracefully, rollback atomically**
   If DNS fails, deployment rolls back. If AdGuard unavailable, show manual instructions. No partial states.

3. **Idempotent by default**
   Re-running any operation is safe. Check state before creating resources.

4. **Manual first, automation second**
   Network discovery is a convenience. Manual IP entry is the reliable path.

5. **Start simple, grow complex**
   Vaultwarden before Immich. In-memory queue before Redis. SQLite before Postgres.

### What Changed from v0.1?

**Removed:**
- ❌ Redis/Asynq dependency (breaks single binary promise)
- ❌ XML-RPC for OPNsense (legacy API)
- ❌ File-based credential encryption (single point of failure)
- ❌ Network discovery as primary flow (unreliable across VLANs)

**Added:**
- ✅ In-memory job queue (Redis migration path later)
- ✅ Rollback manager with LIFO cleanup
- ✅ Idempotency checks for all operations
- ✅ OS keychain integration
- ✅ Pre-flight validation (Docker, ports, resources)
- ✅ Debug/transparency endpoints
- ✅ Design decisions log
- ✅ Monorepo structure with embedded frontend

**Next Steps:**
1. Begin Phase 0 (Foundation Setup) - see Section 10
2. Update this document as implementation progresses
3. Mark checkboxes ✅ as features complete
4. Add new design decisions to Section 9 as they arise

The key inspiration from Laravel/Herd remains: **Hide complexity without removing capability.** But now with proper error handling, rollback, and transparency.
