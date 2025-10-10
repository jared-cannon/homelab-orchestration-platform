import { useEffect } from 'react'
import { useAuthStore } from '../stores/authStore'
import { useValidateToken } from '../hooks/useAuth'

interface AuthProviderProps {
  children: React.ReactNode
}

/**
 * AuthProvider initializes authentication on app load
 * - Reads token from localStorage
 * - Validates token with backend
 * - Sets user in auth store if token is valid
 * - Clears auth state if token is invalid
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const initializeAuth = useAuthStore((state) => state.initializeAuth)
  const token = useAuthStore((state) => state.token)
  const isLoading = useAuthStore((state) => state.isLoading)

  // Initialize auth on mount (reads token from localStorage)
  useEffect(() => {
    initializeAuth()
  }, [initializeAuth])

  // Validate token if it exists
  useValidateToken(token, !!token)

  // Don't render children until auth is initialized
  if (isLoading && token) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Initializing...</div>
      </div>
    )
  }

  return <>{children}</>
}
