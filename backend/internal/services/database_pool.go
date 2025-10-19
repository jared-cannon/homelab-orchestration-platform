package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DatabasePoolManager manages shared database instances and provisions isolated databases
type DatabasePoolManager struct {
	db             *gorm.DB
	sshClient      *ssh.Client
	credService    *CredentialService
	infraConfig    *InfrastructureConfig
	orchestrator   ContainerOrchestrator
}

// NewDatabasePoolManager creates a new database pool manager
func NewDatabasePoolManager(db *gorm.DB, sshClient *ssh.Client, credService *CredentialService, infraConfig *InfrastructureConfig, orchestrator ContainerOrchestrator) *DatabasePoolManager {
	return &DatabasePoolManager{
		db:           db,
		sshClient:    sshClient,
		credService:  credService,
		infraConfig:  infraConfig,
		orchestrator: orchestrator,
	}
}

// GetOrCreateSharedInstance ensures a shared database instance exists on a device
// Returns the shared instance, creating and deploying it if necessary
func (dpm *DatabasePoolManager) GetOrCreateSharedInstance(device *models.Device, engine string, version string) (*models.SharedDatabaseInstance, error) {
	// Validate engine using infrastructure config
	if err := dpm.infraConfig.ValidateDatabaseEngine(engine); err != nil {
		return nil, err
	}

	// Check if shared instance already exists for this device and engine
	var instance models.SharedDatabaseInstance
	err := dpm.db.Where("device_id = ? AND engine = ?", device.ID, engine).First(&instance).Error

	if err == nil {
		// Instance exists
		log.Printf("[DatabasePool] Found existing %s instance on device %s", engine, device.Name)
		return &instance, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to query shared instance: %w", err)
	}

	// Instance doesn't exist - create it
	log.Printf("[DatabasePool] No %s instance found on %s, creating new shared instance", engine, device.Name)
	return dpm.createSharedInstance(device, engine, version)
}

// createSharedInstance creates and deploys a new shared database instance
func (dpm *DatabasePoolManager) createSharedInstance(device *models.Device, engine string, version string) (*models.SharedDatabaseInstance, error) {
	// Generate master credentials
	masterUsername := dpm.infraConfig.GetMasterUsername(engine)
	masterPassword, err := dpm.generateSecurePassword(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate master password: %w", err)
	}

	// Store master credentials
	credKey := fmt.Sprintf("shared-%s-%s", engine, device.ID.String())
	if err := dpm.credService.StoreCredential(credKey, masterPassword); err != nil {
		return nil, fmt.Errorf("failed to store master credentials: %w", err)
	}

	// Determine version from config if not specified
	if version == "" {
		version = dpm.infraConfig.GetDatabaseVersion(engine)
	}

	// Get configuration for this database engine
	dbConfig, err := dpm.infraConfig.GetDatabaseConfig(engine)
	if err != nil {
		return nil, fmt.Errorf("failed to get database config: %w", err)
	}

	// Create database record
	instance := &models.SharedDatabaseInstance{
		DeviceID:       device.ID,
		Engine:         engine,
		Version:        version,
		Status:         "provisioning",
		ContainerName:  fmt.Sprintf("homelab-%s-shared", engine),
		ComposeProject: fmt.Sprintf("homelab-%s-shared", engine),
		Port:           dbConfig.Port,
		InternalPort:   dbConfig.InternalPort,
		MasterUsername: masterUsername,
		CredentialKey:  credKey,
		EstimatedRAMMB: dbConfig.EstimatedRAMMB,
		DatabaseCount:  0,
	}

	if err := dpm.db.Create(instance).Error; err != nil {
		return nil, fmt.Errorf("failed to create shared instance record: %w", err)
	}

	// Deploy the shared database container
	if err := dpm.deploySharedInstance(device, instance, masterPassword); err != nil {
		// Mark as failed
		instance.Status = "failed"
		instance.ErrorDetails = err.Error()
		dpm.db.Save(instance)
		return nil, fmt.Errorf("failed to deploy shared instance: %w", err)
	}

	// Mark as running
	now := time.Now()
	instance.Status = "running"
	instance.DeployedAt = &now
	dpm.db.Save(instance)

	log.Printf("[DatabasePool] Successfully deployed shared %s instance on %s", engine, device.Name)
	return instance, nil
}

