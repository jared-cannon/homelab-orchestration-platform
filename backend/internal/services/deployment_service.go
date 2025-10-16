package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DeploymentService handles deployment operations
type DeploymentService struct {
	db                 *gorm.DB
	sshClient          *ssh.Client
	recipeLoader       RecipeProvider
	deviceService      *DeviceService
	wsHub              WSHub
	firewallService    *FirewallService
	deviceScorer       *DeviceScorer
	dbPoolManager      *DatabasePoolManager
	environmentBuilder *EnvironmentBuilder
	configValidator    *ConfigValidator
	deviceLocks        sync.Map // Map of device ID -> *sync.Mutex to prevent concurrent deployments
	cancelFuncs        sync.Map // Map of deployment ID -> context.CancelFunc for cancellation
}

// WSHub interface for WebSocket broadcasting
type WSHub interface {
	Broadcast(channel string, event string, data interface{})
}

// RecipeProvider interface for accessing recipes
type RecipeProvider interface {
	GetRecipe(slug string) (*models.Recipe, error)
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	db *gorm.DB,
	sshClient *ssh.Client,
	recipeLoader RecipeProvider,
	deviceService *DeviceService,
	credService *CredentialService,
	wsHub WSHub,
) *DeploymentService {
	dbPoolManager := NewDatabasePoolManager(db, sshClient, credService)

	return &DeploymentService{
		db:                 db,
		sshClient:          sshClient,
		recipeLoader:       recipeLoader,
		deviceService:      deviceService,
		wsHub:              wsHub,
		firewallService:    NewFirewallService(sshClient),
		deviceScorer:       NewDeviceScorer(db, sshClient),
		dbPoolManager:      dbPoolManager,
		environmentBuilder: NewEnvironmentBuilder(credService, dbPoolManager),
		configValidator:    NewConfigValidator(),
	}
}

// CreateDeploymentRequest represents a request to create a deployment
type CreateDeploymentRequest struct {
	RecipeSlug     string                 `json:"recipe_slug"`
	DeviceID       uuid.UUID              `json:"device_id,omitempty"`       // Optional - if not provided, will recommend
	AutoSelectDevice bool                   `json:"auto_select_device"`       // Auto-select best device
	Config         map[string]interface{} `json:"config"`
}

// DeviceRecommendation represents a recommended device for a recipe
type DeviceRecommendation struct {
	DeviceID       uuid.UUID `json:"device_id"`
	DeviceName     string    `json:"device_name"`
	DeviceIP       string    `json:"device_ip"`
	Score          int       `json:"score"`          // 0-100
	Recommendation string    `json:"recommendation"` // "best", "good", "acceptable", "not-recommended"
	Reasons        []string  `json:"reasons"`
	Available      bool      `json:"available"`
}

// RecommendDevicesForRecipe recommends devices for deploying a recipe using intelligent placement
func (s *DeploymentService) RecommendDevicesForRecipe(recipeSlug string) ([]DeviceRecommendation, error) {
	// Get the recipe
	recipe, err := s.recipeLoader.GetRecipe(recipeSlug)
	if err != nil {
		return nil, fmt.Errorf("recipe not found: %w", err)
	}

	// Convert recipe requirements to device scorer format
	requirements := RecipeRequirements{
		MinRAMMB:     s.parseMemoryRequirement(recipe.Requirements.Memory.Minimum),
		MinStorageGB: s.parseStorageRequirement(recipe.Requirements.Storage.Minimum),
		CPUCores:     recipe.Requirements.CPU.MinimumCores,
	}

	// Score all devices
	deviceScores, err := s.deviceScorer.ScoreDevicesForRecipe(requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to score devices: %w", err)
	}

	// Convert to DeviceRecommendation format
	recommendations := make([]DeviceRecommendation, len(deviceScores))
	for i, score := range deviceScores {
		recommendations[i] = DeviceRecommendation{
			DeviceID:       score.DeviceID,
			DeviceName:     score.DeviceName,
			DeviceIP:       score.DeviceIP,
			Score:          score.Score,
			Recommendation: score.Recommendation,
			Reasons:        score.Reasons,
			Available:      score.Available,
		}
	}

	return recommendations, nil
}

// parseMemoryRequirement converts memory string (e.g., "512MB", "1GB") to MB
func (s *DeploymentService) parseMemoryRequirement(mem string) int {
	// Simple parser for common formats
	mem = strings.ToUpper(strings.TrimSpace(mem))
	if strings.HasSuffix(mem, "GB") {
		val := strings.TrimSuffix(mem, "GB")
		if gb, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return gb * 1024
		}
	} else if strings.HasSuffix(mem, "MB") {
		val := strings.TrimSuffix(mem, "MB")
		if mb, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return mb
		}
	}
	return 512 // Default
}

// parseStorageRequirement converts storage string (e.g., "1GB", "100GB") to GB
func (s *DeploymentService) parseStorageRequirement(storage string) int {
	// Simple parser for common formats
	storage = strings.ToUpper(strings.TrimSpace(storage))
	if strings.HasSuffix(storage, "GB") {
		val := strings.TrimSuffix(storage, "GB")
		if gb, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return gb
		}
	}
	return 1 // Default
}

