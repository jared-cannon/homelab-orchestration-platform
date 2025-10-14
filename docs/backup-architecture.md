# Backup Architecture

**Version:** 1.0
**Last Updated:** October 2025
**Purpose:** Automated encrypted backup system for homelab applications and infrastructure

---

## Overview

The backup system provides automated, encrypted, incremental backups of application data and platform configurations to S3-compatible cloud storage or local NAS devices. Built on restic, the system ensures data protection with zero-configuration defaults while allowing granular control for advanced users.

### Design Goals

- **Zero-Configuration**: Default backup policy applied to all apps automatically
- **Client-Side Encryption**: All data encrypted before leaving the homelab (AES-256-CTR)
- **Deduplication**: Minimize storage costs through content-defined chunking
- **Flexible Destinations**: Support S3-compatible cloud providers and local storage
- **Granular Control**: Per-app backup schedules and retention policies
- **Unified Management**: Single interface for all backup operations across all devices
- **Multi-Destination**: Backup to multiple targets for redundancy

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────┐
│           Homelab Orchestration Platform            │
│                                                       │
│  ┌──────────────────┐      ┌────────────────────┐  │
│  │  BackupService   │      │  BackupScheduler   │  │
│  │  - Repository mgmt│      │  - Cron jobs       │  │
│  │  - Snapshot ops   │      │  - Retention       │  │
│  │  - Restore ops    │      │  - Health checks   │  │
│  └────────┬─────────┘      └─────────┬──────────┘  │
│           │                           │              │
│           └───────────┬───────────────┘              │
│                       │                              │
│            ┌──────────▼──────────┐                   │
│            │  restic Repository  │                   │
│            │  - Encryption       │                   │
│            │  - Deduplication    │                   │
│            │  - Snapshots        │                   │
│            └──────────┬──────────┘                   │
└───────────────────────┼──────────────────────────────┘
                        │
          ┌─────────────┴─────────────┐
          │                           │
          ▼                           ▼
┌──────────────────┐        ┌──────────────────┐
│  Cloud Storage   │        │  Local Storage   │
│                  │        │                  │
│  S3 Compatible:  │        │  NFS Server      │
│  - Backblaze B2  │        │  - NAS device    │
│  - Wasabi        │        │  - Dedicated     │
│  - AWS S3        │        │    server        │
│  - MinIO         │        │  - USB drive     │
└──────────────────┘        └──────────────────┘
```

### Database Models

#### BackupDestination

```go
type DestinationType string

const (
    DestinationS3    DestinationType = "s3"
    DestinationLocal DestinationType = "local"
    DestinationSFTP  DestinationType = "sftp"
)

