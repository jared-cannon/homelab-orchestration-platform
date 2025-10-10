import type {
  Device,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  UpdateCredentialsRequest,
  TestConnectionRequest,
  TestConnectionResponse,
  StartScanRequest,
  StartScanResponse,
  ScanProgress,
  InstalledSoftware,
  InstallSoftwareRequest,
  SoftwareInstallation,
  NFSExport,
  NFSMount,
  SetupNFSServerRequest,
  CreateExportRequest,
  MountNFSShareRequest,
  Volume,
  CreateVolumeRequest,
  SoftwareUpdateInfo,
  Recipe,
  ValidationResult,
  ValidateDeploymentRequest,
  DeviceScore,
  Deployment,
  CreateDeploymentRequest,
} from './types'
import { useAuthStore } from '../stores/authStore'

const API_BASE_URL = '/api/v1'

// Structured API Error
export class APIError extends Error {
  code?: string
  details?: Record<string, any>

  constructor(message: string, code?: string, details?: Record<string, any>) {
    super(message)
    this.name = 'APIError'
    this.code = code
    this.details = details
  }
}

class APIClient {
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    // Get token from auth store
    const token = useAuthStore.getState().token

    // Build headers with auth token if available
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    // Add existing headers if provided
    if (options?.headers) {
      Object.entries(options.headers).forEach(([key, value]) => {
        if (typeof value === 'string') {
          headers[key] = value
        }
      })
    }