// CreateDeployment creates and deploys a new application
func (s *DeploymentService) CreateDeployment(req CreateDeploymentRequest) (*models.Deployment, error) {
	// Get the recipe
	recipe, err := s.recipeLoader.GetRecipe(req.RecipeSlug)
	if err != nil {
		return nil, fmt.Errorf("recipe not found: %w", err)
	}

	// Validate recipe manifest FIRST (before expensive device selection)
	if err := recipe.Validate(); err != nil {
		return nil, fmt.Errorf("recipe validation failed: %w", err)
	}

	// Validate user configuration against recipe requirements
	if err := s.configValidator.Validate(recipe, req.Config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Handle intelligent device selection
	var deviceID uuid.UUID
	if req.AutoSelectDevice || req.DeviceID == uuid.Nil {
		// Use intelligent placement to select best device
		log.Printf("[Deployment] Auto-selecting device for %s using intelligent placement", recipe.Name)
		recommendations, err := s.RecommendDevicesForRecipe(req.RecipeSlug)
		if err != nil {
			return nil, fmt.Errorf("failed to recommend devices: %w", err)
		}

		if len(recommendations) == 0 {
			return nil, fmt.Errorf("no available devices found")
		}

		// Find first available device
		var bestDevice *DeviceRecommendation
		for i := range recommendations {
			if recommendations[i].Available {
				bestDevice = &recommendations[i]
				break
			}
		}

		if bestDevice == nil {
			return nil, fmt.Errorf("no suitable devices found for %s", recipe.Name)
		}

		deviceID = bestDevice.DeviceID
		log.Printf("[Deployment] Selected device %s (score: %d) for %s", bestDevice.DeviceName, bestDevice.Score, recipe.Name)
	} else {
		deviceID = req.DeviceID
	}

	// Get the device (using intelligently selected deviceID or user-provided)
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	// Sanitize config: Remove sensitive data (passwords) before storing in database
	// We only need passwords during template rendering, not after deployment
	sanitizedConfig := s.sanitizeConfig(req.Config)

	// Create deployment record with sanitized config
	configJSON, err := json.Marshal(sanitizedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	deployment := &models.Deployment{
		RecipeSlug:     req.RecipeSlug,
		RecipeName:     recipe.Name,
		DeviceID:       deviceID, // Use intelligently selected or user-provided device
		Status:         models.DeploymentStatusValidating,
		Config:         configJSON,
		ComposeProject: s.generateProjectName(recipe.Slug),
	}

	// Save to database
	if err := s.db.Create(deployment).Error; err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Important: Store the ORIGINAL config (with passwords) for template rendering
	// But use a temporary copy - don't modify the deployment.Config field
	deployment.Config, _ = json.Marshal(req.Config)

	// Create cancellable context for this deployment
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFuncs.Store(deployment.ID.String(), cancel)

	// Start deployment process asynchronously
	go s.executeDeployment(ctx, deployment, recipe, device)

	return deployment, nil
}

// GetDeployment retrieves a deployment by ID
func (s *DeploymentService) GetDeployment(id string) (*models.Deployment, error) {
	deploymentID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %w", err)
	}

	var deployment models.Deployment
	if err := s.db.Preload("Device").First(&deployment, "id = ?", deploymentID).Error; err != nil {
		return nil, err
	}

	return &deployment, nil
}

// ListDeployments lists all deployments with optional filters
func (s *DeploymentService) ListDeployments(deviceID *uuid.UUID, status *models.DeploymentStatus) ([]models.Deployment, error) {
	var deployments []models.Deployment
	query := s.db.Preload("Device")

	if deviceID != nil {
		query = query.Where("device_id = ?", *deviceID)
	}
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if err := query.Order("created_at DESC").Find(&deployments).Error; err != nil {
		return nil, err
	}

	return deployments, nil
}

// DeleteDeployment stops and removes a deployment
func (s *DeploymentService) DeleteDeployment(id string) error {
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return err
	}

	// Get device for SSH
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Extract ports from deployment before stopping
	portsToClose := ExtractPortsFromCompose(deployment.GeneratedCompose)

	// Stop and remove containers
	if deployment.ComposeProject != "" {
		host := fmt.Sprintf("%s:22", device.IPAddress)
		deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

		// Stop and remove the compose project (WITHOUT removing volumes to preserve data)
		// Users should manually delete volumes if they want to remove data
		stopCmd := fmt.Sprintf("cd %s && docker compose -p %s down", deployDir, deployment.ComposeProject)
		output, err := s.sshClient.ExecuteWithTimeout(host, stopCmd, 2*time.Minute)
		if err != nil {
			return fmt.Errorf("failed to stop deployment: %w (output: %s)", err, output)
		}

		log.Printf("[Deployment] Stopped %s (volumes preserved)", deployment.ComposeProject)
	}

	// Clean up firewall ports (only if no other deployments are using them)
	if len(portsToClose) > 0 {
		if err := s.cleanupFirewallPorts(device, deployment.ID, portsToClose); err != nil {
			// Log warning but don't fail deletion if port cleanup fails
			log.Printf("[Deployment] Warning: Failed to cleanup firewall ports for %s: %v", deployment.ID, err)
		}
	}

	// Delete from database
	if err := s.db.Delete(deployment).Error; err != nil {
		return fmt.Errorf("failed to delete deployment record: %w", err)
	}

	return nil
}

