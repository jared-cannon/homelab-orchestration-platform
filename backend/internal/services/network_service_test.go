package services

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
)

func TestNetworkService_NetworkExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	device := getTestDeviceForNetwork(t)
	sshClient := ssh.NewClient()
	networkService := NewNetworkService(sshClient)

	// Test with a network that should exist (bridge is default)
	exists, err := networkService.NetworkExists(device, "bridge")
	if err != nil {
		t.Fatalf("NetworkExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected bridge network to exist")
	}

	// Test with a network that shouldn't exist
	exists, err = networkService.NetworkExists(device, "nonexistent-network-12345")
	if err != nil {
		t.Fatalf("NetworkExists failed: %v", err)
	}
	if exists {
		t.Error("Expected nonexistent network to not exist")
	}
}

func TestNetworkService_CreateNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	device := getTestDeviceForNetwork(t)
	sshClient := ssh.NewClient()
	networkService := NewNetworkService(sshClient)

	testNetworkName := "test-network-integration"

	// Clean up any existing test network
	defer func() {
		_ = networkService.RemoveNetwork(device, testNetworkName)
	}()

	// Create network (Compose mode)
	err := networkService.CreateNetwork(device, testNetworkName, false)
	if err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Verify network exists
	exists, err := networkService.NetworkExists(device, testNetworkName)
	if err != nil {
		t.Fatalf("NetworkExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected created network to exist")
	}

	// Verify driver is bridge
	driver, err := networkService.GetNetworkDriver(device, testNetworkName)
	if err != nil {
		t.Fatalf("GetNetworkDriver failed: %v", err)
	}
	if driver != "bridge" {
		t.Errorf("Expected driver to be 'bridge', got '%s'", driver)
	}

	// Creating again should not error (idempotent)
	err = networkService.CreateNetwork(device, testNetworkName, false)
	if err != nil {
		t.Fatalf("CreateNetwork (second time) failed: %v", err)
	}
}

func TestNetworkService_EnsureTraefikNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	device := getTestDeviceForNetwork(t)
	sshClient := ssh.NewClient()
	networkService := NewNetworkService(sshClient)

	// Clean up any existing Traefik network
	defer func() {
		_ = networkService.RemoveNetwork(device, TraefikNetworkName)
	}()

	// Ensure Traefik network exists
	err := networkService.EnsureTraefikNetwork(device, false)
	if err != nil {
		t.Fatalf("EnsureTraefikNetwork failed: %v", err)
	}

	// Verify network exists
	exists, err := networkService.NetworkExists(device, TraefikNetworkName)
	if err != nil {
		t.Fatalf("NetworkExists failed: %v", err)
	}
	if !exists {
		t.Error("Expected Traefik network to exist")
	}
}

func TestNetworkService_RemoveNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	device := getTestDeviceForNetwork(t)
	sshClient := ssh.NewClient()
	networkService := NewNetworkService(sshClient)

	testNetworkName := "test-network-remove"

	// Create a test network
	err := networkService.CreateNetwork(device, testNetworkName, false)
	if err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	// Remove the network
	err = networkService.RemoveNetwork(device, testNetworkName)
	if err != nil {
		t.Fatalf("RemoveNetwork failed: %v", err)
	}

	// Verify network no longer exists
	exists, err := networkService.NetworkExists(device, testNetworkName)
	if err != nil {
		t.Fatalf("NetworkExists failed: %v", err)
	}
	if exists {
		t.Error("Expected network to be removed")
	}

	// Removing again should not error
	err = networkService.RemoveNetwork(device, testNetworkName)
	if err != nil {
		t.Fatalf("RemoveNetwork (second time) failed: %v", err)
	}
}

func getTestDeviceForNetwork(t *testing.T) *models.Device {
	// This assumes TEST_DEVICE_IP, TEST_SSH_USER, TEST_SSH_PASSWORD env vars are set
	device := &models.Device{
		ID:               uuid.New(),
		Name:             "Test Device",
		LocalIPAddress:   getEnvOrSkipNetwork(t, "TEST_DEVICE_IP"),
		Username:         getEnvOrSkipNetwork(t, "TEST_SSH_USER"),
		AuthType:         models.AuthTypePassword,
		PrimaryConnection: models.PrimaryConnectionLocal,
	}

	// Store password for SSH service to retrieve
	testPassword := getEnvOrSkipNetwork(t, "TEST_SSH_PASSWORD")
	_ = testPassword // The SSH service will retrieve it from env

	return device
}

func getEnvOrSkipNetwork(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Skipping test: %s environment variable not set", key)
	}
	return value
}
