# Intelligent Orchestration

## Overview

Technical specification for intelligent multi-node orchestration:
1. **Intelligent Placement Algorithm** - Automatic device selection via resource scoring
2. **Database Pooling** - Shared database instances with auto-provisioning
3. **Resource Aggregation** - Unified resource view across devices

---

## 1. Intelligent Placement Algorithm

### Purpose

Automatic optimal device selection based on:
- Available resources (RAM, storage, CPU)
- Current device load
- Device reliability (uptime)

**Workflow:**
```
User: "Deploy NextCloud"
System: "Analyzing... Recommended: Server-02 (8GB RAM free, 40% load)"
User: Clicks "Deploy"
```

### Architecture

```go
type IntelligentScheduler struct {
    deviceService   *DeviceService
    resourceMonitor *ResourceMonitor
    db              *gorm.DB
}

type PlacementScore struct {
    Device            *Device
    RAMScore          float64  // 0-100
    StorageScore      float64  // 0-100
    CPUScore          float64  // 0-100
    LoadScore         float64  // 0-100 (inverse of current load)
    ReliabilityScore  float64  // 0-100 (based on uptime)
    TotalScore        float64  // Weighted average
    Reasoning         string   // Human-readable explanation
    IsQualified       bool     // Meets minimum requirements
}
```

### Scoring Algorithm

**Weights:**
- RAM availability: **40%** (most important for Docker containers)
- Storage availability: **30%** (important for persistent data)
- CPU capability: **15%** (less critical for most homelab apps)
- Current load: **10%** (prefer less-loaded devices)
- Uptime/reliability: **5%** (prefer stable devices)

**Scoring Factors:**

1. **RAM Score (40%)**
   - Below minimum required: 0 (disqualified)
   - Meets recommended: 100
   - Between min and recommended: 50-100 scaled

2. **Storage Score (30%)**
   - Below minimum required: 0 (disqualified)
   - Meets recommended: 100
   - Between min and recommended: 50-100 scaled

3. **CPU Score (15%)**
   - 2x recommended cores: 100
   - Meets recommended: 80-100
   - Below recommended: 0-80 scaled

4. **Load Score (10%)**
   - Lower current load = higher score
   - Score = 100 - load_percentage

5. **Reliability Score (5%)**
   - Based on device uptime percentage
   - Score = uptime_percentage

**Method Signatures:**
```go
func (s *IntelligentScheduler) SelectOptimalDevice(app *Recipe) (*Device, *PlacementScore, error)
func (s *IntelligentScheduler) calculatePlacementScore(app *Recipe, device *Device) PlacementScore
```

### API Endpoints

```
GET /api/v1/devices/recommendations?app_slug=nextcloud
# Returns ranked list of devices for deploying this app

Response:
{
  "recommendations": [
    {
      "device_id": "uuid",
      "device_name": "Server-02",
      "score": 95.2,
      "reasoning": "Server-02: 8GB RAM available, 500GB storage, 40% current load, 99.5% uptime",
      "is_qualified": true,
      "recommended": true
    }
  ]
}

POST /api/v1/deployments
Body: { "recipe_slug": "nextcloud", "auto_select": true }
# If auto_select=true, uses intelligent scheduler to pick device
```

---

## 2. Database Pooling

### Purpose

Eliminate container sprawl by sharing database instances across multiple applications.

**Problem:**
- Traditional approach: Each app deploys its own database container
- Result: 5 apps = 5 Postgres containers = 5GB RAM

**Solution:**
- One shared Postgres container per device
- All apps use separate databases within that instance
- Result: 5 apps = 1 Postgres container = 1.5GB RAM (70% savings)

### Architecture

```go
type DatabasePool struct {
    deviceService   *DeviceService
    sshClient       *ssh.Client
    db              *gorm.DB

    // Track shared instances by device
    postgresInstances map[uuid.UUID]*DatabaseInstance
    mysqlInstances    map[uuid.UUID]*DatabaseInstance
    mu                sync.RWMutex
}

type DatabaseInstance struct {
    DeviceID     uuid.UUID
    Type         string // "postgres", "mysql"
    ContainerID  string
    InternalIP   string
    Port         int
    MasterUser   string
    MasterPass   string
    Databases    []string // List of databases created
    CreatedAt    time.Time
    Healthy      bool
}

type DatabaseCredentials struct {
    Host     string
    Port     int
    Database string
    Username string
    Password string
}
```