type BackupDestination struct {
    ID          uuid.UUID       `gorm:"type:uuid;primaryKey"`
    Name        string          `gorm:"not null"`
    Type        DestinationType `gorm:"not null"`
    Enabled     bool            `gorm:"default:true"`

    // S3-compatible configuration
    S3Endpoint  string          `json:"s3_endpoint,omitempty"` // e.g., s3.us-west-002.backblazeb2.com
    S3Bucket    string          `json:"s3_bucket,omitempty"`
    S3Region    string          `json:"s3_region,omitempty"`
    S3AccessKey string          `json:"-"` // Encrypted at rest
    S3SecretKey string          `json:"-"` // Encrypted at rest

    // Local storage configuration
    LocalDeviceID uuid.UUID     `gorm:"type:uuid" json:"local_device_id,omitempty"`
    LocalDevice   Device        `gorm:"foreignKey:LocalDeviceID" json:"-"`
    LocalPath     string        `json:"local_path,omitempty"` // e.g., /mnt/nfs/backups

    // SFTP configuration
    SFTPHost     string         `json:"sftp_host,omitempty"`
    SFTPPort     int            `json:"sftp_port,omitempty"`
    SFTPUser     string         `json:"sftp_user,omitempty"`
    SFTPPassword string         `json:"-"` // Encrypted at rest
    SFTPPath     string         `json:"sftp_path,omitempty"`

    // Repository configuration
    RepoPassword string         `json:"-"` // Encrypted at rest, restic repository password
    RepoInitialized bool        `gorm:"default:false"`

    // Statistics
    TotalSnapshots  int         `json:"total_snapshots"`
    TotalSize       int64       `json:"total_size"` // bytes
    LastBackupAt    *time.Time  `json:"last_backup_at"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### BackupPolicy

```go
type BackupPolicy struct {
    ID          uuid.UUID   `gorm:"type:uuid;primaryKey"`
    Name        string      `gorm:"not null"`
    Description string
    IsDefault   bool        `gorm:"default:false"` // Default policy for new apps

    // Schedule (cron format)
    Schedule    string      `gorm:"not null"` // e.g., "0 2 * * *" (daily at 2 AM)
    Enabled     bool        `gorm:"default:true"`

    // Retention policy
    KeepLast    int         `json:"keep_last"`    // Keep last N snapshots (0 = disabled)
    KeepHourly  int         `json:"keep_hourly"`  // Keep hourly for last N hours
    KeepDaily   int         `json:"keep_daily"`   // Keep daily for last N days
    KeepWeekly  int         `json:"keep_weekly"`  // Keep weekly for last N weeks
    KeepMonthly int         `json:"keep_monthly"` // Keep monthly for last N months
    KeepYearly  int         `json:"keep_yearly"`  // Keep yearly for last N years

    // Destinations
    Destinations []BackupDestination `gorm:"many2many:backup_policy_destinations;"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### BackupConfiguration

```go
type BackupConfiguration struct {
    ID          uuid.UUID   `gorm:"type:uuid;primaryKey"`

    // What to backup
    AppID       *uuid.UUID  `gorm:"type:uuid;index" json:"app_id,omitempty"`
    App         *Deployment `gorm:"foreignKey:AppID" json:"-"`

    // If AppID is null, this is a system backup (database, configs)
    IsSystemBackup bool     `gorm:"default:false"`
    SystemType     string   `json:"system_type,omitempty"` // "database", "platform_config"

    // Policy
    PolicyID    uuid.UUID     `gorm:"type:uuid;not null"`
    Policy      BackupPolicy  `gorm:"foreignKey:PolicyID"`

    // Backup scope for apps
    BackupVolumes   bool      `gorm:"default:true"`  // Docker volumes
    BackupCompose   bool      `gorm:"default:true"`  // docker-compose.yml
    BackupEnv       bool      `gorm:"default:true"`  // Environment variables

    // Paths to include (relative to app volume)
    IncludePaths    []string  `gorm:"type:json"`
    ExcludePaths    []string  `gorm:"type:json"`

    Enabled     bool          `gorm:"default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### BackupSnapshot

```go
type SnapshotStatus string

const (
    SnapshotStatusSuccess    SnapshotStatus = "success"
    SnapshotStatusFailed     SnapshotStatus = "failed"
    SnapshotStatusInProgress SnapshotStatus = "in_progress"
)

type BackupSnapshot struct {
    ID              uuid.UUID      `gorm:"type:uuid;primaryKey"`
    ConfigurationID uuid.UUID      `gorm:"type:uuid;not null;index"`
    Configuration   BackupConfiguration `gorm:"foreignKey:ConfigurationID"`

    DestinationID   uuid.UUID      `gorm:"type:uuid;not null"`
    Destination     BackupDestination `gorm:"foreignKey:DestinationID"`

    // restic snapshot info
    ResticSnapshotID string        `gorm:"not null;index"` // restic's snapshot ID
    Hostname         string
    Paths            []string      `gorm:"type:json"`

    // Statistics
    Status          SnapshotStatus `gorm:"not null"`
    FilesNew        int            `json:"files_new"`
    FilesChanged    int            `json:"files_changed"`
    FilesUnmodified int            `json:"files_unmodified"`
    DataAdded       int64          `json:"data_added"` // bytes
    TotalSize       int64          `json:"total_size"` // bytes after dedup
    Duration        int            `json:"duration"`   // seconds

    // Error tracking
    ErrorMessage    string         `json:"error_message,omitempty"`

    // Timestamps
    StartedAt   time.Time      `json:"started_at"`
    CompletedAt *time.Time     `json:"completed_at,omitempty"`
    CreatedAt   time.Time
}
```

---

## Backup Workflows

### 1. Initial Setup: Add Backup Destination

**S3-Compatible Cloud Storage:**

```go
func (s *BackupService) AddS3Destination(req AddS3DestinationRequest) (*BackupDestination, error) {
    // 1. Generate secure repository password (32-byte random)
    repoPassword := generateSecurePassword(32)

    // 2. Create destination record
    dest := &BackupDestination{
        ID:          uuid.New(),
        Name:        req.Name,
        Type:        DestinationS3,
        S3Endpoint:  req.Endpoint,
        S3Bucket:    req.Bucket,
        S3Region:    req.Region,
        S3AccessKey: encrypt(req.AccessKey), // Encrypt credentials
        S3SecretKey: encrypt(req.SecretKey),
        RepoPassword: encrypt(repoPassword),
        Enabled:     true,
    }

    // 3. Test S3 connectivity
    if err := s.testS3Connection(dest); err != nil {
        return nil, fmt.Errorf("S3 connection failed: %w", err)
    }

    // 4. Initialize restic repository
    if err := s.initResticRepo(dest); err != nil {
        return nil, fmt.Errorf("failed to initialize repository: %w", err)
    }

    dest.RepoInitialized = true
    s.db.Create(dest)

    return dest, nil
}
```

**restic Repository Initialization:**

```bash
# Set environment variables
export RESTIC_REPOSITORY="s3:s3.us-west-002.backblazeb2.com/my-bucket"
export RESTIC_PASSWORD="<generated-repo-password>"
export AWS_ACCESS_KEY_ID="<s3-access-key>"
export AWS_SECRET_ACCESS_KEY="<s3-secret-key>"

# Initialize repository (creates encryption keys)
restic init

# Output:
# created restic repository abc123 at s3:s3.us-west-002.backblazeb2.com/my-bucket
# Please note that knowledge of your password is required to access the repository.
```

**Local NAS Storage:**

```go
func (s *BackupService) AddLocalDestination(req AddLocalDestinationRequest) (*BackupDestination, error) {
    // 1. Verify device exists and is accessible
    device, err := s.deviceService.GetDevice(req.DeviceID)
    if err != nil {
        return nil, fmt.Errorf("device not found: %w", err)
    }

    // 2. Create backup directory on device
    backupPath := filepath.Join(req.Path, "homelab-backups")
    if err := s.sshService.Exec(device, fmt.Sprintf("mkdir -p %s", backupPath)); err != nil {
        return nil, fmt.Errorf("failed to create backup directory: %w", err)
    }

    // 3. Generate repository password
    repoPassword := generateSecurePassword(32)

    // 4. Create destination record
    dest := &BackupDestination{
        ID:            uuid.New(),
        Name:          req.Name,
        Type:          DestinationLocal,
        LocalDeviceID: device.ID,
        LocalPath:     backupPath,
        RepoPassword:  encrypt(repoPassword),
        Enabled:       true,
    }

    // 5. Initialize restic repository
    if err := s.initResticRepo(dest); err != nil {
        return nil, fmt.Errorf("failed to initialize repository: %w", err)
    }

    dest.RepoInitialized = true
    s.db.Create(dest)

    return dest, nil
}
```

### 2. Configure Backup Policy

**Default Policy (Applied to All New Apps):**

```go
func (s *BackupService) CreateDefaultPolicy(destinations []uuid.UUID) (*BackupPolicy, error) {
    policy := &BackupPolicy{
        ID:          uuid.New(),
        Name:        "Default Daily Backups",
        Description: "Daily backups at 2 AM, retain 7 daily, 4 weekly, 6 monthly",
        IsDefault:   true,
        Schedule:    "0 2 * * *", // Daily at 2 AM
        Enabled:     true,

        // Retention: GFS (Grandfather-Father-Son)
        KeepDaily:   7,  // Keep daily for 1 week
        KeepWeekly:  4,  // Keep weekly for 1 month
        KeepMonthly: 6,  // Keep monthly for 6 months
    }

    // Associate destinations
    for _, destID := range destinations {
        dest, _ := s.GetDestination(destID)
        policy.Destinations = append(policy.Destinations, *dest)
    }

    s.db.Create(policy)
    return policy, nil
}
```

**Custom Per-App Policy:**

```go
// High-frequency backups for critical app (e.g., password manager)
policy := &BackupPolicy{
    Name:        "Critical App - Hourly Backups",
    Schedule:    "0 * * * *", // Every hour
    KeepHourly:  24,  // Keep hourly for 24 hours
    KeepDaily:   30,  // Keep daily for 30 days
    KeepMonthly: 12,  // Keep monthly for 1 year
}

// Low-frequency backups for static data (e.g., media server)
policy := &BackupPolicy{
    Name:        "Media Server - Weekly Backups",
    Schedule:    "0 3 * * 0", // Sundays at 3 AM
    KeepWeekly:  4,   // Keep weekly for 1 month
    KeepMonthly: 12,  // Keep monthly for 1 year
}
```

### 3. Automated Backup Execution

**Backup Scheduler (Cron-based):**

```go
func (s *BackupScheduler) Start() {
    c := cron.New()

    // Check for due backups every minute
    c.AddFunc("* * * * *", func() {
        s.runDueBackups()
    })

    // Cleanup old snapshots daily at 4 AM
    c.AddFunc("0 4 * * *", func() {
        s.pruneSnapshots()
    })

    c.Start()
}

func (s *BackupScheduler) runDueBackups() {
    configs, _ := s.backupService.GetDueBackupConfigurations()

    for _, config := range configs {
        go s.executeBackup(config)
    }
}
```

**Execute Backup:**

```go
func (s *BackupService) ExecuteBackup(config *BackupConfiguration) (*BackupSnapshot, error) {
    // 1. Create snapshot record
    snapshot := &BackupSnapshot{
        ID:              uuid.New(),
        ConfigurationID: config.ID,
        Status:          SnapshotStatusInProgress,
        StartedAt:       time.Now(),
    }
    s.db.Create(snapshot)

    // 2. Determine what to backup
    var paths []string
    if config.AppID != nil {
        // App backup
        app, _ := s.deploymentService.GetDeployment(*config.AppID)

        if config.BackupVolumes {
            // Get Docker volume paths
            volumes := s.dockerService.GetVolumes(app.DeviceID, app.ID)
            for _, vol := range volumes {
                paths = append(paths, vol.Mountpoint)
            }
        }

        if config.BackupCompose {
            // Backup docker-compose.yml
            composePath := fmt.Sprintf("/tmp/backup-%s/compose", app.ID)
            s.exportComposeFile(app, composePath)
            paths = append(paths, composePath)
        }
    } else if config.IsSystemBackup {
        // System backup (database, configs)
        paths = s.getSystemBackupPaths(config.SystemType)
    }

    // 3. Execute restic backup for each destination
    for _, dest := range config.Policy.Destinations {
        if !dest.Enabled {
            continue
        }

        if err := s.resticBackup(dest, paths, snapshot); err != nil {
            snapshot.Status = SnapshotStatusFailed
            snapshot.ErrorMessage = err.Error()
            snapshot.CompletedAt = timePtr(time.Now())
            s.db.Save(snapshot)
            return snapshot, err
        }
    }

    // 4. Update snapshot status
    snapshot.Status = SnapshotStatusSuccess
    snapshot.CompletedAt = timePtr(time.Now())
    snapshot.Duration = int(time.Since(snapshot.StartedAt).Seconds())
    s.db.Save(snapshot)

    return snapshot, nil
}
```

**restic Backup Command:**

```go
func (s *BackupService) resticBackup(dest *BackupDestination, paths []string, snapshot *BackupSnapshot) error {
    // Set environment variables
    env := s.buildResticEnv(dest)

    // Build restic command
    args := []string{"backup"}
    args = append(args, paths...)
    args = append(args, "--json") // Get structured output

    // Execute restic
    cmd := exec.Command("restic", args...)
    cmd.Env = env

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("restic backup failed: %w", err)
    }

    // Parse restic JSON output
    var result ResticBackupResult
    json.Unmarshal(output, &result)

    // Update snapshot statistics
    snapshot.ResticSnapshotID = result.SnapshotID
    snapshot.FilesNew = result.FilesNew
    snapshot.FilesChanged = result.FilesChanged
    snapshot.FilesUnmodified = result.FilesUnmodified
    snapshot.DataAdded = result.DataAdded
    snapshot.TotalSize = result.TotalSize

    return nil
}
```

### 4. Retention Policy Enforcement

**Prune Old Snapshots:**

```go
func (s *BackupService) PruneSnapshots(dest *BackupDestination, policy *BackupPolicy) error {
    env := s.buildResticEnv(dest)

    args := []string{"forget", "--prune", "--json"}

    // Apply retention policy
    if policy.KeepLast > 0 {
        args = append(args, fmt.Sprintf("--keep-last=%d", policy.KeepLast))
    }
    if policy.KeepHourly > 0 {
        args = append(args, fmt.Sprintf("--keep-hourly=%d", policy.KeepHourly))
    }
    if policy.KeepDaily > 0 {
        args = append(args, fmt.Sprintf("--keep-daily=%d", policy.KeepDaily))
    }
    if policy.KeepWeekly > 0 {
        args = append(args, fmt.Sprintf("--keep-weekly=%d", policy.KeepWeekly))
    }
    if policy.KeepMonthly > 0 {
        args = append(args, fmt.Sprintf("--keep-monthly=%d", policy.KeepMonthly))
    }
    if policy.KeepYearly > 0 {
        args = append(args, fmt.Sprintf("--keep-yearly=%d", policy.KeepYearly))
    }

    cmd := exec.Command("restic", args...)
    cmd.Env = env

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("restic forget failed: %w", err)
    }

    return nil
}
```

### 5. Restore Operations

**List Available Snapshots:**

```go
func (s *BackupService) ListSnapshots(dest *BackupDestination, appID *uuid.UUID) ([]ResticSnapshot, error) {
    env := s.buildResticEnv(dest)

    args := []string{"snapshots", "--json"}
    if appID != nil {
        args = append(args, fmt.Sprintf("--tag=app:%s", appID.String()))
    }

    cmd := exec.Command("restic", args...)
    cmd.Env = env

    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var snapshots []ResticSnapshot
    json.Unmarshal(output, &snapshots)

    return snapshots, nil
}
```

**Restore Snapshot:**

```go
func (s *BackupService) RestoreSnapshot(dest *BackupDestination, snapshotID string, targetPath string) error {
    env := s.buildResticEnv(dest)

    args := []string{
        "restore",
        snapshotID,
        "--target", targetPath,
    }

    cmd := exec.Command("restic", args...)
    cmd.Env = env

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("restic restore failed: %w", err)
    }

    return nil
}
```

**Restore Specific Files:**

```go
func (s *BackupService) RestoreFiles(dest *BackupDestination, snapshotID string, files []string, targetPath string) error {
    env := s.buildResticEnv(dest)

    args := []string{
        "restore",
        snapshotID,
        "--target", targetPath,
    }

    // Add include patterns for specific files
    for _, file := range files {
        args = append(args, "--include", file)
    }

    cmd := exec.Command("restic", args...)
    cmd.Env = env

    if err := cmd.Run(); err != nil {
        return fmt.Errorf("restic restore failed: %w", err)
    }

    return nil
}
```

---

## Security

### Encryption

**Client-Side Encryption:**
- All data encrypted on the homelab before transmission
- AES-256-CTR with Poly1305-AES for authentication
- Repository password never transmitted to backup destination
- Encryption keys derived from repository password using scrypt (KDF)

**Key Management:**
- Repository passwords stored encrypted in platform database
- Database encryption key derived from master password or hardware key
- Support for key rotation without re-encrypting all snapshots

### Access Control

**Backup Destination Credentials:**
- S3 credentials encrypted at rest in platform database
- Use IAM policies to limit S3 bucket permissions (write-only for backup user)
- Support for S3 bucket policies to enforce retention locks

**restic Repository Access:**
- Repository password required for all operations (backup, restore, prune)
- No ability to access backups without repository password
- Support for append-only mode (prevents deletion of snapshots)

### Compliance

**Data Residency:**
- Choose backup destination location for compliance (e.g., EU-only S3 regions)
- Support for multiple destinations in different regions

**Immutability:**
- S3 Object Lock support for write-once-read-many (WORM) compliance
- Prevents ransomware from deleting backups

---

## API Reference

### Backup Destinations

**Add S3 Destination:**

```
POST /api/v1/backup/destinations/s3

