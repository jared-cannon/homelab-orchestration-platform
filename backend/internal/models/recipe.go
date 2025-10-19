package models

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Recipe represents a marketplace application recipe loaded from YAML
type Recipe struct {
	// Basic Information
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Slug        string `yaml:"slug" json:"slug"`
	Category    string `yaml:"category" json:"category"`
	Tagline     string `yaml:"tagline" json:"tagline"`
	Description string `yaml:"description" json:"description"`

	// Branding
	IconURL    string `yaml:"icon_url" json:"icon_url"`
	Author     string `yaml:"author" json:"author"`
	Website    string `yaml:"website" json:"website"`
	SourceCode string `yaml:"source_code" json:"source_code"`

	// Resource Requirements (for intelligent scheduler)
	Requirements RecipeRequirements `yaml:"requirements" json:"requirements"`

	// Docker Compose Content (standard format with ${VAR} env var substitution)
	ComposeContent string `yaml:"-" json:"-"` // Loaded from docker-compose.yaml file

	// User Configuration
	ConfigOptions []RecipeConfigOption `yaml:"config_options" json:"config_options"`

	// Database Provisioning (intelligent database pooling)
	Database RecipeDatabaseConfig `yaml:"database" json:"database"`

	// Cache Provisioning
	Cache RecipeCacheConfig `yaml:"cache" json:"cache"`

	// Volume Configuration
	Volumes map[string]RecipeVolumeConfig `yaml:"volumes" json:"volumes"`

	// Post-deployment automation
	PostInstall []RecipePostInstallStep `yaml:"post_install" json:"post_install"`

	// Health monitoring
	Health RecipeHealthConfig `yaml:"health" json:"health"`

	// Update configuration
	Updates RecipeUpdateConfig `yaml:"updates" json:"updates"`

	// Curated Marketplace Features (NEW)
	SaaSReplacements  []SaaSReplacement `yaml:"saas_replacements,omitempty" json:"saas_replacements,omitempty"`
	DifficultyLevel   string            `yaml:"difficulty_level,omitempty" json:"difficulty_level,omitempty"`     // "beginner", "intermediate", "advanced"
	SetupTimeMinutes  int               `yaml:"setup_time_minutes,omitempty" json:"setup_time_minutes,omitempty"` // Estimated setup time
	FeatureHighlights []string          `yaml:"feature_highlights,omitempty" json:"feature_highlights,omitempty"` // Key features for comparison tables
	IsInfrastructure  bool              `yaml:"is_infrastructure,omitempty" json:"is_infrastructure,omitempty"`   // Infrastructure template (e.g., laravel-app-server)
	ServerType        string            `yaml:"server_type,omitempty" json:"server_type,omitempty"`               // "app_server", "web_server", "database_server", "worker_server", "cache_server"

	// Dependency Auto-Provisioning (NEW)
	Dependencies RecipeDependencies `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`

	// Legacy field
	PostDeployInstructions string `yaml:"post_deploy_instructions,omitempty" json:"post_deploy_instructions,omitempty"`

	// Legacy Resources field for backward compatibility
	Resources RecipeResources `yaml:"resources,omitempty" json:"resources,omitempty"`

	// Legacy HealthCheck field
	HealthCheck RecipeHealthCheck `yaml:"health_check,omitempty" json:"health_check,omitempty"`

	// Metadata (not in YAML, populated by recipe sources)
	Metadata RecipeMetadata `yaml:"-" json:"metadata"`
}

// RecipeMetadata contains metadata about the recipe source and versioning
type RecipeMetadata struct {
	Source        string     `json:"source"`         // Recipe source (currently only "local")
	Version       string     `json:"version"`        // Recipe version
	LastUpdated   time.Time  `json:"last_updated"`   // When recipe was last updated
	UpdatedAt     time.Time  `json:"updated_at"`     // When we last fetched it
	SourceURL     string     `json:"source_url"`     // URL to source repository
	ImageVersion  string     `json:"image_version"`  // Latest Docker image version
	QualityScore  int        `json:"quality_score"`  // 0-100 quality score
	Verified      bool       `json:"verified"`       // Is this a verified/official recipe
	DeployCount   int        `json:"deploy_count"`   // How many times deployed (local tracking)
	SuccessRate   float64    `json:"success_rate"`   // Deployment success rate (0-1)
}

