package services

import (
	"fmt"
	"strings"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
)

const (
	// TraefikNetworkName is the standard network name for Traefik
	TraefikNetworkName = "traefik-public"
)

// NetworkService handles Docker network management
type NetworkService struct {
	sshClient *ssh.Client
}

// NewNetworkService creates a new network service
func NewNetworkService(sshClient *ssh.Client) *NetworkService {
	return &NetworkService{
		sshClient: sshClient,
	}
}

// NetworkExists checks if a Docker network exists on the device
func (s *NetworkService) NetworkExists(device *models.Device, networkName string) (bool, error) {
	cmd := fmt.Sprintf("docker network ls --filter name=^%s$ --format '{{.Name}}'", networkName)

	output, err := s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		return false, fmt.Errorf("failed to check network existence: %w", err)
	}

	// Check if output contains the exact network name
	return strings.TrimSpace(output) == networkName, nil
}

// CreateNetwork creates a Docker network
func (s *NetworkService) CreateNetwork(device *models.Device, networkName string, isSwarmMode bool) error {
	// Check if network already exists
	exists, err := s.NetworkExists(device, networkName)
	if err != nil {
		return fmt.Errorf("failed to check if network exists: %w", err)
	}

	if exists {
		return nil // Network already exists, nothing to do
	}

	var cmd string
	if isSwarmMode {
		// For Swarm mode, create overlay network with attachable flag
		// Attachable allows standalone containers to connect to the network
		cmd = fmt.Sprintf("docker network create --driver overlay --attachable %s", networkName)
	} else {
		// For Compose mode, create bridge network
		cmd = fmt.Sprintf("docker network create --driver bridge %s", networkName)
	}

	_, err = s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		return fmt.Errorf("failed to create network %s: %w", networkName, err)
	}

	return nil
}

// EnsureTraefikNetwork ensures the Traefik network exists on the device
func (s *NetworkService) EnsureTraefikNetwork(device *models.Device, isSwarmMode bool) error {
	return s.CreateNetwork(device, TraefikNetworkName, isSwarmMode)
}

// GetNetworkDriver returns the network driver for a given network
func (s *NetworkService) GetNetworkDriver(device *models.Device, networkName string) (string, error) {
	cmd := fmt.Sprintf("docker network inspect %s --format '{{.Driver}}'", networkName)

	output, err := s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		return "", fmt.Errorf("failed to get network driver: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// ConnectContainerToNetwork connects a container to a network
func (s *NetworkService) ConnectContainerToNetwork(device *models.Device, containerName, networkName string) error {
	cmd := fmt.Sprintf("docker network connect %s %s", networkName, containerName)

	_, err := s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		// Check if already connected
		if strings.Contains(err.Error(), "already exists") {
			return nil // Already connected, not an error
		}
		return fmt.Errorf("failed to connect container to network: %w", err)
	}

	return nil
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (s *NetworkService) DisconnectContainerFromNetwork(device *models.Device, containerName, networkName string) error {
	cmd := fmt.Sprintf("docker network disconnect %s %s", networkName, containerName)

	_, err := s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		// Check if not connected
		if strings.Contains(err.Error(), "is not connected") {
			return nil // Not connected, not an error
		}
		return fmt.Errorf("failed to disconnect container from network: %w", err)
	}

	return nil
}

// RemoveNetwork removes a Docker network if it has no containers connected
func (s *NetworkService) RemoveNetwork(device *models.Device, networkName string) error {
	cmd := fmt.Sprintf("docker network rm %s", networkName)

	_, err := s.sshClient.Execute(device.GetSSHHost(), cmd)
	if err != nil {
		// Network might not exist or have containers still connected
		if strings.Contains(err.Error(), "not found") {
			return nil // Network doesn't exist, not an error
		}
		return fmt.Errorf("failed to remove network: %w", err)
	}

	return nil
}
