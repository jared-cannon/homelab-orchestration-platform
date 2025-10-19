// TypeScript types for API requests and responses
// Core models (Device, Application, Deployment) are imported from generated-types

import type { Device, DeviceType } from './generated-types'

// Re-export core types from generated-types for convenience
export type { Device, DeviceType, DeviceStatus, PrimaryConnection, Application, Deployment, DeploymentStatus } from './generated-types'

export interface DeviceCredentials {
  type: 'auto' | 'password' | 'ssh_key' | 'tailscale'
  username: string
  password?: string
  ssh_key?: string
  ssh_key_passwd?: string
}

export interface CreateDeviceRequest {
  name: string
  type: DeviceType
  local_ip_address: string
  tailscale_address?: string
  primary_connection?: 'local' | 'tailscale'
  mac_address?: string
  metadata?: string
  credentials: DeviceCredentials
}

export interface UpdateDeviceRequest {
  name?: string
  type?: DeviceType
  local_ip_address?: string
  tailscale_address?: string
  primary_connection?: 'local' | 'tailscale'
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

// Software Installation types
export type SoftwareInstallationStatus = 'pending' | 'installing' | 'success' | 'failed'

export interface SoftwareInstallation {
  id: string
  device_id: string
  software_name: SoftwareType
  status: SoftwareInstallationStatus
  install_logs?: string
  error_details?: string
  created_at: string
  completed_at?: string
  device?: Device
}

// Marketplace types
export interface Recipe {
  id: string
  name: string
  slug: string
  category: string
  tagline: string
  description: string
  icon_url: string
  resources: RecipeResources
  compose_template: string
  config_options: RecipeConfigOption[]
  post_deploy_instructions: string
  health_check: RecipeHealthCheck
}

export interface RecipeResources {
  min_ram_mb: number
  min_storage_gb: number
  recommended_ram_mb: number
  recommended_storage_gb: number
  cpu_cores: number
}

export interface RecipeConfigOption {
  name: string
  label: string
  type: 'string' | 'number' | 'boolean' | 'password' | 'secret'
  default: string | number | boolean
  required: boolean
  description: string
}

export interface RecipeHealthCheck {
  path: string
  port: number
  expected_status: number
  timeout_seconds: number
}

export interface ValidationResult {
  valid: boolean
  errors?: string[]
  warnings?: string[]
  resource_check?: ResourceCheck
  port_conflicts?: number[]
  rendered_compose?: string
}

export interface ResourceCheck {
  required_ram_mb: number
  available_ram_mb: number
  ram_sufficient: boolean
  required_storage_gb: number
  available_storage_gb: number
  storage_sufficient: boolean
  docker_installed: boolean
  docker_running: boolean
}

export interface ValidateDeploymentRequest {
  device_id: string
  config: Record<string, any>
}

export interface DeviceScore {
  device_id: string
  device_name: string
  device_ip: string
  score: number // 0-100
  recommendation: 'best' | 'good' | 'acceptable' | 'not-recommended'
  reasons: string[]
  available: boolean
}

// Deployment types
export interface CreateDeploymentRequest {
  recipe_slug: string
  device_id: string
  config: Record<string, any>
}

// Curated Marketplace types
export interface SaaSReplacement {
  name: string
  comparison_url?: string
}

export interface CuratedRecipe extends Recipe {
  saas_replacements?: SaaSReplacement[]
  difficulty_level?: 'beginner' | 'intermediate' | 'advanced'
  setup_time_minutes?: number
  feature_highlights?: string[]
}

export interface DeploymentInfo {
  status: string
  device_name: string
  access_url?: string
  deployed_at?: string
}

export interface CuratedMarketplaceStats {
  total_curated: number
  deployed: number
  percentage: number
}

export interface CuratedMarketplaceResponse {
  recipes: CuratedRecipe[]
  user_deployments: Record<string, DeploymentInfo>
  stats: CuratedMarketplaceStats
}

// Dependency Check types
export interface DependencyToProvision {
  type: string
  name?: string
  engine?: string
  purpose?: string
  message?: string
  estimated_time_seconds?: number
  estimated_ram_mb?: number
  estimated_storage_gb?: number
}

export interface MissingDependency {
  type: string
  name?: string
  engine?: string
  purpose?: string
  message?: string
}

export interface DependencyCheckResult {
  satisfied: boolean
  missing: MissingDependency[]
  to_provision: DependencyToProvision[]
  total_estimated_time_seconds: number
  total_estimated_ram_mb: number
  total_estimated_storage_gb: number
}
