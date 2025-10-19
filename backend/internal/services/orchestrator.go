package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
)

// ContainerOrchestrator defines the interface for container orchestration backends
// This abstraction allows switching between Docker Compose, Docker Swarm, or other orchestrators
type ContainerOrchestrator interface {
	// Deploy deploys a stack/service using the orchestrator
	Deploy(ctx context.Context, spec DeploymentSpec) error

	// HealthCheck checks if a deployment is healthy and running
	HealthCheck(ctx context.Context, stackName string, host string) (HealthStatus, error)

	// Remove removes a deployment and optionally its volumes
	Remove(ctx context.Context, stackName string, host string, includeVolumes bool) error

	// WaitForHealthy waits for a deployment to become healthy with timeout
	WaitForHealthy(ctx context.Context, stackName string, host string, timeout time.Duration) error

	// RemoveWithCleanup removes a deployment and cleans up all associated resources
	RemoveWithCleanup(ctx context.Context, spec RemovalSpec) error

	// GetMode returns the orchestration mode (compose, swarm, etc.)
	GetMode() string
}

// DeploymentSpec contains all information needed to deploy a service
type DeploymentSpec struct {
	Host           string            // SSH host (e.g., "192.168.1.10:22")
	StackName      string            // Stack/project name
	DeployDir      string            // Directory to deploy from (e.g., ~/homelab-deployments/xxx)
	ComposeContent string            // Docker compose file content
	Environment    map[string]string // Additional environment variables
	Timeout        time.Duration     // Deployment timeout
}

// Validate ensures the DeploymentSpec has all required fields and is secure
// This validation prevents shell injection and ensures deployment safety
func (ds *DeploymentSpec) Validate() error {
	// Required fields validation
	if ds.Host == "" {
		return fmt.Errorf("deployment spec: host cannot be empty")
	}
	if !isValidHost(ds.Host) {
		return fmt.Errorf("deployment spec: host must be in format 'host:port' (e.g., '192.168.1.10:22')")
	}
	if ds.StackName == "" {
		return fmt.Errorf("deployment spec: stack name cannot be empty")
	}
	if ds.DeployDir == "" {
		return fmt.Errorf("deployment spec: deploy directory cannot be empty")
	}
	if ds.ComposeContent == "" {
		return fmt.Errorf("deployment spec: compose content cannot be empty")
	}

	// Security validation: prevent shell injection in stack name
	// Stack name should only contain alphanumeric characters, hyphens, and underscores
	if !isValidStackName(ds.StackName) {
		return fmt.Errorf("deployment spec: stack name contains invalid characters (only alphanumeric, hyphens, and underscores allowed)")
	}

	// Security validation: prevent path traversal and shell injection in deploy directory
	if !isValidDeployPath(ds.DeployDir) {
		return fmt.Errorf("deployment spec: deploy directory contains invalid or dangerous characters")
	}

	// Timeout validation
	if ds.Timeout < 0 {
		return fmt.Errorf("deployment spec: timeout cannot be negative")
	}

	// Validate environment variable keys and values
	for key, value := range ds.Environment {
		if !isValidEnvVarName(key) {
			return fmt.Errorf("deployment spec: environment variable name %q contains invalid characters", key)
		}
		if !isValidEnvVarValue(value) {
			return fmt.Errorf("deployment spec: environment variable value for %q is invalid (too long or contains illegal characters)", key)
		}
	}

	return nil
}

// HealthStatus represents the health status of a deployment
type HealthStatus struct {
	Healthy   bool
	Running   bool
	Message   string
	Timestamp time.Time
}

// RemovalSpec contains information for removing a deployment
type RemovalSpec struct {
	Host          string
	StackName     string
	DeployDir     string
	ContainerName string // Optional: specific container to remove
	IncludeVolumes bool
}