// deploySharedInstance deploys the Docker container for a shared database instance
func (dpm *DatabasePoolManager) deploySharedInstance(device *models.Device, instance *models.SharedDatabaseInstance, masterPassword string) error {
	host := device.GetSSHHost()
	deployDir := fmt.Sprintf("~/homelab-deployments/%s", instance.ComposeProject)

	// Generate docker-compose.yaml for the shared instance
	composeContent, err := dpm.generateSharedInstanceCompose(instance, masterPassword)
	if err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	// Create deployment spec
	spec := DeploymentSpec{
		Host:           host,
		StackName:      instance.ComposeProject,
		DeployDir:      deployDir,
		ComposeContent: composeContent,
		Timeout:        10 * time.Minute,
	}

	// Deploy using orchestrator
	ctx := context.Background()
	if err := dpm.orchestrator.Deploy(ctx, spec); err != nil {
		return fmt.Errorf("orchestrator deployment failed: %w", err)
	}

	// Wait for database to be ready
	if err := dpm.orchestrator.WaitForHealthy(ctx, instance.ComposeProject, host, 5*time.Minute); err != nil {
		return fmt.Errorf("database failed to become healthy: %w", err)
	}

	log.Printf("[DatabasePool] Shared %s instance deployed successfully on %s", instance.Engine, device.GetPrimaryAddress())
	return nil
}

// generateSharedInstanceCompose generates a docker-compose.yaml for a shared database instance
func (dpm *DatabasePoolManager) generateSharedInstanceCompose(instance *models.SharedDatabaseInstance, masterPassword string) (string, error) {
	switch instance.Engine {
	case "postgres":
		return dpm.generatePostgresCompose(instance, masterPassword), nil
	case "mysql":
		return dpm.generateMySQLCompose(instance, masterPassword), nil
	case "mariadb":
		return dpm.generateMariaDBCompose(instance, masterPassword), nil
	default:
		return "", fmt.Errorf("unsupported database engine: %s", instance.Engine)
	}
}

// generatePostgresCompose generates docker-compose for shared Postgres instance
func (dpm *DatabasePoolManager) generatePostgresCompose(instance *models.SharedDatabaseInstance, masterPassword string) string {
	dockerImage := dpm.infraConfig.GetDatabaseImage("postgres")
	return fmt.Sprintf(`version: '3.8'
services:
  postgres:
    image: %s:%s
    container_name: %s
    restart: unless-stopped
    environment:
      POSTGRES_USER: %s
      POSTGRES_PASSWORD: %s
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "%d:%d"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U %s"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres-data:
    driver: local
`, dockerImage, instance.Version, instance.ContainerName, instance.MasterUsername, masterPassword, instance.Port, instance.InternalPort, instance.MasterUsername)
}

// generateMySQLCompose generates docker-compose for shared MySQL instance
func (dpm *DatabasePoolManager) generateMySQLCompose(instance *models.SharedDatabaseInstance, masterPassword string) string {
	dockerImage := dpm.infraConfig.GetDatabaseImage("mysql")
	return fmt.Sprintf(`version: '3.8'
services:
  mysql:
    image: %s:%s
    container_name: %s
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: %s
    volumes:
      - mysql-data:/var/lib/mysql
    ports:
      - "%d:%d"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-p%s"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mysql-data:
    driver: local
`, dockerImage, instance.Version, instance.ContainerName, masterPassword, instance.Port, instance.InternalPort, masterPassword)
}

