# Intelligent Orchestration

**Version:** 1.0
**Last Updated:** October 2025

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
// backend/internal/services/intelligent_scheduler.go

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

**Implementation:**

```go
func (s *IntelligentScheduler) SelectOptimalDevice(app *Recipe) (*Device, *PlacementScore, error) {
    // 1. Get all online devices
    devices, err := s.deviceService.ListOnlineDevices()
    if err != nil {
        return nil, nil, err
    }

    if len(devices) == 0 {
        return nil, nil, fmt.Errorf("no online devices available")
    }

    // 2. Score each device
    scores := make([]PlacementScore, 0, len(devices))
    for _, device := range devices {
        score := s.calculatePlacementScore(app, device)
        scores = append(scores, score)
    }

    // 3. Filter out disqualified devices (don't meet minimum requirements)
    qualifiedScores := []PlacementScore{}
    for _, score := range scores {
        if score.IsQualified {
            qualifiedScores = append(qualifiedScores, score)
        }
    }

    if len(qualifiedScores) == 0 {
        return nil, nil, fmt.Errorf("no devices meet minimum requirements for %s", app.Name)
    }

    // 4. Sort by total score (descending)
    sort.Slice(qualifiedScores, func(i, j int) bool {
        return qualifiedScores[i].TotalScore > qualifiedScores[j].TotalScore
    })

    // 5. Return best device
    bestScore := qualifiedScores[0]
    return bestScore.Device, &bestScore, nil
}

func (s *IntelligentScheduler) calculatePlacementScore(app *Recipe, device *Device) PlacementScore {
    score := PlacementScore{
        Device:      device,
        IsQualified: true,
    }

    // Get current resource usage
    ramAvailable := device.TotalRAM - device.UsedRAM
    storageAvailable := device.TotalStorage - device.UsedStorage
    loadPercentage := float64(device.UsedRAM) / float64(device.TotalRAM) * 100

    // Factor 1: RAM Score (40% weight)
    appMinRAM := int64(app.Resources.MinRAMMB) * 1024 * 1024 // Convert MB to bytes
    appRecRAM := int64(app.Resources.RecommendedRAMMB) * 1024 * 1024

    if ramAvailable < appMinRAM {
        score.RAMScore = 0
        score.IsQualified = false
    } else if ramAvailable >= appRecRAM {
        // Has recommended RAM or more
        score.RAMScore = 100
    } else {
        // Between minimum and recommended
        ratio := float64(ramAvailable-appMinRAM) / float64(appRecRAM-appMinRAM)
        score.RAMScore = 50 + (ratio * 50) // Scale from 50 to 100
    }

    // Factor 2: Storage Score (30% weight)
    appMinStorage := int64(app.Resources.MinStorageGB) * 1024 * 1024 * 1024 // Convert GB to bytes
    appRecStorage := int64(app.Resources.RecommendedStorageGB) * 1024 * 1024 * 1024

    if storageAvailable < appMinStorage {
        score.StorageScore = 0
        score.IsQualified = false
    } else if storageAvailable >= appRecStorage {
        score.StorageScore = 100
    } else {
        ratio := float64(storageAvailable-appMinStorage) / float64(appRecStorage-appMinStorage)
        score.StorageScore = 50 + (ratio * 50)
    }

    // Factor 3: CPU Score (15% weight)
    // Higher is better - more cores = higher score
    if app.Resources.CPUCores > 0 {
        cpuRatio := float64(device.CPUCores) / float64(app.Resources.CPUCores)
        if cpuRatio >= 2.0 {
            score.CPUScore = 100 // 2x recommended cores
        } else if cpuRatio >= 1.0 {
            score.CPUScore = 80 + (cpuRatio-1.0)*20 // Scale 80-100
        } else {
            score.CPUScore = cpuRatio * 80 // Scale 0-80
        }
    } else {
        score.CPUScore = 100 // No CPU requirement
    }

    // Factor 4: Load Score (10% weight)
    // Lower load = higher score
    score.LoadScore = 100 - loadPercentage
    if score.LoadScore < 0 {
        score.LoadScore = 0
    }

    // Factor 5: Reliability Score (5% weight)
    // Based on uptime percentage
    score.ReliabilityScore = device.UptimePercentage

    // Calculate weighted total
    if score.IsQualified {
        score.TotalScore = (score.RAMScore * 0.40) +
                          (score.StorageScore * 0.30) +
                          (score.CPUScore * 0.15) +
                          (score.LoadScore * 0.10) +
                          (score.ReliabilityScore * 0.05)

        // Generate human-readable reasoning
        score.Reasoning = fmt.Sprintf(
            "%s: %.0fGB RAM available, %.0fGB storage, %.0f%% current load, %.1f%% uptime",
            device.Name,
            float64(ramAvailable)/(1024*1024*1024),
            float64(storageAvailable)/(1024*1024*1024),
            loadPercentage,
            device.UptimePercentage,
        )
    } else {
        score.TotalScore = 0
        score.Reasoning = fmt.Sprintf(
            "%s: Insufficient resources (needs %.0fGB RAM, %.0fGB storage)",
            device.Name,
            float64(appMinRAM)/(1024*1024*1024),
            float64(appMinStorage)/(1024*1024*1024),
        )
    }

    return score
}
```