    // Add Authorization header if token exists
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      throw new APIError(
        errorData.error || `HTTP ${response.status}`,
        errorData.code,
        errorData.details
      )
    }

    if (response.status === 204) {
      return {} as T
    }

    return response.json()
  }

  // Device API
  async listDevices(): Promise<Device[]> {
    return this.request<Device[]>('/devices')
  }

  async getDevice(id: string): Promise<Device> {
    return this.request<Device>(`/devices/${id}`)
  }

  async createDevice(data: CreateDeviceRequest): Promise<Device> {
    return this.request<Device>('/devices', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDevice(
    id: string,
    data: UpdateDeviceRequest
  ): Promise<Device> {
    return this.request<Device>(`/devices/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    })
  }

  async deleteDevice(id: string): Promise<void> {
    return this.request<void>(`/devices/${id}`, {
      method: 'DELETE',
    })
  }

  async testConnection(id: string): Promise<TestConnectionResponse> {
    return this.request<TestConnectionResponse>(
      `/devices/${id}/test-connection`,
      {
        method: 'POST',
      }
    )
  }

  async testConnectionBeforeCreate(
    data: TestConnectionRequest
  ): Promise<TestConnectionResponse> {
    return this.request<TestConnectionResponse>('/devices/test', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateDeviceCredentials(
    id: string,
    data: UpdateCredentialsRequest
  ): Promise<{ message: string }> {
    return this.request<{ message: string }>(`/devices/${id}/credentials`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    })
  }

  // Scanner API
  async startScan(data: StartScanRequest = {}): Promise<StartScanResponse> {
    return this.request<StartScanResponse>('/devices/scan', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getScanProgress(scanId: string): Promise<ScanProgress> {
    return this.request<ScanProgress>(`/devices/scan/${scanId}`)
  }

  async detectNetwork(): Promise<{ cidr: string }> {
    return this.request<{ cidr: string }>('/devices/scan/detect-network')
  }

  // Software Management API
  async listInstalledSoftware(deviceId: string): Promise<InstalledSoftware[]> {
    return this.request<InstalledSoftware[]>(`/devices/${deviceId}/software`)
  }

  async detectInstalledSoftware(deviceId: string): Promise<InstalledSoftware[]> {
    return this.request<InstalledSoftware[]>(`/devices/${deviceId}/software/detect`, {
      method: 'POST',
    })
  }

  async installSoftware(
    deviceId: string,
    data: InstallSoftwareRequest
  ): Promise<SoftwareInstallation> {
    return this.request<SoftwareInstallation>(`/devices/${deviceId}/software`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getSoftwareInstallation(
    deviceId: string,
    installationId: string
  ): Promise<SoftwareInstallation> {
    return this.request<SoftwareInstallation>(
      `/devices/${deviceId}/software/installations/${installationId}`
    )
  }

  async getActiveInstallation(
    deviceId: string
  ): Promise<SoftwareInstallation | null> {
    const response = await this.request<SoftwareInstallation | { installation: null }>(
      `/devices/${deviceId}/software/installations/active`
    )
    // Backend returns {installation: null} when there's no active installation
    if (response && 'installation' in response && response.installation === null) {
      return null
    }
    return response as SoftwareInstallation
  }

  async uninstallSoftware(deviceId: string, name: string): Promise<void> {
    return this.request<void>(`/devices/${deviceId}/software/${name}`, {
      method: 'DELETE',
    })
  }

  async checkSoftwareUpdates(deviceId: string): Promise<SoftwareUpdateInfo[]> {
    return this.request<SoftwareUpdateInfo[]>(`/devices/${deviceId}/software/updates`)
  }

  async updateSoftware(deviceId: string, name: string): Promise<InstalledSoftware> {
    return this.request<InstalledSoftware>(`/devices/${deviceId}/software/${name}/update`, {
      method: 'POST',
    })
  }

  // NFS Server API
  async setupNFSServer(
    deviceId: string,
    data: SetupNFSServerRequest
  ): Promise<NFSExport> {
    return this.request<NFSExport>(`/devices/${deviceId}/nfs/server/setup`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async listNFSExports(deviceId: string): Promise<NFSExport[]> {
    return this.request<NFSExport[]>(`/devices/${deviceId}/nfs/exports`)
  }

  async createNFSExport(
    deviceId: string,
    data: CreateExportRequest
  ): Promise<NFSExport> {
    return this.request<NFSExport>(`/devices/${deviceId}/nfs/exports`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async removeNFSExport(deviceId: string, exportId: string): Promise<void> {
    return this.request<void>(`/devices/${deviceId}/nfs/exports/${exportId}`, {
      method: 'DELETE',
    })
  }

  // NFS Client API
  async listNFSMounts(deviceId: string): Promise<NFSMount[]> {
    return this.request<NFSMount[]>(`/devices/${deviceId}/nfs/mounts`)
  }

  async mountNFSShare(
    deviceId: string,
    data: MountNFSShareRequest
  ): Promise<NFSMount> {
    return this.request<NFSMount>(`/devices/${deviceId}/nfs/mounts`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async unmountNFSShare(
    deviceId: string,
    mountId: string,
    removeFromFstab = true
  ): Promise<void> {
    return this.request<void>(
      `/devices/${deviceId}/nfs/mounts/${mountId}?remove_from_fstab=${removeFromFstab}`,
      {
        method: 'DELETE',
      }
    )
  }

  // Volume Management API
  async listVolumes(deviceId: string): Promise<Volume[]> {
    return this.request<Volume[]>(`/devices/${deviceId}/volumes`)
  }

  async createVolume(
    deviceId: string,
    data: CreateVolumeRequest
  ): Promise<Volume> {
    return this.request<Volume>(`/devices/${deviceId}/volumes`, {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getVolume(deviceId: string, volumeName: string): Promise<Volume> {
    return this.request<Volume>(`/devices/${deviceId}/volumes/${volumeName}`)
  }

  async inspectVolume(
    deviceId: string,
    volumeName: string
  ): Promise<Record<string, any>> {
    return this.request<Record<string, any>>(
      `/devices/${deviceId}/volumes/${volumeName}/inspect`
    )
  }

  async removeVolume(
    deviceId: string,
    volumeName: string,
    force = false
  ): Promise<void> {
    return this.request<void>(
      `/devices/${deviceId}/volumes/${volumeName}?force=${force}`,
      {
        method: 'DELETE',
      }
    )
  }

  // Marketplace API
  async listRecipes(category?: string): Promise<Recipe[]> {
    const params = category ? `?category=${encodeURIComponent(category)}` : ''
    return this.request<Recipe[]>(`/marketplace/recipes${params}`)
  }

  async getRecipe(slug: string): Promise<Recipe> {
    return this.request<Recipe>(`/marketplace/recipes/${slug}`)
  }

  async validateDeployment(
    slug: string,
    data: ValidateDeploymentRequest
  ): Promise<ValidationResult> {
    return this.request<ValidationResult>(
      `/marketplace/recipes/${slug}/validate`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      }
    )
  }

  async getRecipeCategories(): Promise<string[]> {
    return this.request<string[]>('/marketplace/categories')
  }

  async recommendDeviceForRecipe(slug: string): Promise<DeviceScore[]> {
    return this.request<DeviceScore[]>(
      `/marketplace/recipes/${slug}/recommend-device`,
      {
        method: 'POST',
      }
    )
  }

  // Deployment API
  async createDeployment(data: CreateDeploymentRequest): Promise<Deployment> {
    return this.request<Deployment>('/deployments', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async getDeployment(id: string): Promise<Deployment> {
    return this.request<Deployment>(`/deployments/${id}`)
  }

  async listDeployments(deviceId?: string, status?: string): Promise<Deployment[]> {
    const params = new URLSearchParams()
    if (deviceId) params.append('device_id', deviceId)
    if (status) params.append('status', status)
    const queryString = params.toString()
    return this.request<Deployment[]>(`/deployments${queryString ? `?${queryString}` : ''}`)
  }

  async deleteDeployment(id: string): Promise<void> {
    return this.request<void>(`/deployments/${id}`, {
      method: 'DELETE',
    })
  }

  async cancelDeployment(id: string): Promise<void> {
    return this.request<void>(`/deployments/${id}/cancel`, {
      method: 'POST',
    })
  }
}

export const apiClient = new APIClient()
