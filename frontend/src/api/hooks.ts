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

// Marketplace query keys
export const marketplaceKeys = {
  all: ['marketplace'] as const,
  recipes: () => [...marketplaceKeys.all, 'recipes'] as const,
  recipe: (slug: string) => [...marketplaceKeys.recipes(), slug] as const,
  recipesByCategory: (category?: string) => [...marketplaceKeys.recipes(), { category }] as const,
  categories: () => [...marketplaceKeys.all, 'categories'] as const,
}

// Marketplace hooks
export function useRecipes(category?: string) {
  return useQuery({
    queryKey: marketplaceKeys.recipesByCategory(category),
    queryFn: () => apiClient.listRecipes(category),
  })
}

export function useRecipe(slug: string) {
  return useQuery({
    queryKey: marketplaceKeys.recipe(slug),
    queryFn: () => apiClient.getRecipe(slug),
    enabled: !!slug,
  })
}

export function useRecipeCategories() {
  return useQuery({
    queryKey: marketplaceKeys.categories(),
    queryFn: () => apiClient.getRecipeCategories(),
  })
}

export function useValidateDeployment() {
  return useMutation({
    mutationFn: ({ slug, data }: { slug: string; data: import('./types').ValidateDeploymentRequest }) =>
      apiClient.validateDeployment(slug, data),
  })
}

export function useRecommendDevice(slug: string) {
  return useQuery({
    queryKey: [...marketplaceKeys.recipe(slug), 'recommendations'],
    queryFn: () => apiClient.recommendDeviceForRecipe(slug),
    enabled: !!slug,
  })
}

// Deployment query keys
export const deploymentKeys = {
  all: ['deployments'] as const,
  lists: () => [...deploymentKeys.all, 'list'] as const,
  list: (filters: Record<string, any> = {}) =>
    [...deploymentKeys.lists(), filters] as const,
  details: () => [...deploymentKeys.all, 'detail'] as const,
  detail: (id: string) => [...deploymentKeys.details(), id] as const,
}

// Deployment hooks
export function useDeployments(deviceId?: string, status?: string) {
  const queryClient = useQueryClient()

  // Subscribe to WebSocket updates for real-time deployment list updates
  useEffect(() => {
    const handleDeploymentUpdates = (event: string) => {
      if (event === 'deployment:status' || event === 'deployment:created' || event === 'deployment:deleted') {
        // Invalidate all deployment list queries to refetch with updated data
        queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() })
      }
    }

    // Subscribe to deployments channel
    const unsubscribe = wsService.on('deployments', handleDeploymentUpdates)

    return () => {
      unsubscribe()
    }
  }, [queryClient])

  return useQuery({
    queryKey: deploymentKeys.list({ deviceId, status }),
    queryFn: () => apiClient.listDeployments(deviceId, status),
  })
}

export function useDeployment(id: string) {
  const queryClient = useQueryClient()

  // Subscribe to WebSocket updates for real-time status and logs
  useEffect(() => {
    if (!id) return

    const handleDeploymentUpdates = (event: string, data: unknown) => {
      if (event === 'deployment:status') {
        const statusUpdate = data as { id: string; status: string; error_details?: string }
        // Only update if this is our deployment
        if (statusUpdate.id === id) {
          queryClient.setQueryData(deploymentKeys.detail(id), (old: any) => ({
            ...old,
            status: statusUpdate.status,
            error_details: statusUpdate.error_details,
          }))
        }
      } else if (event === 'deployment:log') {
        const logUpdate = data as { id: string; message: string }
        // Only update if this is our deployment
        if (logUpdate.id === id) {
          queryClient.setQueryData(deploymentKeys.detail(id), (old: any) => ({
            ...old,
            deployment_logs: (old?.deployment_logs || '') + logUpdate.message,
          }))
        }
      }
    }

    // Subscribe to deployments channel
    const unsubscribe = wsService.on('deployments', handleDeploymentUpdates)

    return () => {
      unsubscribe()
    }
  }, [id, queryClient])

  return useQuery({
    queryKey: deploymentKeys.detail(id),
    queryFn: () => apiClient.getDeployment(id),
    enabled: !!id,
    // Reduced polling as fallback (WebSocket is primary)
    refetchInterval: (query) => {
      const data = query.state.data as import('./types').Deployment | undefined
      // Stop polling when deployment is complete or failed
      if (data?.status === 'running' || data?.status === 'failed') {
        return false
      }
      // Poll every 10 seconds as fallback (WebSocket should be faster)
      return 10000
    },
  })
}