Request:
{
  "name": "Backblaze B2",
  "endpoint": "s3.us-west-002.backblazeb2.com",
  "bucket": "homelab-backups",
  "region": "us-west-002",
  "access_key": "...",
  "secret_key": "..."
}

Response:
{
  "id": "uuid",
  "name": "Backblaze B2",
  "type": "s3",
  "repo_initialized": true,
  "created_at": "2025-10-13T..."
}
```

**Add Local Destination:**

```
POST /api/v1/backup/destinations/local

Request:
{
  "name": "NAS Backups",
  "device_id": "uuid",
  "path": "/mnt/nfs/backups"
}

Response:
{
  "id": "uuid",
  "name": "NAS Backups",
  "type": "local",
  "local_device_id": "uuid",
  "local_path": "/mnt/nfs/backups/homelab-backups",
  "repo_initialized": true
}
```

### Backup Policies

**Create Policy:**

```
POST /api/v1/backup/policies

Request:
{
  "name": "Daily Backups",
  "schedule": "0 2 * * *",
  "keep_daily": 7,
  "keep_weekly": 4,
  "keep_monthly": 6,
  "destination_ids": ["uuid1", "uuid2"]
}

Response:
{
  "id": "uuid",
  "name": "Daily Backups",
  "schedule": "0 2 * * *",
  "enabled": true,
  "destinations": [...]
}
```

### Backup Configuration

**Enable Backup for App:**

```
POST /api/v1/backup/configurations

