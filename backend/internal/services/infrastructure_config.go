package services

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// InfrastructureConfig holds configuration for infrastructure components
// Loaded from infrastructure-defaults.yaml to avoid hardcoding versions
type InfrastructureConfig struct {
	Databases      map[string]DatabaseConfig      `yaml:"databases"`
	Caches         map[string]CacheConfig         `yaml:"caches"`
	ReverseProxies map[string]ReverseProxyConfig  `yaml:"reverse_proxies"`
	Orchestration  OrchestrationConfig            `yaml:"orchestration"`
	Version        string                         `yaml:"version"`
	LastUpdated    string                         `yaml:"last_updated"`
	Notes          string                         `yaml:"notes"`
}

// OrchestrationConfig holds configuration for container orchestration
type OrchestrationConfig struct {
	Mode         string `yaml:"mode"`          // "compose" or "swarm"
	SwarmEnabled bool   `yaml:"swarm_enabled"` // Enable Swarm-specific features
	Description  string `yaml:"description"`
}

// DatabaseConfig holds configuration for a database engine
type DatabaseConfig struct {
	DefaultVersion     string `yaml:"default_version"`
	DockerImage        string `yaml:"docker_image"`
	Port               int    `yaml:"port"`
	InternalPort       int    `yaml:"internal_port"`
	EstimatedRAMMB     int    `yaml:"estimated_ram_mb"`
	MasterUsername     string `yaml:"master_username"`
	HealthCheckCommand string `yaml:"health_check_command"`
	Description        string `yaml:"description"`
}

// CacheConfig holds configuration for a cache engine
type CacheConfig struct {
	DefaultVersion    string `yaml:"default_version"`
	DockerImage       string `yaml:"docker_image"`
	Port              int    `yaml:"port"`
	EstimatedRAMMB    int    `yaml:"estimated_ram_mb"`
	SupportsDatabases bool   `yaml:"supports_databases"`
	MaxDatabases      int    `yaml:"max_databases"`
	IsDefault         bool   `yaml:"is_default"`
	Description       string `yaml:"description"`
}

// ReverseProxyConfig holds configuration for a reverse proxy
type ReverseProxyConfig struct {
	DefaultVersion     string `yaml:"default_version"`
	DockerImage        string `yaml:"docker_image"`
	EstimatedRAMMB     int    `yaml:"estimated_ram_mb"`
	EstimatedStorageGB int    `yaml:"estimated_storage_gb"`
	IsDefault          bool   `yaml:"is_default"`
	Description        string `yaml:"description"`
}

// LoadInfrastructureConfig loads configuration from YAML file
func LoadInfrastructureConfig(configPath string) (*InfrastructureConfig, error) {
	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read infrastructure config: %w", err)
	}

	// Parse YAML
	var config InfrastructureConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse infrastructure config: %w", err)
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid infrastructure config: %w", err)
	}

	return &config, nil
}

