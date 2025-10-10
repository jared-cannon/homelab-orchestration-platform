// TypeScript types for API requests and responses
// Core models (Device, Application, Deployment) are imported from generated-types

import type { DeviceType } from './generated-types'

// Re-export core types from generated-types for convenience
export type { Device, DeviceType, DeviceStatus, Application, Deployment, DeploymentStatus } from './generated-types'

export interface DeviceCredentials {
  type: 'auto' | 'password' | 'ssh_key'
  username: string
  password?: string
  ssh_key?: string
  ssh_key_passwd?: string
}

export interface CreateDeviceRequest {
  name: string
  type: DeviceType
  ip_address: string
  mac_address?: string
  metadata?: string
  credentials: DeviceCredentials
}

export interface UpdateDeviceRequest {
  name?: string
  type?: DeviceType
  mac_address?: string
  metadata?: string
}

export interface UpdateCredentialsRequest {
  credentials: DeviceCredentials
}

export interface TestConnectionRequest {
  ip_address: string
  credentials: DeviceCredentials
}

export interface TestConnectionResponse {
  success?: boolean
  ssh_connection?: boolean
  docker_installed?: boolean
  docker_version?: string
  docker_running?: boolean
  docker_error?: string
  docker_compose_installed?: boolean
  docker_compose_version?: string
  system_info?: string
  error?: string
  details?: string
}

export interface WebSocketMessage {
  channel: string
  event: string
  data: any
}

// Scanner types
export interface DiscoveredDevice {
  ip_address: string
  mac_address?: string
  hostname?: string
  type: DeviceType
  ssh_available: boolean
  docker_detected: boolean
  services_detected?: string[] // e.g., ["docker", "portainer", "proxmox"]
  os?: string // e.g., "Ubuntu 22.04", "Synology DSM"
  status: 'discovered' | 'checking_credentials' | 'ready' | 'needs_credentials' | 'already_added'
  credential_status?: 'working' | 'failed' | 'untested'
  credential_id?: string // ID of working credential
  already_added?: boolean // True if device already exists in database
}

export interface ScanProgress {
  id: string
  status: 'scanning' | 'completed' | 'failed'
  phase?: 'ping' | 'ssh_scan' | 'credential_test' | 'completed' // Current phase of the scan
  total_hosts: number
  scanned_hosts: number
  discovered_count: number
  current_ip?: string // IP currently being scanned
  scan_rate?: number // IPs per second
  devices: DiscoveredDevice[]
  error?: string
  started_at: string
  completed_at?: string
}

export interface StartScanRequest {
  cidr?: string
}

export interface StartScanResponse {
  scan_id: string
  cidr: string
  message: string
}

// Software Management types
export type SoftwareType = 'docker' | 'docker-compose' | 'nfs-server' | 'nfs-client'

export interface InstalledSoftware {
  id: string
  device_id: string
  name: SoftwareType
  version: string
  installed_at: string
  installed_by: string
  created_at: string
  updated_at: string
}

export interface InstallSoftwareRequest {
  software_type: SoftwareType
  add_user_to_group?: boolean // For Docker only
}

// NFS types
export interface NFSExport {
  id: string
  device_id: string
  path: string
  client_cidr: string
  options: string
  active: boolean
  created_at: string
  updated_at: string
}

export interface NFSMount {
  id: string
  device_id: string
  server_ip: string
  remote_path: string
  local_path: string
  options: string
  permanent: boolean
  active: boolean
  created_at: string
  updated_at: string
}

export interface SetupNFSServerRequest {
  export_path: string
  client_cidr?: string
  options?: string
}

export interface CreateExportRequest {
  export_path: string
  client_cidr?: string
  options?: string
}

export interface MountNFSShareRequest {
  server_ip: string
  remote_path: string
  local_path: string
  options?: string
  permanent: boolean
}

// Volume types
export type VolumeType = 'local' | 'nfs'

export interface Volume {
  id: string
  device_id: string
  name: string
  type: VolumeType
  driver: string
  driver_opts?: Record<string, string>
  nfs_server_ip?: string
  nfs_path?: string
  size: number
  in_use: boolean
  created_at: string
  updated_at: string
}

export interface CreateVolumeRequest {
  name: string
  type: VolumeType
  nfs_server_ip?: string
  nfs_path?: string
  options?: Record<string, string>
}

// Software update types
export interface SoftwareUpdateInfo {
  software_id: string
  current_version: string
  available_version?: string
  update_available: boolean
  message?: string
}
