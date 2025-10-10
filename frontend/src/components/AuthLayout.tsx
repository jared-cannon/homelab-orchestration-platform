import { Sidebar } from './Sidebar'

interface AuthLayoutProps {
  children: React.ReactNode
}

/**
 * AuthLayout provides a common layout for authenticated pages
 * Includes a collapsible sidebar with navigation
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="min-h-screen flex">
      {/* Sidebar Navigation */}
      <Sidebar />

      {/* Main Content */}
      <main className="flex-1 overflow-auto pt-14 md:pt-0">{children}</main>
    </div>
  )
}