### API Endpoints

```go
// GET /api/v1/devices/recommendations?app_slug=nextcloud
// Returns ranked list of devices for deploying this app

type DeviceRecommendation struct {
    DeviceID     uuid.UUID `json:"device_id"`
    DeviceName   string    `json:"device_name"`
    Score        float64   `json:"score"`
    Reasoning    string    `json:"reasoning"`
    IsQualified  bool      `json:"is_qualified"`
    Recommended  bool      `json:"recommended"` // true for highest score
}

// POST /api/v1/deployments
// Body: { "recipe_slug": "nextcloud", "auto_select": true }
// If auto_select=true, uses intelligent scheduler to pick device
```

### UI Integration

**Deployment Wizard - Step 2: Device Selection**

```tsx
function DeviceSelectionStep({ app }: { app: Recipe }) {
    const { data: recommendations } = useDeviceRecommendations(app.slug);
    const [selectedDevice, setSelectedDevice] = useState<string | null>(null);
    const [showOverride, setShowOverride] = useState(false);

    // Auto-select recommended device on load
    useEffect(() => {
        if (recommendations && recommendations.length > 0) {
            setSelectedDevice(recommendations[0].device_id);
        }
    }, [recommendations]);

    const recommendedDevice = recommendations?.[0];

    return (
        <div>
            <h2>Analyzing your homelab...</h2>

            {recommendedDevice && (
                <div className="recommended-device">
                    <CheckIcon /> Recommended: {recommendedDevice.device_name}
                    <div className="score">Score: {recommendedDevice.score}/100</div>
                    <div className="reasoning">{recommendedDevice.reasoning}</div>
                </div>
            )}

            {!showOverride && (
                <Button onClick={() => setShowOverride(true)}>
                    Deploy to a different device
                </Button>
            )}

            {showOverride && (
                <div className="device-comparison">
                    {recommendations?.map(rec => (
                        <DeviceCard
                            key={rec.device_id}
                            device={rec}
                            selected={selectedDevice === rec.device_id}
                            onClick={() => setSelectedDevice(rec.device_id)}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}
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
// backend/internal/services/database_pool.go

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

### Implementation

**Step 1: Get or Create Shared Instance**

```go
func (p *DatabasePool) GetOrCreatePostgresInstance(deviceID uuid.UUID) (*DatabaseInstance, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Check if instance already exists
    if instance, exists := p.postgresInstances[deviceID]; exists {
        // Verify it's still healthy
        if p.checkInstanceHealth(instance) {
            return instance, nil
        }
        // Instance unhealthy, recreate
        log.Warn("Postgres instance on device %s is unhealthy, recreating", deviceID)
        p.removeInstance(instance)
    }

    // Deploy new shared Postgres instance
    log.Info("Deploying shared Postgres instance on device %s", deviceID)

    device, err := p.deviceService.GetDevice(deviceID)
    if err != nil {
        return nil, err
    }

    // Generate secure master credentials
    masterUser := "homelab_admin"
    masterPass := generateSecurePassword(32)

    // Deploy using docker-compose
    composeContent := generatePostgresCompose(masterUser, masterPass)
    projectName := fmt.Sprintf("homelab-postgres-shared-%s", deviceID.String()[:8])

    if err := p.deployToDevice(device, projectName, composeContent); err != nil {
        return nil, fmt.Errorf("failed to deploy Postgres: %w", err)
    }

    // Get container internal IP
    internalIP, err := p.getContainerIP(device, projectName)
    if err != nil {
        return nil, err
    }

    instance := &DatabaseInstance{
        DeviceID:    deviceID,
        Type:        "postgres",
        ContainerID: projectName,
        InternalIP:  internalIP,
        Port:        5432,
        MasterUser:  masterUser,
        MasterPass:  masterPass,
        Databases:   []string{},
        CreatedAt:   time.Now(),
        Healthy:     true,
    }

    p.postgresInstances[deviceID] = instance

    // Store in database
    if err := p.savInstance(instance); err != nil {
        return nil, err
    }

    return instance, nil
}
```

**Step 2: Provision Database for App**

```go
func (p *DatabasePool) ProvisionDatabase(appName string, deviceID uuid.UUID) (*DatabaseCredentials, error) {
    // 1. Get or create shared instance
    instance, err := p.GetOrCreatePostgresInstance(deviceID)
    if err != nil {
        return nil, err
    }

    // 2. Generate database and user names
    dbName := fmt.Sprintf("%s_db", sanitizeName(appName))
    dbUser := fmt.Sprintf("%s_user", sanitizeName(appName))
    dbPassword := generateSecurePassword(24)

    // 3. Check if database already exists (idempotency)
    if p.databaseExists(instance, dbName) {
        log.Info("Database %s already exists, returning existing credentials", dbName)
        // TODO: Retrieve existing credentials from secure storage
        return &DatabaseCredentials{
            Host:     instance.InternalIP,
            Port:     instance.Port,
            Database: dbName,
            Username: dbUser,
            Password: dbPassword,
        }, nil
    }

    // 4. Execute SQL commands to create database and user
    device, _ := p.deviceService.GetDevice(deviceID)
    host := fmt.Sprintf("%s:22", device.IPAddress)

    sqlCommands := []string{
        fmt.Sprintf("CREATE DATABASE %s;", dbName),
        fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s';", dbUser, dbPassword),
        fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", dbName, dbUser),
    }

    for _, sql := range sqlCommands {
        execCmd := fmt.Sprintf(
            "docker exec -i %s psql -U %s -c \"%s\"",
            instance.ContainerID,
            instance.MasterUser,
            sql,
        )
        if _, err := p.sshClient.ExecuteWithTimeout(host, execCmd, 30*time.Second); err != nil {
            return nil, fmt.Errorf("failed to execute SQL: %w", err)
        }
    }

    // 5. Update instance record
    instance.Databases = append(instance.Databases, dbName)
    p.saveInstance(instance)

    // 6. Store credentials securely
    creds := &DatabaseCredentials{
        Host:     instance.InternalIP,
        Port:     instance.Port,
        Database: dbName,
        Username: dbUser,
        Password: dbPassword,
    }

    if err := p.storeCredentials(appName, deviceID, creds); err != nil {
        return nil, err
    }

    log.Info("Provisioned database %s for app %s", dbName, appName)
    return creds, nil
}
```

**Step 3: Recipe Integration**

```yaml
# backend/marketplace-recipes/nextcloud.yaml
id: nextcloud
name: NextCloud
requires_database: postgres  # NEW: Declare database requirement