// Validate ensures the RemovalSpec has all required fields and is secure
// This validation prevents shell injection during cleanup operations
func (rs *RemovalSpec) Validate() error {
	// Required fields validation
	if rs.Host == "" {
		return fmt.Errorf("removal spec: host cannot be empty")
	}
	if !isValidHost(rs.Host) {
		return fmt.Errorf("removal spec: host must be in format 'host:port' (e.g., '192.168.1.10:22')")
	}
	if rs.StackName == "" {
		return fmt.Errorf("removal spec: stack name cannot be empty")
	}

	// Security validation: prevent shell injection in stack name
	if !isValidStackName(rs.StackName) {
		return fmt.Errorf("removal spec: stack name contains invalid characters (only alphanumeric, hyphens, and underscores allowed)")
	}

	// Security validation: validate deploy directory if provided
	if rs.DeployDir != "" && !isValidDeployPath(rs.DeployDir) {
		return fmt.Errorf("removal spec: deploy directory contains invalid or dangerous characters")
	}

	// Security validation: validate container name if provided (same rules as stack name)
	if rs.ContainerName != "" && !isValidStackName(rs.ContainerName) {
		return fmt.Errorf("removal spec: container name contains invalid characters (only alphanumeric, hyphens, and underscores allowed)")
	}

	return nil
}

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	Mode         string // "compose" or "swarm"
	SwarmEnabled bool
}

// Validation helpers for security and input sanitization

var (
	// stackNameRegex defines valid characters for stack names (alphanumeric, hyphens, underscores)
	stackNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

	// envVarNameRegex defines valid environment variable names (POSIX compliant)
	envVarNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// isValidStackName validates that a stack name only contains safe characters
// This prevents shell injection attacks through stack names
func isValidStackName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	return stackNameRegex.MatchString(name)
}

// isValidDeployPath validates that a deployment path is safe
// Prevents path traversal (..) and shell injection (;|&$`<>)
// Uses whitelist approach: only allows safe characters in paths
func isValidDeployPath(path string) bool {
	if len(path) == 0 || len(path) > 4096 {
		return false
	}

	// Disallow path traversal attempts (.. sequence)
	if strings.Contains(path, "..") {
		return false
	}

	// Disallow tilde paths to prevent traversal after shell expansion
	// Tilde expansion happens before our path checks, so ~/../../etc could bypass validation
	// If tilde support is needed, implement it explicitly with proper post-expansion validation
	if strings.HasPrefix(path, "~") || strings.Contains(path, "/~") {
		return false
	}

	// Whitelist approach: only allow safe characters in paths
	// Allowed: alphanumeric, forward slash, underscore, hyphen, dot (but not ..)
	for _, r := range path {
		isAllowed := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '/' ||
			r == '_' ||
			r == '-' ||
			r == '.'
		if !isAllowed {
			return false
		}
	}

	return true
}

