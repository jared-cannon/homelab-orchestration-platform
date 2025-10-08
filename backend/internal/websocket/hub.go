package websocket

import (
	"encoding/json"
	"log"
	"sync"

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
}

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected (total: %d)", len(h.clients))

		case message := <-h.broadcast:
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				// Only send to clients subscribed to this channel
				client.mu.RLock()
				subscribed := client.channels[message.Channel]
				client.mu.RUnlock()

				if subscribed {
					select {
					case client.send <- data:
					default:
						// Client buffer full, close connection
						h.mu.RUnlock()
						h.mu.Lock()
						close(client.send)
						delete(h.clients, client)
						h.mu.Unlock()
						h.mu.RLock()
					}
				}
			}
			h.mu.RUnlock()
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

// readPump handles incoming messages from clients
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
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
	defer func() {
		c.conn.Close()
	}()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
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
	}
}

// Start begins processing for a client
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
	c.hub.register <- c
}