// BulkDeleteDeployments deletes all deployments with a specific status
func (s *DeploymentService) BulkDeleteDeployments(status models.DeploymentStatus) (int, error) {
	// Query all deployments with the specified status
	var deployments []models.Deployment
	if err := s.db.Where("status = ?", status).Find(&deployments).Error; err != nil {
		return 0, fmt.Errorf("failed to query deployments: %w", err)
	}

	deletedCount := 0
	var errors []string

	// Delete each deployment
	for _, deployment := range deployments {
		if err := s.DeleteDeployment(deployment.ID.String()); err != nil {
			// Log error but continue with other deletions
			log.Printf("[Deployment] Failed to delete deployment %s: %v", deployment.ID, err)
			errors = append(errors, fmt.Sprintf("%s: %v", deployment.RecipeName, err))
		} else {
			deletedCount++
		}
	}

	// If some deletions failed, return error with details
	if len(errors) > 0 {
		return deletedCount, fmt.Errorf("deleted %d/%d deployments, errors: %s",
			deletedCount, len(deployments), strings.Join(errors, "; "))
	}

	log.Printf("[Deployment] Bulk deleted %d deployments with status: %s", deletedCount, status)
	return deletedCount, nil
}

// CancelDeployment cancels a running or pending deployment
func (s *DeploymentService) CancelDeployment(id string) error {
	// Get deployment to verify it exists
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Check if deployment is in a cancellable state
	// Only deployments that are still in progress can be cancelled
	cancellableStatuses := map[models.DeploymentStatus]bool{
		models.DeploymentStatusValidating:  true,
		models.DeploymentStatusPreparing:   true,
		models.DeploymentStatusDeploying:   true,
		models.DeploymentStatusConfiguring: true,
		models.DeploymentStatusHealthCheck: true,
	}

	if !cancellableStatuses[deployment.Status] {
		return fmt.Errorf("deployment cannot be cancelled (current status: %s)", deployment.Status)
	}

	// Look up the cancel function for this deployment
	cancelFunc, exists := s.cancelFuncs.Load(id)
	if !exists {
		// Deployment may have already completed or been cancelled
		return fmt.Errorf("deployment is not active (may have already completed)")
	}

	// Call the cancel function to stop the deployment goroutine
	if cancel, ok := cancelFunc.(context.CancelFunc); ok {
		log.Printf("[Deployment] Cancelling deployment %s", id)
		cancel()
	}

	return nil
}

// RestartDeployment restarts a running deployment without recreating containers
func (s *DeploymentService) RestartDeployment(id string) error {
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Check if deployment can be restarted
	if deployment.Status != models.DeploymentStatusRunning && deployment.Status != models.DeploymentStatusStopped {
		return fmt.Errorf("deployment cannot be restarted (current status: %s)", deployment.Status)
	}

	// Get device for SSH
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

	// Use appropriate command based on current status
	var restartCmd string
	if deployment.Status == models.DeploymentStatusStopped {
		// If stopped, use 'start' instead of 'restart'
		restartCmd = fmt.Sprintf("cd %s && docker compose -p %s start", deployDir, deployment.ComposeProject)
	} else {
		// If running, use 'restart'
		restartCmd = fmt.Sprintf("cd %s && docker compose -p %s restart", deployDir, deployment.ComposeProject)
	}

	output, err := s.sshClient.ExecuteWithTimeout(host, restartCmd, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to restart deployment: %w (output: %s)", err, output)
	}

	log.Printf("[Deployment] Restarted %s on %s", deployment.ComposeProject, device.Name)

	// Update status to running
	s.updateStatus(deployment, models.DeploymentStatusRunning, "")

	return nil
}

// StopDeployment stops a running deployment without removing containers
func (s *DeploymentService) StopDeployment(id string) error {
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Check if deployment can be stopped
	if deployment.Status != models.DeploymentStatusRunning {
		return fmt.Errorf("deployment cannot be stopped (current status: %s)", deployment.Status)
	}

	// Get device for SSH
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

	// Stop containers (keeps containers and volumes)
	stopCmd := fmt.Sprintf("cd %s && docker compose -p %s stop", deployDir, deployment.ComposeProject)
	output, err := s.sshClient.ExecuteWithTimeout(host, stopCmd, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to stop deployment: %w (output: %s)", err, output)
	}

	log.Printf("[Deployment] Stopped %s on %s", deployment.ComposeProject, device.Name)

	// Update status to stopped
	s.updateStatus(deployment, models.DeploymentStatusStopped, "")

	return nil
}