### Implementation Workflow

**1. Get or Create Shared Instance:**
- Check if shared database instance exists on device
- If not exists: Deploy shared Postgres/MySQL container
- Generate secure master credentials
- Store instance metadata in database

**2. Provision Database for App:**
- Connect to shared instance
- Create new database (e.g., `nextcloud_db`)
- Create dedicated user with secure password
- Grant permissions on new database
- Return credentials to app

**3. Recipe Integration:**
```yaml
# manifest.yaml
database:
  engine: postgres  # postgres, mysql, mariadb, sqlite, none
  auto_provision: true

# docker-compose.yaml
services:
  nextcloud:
    environment:
      POSTGRES_HOST: ${DATABASE_HOST}        # Injected by system
      POSTGRES_DB: ${DATABASE_NAME}          # Injected by system
      POSTGRES_USER: ${DATABASE_USERNAME}    # Injected by system
      POSTGRES_PASSWORD: ${DATABASE_PASSWORD} # Injected by system
```

**Method Signatures:**
```go
func (p *DatabasePool) GetOrCreatePostgresInstance(deviceID uuid.UUID) (*DatabaseInstance, error)
func (p *DatabasePool) ProvisionDatabase(appName string, deviceID uuid.UUID) (*DatabaseCredentials, error)
func (p *DatabasePool) CalculateSavings(deviceID uuid.UUID) (*ResourceSavings, error)
```

### Resource Savings

**Example:**
- 5 apps requiring Postgres
- Without pooling: 5 containers × 1GB = 5GB RAM
- With pooling: 1 shared container = 1.5GB RAM
- **Savings: 3.5GB (70% reduction)**

---

## 3. Resource Aggregation

### Purpose

Provide a unified view of total resources across all devices.

**User Experience:**
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

### Architecture

```go
type ResourceMonitor struct {
    deviceService *DeviceService
    databasePool  *DatabasePool
    db            *gorm.DB
}

type AggregateResources struct {
    // Totals across all devices
    TotalRAM          uint64
    UsedRAM           uint64
    TotalStorage      uint64
    UsedStorage       uint64
    TotalCPUCores     int
    AvgCPUUsage       float64

    // Device stats
    OnlineDevices     int
    OfflineDevices    int
    Devices           []DeviceSummary

    // App stats
    TotalApps         int
    RunningApps       int
    StoppedApps       int

    // Resource savings
    RAMSavedByPooling int64
    SavingsPercentage float64

    LastUpdated       time.Time
}

type DeviceSummary struct {
    ID              uuid.UUID
    Name            string
    Status          string
    RAMUsagePercent float64
    CPUUsagePercent float64
    AppsRunning     int
}
```

**Method Signature:**
```go
func (m *ResourceMonitor) GetAggregateResources() (*AggregateResources, error)
```

**Aggregation Process:**
1. Fetch all devices
2. Sum total and used RAM/storage across online devices
3. Calculate average CPU usage
4. Count apps by status
5. Calculate RAM savings from database pooling
6. Return unified view

### API Endpoint

```
GET /api/v1/resources/aggregate

Response:
{
  "total_ram": 34359738368,
  "used_ram": 24159738368,
  "total_storage": 2199023255552,
  "used_storage": 858993459200,
  "total_cpu_cores": 12,
  "avg_cpu_usage": 58.3,
  "online_devices": 3,
  "offline_devices": 0,
  "total_apps": 12,
  "running_apps": 10,
  "stopped_apps": 2,
  "ram_saved_by_pooling": 3006477107,
  "savings_percentage": 12.4,
  "devices": [...]
}
```

---

## Summary

These three features work together to provide a **unified homelab experience**:

1. **Intelligent Placement** - User doesn't choose device, system picks optimal one
2. **Database Pooling** - Apps share resources, save 60-70% RAM
3. **Resource Aggregation** - User sees homelab as ONE system, not many devices

**Competitive Advantage:**
- CasaOS: Single node only
- Coolify: Manual device selection, no pooling
- Proxmox: Manual allocation, complex
- Kubernetes: Has these features but too complex

**We provide:** Kubernetes-level intelligence with CasaOS-level simplicity.

---

**Version:** 1.0
**Last Updated:** October 2025