// RecipeResources defines the resource requirements for a recipe
type RecipeResources struct {
	MinRAMMB          int `yaml:"min_ram_mb" json:"min_ram_mb"`
	MinStorageGB      int `yaml:"min_storage_gb" json:"min_storage_gb"`
	RecommendedRAMMB  int `yaml:"recommended_ram_mb" json:"recommended_ram_mb"`
	RecommendedStorageGB int `yaml:"recommended_storage_gb" json:"recommended_storage_gb"`
	CPUCores          int `yaml:"cpu_cores" json:"cpu_cores"`
}

// RecipeConfigOption defines a user-configurable option
type RecipeConfigOption struct {
	Name        string      `yaml:"name" json:"name"`
	Label       string      `yaml:"label" json:"label"`
	Type        string      `yaml:"type" json:"type"` // string, number, boolean
	Default     interface{} `yaml:"default" json:"default"`
	Required    bool        `yaml:"required" json:"required"`
	Description string      `yaml:"description" json:"description"`
}

// RecipeHealthCheck defines health check parameters (legacy)
type RecipeHealthCheck struct {
	Path           string `yaml:"path" json:"path"`
	Port           int    `yaml:"port" json:"port"` // Port to check (defaults to 80 if not specified)
	ExpectedStatus int    `yaml:"expected_status" json:"expected_status"`
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"`
}

// RecipeRequirements defines resource requirements for intelligent scheduling
type RecipeRequirements struct {
	Memory struct {
		Minimum     string `yaml:"minimum" json:"minimum"`         // e.g., "512MB"
		Recommended string `yaml:"recommended" json:"recommended"` // e.g., "1GB"
	} `yaml:"memory" json:"memory"`
	Storage struct {
		Minimum     string `yaml:"minimum" json:"minimum"`         // e.g., "1GB"
		Recommended string `yaml:"recommended" json:"recommended"` // e.g., "5GB"
		Type        string `yaml:"type" json:"type"`               // "ssd", "hdd", "any"
	} `yaml:"storage" json:"storage"`
	CPU struct {
		MinimumCores     int `yaml:"minimum_cores" json:"minimum_cores"`
		RecommendedCores int `yaml:"recommended_cores" json:"recommended_cores"`
	} `yaml:"cpu" json:"cpu"`
	Reliability string `yaml:"reliability" json:"reliability"` // "high", "medium", "low"
	AlwaysOn    bool   `yaml:"always_on" json:"always_on"`
}

// RecipeDatabaseConfig defines database provisioning configuration
type RecipeDatabaseConfig struct {
	Engine        string `yaml:"engine" json:"engine"`                   // "postgres", "mysql", "mariadb", "sqlite", "none"
	AutoProvision bool   `yaml:"auto_provision" json:"auto_provision"`   // Enable automatic database provisioning
	Version       string `yaml:"version,omitempty" json:"version"`       // Database version (e.g., "15" for postgres)
	EnvPrefix     string `yaml:"env_prefix,omitempty" json:"env_prefix"` // Prefix for env vars (default: "DB_")
}

// RecipeCacheConfig defines cache provisioning configuration
type RecipeCacheConfig struct {
	Engine        string `yaml:"engine" json:"engine"`                               // "redis", "memcached", "none"
	AutoProvision bool   `yaml:"auto_provision" json:"auto_provision"`               // Enable automatic cache provisioning
	Version       string `yaml:"version,omitempty" json:"version"`                   // Cache version
	EnvPrefix     string `yaml:"env_prefix,omitempty" json:"env_prefix,omitempty"`   // Prefix for env vars (default: "REDIS_" or "MEMCACHED_")
}

// RecipeVolumeConfig defines volume configuration
type RecipeVolumeConfig struct {
	Description      string `yaml:"description" json:"description"`
	SizeEstimate     string `yaml:"size_estimate" json:"size_estimate"`         // e.g., "5GB"
	BackupPriority   string `yaml:"backup_priority" json:"backup_priority"`     // "high", "medium", "low"
	BackupFrequency  string `yaml:"backup_frequency" json:"backup_frequency"`   // "daily", "weekly", "monthly"
}

