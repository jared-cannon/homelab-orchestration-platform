# Database Schema

## Migration Strategy

This project uses **GORM AutoMigrate** for database schema management. When the server starts, GORM automatically:
- Creates missing tables
- Adds missing columns to existing tables
- Creates missing indexes
- **Does not** drop columns, tables, or modify existing constraints

### How AutoMigrate Works

In `cmd/server/main.go`, the `initDB()` function calls:

```go
db.AutoMigrate(
    &models.User{},
    &models.Device{},
    &models.DeviceMetrics{},
    &models.Application{},
    &models.Deployment{},
    &models.Credential{},
    &models.InstalledSoftware{},
    &models.SoftwareInstallation{},
    &models.NFSExport{},
    &models.NFSMount{},
    &models.Volume{},
)
```

This means:
- Schema changes are defined in Go model structs
- No separate SQL migration files needed
- Changes apply automatically on server restart
- Backward compatible (adds columns, doesn't remove)

### When to Manually Migrate

AutoMigrate **cannot**:
- Rename columns
- Drop columns
- Modify column types in incompatible ways
- Migrate data

For these cases, you must:
1. Back up the database: `cp homelab.db homelab.db.backup`
2. Write custom migration code or SQL
3. Test thoroughly before deploying

## Resource Monitoring Schema Changes

### Device Model Extensions

The `Device` model now includes real-time resource metrics:

```go
type Device struct {
    // ... existing fields ...

    // Current resource metrics (updated by ResourceMonitoringService)
    CPUUsagePercent    *float64   `json:"cpu_usage_percent" gorm:"column:cpu_usage_percent"`
    CPUCores           *int       `json:"cpu_cores" gorm:"column:cpu_cores"`
    TotalRAMMB         *int       `json:"total_ram_mb" gorm:"column:total_ram_mb"`
    UsedRAMMB          *int       `json:"used_ram_mb" gorm:"column:used_ram_mb"`
    AvailableRAMMB     *int       `json:"available_ram_mb" gorm:"column:available_ram_mb"`
    TotalStorageGB     *int       `json:"total_storage_gb" gorm:"column:total_storage_gb"`
    UsedStorageGB      *int       `json:"used_storage_gb" gorm:"column:used_storage_gb"`
    AvailableStorageGB *int       `json:"available_storage_gb" gorm:"column:available_storage_gb"`
    ResourcesUpdatedAt *time.Time `json:"resources_updated_at" gorm:"column:resources_updated_at"`
}
```

**Design Decisions:**
- All resource fields are **nullable pointers** (`*float64`, `*int`, `*time.Time`)
- This allows distinguishing between "no data" and "zero value"
- `ResourcesUpdatedAt` tracks freshness of metrics
- Metrics are cleared after 3 consecutive collection failures (stale data handling)

**When Applied:** Automatically on server restart after merging resource monitoring changes

### DeviceMetrics Table (New)

Historical metrics storage for time-series data:

```go
type DeviceMetrics struct {
    ID                 uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
    DeviceID           uuid.UUID `json:"device_id" gorm:"type:uuid;not null;index"`
    Device             *Device   `json:"device,omitempty" gorm:"foreignKey:DeviceID"`

    CPUUsagePercent    float64   `json:"cpu_usage_percent" gorm:"not null"`
    CPUCores           int       `json:"cpu_cores" gorm:"not null"`

    TotalRAMMB         int       `json:"total_ram_mb" gorm:"not null"`
    UsedRAMMB          int       `json:"used_ram_mb" gorm:"not null"`
    AvailableRAMMB     int       `json:"available_ram_mb" gorm:"not null"`

    TotalStorageGB     int       `json:"total_storage_gb" gorm:"not null"`
    UsedStorageGB      int       `json:"used_storage_gb" gorm:"not null"`
    AvailableStorageGB int       `json:"available_storage_gb" gorm:"not null"`

    RecordedAt         time.Time `json:"recorded_at" gorm:"not null;index"`
    CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
}
```

**Indexes:**
- `device_id` - for querying metrics by device
- `recorded_at` - for time-range queries and cleanup

**Retention Policy:**
- Configurable via `ResourceMonitoringConfig.RetentionPeriod` (default: 24 hours)
- Old metrics are automatically cleaned up during monitoring cycles
- Prevents unbounded database growth

**When Applied:** Automatically on server restart after merging resource monitoring changes

## Complete Schema Reference

### Core Tables

#### users
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | User ID |
| username | string | NOT NULL, UNIQUE | Login username |
| password_hash | string | NOT NULL | Bcrypt hashed password |
| email | string | | Optional email |
| is_admin | boolean | NOT NULL | Admin privileges |
| created_at | timestamp | | Account creation time |
| updated_at | timestamp | | Last update time |

#### devices
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Device ID |
| name | string | NOT NULL | Device name |
| type | string | NOT NULL | router, server, nas, switch |
| ip_address | string | NOT NULL | IP address |
| mac_address | string | | MAC address (optional) |
| status | string | NOT NULL | online, offline, error, unknown |
| username | string | NOT NULL | SSH username |
| auth_type | string | NOT NULL | auto, password, ssh_key, tailscale |
| metadata | json | | Device-specific configuration |
| cpu_usage_percent | float64 | NULLABLE | Current CPU usage % |
| cpu_cores | int | NULLABLE | Total CPU cores |
| total_ram_mb | int | NULLABLE | Total RAM in MB |
| used_ram_mb | int | NULLABLE | Used RAM in MB |
| available_ram_mb | int | NULLABLE | Available RAM in MB |
| total_storage_gb | int | NULLABLE | Total storage in GB |
| used_storage_gb | int | NULLABLE | Used storage in GB |
| available_storage_gb | int | NULLABLE | Available storage in GB |
| resources_updated_at | timestamp | NULLABLE | Last metrics update time |
| last_seen | timestamp | | Last successful health check |
| created_at | timestamp | | Device added time |
| updated_at | timestamp | | Last update time |

#### device_metrics
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Metrics record ID |
| device_id | uuid | NOT NULL, INDEX, FK | Device reference |
| cpu_usage_percent | float64 | NOT NULL | CPU usage % |
| cpu_cores | int | NOT NULL | Total CPU cores |
| total_ram_mb | int | NOT NULL | Total RAM in MB |
| used_ram_mb | int | NOT NULL | Used RAM in MB |
| available_ram_mb | int | NOT NULL | Available RAM in MB |
| total_storage_gb | int | NOT NULL | Total storage in GB |
| used_storage_gb | int | NOT NULL | Used storage in GB |
| available_storage_gb | int | NOT NULL | Available storage in GB |
| recorded_at | timestamp | NOT NULL, INDEX | When metrics were collected |
| created_at | timestamp | | Record creation time |

#### applications
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Application ID |
| name | string | NOT NULL | Application name |
| slug | string | NOT NULL, UNIQUE | URL-friendly identifier |
| category | string | NOT NULL | Application category |
| description | text | NOT NULL | Full description |
| icon_url | string | | Icon image URL |
| docker_image | string | NOT NULL | Docker image name |
| required_ram | int64 | NOT NULL | Minimum RAM in bytes |
| required_storage | int64 | NOT NULL | Minimum storage in bytes |
| config_template | text | NOT NULL | Docker Compose template |
| setup_steps | text | | Post-deployment instructions |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

#### deployments
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Deployment ID |
| recipe_slug | string | NOT NULL | Marketplace recipe identifier |
| recipe_name | string | NOT NULL | Cached recipe name |
| application_id | uuid | NULLABLE, FK | Legacy application reference |
| device_id | uuid | NOT NULL, FK | Target device |
| status | string | NOT NULL | Deployment status |
| config | json | | User configuration |
| domain | string | | Custom domain |
| internal_port | int | NOT NULL | Container port |
| external_port | int | | Host-exposed port |
| container_id | string | | Docker container ID |
| compose_project | string | | Docker Compose project name |
| generated_compose | text | | Generated compose file |
| deployment_logs | text | | Deployment process logs |
| ssh_commands | text | | Commands executed |
| rollback_log | text | | Rollback operation logs |
| error_details | text | | Error information |
| deployed_at | timestamp | | Successful deployment time |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

#### credentials
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Credential ID |
| name | string | NOT NULL | User-friendly name |
| type | string | NOT NULL | password, ssh_key |
| username | string | NOT NULL | SSH username |
| password | string | | Encrypted password |
| ssh_key | string | | Encrypted private key |
| network_cidr | string | | Auto-match by network |
| device_type | string | | Auto-match by device type |
| host_pattern | string | | Auto-match by hostname pattern |
| last_used | timestamp | | Last usage time |
| use_count | int | NOT NULL | Usage counter |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

### Infrastructure Tables

#### installed_software
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Record ID |
| device_id | uuid | NOT NULL, FK | Device reference |
| name | string | NOT NULL | Software name (docker, nfs-server, etc.) |
| version | string | NOT NULL | Installed version |
| installed_at | timestamp | NOT NULL | Installation time |
| installed_by | string | NOT NULL | Username or "system" |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

#### software_installations
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Installation job ID |
| device_id | uuid | NOT NULL, FK | Target device |
| software_name | string | NOT NULL | Software to install |
| status | string | NOT NULL | pending, installing, success, failed |
| install_logs | text | | Installation output |
| error_details | text | | Error information |
| created_at | timestamp | | Job creation time |
| completed_at | timestamp | | Job completion time |

#### nfs_exports
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Export ID |
| device_id | uuid | NOT NULL, FK | NFS server device |
| path | string | NOT NULL | Export path |
| client_cidr | string | NOT NULL | Allowed clients |
| options | string | NOT NULL | NFS options |
| active | boolean | NOT NULL | Export is active |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

#### nfs_mounts
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Mount ID |
| device_id | uuid | NOT NULL, FK | NFS client device |
| server_ip | string | NOT NULL | NFS server IP |
| remote_path | string | NOT NULL | Remote export path |
| local_path | string | NOT NULL | Local mount point |
| options | string | NOT NULL | Mount options |
| permanent | boolean | NOT NULL | Add to /etc/fstab |
| active | boolean | NOT NULL | Mount is active |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

#### volumes
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | uuid | PRIMARY KEY | Volume ID |
| device_id | uuid | NOT NULL, FK | Device hosting volume |
| name | string | NOT NULL | Volume name |
| type | string | NOT NULL | local, nfs |
| driver | string | NOT NULL | Docker volume driver |
| driver_opts | json | | Driver options |
| nfs_server_ip | string | | NFS server (if type=nfs) |
| nfs_path | string | | NFS path (if type=nfs) |
| size | int64 | NOT NULL | Size in bytes |
| in_use | boolean | NOT NULL | Volume is mounted |
| created_at | timestamp | | Record creation time |
| updated_at | timestamp | | Last update time |

## Backup and Restore

### SQLite Database

**Location:** `./homelab.db` (configurable via `DB_PATH` environment variable)

**Backup:**
```bash
# Stop the server first
cp homelab.db homelab.db.backup-$(date +%Y%m%d-%H%M%S)
```

**Restore:**
```bash
# Stop the server first
cp homelab.db.backup-20251013-120000 homelab.db
# Restart server - AutoMigrate will add any new columns
```

### Migration to PostgreSQL (Future)

When scaling beyond SQLite:

1. Install and configure PostgreSQL
2. Update `initDB()` in `cmd/server/main.go`:
   ```go
   import "gorm.io/driver/postgres"

   dsn := "host=localhost user=homelab password=... dbname=homelab"
   db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
   ```
3. Use `pgloader` or custom migration script to transfer data
4. GORM models remain unchanged - abstraction layer handles differences

## Performance Considerations

### Indexes

Current indexes:
- All `id` fields (primary keys)
- `device_metrics.device_id` - for device-specific queries
- `device_metrics.recorded_at` - for time-range queries

**Future optimization opportunities:**
- Composite index on `(device_id, recorded_at)` for device history queries
- Index on `devices.status` for filtering online/offline devices
- Index on `deployments.status` for active deployment queries

### Query Optimization

The resource monitoring service uses efficient queries:

```go
// Aggregate resources - single query with conditional aggregation
db.Model(&models.Device{}).
    Select("COUNT(*) as total_devices, ...").
    First(&result)

// Device metrics history - indexed time-range query
db.Where("device_id = ? AND recorded_at > ?", deviceID, since).
    Order("recorded_at ASC").
    Find(&metrics)
```

### Data Retention

Configure retention via `ResourceMonitoringConfig`:
```go
resourceMonitoring := services.NewResourceMonitoringService(db, sshClient, deviceService, credService, &services.ResourceMonitoringConfig{
    PollInterval:    30 * time.Second,   // How often to collect metrics
    RetentionPeriod: 24 * time.Hour,      // How long to keep history
})
```

Metrics older than `RetentionPeriod` are automatically deleted during each poll cycle.

## Troubleshooting

### Schema Mismatch Errors

If you encounter errors like `no such column`:
1. Stop the server
2. Back up the database
3. Restart the server (AutoMigrate will add missing columns)

### Data Type Conflicts

If AutoMigrate fails with type conflicts:
1. Back up the database
2. Use SQLite CLI to inspect schema:
   ```bash
   sqlite3 homelab.db
   .schema devices
   ```
3. Manually alter the table or migrate data to a new database

### Testing Migrations

Use in-memory databases in tests:
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
db.AutoMigrate(&models.Device{}, &models.DeviceMetrics{})
```

This ensures tests always run against the latest schema.