export function useCreateDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: import('./types').CreateDeploymentRequest) =>
      apiClient.createDeployment(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() })
    },
  })
}

export function useDeleteDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiClient.deleteDeployment(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() })
    },
  })
}

export function useCancelDeployment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => apiClient.cancelDeployment(id),
    onSuccess: (_, id) => {
      // Invalidate the specific deployment and lists to refetch with updated status
      queryClient.invalidateQueries({ queryKey: deploymentKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: deploymentKeys.lists() })
    },
  })
}

// Software Installation query keys
export const softwareInstallationKeys = {
  all: ['software-installations'] as const,
  installation: (id: string) => [...softwareInstallationKeys.all, id] as const,
  active: (deviceId: string) => [...softwareInstallationKeys.all, 'active', deviceId] as const,
}

// Software Installation hooks
export function useSoftwareInstallation(deviceId: string, installationId: string) {
  const queryClient = useQueryClient()

  // Subscribe to WebSocket updates for real-time status and logs
  useEffect(() => {
    if (!installationId) return

    const handleInstallationUpdates = (event: string, data: unknown) => {
      if (event === 'software:status') {
        const statusUpdate = data as { id: string; status: string; error_details?: string }
        // Only update if this is our installation
        if (statusUpdate.id === installationId) {
          queryClient.setQueryData(softwareInstallationKeys.installation(installationId), (old: any) => ({
            ...old,
            status: statusUpdate.status,
            error_details: statusUpdate.error_details,
          }))
        }
      } else if (event === 'software:log') {
        const logUpdate = data as { id: string; message: string }
        // Only update if this is our installation
        if (logUpdate.id === installationId) {
          queryClient.setQueryData(softwareInstallationKeys.installation(installationId), (old: any) => ({
            ...old,
            install_logs: (old?.install_logs || '') + logUpdate.message,
          }))
        }
      }
    }

    // Subscribe to software channel
    const unsubscribe = wsService.on('software', handleInstallationUpdates)

    return () => {
      unsubscribe()
    }
  }, [installationId, queryClient])

  return useQuery({
    queryKey: softwareInstallationKeys.installation(installationId),
    queryFn: () => apiClient.getSoftwareInstallation(deviceId, installationId),
    enabled: !!installationId && !!deviceId,
    // Reduced polling as fallback (WebSocket is primary)
    refetchInterval: (query) => {
      const data = query.state.data as import('./types').SoftwareInstallation | undefined
      // Stop polling when installation is complete or failed
      if (data?.status === 'success' || data?.status === 'failed') {
        return false
      }
      // Poll every 5 seconds as fallback (WebSocket should be faster)
      return 5000
    },
  })
}

export function useActiveInstallation(deviceId: string) {
  const queryClient = useQueryClient()

  // Subscribe to WebSocket updates for real-time status and logs
  useEffect(() => {
    if (!deviceId) return

    const handleInstallationUpdates = (event: string, _data: unknown) => {
      // When software status changes, refetch active installation
      if (event === 'software:status' || event === 'software:log') {
        queryClient.invalidateQueries({ queryKey: softwareInstallationKeys.active(deviceId) })
      }
    }

    // Subscribe to software channel
    const unsubscribe = wsService.on('software', handleInstallationUpdates)

    return () => {
      unsubscribe()
    }
  }, [deviceId, queryClient])

  return useQuery({
    queryKey: softwareInstallationKeys.active(deviceId),
    queryFn: () => apiClient.getActiveInstallation(deviceId),
    enabled: !!deviceId,
    // Poll every 5 seconds as fallback
    refetchInterval: 5000,
  })
}
