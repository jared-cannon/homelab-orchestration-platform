package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

// Message represents a WebSocket message
type Message struct {
	Channel string      `json:"channel"`
	Event   string      `json:"event"`
	Data    interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	channels map[string]bool
	mu       sync.RWMutex
	closed   bool      // Track if send channel is closed
	done     chan bool // Signal when client is finished
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients      map[*Client]bool
	broadcast    chan *Message
	register     chan *Client
	unregister   chan *Client
	shutdownChan chan struct{}
	mu           sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan *Message, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		shutdownChan: make(chan struct{}),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case <-h.shutdownChan:
			log.Printf("[WebSocket] Hub shutting down")
			// Close all client connections
			h.mu.Lock()
			for client := range h.clients {
				client.mu.Lock()
				if !client.closed {
					close(client.send)
					client.closed = true
				}
				client.mu.Unlock()
				client.conn.Close()
			}
			h.clients = make(map[*Client]bool)
			h.mu.Unlock()
			log.Printf("[WebSocket] Hub shutdown complete")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			// Only log in production or when debugging
			// log.Printf("WebSocket client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				// Safe close - check if already closed
				client.mu.Lock()
				if !client.closed {
					close(client.send)
					client.closed = true
				}
				client.mu.Unlock()
			}
			h.mu.Unlock()
			// Only log in production or when debugging
			// log.Printf("WebSocket client disconnected (total: %d)", len(h.clients))

		case message := <-h.broadcast:
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			h.mu.RLock()
			// Collect clients to remove to avoid modifying map during iteration
			var clientsToRemove []*Client

			for client := range h.clients {
				// Only send to clients subscribed to this channel
				client.mu.RLock()
				subscribed := client.channels[message.Channel]
				client.mu.RUnlock()

				if subscribed {
					select {
					case client.send <- data:
						// Successfully sent
					default:
						// Client buffer full, mark for removal
						clientsToRemove = append(clientsToRemove, client)
					}
				}
			}
			h.mu.RUnlock()

			// Remove slow clients after iteration
			if len(clientsToRemove) > 0 {
				h.mu.Lock()
				for _, client := range clientsToRemove {
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						// Safe close - check if already closed
						client.mu.Lock()
						if !client.closed {
							close(client.send)
							client.closed = true
						}
						client.mu.Unlock()
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

// Broadcast sends a message to all subscribed clients
func (h *Hub) Broadcast(channel string, event string, data interface{}) {
	message := &Message{
		Channel: channel,
		Event:   event,
		Data:    data,
	}
	h.broadcast <- message
}

// Shutdown gracefully shuts down the WebSocket hub
func (h *Hub) Shutdown() {
	close(h.shutdownChan)
}

// readPump handles incoming messages from clients
func (c *Client) readPump() {
	defer func() {
		log.Printf("[WebSocket] readPump exiting, unregistering client")
		c.hub.unregister <- c
		c.conn.Close()
		close(c.done) // Signal that client is finished
	}()

	log.Printf("[WebSocket] readPump started for new client")

	// Set read deadline and configure pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("[WebSocket] readPump error: %v", err)
			break
		}

		// Handle subscription messages
		var msg struct {
			Action   string   `json:"action"`
			Channels []string `json:"channels"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Action == "subscribe" {
			c.mu.Lock()
			for _, channel := range msg.Channels {
				c.channels[channel] = true
			}
			c.mu.Unlock()
		} else if msg.Action == "unsubscribe" {
			c.mu.Lock()
			for _, channel := range msg.Channels {
				delete(c.channels, channel)
			}
			c.mu.Unlock()
		}
	}
}

// writePump handles outgoing messages to clients
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping every 30 seconds
	defer func() {
		log.Printf("[WebSocket] writePump exiting")
		ticker.Stop()
		c.conn.Close()
	}()

	log.Printf("[WebSocket] writePump started for new client")

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				log.Printf("[WebSocket] send channel closed, sending close message")
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WebSocket] error writing text message: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WebSocket] error writing ping message: %v", err)
				return
			}
		}
	}
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		channels: make(map[string]bool),
		done:     make(chan bool),
	}
}

// Start begins processing for a client
func (c *Client) Start() {
	log.Printf("[WebSocket] Starting new client")
	go c.writePump()
	go c.readPump()
	c.hub.register <- c
	log.Printf("[WebSocket] Client registered with hub")
}

// Wait blocks until the client connection is closed
func (c *Client) Wait() {
	<-c.done
	log.Printf("[WebSocket] Client connection finished")
}
