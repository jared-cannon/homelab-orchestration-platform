import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { apiClient } from './client'
import { wsService } from '../services/websocket'
import type {
  CreateDeviceRequest,
  UpdateDeviceRequest,
  UpdateCredentialsRequest,
  TestConnectionRequest,
  StartScanRequest,
  ScanProgress,
} from './types'

// Query keys
export const deviceKeys = {
  all: ['devices'] as const,
  lists: () => [...deviceKeys.all, 'list'] as const,
  list: (filters: Record<string, any> = {}) =>
    [...deviceKeys.lists(), filters] as const,
  details: () => [...deviceKeys.all, 'detail'] as const,
  detail: (id: string) => [...deviceKeys.details(), id] as const,
}

// Device hooks
export function useDevices() {
  return useQuery({
    queryKey: deviceKeys.lists(),
    queryFn: () => apiClient.listDevices(),
  })
}

export function useDevice(id: string, options?: { refetchInterval?: number }) {
  return useQuery({
    queryKey: deviceKeys.detail(id),
    queryFn: () => apiClient.getDevice(id),
    enabled: !!id,
    refetchInterval: options?.refetchInterval,
  })
}

export function useCreateDevice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateDeviceRequest) => apiClient.createDevice(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() })
    },
  })
}

export function useUpdateDevice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateDeviceRequest }) =>
      apiClient.updateDevice(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: deviceKeys.detail(variables.id) })
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() })
    },
  })
}

export function useDeleteDevice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiClient.deleteDevice(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deviceKeys.lists() })
    },
  })
}

export function useTestConnection() {
  return useMutation({
    mutationFn: (id: string) => apiClient.testConnection(id),
  })
}

export function useTestConnectionBeforeCreate() {
  return useMutation({
    mutationFn: (data: TestConnectionRequest) =>
      apiClient.testConnectionBeforeCreate(data),
  })
}

export function useUpdateDeviceCredentials() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateCredentialsRequest }) =>
      apiClient.updateDeviceCredentials(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: deviceKeys.detail(variables.id) })
    },
  })
}

// Scanner query keys
export const scannerKeys = {
  all: ['scanner'] as const,
  scans: () => [...scannerKeys.all, 'scan'] as const,
  scan: (id: string) => [...scannerKeys.scans(), id] as const,
}

// Scanner hooks
export function useStartScan() {
  return useMutation({
    mutationFn: (data: StartScanRequest = {}) => apiClient.startScan(data),
  })
}

export function useScanProgress(scanId: string) {
  const queryClient = useQueryClient()

  // Subscribe to WebSocket updates for real-time scan progress
  useEffect(() => {
    if (!scanId) return

    const handleScanProgress = (event: string, data: unknown) => {
      if (event === 'scan:progress') {
        const progress = data as ScanProgress
        // Only update if this is our scan
        if (progress.id === scanId) {
          queryClient.setQueryData(scannerKeys.scan(scanId), progress)
        }
      }
    }

    // Subscribe to scanner channel
    const unsubscribe = wsService.on('scanner', handleScanProgress)

    return () => {
      unsubscribe()
    }
  }, [scanId, queryClient])

  return useQuery({
    queryKey: scannerKeys.scan(scanId),
    queryFn: () => apiClient.getScanProgress(scanId),
    enabled: !!scanId,
    refetchInterval: (query) => {
      const data = query.state.data
      // Stop polling when scan is completed or failed
      if (data?.status === 'completed' || data?.status === 'failed') {
        return false
      }
      // Reduce polling frequency to every 5 seconds as fallback (WebSocket is primary)
      return 5000
    },
  })
}

export function useDetectNetwork() {
  return useQuery({
    queryKey: [...scannerKeys.all, 'detect-network'],
    queryFn: () => apiClient.detectNetwork(),
  })
}
