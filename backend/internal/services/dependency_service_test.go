package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestBuildTraefikEnvironment tests environment variable generation for Traefik
func TestBuildTraefikEnvironment(t *testing.T) {
	// Create test device
	device := &models.Device{
		ID:           uuid.New(),
		Name:         "test-server",
		DomainSuffix: "test.home.arpa",
	}

	// Create dependency service with nil infrastructure config
	ds := &DependencyService{
		infraConfig: nil,
	}

	// Test with nil infrastructure config (should use defaults)
	env := ds.buildTraefikEnvironment(device)
	assert.Equal(t, "v3.2", env["TRAEFIK_VERSION"], "Should use default Traefik version")
	assert.Equal(t, "traefik", env["CONTAINER_NAME"], "Should set container name")
	assert.Equal(t, "admin@homelab.local", env["ACME_EMAIL"], "Should set default ACME email")
	assert.Equal(t, "traefik.test.home.arpa", env["DASHBOARD_DOMAIN"], "Should generate dashboard domain from device domain suffix")
	assert.Equal(t, "8080", env["DASHBOARD_PORT"], "Should set dashboard port")
	assert.Equal(t, "admin", env["DASHBOARD_USERNAME"], "Should set dashboard username")
	assert.Equal(t, "", env["DASHBOARD_PASSWORD_HASH"], "Should leave password empty by default")

	// Test with device without domain suffix (should generate default)
	deviceNoSuffix := &models.Device{
		ID:   uuid.New(),
		Name: "my-server",
	}
	env2 := ds.buildTraefikEnvironment(deviceNoSuffix)
	assert.Contains(t, env2["DASHBOARD_DOMAIN"], "traefik.", "Should generate dashboard domain")
	assert.Contains(t, env2["DASHBOARD_DOMAIN"], "my-server", "Should use device name in domain")

	// Test with infrastructure config
	infraConfig := &InfrastructureConfig{
		ReverseProxies: map[string]ReverseProxyConfig{
			"traefik": {
				DefaultVersion: "v3.3",
			},
		},
	}
	ds.infraConfig = infraConfig
	env3 := ds.buildTraefikEnvironment(device)
	assert.Equal(t, "v3.3", env3["TRAEFIK_VERSION"], "Should use infrastructure config version")
}

// TestContainsIgnoreCase tests case-insensitive string matching
func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{
			name:     "Exact match same case",
			s:        "hello world",
			substr:   "hello",
			expected: true,
		},
		{
			name:     "Exact match different case",
			s:        "Hello World",
			substr:   "WORLD",
			expected: true,
		},
		{
			name:     "Substring in middle",
			s:        "network already exists",
			substr:   "already",
			expected: true,
		},
		{
			name:     "Substring at end",
			s:        "network with name homelab-proxy already exists",
			substr:   "already exists",
			expected: true,
		},
		{
			name:     "Substring at start",
			s:        "Error: network already exists",
			substr:   "error",
			expected: true,
		},
		{
			name:     "Not found",
			s:        "hello world",
			substr:   "goodbye",
			expected: false,
		},
		{
			name:     "Empty substring",
			s:        "hello",
			substr:   "",
			expected: false,
		},
		{
			name:     "Empty string",
			s:        "",
			substr:   "hello",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreCase(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Note: Network creation tests would require MockSSHClient which is already declared
// in other test files. These tests are covered by integration testing.