compose_template: |
  version: '3.8'
  services:
    nextcloud:
      image: nextcloud:{{.Version}}
      environment:
        POSTGRES_HOST: {{.DatabaseHost}}        # Injected by system
        POSTGRES_DB: {{.DatabaseName}}          # Injected by system
        POSTGRES_USER: {{.DatabaseUsername}}    # Injected by system
        POSTGRES_PASSWORD: {{.DatabasePassword}} # Injected by system
      volumes:
        - nextcloud-data:/var/www/html
  volumes:
    nextcloud-data:
```

**Deployment Service Integration:**

```go
func (s *DeploymentService) CreateDeployment(req CreateDeploymentRequest) (*Deployment, error) {
    recipe, _ := s.recipeLoader.GetRecipe(req.RecipeSlug)

    // NEW: Check if app requires database
    if recipe.RequiresDatabase != "" {
        dbType := recipe.RequiresDatabase // "postgres", "mysql"
        creds, err := s.databasePool.ProvisionDatabase(recipe.Slug, req.DeviceID)
        if err != nil {
            return nil, fmt.Errorf("failed to provision database: %w", err)
        }

        // Inject database credentials into config
        req.Config["DatabaseHost"] = creds.Host
        req.Config["DatabaseName"] = creds.Database
        req.Config["DatabaseUsername"] = creds.Username
        req.Config["DatabasePassword"] = creds.Password

        log.Info("Using shared %s instance - saved RAM by not deploying separate database", dbType)
    }

    // Continue with normal deployment...
}
```

### Resource Savings Tracking

```go
type ResourceSavings struct {
    DeviceID             uuid.UUID
    SharedPostgresRAM    int64 // RAM used by shared instance
    AppsUsingPostgres    int   // Number of apps sharing it
    RAMIfSeparate        int64 // RAM that would be used if each had own DB
    RAMSaved             int64 // Difference
    SavingsPercentage    float64
}