// generateMariaDBCompose generates docker-compose for shared MariaDB instance
func (dpm *DatabasePoolManager) generateMariaDBCompose(instance *models.SharedDatabaseInstance, masterPassword string) string {
	dockerImage := dpm.infraConfig.GetDatabaseImage("mariadb")
	return fmt.Sprintf(`version: '3.8'
services:
  mariadb:
    image: %s:%s
    container_name: %s
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: %s
    volumes:
      - mariadb-data:/var/lib/mysql
    ports:
      - "%d:%d"
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-p%s"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mariadb-data:
    driver: local
`, dockerImage, instance.Version, instance.ContainerName, masterPassword, instance.Port, instance.InternalPort, masterPassword)
}

// ProvisionDatabase creates an isolated database within a shared instance for a deployment
func (dpm *DatabasePoolManager) ProvisionDatabase(deployment *models.Deployment, device *models.Device, dbConfig models.RecipeDatabaseConfig) (*models.ProvisionedDatabase, error) {
	// Get or create shared instance
	instance, err := dpm.GetOrCreateSharedInstance(device, dbConfig.Engine, dbConfig.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared instance: %w", err)
	}

	if instance.Status != "running" {
		return nil, fmt.Errorf("shared instance is not running (status: %s)", instance.Status)
	}

	// Generate database name and username
	dbName := dpm.generateDatabaseName(deployment.RecipeSlug, deployment.ID)
	username := dpm.generateUsername(deployment.RecipeSlug)
	password, err := dpm.generateSecurePassword(24)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// Store credentials
	credKey := fmt.Sprintf("db-%s", deployment.ID.String())
	if err := dpm.credService.StoreCredential(credKey, password); err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}

	// Create provisioned database record
	provisionedDB := &models.ProvisionedDatabase{
		SharedDatabaseInstanceID: instance.ID,
		DeploymentID:             deployment.ID,
		DatabaseName:             dbName,
		Username:                 username,
		CredentialKey:            credKey,
		Host:                     device.GetPrimaryAddress(),
		Port:                     instance.Port,
		Status:                   "provisioning",
	}

	if err := dpm.db.Create(provisionedDB).Error; err != nil {
		return nil, fmt.Errorf("failed to create provisioned database record: %w", err)
	}

	// Create database and user in the shared instance
	if err := dpm.createDatabaseAndUser(device, instance, dbName, username, password); err != nil {
		provisionedDB.Status = "failed"
		provisionedDB.ErrorDetails = err.Error()
		dpm.db.Save(provisionedDB)
		return nil, fmt.Errorf("failed to create database and user: %w", err)
	}

	// Mark as ready
	now := time.Now()
	provisionedDB.Status = "ready"
	provisionedDB.ProvisionedAt = &now
	dpm.db.Save(provisionedDB)

	// Update shared instance database count
	dpm.db.Model(instance).Update("database_count", gorm.Expr("database_count + 1"))

	log.Printf("[DatabasePool] Provisioned database %s for deployment %s", dbName, deployment.ID)
	return provisionedDB, nil
}

// createDatabaseAndUser creates a database and user in the shared instance
func (dpm *DatabasePoolManager) createDatabaseAndUser(device *models.Device, instance *models.SharedDatabaseInstance, dbName, username, password string) error {
	host := device.GetSSHHost()

	// Get master password
	masterPassword, err := dpm.credService.GetCredential(instance.CredentialKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve master password: %w", err)
	}

	var createCmd string
	switch instance.Engine {
	case "postgres":
		createCmd = dpm.generatePostgresCreateCommands(instance, dbName, username, password, masterPassword)
	case "mysql", "mariadb":
		createCmd = dpm.generateMySQLCreateCommands(instance, dbName, username, password, masterPassword)
	default:
		return fmt.Errorf("unsupported engine: %s", instance.Engine)
	}

	output, err := dpm.sshClient.ExecuteWithTimeout(host, createCmd, 30*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create database: %w (output: %s)", err, output)
	}

	log.Printf("[DatabasePool] Created database %s and user %s in shared %s instance", dbName, username, instance.Engine)
	return nil
}

