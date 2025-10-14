# Software and Storage Management

## Overview

Automated infrastructure management for homelab devices:

1. **Automated Docker Installation** - One-click Docker Engine installation
2. **NFS Server Configuration** - Shared storage provider setup
3. **NFS Client Mounting** - Cross-device storage access
4. **Docker NFS Volumes** - Shared persistent storage for containers
5. **Tailscale Integration** - Mesh VPN for cross-VLAN orchestration

### Design Principles

- **Idempotent Operations** - Safe to run multiple times
- **Pre-flight Validation** - Check requirements before changes
- **Automatic Rollback** - Clean up on failure
- **Progress Transparency** - Real-time feedback via WebSocket
- **State Tracking** - Database records installations

---

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────┐
│                   Control Plane                         │
│                                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Software   │  │     NFS      │  │    Volume    │  │
│  │   Service    │  │   Service    │  │   Service    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
│         │                  │                  │          │
│         └──────────────────┴──────────────────┘          │
│                           │                               │
│                    SSH + Commands                        │
└───────────────────────────┼───────────────────────────────┘
                            │
            ┌───────────────┴──────────────┐
            │                              │
            ▼                              ▼
    ┌──────────────┐              ┌──────────────┐
    │  Server A    │              │  Server B    │
    │  (NFS Server)│              │ (NFS Client) │
    │              │    NFS       │              │
    │ /srv/nfs/    ├──────────────┤ /mnt/nfs/    │
    │  shared/     │   Port 2049  │  shared/     │
    │              │              │              │
    │ Docker       │              │ Docker       │
    │ + NFS-server │              │ + NFS-common │
    └──────────────┘              └──────────────┘
```

### Backend Services

#### SoftwareService

Manages system software installation and state tracking.

**Responsibilities:**
- Check if software is installed
- Install software with validation
- Uninstall software safely
- Track installation state in database

**Interface:**
```go
type SoftwareService interface {
    IsInstalled(host, softwareName string) (bool, string, error)
    Install(deviceID uuid.UUID, softwareName string) error
    Uninstall(deviceID uuid.UUID, softwareName string) error
    ListInstalled(deviceID uuid.UUID) ([]InstalledSoftware, error)
}
```

#### NFSService

Manages NFS server and client configuration.

**Responsibilities:**
- Configure device as NFS server
- Create and manage NFS exports
- Mount NFS shares on client devices
- Manage mount points and fstab entries

**Interface:**
```go
type NFSService interface {
    // Server operations
    SetupServer(deviceID uuid.UUID, exportPath string, options NFSOptions) error
    CreateExport(deviceID uuid.UUID, export NFSExportConfig) error
    RemoveExport(deviceID uuid.UUID, exportPath string) error
    ListExports(deviceID uuid.UUID) ([]NFSExport, error)

    // Client operations
    MountShare(deviceID uuid.UUID, mount NFSMountConfig) error
    UnmountShare(deviceID uuid.UUID, mountPath string) error
    ListMounts(deviceID uuid.UUID) ([]NFSMount, error)
}
```

#### VolumeService

Manages Docker volumes (local and NFS-backed).

**Interface:**
```go
type VolumeService interface {
    CreateVolume(deviceID uuid.UUID, config VolumeConfig) error
    ListVolumes(deviceID uuid.UUID) ([]Volume, error)
    RemoveVolume(deviceID uuid.UUID, volumeName string) error
    InspectVolume(deviceID uuid.UUID, volumeName string) (*VolumeDetails, error)
}
```

---

## Data Models

### InstalledSoftware

Tracks software installed on each device.

```go
type SoftwareType string

const (
    SoftwareDocker       SoftwareType = "docker"
    SoftwareDockerCompose SoftwareType = "docker-compose"
    SoftwareNFSServer    SoftwareType = "nfs-server"
    SoftwareNFSClient    SoftwareType = "nfs-client"
)

