type WebSocketMessage = {
  channel: string
  event: string
  data: unknown
}

type SubscribeMessage = {
  action: 'subscribe' | 'unsubscribe'
  channels: string[]
}

type MessageHandler = (event: string, data: unknown) => void

type ConnectionStatus = 'connected' | 'disconnected' | 'reconnecting' | 'error'
type StatusChangeHandler = (status: ConnectionStatus, details?: string) => void

// Debug logging - enable with VITE_DEBUG_WS=true
const DEBUG = import.meta.env.VITE_DEBUG_WS === 'true'
const debugLog = (...args: unknown[]) => {
  if (DEBUG) {
    console.log('[WebSocket]', ...args)
  }
}

export class WebSocketService {
  private ws: WebSocket | null = null
  private handlers = new Map<string, Set<MessageHandler>>()
  private subscribedChannels = new Set<string>()
  private reconnectAttempts = 0
  private initialReconnectDelay = 1000 // 1 second
  private maxReconnectDelay = 30000 // 30 seconds
  private isConnecting = false
  private shouldReconnect = true
  private statusChangeHandlers = new Set<StatusChangeHandler>()
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null

  constructor(private url: string) {}

  connect() {
    // Prevent duplicate connection attempts
    if (this.isConnecting) {
      debugLog('Already connecting, skipping')
      return
    }

    if (this.ws?.readyState === WebSocket.OPEN) {
      debugLog('Already connected, skipping')
      return
    }

    if (this.ws?.readyState === WebSocket.CONNECTING) {
      debugLog('Connection in progress, skipping')
      return
    }

    // Clear any pending reconnection timeout when explicitly connecting
    if (this.reconnectTimeout) {
      debugLog('Clearing pending reconnection timeout')
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }

    // Clean up any existing closed connection
    if (this.ws?.readyState === WebSocket.CLOSED) {
      debugLog('Cleaning up closed connection')
      this.ws = null
    }

    this.isConnecting = true
    this.shouldReconnect = true
    debugLog('Creating new WebSocket connection to', this.url)
    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      debugLog('Connected')
      this.isConnecting = false
      this.reconnectAttempts = 0
      this.notifyStatusChange('connected', 'Connection established')

      // Resubscribe to all channels after reconnection
      if (this.subscribedChannels.size > 0) {
        this.subscribe([...this.subscribedChannels])
      }
    }

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        const handlers = this.handlers.get(message.channel)

        if (handlers) {
          handlers.forEach((handler) => handler(message.event, message.data))
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      this.isConnecting = false
      this.notifyStatusChange('error', 'Connection error occurred')
    }

    this.ws.onclose = (event) => {
      this.isConnecting = false
      debugLog('Disconnected - Code:', event.code, 'Reason:', event.reason, 'Clean:', event.wasClean)

      // Only attempt to reconnect if we should (i.e., not manually disconnected)
      if (!this.shouldReconnect) {
        this.notifyStatusChange('disconnected', 'Manually disconnected')
        return
      }

      // Prevent immediate reconnection loop - wait at least 1 second
      this.reconnectAttempts++

      // Calculate delay with exponential backoff, capped at maxReconnectDelay
      const exponentialDelay = this.initialReconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
      const delay = Math.min(exponentialDelay, this.maxReconnectDelay)

      const delaySeconds = (delay / 1000).toFixed(1)
      debugLog(`Reconnecting in ${delaySeconds}s (attempt ${this.reconnectAttempts})`)
      this.notifyStatusChange('reconnecting', `Reconnecting in ${delaySeconds}s (attempt ${this.reconnectAttempts})`)

      // Clear any existing reconnect timeout
      if (this.reconnectTimeout) {
        clearTimeout(this.reconnectTimeout)
      }

      this.reconnectTimeout = setTimeout(() => {
        this.reconnectTimeout = null
        this.connect()
      }, delay)
    }
  }

  disconnect() {
    this.shouldReconnect = false
    this.isConnecting = false

    // Clear reconnect timeout
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }

    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.handlers.clear()
    this.subscribedChannels.clear()
  }

  subscribe(channels: string[]) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      debugLog('Not connected, queuing subscription for channels:', channels)
      channels.forEach((ch) => this.subscribedChannels.add(ch))
      return
    }

    const message: SubscribeMessage = {
      action: 'subscribe',
      channels,
    }

    this.ws.send(JSON.stringify(message))
    channels.forEach((ch) => this.subscribedChannels.add(ch))
    debugLog('Subscribed to channels:', channels)
  }

  unsubscribe(channels: string[]) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return
    }

    const message: SubscribeMessage = {
      action: 'unsubscribe',
      channels,
    }

    this.ws.send(JSON.stringify(message))
    channels.forEach((ch) => this.subscribedChannels.delete(ch))
    debugLog('Unsubscribed from channels:', channels)
  }

  on(channel: string, handler: MessageHandler) {
    if (!this.handlers.has(channel)) {
      this.handlers.set(channel, new Set())
    }

    this.handlers.get(channel)!.add(handler)

    // Auto-subscribe to the channel
    if (!this.subscribedChannels.has(channel)) {
      this.subscribe([channel])
    }

    // Return cleanup function
    return () => this.off(channel, handler)
  }

  off(channel: string, handler: MessageHandler) {
    const handlers = this.handlers.get(channel)
    if (handlers) {
      handlers.delete(handler)

      // If no more handlers for this channel, unsubscribe
      if (handlers.size === 0) {
        this.handlers.delete(channel)
        this.unsubscribe([channel])
      }
    }
  }

  /**
   * Register a callback for WebSocket connection status changes
   * @param handler - Function to call when status changes
   * @returns Cleanup function to remove the handler
   */
  onStatusChange(handler: StatusChangeHandler) {
    this.statusChangeHandlers.add(handler)

    // Return cleanup function
    return () => this.statusChangeHandlers.delete(handler)
  }

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus {
    if (!this.ws) return 'disconnected'
    if (this.ws.readyState === WebSocket.OPEN) return 'connected'
    if (this.ws.readyState === WebSocket.CONNECTING) return 'reconnecting'
    return 'disconnected'
  }

  /**
   * Notify all registered handlers of a status change
   */
  private notifyStatusChange(status: ConnectionStatus, details?: string) {
    this.statusChangeHandlers.forEach((handler) => handler(status, details))
  }
}

// Create singleton instance with HMR support
const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'

// Preserve singleton across HMR updates in development
let wsService: WebSocketService

if (import.meta.hot) {
  // In development with HMR
  if (!import.meta.hot.data.wsService) {
    // First time - create new instance
    import.meta.hot.data.wsService = new WebSocketService(wsUrl)
    debugLog('Created new WebSocket service instance (HMR-aware)')
  } else {
    debugLog('Reusing existing WebSocket service instance from HMR')
  }
  wsService = import.meta.hot.data.wsService

  // Clean up on module disposal (before reload)
  import.meta.hot.dispose((data) => {
    debugLog('HMR: Preserving WebSocket service for next reload')
    data.wsService = wsService
    // Don't disconnect - keep the connection alive across HMR
  })
} else {
  // Production - simple singleton
  wsService = new WebSocketService(wsUrl)
}

export { wsService }
