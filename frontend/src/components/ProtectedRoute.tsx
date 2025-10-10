import { Navigate } from 'react-router-dom'
import { useCurrentUser } from '../hooks/useAuth'

interface ProtectedRouteProps {
  children: React.ReactNode
}

/**
 * ProtectedRoute component ensures the user is authenticated before
 * rendering the children. If not authenticated, redirects to login.
 */
export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const { isAuthenticated, isLoading } = useCurrentUser()

  // Show loading state while checking authentication
  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-lg">Loading...</div>
      </div>
    )
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  // Render children if authenticated
  return <>{children}</>
}
