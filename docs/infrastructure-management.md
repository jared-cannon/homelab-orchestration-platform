# Software and Storage Management
**Version:** 1.0
**Last Updated:** October 2025
**Target OS:** Ubuntu 24.04 LTS

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Docker Installation](#docker-installation)
4. [NFS Server Setup](#nfs-server-setup)
5. [NFS Client Setup](#nfs-client-setup)
6. [Docker NFS Volumes](#docker-nfs-volumes)
7. [API Reference](#api-reference)
8. [Frontend UX](#frontend-ux)
9. [Security Considerations](#security-considerations)
10. [Testing Strategy](#testing-strategy)
11. [Troubleshooting](#troubleshooting)

---

## Overview

This document outlines the software and storage management capabilities that enable:

1. **Automated Docker Installation** - One-click Docker Engine installation on Ubuntu 24.04
2. **NFS Server Configuration** - Turn any server into a shared storage provider
3. **NFS Client Mounting** - Mount shared storage across multiple servers
4. **Docker NFS Volumes** - Containers can use NFS for persistent, shared storage

### Design Principles

- **Idempotent Operations** - Safe to run multiple times
- **Pre-flight Validation** - Check requirements before attempting changes
- **Automatic Rollback** - Clean up on failure
- **Progress Transparency** - Real-time feedback via WebSocket
- **State Tracking** - Database records what's installed where
- **User-Friendly Errors** - Clear explanations with actionable fixes

### Use Cases

**Scenario 1: Fresh Server Setup**
```
User adds new Ubuntu 24.04 server → Check shows "Docker not installed"
→ Click "Install Docker" → Progress bar → "Docker installed successfully"
```

**Scenario 2: Shared Storage for Applications**
```
Server A (NAS): 2TB disk → Configure as NFS server → Export /srv/nfs/shared
Server B, C (App hosts): Mount NFS storage → Docker volumes use NFS
Result: Apps on B and C share persistent storage from A
```

**Scenario 3: Multi-Container Applications**
```
Deploy database on Server B → Uses NFS volume from Server A
Deploy app server on Server C → Uses same NFS volume
Result: Database and app can share files, survive container restarts
```

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

#### 1. **SoftwareService** (`backend/internal/services/software.go`)

Manages system software installation and state tracking.

**Responsibilities:**
- Check if software is installed
- Install software with proper validation
- Uninstall software safely
- Track installation state in database
- Provide rollback on failure

**Key Methods:**
```go
type SoftwareService interface {
    // Check if software is installed
    IsInstalled(host, softwareName string) (bool, string, error)

    // Install software (Docker, NFS server, NFS client)
    Install(deviceID uuid.UUID, softwareName string) error

    // Uninstall software
    Uninstall(deviceID uuid.UUID, softwareName string) error

    // List all installed software on device
    ListInstalled(deviceID uuid.UUID) ([]InstalledSoftware, error)
}
```

#### 2. **NFSService** (`backend/internal/services/nfs.go`)

Manages NFS server and client configuration.

**Responsibilities:**
- Configure device as NFS server
- Create and manage NFS exports
- Mount NFS shares on client devices
- Manage mount points and fstab entries
- Validate network connectivity between server/client

**Key Methods:**
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

#### 3. **VolumeService** (`backend/internal/services/volume.go`)

Manages Docker volumes (local and NFS-backed).

**Responsibilities:**
- Create Docker volumes (local or NFS)
- List volumes on device
- Remove volumes
- Validate volume configurations

**Key Methods:**
```go
type VolumeService interface {
    CreateVolume(deviceID uuid.UUID, config VolumeConfig) error
    ListVolumes(deviceID uuid.UUID) ([]Volume, error)
    RemoveVolume(deviceID uuid.UUID, volumeName string) error
    InspectVolume(deviceID uuid.UUID, volumeName string) (*VolumeDetails, error)
}
```

### Database Models

#### 1. **InstalledSoftware**

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
    InstalledBy string       `json:"installed_by"` // username or "system"
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### 2. **NFSExport**

Tracks NFS exports configured on server devices.

```go
type NFSExport struct {
    ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID   uuid.UUID `gorm:"type:uuid;not null;index"` // The NFS server
    Device     Device    `gorm:"foreignKey:DeviceID"`
    Path       string    `gorm:"not null"` // e.g., "/srv/nfs/shared"
    ClientCIDR string    `gorm:"default:*"` // e.g., "*", "192.168.1.0/24"
    Options    string    `gorm:"default:rw,sync,no_subtree_check,no_root_squash"`
    Active     bool      `gorm:"default:true"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

#### 3. **NFSMount**

Tracks NFS mounts on client devices.

```go
type NFSMount struct {
    ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID   uuid.UUID `gorm:"type:uuid;not null;index"` // The NFS client
    Device     Device    `gorm:"foreignKey:DeviceID"`
    ServerIP   string    `gorm:"not null"` // NFS server IP
    RemotePath string    `gorm:"not null"` // e.g., "/srv/nfs/shared"
    LocalPath  string    `gorm:"not null"` // e.g., "/mnt/nfs/shared"
    Options    string    `gorm:"default:defaults"`
    Permanent  bool      `gorm:"default:true"` // Add to /etc/fstab
    Active     bool      `gorm:"default:true"`
    CreatedAt  time.Time
    UpdatedAt  time.Time
}
```

#### 4. **Volume**

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
    DriverOpts   []byte     `gorm:"type:json"` // JSON map of driver options
    NFSServerIP  string     `json:"nfs_server_ip,omitempty"`
    NFSPath      string     `json:"nfs_path,omitempty"`
    Size         int64      `json:"size"` // bytes, if known
    InUse        bool       `gorm:"default:false"`
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

---

## Docker Installation

### Overview

Automated Docker Engine installation for Ubuntu 24.04 using the official convenience script.

### Technical Approach

**Method:** Official Docker installation script from `get.docker.com`

**Why This Approach:**
- ✅ Officially maintained by Docker Inc.
- ✅ Handles all dependencies (containerd, CLI, plugins)
- ✅ Idempotent (safe to re-run)
- ✅ Sets up systemd service automatically
- ✅ Works on Ubuntu 24.04 (Noble Numbat)

### Installation Steps

The service executes these commands via SSH:

```bash
# 1. Pre-flight checks
## Check if Docker already installed
docker --version  # If succeeds, skip installation

## Check sudo access
sudo -n true  # Must succeed

## Check internet connectivity
curl -I https://get.docker.com  # Must succeed

# 2. Download and run installation script
curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
sudo sh /tmp/get-docker.sh

# 3. Post-installation
## Start and enable Docker service
sudo systemctl start docker
sudo systemctl enable docker

## Verify installation
sudo docker --version
sudo docker ps  # Test daemon is running

## Add current user to docker group (optional)
sudo usermod -aG docker $USER

# 4. Cleanup
rm /tmp/get-docker.sh
```

### Docker Compose Plugin

The official Docker installation script now includes Docker Compose v2 as a plugin.

**Verification:**
```bash
docker compose version  # Should show "Docker Compose version v2.x.x"
```

### State Tracking

After successful installation:
```go
InstalledSoftware{
    DeviceID:    deviceID,
    Name:        "docker",
    Version:     "24.0.7",  // From docker --version
    InstalledAt: time.Now(),
    InstalledBy: "admin",
}
```

### Rollback Strategy

If installation fails:
1. Stop Docker service: `sudo systemctl stop docker`
2. Remove packages: `sudo apt-get remove -y docker-ce docker-ce-cli containerd.io`
3. Clean up files: `sudo rm -rf /var/lib/docker`
4. Remove from database

### Error Handling

| Error | Detection | User Message | Suggested Fix |
|-------|-----------|--------------|---------------|
| No sudo access | `sudo -n true` fails | "Need admin privileges" | "Ensure SSH user has passwordless sudo" |
| No internet | `curl -I` fails | "Can't reach Docker servers" | "Check device internet connection" |
| Script fails | Exit code != 0 | "Docker installation failed" | "Check device logs at /var/log/docker-install.log" |
| Already installed | `docker --version` succeeds | "Docker already installed" | "Use existing Docker installation" |

---

## NFS Server Setup

### Overview

Configure a device as an NFS (Network File System) server to provide shared storage to other devices.

### Prerequisites

- Device must be Ubuntu 24.04
- Device must have available disk space (recommend dedicated partition/disk)
- SSH user must have sudo access
- Firewall must allow NFS traffic (port 2049)

### Installation Steps

```bash
# 1. Install NFS server package
sudo apt-get update
sudo apt-get install -y nfs-kernel-server

# 2. Create export directory
sudo mkdir -p /srv/nfs/shared
sudo chown nobody:nogroup /srv/nfs/shared
sudo chmod 755 /srv/nfs/shared

# 3. Configure exports
## Add to /etc/exports
echo "/srv/nfs/shared *(rw,sync,no_subtree_check,no_root_squash)" | sudo tee -a /etc/exports

# 4. Apply configuration
sudo exportfs -ra  # Re-export all shares
sudo systemctl restart nfs-kernel-server

# 5. Enable on boot
sudo systemctl enable nfs-kernel-server

# 6. Verify
showmount -e localhost  # Should show /srv/nfs/shared
```

### Export Configuration

**Default Export Options:**
```
/srv/nfs/shared *(rw,sync,no_subtree_check,no_root_squash)
```

**Option Breakdown:**
- `*` - Allow any client to connect (can restrict to specific IPs/subnets)
- `rw` - Read-write access
- `sync` - Synchronous writes (safer, slightly slower)
- `no_subtree_check` - Disable subtree checking (performance improvement)
- `no_root_squash` - Allow root user on client to write as root (needed for Docker)

**Security Note:** Using `*` allows any device on the network to mount. For production:
```
/srv/nfs/shared 192.168.1.0/24(rw,sync,no_subtree_check,no_root_squash)
```

### Custom Export Paths

Users can create multiple exports:
```go
NFSExport{
    Path:       "/srv/nfs/media",      // For media files
    ClientCIDR: "192.168.1.0/24",
    Options:    "rw,sync,no_subtree_check",
}

NFSExport{
    Path:       "/srv/nfs/backups",    // For backups
    ClientCIDR: "192.168.1.0/24",
    Options:    "rw,sync,no_subtree_check",
}
```

### Firewall Configuration

If UFW (Uncomplicated Firewall) is active:
```bash
sudo ufw allow from 192.168.1.0/24 to any port nfs
sudo ufw reload
```

For `iptables`:
```bash
sudo iptables -A INPUT -p tcp --dport 2049 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p udp --dport 2049 -s 192.168.1.0/24 -j ACCEPT
```

### State Tracking

```go
InstalledSoftware{
    DeviceID: deviceID,
    Name:     "nfs-server",
    Version:  "1.3.4",  // From apt-cache policy nfs-kernel-server
}

NFSExport{
    DeviceID:   deviceID,
    Path:       "/srv/nfs/shared",
    ClientCIDR: "*",
    Options:    "rw,sync,no_subtree_check,no_root_squash",
    Active:     true,
}
```

---

## NFS Client Setup

### Overview

Mount NFS shares from an NFS server to access shared storage.

### Prerequisites

- NFS server must be configured and accessible
- Network connectivity to NFS server (test with `ping`)
- SSH user must have sudo access
- Firewall allows NFS traffic (port 2049)

### Installation Steps

```bash
# 1. Install NFS client package
sudo apt-get update
sudo apt-get install -y nfs-common

# 2. Create mount point
sudo mkdir -p /mnt/nfs/shared

# 3. Test mount (temporary)
sudo mount -t nfs 192.168.1.100:/srv/nfs/shared /mnt/nfs/shared

# 4. Verify mount
df -h | grep nfs  # Should show mounted NFS share
ls -la /mnt/nfs/shared  # Should list files

# 5. Make permanent (add to /etc/fstab)
echo "192.168.1.100:/srv/nfs/shared /mnt/nfs/shared nfs defaults 0 0" | sudo tee -a /etc/fstab

# 6. Test fstab entry
sudo umount /mnt/nfs/shared
sudo mount -a  # Mount all from fstab
```

### Mount Options

**Default Options:**
```
defaults  # Equivalent to: rw,suid,dev,exec,auto,nouser,async
```

**Common Options:**
- `rw` - Read-write
- `ro` - Read-only
- `soft` - Return error if server unavailable (vs. `hard` which hangs)
- `timeo=14` - Timeout in deciseconds (1.4 seconds)
- `retrans=2` - Number of retries before giving up

**Example with custom options:**
```
192.168.1.100:/srv/nfs/shared /mnt/nfs/shared nfs rw,soft,timeo=14,retrans=2 0 0
```

### Auto-Mount on Boot

The fstab entry ensures the mount happens on boot. To test:
```bash
# Reboot device
sudo reboot

# After reboot, verify
df -h | grep nfs
```

### Troubleshooting Client Mounts

**Test connectivity to NFS server:**
```bash
# Ping server
ping -c 3 192.168.1.100

# Check if NFS port is open
nc -zv 192.168.1.100 2049

# List available exports from server
showmount -e 192.168.1.100
```

**Common mount errors:**

| Error | Cause | Fix |
|-------|-------|-----|
| `mount.nfs: Connection refused` | NFS server not running | Start server: `sudo systemctl start nfs-kernel-server` |
| `mount.nfs: No such file or directory` | Export path doesn't exist | Check server exports: `showmount -e <server-ip>` |
| `mount.nfs: access denied` | Client IP not allowed | Update server `/etc/exports` to include client |
| `mount.nfs: Connection timed out` | Firewall blocking | Open port 2049 on server |

### State Tracking

```go
InstalledSoftware{
    DeviceID: deviceID,
    Name:     "nfs-client",
    Version:  "1.3.4",
}

NFSMount{
    DeviceID:   deviceID,
    ServerIP:   "192.168.1.100",
    RemotePath: "/srv/nfs/shared",
    LocalPath:  "/mnt/nfs/shared",
    Options:    "defaults",
    Permanent:  true,  // Added to fstab
    Active:     true,
}
```

---

## Docker NFS Volumes

### Overview

Docker supports NFS volumes natively using the `local` driver with NFS options. This allows containers to store data on shared NFS storage.

### Why Use NFS Volumes?

**Benefits:**
- ✅ **Shared Storage** - Multiple containers across different hosts can access the same data
- ✅ **Data Persistence** - Data survives container restarts and host failures
- ✅ **Centralized Backups** - Backup NFS server instead of individual containers
- ✅ **Scalability** - Add more app servers without duplicating data
- ✅ **No Vendor Lock-in** - Standard protocol, works anywhere

**Use Cases:**
- Shared media libraries (Jellyfin, Plex)
- Database files (PostgreSQL, MySQL)
- User uploads (Nextcloud, Immich)
- Application configs
- Log aggregation

### Creating NFS Volumes

#### Method 1: Docker CLI

```bash
docker volume create \
  --driver local \
  --opt type=nfs \
  --opt o=addr=192.168.1.100,rw \
  --opt device=:/srv/nfs/shared \
  my-nfs-volume
```

**Option Breakdown:**
- `--driver local` - Use local driver (supports NFS via options)
- `--opt type=nfs` - Specify NFS filesystem type
- `--opt o=addr=192.168.1.100,rw` - NFS server address and mount options
- `--opt device=:/srv/nfs/shared` - Remote path to mount

#### Method 2: Docker Compose

```yaml
version: '3.8'

services:
  app:
    image: nginx:latest
    volumes:
      - nfs-data:/usr/share/nginx/html

volumes:
  nfs-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw,soft,timeo=30
      device: ":/srv/nfs/shared"
```

### Advanced NFS Volume Options

**Performance tuning:**
```yaml
volumes:
  nfs-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw,soft,timeo=30,rsize=8192,wsize=8192,tcp
      device: ":/srv/nfs/shared"
```

**Options explained:**
- `soft` - Return error if server unavailable (vs. `hard` which hangs indefinitely)
- `timeo=30` - Timeout in deciseconds (3 seconds)
- `rsize=8192` - Read buffer size (8KB)
- `wsize=8192` - Write buffer size (8KB)
- `tcp` - Use TCP instead of UDP (more reliable over unreliable networks)

### Volume Lifecycle

**Create Volume:**
```bash
docker volume create --name my-nfs-vol \
  --driver local \
  --opt type=nfs \
  --opt o=addr=192.168.1.100,rw \
  --opt device=:/srv/nfs/shared
```

**List Volumes:**
```bash
docker volume ls
# DRIVER    VOLUME NAME
# local     my-nfs-vol
```

**Inspect Volume:**
```bash
docker volume inspect my-nfs-vol
```

Output:
```json
[
    {
        "CreatedAt": "2025-10-09T20:00:00Z",
        "Driver": "local",
        "Labels": {},
        "Mountpoint": "/var/lib/docker/volumes/my-nfs-vol/_data",
        "Name": "my-nfs-vol",
        "Options": {
            "device": ":/srv/nfs/shared",
            "o": "addr=192.168.1.100,rw",
            "type": "nfs"
        },
        "Scope": "local"
    }
]
```

**Remove Volume:**
```bash
docker volume rm my-nfs-vol
# Note: Volume must not be in use by any containers
```

### Using NFS Volumes in Containers

**Simple example:**
```bash
docker run -d \
  --name web \
  -v my-nfs-vol:/usr/share/nginx/html \
  nginx:latest
```

**Docker Compose example:**
```yaml
version: '3.8'

services:
  database:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - db-data:/var/lib/postgresql/data  # Database files on NFS

  app:
    image: myapp:latest
    volumes:
      - app-uploads:/app/uploads  # User uploads on NFS
    depends_on:
      - database

volumes:
  db-data:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw
      device: ":/srv/nfs/shared/database"

  app-uploads:
    driver: local
    driver_opts:
      type: nfs
      o: addr=192.168.1.100,rw
      device: ":/srv/nfs/shared/uploads"
```

### Multi-Host Scenario

**Setup:**
- **Server A** (192.168.1.100): NFS server with 2TB disk
- **Server B** (192.168.1.101): Docker host #1
- **Server C** (192.168.1.102): Docker host #2

**On Server B:**
```bash
docker volume create --name shared-media \
  --driver local \
  --opt type=nfs \
  --opt o=addr=192.168.1.100,rw \
  --opt device=:/srv/nfs/shared/media

docker run -d --name jellyfin \
  -v shared-media:/media \
  jellyfin/jellyfin:latest
```

**On Server C:**
```bash
docker volume create --name shared-media \
  --driver local \
  --opt type=nfs \
  --opt o=addr=192.168.1.100,rw \
  --opt device=:/srv/nfs/shared/media

docker run -d --name radarr \
  -v shared-media:/media \
  linuxserver/radarr:latest
```

**Result:** Both Jellyfin (Server B) and Radarr (Server C) access the same media library stored on Server A.

### State Tracking

```go
Volume{
    DeviceID:    deviceID,
    Name:        "my-nfs-volume",
    Type:        VolumeTypeNFS,
    Driver:      "local",
    DriverOpts:  `{"type":"nfs","o":"addr=192.168.1.100,rw","device":":/srv/nfs/shared"}`,
    NFSServerIP: "192.168.1.100",
    NFSPath:     "/srv/nfs/shared",
    InUse:       false,  // Updated when container uses it
}
```

### Performance Considerations

**NFS vs Local Storage:**
- **Latency**: NFS adds network round-trip (~1-5ms on local network)
- **Throughput**: Gigabit Ethernet = ~125 MB/s theoretical (80-100 MB/s real-world)
- **IOPS**: Lower than local SSD, but sufficient for most homelab workloads

**Best Practices:**
- Use NFS for large sequential files (media, backups)
- Use local storage for databases requiring high IOPS (unless NFS server has fast storage)
- Consider SSD-backed NFS for better performance
- Use `rsize` and `wsize` to tune buffer sizes for your workload

---

## Tailscale Integration

### Overview

Tailscale provides mesh VPN connectivity for secure cross-device communication. Integration enables:

- Cross-VLAN orchestration without firewall configuration
- Secure remote access to devices
- Simplified network topology for multi-site deployments
- Automatic device discovery via Tailscale DNS

### Prerequisites

- Device must be Ubuntu 24.04 or compatible Linux distribution
- SSH user must have sudo access
- Tailscale account with authentication key
- Network allows UDP traffic (port 41641)

### Installation Steps

```bash
# 1. Add Tailscale repository
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/noble.noarmor.gpg | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg >/dev/null
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/noble.tailscale-keyring.list | sudo tee /etc/apt/sources.list.d/tailscale.list

# 2. Install Tailscale package
sudo apt-get update
sudo apt-get install -y tailscale

# 3. Verify installation
tailscale version
```

### Device Authentication

**Interactive Authentication:**
```bash
sudo tailscale up
# Opens browser for authentication
```

**Automated Authentication (recommended for orchestration):**
```bash
# Using pre-authorized key from Tailscale admin console
sudo tailscale up --authkey tskey-auth-xxxxx --advertise-tags=tag:homelab
```

**Ephemeral Nodes (for temporary devices):**
```bash
sudo tailscale up --authkey tskey-auth-xxxxx --ephemeral
```

### Tag Configuration

Tags control ACL policies and device grouping in Tailscale.

**Common Tags:**
- `tag:homelab` - All homelab devices
- `tag:server` - Server devices
- `tag:client` - Client devices
- `tag:nas` - Storage devices

**Apply Tags During Setup:**
```bash
sudo tailscale up --authkey tskey-auth-xxxxx --advertise-tags=tag:homelab,tag:server
```

**Update Tags on Running Device:**
```bash
sudo tailscale set --advertise-tags=tag:homelab,tag:nas
```

### State Tracking

```go
type TailscaleConfig struct {
    ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
    DeviceID      uuid.UUID `gorm:"type:uuid;not null;index"`
    Device        Device    `gorm:"foreignKey:DeviceID"`
    Enabled       bool      `gorm:"default:true"`
    TailscaleIP   string    `json:"tailscale_ip"`   // 100.x.x.x address
    Hostname      string    `json:"hostname"`
    Tags          string    `json:"tags"`           // Comma-separated
    ExitNode      bool      `gorm:"default:false"`
    SubnetRouter  bool      `gorm:"default:false"`
    AdvertiseRoute string   `json:"advertise_route,omitempty"` // e.g., "192.168.1.0/24"
    InstalledAt   time.Time `json:"installed_at"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### API Endpoints

**Install Tailscale:**
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

**Get Status:**
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

**Update Configuration:**
```
PATCH /api/v1/devices/:id/tailscale/config

Request:
{
  "tags": ["homelab", "nas"],
  "exit_node": true
}
```

### Frontend UX

**Device Setup Wizard Integration:**

During device onboarding, offer optional Tailscale setup:

```
┌───────────────────────────────────────────────────┐
│ Enable Tailscale (Optional)                       │
├───────────────────────────────────────────────────┤
│                                                    │
│ Connect this device to your Tailscale network     │
│ for secure remote access and cross-VLAN support.  │
│                                                    │
│ Authentication Key:                                │
│ [tskey-auth-___________________________]          │
│                                                    │
│ Tags (optional):                                   │
│ [homelab, server_____________________]            │
│                                                    │
│ [Skip]  [Setup Tailscale]                         │
└───────────────────────────────────────────────────┘
```

**Device Detail Page:**

Show Tailscale status in device overview:

```
┌─────────────────────────────────────────────────┐
│ Network Information                              │
├─────────────────────────────────────────────────┤
│ Local IP:      192.168.1.100                    │
│ Tailscale IP:  100.64.0.5 [Connected]           │
│ Hostname:      server-01.tail-xxxxx.ts.net      │
│ Tags:          homelab, server                  │
└─────────────────────────────────────────────────┘
```

### Use Cases

**Cross-VLAN Orchestration:**

Manage devices across multiple VLANs without complex firewall rules:

```
Control Plane (VLAN 10): 192.168.10.5 / 100.64.0.1
Device A (VLAN 20):      192.168.20.10 / 100.64.0.5
Device B (VLAN 30):      192.168.30.15 / 100.64.0.6

Orchestration uses Tailscale IPs for communication
```

**Remote Access:**

Access homelab from anywhere without port forwarding:

```
ssh user@100.64.0.5
# OR
ssh server-01.tail-xxxxx.ts.net
```

**Multi-Site Deployment:**

Orchestrate devices across physical locations:

```
Site A (Home):   Devices 1-3 (100.64.0.1-3)
Site B (Office): Devices 4-6 (100.64.0.4-6)

Single control plane manages all devices via Tailscale mesh
```

### Security Considerations

**Authentication Keys:**
- Store auth keys encrypted in OS keychain
- Use ephemeral keys for temporary devices
- Rotate keys periodically

**ACL Policies:**
- Define policies in Tailscale admin console
- Use tags to control access between devices
- Example: Only `tag:homelab` devices can access each other

**Network Isolation:**
- Tailscale traffic is encrypted end-to-end
- Uses WireGuard protocol
- No need for additional VPN tunnels

### Troubleshooting

**Problem:** "tailscale: command not found"
- **Cause:** Package not installed or not in PATH
- **Fix:** Verify installation: `which tailscale`

**Problem:** "Authentication failed"
- **Cause:** Invalid or expired auth key
- **Fix:** Generate new key from Tailscale admin console

**Problem:** "Unable to connect to coordination server"
- **Cause:** Firewall blocking UDP port 41641
- **Fix:** Allow UDP traffic: `sudo ufw allow 41641/udp`

**Problem:** "Device not appearing in Tailscale admin"
- **Cause:** Authentication not completed
- **Fix:** Check status: `sudo tailscale status`

---

## API Reference

### Software Management

#### Install Software

**Endpoint:** `POST /api/v1/devices/:id/software/install`

**Request:**
```json
{
  "software": "docker",  // "docker", "nfs-server", "nfs-client"
  "options": {
    "add_user_to_group": true  // For docker: add SSH user to docker group
  }
}
```

**Response (Success):**
```json
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

**Response (Already Installed):**
```json
{
  "success": true,
  "message": "Docker 24.0.7 is already installed",
  "installed_software": {
    "id": "uuid",
    "name": "docker",
    "version": "24.0.7"
  }
}
```

**Response (Error):**
```json
{
  "success": false,
  "error": "Installation failed: insufficient disk space",
  "details": "Need 10GB available, only 2GB free"
}
```

#### List Installed Software

**Endpoint:** `GET /api/v1/devices/:id/software`

**Response:**
```json
{
  "installed": [
    {
      "id": "uuid",
      "name": "docker",
      "version": "24.0.7",
      "installed_at": "2025-10-09T20:00:00Z"
    },
    {
      "id": "uuid",
      "name": "nfs-client",
      "version": "1.3.4",
      "installed_at": "2025-10-09T21:00:00Z"
    }
  ]
}
```

#### Uninstall Software

**Endpoint:** `DELETE /api/v1/devices/:id/software/:name`

**Response:**
```json
{
  "success": true,
  "message": "Docker uninstalled successfully"
}
```

### NFS Server Management

#### Setup NFS Server

**Endpoint:** `POST /api/v1/devices/:id/nfs/server/setup`

**Request:**
```json
{
  "export_path": "/srv/nfs/shared",
  "client_cidr": "*",  // Or "192.168.1.0/24"
  "options": "rw,sync,no_subtree_check,no_root_squash"
}
```

**Response:**
```json
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

**Endpoint:** `GET /api/v1/devices/:id/nfs/exports`

**Response:**
```json
{
  "exports": [
    {
      "id": "uuid",
      "path": "/srv/nfs/shared",
      "client_cidr": "*",
      "options": "rw,sync,no_subtree_check,no_root_squash",
      "active": true,
      "created_at": "2025-10-09T20:00:00Z"
    }
  ]
}
```

#### Remove NFS Export

**Endpoint:** `DELETE /api/v1/devices/:id/nfs/exports/:id`

**Response:**
```json
{
  "success": true,
  "message": "NFS export removed successfully"
}
```

### NFS Client Management

#### Mount NFS Share

**Endpoint:** `POST /api/v1/devices/:id/nfs/client/mount`

**Request:**
```json
{
  "server_ip": "192.168.1.100",
  "remote_path": "/srv/nfs/shared",
  "local_path": "/mnt/nfs/shared",
  "options": "defaults",
  "permanent": true  // Add to fstab
}
```

**Response:**
```json
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

**Endpoint:** `GET /api/v1/devices/:id/nfs/mounts`

**Response:**
```json
{
  "mounts": [
    {
      "id": "uuid",
      "server_ip": "192.168.1.100",
      "remote_path": "/srv/nfs/shared",
      "local_path": "/mnt/nfs/shared",
      "options": "defaults",
      "permanent": true,
      "active": true,
      "created_at": "2025-10-09T20:00:00Z"
    }
  ]
}
```

#### Unmount NFS Share

**Endpoint:** `DELETE /api/v1/devices/:id/nfs/mounts/:id`

**Query Parameters:**
- `remove_from_fstab` (boolean): Remove from /etc/fstab as well

**Response:**
```json
{
  "success": true,
  "message": "NFS share unmounted successfully"
}
```

### Docker Volume Management

#### Create Docker Volume

**Endpoint:** `POST /api/v1/devices/:id/volumes`

**Request (Local Volume):**
```json
{
  "name": "my-local-volume",
  "type": "local"
}
```

**Request (NFS Volume):**
```json
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
```

**Response:**
```json
{
  "success": true,
  "message": "Volume created successfully",
  "volume": {
    "id": "uuid",
    "device_id": "uuid",
    "name": "my-nfs-volume",
    "type": "nfs",
    "driver": "local",
    "nfs_server_ip": "192.168.1.100",
    "nfs_path": "/srv/nfs/shared",
    "in_use": false,
    "created_at": "2025-10-09T20:00:00Z"
  }
}
```

#### List Docker Volumes

**Endpoint:** `GET /api/v1/devices/:id/volumes`

**Response:**
```json
{
  "volumes": [
    {
      "id": "uuid",
      "name": "my-nfs-volume",
      "type": "nfs",
      "driver": "local",
      "nfs_server_ip": "192.168.1.100",
      "nfs_path": "/srv/nfs/shared",
      "size": 1073741824,  // bytes
      "in_use": true,
      "created_at": "2025-10-09T20:00:00Z"
    }
  ]
}
```

#### Remove Docker Volume

**Endpoint:** `DELETE /api/v1/devices/:id/volumes/:name`

**Response:**
```json
{
  "success": true,
  "message": "Volume removed successfully"
}
```

**Error (Volume in use):**
```json
{
  "success": false,
  "error": "Volume is in use by container 'web-app'",
  "details": "Stop the container before removing the volume"
}
```

---

## Frontend UX

### Software Management Tab (Device Detail Page)

**Location:** Device Detail Page → "Software" tab

**UI Components:**

1. **Software Status Cards**
   - Docker: "Installed ✓" or "Not Installed"
   - NFS Server: "Configured ✓" or "Not Configured"
   - NFS Client: "Active (2 mounts)" or "Not Active"

2. **Quick Actions**
   - "Install Docker" button (if not installed)
   - "Configure NFS Server" button
   - "Mount NFS Storage" button

3. **Installed Software List**
   ```
   ┌─────────────────────────────────────────────────┐
   │ Installed Software                              │
   ├─────────────────────────────────────────────────┤
   │ 🐳 Docker                                        │
   │    Version: 24.0.7                              │
   │    Installed: 2 hours ago                       │
   │    [View Logs] [Uninstall]                      │
   ├─────────────────────────────────────────────────┤
   │ 📁 NFS Client                                   │
   │    Version: 1.3.4                               │
   │    Installed: 1 hour ago                        │
   │    [View Mounts] [Uninstall]                    │
   └─────────────────────────────────────────────────┘
   ```

### Docker Installation Dialog

**Trigger:** Click "Install Docker" button

**Dialog Content:**
```
┌───────────────────────────────────────────────────┐
│ Install Docker Engine                             │
├───────────────────────────────────────────────────┤
│                                                    │
│ This will install Docker Engine 24.x on the       │
│ device using the official installation script.    │
│                                                    │
│ Requirements:                                      │
│ ✓ Ubuntu 24.04                                    │
│ ✓ 10GB free disk space                           │
│ ✓ Sudo access                                     │
│ ✓ Internet connectivity                           │
│                                                    │
│ ☑ Add [username] to docker group (recommended)   │
│                                                    │
│ [Cancel]  [Install Docker]                        │
└───────────────────────────────────────────────────┘
```

**During Installation:**
```
┌───────────────────────────────────────────────────┐
│ Installing Docker...                               │
├───────────────────────────────────────────────────┤
│                                                    │
│ ⏳ Downloading installation script... ████████    │
│ ✓  Script downloaded                              │
│ ⏳ Installing packages... ████                     │
│                                                    │
│ Progress: 60%                                      │
│                                                    │
│ [View Detailed Logs]                              │
└───────────────────────────────────────────────────┘
```

**Success:**
```
┌───────────────────────────────────────────────────┐
│ ✓ Docker Installed Successfully                   │
├───────────────────────────────────────────────────┤
│                                                    │
│ Docker 24.0.7 is now running on your device.      │
│                                                    │
│ Next steps:                                        │
│ • Deploy your first container                     │
│ • Configure NFS volumes for shared storage        │
│ • Browse the app catalog                          │
│                                                    │
│ [Deploy an App] [Close]                           │
└───────────────────────────────────────────────────┘
```

### NFS Server Setup Wizard

**Trigger:** Click "Configure NFS Server" button

**Step 1: Choose Export Path**
```
┌───────────────────────────────────────────────────┐
│ Configure NFS Server - Step 1 of 3                │
├───────────────────────────────────────────────────┤
│                                                    │
│ Where should shared files be stored?              │
│                                                    │
│ Export Path: [/srv/nfs/shared________]            │
│                                                    │
│ This directory will be accessible to NFS clients. │
│                                                    │
│ Available Disk Space: 1.5 TB                      │
│                                                    │
│ [Back]  [Next]                                    │
└───────────────────────────────────────────────────┘
```

**Step 2: Configure Access**
```
┌───────────────────────────────────────────────────┐
│ Configure NFS Server - Step 2 of 3                │
├───────────────────────────────────────────────────┤
│                                                    │
│ Who can access this storage?                      │
│                                                    │
│ ○ Any device on my network (*)                    │
│ ● Specific subnet: [192.168.1.0/24]               │
│ ○ Specific IPs: [Add IP addresses...]             │
│                                                    │
│ Permissions:                                       │
│ ☑ Read and write access                           │
│ ☑ Allow root user access (needed for Docker)     │
│                                                    │
│ [Back]  [Next]                                    │
└───────────────────────────────────────────────────┘
```

**Step 3: Review and Confirm**
```
┌───────────────────────────────────────────────────┐
│ Configure NFS Server - Step 3 of 3                │
├───────────────────────────────────────────────────┤
│                                                    │
│ Review Configuration:                              │
│                                                    │
│ Export Path: /srv/nfs/shared                      │
│ Access: 192.168.1.0/24                            │
│ Permissions: Read-Write, Root Access              │
│                                                    │
│ What will happen:                                  │
│ 1. Install nfs-kernel-server package             │
│ 2. Create /srv/nfs/shared directory               │
│ 3. Configure /etc/exports                         │
│ 4. Start NFS server                               │
│ 5. Open firewall port 2049                        │
│                                                    │
│ [Back]  [Configure NFS Server]                    │
└───────────────────────────────────────────────────┘
```

### NFS Client Mount Dialog

**Trigger:** Click "Mount NFS Storage" button

```
┌───────────────────────────────────────────────────┐
│ Mount NFS Storage                                  │
├───────────────────────────────────────────────────┤
│                                                    │
│ NFS Server:                                        │
│ [Select device...  ▼] or [192.168.1.100]         │
│                                                    │
│ Remote Path: [/srv/nfs/shared________]            │
│                                                    │
│ Local Mount Point: [/mnt/nfs/shared__]            │
│                                                    │
│ Options:                                           │
│ ☑ Make permanent (add to fstab)                   │
│ ☑ Mount on boot                                   │
│                                                    │
│ Advanced Options (optional):                       │
│ Mount Options: [defaults____________]             │
│                                                    │
│ [Test Connection]  [Cancel]  [Mount]              │
└───────────────────────────────────────────────────┘
```

**Test Connection Result:**
```
✓ NFS server reachable
✓ Export /srv/nfs/shared is available
✓ Have permissions to mount
Ready to mount!
```

### Volume Manager

**Location:** Device Detail Page → "Volumes" tab

**UI:**
```
┌─────────────────────────────────────────────────┐
│ Docker Volumes                      [+ New Volume]│
├─────────────────────────────────────────────────┤
│                                                  │
│ 📦 my-nfs-volume                    [⋮]          │
│    Type: NFS                                     │
│    Server: 192.168.1.100:/srv/nfs/shared        │
│    Size: 50 GB used                              │
│    Status: ● In use by 2 containers              │
│                                                  │
├─────────────────────────────────────────────────┤
│ 📦 local-data                       [⋮]          │
│    Type: Local                                   │
│    Size: 10 GB                                   │
│    Status: ○ Not in use                          │
│                                                  │
└─────────────────────────────────────────────────┘
```

**Create Volume Dialog:**
```
┌───────────────────────────────────────────────────┐
│ Create Docker Volume                               │
├───────────────────────────────────────────────────┤
│                                                    │
│ Volume Name: [my-volume__________]                │
│                                                    │
│ Type:                                              │
│ ○ Local storage (on this device)                  │
│ ● Network storage (NFS)                           │
│                                                    │
│ NFS Configuration:                                 │
│ Server: [Select NFS server...  ▼]                 │
│         → 192.168.1.100 (NAS)                     │
│                                                    │
│ Path: [/srv/nfs/shared________]                   │
│                                                    │
│ Advanced Options:                                  │
│ [Show] Mount Options: defaults                    │
│                                                    │
│ [Cancel]  [Create Volume]                         │
└───────────────────────────────────────────────────┘
```

---

## Security Considerations

### Authentication & Authorization

**SSH User Permissions:**
- Must have passwordless sudo access for installation operations
- Recommend dedicated homelab admin user with limited sudo scope
- Store credentials securely in OS keychain

**Sudo Scope:**
```bash
# Example: Limited sudo for homelab user
homelab ALL=(ALL) NOPASSWD: /usr/bin/apt-get, /usr/bin/systemctl, /usr/bin/docker, /usr/sbin/exportfs, /usr/bin/mount, /usr/bin/umount
```

### Network Security

**NFS Security:**
- **Default:** Allow all clients (`*`) - convenient but insecure
- **Recommended:** Restrict to subnet (`192.168.1.0/24`)
- **Best:** Whitelist specific IPs

**Firewall Rules:**
- Use UFW or iptables to restrict NFS port 2049 to trusted subnet
- Don't expose NFS to public internet (no port forwarding!)

**Example UFW Rules:**
```bash
# Allow NFS only from local subnet
sudo ufw allow from 192.168.1.0/24 to any port nfs

# Deny from everywhere else
sudo ufw deny nfs
```

### Data Security

**NFS Export Options:**
- `no_root_squash` - Allows root access (needed for Docker, but risky)
- Consider `root_squash` for non-Docker shares
- Use `ro` (read-only) when possible

**Encryption:**
- NFS traffic is **not encrypted** by default
- For sensitive data, use:
  - VPN (Tailscale, WireGuard)
  - NFS over SSH tunnel
  - eCryptfs or LUKS for at-rest encryption

### Docker Security

**Volume Permissions:**
- Docker containers run as root by default
- NFS `no_root_squash` allows containers to write as root
- Consider rootless Docker for improved security

**Container Isolation:**
- Containers sharing NFS volumes can read each other's data
- Use separate NFS exports per application if needed

---

## Testing Strategy

### Unit Tests

**SoftwareService:**
```go
func TestInstallDocker_AlreadyInstalled(t *testing.T) {
    // Should return early without attempting installation
}

func TestInstallDocker_NoSudoAccess(t *testing.T) {
    // Should fail with clear error message
}

func TestInstallDocker_NoInternet(t *testing.T) {
    // Should fail with connectivity error
}
```

**NFSService:**
```go
func TestSetupNFSServer_Success(t *testing.T) {
    // Should create export directory and configure /etc/exports
}

func TestMountNFSShare_ServerUnreachable(t *testing.T) {
    // Should fail with connectivity error
}
```

### Integration Tests

**End-to-End Docker Installation:**
```go
func TestDockerInstallation_E2E(t *testing.T) {
    // 1. Add device
    // 2. Install Docker
    // 3. Verify Docker is running
    // 4. Verify database records installation
    // 5. Verify can run Docker commands
}
```

**NFS Server and Client:**
```go
func TestNFS_E2E(t *testing.T) {
    // 1. Setup server on Device A
    // 2. Setup client on Device B
    // 3. Write file on client
    // 4. Verify file appears on server
}
```

**Docker NFS Volume:**
```go
func TestDockerNFSVolume_E2E(t *testing.T) {
    // 1. Create NFS volume
    // 2. Start container using volume
    // 3. Write file in container
    // 4. Stop container
    // 5. Start new container with same volume
    // 6. Verify file still exists
}
```

### Manual Testing Checklist

**Docker Installation:**
- [ ] Fresh Ubuntu 24.04 - installs successfully
- [ ] Already has Docker - detects and skips
- [ ] No sudo access - fails with clear error
- [ ] No internet - fails with clear error
- [ ] Post-install - can run `docker ps` as user

**NFS Server:**
- [ ] Creates export directory with correct permissions
- [ ] Configures /etc/exports correctly
- [ ] NFS server starts and enables on boot
- [ ] `showmount -e localhost` shows export
- [ ] Firewall allows NFS port

**NFS Client:**
- [ ] Mounts share successfully
- [ ] Can read/write files
- [ ] Mount persists after reboot (if permanent)
- [ ] Unmount works correctly

**Docker NFS Volume:**
- [ ] Creates volume with correct driver options
- [ ] Container can use volume
- [ ] Data persists across container restarts
- [ ] Multiple containers can share volume
- [ ] Volume removal works (when not in use)

---

## Troubleshooting

### Docker Installation Issues

**Problem:** "curl: (6) Could not resolve host: get.docker.com"
- **Cause:** No internet connectivity
- **Fix:** Check network configuration, DNS settings

**Problem:** "E: Unable to locate package docker-ce"
- **Cause:** Docker repository not added correctly
- **Fix:** Manually add Docker's GPG key and repository

**Problem:** "docker: permission denied"
- **Cause:** User not in docker group
- **Fix:** `sudo usermod -aG docker $USER && newgrp docker`

**Problem:** "Cannot connect to the Docker daemon"
- **Cause:** Docker service not running
- **Fix:** `sudo systemctl start docker`

### NFS Server Issues

**Problem:** "exportfs: /srv/nfs/shared does not support NFS export"
- **Cause:** Directory doesn't exist or wrong filesystem type
- **Fix:** Ensure directory exists and is on local filesystem (not tmpfs)

**Problem:** "NFS server not starting"
- **Cause:** Port 2049 already in use
- **Fix:** `sudo lsof -i :2049` to find conflicting process

**Problem:** "Permission denied" when clients try to mount
- **Cause:** Client IP not in allowed list
- **Fix:** Update /etc/exports to include client subnet

### NFS Client Issues

**Problem:** "mount.nfs: Connection refused"
- **Cause:** NFS server not running
- **Fix:** On server: `sudo systemctl start nfs-kernel-server`

**Problem:** "mount.nfs: access denied by server"
- **Cause:** Client IP not allowed in /etc/exports
- **Fix:** Add client IP to server's /etc/exports

**Problem:** "mount.nfs: Connection timed out"
- **Cause:** Firewall blocking port 2049
- **Fix:** Open NFS port on server: `sudo ufw allow from <client-ip> to any port nfs`

**Problem:** "Stale file handle" errors
- **Cause:** Server rebooted while client had share mounted
- **Fix:** Unmount and remount: `sudo umount /mnt/nfs/shared && sudo mount -a`

### Docker NFS Volume Issues

**Problem:** "Error mounting volume: invalid argument"
- **Cause:** Incorrect driver options
- **Fix:** Verify NFS server is reachable, path exists

**Problem:** "Permission denied" writing to NFS volume
- **Cause:** `root_squash` enabled on server
- **Fix:** Change export to use `no_root_squash` (or use user namespaces)

**Problem:** "Device or resource busy" when removing volume
- **Cause:** Volume is in use by a container
- **Fix:** Stop containers first: `docker ps -a --filter volume=<volume-name>`

**Problem:** Poor performance with NFS volumes
- **Cause:** Default mount options not optimized
- **Fix:** Add performance tuning options:
  ```
  o: addr=<ip>,rw,soft,timeo=30,rsize=32768,wsize=32768,tcp
  ```

---

## Appendices

### A. Ubuntu 24.04 Package Versions

As of October 2025:
- **Docker Engine:** 24.0.7
- **nfs-kernel-server:** 1:2.6.4-1ubuntu1
- **nfs-common:** 1:2.6.4-1ubuntu1

### B. Useful Commands Reference

**Docker:**
```bash
# Check Docker version
docker --version

# Check Docker daemon status
sudo systemctl status docker

# View Docker info
docker info

# List volumes
docker volume ls

# Inspect volume
docker volume inspect <volume-name>
```

**NFS Server:**
```bash
# Show current exports
sudo exportfs -v

# Re-export all shares (after editing /etc/exports)
sudo exportfs -ra

# Check NFS server status
sudo systemctl status nfs-kernel-server

# Show who is connected
sudo showmount -a
```

**NFS Client:**
```bash
# Show available exports from server
showmount -e <server-ip>

# Mount NFS share
sudo mount -t nfs <server-ip>:/path /mount/point

# Unmount NFS share
sudo umount /mount/point

# View active NFS mounts
mount | grep nfs

# Test NFS connectivity
rpcinfo -p <server-ip>
```

**Networking:**
```bash
# Test port 2049 is open
nc -zv <server-ip> 2049

# Check firewall rules
sudo ufw status verbose

# Monitor NFS traffic
sudo tcpdump -i any port 2049
```

### C. Performance Tuning

**NFS Mount Options for Performance:**
```
# For bulk data (large files, sequential access)
rw,sync,hard,intr,rsize=131072,wsize=131072,tcp

# For databases (small random I/O)
rw,sync,hard,intr,rsize=8192,wsize=8192,tcp,actimeo=3

# For read-heavy workloads
rw,async,noatime,rsize=131072,wsize=131072,tcp
```

**Docker Volume Driver Options:**
```yaml
# Optimized for media libraries (large files)
driver_opts:
  type: nfs
  o: addr=<ip>,rw,soft,timeo=30,rsize=32768,wsize=32768,tcp,noatime
  device: ":/srv/nfs/media"

# Optimized for databases (safety over performance)
driver_opts:
  type: nfs
  o: addr=<ip>,rw,hard,intr,rsize=8192,wsize=8192,tcp,sync
  device: ":/srv/nfs/database"
```

### D. Future Enhancements

**Planned Features:**
- Automated NFS server discovery on local network
- Storage quota management per export
- Automated backup of NFS exports
- NFS over TLS (NFSv4.2 with Kerberos)
- RAID configuration wizard
- S3-compatible object storage (MinIO) integration
- Automated performance testing and recommendations

---

**End of Document**

**Document Version:** 1.0
**Last Updated:** October 2025
**Maintained By:** Homelab Orchestration Platform Team
**Feedback:** Please submit issues or suggestions to the GitHub repository
