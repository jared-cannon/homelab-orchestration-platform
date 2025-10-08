import type {
  Device,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  TestConnectionRequest,
  TestConnectionResponse,
  StartScanRequest,
  StartScanResponse,
  ScanProgress,
} from './types'

const API_BASE_URL = '/api/v1'

class APIClient {
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${API_BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({}))
      throw new Error(error.error || `HTTP ${response.status}`)
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
}

export const apiClient = new APIClient()