// RecipePostInstallStep defines a post-installation action
type RecipePostInstallStep struct {
	Type    string `yaml:"type" json:"type"`       // "message", "command", "webhook"
	Title   string `yaml:"title,omitempty" json:"title,omitempty"`
	Message string `yaml:"message,omitempty" json:"message,omitempty"`
	Command string `yaml:"command,omitempty" json:"command,omitempty"`
	URL     string `yaml:"url,omitempty" json:"url,omitempty"`
}

// RecipeHealthConfig defines health monitoring configuration
type RecipeHealthConfig struct {
	Endpoint           string `yaml:"endpoint" json:"endpoint"`                       // Health check HTTP path
	Interval           string `yaml:"interval" json:"interval"`                       // e.g., "30s"
	Timeout            string `yaml:"timeout" json:"timeout"`                         // e.g., "10s"
	UnhealthyThreshold int    `yaml:"unhealthy_threshold" json:"unhealthy_threshold"` // Failures before marking unhealthy
}

// RecipeUpdateConfig defines update behavior
type RecipeUpdateConfig struct {
	Strategy            string `yaml:"strategy" json:"strategy"`                           // "automatic", "manual", "notify"
	BackupBeforeUpdate  bool   `yaml:"backup_before_update" json:"backup_before_update"`   // Create backup before updating
	RollbackOnFailure   bool   `yaml:"rollback_on_failure" json:"rollback_on_failure"`     // Auto-rollback if update fails
}

// SaaSReplacement defines which SaaS service this recipe replaces
type SaaSReplacement struct {
	Name           string `yaml:"name" json:"name"`                         // e.g., "Google Photos"
	ComparisonURL  string `yaml:"comparison_url,omitempty" json:"comparison_url,omitempty"` // URL to comparison guide
}

// RecipeDependencies defines required and recommended dependencies
type RecipeDependencies struct {
	Required    []RecipeDependency `yaml:"required,omitempty" json:"required,omitempty"`
	Recommended []RecipeDependency `yaml:"recommended,omitempty" json:"recommended,omitempty"`
}

// RecipeDependency represents a single dependency
type RecipeDependency struct {
	Type          string   `yaml:"type" json:"type"`                                   // "reverse_proxy", "database", "cache", "application", "infrastructure"
	Name          string   `yaml:"name,omitempty" json:"name,omitempty"`               // Specific app name (for application dependencies)
	Engine        string   `yaml:"engine,omitempty" json:"engine,omitempty"`           // Database/cache engine (for database/cache dependencies)
	MinVersion    string   `yaml:"min_version,omitempty" json:"min_version,omitempty"` // Minimum version required
	Prefer        string   `yaml:"prefer,omitempty" json:"prefer,omitempty"`           // Preferred option (e.g., "traefik")
	Alternatives  []string `yaml:"alternatives,omitempty" json:"alternatives,omitempty"` // Alternative options
	Shared        bool     `yaml:"shared,omitempty" json:"shared,omitempty"`           // Use shared instance (default true for DB/cache)
	AutoProvision bool     `yaml:"auto_provision,omitempty" json:"auto_provision,omitempty"` // Auto-provision if missing (default true)
	AutoConfigure bool     `yaml:"auto_configure,omitempty" json:"auto_configure,omitempty"` // Auto-configure connection
	Purpose       string   `yaml:"purpose,omitempty" json:"purpose,omitempty"`         // Human-readable purpose
	Message       string   `yaml:"message,omitempty" json:"message,omitempty"`         // Custom message to show user
	ForVolumes    []string `yaml:"for_volumes,omitempty" json:"for_volumes,omitempty"` // Volumes to backup (for backup dependencies)
}

