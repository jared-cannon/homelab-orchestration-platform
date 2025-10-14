package models

import "time"

// Recipe represents a marketplace application recipe loaded from YAML
type Recipe struct {
	ID                     string               `yaml:"id" json:"id"`
	Name                   string               `yaml:"name" json:"name"`
	Slug                   string               `yaml:"slug" json:"slug"`
	Category               string               `yaml:"category" json:"category"`
	Tagline                string               `yaml:"tagline" json:"tagline"`
	Description            string               `yaml:"description" json:"description"`
	IconURL                string               `yaml:"icon_url" json:"icon_url"`
	Resources              RecipeResources      `yaml:"resources" json:"resources"`
	ComposeTemplate        string               `yaml:"compose_template" json:"compose_template"`
	ConfigOptions          []RecipeConfigOption `yaml:"config_options" json:"config_options"`
	PostDeployInstructions string               `yaml:"post_deploy_instructions" json:"post_deploy_instructions"`
	HealthCheck            RecipeHealthCheck    `yaml:"health_check" json:"health_check"`

	// Metadata (not in YAML, populated by recipe sources)
	Metadata RecipeMetadata `yaml:"-" json:"metadata"`
}

// RecipeMetadata contains metadata about the recipe source and versioning
type RecipeMetadata struct {
	Source        string     `json:"source"`         // "local", "coolify", "portainer", etc.
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

// RecipeHealthCheck defines health check parameters
type RecipeHealthCheck struct {
	Path           string `yaml:"path" json:"path"`
	Port           int    `yaml:"port" json:"port"` // Port to check (defaults to 80 if not specified)
	ExpectedStatus int    `yaml:"expected_status" json:"expected_status"`
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"`
}
