package api

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	ws "github.com/jaredcannon/homelab-orchestration-platform/internal/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *ws.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// HandleConnection handles WebSocket connections
func (h *WebSocketHandler) HandleConnection(c *websocket.Conn) {
	log.Printf("[WebSocket] New connection from %s", c.RemoteAddr())
	client := ws.NewClient(h.hub, c)
	client.Start()
	log.Printf("[WebSocket] Client started, waiting for connection to close...")

	// Block until the client connection is closed
	// This prevents Fiber from closing the connection prematurely
	client.Wait()

	log.Printf("[WebSocket] Connection closed for %s", c.RemoteAddr())
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(app *fiber.App) {
	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket endpoint
	app.Get("/ws", websocket.New(h.HandleConnection))
}
