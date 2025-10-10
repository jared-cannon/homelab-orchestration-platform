import { useEffect, useState } from 'react'
import { wsService } from '../services/websocket'

export function useWebSocket() {
  const [isConnected, setIsConnected] = useState(false)

  useEffect(() => {
    // Track actual connection status from the service
    const unsubscribe = wsService.onStatusChange((status) => {
      setIsConnected(status === 'connected')
    })

    // Set initial state based on current connection
    setIsConnected(wsService.getStatus() === 'connected')

    // Connect only if not already connected or connecting
    // The singleton service handles duplicate connection attempts
    if (wsService.getStatus() === 'disconnected') {
      wsService.connect()
    }

    // Cleanup: Don't disconnect on unmount - keep connection alive for other components
    return () => {
      unsubscribe()
      // Connection stays alive even when this component unmounts
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