// StartDeployment starts a stopped deployment
func (s *DeploymentService) StartDeployment(id string) error {
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return fmt.Errorf("deployment not found: %w", err)
	}

	// Check if deployment can be started
	if deployment.Status != models.DeploymentStatusStopped {
		return fmt.Errorf("deployment cannot be started (current status: %s)", deployment.Status)
	}

	// Get device for SSH
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

	// Start containers
	startCmd := fmt.Sprintf("cd %s && docker compose -p %s start", deployDir, deployment.ComposeProject)
	output, err := s.sshClient.ExecuteWithTimeout(host, startCmd, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to start deployment: %w (output: %s)", err, output)
	}

	log.Printf("[Deployment] Started %s on %s", deployment.ComposeProject, device.Name)

	// Update status to running
	s.updateStatus(deployment, models.DeploymentStatusRunning, "")

	return nil
}

// GetAccessURLs generates access URLs for a deployment based on exposed ports
func (s *DeploymentService) GetAccessURLs(id string) ([]map[string]interface{}, error) {
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}

	// Get device for IP address
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Extract ports from compose
	portSpecs := ExtractPortsFromCompose(deployment.GeneratedCompose)

	urls := []map[string]interface{}{}
	for _, spec := range portSpecs {
		// Only generate HTTP URLs for TCP ports
		if spec.Protocol == "tcp" {
			// Determine protocol (http vs https)
			protocol := "http"
			if spec.Port == 443 || spec.Port == 8443 {
				protocol = "https"
			}

			url := fmt.Sprintf("%s://%s:%d", protocol, device.IPAddress, spec.Port)

			// Add description - only label ports we're absolutely certain about
			// Recipe-specific port descriptions will be added in a future enhancement
			description := fmt.Sprintf("Port %d", spec.Port)
			switch spec.Port {
			case 80:
				description = "HTTP"
			case 443:
				description = "HTTPS"
			}

			urls = append(urls, map[string]interface{}{
				"url":         url,
				"port":        spec.Port,
				"protocol":    protocol,
				"description": description,
			})
		}
	}

	return urls, nil
}

