// TypeScript types for API requests and responses
// Core models (Device, Application, Deployment) are imported from generated-types

import type { DeviceType } from './generated-types'

// Re-export core types from generated-types for convenience
export type { Device, DeviceType, DeviceStatus, Application, Deployment, DeploymentStatus } from './generated-types'

export interface DeviceCredentials {
  type: 'password' | 'ssh_key'
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
  total_hosts: number
  scanned_hosts: number
  discovered_count: number
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