// Validate ensures the configuration is valid
func (ic *InfrastructureConfig) Validate() error {
	// Check that at least one database engine exists
	if len(ic.Databases) == 0 {
		return fmt.Errorf("no database engines configured")
	}

	// Check that at least one cache engine exists
	if len(ic.Caches) == 0 {
		return fmt.Errorf("no cache engines configured")
	}

	// Check that at least one reverse proxy exists
	if len(ic.ReverseProxies) == 0 {
		return fmt.Errorf("no reverse proxies configured")
	}

	// Validate each database config
	for engine, dbConfig := range ic.Databases {
		if dbConfig.DefaultVersion == "" {
			return fmt.Errorf("database %s: default_version is required", engine)
		}
		if dbConfig.DockerImage == "" {
			return fmt.Errorf("database %s: docker_image is required", engine)
		}
		if dbConfig.Port == 0 {
			return fmt.Errorf("database %s: port is required", engine)
		}
		// Validate port range
		if dbConfig.Port < 1 || dbConfig.Port > 65535 {
			return fmt.Errorf("database %s: port %d out of valid range (1-65535)", engine, dbConfig.Port)
		}
		// Validate RAM estimation
		if dbConfig.EstimatedRAMMB < 0 {
			return fmt.Errorf("database %s: estimated_ram_mb cannot be negative", engine)
		}
		// Validate internal port if specified
		if dbConfig.InternalPort != 0 && (dbConfig.InternalPort < 1 || dbConfig.InternalPort > 65535) {
			return fmt.Errorf("database %s: internal_port %d out of valid range (1-65535)", engine, dbConfig.InternalPort)
		}
	}

	// Validate each cache config
	hasDefaultCache := false
	defaultCacheCount := 0
	for engine, cacheConfig := range ic.Caches {
		if cacheConfig.DefaultVersion == "" {
			return fmt.Errorf("cache %s: default_version is required", engine)
		}
		if cacheConfig.DockerImage == "" {
			return fmt.Errorf("cache %s: docker_image is required", engine)
		}
		if cacheConfig.Port == 0 {
			return fmt.Errorf("cache %s: port is required", engine)
		}
		// Validate port range
		if cacheConfig.Port < 1 || cacheConfig.Port > 65535 {
			return fmt.Errorf("cache %s: port %d out of valid range (1-65535)", engine, cacheConfig.Port)
		}
		// Validate RAM estimation
		if cacheConfig.EstimatedRAMMB < 0 {
			return fmt.Errorf("cache %s: estimated_ram_mb cannot be negative", engine)
		}
		// Validate database support configuration
		if cacheConfig.SupportsDatabases && cacheConfig.MaxDatabases <= 0 {
			return fmt.Errorf("cache %s: supports_databases is true but max_databases is not set", engine)
		}
		if cacheConfig.IsDefault {
			hasDefaultCache = true
			defaultCacheCount++
		}
	}
	if !hasDefaultCache {
		return fmt.Errorf("no default cache engine configured (set is_default: true)")
	}
	if defaultCacheCount > 1 {
		return fmt.Errorf("multiple default cache engines configured (only one should have is_default: true)")
	}

	// Validate each reverse proxy config
	hasDefaultProxy := false
	defaultProxyCount := 0
	for name, proxyConfig := range ic.ReverseProxies {
		if proxyConfig.DefaultVersion == "" {
			return fmt.Errorf("reverse proxy %s: default_version is required", name)
		}
		if proxyConfig.DockerImage == "" {
			return fmt.Errorf("reverse proxy %s: docker_image is required", name)
		}
		// Validate RAM estimation
		if proxyConfig.EstimatedRAMMB < 0 {
			return fmt.Errorf("reverse proxy %s: estimated_ram_mb cannot be negative", name)
		}
		// Validate storage estimation
		if proxyConfig.EstimatedStorageGB < 0 {
			return fmt.Errorf("reverse proxy %s: estimated_storage_gb cannot be negative", name)
		}
		if proxyConfig.IsDefault {
			hasDefaultProxy = true
			defaultProxyCount++
		}
	}
	if !hasDefaultProxy {
		return fmt.Errorf("no default reverse proxy configured (set is_default: true)")
	}
	if defaultProxyCount > 1 {
		return fmt.Errorf("multiple default reverse proxies configured (only one should have is_default: true)")
	}

	// Validate orchestration config
	if ic.Orchestration.Mode != "" && ic.Orchestration.Mode != "compose" && ic.Orchestration.Mode != "swarm" {
		return fmt.Errorf("orchestration mode must be 'compose' or 'swarm', got: %s", ic.Orchestration.Mode)
	}

	return nil
}

// Database helper methods

// GetDatabaseConfig returns configuration for a database engine
func (ic *InfrastructureConfig) GetDatabaseConfig(engine string) (DatabaseConfig, error) {
	config, exists := ic.Databases[engine]
	if !exists {
		return DatabaseConfig{}, fmt.Errorf("unsupported database engine: %s", engine)
	}
	return config, nil
}

// GetDatabaseVersion returns the default version for a database engine
func (ic *InfrastructureConfig) GetDatabaseVersion(engine string) string {
	if config, exists := ic.Databases[engine]; exists {
		return config.DefaultVersion
	}
	return "latest" // Fallback
}

// GetDatabasePort returns the port for a database engine
func (ic *InfrastructureConfig) GetDatabasePort(engine string) int {
	if config, exists := ic.Databases[engine]; exists {
		return config.Port
	}
	return 5432 // Fallback to postgres default
}

// GetDatabaseRAM returns estimated RAM for a database engine
func (ic *InfrastructureConfig) GetDatabaseRAM(engine string) int {
	if config, exists := ic.Databases[engine]; exists {
		return config.EstimatedRAMMB
	}
	return 256 // Fallback
}

// GetDatabaseImage returns Docker image for a database engine
func (ic *InfrastructureConfig) GetDatabaseImage(engine string) string {
	if config, exists := ic.Databases[engine]; exists {
		return config.DockerImage
	}
	return engine // Fallback to engine name
}

// GetMasterUsername returns master username for a database engine
func (ic *InfrastructureConfig) GetMasterUsername(engine string) string {
	if config, exists := ic.Databases[engine]; exists {
		return config.MasterUsername
	}
	return "root" // Fallback
}

// ValidateDatabaseEngine checks if a database engine is supported
func (ic *InfrastructureConfig) ValidateDatabaseEngine(engine string) error {
	if _, exists := ic.Databases[engine]; !exists {
		return fmt.Errorf("unsupported database engine: %s", engine)
	}
	return nil
}

// Cache helper methods

// GetCacheConfig returns configuration for a cache engine
func (ic *InfrastructureConfig) GetCacheConfig(engine string) (CacheConfig, error) {
	config, exists := ic.Caches[engine]
	if !exists {
		return CacheConfig{}, fmt.Errorf("unsupported cache engine: %s", engine)
	}
	return config, nil
}