// executeDeployment performs the actual deployment steps
func (s *DeploymentService) executeDeployment(ctx context.Context, deployment *models.Deployment, recipe *models.Recipe, device *models.Device) {
	// Clean up cancel function when deployment completes (success, failure, or cancellation)
	defer func() {
		s.cancelFuncs.Delete(deployment.ID.String())
	}()

	// Check if already cancelled before starting
	select {
	case <-ctx.Done():
		s.appendLog(deployment, "Deployment cancelled before starting")
		s.updateStatus(deployment, models.DeploymentStatusFailed, "Deployment was cancelled")
		return
	default:
	}

	// Acquire device lock to prevent concurrent deployments to the same device
	deviceLock := s.acquireDeviceLock(device.ID)
	defer s.releaseDeviceLock(device.ID)

	deviceLock.Lock()
	defer deviceLock.Unlock()

	s.appendLog(deployment, fmt.Sprintf("Starting deployment of %s to device %s (%s)", recipe.Name, device.Name, device.IPAddress))
	s.appendLog(deployment, fmt.Sprintf("Acquired deployment lock for device %s", device.Name))

	// Update status to preparing
	s.updateStatus(deployment, models.DeploymentStatusPreparing, "")

	// Parse user config from deployment
	var userConfig map[string]interface{}
	if err := json.Unmarshal(deployment.Config, &userConfig); err != nil {
		s.appendLog(deployment, fmt.Sprintf("❌ Failed to parse config: %v", err))
		s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Failed to parse config: %v", err))
		return
	}

	// INTELLIGENT ORCHESTRATION: Database Provisioning
	var provisionedDB *models.ProvisionedDatabase
	if recipe.Database.AutoProvision && recipe.Database.Engine != "none" && recipe.Database.Engine != "" {
		s.appendLog(deployment, fmt.Sprintf("Provisioning %s database using intelligent pooling...", recipe.Database.Engine))

		db, err := s.dbPoolManager.ProvisionDatabase(deployment, device, recipe.Database)
		if err != nil {
			s.appendLog(deployment, fmt.Sprintf("❌ Database provisioning failed: %v", err))
			s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Failed to provision database: %v", err))
			return
		}
		provisionedDB = db
		s.appendLog(deployment, fmt.Sprintf("✓ Database provisioned: %s", provisionedDB.DatabaseName))
		s.appendLog(deployment, fmt.Sprintf("  Using shared %s instance (saving ~%dMB RAM)", recipe.Database.Engine, 200))
	}

	// Build environment variables (replaces template rendering)
	s.appendLog(deployment, "Building environment variables...")
	envMap, envFileContent, err := s.environmentBuilder.BuildEnvironment(
		deployment,
		recipe,
		userConfig,
		device,
		provisionedDB,
	)
	if err != nil {
		s.appendLog(deployment, fmt.Sprintf("❌ Environment building failed: %v", err))
		s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Failed to build environment: %v", err))
		return
	}
	s.appendLog(deployment, fmt.Sprintf("✓ Environment variables built (%d vars)", len(envMap)))

	// Get docker-compose content (standard format with ${VAR} substitution via .env file)
	composeContent := recipe.ComposeContent
	s.appendLog(deployment, "✓ Docker Compose prepared")

	// Check for cancellation after template rendering
	select {
	case <-ctx.Done():
		s.appendLog(deployment, "Deployment cancelled during preparation")
		s.updateStatus(deployment, models.DeploymentStatusFailed, "Deployment was cancelled")
		return
	default:
	}

	// Store generated compose and env for debugging
	deployment.GeneratedCompose = composeContent

	// Now sanitize the config in the database
	// This ensures passwords are not kept in memory or database after use
	sanitizedConfig := s.sanitizeConfig(userConfig)
	deployment.Config, _ = json.Marshal(sanitizedConfig)
	s.db.Save(deployment)

	// Ensure homelab-proxy network exists (required for Traefik and other proxy-based services)
	s.appendLog(deployment, "Ensuring Docker networks are ready...")
	if err := s.ensureProxyNetworkExists(device); err != nil {
		s.appendLog(deployment, fmt.Sprintf("⚠️  Warning: Failed to ensure proxy network exists: %v", err))
		// Don't fail deployment, just warn - the network might not be needed
	} else {
		s.appendLog(deployment, "✓ Docker network 'homelab-proxy' is ready")
	}

	// Extract ports from compose
	portsToOpen := ExtractPortsFromCompose(composeContent)
	if len(portsToOpen) > 0 {
		// Format port list for logging
		portList := formatPortSpecs(portsToOpen)
		s.appendLog(deployment, fmt.Sprintf("Detected ports to expose: %s", portList))

		// Check firewall status
		s.appendLog(deployment, "Checking firewall configuration...")
		firewallStatus, err := s.firewallService.CheckFirewall(device)
		if err != nil {
			s.appendLog(deployment, fmt.Sprintf("⚠️  Could not check firewall: %v", err))
		} else if firewallStatus.Installed && firewallStatus.Enabled {
			s.appendLog(deployment, fmt.Sprintf("Firewall detected: %s (active)", firewallStatus.Type))
			s.appendLog(deployment, "Opening required ports on firewall...")

			if err := s.firewallService.OpenPorts(device, portsToOpen); err != nil {
				s.appendLog(deployment, fmt.Sprintf("⚠️  Warning: Failed to open firewall ports: %v", err))
				s.appendLog(deployment, "You may need to manually open ports. See post-deployment instructions.")
			} else {
				s.appendLog(deployment, "✓ Firewall ports opened successfully")
			}
		} else {
			s.appendLog(deployment, "ℹ️  No active firewall detected - ports should be accessible")
		}
	}

	// Update status to deploying
	s.updateStatus(deployment, models.DeploymentStatusDeploying, "")

	// Deploy to device (with env file)
	s.appendLog(deployment, fmt.Sprintf("Deploying containers (project: %s)...", deployment.ComposeProject))
	if err := s.deployToDeviceWithEnv(device, deployment.ComposeProject, composeContent, envFileContent); err != nil {
		s.appendLog(deployment, fmt.Sprintf("❌ Deployment failed: %v", err))
		s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Deployment failed: %v", err))
		// Attempt cleanup of partial deployment
		s.appendLog(deployment, "Attempting cleanup of failed deployment...")
		s.cleanupFailedDeployment(device, deployment.ComposeProject)
		return
	}
	s.appendLog(deployment, "✓ Containers deployed successfully")

	// Check for cancellation after deployment
	select {
	case <-ctx.Done():
		s.appendLog(deployment, "Deployment cancelled during deployment phase")
		s.updateStatus(deployment, models.DeploymentStatusFailed, "Deployment was cancelled")
		// Attempt cleanup of partial deployment
		s.appendLog(deployment, "Attempting cleanup of cancelled deployment...")
		s.cleanupFailedDeployment(device, deployment.ComposeProject)
		return
	default:
	}

	// Update status to health check
	s.updateStatus(deployment, models.DeploymentStatusHealthCheck, "")

	// Wait a bit for containers to start
	s.appendLog(deployment, "Waiting 5 seconds for containers to initialize...")
	time.Sleep(5 * time.Second)

	// Check health
	s.appendLog(deployment, "Running health checks...")
	if err := s.checkDeploymentHealth(device, deployment, recipe); err != nil {
		s.appendLog(deployment, fmt.Sprintf("❌ Health check failed: %v", err))
		s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Health check failed: %v", err))
		// Attempt cleanup of failed deployment
		s.appendLog(deployment, "Attempting cleanup of failed deployment...")
		s.cleanupFailedDeployment(device, deployment.ComposeProject)
		return
	}
	s.appendLog(deployment, "✓ Health checks passed")

	// Update to running
	now := time.Now()
	deployment.DeployedAt = &now
	s.appendLog(deployment, "🎉 Deployment completed successfully!")
	s.updateStatus(deployment, models.DeploymentStatusRunning, "")
}


// sanitizeConfig removes sensitive data (passwords, keys, tokens) from config before storage
func (s *DeploymentService) sanitizeConfig(config map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	// List of field names that contain sensitive data
	sensitiveFields := map[string]bool{
		"password":           true,
		"dashboard_password": true,
		"admin_password":     true,
		"db_password":        true,
		"api_key":            true,
		"api_token":          true,
		"secret_key":         true,
		"private_key":        true,
		"ssh_key":            true,
		"token":              true,
	}

	// Copy all non-sensitive fields
	for key, value := range config {
		lowerKey := strings.ToLower(key)
		if !sensitiveFields[lowerKey] && !strings.Contains(lowerKey, "password") && !strings.Contains(lowerKey, "secret") {
			sanitized[key] = value
		} else {
			// Replace sensitive values with placeholder
			sanitized[key] = "[REDACTED]"
			log.Printf("[Deployment] Sanitized sensitive field: %s", key)
		}
	}

	return sanitized
}