// isValidHost validates that a host string has the format "host:port"
// This ensures the SSH client receives a properly formatted connection string
func isValidHost(host string) bool {
	if len(host) == 0 || len(host) > 255 {
		return false
	}

	// Must contain exactly one colon separating host and port
	parts := strings.Split(host, ":")
	if len(parts) != 2 {
		return false
	}

	// Host part should not be empty
	if len(parts[0]) == 0 {
		return false
	}

	// Port part should not be empty and should be numeric
	if len(parts[1]) == 0 {
		return false
	}

	// Validate port is numeric (basic check)
	for _, r := range parts[1] {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// isValidEnvVarName validates that an environment variable name is POSIX compliant
// This prevents injection through malicious env var names
func isValidEnvVarName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	return envVarNameRegex.MatchString(name)
}

// isValidEnvVarValue validates that an environment variable value is safe
// Prevents DoS through extremely long values and ensures compatibility with .env format
func isValidEnvVarValue(value string) bool {
	// Maximum 64KB per environment variable value
	if len(value) > 65536 {
		return false
	}

	// Disallow null bytes and other control characters that could break .env parsing
	// Allow: printable ASCII, tabs, newlines, and common Unicode
	for _, r := range value {
		// Null byte is never allowed
		if r == 0 {
			return false
		}
		// Control characters except \t, \n, \r are not allowed
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}

	return true
}

// checkContextCancelled performs a non-blocking check if context is cancelled
// Returns immediately with ctx.Err() if cancelled, nil otherwise
// Does not block or wait for cancellation - this is a point-in-time check
//
// Use this before long-running operations to fail fast when context is cancelled
func checkContextCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// NewOrchestrator creates an orchestrator based on the configuration
func NewOrchestrator(config OrchestratorConfig, sshClient *ssh.Client) ContainerOrchestrator {
	switch config.Mode {
	case "swarm":
		// Swarm orchestrator will be implemented in the future
		log.Printf("[Orchestrator] Swarm mode requested but not yet implemented, falling back to Docker Compose")
		return NewDockerComposeOrchestrator(sshClient)
	case "compose":
		fallthrough
	default:
		return NewDockerComposeOrchestrator(sshClient)
	}
}

// DockerComposeOrchestrator implements ContainerOrchestrator for Docker Compose
type DockerComposeOrchestrator struct {
	sshClient *ssh.Client
}

// NewDockerComposeOrchestrator creates a new Docker Compose orchestrator
// Note: sshClient can be nil for testing, but actual deployment operations will fail
// Production code should always provide a valid SSH client
func NewDockerComposeOrchestrator(sshClient *ssh.Client) *DockerComposeOrchestrator {
	if sshClient == nil {
		log.Printf("[Orchestrator] Warning: SSH client is nil - deployment operations will fail")
	}
	return &DockerComposeOrchestrator{
		sshClient: sshClient,
	}
}

// GetMode returns the orchestration mode
func (dco *DockerComposeOrchestrator) GetMode() string {
	return "compose"
}

// Deploy deploys a service using Docker Compose
func (dco *DockerComposeOrchestrator) Deploy(ctx context.Context, spec DeploymentSpec) error {
	// Validate SSH client is available
	if dco.sshClient == nil {
		return fmt.Errorf("SSH client is nil - cannot perform deployment operations")
	}

	// Validate spec (includes security checks for injection prevention)
	if err := spec.Validate(); err != nil {
		return err
	}

	// Set default timeout if not specified
	if spec.Timeout == 0 {
		spec.Timeout = 10 * time.Minute
	}

	// Check context before starting (fail fast if already cancelled)
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("deployment cancelled before start: %w", err)
	}

	// Create deployment directory
	mkdirCmd := fmt.Sprintf("mkdir -p %s", spec.DeployDir)
	if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, mkdirCmd, 30*time.Second); err != nil {
		return fmt.Errorf("failed to create deployment directory: %w", err)
	}

	// Check context after directory creation
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("deployment cancelled during setup: %w", err)
	}

	// Write compose file
	composeFile := fmt.Sprintf("%s/docker-compose.yml", spec.DeployDir)

	// Validate constructed path (defense-in-depth: ensure path is still safe after concatenation)
	if !isValidDeployPath(composeFile) {
		return fmt.Errorf("invalid compose file path after construction: %s", composeFile)
	}

	// SECURITY: heredoc MUST use single quotes ('EOF') to prevent shell expansion
	// Single quotes prevent variable expansion and command substitution in the content
	// This is critical for preventing injection through spec.ComposeContent
	writeCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", composeFile, spec.ComposeContent)
	if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, writeCmd, 1*time.Minute); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	// Write environment file if provided
	if len(spec.Environment) > 0 {
		envFile := fmt.Sprintf("%s/.env", spec.DeployDir)

		// Validate constructed path (defense-in-depth: ensure path is still safe after concatenation)
		if !isValidDeployPath(envFile) {
			return fmt.Errorf("invalid env file path after construction: %s", envFile)
		}

		// Sort keys for deterministic output
		keys := make([]string, 0, len(spec.Environment))
		for k := range spec.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Build env content with sorted keys and properly escaped values
		envContent := ""
		for _, key := range keys {
			escapedValue := escapeEnvValue(spec.Environment[key])
			envContent += fmt.Sprintf("%s=%s\n", key, escapedValue)
		}

		// SECURITY: heredoc MUST use single quotes ('EOF') to prevent shell expansion
		// Combined with escapeEnvValue(), this ensures env var values are safely written
		writeEnvCmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", envFile, envContent)
		if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, writeEnvCmd, 30*time.Second); err != nil {
			return fmt.Errorf("failed to write environment file: %w", err)
		}
	}

	// Check context before deployment
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("deployment cancelled before docker compose up: %w", err)
	}

	// Deploy with docker compose
	deployCmd := fmt.Sprintf("cd %s && docker compose -p %s up -d", spec.DeployDir, spec.StackName)
	output, err := dco.sshClient.ExecuteWithTimeout(spec.Host, deployCmd, spec.Timeout)
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w (output: %s)", err, output)
	}

	log.Printf("[DockerCompose] Successfully deployed stack %s", spec.StackName)
	return nil
}