// Validate checks if the recipe configuration is valid
func (r *Recipe) Validate() error {
	// Validate basic fields
	if r.Name == "" {
		return fmt.Errorf("recipe name is required")
	}
	if r.Slug == "" {
		return fmt.Errorf("recipe slug is required")
	}

	// Validate memory requirements format
	if r.Requirements.Memory.Minimum != "" {
		if !isValidMemoryString(r.Requirements.Memory.Minimum) {
			return fmt.Errorf("invalid memory minimum format: %s (use format like '512MB' or '1GB')", r.Requirements.Memory.Minimum)
		}
	}
	if r.Requirements.Memory.Recommended != "" {
		if !isValidMemoryString(r.Requirements.Memory.Recommended) {
			return fmt.Errorf("invalid memory recommended format: %s (use format like '512MB' or '1GB')", r.Requirements.Memory.Recommended)
		}
	}

	// Validate storage requirements format
	if r.Requirements.Storage.Minimum != "" {
		if !isValidStorageString(r.Requirements.Storage.Minimum) {
			return fmt.Errorf("invalid storage minimum format: %s (use format like '1GB' or '100GB')", r.Requirements.Storage.Minimum)
		}
	}
	if r.Requirements.Storage.Recommended != "" {
		if !isValidStorageString(r.Requirements.Storage.Recommended) {
			return fmt.Errorf("invalid storage recommended format: %s (use format like '1GB' or '100GB')", r.Requirements.Storage.Recommended)
		}
	}

	// Validate storage type
	if r.Requirements.Storage.Type != "" {
		validTypes := []string{"ssd", "hdd", "any"}
		if !contains(validTypes, r.Requirements.Storage.Type) {
			return fmt.Errorf("invalid storage type: %s (must be 'ssd', 'hdd', or 'any')", r.Requirements.Storage.Type)
		}
	}

	// Validate CPU cores
	if r.Requirements.CPU.MinimumCores < 0 {
		return fmt.Errorf("minimum CPU cores cannot be negative: %d", r.Requirements.CPU.MinimumCores)
	}
	if r.Requirements.CPU.RecommendedCores < 0 {
		return fmt.Errorf("recommended CPU cores cannot be negative: %d", r.Requirements.CPU.RecommendedCores)
	}
	if r.Requirements.CPU.RecommendedCores > 0 && r.Requirements.CPU.MinimumCores > r.Requirements.CPU.RecommendedCores {
		return fmt.Errorf("minimum CPU cores (%d) cannot exceed recommended cores (%d)", r.Requirements.CPU.MinimumCores, r.Requirements.CPU.RecommendedCores)
	}

	// Validate reliability
	if r.Requirements.Reliability != "" {
		validReliability := []string{"high", "medium", "low"}
		if !contains(validReliability, r.Requirements.Reliability) {
			return fmt.Errorf("invalid reliability: %s (must be 'high', 'medium', or 'low')", r.Requirements.Reliability)
		}
	}

	// Validate database configuration
	if err := r.ValidateDatabaseConfig(); err != nil {
		return fmt.Errorf("database config: %w", err)
	}

	// Validate cache configuration
	if err := r.ValidateCacheConfig(); err != nil {
		return fmt.Errorf("cache config: %w", err)
	}

	// Validate curated marketplace fields
	if err := r.ValidateMarketplaceFields(); err != nil {
		return fmt.Errorf("marketplace fields: %w", err)
	}

	// Validate dependencies
	if err := r.ValidateDependencies(); err != nil {
		return fmt.Errorf("dependencies: %w", err)
	}

	return nil
}

// ValidateDatabaseConfig validates the database configuration
func (r *Recipe) ValidateDatabaseConfig() error {
	if r.Database.Engine == "" {
		return nil // No database configured
	}

	validEngines := []string{"postgres", "mysql", "mariadb", "sqlite", "none"}
	if !contains(validEngines, r.Database.Engine) {
		return fmt.Errorf("invalid database engine: %s (must be one of: %s)", r.Database.Engine, strings.Join(validEngines, ", "))
	}

	// If auto-provision is enabled, engine must not be "none"
	if r.Database.AutoProvision && r.Database.Engine == "none" {
		return fmt.Errorf("cannot auto-provision database with engine 'none'")
	}

	// Validate env prefix if provided
	if r.Database.EnvPrefix != "" {
		if !isValidEnvPrefix(r.Database.EnvPrefix) {
			return fmt.Errorf("invalid env_prefix: %s (must be uppercase letters and underscores, ending with underscore)", r.Database.EnvPrefix)
		}
	}

	return nil
}