// ensureProxyNetworkExists ensures the homelab-proxy network exists on the target device
func (s *DeploymentService) ensureProxyNetworkExists(device *models.Device) error {
	host := fmt.Sprintf("%s:22", device.IPAddress)

	// Check if network exists using Docker's native filtering (more portable than grep)
	checkCmd := "docker network ls --filter name=^homelab-proxy$ --format '{{.Name}}'"
	output, err := s.sshClient.ExecuteWithTimeout(host, checkCmd, 10*time.Second)

	// If output is empty or error occurred, network doesn't exist
	if err != nil || strings.TrimSpace(output) == "" {
		// Network doesn't exist, create it
		createCmd := "docker network create homelab-proxy --driver bridge"
		if _, err := s.sshClient.ExecuteWithTimeout(host, createCmd, 30*time.Second); err != nil {
			return fmt.Errorf("failed to create homelab-proxy network: %w", err)
		}
		log.Printf("[Deployment] Created homelab-proxy network on device %s", device.Name)
	}

	return nil
}

// deployToDevice deploys the rendered compose file to the target device (legacy)
func (s *DeploymentService) deployToDevice(device *models.Device, projectName, composeContent string) error {
	return s.deployToDeviceWithEnv(device, projectName, composeContent, "")
}

// deployToDeviceWithEnv deploys docker-compose.yaml + .env file to the target device
func (s *DeploymentService) deployToDeviceWithEnv(device *models.Device, projectName, composeContent, envFileContent string) error {
	host := fmt.Sprintf("%s:22", device.IPAddress)

	// Use home directory instead of /opt to avoid needing sudo
	// ~/homelab-deployments is user-writable and Docker can still access it
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", projectName)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", deployDir)
	if _, err := s.sshClient.ExecuteWithTimeout(host, mkdirCmd, 30*time.Second); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	// Write compose file
	composeFile := fmt.Sprintf("%s/docker-compose.yml", deployDir)
	writeCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", composeFile, composeContent)
	if _, err := s.sshClient.ExecuteWithTimeout(host, writeCmd, 1*time.Minute); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Write .env file if provided
	if envFileContent != "" {
		envFile := fmt.Sprintf("%s/.env", deployDir)
		writeEnvCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", envFile, envFileContent)
		if _, err := s.sshClient.ExecuteWithTimeout(host, writeEnvCmd, 1*time.Minute); err != nil {
			return fmt.Errorf("failed to write .env file: %w", err)
		}
		log.Printf("[Deployment] Wrote .env file with environment variables")
	}

	// Deploy with docker compose
	// Docker Compose will automatically read the .env file
	deployCmd := fmt.Sprintf("cd %s && docker compose -p %s up -d", deployDir, projectName)
	output, err := s.sshClient.ExecuteWithTimeout(host, deployCmd, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w (output: %s)", err, output)
	}

	log.Printf("[Deployment] Successfully deployed %s to %s", projectName, device.Name)
	return nil
}

// checkDeploymentHealth performs health check on the deployment
func (s *DeploymentService) checkDeploymentHealth(device *models.Device, deployment *models.Deployment, recipe *models.Recipe) error {
	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

	// Step 1: Check if containers are running
	checkCmd := fmt.Sprintf("cd %s && docker compose -p %s ps -q", deployDir, deployment.ComposeProject)
	output, err := s.sshClient.ExecuteWithTimeout(host, checkCmd, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to check containers: %w", err)
	}

	containerIDs := strings.TrimSpace(output)
	if containerIDs == "" {
		return fmt.Errorf("no containers found for project %s", deployment.ComposeProject)
	}

	// Step 2: Check container status
	statusCmd := fmt.Sprintf("cd %s && docker compose -p %s ps --format json", deployDir, deployment.ComposeProject)
	statusOutput, err := s.sshClient.ExecuteWithTimeout(host, statusCmd, 1*time.Minute)
	if err == nil {
		// Parse status output
		if !strings.Contains(statusOutput, "\"State\":\"running\"") && !strings.Contains(statusOutput, "Up") {
			return fmt.Errorf("containers are not running properly")
		}
	}

	// Step 3: If recipe defines HTTP health check, test it
	if recipe.HealthCheck.Path != "" {
		s.appendLog(deployment, "Running HTTP health checks...")
		log.Printf("[Deployment] Running HTTP health check for %s", deployment.ComposeProject)

		// Get port from recipe health check configuration
		port := recipe.HealthCheck.Port
		if port == 0 {
			// Default to port 80 if not specified
			port = 80
		}

		// Build health check URL
		healthURL := fmt.Sprintf("http://%s:%d%s", device.IPAddress, port, recipe.HealthCheck.Path)

		// Use curl with timeout on the target device
		timeout := recipe.HealthCheck.TimeoutSeconds
		if timeout == 0 {
			timeout = 30
		}

		curlCmd := fmt.Sprintf("curl -f -s -o /dev/null -w '%%{http_code}' --max-time %d %s", timeout, healthURL)
		// Add extra time beyond curl's timeout for SSH overhead
		sshTimeout := time.Duration(timeout+10) * time.Second
		httpCode, err := s.sshClient.ExecuteWithTimeout(host, curlCmd, sshTimeout)

		if err != nil {
			log.Printf("[Deployment] Health check HTTP request failed: %v", err)
			s.appendLog(deployment, fmt.Sprintf("⚠️  HTTP health check failed: %v (continuing anyway)", err))
			// Don't fail deployment, just warn
			return nil
		}

		statusCode := strings.TrimSpace(httpCode)
		expectedCode := fmt.Sprintf("%d", recipe.HealthCheck.ExpectedStatus)
		if statusCode != expectedCode {
			log.Printf("[Deployment] Health check returned %s, expected %s", statusCode, expectedCode)
			s.appendLog(deployment, fmt.Sprintf("⚠️  HTTP health check returned status %s, expected %s (continuing anyway)", statusCode, expectedCode))
			// Don't fail deployment, just warn
			return nil
		}

		log.Printf("[Deployment] HTTP health check passed: %s returned %s", healthURL, statusCode)
		s.appendLog(deployment, fmt.Sprintf("✓ HTTP health check passed: %s", healthURL))
	}

	log.Printf("[Deployment] Health check passed for %s", deployment.ComposeProject)
	return nil
}