// generatePostgresCreateCommands generates SQL commands to create database and user in Postgres
func (dpm *DatabasePoolManager) generatePostgresCreateCommands(instance *models.SharedDatabaseInstance, dbName, username, password, masterPassword string) string {
	return fmt.Sprintf(`docker exec %s psql -U %s -c "CREATE DATABASE %s;" && \
docker exec %s psql -U %s -c "CREATE USER %s WITH PASSWORD '%s';" && \
docker exec %s psql -U %s -c "GRANT ALL PRIVILEGES ON DATABASE %s TO %s;"`,
		instance.ContainerName, instance.MasterUsername, dbName,
		instance.ContainerName, instance.MasterUsername, username, password,
		instance.ContainerName, instance.MasterUsername, dbName, username)
}

// generateMySQLCreateCommands generates SQL commands to create database and user in MySQL/MariaDB
func (dpm *DatabasePoolManager) generateMySQLCreateCommands(instance *models.SharedDatabaseInstance, dbName, username, password, masterPassword string) string {
	return fmt.Sprintf(`docker exec %s mysql -u%s -p%s -e "CREATE DATABASE %s;" && \
docker exec %s mysql -u%s -p%s -e "CREATE USER '%s'@'%%' IDENTIFIED BY '%s';" && \
docker exec %s mysql -u%s -p%s -e "GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%'; FLUSH PRIVILEGES;"`,
		instance.ContainerName, instance.MasterUsername, masterPassword, dbName,
		instance.ContainerName, instance.MasterUsername, masterPassword, username, password,
		instance.ContainerName, instance.MasterUsername, masterPassword, dbName, username)
}

// Helper methods

func (dpm *DatabasePoolManager) generateDatabaseName(recipeSlug string, deploymentID uuid.UUID) string {
	// Generate a database name like: nextcloud_abc123de
	shortID := deploymentID.String()[:8]
	// Replace hyphens with underscores for database name compatibility
	sanitized := strings.ReplaceAll(recipeSlug, "-", "_")
	return fmt.Sprintf("%s_%s", sanitized, shortID)
}

func (dpm *DatabasePoolManager) generateUsername(recipeSlug string) string {
	// Generate a username like: nextcloud_user
	sanitized := strings.ReplaceAll(recipeSlug, "-", "_")
	return fmt.Sprintf("%s_user", sanitized)
}

func (dpm *DatabasePoolManager) generateSecurePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GetProvisionedDatabase retrieves a provisioned database for a deployment
func (dpm *DatabasePoolManager) GetProvisionedDatabase(deploymentID uuid.UUID) (*models.ProvisionedDatabase, error) {
	var db models.ProvisionedDatabase
	err := dpm.db.Preload("SharedDatabaseInstance").Where("deployment_id = ?", deploymentID).First(&db).Error
	if err != nil {
		return nil, err
	}
	return &db, nil
}

// GetSharedInstanceStats returns statistics about shared database instances
func (dpm *DatabasePoolManager) GetSharedInstanceStats() (map[string]interface{}, error) {
	var instances []models.SharedDatabaseInstance
	if err := dpm.db.Where("status = ?", "running").Find(&instances).Error; err != nil {
		return nil, err
	}

	totalInstances := len(instances)
	totalDatabases := 0
	estimatedRAMSaved := 0

	for _, instance := range instances {
		totalDatabases += instance.DatabaseCount
		// Calculate RAM saved: if we deployed individual containers instead of shared
		// Each individual container would use EstimatedRAMMB
		// Shared instance uses EstimatedRAMMB + (DatabaseCount * small overhead)
		individualRAM := instance.DatabaseCount * instance.EstimatedRAMMB
		sharedRAM := instance.EstimatedRAMMB + (instance.DatabaseCount * 50) // 50MB overhead per database
		if instance.DatabaseCount > 0 {
			estimatedRAMSaved += (individualRAM - sharedRAM)
		}
	}

	return map[string]interface{}{
		"shared_instances":    totalInstances,
		"total_databases":     totalDatabases,
		"estimated_ram_saved_mb": estimatedRAMSaved,
		"estimated_ram_saved_percent": dpm.calculateRAMSavedPercent(totalDatabases, estimatedRAMSaved),
	}, nil
}

