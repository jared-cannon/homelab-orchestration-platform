import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import {
  Server,
  Rocket,
  ShoppingBag,
  ChevronLeft,
  LogOut,
  User,
  Menu,
  X
} from 'lucide-react'
import { useCurrentUser, useLogout } from '../hooks/useAuth'

interface NavItem {
  name: string
  path: string
  icon: React.ReactNode
}

const navItems: NavItem[] = [
  {
    name: 'Devices',
    path: '/',
    icon: <Server className="w-5 h-5" />,
  },
  {
    name: 'Apps',
    path: '/apps',
    icon: <Rocket className="w-5 h-5" />,
  },
  {
    name: 'Marketplace',
    path: '/marketplace',
    icon: <ShoppingBag className="w-5 h-5" />,
  },
]

export function Sidebar() {
  const location = useLocation()
  const { user } = useCurrentUser()
  const logout = useLogout()
  const [isCollapsed, setIsCollapsed] = useState(() => {
    try {
      const saved = localStorage.getItem('sidebar-collapsed')
      return saved ? JSON.parse(saved) : false
    } catch {
      return false
    }
  })
  const [isMobileOpen, setIsMobileOpen] = useState(false)

  useEffect(() => {
    localStorage.setItem('sidebar-collapsed', JSON.stringify(isCollapsed))
  }, [isCollapsed])

  // Handle escape key to close mobile drawer
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isMobileOpen) {
        setIsMobileOpen(false)
      }
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [isMobileOpen])

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/' || location.pathname.startsWith('/devices')
    }
    // Match exact path or path with slash (sub-routes)
    return location.pathname === path || location.pathname.startsWith(path + '/')
  }

  const toggleCollapse = () => {
    setIsCollapsed(!isCollapsed)
  }

  const closeMobile = () => {
    setIsMobileOpen(false)
  }

  const SidebarContent = ({ forceExpanded = false }: { forceExpanded?: boolean }) => {
    const collapsed = forceExpanded ? false : isCollapsed

    return (
      <>
        {/* Logo/Brand */}
        <div className="flex items-center gap-3 px-4 py-6 border-b border-border">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center flex-shrink-0">
            <Server className="w-5 h-5 text-primary-foreground" />
          </div>
          {!collapsed && (
            <div className="flex-1 min-w-0">
              <h1 className="text-sm font-semibold truncate">Homelab</h1>
              <p className="text-xs text-muted-foreground truncate">Orchestration</p>
            </div>
          )}
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto">
          {navItems.map((item) => {
            const active = isActive(item.path)
            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={closeMobile}
                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg transition-colors ${
                  active
                    ? 'bg-primary/10 text-primary font-medium'
                    : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                }`}
                title={collapsed ? item.name : undefined}
              >
                <span className="flex-shrink-0">{item.icon}</span>
                {!collapsed && <span className="text-sm">{item.name}</span>}
              </Link>
            )
          })}
        </nav>

        {/* User Section */}
        <div className="border-t border-border p-2">
          {user && (
            <div
              className={`flex items-center gap-3 px-3 py-2.5 rounded-lg ${
                collapsed ? 'justify-center' : ''
              }`}
            >
              <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                <User className="w-4 h-4 text-primary" />
              </div>
              {!collapsed && (
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{user.username}</p>
                  {user.is_admin && (
                    <p className="text-xs text-muted-foreground truncate">Administrator</p>
                  )}
                </div>
              )}
            </div>
          )}
          <button
            onClick={logout}
            className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors ${
              collapsed ? 'justify-center' : ''
            }`}
            title={collapsed ? 'Logout' : undefined}
          >
            <LogOut className="w-5 h-5 flex-shrink-0" />
            {!collapsed && <span className="text-sm">Logout</span>}
          </button>
        </div>

        {/* Collapse Toggle (Desktop) */}
        <div className="hidden md:block border-t border-border p-2">
          <button
            onClick={toggleCollapse}
            className="w-full flex items-center justify-center p-2.5 rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            <ChevronLeft
              className={`w-5 h-5 transition-transform ${
                isCollapsed ? 'rotate-180' : ''
              }`}
            />
          </button>
        </div>
      </>
    )
  }

  return (
    <>
      {/* Mobile Top Navbar */}
      <div className="md:hidden fixed top-0 left-0 right-0 z-30 bg-card border-b border-border">
        <div className="flex items-center justify-between px-4 h-14">
          <button
            onClick={() => setIsMobileOpen(true)}
            className="p-2 -ml-2 rounded-lg hover:bg-muted transition-colors"
            aria-label="Open menu"
          >
            <Menu className="w-5 h-5" />
          </button>

          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-primary flex items-center justify-center">
              <Server className="w-4 h-4 text-primary-foreground" />
            </div>
            <h1 className="text-sm font-semibold">Homelab</h1>
          </div>

          {user && (
            <div className="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center">
              <User className="w-4 h-4 text-primary" />
            </div>
          )}
        </div>
      </div>

      {/* Mobile Overlay */}
      {isMobileOpen && (
        <div
          className="md:hidden fixed inset-0 bg-black/50 z-40"
          onClick={closeMobile}
        />
      )}

      {/* Mobile Sidebar */}
      <aside
        className={`md:hidden fixed top-0 left-0 bottom-0 z-50 w-64 bg-card border-r border-border flex flex-col transition-transform ${
          isMobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        {/* Close button */}
        <button
          onClick={closeMobile}
          className="absolute top-4 right-4 p-2 rounded-lg hover:bg-muted"
          aria-label="Close menu"
        >
          <X className="w-5 h-5" />
        </button>
        <SidebarContent forceExpanded={true} />
      </aside>

      {/* Desktop Sidebar */}
      <aside
        className={`hidden md:flex flex-col bg-card border-r border-border transition-all duration-300 ${
          isCollapsed ? 'w-20' : 'w-64'
        }`}
      >
        <SidebarContent forceExpanded={false} />
      </aside>
    </>
  )
}
