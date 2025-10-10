package main

import (
	// Note: embed will be configured during production build (make build)
	// Development uses Vite dev server at :5173
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/api"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/middleware"
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
		&models.User{},
		&models.Device{},
		&models.Application{},
		&models.Deployment{},
		&models.Credential{},
		&models.InstalledSoftware{},
		&models.NFSExport{},
		&models.NFSMount{},
		&models.Volume{},
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
	userService := services.NewUserService(db)

	// Initialize software and infrastructure services
	softwareService := services.NewSoftwareService(db, sshClient)
	nfsService := services.NewNFSService(db, sshClient, softwareService)
	volumeService := services.NewVolumeService(db, sshClient, softwareService)

	// Initialize health check service
	healthCheckService := services.NewHealthCheckService(db, sshClient, credService)
	healthCheckService.SetDeviceService(deviceService)
	log.Printf("🔧 Services initialized")

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Printf("🔌 WebSocket hub initialized")

	// Start health check service
	healthCtx := context.Background()
	healthCheckService.Start(healthCtx)
	log.Printf("🏥 Health check service started")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Homelab Orchestration Platform",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// CORS configuration - restrict to known origins in production
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		// Default to localhost for development
		allowedOrigins = "http://localhost:5173,http://localhost:3000"
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, PATCH, OPTIONS",
	}))

	// API routes
	apiGroup := app.Group("/api/v1")

	// Public endpoints (no authentication required)
	// Health check endpoint
	apiGroup.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "homelab-orchestration-platform",
			"version": "0.1.0",
		})
	})

	// Register auth handler (public routes - no authentication required)
	authHandler := api.NewAuthHandler(userService)
	authHandler.RegisterRoutes(apiGroup)

	// Auth-specific protected routes (ALWAYS require authentication)
	authProtectedGroup := apiGroup.Group("/auth")
	authProtectedGroup.Use(middleware.AuthMiddleware())
	authProtectedGroup.Get("/me", authHandler.GetCurrentUser)
	authProtectedGroup.Post("/change-password", authHandler.ChangePassword)

	// Protected endpoints (authentication required based on REQUIRE_AUTH env)
	protectedGroup := apiGroup.Group("")
	if os.Getenv("REQUIRE_AUTH") == "true" {
		protectedGroup.Use(middleware.AuthMiddleware())
		log.Printf("🔒 Authentication middleware ENABLED - all API endpoints require valid JWT token")
	} else {
		log.Printf("⚠️  Authentication middleware DISABLED - device/scanner endpoints are publicly accessible")
	}

	// Register device routes
	deviceHandler := api.NewDeviceHandler(deviceService)
	deviceHandler.RegisterRoutes(protectedGroup)

	// Register scanner routes
	scannerHandler := api.NewScannerHandler(scannerService)
	scannerHandler.RegisterRoutes(protectedGroup)

	// Register software, NFS, and volume handlers
	softwareHandler := api.NewSoftwareHandler(softwareService)
	nfsHandler := api.NewNFSHandler(nfsService)
	volumeHandler := api.NewVolumeHandler(volumeService)

	// Register nested routes under devices
	devices := protectedGroup.Group("/devices/:id")

	// Software management routes
	devices.Get("/software", softwareHandler.ListInstalled)
	devices.Post("/software", softwareHandler.InstallSoftware)
	devices.Post("/software/detect", softwareHandler.DetectInstalled)
	devices.Delete("/software/:name", softwareHandler.UninstallSoftware)

	// NFS server routes
	devices.Post("/nfs/server/setup", nfsHandler.SetupServer)
	devices.Get("/nfs/exports", nfsHandler.ListExports)
	devices.Post("/nfs/exports", nfsHandler.CreateExport)
	devices.Delete("/nfs/exports/:exportId", nfsHandler.RemoveExport)

	// NFS client routes
	devices.Get("/nfs/mounts", nfsHandler.ListMounts)
	devices.Post("/nfs/mounts", nfsHandler.MountShare)
	devices.Delete("/nfs/mounts/:mountId", nfsHandler.UnmountShare)

	// Volume management routes
	devices.Get("/volumes", volumeHandler.ListVolumes)
	devices.Post("/volumes", volumeHandler.CreateVolume)
	devices.Get("/volumes/:name", volumeHandler.GetVolume)
	devices.Get("/volumes/:name/inspect", volumeHandler.InspectVolume)
	devices.Delete("/volumes/:name", volumeHandler.RemoveVolume)

	// Register WebSocket routes (websocket auth is handled separately)
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

	// Start server in a goroutine for graceful shutdown
	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("🛑 Received signal %v, initiating graceful shutdown...", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown services in order
	log.Printf("🏥 Shutting down health check service...")
	healthCheckService.Stop()

	log.Printf("📡 Shutting down WebSocket hub...")
	wsHub.Shutdown()

	log.Printf("🔍 Shutting down scanner service...")
	scannerService.Shutdown()

	log.Printf("🔐 Closing SSH connections...")
	sshClient.Shutdown()

	log.Printf("🌐 Shutting down HTTP server...")
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("Error during HTTP server shutdown: %v", err)
	}

	log.Printf("💾 Closing database connection...")
	if err := sqlDB.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Printf("✅ Graceful shutdown complete")
}