// HealthCheck checks if a Docker Compose stack is healthy
func (dco *DockerComposeOrchestrator) HealthCheck(ctx context.Context, stackName string, host string) (HealthStatus, error) {
	status := HealthStatus{
		Timestamp: time.Now(),
		Healthy:   false,
		Running:   false,
	}

	// Validate SSH client is available
	if dco.sshClient == nil {
		status.Message = "SSH client is nil"
		return status, fmt.Errorf("SSH client is nil - cannot perform health check operations")
	}

	// Validate stack name (prevent shell injection)
	if !isValidStackName(stackName) {
		status.Message = "Invalid stack name"
		return status, fmt.Errorf("invalid stack name: only alphanumeric, hyphens, and underscores allowed")
	}

	// Check context before starting (fail fast if already cancelled)
	if err := checkContextCancelled(ctx); err != nil {
		status.Message = "Health check cancelled"
		return status, err
	}

	// Check if any containers with this project label exist and are running
	checkCmd := fmt.Sprintf("docker ps --filter label=com.docker.compose.project=%s --format '{{.Status}}'", stackName)
	output, err := dco.sshClient.ExecuteWithTimeout(host, checkCmd, 10*time.Second)

	if err != nil {
		status.Message = fmt.Sprintf("Failed to check container status: %v", err)
		return status, err
	}

	// If output is empty, no containers found
	if strings.TrimSpace(output) == "" {
		status.Message = "No containers found"
		return status, nil
	}

	// Check if status contains "Up"
	if strings.Contains(output, "Up") {
		status.Running = true

		// Try to check health status
		healthCmd := fmt.Sprintf("docker ps --filter label=com.docker.compose.project=%s --format '{{.Status}}' | grep -o '(healthy)\\|(unhealthy)\\|(health: starting)' || echo 'no-health'", stackName)
		healthOutput, err := dco.sshClient.ExecuteWithTimeout(host, healthCmd, 10*time.Second)

		if err == nil {
			healthStr := strings.TrimSpace(healthOutput)
			if healthStr == "(healthy)" || healthStr == "no-health" {
				status.Healthy = true
				status.Message = "Container is running and healthy"
			} else if healthStr == "(health: starting)" {
				status.Message = "Container is starting"
			} else {
				status.Message = "Container is unhealthy"
			}
		} else {
			// If we can't check health but container is up, consider it healthy
			status.Healthy = true
			status.Message = "Container is running"
		}
	} else {
		status.Message = "Container is not running"
	}

	return status, nil
}

// Remove removes a Docker Compose stack
func (dco *DockerComposeOrchestrator) Remove(ctx context.Context, stackName string, host string, includeVolumes bool) error {
	// Validate SSH client is available
	if dco.sshClient == nil {
		return fmt.Errorf("SSH client is nil - cannot perform removal operations")
	}

	// Validate stack name (prevent shell injection)
	if !isValidStackName(stackName) {
		return fmt.Errorf("invalid stack name: only alphanumeric, hyphens, and underscores allowed")
	}

	// Check context before starting (fail fast if already cancelled)
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("removal cancelled: %w", err)
	}

	// Build docker compose down command
	downCmd := fmt.Sprintf("docker compose -p %s down", stackName)
	if includeVolumes {
		downCmd += " --volumes"
	}
	downCmd += " 2>/dev/null || true"

	// Execute docker compose down
	if _, err := dco.sshClient.ExecuteWithTimeout(host, downCmd, 2*time.Minute); err != nil {
		log.Printf("[DockerCompose] Warning: docker compose down failed for %s: %v", stackName, err)
	}

	log.Printf("[DockerCompose] Removed stack %s (volumes: %v)", stackName, includeVolumes)
	return nil
}

