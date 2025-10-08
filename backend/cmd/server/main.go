package main

import (
	// Note: embed will be configured during production build (make build)
	// Development uses Vite dev server at :5173
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/api"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/services"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// embedFrontend will be configured in production build
// var embedFrontend embed.FS

// initDB initializes the database connection and runs migrations
func initDB() (*gorm.DB, error) {
	// Get database path from environment or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./homelab.db"
	}

	// Open database connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Run auto-migrations
	err = db.AutoMigrate(
		&models.Device{},
		&models.Application{},
		&models.Deployment{},
		&models.Credential{},
	)
	if err != nil {
		return nil, err
	}

	log.Printf("📦 Database initialized at %s", dbPath)
	return db, nil
}

func main() {
	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get DB connection for health checks
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}
	defer sqlDB.Close()

	// Initialize services
	credService, err := services.NewCredentialService()
	if err != nil {
		log.Fatalf("Failed to initialize credential service: %v", err)
	}
	log.Printf("🔐 Credential service initialized")

	sshClient := ssh.NewClient()
	credMatcher := services.NewCredentialMatcher(db, credService, sshClient)
	deviceService := services.NewDeviceService(db, credService, sshClient)
	scannerService := services.NewScannerService(db, sshClient, credMatcher)
	log.Printf("🔧 Services initialized")

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Printf("🔌 WebSocket hub initialized")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Homelab Orchestration Platform",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// API routes
	apiGroup := app.Group("/api/v1")

	// Health check endpoint
	apiGroup.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "homelab-orchestration-platform",
			"version": "0.1.0",
		})
	})

	// Register device routes
	deviceHandler := api.NewDeviceHandler(deviceService)
	deviceHandler.RegisterRoutes(apiGroup)

	// Register scanner routes
	scannerHandler := api.NewScannerHandler(scannerService)
	scannerHandler.RegisterRoutes(apiGroup)

	// Register WebSocket routes
	wsHandler := api.NewWebSocketHandler(wsHub)
	wsHandler.RegisterRoutes(app)

	// Development mode - frontend is served by Vite at :5173
	// Production mode will have embedded frontend (configured during build)
	app.Get("/", func(c *fiber.Ctx) error {
		if os.Getenv("ENV") == "production" {
			return c.SendString("Production mode - frontend will be embedded here")
		}
		return c.SendString("Development mode - frontend at http://localhost:5173")
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if os.Getenv("ENV") == "production" {
		log.Printf("📦 Serving embedded frontend")
	} else {
		log.Printf("🔧 Development mode - frontend at http://localhost:5173")
	}

	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