// ValidateCacheConfig validates the cache configuration
func (r *Recipe) ValidateCacheConfig() error {
	if r.Cache.Engine == "" {
		return nil // No cache configured
	}

	validEngines := []string{"redis", "memcached", "none"}
	if !contains(validEngines, r.Cache.Engine) {
		return fmt.Errorf("invalid cache engine: %s (must be one of: %s)", r.Cache.Engine, strings.Join(validEngines, ", "))
	}

	// If auto-provision is enabled, engine must not be "none"
	if r.Cache.AutoProvision && r.Cache.Engine == "none" {
		return fmt.Errorf("cannot auto-provision cache with engine 'none'")
	}

	return nil
}

// ValidateMarketplaceFields validates curated marketplace fields
func (r *Recipe) ValidateMarketplaceFields() error {
	// Validate difficulty level
	if r.DifficultyLevel != "" {
		validLevels := []string{"beginner", "intermediate", "advanced"}
		if !contains(validLevels, r.DifficultyLevel) {
			return fmt.Errorf("invalid difficulty_level: %s (must be 'beginner', 'intermediate', or 'advanced')", r.DifficultyLevel)
		}
	}

	// Validate setup time
	if r.SetupTimeMinutes < 0 {
		return fmt.Errorf("setup_time_minutes cannot be negative: %d", r.SetupTimeMinutes)
	}

	// Validate server type (for infrastructure recipes)
	if r.ServerType != "" {
		validTypes := []string{"app_server", "web_server", "database_server", "worker_server", "cache_server"}
		if !contains(validTypes, r.ServerType) {
			return fmt.Errorf("invalid server_type: %s (must be one of: %s)", r.ServerType, strings.Join(validTypes, ", "))
		}
	}

	// If infrastructure recipe, server_type must be specified
	if r.IsInfrastructure && r.ServerType == "" {
		return fmt.Errorf("infrastructure recipes must specify server_type")
	}

	return nil
}

// ValidateDependencies validates dependency configuration
func (r *Recipe) ValidateDependencies() error {
	// Validate required dependencies
	for i, dep := range r.Dependencies.Required {
		if err := validateDependency(dep); err != nil {
			return fmt.Errorf("required dependency %d: %w", i, err)
		}
	}

	// Validate recommended dependencies
	for i, dep := range r.Dependencies.Recommended {
		if err := validateDependency(dep); err != nil {
			return fmt.Errorf("recommended dependency %d: %w", i, err)
		}
	}

	return nil
}

// validateDependency validates a single dependency configuration
func validateDependency(dep RecipeDependency) error {
	// Type is required
	if dep.Type == "" {
		return fmt.Errorf("dependency type is required")
	}

	// Validate dependency type
	validTypes := []string{"reverse_proxy", "database", "cache", "application", "infrastructure", "backup"}
	if !contains(validTypes, dep.Type) {
		return fmt.Errorf("invalid dependency type: %s (must be one of: %s)", dep.Type, strings.Join(validTypes, ", "))
	}

	// Type-specific validation
	switch dep.Type {
	case "database":
		if dep.Engine == "" && dep.Name == "" {
			return fmt.Errorf("database dependency must specify either engine or name")
		}
		if dep.Engine != "" {
			validEngines := []string{"postgres", "mysql", "mariadb", "sqlite"}
			if !contains(validEngines, dep.Engine) {
				return fmt.Errorf("invalid database engine: %s (must be one of: %s)", dep.Engine, strings.Join(validEngines, ", "))
			}
		}

	case "cache":
		if dep.Engine == "" && dep.Name == "" {
			return fmt.Errorf("cache dependency must specify either engine or name")
		}
		if dep.Engine != "" {
			validEngines := []string{"redis", "memcached"}
			if !contains(validEngines, dep.Engine) {
				return fmt.Errorf("invalid cache engine: %s (must be one of: %s)", dep.Engine, strings.Join(validEngines, ", "))
			}
		}

	case "reverse_proxy":
		if dep.Prefer == "" && len(dep.Alternatives) == 0 && dep.Name == "" {
			return fmt.Errorf("reverse_proxy dependency must specify prefer, alternatives, or name")
		}

	case "application", "infrastructure":
		if dep.Name == "" {
			return fmt.Errorf("%s dependency must specify name", dep.Type)
		}

	case "backup":
		// Backup dependencies are optional and don't require specific fields
	}

	return nil
}