func (dpm *DatabasePoolManager) calculateRAMSavedPercent(dbCount, savedMB int) float64 {
	if dbCount == 0 {
		return 0
	}
	// Rough calculation: average DB container ~300MB, shared overhead ~50MB per DB
	wouldHaveUsed := dbCount * 300
	if wouldHaveUsed == 0 {
		return 0
	}
	return (float64(savedMB) / float64(wouldHaveUsed)) * 100
}

// ====== DEPENDENCY SERVICE INTEGRATION ======
// These methods provide a clean interface for DependencyService

// SharedInstanceExists checks if a shared database instance exists on a device
// Used by DependencyService for dependency checking
func (dpm *DatabasePoolManager) SharedInstanceExists(ctx context.Context, deviceID uuid.UUID, engine string) (bool, error) {
	var instance models.SharedDatabaseInstance
	err := dpm.db.WithContext(ctx).Where("device_id = ? AND engine = ? AND status = ?",
		deviceID, engine, "running").First(&instance).Error

	if err == nil {
		return true, nil
	}
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	return false, err
}

// ProvisionDatabaseInSharedInstance creates a database in a shared instance
// Used by DependencyService for auto-provisioning
// This is a context-aware wrapper around ProvisionDatabase
func (dpm *DatabasePoolManager) ProvisionDatabaseInSharedInstance(
	ctx context.Context,
	deviceID uuid.UUID,
	engine string,
	dbName string,
	appSlug string,
) error {
	// Get device
	var device models.Device
	if err := dpm.db.First(&device, deviceID).Error; err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	// Get or create shared instance
	version := "" // Use default version
	instance, err := dpm.GetOrCreateSharedInstance(&device, engine, version)
	if err != nil {
		return fmt.Errorf("failed to get/create shared instance: %w", err)
	}

	if instance.Status != "running" {
		return fmt.Errorf("shared instance is not running (status: %s)", instance.Status)
	}

	// Generate credentials
	username := fmt.Sprintf("%s_user", strings.ReplaceAll(appSlug, "-", "_"))
	password, err := dpm.generateSecurePassword(24)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	// Store credentials with app-specific key
	credKey := fmt.Sprintf("db-%s-%s", appSlug, deviceID.String())
	if err := dpm.credService.StoreCredential(credKey, password); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Create database and user in the shared instance
	if err := dpm.createDatabaseAndUser(&device, instance, dbName, username, password); err != nil {
		return fmt.Errorf("failed to create database and user: %w", err)
	}

	// Update shared instance database count
	dpm.db.Model(instance).Update("database_count", gorm.Expr("database_count + 1"))

	log.Printf("[DatabasePool] Provisioned database %s for app %s in shared %s instance",
		dbName, appSlug, engine)

	return nil
}

// GetDatabaseCredentials retrieves database credentials for an app
// Returns connection details that can be injected into app environment
func (dpm *DatabasePoolManager) GetDatabaseCredentials(
	deviceID uuid.UUID,
	engine string,
	appSlug string,
) (map[string]string, error) {
	// Get device
	var device models.Device
	if err := dpm.db.First(&device, deviceID).Error; err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	// Get shared instance
	var instance models.SharedDatabaseInstance
	err := dpm.db.Where("device_id = ? AND engine = ? AND status = ?",
		deviceID, engine, "running").First(&instance).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get shared instance: %w", err)
	}

	// Get stored credentials
	credKey := fmt.Sprintf("db-%s-%s", appSlug, deviceID.String())
	password, err := dpm.credService.GetCredential(credKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	// Generate database name
	dbName := fmt.Sprintf("%s_db", strings.ReplaceAll(appSlug, "-", "_"))
	username := fmt.Sprintf("%s_user", strings.ReplaceAll(appSlug, "-", "_"))

	// Return connection details
	return map[string]string{
		"DB_HOST":     device.GetPrimaryAddress(),
		"DB_PORT":     fmt.Sprintf("%d", instance.Port),
		"DB_DATABASE": dbName,
		"DB_USERNAME": username,
		"DB_PASSWORD": password,
		"DB_ENGINE":   engine,
	}, nil
}