func (p *DatabasePool) CalculateSavings(deviceID uuid.UUID) (*ResourceSavings, error) {
    instance, exists := p.postgresInstances[deviceID]
    if !exists {
        return nil, nil // No shared instance on this device
    }

    numApps := len(instance.Databases)
    if numApps == 0 {
        return nil, nil
    }

    sharedRAM := int64(1.5 * 1024 * 1024 * 1024) // 1.5GB for shared instance
    ramPerSeparateDB := int64(1.0 * 1024 * 1024 * 1024) // 1GB per separate instance
    totalIfSeparate := ramPerSeparateDB * int64(numApps)
    saved := totalIfSeparate - sharedRAM
    savingsPercent := (float64(saved) / float64(totalIfSeparate)) * 100

    return &ResourceSavings{
        DeviceID:             deviceID,
        SharedPostgresRAM:    sharedRAM,
        AppsUsingPostgres:    numApps,
        RAMIfSeparate:        totalIfSeparate,
        RAMSaved:             saved,
        SavingsPercentage:    savingsPercent,
    }, nil
}
```

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

### Implementation

```go
// backend/internal/services/resource_monitor.go

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

func (m *ResourceMonitor) GetAggregateResources() (*AggregateResources, error) {
    devices, err := m.deviceService.ListDevices()
    if err != nil {
        return nil, err
    }

    agg := &AggregateResources{
        LastUpdated: time.Now(),
    }

    for _, device := range devices {
        if device.Status == "online" {
            agg.TotalRAM += device.TotalRAM
            agg.UsedRAM += device.UsedRAM
            agg.TotalStorage += device.TotalStorage
            agg.UsedStorage += device.UsedStorage
            agg.TotalCPUCores += device.CPUCores
            agg.AvgCPUUsage += device.CPUUsage
            agg.OnlineDevices++

            agg.Devices = append(agg.Devices, DeviceSummary{
                ID:              device.ID,
                Name:            device.Name,
                Status:          "online",
                RAMUsagePercent: float64(device.UsedRAM) / float64(device.TotalRAM) * 100,
                CPUUsagePercent: device.CPUUsage,
                AppsRunning:     m.getAppsOnDevice(device.ID),
            })
        } else {
            agg.OfflineDevices++
            agg.Devices = append(agg.Devices, DeviceSummary{
                ID:     device.ID,
                Name:   device.Name,
                Status: "offline",
            })
        }
    }

    if agg.OnlineDevices > 0 {
        agg.AvgCPUUsage /= float64(agg.OnlineDevices)
    }

    // Calculate total apps
    deployments, _ := m.db.Find(&[]Deployment{}).Count(&agg.TotalApps)

    // Calculate RAM savings from database pooling
    for _, device := range devices {
        if device.Status == "online" {
            savings, _ := m.databasePool.CalculateSavings(device.ID)
            if savings != nil {
                agg.RAMSavedByPooling += savings.RAMSaved
            }
        }
    }

    if agg.UsedRAM > 0 {
        agg.SavingsPercentage = float64(agg.RAMSavedByPooling) / float64(agg.UsedRAM) * 100
    }

    return agg, nil
}
```

### API Endpoint

```go
// GET /api/v1/resources/aggregate
// Returns aggregate resources across all devices
```

### UI Component

```tsx
function UnifiedDashboard() {
    const { data: resources } = useAggregateResources();

    if (!resources) return <Loading />;

    return (
        <div className="unified-dashboard">
            <h1>Your Homelab</h1>
            <p>{resources.online_devices} devices online</p>

            <ResourceBars>
                <ResourceBar
                    label="RAM"
                    used={resources.used_ram}
                    total={resources.total_ram}
                    percentage={(resources.used_ram / resources.total_ram) * 100}
                />
                <ResourceBar
                    label="Storage"
                    used={resources.used_storage}
                    total={resources.total_storage}
                    percentage={(resources.used_storage / resources.total_storage) * 100}
                />
                <ResourceBar
                    label="CPU"
                    used={resources.avg_cpu_usage}
                    total={100}
                    percentage={resources.avg_cpu_usage}
                />
            </ResourceBars>

            <Stats>
                <Stat label="Apps Running" value={resources.running_apps} />
                <Stat label="Total Devices" value={resources.online_devices + resources.offline_devices} />
                <Stat
                    label="RAM Saved by Pooling"
                    value={formatBytes(resources.ram_saved_by_pooling)}
                    highlight
                />
            </Stats>

            <DeviceList devices={resources.devices} />
        </div>
    );
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
