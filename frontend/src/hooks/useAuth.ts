import { useEffect } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { authAPI } from '../api/auth'
import { useAuthStore } from '../stores/authStore'
import type { User } from '../stores/authStore'

// Query keys for auth-related queries
export const authKeys = {
  all: ['auth'] as const,
  currentUser: () => [...authKeys.all, 'current-user'] as const,
  setupRequired: () => [...authKeys.all, 'setup-required'] as const,
}

/**
 * Hook for login functionality
 * Handles login mutation and updates auth store on success
 */
export function useLogin() {
  const login = useAuthStore((state) => state.login)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ username, password }: { username: string; password: string }) =>
      authAPI.login(username, password),
    onSuccess: (data) => {
      // Update auth store with token and user
      login(data.token, data.user)
      // Invalidate and refetch current user
      queryClient.invalidateQueries({ queryKey: authKeys.currentUser() })
      // Navigate to home page
      navigate('/')
    },
  })
}

/**
 * Hook for registration functionality
 * Handles registration mutation and updates auth store on success
 */
export function useRegister() {
  const login = useAuthStore((state) => state.login)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      username,
      password,
      email,
    }: {
      username: string
      password: string
      email?: string
    }) => authAPI.register(username, password, email),
    onSuccess: (data) => {
      // Update auth store with token and user
      login(data.token, data.user)
      // Invalidate queries
      queryClient.invalidateQueries({ queryKey: authKeys.all })
      // Navigate to home page
      navigate('/')
    },
  })
}

/**
 * Hook for logout functionality
 * Clears auth state and navigates to login
 */
export function useLogout() {
  const logout = useAuthStore((state) => state.logout)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  return () => {
    logout()
    // Clear all queries
    queryClient.clear()
    // Navigate to login
    navigate('/login')
  }
}

/**
 * Hook to get current user from auth store
 * Provides easy access to auth state
 */
export function useCurrentUser() {
  const user = useAuthStore((state) => state.user)
  const token = useAuthStore((state) => state.token)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const isLoading = useAuthStore((state) => state.isLoading)

  return { user, token, isAuthenticated, isLoading }
}

/**
 * Hook to validate token and fetch current user
 * Used by AuthProvider to validate token on app load
 */
export function useValidateToken(token: string | null, enabled: boolean = true) {
  const setUser = useAuthStore((state) => state.setUser)
  const setLoading = useAuthStore((state) => state.setLoading)
  const logout = useAuthStore((state) => state.logout)

  const query = useQuery<User>({
    queryKey: authKeys.currentUser(),
    queryFn: () => {
      if (!token) {
        throw new Error('No token available')
      }
      return authAPI.getCurrentUser(token)
    },
    enabled: enabled && !!token,
    retry: false,
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  // Handle success and error states separately (TanStack Query v5 pattern)
  useEffect(() => {
    if (query.isSuccess && query.data) {
      setUser(query.data)
      setLoading(false)
    }
  }, [query.isSuccess, query.data, setUser, setLoading])

  useEffect(() => {
    if (query.isError) {
      // Token is invalid, clear auth state
      logout()
    }
  }, [query.isError, logout])

  return query
}

/**
 * Hook for change password functionality
 */
export function useChangePassword() {
  const token = useAuthStore((state) => state.token)
  const logout = useLogout()

  return useMutation({
    mutationFn: ({
      oldPassword,
      newPassword,
    }: {
      oldPassword: string
      newPassword: string
    }) => {
      if (!token) {
        throw new Error('No authentication token')
      }
      return authAPI.changePassword(token, oldPassword, newPassword)
    },
    onSuccess: () => {
      // Logout user after password change (they'll need to login with new password)
      logout()
    },
  })
}

/**
 * Hook to check if setup is required
 * Returns true if no users exist in the system
 */
export function useSetupRequired() {
  return useQuery({
    queryKey: authKeys.setupRequired(),
    queryFn: () => authAPI.checkSetupRequired(),
    staleTime: Infinity, // Only check once
    retry: false,
  })
}