// cleanupFailedDeployment attempts to clean up a failed deployment
func (s *DeploymentService) cleanupFailedDeployment(device *models.Device, projectName string) {
	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", projectName)

	log.Printf("[Deployment] Cleaning up failed deployment: %s", projectName)

	// Try to stop and remove any containers that were created
	cleanupCmd := fmt.Sprintf("cd %s && docker compose -p %s down 2>/dev/null || true", deployDir, projectName)
	output, err := s.sshClient.ExecuteWithTimeout(host, cleanupCmd, 2*time.Minute)
	if err != nil {
		log.Printf("[Deployment] Warning: cleanup may have failed for %s: %v (output: %s)", projectName, err, output)
	} else {
		log.Printf("[Deployment] Cleanup completed for %s", projectName)
	}

	// Try to remove the deployment directory
	removeCmd := fmt.Sprintf("rm -rf %s 2>/dev/null || true", deployDir)
	_, err = s.sshClient.ExecuteWithTimeout(host, removeCmd, 30*time.Second)
	if err != nil {
		log.Printf("[Deployment] Warning: failed to remove deployment directory %s: %v", deployDir, err)
	}
}

// appendLog adds a timestamped log entry to the deployment
func (s *DeploymentService) appendLog(deployment *models.Deployment, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	deployment.DeploymentLogs += logEntry

	// Save logs to database
	s.db.Model(deployment).Update("deployment_logs", deployment.DeploymentLogs)

	// Broadcast log update via WebSocket
	if s.wsHub != nil {
		s.wsHub.Broadcast("deployments", "deployment:log", map[string]interface{}{
			"id":      deployment.ID,
			"message": logEntry,
		})
	}
}

// updateStatus updates the deployment status and broadcasts to WebSocket
func (s *DeploymentService) updateStatus(deployment *models.Deployment, status models.DeploymentStatus, errorDetails string) {
	deployment.Status = status
	deployment.ErrorDetails = errorDetails
	s.db.Save(deployment)

	// Also log the status change
	s.appendLog(deployment, fmt.Sprintf("Status changed to: %s", status))

	// Broadcast status update via WebSocket
	if s.wsHub != nil {
		s.wsHub.Broadcast("deployments", "deployment:status", map[string]interface{}{
			"id":            deployment.ID,
			"status":        deployment.Status,
			"error_details": deployment.ErrorDetails,
		})
	}
}

// generateProjectName generates a unique project name for Docker Compose
func (s *DeploymentService) generateProjectName(recipeSlug string) string {
	// Use recipe slug + short UUID for uniqueness
	shortID := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", recipeSlug, shortID)
}

