import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'sonner'
import { DevicesPage } from './pages/Devices'
import { DeviceDetailPage } from './pages/DeviceDetail'
import { LoginPage } from './pages/Login'
import { SetupPage } from './pages/Setup'
import { AuthProvider } from './components/AuthProvider'
import { ProtectedRoute } from './components/ProtectedRoute'
import { AuthLayout } from './components/AuthLayout'
import { useWebSocket } from './hooks/useWebSocket'

function AppContent() {
  return (
    <Routes>
      {/* Public Routes */}
      <Route path="/login" element={<LoginPage />} />
      <Route path="/setup" element={<SetupPage />} />

      {/* Protected Routes */}
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AuthLayout>
              <DevicesPage />
            </AuthLayout>
          </ProtectedRoute>
        }
      />
      <Route
        path="/devices/:id"
        element={
          <ProtectedRoute>
            <AuthLayout>
              <DeviceDetailPage />
            </AuthLayout>
          </ProtectedRoute>
        }
      />

      {/* Redirect unknown routes to home */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

function App() {
  // Establish WebSocket connection at app level (outside router)
  // This prevents reconnections on route changes
  useWebSocket()

  return (
    <BrowserRouter>
      <AuthProvider>
        <AppContent />
        <Toaster position="top-right" richColors />
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
