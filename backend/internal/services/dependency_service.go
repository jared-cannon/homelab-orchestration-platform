package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DependencyService handles dependency checking and auto-provisioning
// Follows clean architecture: separates dependency logic from deployment logic
type DependencyService struct {
	db                *gorm.DB
	sshClient         *ssh.Client
	recipeLoader      *RecipeLoader
	deviceService     *DeviceService
	softwareService   *SoftwareService
	databasePool      *DatabasePoolManager
	cachePool         *CachePoolManager
	infraConfig       *InfrastructureConfig
	orchestrator      ContainerOrchestrator
}

// NewDependencyService creates a new dependency service instance
func NewDependencyService(
	db *gorm.DB,
	sshClient *ssh.Client,
	recipeLoader *RecipeLoader,
	deviceService *DeviceService,
	softwareService *SoftwareService,
	databasePool *DatabasePoolManager,
	cachePool *CachePoolManager,
	infraConfig *InfrastructureConfig,
	orchestrator ContainerOrchestrator,
) *DependencyService {
	return &DependencyService{
		db:              db,
		sshClient:       sshClient,
		recipeLoader:    recipeLoader,
		deviceService:   deviceService,
		softwareService: softwareService,
		databasePool:    databasePool,
		cachePool:       cachePool,
		infraConfig:     infraConfig,
		orchestrator:    orchestrator,
	}
}

// DependencyCheckResult contains the result of dependency analysis
type DependencyCheckResult struct {
	Satisfied       bool                 `json:"satisfied"`        // All required dependencies satisfied
	Missing         []MissingDependency  `json:"missing"`          // List of missing dependencies
	ToProvision     []ProvisionPlan      `json:"to_provision"`     // Dependencies that will be auto-provisioned
	Warnings        []string             `json:"warnings"`         // Non-critical warnings
	EstimatedTime   int                  `json:"estimated_time"`   // Estimated total provisioning time (seconds)
	ResourceImpact  ResourceImpact       `json:"resource_impact"`  // RAM/storage that will be consumed
}

// MissingDependency represents a dependency that is not satisfied
type MissingDependency struct {
	Dependency models.RecipeDependency `json:"dependency"`
	Critical   bool                    `json:"critical"`  // Required (true) or recommended (false)
	Reason     string                  `json:"reason"`    // Why it's missing
	CanProvision bool                  `json:"can_provision"` // Can be auto-provisioned
}

// ProvisionPlan describes how a dependency will be provisioned
type ProvisionPlan struct {
	Type              string            `json:"type"`               // reverse_proxy, database, cache, etc.
	Name              string            `json:"name"`               // Human-readable name
	Action            string            `json:"action"`             // "deploy", "create_in_shared", "configure"
	UseSharedInstance bool              `json:"use_shared_instance"` // Using shared DB/cache instance
	EstimatedTime     int               `json:"estimated_time"`     // Seconds
	RAMRequired       int               `json:"ram_required"`       // MB
	StorageRequired   int               `json:"storage_required"`   // GB
	RecipeSlug        string            `json:"recipe_slug,omitempty"` // Recipe to deploy (for apps)
	Config            map[string]string `json:"config,omitempty"`   // Configuration for provisioning
}

// ResourceImpact describes resource consumption of dependencies
type ResourceImpact struct {
	TotalRAMMB      int    `json:"total_ram_mb"`
	TotalStorageGB  int    `json:"total_storage_gb"`
	Breakdown       string `json:"breakdown"` // Human-readable breakdown
}

