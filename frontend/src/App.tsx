import { Toaster } from 'sonner'
import { DevicesPage } from './pages/Devices'
import { useWebSocket } from './hooks/useWebSocket'

function App() {
  // Establish WebSocket connection on app load
  useWebSocket()

  return (
    <>
      <DevicesPage />
      <Toaster position="top-right" richColors />
    </>
  )
}

export default App