// acquireDeviceLock gets or creates a mutex for a specific device
func (s *DeploymentService) acquireDeviceLock(deviceID uuid.UUID) *sync.Mutex {
	// Get existing lock or create new one
	lock, _ := s.deviceLocks.LoadOrStore(deviceID.String(), &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// releaseDeviceLock is called when deployment is complete (for future cleanup if needed)
func (s *DeploymentService) releaseDeviceLock(deviceID uuid.UUID) {
	// Currently a no-op, but reserved for potential cleanup logic
	// We keep locks in memory for the lifetime of the service
	// Could implement lock cleanup after X minutes of inactivity in the future
}

// TroubleshootDeployment provides detailed troubleshooting information for a deployment
func (s *DeploymentService) TroubleshootDeployment(id string) (map[string]interface{}, error) {
	// Get deployment
	deployment, err := s.GetDeployment(id)
	if err != nil {
		return nil, err
	}

	// Get device
	device, err := s.deviceService.GetDevice(deployment.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	troubleshoot := make(map[string]interface{})
	troubleshoot["deployment_id"] = deployment.ID
	troubleshoot["deployment_status"] = deployment.Status
	troubleshoot["device_name"] = device.Name
	troubleshoot["device_ip"] = device.IPAddress

	host := fmt.Sprintf("%s:22", device.IPAddress)
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", deployment.ComposeProject)

	// Check container status
	checkCmd := fmt.Sprintf("cd %s && docker compose -p %s ps --format json 2>/dev/null || cd %s && docker compose -p %s ps", deployDir, deployment.ComposeProject, deployDir, deployment.ComposeProject)
	containerStatus, err := s.sshClient.ExecuteWithTimeout(host, checkCmd, 30*time.Second)
	if err != nil {
		troubleshoot["container_status"] = fmt.Sprintf("Error: %v", err)
	} else {
		troubleshoot["container_status"] = containerStatus
	}

	// Extract ports from compose
	ports := ExtractPortsFromCompose(deployment.GeneratedCompose)
	troubleshoot["required_ports"] = ports

	// Get firewall status (structured data)
	firewallStatus, err := s.firewallService.CheckFirewall(device)
	if err != nil {
		troubleshoot["firewall_status"] = map[string]interface{}{
			"enabled": false,
			"error":   err.Error(),
		}
	} else {
		troubleshoot["firewall_status"] = map[string]interface{}{
			"enabled":    firewallStatus.Enabled,
			"type":       firewallStatus.Type,
			"installed":  firewallStatus.Installed,
			"open_ports": firewallStatus.OpenPorts,
		}
	}

	// Get firewall troubleshooting recommendations (text)
	firewallSteps, err := s.firewallService.GetFirewallTroubleshootingSteps(device, ports)
	if err == nil {
		troubleshoot["recommendations"] = []string{firewallSteps}
	}

	// Get container logs (last 50 lines)
	logsCmd := fmt.Sprintf("cd %s && docker compose -p %s logs --tail=50", deployDir, deployment.ComposeProject)
	containerLogs, err := s.sshClient.ExecuteWithTimeout(host, logsCmd, 30*time.Second)
	if err != nil {
		troubleshoot["recent_logs"] = fmt.Sprintf("Error: %v", err)
	} else {
		troubleshoot["recent_logs"] = containerLogs
	}

	// Check which ports are actually listening
	listeningCmd := "ss -tuln | grep LISTEN"
	listeningPorts, err := s.sshClient.ExecuteWithTimeout(host, listeningCmd, 10*time.Second)
	if err == nil {
		troubleshoot["listening_ports"] = listeningPorts
	}

	troubleshoot["generated_compose"] = deployment.GeneratedCompose
	troubleshoot["deployment_logs"] = deployment.DeploymentLogs

	return troubleshoot, nil
}

// formatPortSpecs formats a slice of PortSpec for logging
func formatPortSpecs(portSpecs []PortSpec) string {
	if len(portSpecs) == 0 {
		return "none"
	}

	parts := make([]string, len(portSpecs))
	for i, spec := range portSpecs {
		parts[i] = fmt.Sprintf("%d/%s", spec.Port, spec.Protocol)
	}
	return strings.Join(parts, ", ")
}

// cleanupFirewallPorts closes firewall ports that are no longer needed after deployment deletion
// Only closes ports if no other deployments on the same device are using them
func (s *DeploymentService) cleanupFirewallPorts(device *models.Device, deletedDeploymentID uuid.UUID, ports []PortSpec) error {
	// Get all other deployments on this device
	var otherDeployments []models.Deployment
	if err := s.db.Where("device_id = ? AND id != ? AND status IN ?",
		device.ID,
		deletedDeploymentID,
		[]models.DeploymentStatus{
			models.DeploymentStatusRunning,
			models.DeploymentStatusDeploying,
			models.DeploymentStatusHealthCheck,
		}).Find(&otherDeployments).Error; err != nil {
		return fmt.Errorf("failed to query deployments: %w", err)
	}

	// Build map of ports in use by other deployments
	portsInUse := make(map[string]bool) // key: "port/protocol"
	for _, deployment := range otherDeployments {
		otherPorts := ExtractPortsFromCompose(deployment.GeneratedCompose)
		for _, spec := range otherPorts {
			key := fmt.Sprintf("%d/%s", spec.Port, spec.Protocol)
			portsInUse[key] = true
		}
	}

	// Filter out ports that are still in use
	portsToClose := []PortSpec{}
	for _, spec := range ports {
		key := fmt.Sprintf("%d/%s", spec.Port, spec.Protocol)
		if !portsInUse[key] {
			portsToClose = append(portsToClose, spec)
		}
	}

	// Close unused ports
	if len(portsToClose) > 0 {
		log.Printf("[Deployment] Closing firewall ports no longer in use: %s", formatPortSpecs(portsToClose))
		if err := s.firewallService.ClosePorts(device, portsToClose); err != nil {
			return fmt.Errorf("failed to close ports: %w", err)
		}
		log.Printf("[Deployment] Successfully closed %d firewall port(s)", len(portsToClose))
	} else {
		log.Printf("[Deployment] All ports still in use by other deployments, skipping cleanup")
	}

	return nil
}
