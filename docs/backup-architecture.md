# Backup Architecture

## Overview

Automated encrypted backup system for homelab applications and infrastructure. Built on restic with S3-compatible cloud storage and local NAS support.

### Design Goals

- **Zero-Configuration** - Default backup policy applied to all apps automatically
- **Client-Side Encryption** - AES-256-CTR encryption before data leaves homelab
- **Deduplication** - Content-defined chunking minimizes storage costs
- **Flexible Destinations** - S3-compatible cloud providers and local storage
- **Granular Control** - Per-app backup schedules and retention policies
- **Unified Management** - Single interface across all devices
- **Multi-Destination** - Backup to multiple targets for redundancy

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

---

## Data Models

### BackupDestination

Stores backup target configuration (S3, local, SFTP).

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
    S3Endpoint  string          `json:"s3_endpoint,omitempty"`
    S3Bucket    string          `json:"s3_bucket,omitempty"`
    S3Region    string          `json:"s3_region,omitempty"`
    S3AccessKey string          `json:"-"` // Encrypted at rest
    S3SecretKey string          `json:"-"` // Encrypted at rest

    // Local storage configuration
    LocalDeviceID uuid.UUID     `gorm:"type:uuid" json:"local_device_id,omitempty"`
    LocalDevice   Device        `gorm:"foreignKey:LocalDeviceID" json:"-"`
    LocalPath     string        `json:"local_path,omitempty"`

    // SFTP configuration
    SFTPHost     string         `json:"sftp_host,omitempty"`
    SFTPPort     int            `json:"sftp_port,omitempty"`
    SFTPUser     string         `json:"sftp_user,omitempty"`
    SFTPPassword string         `json:"-"` // Encrypted at rest
    SFTPPath     string         `json:"sftp_path,omitempty"`

    // Repository configuration
    RepoPassword string         `json:"-"` // Encrypted at rest
    RepoInitialized bool        `gorm:"default:false"`

    // Statistics
    TotalSnapshots  int         `json:"total_snapshots"`
    TotalSize       int64       `json:"total_size"` // bytes
    LastBackupAt    *time.Time  `json:"last_backup_at"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### BackupPolicy

Defines backup schedule and retention rules.

```go
type BackupPolicy struct {
    ID          uuid.UUID   `gorm:"type:uuid;primaryKey"`
    Name        string      `gorm:"not null"`
    Description string
    IsDefault   bool        `gorm:"default:false"`

    // Schedule (cron format)
    Schedule    string      `gorm:"not null"` // e.g., "0 2 * * *"
    Enabled     bool        `gorm:"default:true"`

    // Retention policy (GFS - Grandfather-Father-Son)
    KeepLast    int         `json:"keep_last"`
    KeepHourly  int         `json:"keep_hourly"`
    KeepDaily   int         `json:"keep_daily"`
    KeepWeekly  int         `json:"keep_weekly"`
    KeepMonthly int         `json:"keep_monthly"`
    KeepYearly  int         `json:"keep_yearly"`

    // Destinations
    Destinations []BackupDestination `gorm:"many2many:backup_policy_destinations;"`

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### BackupConfiguration

Links apps/system components to backup policies.

```go
type BackupConfiguration struct {
    ID          uuid.UUID   `gorm:"type:uuid;primaryKey"`

    // What to backup
    AppID       *uuid.UUID  `gorm:"type:uuid;index" json:"app_id,omitempty"`
    App         *Deployment `gorm:"foreignKey:AppID" json:"-"`

    // System backup (database, configs)
    IsSystemBackup bool     `gorm:"default:false"`
    SystemType     string   `json:"system_type,omitempty"` // "database", "platform_config"

    // Policy
    PolicyID    uuid.UUID     `gorm:"type:uuid;not null"`
    Policy      BackupPolicy  `gorm:"foreignKey:PolicyID"`

    // Backup scope
    BackupVolumes   bool      `gorm:"default:true"`
    BackupCompose   bool      `gorm:"default:true"`
    BackupEnv       bool      `gorm:"default:true"`

    // Paths
    IncludePaths    []string  `gorm:"type:json"`
    ExcludePaths    []string  `gorm:"type:json"`

    Enabled     bool          `gorm:"default:true"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### BackupSnapshot

Records individual backup snapshots.

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
    ResticSnapshotID string        `gorm:"not null;index"`
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

### 1. Initial Setup

**Add S3 Destination:**
- Generate secure repository password (32-byte random)
- Test S3 connectivity
- Initialize restic repository with encryption
- Store encrypted credentials in database

**Add Local Destination:**
- Verify device accessibility via SSH
- Create backup directory
- Initialize restic repository
- Store configuration in database

### 2. Configure Backup Policy

**Default Policy (Auto-applied to new apps):**
- Schedule: Daily at 2 AM (`0 2 * * *`)
- Retention: 7 daily, 4 weekly, 6 monthly snapshots
- GFS (Grandfather-Father-Son) rotation

**Custom Policies:**
- High-frequency: Hourly backups for critical apps
- Low-frequency: Weekly backups for static data
- Per-app retention overrides

### 3. Automated Backup Execution

**Scheduler:**
- Cron-based scheduler checks for due backups every minute
- Executes backups in parallel for multiple apps
- Tracks snapshot status in database

**Backup Process:**
1. Determine backup paths (Docker volumes, compose files, env vars)
2. Execute restic backup to all configured destinations
3. Parse restic JSON output for statistics
4. Update snapshot record with results

### 4. Retention Policy Enforcement

**Prune Snapshots:**
- Daily cleanup job removes old snapshots per retention policy
- Runs restic `forget --prune` with policy parameters
- Frees storage space from deleted snapshots

### 5. Restore Operations

**List Snapshots:**
- Query restic repository for available snapshots
- Filter by app ID or date range
- Display snapshot statistics

**Restore Process:**
- Full restore: Extract all files to target path
- Selective restore: Extract specific files/directories
- Verify restoration success

---

## Security

### Encryption

**Client-Side Encryption:**
- AES-256-CTR with Poly1305-AES authentication
- Repository password never transmitted to destination
- Encryption keys derived from password using scrypt KDF

**Key Management:**
- Repository passwords stored encrypted in platform database
- Database encryption key from master password or hardware key
- Support for key rotation

### Access Control

**Backup Destination Credentials:**
- S3 credentials encrypted at rest
- IAM policies limit S3 bucket permissions (write-only for backup user)
- S3 bucket policies enforce retention locks

**restic Repository Access:**
- Repository password required for all operations
- No access to backups without password
- Append-only mode prevents snapshot deletion

### Compliance

**Data Residency:**
- Choose destination location for compliance requirements
- Support for multiple destinations in different regions

**Immutability:**
- S3 Object Lock for WORM (write-once-read-many) compliance
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
  "repo_initialized": true
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
```

### Backup Operations

**Trigger Immediate Backup:**
```
POST /api/v1/backup/configurations/:id/backup

Response:
{
  "snapshot_id": "uuid",
  "status": "in_progress"
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
      "duration": 120
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
  "files": ["path/to/file1", "path/to/file2"]
}

Response:
{
  "success": true,
  "message": "Restore completed successfully",
  "restored_path": "/tmp/restore"
}
```

---

## Storage Efficiency

### Deduplication Savings

**Example: 5 Apps with 200 GB Total Data**

Without deduplication (separate backup tools):
- Daily backups, 7 days retention
- 200 GB × 7 days = 1,400 GB per week
- Backblaze B2 cost: $9.80/month

With restic deduplication:
- Initial: 200 GB
- Daily change: 2% = 4 GB/day
- 7 days: 200 GB + (28 GB) = 228 GB
- Backblaze B2 cost: $1.60/month

**Savings: 85% reduction in storage costs**

---

**Version:** 1.0
**Last Updated:** October 2025