type InstalledSoftware struct {
    ID          uuid.UUID    `gorm:"type:uuid;primaryKey"`
    DeviceID    uuid.UUID    `gorm:"type:uuid;not null;index"`
    Device      Device       `gorm:"foreignKey:DeviceID"`
    Name        SoftwareType `gorm:"not null"`
    Version     string       `json:"version"`
    InstalledAt time.Time    `json:"installed_at"`
    InstalledBy string       `json:"installed_by"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### NFSExport

Tracks NFS exports configured on server devices.

```go
type NFSExport struct {
    ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID   uuid.UUID `gorm:"type:uuid;not null;index"`
    Device     Device    `gorm:"foreignKey:DeviceID"`
    Path       string    `gorm:"not null"`
    ClientCIDR string    `gorm:"default:*"`
    Options    string    `gorm:"default:rw,sync,no_subtree_check,no_root_squash"`
    Active     bool      `gorm:"default:true"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### NFSMount

Tracks NFS mounts on client devices.

```go
type NFSMount struct {
    ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID   uuid.UUID `gorm:"type:uuid;not null;index"`
    Device     Device    `gorm:"foreignKey:DeviceID"`
    ServerIP   string    `gorm:"not null"`
    RemotePath string    `gorm:"not null"`
    LocalPath  string    `gorm:"not null"`
    Options    string    `gorm:"default:defaults"`
    Permanent  bool      `gorm:"default:true"`
    Active     bool      `gorm:"default:true"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

### Volume

Tracks Docker volumes.

```go
type VolumeType string

const (
    VolumeTypeLocal VolumeType = "local"
    VolumeTypeNFS   VolumeType = "nfs"
)

type Volume struct {
    ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
    DeviceID     uuid.UUID  `gorm:"type:uuid;not null;index"`
    Device       Device     `gorm:"foreignKey:DeviceID"`
    Name         string     `gorm:"not null"`
    Type         VolumeType `gorm:"not null"`
    Driver       string     `gorm:"default:local"`
    DriverOpts   []byte     `gorm:"type:json"`
    NFSServerIP  string     `json:"nfs_server_ip,omitempty"`
    NFSPath      string     `json:"nfs_path,omitempty"`
    Size         int64      `json:"size"`
    InUse        bool       `gorm:"default:false"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

## Docker Installation

### Technical Approach

**Method:** Official Docker installation script from `get.docker.com`

**Advantages:**
- Officially maintained by Docker Inc.
- Handles all dependencies (containerd, CLI, plugins)
- Idempotent (safe to re-run)
- Sets up systemd service automatically
- Includes Docker Compose v2 as plugin

**Verification:**
```bash
docker --version
docker compose version
```

**State Tracking:**
```go
InstalledSoftware{
    DeviceID:    deviceID,
    Name:        "docker",
    Version:     "24.0.7",
    InstalledAt: time.Now(),
    InstalledBy: "admin",
}
```

---

## NFS Server Setup

### Overview

Configure device as NFS server to provide shared storage.

**Prerequisites:**
- Ubuntu 24.04
- Available disk space
- SSH sudo access
- Firewall allows port 2049

**Default Export Configuration:**
```
/srv/nfs/shared *(rw,sync,no_subtree_check,no_root_squash)
```

**Option Breakdown:**
- `*` - Allow any client (can restrict to specific subnet)
- `rw` - Read-write access
- `sync` - Synchronous writes
- `no_subtree_check` - Performance improvement
- `no_root_squash` - Allow root access (needed for Docker)

**Security Note:** For production, restrict to specific subnet:
```
/srv/nfs/shared 192.168.1.0/24(rw,sync,no_subtree_check,no_root_squash)
```

---

## NFS Client Setup

### Overview

Mount NFS shares from server to access shared storage.

**Prerequisites:**
- NFS server configured and accessible
- Network connectivity to server
- SSH sudo access

**Mount Options:**
- `defaults` - Standard mount options (rw,suid,dev,exec,auto,nouser,async)
- `soft` - Return error if server unavailable (vs. `hard` which hangs)
- `timeo=14` - Timeout in deciseconds (1.4 seconds)
- `retrans=2` - Retry attempts before giving up

**fstab Entry Format:**
```
192.168.1.100:/srv/nfs/shared /mnt/nfs/shared nfs defaults 0 0
```

---

## Docker NFS Volumes

### Overview

Docker supports NFS volumes using `local` driver with NFS options. Enables shared storage across multiple containers and hosts.

**Benefits:**
- Shared storage across multiple containers/hosts
- Data persistence beyond container lifecycle
- Centralized backups
- Scalability without data duplication

**Use Cases:**
- Shared media libraries (Jellyfin, Plex)
- Database files (PostgreSQL, MySQL)
- User uploads (Nextcloud, Immich)
- Application configs

### Creating NFS Volumes

**Docker CLI:**
```bash
docker volume create \
  --driver local \
  --opt type=nfs \
  --opt o=addr=192.168.1.100,rw \
  --opt device=:/srv/nfs/shared \
  my-nfs-volume
```

**Docker Compose:**
```yaml
volumes:
  nfs-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw,soft,timeo=30
      device: ":/srv/nfs/shared"
```

### Performance Considerations

**NFS vs Local Storage:**
- Latency: +1-5ms network round-trip
- Throughput: ~80-100 MB/s on gigabit Ethernet
- IOPS: Lower than local SSD

**Best Practices:**
- Use NFS for large sequential files (media, backups)
- Use local storage for high IOPS databases (unless NFS has SSD)
- Tune `rsize` and `wsize` buffer sizes for workload

---

## Tailscale Integration

### Overview

Tailscale mesh VPN enables secure cross-device communication:

- Cross-VLAN orchestration without firewall configuration
- Secure remote access
- Simplified multi-site deployments
- Automatic device discovery via Tailscale DNS

**Prerequisites:**
- Ubuntu 24.04 or compatible Linux
- SSH sudo access
- Tailscale account with auth key
- Network allows UDP port 41641

### Authentication Methods

**Interactive:**
```bash
sudo tailscale up  # Opens browser for auth
```

**Automated (recommended for orchestration):**
```bash
sudo tailscale up --authkey tskey-auth-xxxxx --advertise-tags=tag:homelab
```

**Ephemeral (temporary devices):**
```bash
sudo tailscale up --authkey tskey-auth-xxxxx --ephemeral
```

### Tag Configuration

Tags control ACL policies and device grouping.

**Common Tags:**
- `tag:homelab` - All homelab devices
- `tag:server` - Server devices
- `tag:client` - Client devices
- `tag:nas` - Storage devices

### Data Model

```go
type TailscaleConfig struct {
    ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID      uuid.UUID `gorm:"type:uuid;not null;index"`
    Device        Device    `gorm:"foreignKey:DeviceID"`
    Enabled       bool      `gorm:"default:true"`
    TailscaleIP   string    `json:"tailscale_ip"`   // 100.x.x.x
    Hostname      string    `json:"hostname"`
    Tags          string    `json:"tags"`
    ExitNode      bool      `gorm:"default:false"`
    SubnetRouter  bool      `gorm:"default:false"`
    AdvertiseRoute string   `json:"advertise_route,omitempty"`
    InstalledAt   time.Time `json:"installed_at"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Use Cases

**Cross-VLAN Orchestration:**
```
Control Plane (VLAN 10): 192.168.10.5 / 100.64.0.1
Device A (VLAN 20):      192.168.20.10 / 100.64.0.5
Device B (VLAN 30):      192.168.30.15 / 100.64.0.6

Orchestration uses Tailscale IPs for communication
```

**Multi-Site Deployment:**
```
Site A (Home):   Devices 1-3 (100.64.0.1-3)
Site B (Office): Devices 4-6 (100.64.0.4-6)

Single control plane manages all via Tailscale mesh
```

---

## API Reference

### Software Management

#### Install Software
```
POST /api/v1/devices/:id/software/install

Request:
{
  "software": "docker",  // "docker", "nfs-server", "nfs-client"
  "options": {
    "add_user_to_group": true
  }
}

Response:
{
  "success": true,
  "message": "Docker 24.0.7 installed successfully",
  "installed_software": {
    "id": "uuid",
    "device_id": "uuid",
    "name": "docker",
    "version": "24.0.7",
    "installed_at": "2025-10-09T20:00:00Z"
  }
}
```

#### List Installed Software
```
GET /api/v1/devices/:id/software

Response:
{
  "installed": [
    {
      "id": "uuid",
      "name": "docker",
      "version": "24.0.7",
      "installed_at": "2025-10-09T20:00:00Z"
    }
  ]
}
```

#### Uninstall Software
```
DELETE /api/v1/devices/:id/software/:name
```

### NFS Server Management

#### Setup NFS Server
```
POST /api/v1/devices/:id/nfs/server/setup

Request:
{
  "export_path": "/srv/nfs/shared",
  "client_cidr": "*",
  "options": "rw,sync,no_subtree_check,no_root_squash"
}

Response:
{
  "success": true,
  "message": "NFS server configured successfully",
  "export": {
    "id": "uuid",
    "device_id": "uuid",
    "path": "/srv/nfs/shared",
    "client_cidr": "*",
    "options": "rw,sync,no_subtree_check,no_root_squash",
    "active": true
  }
}
```

#### List NFS Exports
```
GET /api/v1/devices/:id/nfs/exports
```

#### Remove NFS Export
```
DELETE /api/v1/devices/:id/nfs/exports/:id
```

### NFS Client Management

#### Mount NFS Share
```
POST /api/v1/devices/:id/nfs/client/mount

Request:
{
  "server_ip": "192.168.1.100",
  "remote_path": "/srv/nfs/shared",
  "local_path": "/mnt/nfs/shared",
  "options": "defaults",
  "permanent": true  // Add to fstab
}

Response:
{
  "success": true,
  "message": "NFS share mounted successfully",
  "mount": {
    "id": "uuid",
    "device_id": "uuid",
    "server_ip": "192.168.1.100",
    "remote_path": "/srv/nfs/shared",
    "local_path": "/mnt/nfs/shared",
    "options": "defaults",
    "permanent": true,
    "active": true
  }
}
```

#### List NFS Mounts
```
GET /api/v1/devices/:id/nfs/mounts
```

#### Unmount NFS Share
```
DELETE /api/v1/devices/:id/nfs/mounts/:id
Query: remove_from_fstab (boolean)
```

### Docker Volume Management

#### Create Docker Volume
```
POST /api/v1/devices/:id/volumes

Request (Local):
{
  "name": "my-local-volume",
  "type": "local"
}

Request (NFS):
{
  "name": "my-nfs-volume",
  "type": "nfs",
  "nfs_server_ip": "192.168.1.100",
  "nfs_path": "/srv/nfs/shared",
  "driver_opts": {
    "o": "addr=192.168.1.100,rw,soft,timeo=30",
    "device": ":/srv/nfs/shared"
  }
}

Response:
{
  "success": true,
  "message": "Volume created successfully",
  "volume": {
    "id": "uuid",
    "name": "my-nfs-volume",
    "type": "nfs",
    "driver": "local",
    "nfs_server_ip": "192.168.1.100",
    "nfs_path": "/srv/nfs/shared",
    "in_use": false
  }
}
```

#### List Docker Volumes
```
GET /api/v1/devices/:id/volumes
```

#### Remove Docker Volume
```
DELETE /api/v1/devices/:id/volumes/:name
```

### Tailscale Management

#### Install Tailscale
```
POST /api/v1/devices/:id/tailscale/install

Request:
{
  "authkey": "tskey-auth-xxxxx",
  "tags": ["homelab", "server"],
  "ephemeral": false
}

Response:
{
  "success": true,
  "message": "Tailscale installed and authenticated",
  "config": {
    "tailscale_ip": "100.64.0.5",
    "hostname": "server-01",
    "tags": "tag:homelab,tag:server"
  }
}
```

#### Get Tailscale Status
```
GET /api/v1/devices/:id/tailscale/status

Response:
{
  "enabled": true,
  "tailscale_ip": "100.64.0.5",
  "hostname": "server-01",
  "connected": true,
  "exit_node": false
}
```

#### Update Tailscale Config
```
PATCH /api/v1/devices/:id/tailscale/config

Request:
{
  "tags": ["homelab", "nas"],
  "exit_node": true
}
```

---

## Security Considerations

### Authentication & Authorization

**SSH Permissions:**
- Passwordless sudo required for installation
- Recommend dedicated homelab admin user
- Store credentials in OS keychain

**Limited Sudo Scope:**
```bash
homelab ALL=(ALL) NOPASSWD: /usr/bin/apt-get, /usr/bin/systemctl, /usr/bin/docker, /usr/sbin/exportfs, /usr/bin/mount, /usr/bin/umount
```

### Network Security

**NFS Security:**
- Default `*` allows all clients (convenient but insecure)
- Recommended: Restrict to subnet (`192.168.1.0/24`)
- Best: Whitelist specific IPs
- Never expose NFS to public internet

**Firewall Configuration:**
```bash
# UFW example: Allow NFS from local subnet only
sudo ufw allow from 192.168.1.0/24 to any port nfs
sudo ufw deny nfs
```

### Data Security

**NFS Export Options:**
- `no_root_squash` - Allows root access (needed for Docker, security risk)
- Consider `root_squash` for non-Docker shares
- Use `ro` (read-only) when possible

**Encryption:**
- NFS traffic is NOT encrypted by default
- For sensitive data use: Tailscale VPN, NFS over SSH tunnel, or eCryptfs/LUKS

**Tailscale Security:**
- Store auth keys encrypted in OS keychain
- Use ephemeral keys for temporary devices
- Define ACL policies in Tailscale admin
- Use tags to control inter-device access

### Docker Security

**Volume Permissions:**
- Docker containers run as root by default
- `no_root_squash` allows containers to write as root
- Consider rootless Docker for improved security

**Container Isolation:**
- Containers sharing NFS volumes can access each other's data
- Use separate NFS exports per application if needed

---

**Version:** 1.0
**Last Updated:** October 2025