// CheckDependencies analyzes recipe dependencies and determines what needs to be provisioned
// This is the main entry point for dependency checking
func (s *DependencyService) CheckDependencies(
	ctx context.Context,
	recipe *models.Recipe,
	deviceID uuid.UUID,
) (*DependencyCheckResult, error) {
	result := &DependencyCheckResult{
		Satisfied:      true,
		Missing:        []MissingDependency{},
		ToProvision:    []ProvisionPlan{},
		Warnings:       []string{},
		EstimatedTime:  0,
		ResourceImpact: ResourceImpact{},
	}

	// Check required dependencies
	for _, dep := range recipe.Dependencies.Required {
		satisfied, reason, err := s.checkDependency(ctx, dep, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to check dependency %s: %w", dep.Type, err)
		}

		if !satisfied {
			result.Satisfied = false

			missing := MissingDependency{
				Dependency:   dep,
				Critical:     true,
				Reason:       reason,
				CanProvision: dep.AutoProvision,
			}
			result.Missing = append(result.Missing, missing)

			// Create provision plan if auto-provision is enabled
			if dep.AutoProvision {
				plan, err := s.createProvisionPlan(dep, deviceID, recipe)
				if err != nil {
					result.Warnings = append(result.Warnings,
						fmt.Sprintf("Cannot auto-provision %s: %v", dep.Type, err))
					missing.CanProvision = false
				} else {
					result.ToProvision = append(result.ToProvision, plan)
					result.EstimatedTime += plan.EstimatedTime
					result.ResourceImpact.TotalRAMMB += plan.RAMRequired
					result.ResourceImpact.TotalStorageGB += plan.StorageRequired
				}
			}
		}
	}

	// Check recommended dependencies (non-critical)
	for _, dep := range recipe.Dependencies.Recommended {
		satisfied, reason, _ := s.checkDependency(ctx, dep, deviceID)
		if !satisfied {
			result.Missing = append(result.Missing, MissingDependency{
				Dependency:   dep,
				Critical:     false,
				Reason:       reason,
				CanProvision: dep.AutoProvision,
			})

			// Add as warning, not error
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Recommended: %s - %s", dep.Purpose, reason))
		}
	}

	// Generate resource impact breakdown
	result.ResourceImpact.Breakdown = s.generateResourceBreakdown(result.ToProvision)

	return result, nil
}

// checkDependency checks if a specific dependency is satisfied
// Returns (satisfied, reason, error)
func (s *DependencyService) checkDependency(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	switch dep.Type {
	case "reverse_proxy":
		return s.checkReverseProxy(ctx, dep, deviceID)
	case "database":
		return s.checkDatabase(ctx, dep, deviceID)
	case "cache":
		return s.checkCache(ctx, dep, deviceID)
	case "application":
		return s.checkApplication(ctx, dep, deviceID)
	case "infrastructure":
		return s.checkInfrastructure(ctx, dep, deviceID)
	case "backup":
		return s.checkBackup(ctx, dep, deviceID)
	default:
		return false, fmt.Sprintf("unknown dependency type: %s", dep.Type), nil
	}
}

// checkReverseProxy checks if a reverse proxy is running on the device
func (s *DependencyService) checkReverseProxy(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	// Get all deployments on this device
	var deployments []models.Deployment
	err := s.db.WithContext(ctx).Where("device_id = ? AND status = ?", deviceID, models.DeploymentStatusRunning).
		Find(&deployments).Error
	if err != nil {
		return false, "", fmt.Errorf("failed to query deployments: %w", err)
	}

	// Check if any deployment is a reverse proxy
	reverseProxies := []string{"traefik", "caddy", "nginx-proxy-manager"}
	if dep.Prefer != "" {
		reverseProxies = append([]string{dep.Prefer}, reverseProxies...)
	}
	if len(dep.Alternatives) > 0 {
		reverseProxies = append(reverseProxies, dep.Alternatives...)
	}

	for _, deployment := range deployments {
		// Load recipe to get slug
		recipe, err := s.recipeLoader.GetRecipe(deployment.RecipeSlug)
		if err != nil {
			continue
		}

		for _, proxy := range reverseProxies {
			if recipe.Slug == proxy {
				return true, "", nil
			}
		}
	}

	preferredProxy := "traefik"
	if dep.Prefer != "" {
		preferredProxy = dep.Prefer
	}

	return false, fmt.Sprintf("No reverse proxy found (will deploy %s)", preferredProxy), nil
}

// checkDatabase checks if database is available (shared instance or dedicated)
func (s *DependencyService) checkDatabase(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	// If shared instance requested (default), check if shared DB exists
	if dep.Shared {
		engine := dep.Engine
		if engine == "" {
			engine = "postgres" // Default to postgres
		}

		exists, err := s.databasePool.SharedInstanceExists(ctx, deviceID, engine)
		if err != nil {
			return false, "", fmt.Errorf("failed to check shared database: %w", err)
		}

		if exists {
			// Shared instance exists, dependency is satisfied
			return true, "", nil
		}

		return false, fmt.Sprintf("No shared %s instance (will create)", engine), nil
	}

	// For dedicated instances, check if specific database deployment exists
	if dep.Name != "" {
		var deployment models.Deployment
		err := s.db.WithContext(ctx).Where("device_id = ? AND recipe_slug = ? AND status = ?",
			deviceID, dep.Name, models.DeploymentStatusRunning).
			First(&deployment).Error

		if err == nil {
			return true, "", nil
		}
		if err != gorm.ErrRecordNotFound {
			return false, "", fmt.Errorf("failed to check database deployment: %w", err)
		}
	}

	return false, "Database not available", nil
}