// CalculateQualityScore computes a quality score (0-100) based on various factors
// This follows DRY principle by centralizing the scoring logic
func (r *Recipe) CalculateQualityScore() int {
	score := 0

	// GitHub stars from metadata (0-30 points, capped at 30k stars)
	if r.Metadata.QualityScore > 0 {
		// If already calculated externally, use it
		return r.Metadata.QualityScore
	}

	// Recency (0-20 points, based on last 6 months)
	if !r.Metadata.LastUpdated.IsZero() {
		daysSinceUpdate := int(time.Since(r.Metadata.LastUpdated).Hours() / 24)
		if daysSinceUpdate < 180 {
			score += int((180 - float64(daysSinceUpdate)) * 20 / 180)
		}
	} else {
		// If no update info, give partial credit
		score += 10
	}

	// Deployment success rate (0-15 points)
	score += int(r.Metadata.SuccessRate * 15)

	// Metadata completeness (0-10 points)
	completeness := 0
	if r.Description != "" {
		completeness += 2
	}
	if r.IconURL != "" {
		completeness += 2
	}
	if len(r.FeatureHighlights) >= 3 {
		completeness += 3
	}
	if len(r.SaaSReplacements) > 0 {
		completeness += 3
	}
	score += completeness

	// Difficulty bonus (0-5 points, easier = higher score for general users)
	switch r.DifficultyLevel {
	case "beginner":
		score += 5
	case "intermediate":
		score += 3
	case "advanced":
		score += 1
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// GetEstimatedRAMMB returns the minimum RAM requirement in MB
// Supports both new Requirements format and legacy Resources format
func (r *Recipe) GetEstimatedRAMMB() int {
	// Try new format first
	if r.Requirements.Memory.Minimum != "" {
		mb := parseMemoryString(r.Requirements.Memory.Minimum)
		if mb > 0 {
			return mb
		}
	}

	// Fall back to legacy format
	if r.Resources.MinRAMMB > 0 {
		return r.Resources.MinRAMMB
	}

	return 512 // Default minimum
}

// GetEstimatedStorageGB returns the minimum storage requirement in GB
func (r *Recipe) GetEstimatedStorageGB() int {
	// Try new format first
	if r.Requirements.Storage.Minimum != "" {
		gb := parseStorageString(r.Requirements.Storage.Minimum)
		if gb > 0 {
			return gb
		}
	}

	// Fall back to legacy format
	if r.Resources.MinStorageGB > 0 {
		return r.Resources.MinStorageGB
	}

	return 1 // Default minimum
}

// Helper functions

// parseMemoryString converts memory string like "512MB" or "1GB" to MB
func parseMemoryString(s string) int {
	if s == "" {
		return 0
	}

	// Extract number and unit
	var value int
	var unit string
	fmt.Sscanf(s, "%d%s", &value, &unit)

	switch strings.ToUpper(unit) {
	case "GB":
		return value * 1024
	case "MB":
		return value
	default:
		return 0
	}
}

// parseStorageString converts storage string like "1GB" or "1TB" to GB
func parseStorageString(s string) int {
	if s == "" {
		return 0
	}

	// Extract number and unit
	var value int
	var unit string
	fmt.Sscanf(s, "%d%s", &value, &unit)

	switch strings.ToUpper(unit) {
	case "TB":
		return value * 1024
	case "GB":
		return value
	default:
		return 0
	}
}

func isValidMemoryString(s string) bool {
	// Match formats like "512MB", "1GB", "2048MB"
	matched, _ := regexp.MatchString(`^[0-9]+(?:MB|GB)$`, s)
	return matched
}

func isValidStorageString(s string) bool {
	// Match formats like "1GB", "100GB", "1TB"
	matched, _ := regexp.MatchString(`^[0-9]+(?:GB|TB)$`, s)
	return matched
}

func isValidEnvPrefix(s string) bool {
	// Match formats like "DB_", "POSTGRES_", "MYSQL_"
	matched, _ := regexp.MatchString(`^[A-Z_]+_$`, s)
	return matched
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