// RemoveWithCleanup removes a deployment and cleans up associated resources
func (dco *DockerComposeOrchestrator) RemoveWithCleanup(ctx context.Context, spec RemovalSpec) error {
	// Validate SSH client is available
	if dco.sshClient == nil {
		return fmt.Errorf("SSH client is nil - cannot perform cleanup operations")
	}

	// Validate spec (includes security checks for injection prevention)
	if err := spec.Validate(); err != nil {
		return err
	}

	// Check context before starting (fail fast if already cancelled)
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("cleanup cancelled before start: %w", err)
	}

	// Try docker compose down first (graceful shutdown)
	if spec.DeployDir != "" {
		downCmd := fmt.Sprintf("cd %s && docker compose -p %s down", spec.DeployDir, spec.StackName)
		if spec.IncludeVolumes {
			downCmd += " --volumes"
		}
		downCmd += " 2>/dev/null || true"

		if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, downCmd, 2*time.Minute); err != nil {
			log.Printf("[DockerCompose] Warning: docker compose down failed for %s: %v", spec.StackName, err)
		}
	}

	// Check context after compose down
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("cleanup cancelled during container removal: %w", err)
	}

	// Fallback: force remove specific container if specified
	if spec.ContainerName != "" {
		forceRemoveCmd := fmt.Sprintf("docker rm -f %s 2>/dev/null || true", spec.ContainerName)
		if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, forceRemoveCmd, 30*time.Second); err != nil {
			log.Printf("[DockerCompose] Warning: force remove container failed for %s: %v", spec.ContainerName, err)
		}
	}

	// Check context before directory cleanup
	if err := checkContextCancelled(ctx); err != nil {
		return fmt.Errorf("cleanup cancelled before directory removal: %w", err)
	}

	// Clean up deployment directory if specified
	if spec.DeployDir != "" {
		cleanupDirCmd := fmt.Sprintf("rm -rf %s", spec.DeployDir)
		if _, err := dco.sshClient.ExecuteWithTimeout(spec.Host, cleanupDirCmd, 30*time.Second); err != nil {
			log.Printf("[DockerCompose] Warning: Failed to cleanup deployment directory %s: %v", spec.DeployDir, err)
		}
	}

	log.Printf("[DockerCompose] Cleanup completed for %s", spec.StackName)
	return nil
}

// WaitForHealthy waits for a deployment to become healthy
// Respects both the timeout parameter and the context deadline (whichever comes first)
func (dco *DockerComposeOrchestrator) WaitForHealthy(ctx context.Context, stackName string, host string, timeout time.Duration) error {
	// Validate SSH client is available
	if dco.sshClient == nil {
		return fmt.Errorf("SSH client is nil - cannot perform health check operations")
	}

	// Validate stack name (prevent shell injection)
	if !isValidStackName(stackName) {
		return fmt.Errorf("invalid stack name: only alphanumeric, hyphens, and underscores allowed")
	}

	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	// Create a timeout context that respects both the passed context and our timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	attempt := 0
	maxAttempts := int(timeout.Seconds() / 5)

	for {
		attempt++

		// Explicit attempt limit as fallback (defense-in-depth)
		// Primary timeout mechanism is context deadline, but this prevents infinite loops
		// in case of ticker or context handling bugs
		if attempt > maxAttempts {
			return fmt.Errorf("exceeded maximum health check attempts (%d)", maxAttempts)
		}

		// Check if context was cancelled or timed out
		if err := checkContextCancelled(timeoutCtx); err != nil {
			return fmt.Errorf("waiting for health cancelled after %d attempts: %w", attempt, err)
		}

		status, err := dco.HealthCheck(timeoutCtx, stackName, host)
		if err == nil && status.Healthy {
			log.Printf("[DockerCompose] Stack %s is healthy", stackName)
			return nil
		}

		log.Printf("[DockerCompose] Waiting for %s to become healthy (attempt %d/%d): %s", stackName, attempt, maxAttempts, status.Message)

		// Context-aware sleep: either timeout or ticker fires
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("deployment did not become healthy: %w", timeoutCtx.Err())
		case <-ticker.C:
			// Continue to next iteration
		}
	}
}