// checkCache checks if cache is available (shared instance or dedicated)
func (s *DependencyService) checkCache(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	// If shared instance requested (default), check if shared cache exists
	if dep.Shared {
		engine := dep.Engine
		if engine == "" {
			engine = s.infraConfig.GetDefaultCacheEngine() // Default to valkey
		}

		exists, err := s.cachePool.SharedInstanceExists(ctx, deviceID, engine)
		if err != nil {
			return false, "", fmt.Errorf("failed to check shared cache: %w", err)
		}

		if exists {
			// Shared instance exists, dependency is satisfied
			return true, "", nil
		}

		return false, fmt.Sprintf("No shared %s instance (will create)", engine), nil
	}

	// Check for dedicated cache deployment
	if dep.Name != "" {
		var deployment models.Deployment
		err := s.db.WithContext(ctx).Where("device_id = ? AND recipe_slug = ? AND status = ?",
			deviceID, dep.Name, models.DeploymentStatusRunning).
			First(&deployment).Error

		if err == nil {
			return true, "", nil
		}
		if err != gorm.ErrRecordNotFound {
			return false, "", fmt.Errorf("failed to check cache deployment: %w", err)
		}
	}

	return false, "Cache not available", nil
}

// checkApplication checks if a specific application is deployed
func (s *DependencyService) checkApplication(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	if dep.Name == "" {
		return false, "Application dependency must specify name", nil
	}

	var deployment models.Deployment
	err := s.db.WithContext(ctx).Where("device_id = ? AND recipe_slug = ? AND status = ?",
		deviceID, dep.Name, models.DeploymentStatusRunning).
		First(&deployment).Error

	if err == nil {
		// Check version if specified
		if dep.MinVersion != "" {
			// TODO: Version comparison logic
			// For now, assume version is satisfied
		}
		return true, "", nil
	}

	if err == gorm.ErrRecordNotFound {
		return false, fmt.Sprintf("Application %s not deployed", dep.Name), nil
	}

	return false, "", fmt.Errorf("failed to check application: %w", err)
}

// checkInfrastructure checks if infrastructure component is installed
func (s *DependencyService) checkInfrastructure(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	// Check if software/infrastructure is installed
	// This would check things like Docker, specific versions of software, etc.

	// TODO: Implement infrastructure checking
	// This requires methods like IsDockerInstalled, which don't exist yet
	// For now, assume infrastructure dependencies are satisfied

	if dep.Name == "docker" {
		// TODO: Check if Docker is actually installed
		// For now, assume it's installed since we require it for deployments anyway
		return true, "", nil
	}

	return false, fmt.Sprintf("Infrastructure %s checking not yet implemented", dep.Name), nil
}