Request:
{
  "app_id": "uuid",
  "policy_id": "uuid",
  "backup_volumes": true,
  "backup_compose": true,
  "backup_env": false
}

Response:
{
  "id": "uuid",
  "app_id": "uuid",
  "policy_id": "uuid",
  "enabled": true
}
```

### Backup Operations

**Trigger Immediate Backup:**

```
POST /api/v1/backup/configurations/:id/backup

Response:
{
  "snapshot_id": "uuid",
  "status": "in_progress",
  "started_at": "2025-10-13T..."
}
```

**List Snapshots:**

```
GET /api/v1/backup/snapshots?app_id=uuid

Response:
{
  "snapshots": [
    {
      "id": "uuid",
      "restic_snapshot_id": "abc123",
      "status": "success",
      "data_added": 1073741824,
      "total_size": 5368709120,
      "duration": 120,
      "completed_at": "2025-10-13T..."
    }
  ]
}
```

**Restore Snapshot:**

```
POST /api/v1/backup/snapshots/:id/restore

Request:
{
  "target_device_id": "uuid",
  "target_path": "/tmp/restore",
  "files": ["path/to/file1", "path/to/file2"] // Optional: specific files
}

Response:
{
  "success": true,
  "message": "Restore completed successfully",
  "restored_path": "/tmp/restore"
}
```

---

## Frontend UX

### Backup Settings Page

**Location:** Settings → Backups

```
┌───────────────────────────────────────────────────────────┐
│ Backup Settings                          [+ Add Destination]│
├───────────────────────────────────────────────────────────┤
│                                                             │
│ Backup Destinations                                         │
│ ┌─────────────────────────────────────────────────────┐   │
│ │ 🌐 Backblaze B2                            [Enabled] │   │
│ │    Bucket: homelab-backups                          │   │
│ │    Last backup: 2 hours ago                         │   │
│ │    Total size: 45.2 GB (120 snapshots)              │   │
│ │    [Test Connection] [Edit] [Remove]                 │   │
│ └─────────────────────────────────────────────────────┘   │
│                                                             │
│ ┌─────────────────────────────────────────────────────┐   │
│ │ 💾 NAS Backups                             [Enabled] │   │
│ │    Device: nas-01                                   │   │
│ │    Path: /mnt/nfs/backups                           │   │
│ │    Last backup: 3 hours ago                         │   │
│ │    Total size: 42.8 GB (115 snapshots)              │   │
│ │    [Test Connection] [Edit] [Remove]                 │   │
│ └─────────────────────────────────────────────────────┘   │
│                                                             │
│ Backup Policies                            [+ Create Policy]│
│ ┌─────────────────────────────────────────────────────┐   │
│ │ Default Daily Backups                   [Default ✓] │   │
│ │    Schedule: Daily at 2:00 AM                       │   │
│ │    Retention: 7 daily, 4 weekly, 6 monthly          │   │
│ │    Destinations: Backblaze B2, NAS Backups          │   │
│ │    Apps using: 12                                    │   │
│ │    [Edit] [Duplicate]                                │   │
│ └─────────────────────────────────────────────────────┘   │
└───────────────────────────────────────────────────────────┘
```

### App Backup Settings

**Location:** App Detail Page → Backups Tab

```
┌───────────────────────────────────────────────────────────┐
│ NextCloud - Backups                     [Backup Now]      │
├───────────────────────────────────────────────────────────┤
│                                                             │
│ Backup Configuration                                        │
│   Policy: [Default Daily Backups      ▼]                  │
│   Status: ● Enabled                                        │
│                                                             │
│   What to backup:                                          │
│   ☑ Docker volumes (nextcloud_data, nextcloud_config)     │
│   ☑ Docker Compose file                                   │
│   ☐ Environment variables                                 │
│                                                             │
│ Backup History                                              │
│ ┌─────────────────────────────────────────────────────┐   │
│ │ 2 hours ago         ✓  45.2 GB  (+125 MB)    [Restore] │ │
│ │ 1 day ago           ✓  45.1 GB  (+89 MB)     [Restore] │ │
│ │ 2 days ago          ✓  45.0 GB  (+210 MB)    [Restore] │ │
│ │ 3 days ago          ✓  44.8 GB  (+45 MB)     [Restore] │ │
│ └─────────────────────────────────────────────────────┘   │
│                                                             │
│ Storage Usage                                               │
│   Total: 180.1 GB across 4 destinations                    │
│   Deduplication savings: 42% (130 GB saved)                │
└───────────────────────────────────────────────────────────┘
```

### Add S3 Destination Wizard

```
┌───────────────────────────────────────────────────────┐
│ Add S3-Compatible Backup Destination                   │
├───────────────────────────────────────────────────────┤
│                                                         │
│ Provider: [Custom S3-Compatible ▼]                    │
│           (Backblaze B2, Wasabi, AWS S3, MinIO...)    │
│                                                         │
│ Name: [Backblaze B2_____________]                      │
│                                                         │
│ S3 Configuration:                                       │
│   Endpoint:   [s3.us-west-002.backblazeb2.com_____]   │
│   Bucket:     [homelab-backups________________]        │
│   Region:     [us-west-002____________________]        │
│   Access Key: [0012abc...___________________]          │
│   Secret Key: [••••••••••••••••••••••••••••]          │
│                                                         │
│ Encryption:                                             │
│   ● Generate secure repository password (recommended)  │
│   ○ Use custom repository password                     │
│                                                         │
│ [Test Connection]  [Cancel]  [Add Destination]         │
└───────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Core Backup Infrastructure

