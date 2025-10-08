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

export class WebSocketService {
  private ws: WebSocket | null = null
  private handlers = new Map<string, Set<MessageHandler>>()
  private subscribedChannels = new Set<string>()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000

  constructor(private url: string) {}

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return
    }

    this.ws = new WebSocket(this.url)

    this.ws.onopen = () => {
      console.log('WebSocket connected')
      this.reconnectAttempts = 0

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
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')

      // Attempt to reconnect
      if (this.reconnectAttempts < this.maxReconnectAttempts) {
        this.reconnectAttempts++
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
        console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

        setTimeout(() => this.connect(), delay)
      }
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.handlers.clear()
    this.subscribedChannels.clear()
  }

  subscribe(channels: string[]) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.warn('WebSocket not connected, queuing subscription')
      channels.forEach((ch) => this.subscribedChannels.add(ch))
      return
    }

    const message: SubscribeMessage = {
      action: 'subscribe',
      channels,
    }

    this.ws.send(JSON.stringify(message))
    channels.forEach((ch) => this.subscribedChannels.add(ch))
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
}

// Create singleton instance
const wsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws'
export const wsService = new WebSocketService(wsUrl)