// checkBackup checks if backup is configured
func (s *DependencyService) checkBackup(
	ctx context.Context,
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (bool, string, error) {
	// Backup is always a recommendation, never required
	// For now, we'll just return that it's not configured
	return false, "Backup not configured (recommended)", nil
}

// createProvisionPlan creates a plan for provisioning a dependency
func (s *DependencyService) createProvisionPlan(
	dep models.RecipeDependency,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
) (ProvisionPlan, error) {
	plan := ProvisionPlan{
		Type:   dep.Type,
		Config: make(map[string]string),
	}

	switch dep.Type {
	case "reverse_proxy":
		return s.createReverseProxyPlan(dep, deviceID)
	case "database":
		return s.createDatabasePlan(dep, deviceID, appRecipe)
	case "cache":
		return s.createCachePlan(dep, deviceID, appRecipe)
	case "application":
		return s.createApplicationPlan(dep, deviceID)
	default:
		return plan, fmt.Errorf("cannot create provision plan for type: %s", dep.Type)
	}
}

// createReverseProxyPlan creates a plan for deploying reverse proxy
func (s *DependencyService) createReverseProxyPlan(
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (ProvisionPlan, error) {
	proxySlug := "traefik" // Default
	if dep.Prefer != "" {
		proxySlug = dep.Prefer
	}

	// Get recipe to estimate resources
	recipe, err := s.recipeLoader.GetRecipe(proxySlug)
	if err != nil {
		return ProvisionPlan{}, fmt.Errorf("failed to load reverse proxy recipe: %w", err)
	}

	return ProvisionPlan{
		Type:            "reverse_proxy",
		Name:            fmt.Sprintf("%s (Reverse Proxy)", recipe.Name),
		Action:          "deploy",
		UseSharedInstance: false,
		EstimatedTime:   60, // ~1 minute for traefik deployment
		RAMRequired:     recipe.GetEstimatedRAMMB(),
		StorageRequired: recipe.GetEstimatedStorageGB(),
		RecipeSlug:      proxySlug,
		Config: map[string]string{
			"purpose": "Provides HTTPS and automatic SSL certificates",
		},
	}, nil
}

// createDatabasePlan creates a plan for provisioning database
func (s *DependencyService) createDatabasePlan(
	dep models.RecipeDependency,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
) (ProvisionPlan, error) {
	engine := dep.Engine
	if engine == "" {
		engine = "postgres" // Default
	}

	dbName := fmt.Sprintf("%s_db", appRecipe.Slug)

	plan := ProvisionPlan{
		Type:              "database",
		Name:              fmt.Sprintf("%s Database", engine),
		UseSharedInstance: dep.Shared,
		Config: map[string]string{
			"engine":  engine,
			"db_name": dbName,
		},
	}

	if dep.Shared {
		// Creating database in shared instance
		plan.Action = "create_in_shared"
		plan.EstimatedTime = 30 // ~30 seconds to create DB in shared instance
		plan.RAMRequired = 0    // No additional RAM (using shared)
		plan.StorageRequired = 0 // Minimal storage
	} else {
		// Deploying dedicated database instance
		plan.Action = "deploy"
		plan.EstimatedTime = 60 // ~1 minute to deploy dedicated instance
		plan.RAMRequired = 512  // Typical database RAM
		plan.StorageRequired = 5 // Typical database storage
	}

	return plan, nil
}

// createCachePlan creates a plan for provisioning cache
func (s *DependencyService) createCachePlan(
	dep models.RecipeDependency,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
) (ProvisionPlan, error) {
	engine := dep.Engine
	if engine == "" {
		engine = s.infraConfig.GetDefaultCacheEngine() // Default to valkey
	}

	plan := ProvisionPlan{
		Type:              "cache",
		Name:              fmt.Sprintf("%s Cache", engine),
		UseSharedInstance: dep.Shared,
		Config: map[string]string{
			"engine":     engine,
			"key_prefix": fmt.Sprintf("%s:", appRecipe.Slug),
		},
	}

	if dep.Shared {
		plan.Action = "configure_in_shared"
		plan.EstimatedTime = 10 // ~10 seconds to configure key prefix
		plan.RAMRequired = 0    // No additional RAM
		plan.StorageRequired = 0
	} else {
		plan.Action = "deploy"
		plan.EstimatedTime = 45 // ~45 seconds to deploy cache
		plan.RAMRequired = s.infraConfig.GetCacheRAM(engine)
		plan.StorageRequired = 1
	}

	return plan, nil
}

// createApplicationPlan creates a plan for deploying another application
func (s *DependencyService) createApplicationPlan(
	dep models.RecipeDependency,
	deviceID uuid.UUID,
) (ProvisionPlan, error) {
	if dep.Name == "" {
		return ProvisionPlan{}, fmt.Errorf("application dependency must specify name")
	}

	recipe, err := s.recipeLoader.GetRecipe(dep.Name)
	if err != nil {
		return ProvisionPlan{}, fmt.Errorf("failed to load application recipe: %w", err)
	}

	return ProvisionPlan{
		Type:            "application",
		Name:            recipe.Name,
		Action:          "deploy",
		UseSharedInstance: false,
		EstimatedTime:   recipe.SetupTimeMinutes * 60, // Convert to seconds
		RAMRequired:     recipe.GetEstimatedRAMMB(),
		StorageRequired: recipe.GetEstimatedStorageGB(),
		RecipeSlug:      dep.Name,
		Config: map[string]string{
			"purpose": dep.Purpose,
		},
	}, nil
}

// generateResourceBreakdown creates human-readable resource impact description
func (s *DependencyService) generateResourceBreakdown(plans []ProvisionPlan) string {
	if len(plans) == 0 {
		return "No additional resources required"
	}

	breakdown := "Resource requirements:\n"
	for _, plan := range plans {
		breakdown += fmt.Sprintf("- %s: ", plan.Name)
		if plan.UseSharedInstance {
			breakdown += "Uses shared instance (no additional RAM)\n"
		} else {
			breakdown += fmt.Sprintf("%dMB RAM, %dGB storage\n",
				plan.RAMRequired, plan.StorageRequired)
		}
	}

	return breakdown
}

// ProvisionDependencies provisions all dependencies in the plan
// This is called during deployment if user confirms dependency provisioning
func (s *DependencyService) ProvisionDependencies(
	ctx context.Context,
	result *DependencyCheckResult,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
	progressCallback func(step int, total int, message string),
) error {
	total := len(result.ToProvision)

	for i, plan := range result.ToProvision {
		// Send progress update
		if progressCallback != nil {
			progressCallback(i+1, total, fmt.Sprintf("Provisioning %s", plan.Name))
		}

		// Provision based on type
		switch plan.Type {
		case "reverse_proxy":
			err := s.provisionReverseProxy(ctx, plan, deviceID)
			if err != nil {
				return fmt.Errorf("failed to provision reverse proxy: %w", err)
			}

		case "database":
			err := s.provisionDatabase(ctx, plan, deviceID, appRecipe)
			if err != nil {
				return fmt.Errorf("failed to provision database: %w", err)
			}

		case "cache":
			err := s.provisionCache(ctx, plan, deviceID, appRecipe)
			if err != nil {
				return fmt.Errorf("failed to provision cache: %w", err)
			}

		case "application":
			err := s.provisionApplication(ctx, plan, deviceID)
			if err != nil {
				return fmt.Errorf("failed to provision application %s: %w", plan.Name, err)
			}

		default:
			return fmt.Errorf("unknown provision type: %s", plan.Type)
		}

		// Wait for provisioned resource to be healthy
		if plan.Action == "deploy" {
			if err := s.waitForHealthy(ctx, plan, deviceID); err != nil {
				// Check if error is due to context cancellation
				if ctx.Err() != nil {
					return fmt.Errorf("dependency provisioning cancelled by user: %w", ctx.Err())
				}
				return fmt.Errorf("dependency %s did not become healthy: %w", plan.Name, err)
			}
		}
	}

	return nil
}

// provisionReverseProxy deploys a reverse proxy using the orchestrator
func (s *DependencyService) provisionReverseProxy(
	ctx context.Context,
	plan ProvisionPlan,
	deviceID uuid.UUID,
) error {
	// Load the reverse proxy recipe
	recipe, err := s.recipeLoader.GetRecipe(plan.RecipeSlug)
	if err != nil {
		return fmt.Errorf("failed to load reverse proxy recipe %s: %w", plan.RecipeSlug, err)
	}

	// Get device
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Generate deployment directory and project name
	projectName := fmt.Sprintf("%s-dep-%s", plan.RecipeSlug, uuid.New().String()[:8])
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", projectName)
	host := device.GetSSHHost()

	// Create deployment spec
	spec := DeploymentSpec{
		Host:           host,
		StackName:      projectName,
		DeployDir:      deployDir,
		ComposeContent: recipe.ComposeContent,
		Timeout:        10 * time.Minute,
	}

	// Deploy using orchestrator
	if err := s.orchestrator.Deploy(ctx, spec); err != nil {
		return fmt.Errorf("orchestrator deployment failed: %w", err)
	}

	log.Printf("[DependencyService] Successfully deployed %s as dependency on device %s", recipe.Name, device.Name)
	return nil
}

// provisionDatabase provisions database (shared or dedicated)
func (s *DependencyService) provisionDatabase(
	ctx context.Context,
	plan ProvisionPlan,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
) error {
	engine := plan.Config["engine"]
	dbName := plan.Config["db_name"]

	if plan.UseSharedInstance {
		// Create database in shared instance
		return s.databasePool.ProvisionDatabaseInSharedInstance(
			ctx,
			deviceID,
			engine,
			dbName,
			appRecipe.Slug,
		)
	}

	// Deploy dedicated database instance
	// TODO: Implement dedicated database deployment
	return fmt.Errorf("dedicated database provisioning not yet implemented")
}

// provisionCache provisions cache (shared or dedicated)
func (s *DependencyService) provisionCache(
	ctx context.Context,
	plan ProvisionPlan,
	deviceID uuid.UUID,
	appRecipe *models.Recipe,
) error {
	engine := plan.Config["engine"]

	if plan.UseSharedInstance {
		// Create cache access in shared instance
		return s.cachePool.ProvisionCacheInSharedInstance(
			ctx,
			deviceID,
			engine,
			appRecipe.Slug,
		)
	}

	// Deploy dedicated cache instance
	// TODO: Implement dedicated cache deployment
	return fmt.Errorf("dedicated cache provisioning not yet implemented")
}

// provisionApplication deploys another application as a dependency using the orchestrator
func (s *DependencyService) provisionApplication(
	ctx context.Context,
	plan ProvisionPlan,
	deviceID uuid.UUID,
) error {
	// Load the dependency application recipe
	recipe, err := s.recipeLoader.GetRecipe(plan.RecipeSlug)
	if err != nil {
		return fmt.Errorf("failed to load application recipe %s: %w", plan.RecipeSlug, err)
	}

	// Check if this app itself has dependencies (prevent infinite recursion)
	// We allow depth of 1 (dependencies can have dependencies, but not beyond that)
	if len(recipe.Dependencies.Required) > 0 {
		log.Printf("[DependencyService] Warning: Dependency %s has its own dependencies - these will not be auto-provisioned (depth limit)", recipe.Name)
	}

	// Get device
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Generate deployment directory and project name
	projectName := fmt.Sprintf("%s-dep-%s", plan.RecipeSlug, uuid.New().String()[:8])
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", projectName)
	host := device.GetSSHHost()

	// Create deployment spec
	spec := DeploymentSpec{
		Host:           host,
		StackName:      projectName,
		DeployDir:      deployDir,
		ComposeContent: recipe.ComposeContent,
		Timeout:        15 * time.Minute,
	}

	// Deploy using orchestrator
	if err := s.orchestrator.Deploy(ctx, spec); err != nil {
		return fmt.Errorf("orchestrator deployment failed: %w", err)
	}

	log.Printf("[DependencyService] Successfully deployed %s as application dependency on device %s", recipe.Name, device.Name)
	return nil
}

// waitForHealthy waits for a provisioned resource to become healthy
func (s *DependencyService) waitForHealthy(
	ctx context.Context,
	plan ProvisionPlan,
	deviceID uuid.UUID,
) error {
	// Get device for SSH access
	device, err := s.deviceService.GetDevice(deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	host := device.GetSSHHost()

	// Project names follow the pattern: {recipe-slug}-dep-{uuid}
	// Use pattern matching to find containers with this prefix
	projectPrefix := fmt.Sprintf("%s-dep-", plan.RecipeSlug)

	// Calculate timeout with a maximum of 5 minutes
	timeout := time.Duration(plan.EstimatedTime) * time.Second
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}
	if timeout < 30*time.Second {
		timeout = 30 * time.Second // Minimum 30 seconds
	}

	// Poll for healthy status
	maxAttempts := int(timeout.Seconds() / 5) // Check every 5 seconds
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check container status using docker ps with pattern matching
		// Look for any containers with project labels matching our prefix
		checkCmd := fmt.Sprintf("docker ps --filter label=com.docker.compose.project --format '{{.Labels}}' | grep -o 'com.docker.compose.project=%s[^,]*' | head -1", projectPrefix)
		output, err := s.sshClient.ExecuteWithTimeout(host, checkCmd, 10*time.Second)

		if err == nil && len(output) > 0 {
			// Found a matching container, now check if it's running
			log.Printf("[DependencyService] Dependency %s is running (attempt %d/%d)", plan.Name, attempt+1, maxAttempts)

			// Give it an extra few seconds to fully initialize
			time.Sleep(5 * time.Second)
			return nil
		}

		log.Printf("[DependencyService] Waiting for %s to become healthy (attempt %d/%d)", plan.Name, attempt+1, maxAttempts)
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("dependency %s did not become healthy after %v", plan.Name, timeout)
}