- [ ] BackupService with restic integration
- [ ] BackupDestination model and API (S3 + local)
- [ ] Repository initialization and testing
- [ ] Basic backup execution (single snapshot)
- [ ] Backup status tracking

### Phase 2: Scheduling and Policies

- [ ] BackupPolicy model and configuration
- [ ] Cron-based scheduler
- [ ] Retention policy enforcement (forget/prune)
- [ ] Default policy for all apps
- [ ] Per-app policy overrides

### Phase 3: Restore Operations

- [ ] List snapshots API
- [ ] Full restore to target path
- [ ] Selective file restore
- [ ] Restore validation
- [ ] One-click app state restore

### Phase 4: UI and Monitoring

- [ ] Backup settings page (destinations, policies)
- [ ] Per-app backup configuration
- [ ] Backup history view
- [ ] Storage usage dashboard
- [ ] Backup health monitoring and alerts

### Phase 5: Advanced Features

- [ ] Multi-destination redundancy
- [ ] Backup verification (check repository integrity)
- [ ] Bandwidth limiting for backups
- [ ] Pre/post backup hooks (database dumps, app-specific prep)
- [ ] Backup import/export for migration

---

## Storage Cost Estimation

### Example: 5 Apps with 200 GB Total Data

**Without Deduplication (5 separate backup tools):**
- Daily backups, 7 days retention
- 200 GB × 7 days = **1,400 GB per week**
- Monthly cost (Backblaze B2): $7/TB = **$9.80/month**

