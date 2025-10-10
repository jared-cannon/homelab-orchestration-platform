package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// Template execution limits to prevent resource exhaustion attacks
	templateExecutionTimeout = 10 * time.Second
	maxTemplateOutputSize    = 1024 * 1024 // 1MB max for docker-compose.yml
)

// DeploymentService handles deployment operations
type DeploymentService struct {
	db            *gorm.DB
	sshClient     *ssh.Client
	recipeLoader  *RecipeLoader
	deviceService *DeviceService
	wsHub         WSHub
	deviceLocks   sync.Map // Map of device ID -> *sync.Mutex to prevent concurrent deployments
	cancelFuncs   sync.Map // Map of deployment ID -> context.CancelFunc for cancellation
}

// WSHub interface for WebSocket broadcasting
type WSHub interface {
	Broadcast(channel string, event string, data interface{})
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(
	db *gorm.DB,
	sshClient *ssh.Client,
	recipeLoader *RecipeLoader,
	deviceService *DeviceService,
	wsHub WSHub,
) *DeploymentService {
	return &DeploymentService{
		db:            db,
		sshClient:     sshClient,
		recipeLoader:  recipeLoader,
		deviceService: deviceService,
		wsHub:         wsHub,
	}
}

// CreateDeploymentRequest represents a request to create a deployment
type CreateDeploymentRequest struct {
	RecipeSlug string                 `json:"recipe_slug"`
	DeviceID   uuid.UUID              `json:"device_id"`
	Config     map[string]interface{} `json:"config"`
}

// CreateDeployment creates and deploys a new application
func (s *DeploymentService) CreateDeployment(req CreateDeploymentRequest) (*models.Deployment, error) {
	// Get the recipe
	recipe, err := s.recipeLoader.GetRecipe(req.RecipeSlug)
	if err != nil {
		return nil, fmt.Errorf("recipe not found: %w", err)
	}

	// Validate the recipe template before deployment
	if err := s.validateRecipeTemplate(recipe); err != nil {
		return nil, fmt.Errorf("invalid recipe template: %w", err)
	}

	// Validate that all required fields are provided
	if err := s.validateRequiredFields(recipe, req.Config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get the device
	device, err := s.deviceService.GetDevice(req.DeviceID)
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
		DeviceID:       req.DeviceID,
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

	// Stop and remove containers
	if deployment.ComposeProject != "" {
		host := fmt.Sprintf("%s:22", device.IPAddress)

		// Stop and remove the compose project (WITHOUT removing volumes to preserve data)
		// Users should manually delete volumes if they want to remove data
		stopCmd := fmt.Sprintf("docker compose -p %s down", deployment.ComposeProject)
		output, err := s.sshClient.ExecuteWithTimeout(host, stopCmd, 2*time.Minute)
		if err != nil {
			return fmt.Errorf("failed to stop deployment: %w (output: %s)", err, output)
		}

		log.Printf("[Deployment] Stopped %s (volumes preserved)", deployment.ComposeProject)
	}

	// Delete from database
	if err := s.db.Delete(deployment).Error; err != nil {
		return fmt.Errorf("failed to delete deployment record: %w", err)
	}

	return nil
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

	// Render the Docker Compose template
	s.appendLog(deployment, "Rendering Docker Compose template...")
	rendered, err := s.renderComposeTemplate(recipe, deployment, device)
	if err != nil {
		s.appendLog(deployment, fmt.Sprintf("❌ Template rendering failed: %v", err))
		s.updateStatus(deployment, models.DeploymentStatusFailed, fmt.Sprintf("Failed to render template: %v", err))
		return
	}
	s.appendLog(deployment, "✓ Template rendered successfully")

	// Check for cancellation after template rendering
	select {
	case <-ctx.Done():
		s.appendLog(deployment, "Deployment cancelled during preparation")
		s.updateStatus(deployment, models.DeploymentStatusFailed, "Deployment was cancelled")
		return
	default:
	}

	// Store generated compose for debugging
	deployment.GeneratedCompose = rendered

	// Now that we've rendered the template, sanitize the config in the database
	// This ensures passwords are not kept in memory or database after use
	sanitizedConfig := s.sanitizeConfig(make(map[string]interface{})) // Empty for now, will be updated below
	var originalConfig map[string]interface{}
	json.Unmarshal(deployment.Config, &originalConfig)
	sanitizedConfig = s.sanitizeConfig(originalConfig)
	deployment.Config, _ = json.Marshal(sanitizedConfig)

	s.db.Save(deployment)

	// Update status to deploying
	s.updateStatus(deployment, models.DeploymentStatusDeploying, "")

	// Deploy to device
	s.appendLog(deployment, fmt.Sprintf("Deploying containers (project: %s)...", deployment.ComposeProject))
	if err := s.deployToDevice(device, deployment.ComposeProject, rendered); err != nil {
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

// renderComposeTemplate renders the Docker Compose template with user config
func (s *DeploymentService) renderComposeTemplate(recipe *models.Recipe, deployment *models.Deployment, device *models.Device) (string, error) {
	// Parse user config
	var config map[string]interface{}
	if err := json.Unmarshal(deployment.Config, &config); err != nil {
		return "", fmt.Errorf("failed to parse config: %w", err)
	}

	// Pre-process config: Generate htpasswd hashes if needed
	if err := s.preprocessConfig(config, device); err != nil {
		return "", fmt.Errorf("failed to preprocess config: %w", err)
	}

	// Normalize config keys: Convert snake_case to PascalCase for Go templates
	// This allows YAML to use natural snake_case while templates use Go conventions
	normalizedConfig := s.normalizeConfigKeys(config)

	// Add deployment-specific variables
	normalizedConfig["DEPLOYMENT_ID"] = deployment.ID.String()
	normalizedConfig["COMPOSE_PROJECT"] = deployment.ComposeProject

	// Parse and execute template with security limits
	tmpl, err := template.New("compose").Parse(recipe.ComposeTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template with timeout and size limits to prevent resource exhaustion
	rendered, err := s.executeTemplateWithLimits(tmpl, normalizedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return rendered, nil
}

// executeTemplateWithLimits executes a template with timeout and size limits to prevent resource exhaustion
func (s *DeploymentService) executeTemplateWithLimits(tmpl *template.Template, data interface{}) (string, error) {
	// Create a channel to receive the result or error
	type result struct {
		output string
		err    error
	}
	resultChan := make(chan result, 1)

	// Execute template in a goroutine with timeout protection
	ctx, cancel := context.WithTimeout(context.Background(), templateExecutionTimeout)
	defer cancel()

	go func() {
		var buf bytes.Buffer
		err := tmpl.Execute(&buf, data)

		if err != nil {
			resultChan <- result{err: err}
			return
		}

		output := buf.String()

		// Check output size to prevent memory exhaustion
		if len(output) > maxTemplateOutputSize {
			resultChan <- result{
				err: fmt.Errorf("template output exceeds maximum size of %d bytes (got %d bytes)",
					maxTemplateOutputSize, len(output)),
			}
			return
		}

		resultChan <- result{output: output}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			return "", res.err
		}
		return res.output, nil
	case <-ctx.Done():
		return "", fmt.Errorf("template execution timed out after %v (possible infinite loop)", templateExecutionTimeout)
	}
}

// preprocessConfig handles special config values that need processing before template rendering
func (s *DeploymentService) preprocessConfig(config map[string]interface{}, device *models.Device) error {
	// Handle Traefik dashboard password - generate htpasswd hash
	if username, ok := config["dashboard_username"].(string); ok {
		if password, ok := config["dashboard_password"].(string); ok && password != "" {
			hash, err := s.generateHtpasswdHash(device, username, password)
			if err != nil {
				return fmt.Errorf("failed to generate password hash: %w", err)
			}
			// Add the hash to config for template rendering
			config["DashboardPasswordHash"] = hash
			log.Printf("[Deployment] Generated htpasswd hash for user: %s", username)
		}
	}

	return nil
}

// generateHtpasswdHash generates a bcrypt hash compatible with Apache htpasswd format
// This is done on the backend server, not on the target device, so no SSH or htpasswd binary is needed
func (s *DeploymentService) generateHtpasswdHash(device *models.Device, username, password string) (string, error) {
	// Validate username contains no special characters
	if username == "" || strings.ContainsAny(username, ":$\n\r") {
		return "", fmt.Errorf("invalid username: must not contain special characters")
	}

	// Generate bcrypt hash (cost 10 is the default, equivalent to htpasswd -B)
	// bcrypt is safe for passwords up to 72 bytes
	if len(password) > 72 {
		return "", fmt.Errorf("password too long: bcrypt supports maximum 72 bytes")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}

	// Format: username:$2a$10$hash...
	// Go's bcrypt generates $2a$ format which is compatible with Apache htpasswd and Traefik
	htpasswdLine := fmt.Sprintf("%s:%s", username, string(hashedPassword))

	// Escape $ as $$ for Docker Compose YAML (Docker Compose uses $$ to represent literal $)
	htpasswdLine = strings.ReplaceAll(htpasswdLine, "$", "$$")

	log.Printf("[Deployment] Generated htpasswd-compatible bcrypt hash locally (no SSH required)")
	return htpasswdLine, nil
}

// validateCredentialInput validates that input doesn't contain dangerous characters
func (s *DeploymentService) validateCredentialInput(input string) error {
	// Check for common shell metacharacters that could be used for injection
	dangerous := []string{";", "|", "&", "$", "`", "\n", "\r", "$(", "${", ">>", "<<"}
	for _, char := range dangerous {
		if strings.Contains(input, char) {
			return fmt.Errorf("input contains forbidden character: %s", char)
		}
	}
	return nil
}

// shellEscape properly escapes a string for safe use in shell commands
func (s *DeploymentService) shellEscape(input string) string {
	// Use single quotes and escape any single quotes in the input
	// This is the safest approach for arbitrary strings
	escaped := strings.ReplaceAll(input, "'", "'\"'\"'")
	return fmt.Sprintf("'%s'", escaped)
}

// normalizeConfigKeys converts snake_case keys to PascalCase for Go template compatibility
// Example: dashboard_username -> DashboardUsername, enable_ssl -> EnableSsl
func (s *DeploymentService) normalizeConfigKeys(config map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for key, value := range config {
		// Convert snake_case to PascalCase
		pascalKey := s.snakeToPascalCase(key)
		normalized[pascalKey] = value

		// Also keep original key for backwards compatibility
		normalized[key] = value
	}

	return normalized
}

// snakeToPascalCase converts snake_case to PascalCase
// Examples: dashboard_username -> DashboardUsername, enable_ssl -> EnableSsl
func (s *DeploymentService) snakeToPascalCase(input string) string {
	parts := strings.Split(input, "_")
	for i, part := range parts {
		if len(part) > 0 {
			// Capitalize first letter of each part
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
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

// deployToDevice deploys the rendered compose file to the target device
func (s *DeploymentService) deployToDevice(device *models.Device, projectName, composeContent string) error {
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

	// Deploy with docker compose
	// This can take a long time if images need to be pulled
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

	// Step 1: Check if containers are running
	checkCmd := fmt.Sprintf("docker compose -p %s ps -q", deployment.ComposeProject)
	output, err := s.sshClient.ExecuteWithTimeout(host, checkCmd, 1*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to check containers: %w", err)
	}

	containerIDs := strings.TrimSpace(output)
	if containerIDs == "" {
		return fmt.Errorf("no containers found for project %s", deployment.ComposeProject)
	}

	// Step 2: Check container status
	statusCmd := fmt.Sprintf("docker compose -p %s ps --format json", deployment.ComposeProject)
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

	log.Printf("[Deployment] Cleaning up failed deployment: %s", projectName)

	// Try to stop and remove any containers that were created
	cleanupCmd := fmt.Sprintf("docker compose -p %s down 2>/dev/null || true", projectName)
	output, err := s.sshClient.ExecuteWithTimeout(host, cleanupCmd, 2*time.Minute)
	if err != nil {
		log.Printf("[Deployment] Warning: cleanup may have failed for %s: %v (output: %s)", projectName, err, output)
	} else {
		log.Printf("[Deployment] Cleanup completed for %s", projectName)
	}

	// Try to remove the deployment directory
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", projectName)
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

// validateRequiredFields checks that all required config options are provided
func (s *DeploymentService) validateRequiredFields(recipe *models.Recipe, config map[string]interface{}) error {
	var missingFields []string

	for _, option := range recipe.ConfigOptions {
		if option.Required {
			value, exists := config[option.Name]
			if !exists {
				missingFields = append(missingFields, option.Name)
			} else {
				// Check for empty string values
				if strValue, ok := value.(string); ok && strValue == "" {
					missingFields = append(missingFields, option.Name)
				}
			}
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("missing required fields: %s", strings.Join(missingFields, ", "))
	}

	log.Printf("[Deployment] All required fields validated for recipe: %s", recipe.Slug)
	return nil
}

// validateRecipeTemplate validates that the recipe template is valid Go template syntax
func (s *DeploymentService) validateRecipeTemplate(recipe *models.Recipe) error {
	// Try to parse the template
	tmpl, err := template.New("validation").Parse(recipe.ComposeTemplate)
	if err != nil {
		return fmt.Errorf("template syntax error: %w", err)
	}

	// Create a dummy config with all required fields
	dummyConfig := make(map[string]interface{})
	for _, option := range recipe.ConfigOptions {
		// Use default values for validation
		dummyConfig[option.Name] = option.Default
	}

	// Normalize keys for validation (same as runtime)
	normalizedDummyConfig := s.normalizeConfigKeys(dummyConfig)

	// Add deployment-specific variables
	normalizedDummyConfig["DEPLOYMENT_ID"] = "test-id"
	normalizedDummyConfig["COMPOSE_PROJECT"] = "test-project"

	// Try to execute the template with dummy data using secure execution
	rendered, err := s.executeTemplateWithLimits(tmpl, normalizedDummyConfig)
	if err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}

	// Basic YAML syntax validation - check for common issues
	if rendered == "" {
		return fmt.Errorf("template rendered to empty string")
	}

	log.Printf("[Deployment] Template validation passed for recipe: %s", recipe.Slug)
	return nil
}
