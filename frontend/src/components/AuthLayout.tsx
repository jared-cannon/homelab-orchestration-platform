import { useCurrentUser, useLogout } from '../hooks/useAuth'

interface AuthLayoutProps {
  children: React.ReactNode
}

/**
 * AuthLayout provides a common layout for authenticated pages
 * Includes a top navigation bar with user info and logout button
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  const { user } = useCurrentUser()
  const logout = useLogout()

  return (
    <div className="min-h-screen">
      {/* Top Navigation */}
      <nav className="bg-card border-b border-border">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            {/* Logo/Title */}
            <div className="flex items-center">
              <h1 className="text-xl font-semibold">
                Homelab Orchestration Platform
              </h1>
            </div>

            {/* User Menu */}
            <div className="flex items-center space-x-4">
              {user && (
                <div className="flex items-center space-x-3">
                  <div className="text-sm">
                    <div className="font-medium">{user.username}</div>
                    {user.is_admin && (
                      <div className="text-xs text-muted-foreground">
                        Administrator
                      </div>
                    )}
                  </div>
                  <button
                    onClick={logout}
                    className="px-3 py-1.5 text-sm bg-muted hover:bg-muted/80 border border-border rounded-md transition-colors"
                  >
                    Logout
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main>{children}</main>
    </div>
  )
}