**With restic Deduplication:**
- Initial backup: 200 GB
- Daily change rate: 2% = 4 GB/day
- 7 days: 200 GB + (4 GB × 7) = **228 GB**
- Monthly cost: **$1.60/month**

**Savings: 85% reduction in storage costs**

---

## Best Practices

### Repository Password Management

- Store repository passwords in secure secret manager
- Use different repository passwords per destination
- Backup repository passwords offline (encrypted password manager)
- Test restoration regularly to verify password validity

### Backup Testing

- Schedule monthly restore tests
- Verify backup integrity with `restic check`
- Test restores to different devices
- Document restore procedures

### Performance Optimization

- Schedule backups during low-usage hours
- Use bandwidth limiting for large backups
- Exclude temporary files and caches
- Consider separate policies for large static data

### Security Hardening

- Use append-only S3 bucket policies to prevent ransomware deletion
- Enable MFA for S3 bucket access
- Restrict S3 IAM user to minimum required permissions
- Enable S3 Object Lock for immutable backups
- Monitor backup logs for suspicious activity

---

## Troubleshooting

### restic Repository Locked

**Problem:** "repository is already locked"

**Cause:** Previous backup operation did not complete cleanly

**Fix:**
```bash
restic unlock
```

### S3 Connection Failed

**Problem:** "connection refused" or "403 Forbidden"

**Cause:** Invalid credentials or incorrect endpoint

**Fix:**
- Verify S3 access key and secret key
- Check S3 endpoint URL format
- Ensure bucket exists and region is correct
- Test with `restic cat config` to verify repository access

### Backup Too Slow

**Problem:** Backup takes hours to complete

**Cause:** Network bandwidth constraints or CPU-bound deduplication

**Fix:**
- Enable bandwidth limiting: `--limit-upload 10240` (KB/s)
- Exclude unnecessary paths
- Use faster storage for cache directory
- Consider local backup destination for speed

---

**End of Document**