// GetDefaultCacheEngine returns the default cache engine
func (ic *InfrastructureConfig) GetDefaultCacheEngine() string {
	for engine, config := range ic.Caches {
		if config.IsDefault {
			return engine
		}
	}
	return "valkey" // Fallback to valkey
}

// GetCacheVersion returns the default version for a cache engine
func (ic *InfrastructureConfig) GetCacheVersion(engine string) string {
	if config, exists := ic.Caches[engine]; exists {
		return config.DefaultVersion
	}
	return "latest" // Fallback
}

// GetCachePort returns the port for a cache engine
func (ic *InfrastructureConfig) GetCachePort(engine string) int {
	if config, exists := ic.Caches[engine]; exists {
		return config.Port
	}
	return 6379 // Fallback to redis/valkey default
}

// GetCacheRAM returns estimated RAM for a cache engine
func (ic *InfrastructureConfig) GetCacheRAM(engine string) int {
	if config, exists := ic.Caches[engine]; exists {
		return config.EstimatedRAMMB
	}
	return 128 // Fallback
}

// GetCacheImage returns Docker image for a cache engine
func (ic *InfrastructureConfig) GetCacheImage(engine string) string {
	if config, exists := ic.Caches[engine]; exists {
		return config.DockerImage
	}
	return engine // Fallback to engine name
}

// SupportsDatabases checks if a cache engine supports database numbers
func (ic *InfrastructureConfig) SupportsDatabases(engine string) bool {
	if config, exists := ic.Caches[engine]; exists {
		return config.SupportsDatabases
	}
	return false
}

// GetMaxDatabases returns maximum database numbers for a cache engine
func (ic *InfrastructureConfig) GetMaxDatabases(engine string) int {
	if config, exists := ic.Caches[engine]; exists {
		return config.MaxDatabases
	}
	return 0
}

// ValidateCacheEngine checks if a cache engine is supported
func (ic *InfrastructureConfig) ValidateCacheEngine(engine string) error {
	if _, exists := ic.Caches[engine]; !exists {
		return fmt.Errorf("unsupported cache engine: %s", engine)
	}
	return nil
}

// Reverse Proxy helper methods

// GetReverseProxyConfig returns configuration for a reverse proxy
func (ic *InfrastructureConfig) GetReverseProxyConfig(name string) (ReverseProxyConfig, error) {
	config, exists := ic.ReverseProxies[name]
	if !exists {
		return ReverseProxyConfig{}, fmt.Errorf("unsupported reverse proxy: %s", name)
	}
	return config, nil
}

// GetDefaultReverseProxy returns the default reverse proxy
func (ic *InfrastructureConfig) GetDefaultReverseProxy() string {
	for name, config := range ic.ReverseProxies {
		if config.IsDefault {
			return name
		}
	}
	return "traefik" // Fallback
}

// GetReverseProxyVersion returns the default version for a reverse proxy
func (ic *InfrastructureConfig) GetReverseProxyVersion(name string) string {
	if config, exists := ic.ReverseProxies[name]; exists {
		return config.DefaultVersion
	}
	return "latest" // Fallback
}

// GetReverseProxyRAM returns estimated RAM for a reverse proxy
func (ic *InfrastructureConfig) GetReverseProxyRAM(name string) int {
	if config, exists := ic.ReverseProxies[name]; exists {
		return config.EstimatedRAMMB
	}
	return 64 // Fallback
}

// GetReverseProxyStorage returns estimated storage for a reverse proxy
func (ic *InfrastructureConfig) GetReverseProxyStorage(name string) int {
	if config, exists := ic.ReverseProxies[name]; exists {
		return config.EstimatedStorageGB
	}
	return 1 // Fallback
}

// GetReverseProxyImage returns Docker image for a reverse proxy
func (ic *InfrastructureConfig) GetReverseProxyImage(name string) string {
	if config, exists := ic.ReverseProxies[name]; exists {
		return config.DockerImage
	}
	return name // Fallback to proxy name
}

// ValidateReverseProxy checks if a reverse proxy is supported
func (ic *InfrastructureConfig) ValidateReverseProxy(name string) error {
	if _, exists := ic.ReverseProxies[name]; !exists {
		return fmt.Errorf("unsupported reverse proxy: %s", name)
	}
	return nil
}

// Orchestration helper methods

// GetOrchestrationMode returns the orchestration mode
func (ic *InfrastructureConfig) GetOrchestrationMode() string {
	if ic.Orchestration.Mode == "" {
		return "compose" // Default to compose
	}
	return ic.Orchestration.Mode
}

// IsSwarmEnabled returns whether Swarm features are enabled
func (ic *InfrastructureConfig) IsSwarmEnabled() bool {
	return ic.Orchestration.SwarmEnabled
}

// GetOrchestratorConfig returns the orchestrator configuration
func (ic *InfrastructureConfig) GetOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		Mode:         ic.GetOrchestrationMode(),
		SwarmEnabled: ic.IsSwarmEnabled(),
	}
}
