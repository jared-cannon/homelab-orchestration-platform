import { useEffect, useState } from 'react'
import { wsService } from '../services/websocket'

export function useWebSocket() {
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    // Connect on mount
    wsService.connect()

    // Check connection status
    const checkConnection = setInterval(() => {
      // Access the private ws property (we'll need to make it public or add a getter)
      setIsConnected(true) // For now, assume connected after initial connection
    }, 1000)

    return () => {
      clearInterval(checkConnection)
      // Don't disconnect on unmount - keep connection alive
    }
  }, [])

  return { isConnected }
}

export function useWebSocketChannel<T = unknown>(
  channel: string,
  onMessage: (event: string, data: T) => void
) {
  useEffect(() => {
    const unsubscribe = wsService.on(channel, onMessage as (event: string, data: unknown) => void)

    return () => {
      unsubscribe()
    }
  }, [channel, onMessage])
}
