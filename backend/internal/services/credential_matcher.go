package services

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// CredentialMatcher handles finding and testing credentials for devices
type CredentialMatcher struct {
	db        *gorm.DB
	credSvc   *CredentialService
	sshClient *ssh.Client
}

// NewCredentialMatcher creates a new credential matcher
func NewCredentialMatcher(db *gorm.DB, credSvc *CredentialService, sshClient *ssh.Client) *CredentialMatcher {
	return &CredentialMatcher{
		db:        db,
		credSvc:   credSvc,
		sshClient: sshClient,
	}
}

// FindMatchingCredentials returns credentials that could apply to a device
func (cm *CredentialMatcher) FindMatchingCredentials(ipAddress, hostname string, deviceType models.DeviceType) ([]models.Credential, error) {
	var credentials []models.Credential

	// Get all credentials ordered by last used (most recent first)
	if err := cm.db.Order("last_used DESC NULLS LAST").Find(&credentials).Error; err != nil {
		return nil, err
	}

	var matches []models.Credential

	for _, cred := range credentials {
		if cm.credentialMatches(cred, ipAddress, hostname, deviceType) {
			matches = append(matches, cred)
		}
	}

	return matches, nil
}

// credentialMatches checks if a credential matches the given device criteria
func (cm *CredentialMatcher) credentialMatches(cred models.Credential, ipAddress, hostname string, deviceType models.DeviceType) bool {
	// Check network CIDR match
	if cred.NetworkCIDR != "" {
		_, network, err := net.ParseCIDR(cred.NetworkCIDR)
		if err == nil {
			ip := net.ParseIP(ipAddress)
			if ip != nil && network.Contains(ip) {
				return true
			}
		}
	}

	// Check device type match
	if cred.DeviceType != "" && string(deviceType) == cred.DeviceType {
		return true
	}

	// Check hostname pattern match
	if cred.HostPattern != "" && hostname != "" {
		pattern := strings.ToLower(cred.HostPattern)
		hostLower := strings.ToLower(hostname)

		// Simple wildcard matching
		pattern = strings.ReplaceAll(pattern, "*", "")
		if strings.Contains(hostLower, pattern) {
			return true
		}
	}

	// If no specific criteria set, it's a general credential
	if cred.NetworkCIDR == "" && cred.DeviceType == "" && cred.HostPattern == "" {
		return true
	}

	return false
}

// TestCredential tests a credential against a device
func (cm *CredentialMatcher) TestCredential(ctx context.Context, credential models.Credential, ipAddress string, port int) error {
	if port == 0 {
		port = 22
	}

	address := fmt.Sprintf("%s:%d", ipAddress, port)

	// Decrypt credentials
	password, err := cm.credSvc.DecryptData(credential.Password)
	if err != nil && credential.Type == models.CredentialTypePassword {
		return fmt.Errorf("failed to decrypt password: %w", err)
	}

	var sshKey, keyPassphrase string
	if credential.Type == models.CredentialTypeSSHKey {
		sshKey, err = cm.credSvc.DecryptData(credential.SSHKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt SSH key: %w", err)
		}

		if credential.SSHKeyPassphrase != "" {
			keyPassphrase, err = cm.credSvc.DecryptData(credential.SSHKeyPassphrase)
			if err != nil {
				return fmt.Errorf("failed to decrypt key passphrase: %w", err)
			}
		}
	}

	// Test the connection by attempting to connect
	var connErr error
	if credential.Type == models.CredentialTypePassword {
		_, connErr = cm.sshClient.ConnectWithPassword(address, credential.Username, password)
	} else {
		_, connErr = cm.sshClient.ConnectWithKey(address, credential.Username, sshKey, keyPassphrase)
	}

	if connErr == nil {
		// Update last used timestamp
		now := time.Now()
		cm.db.Model(&credential).Updates(map[string]interface{}{
			"last_used":  now,
			"use_count":  gorm.Expr("use_count + 1"),
		})
	}

	return connErr
}

// TestAllCredentials tests all matching credentials and returns the first working one
func (cm *CredentialMatcher) TestAllCredentials(ctx context.Context, ipAddress, hostname string, deviceType models.DeviceType, port int) (*models.Credential, error) {
	credentials, err := cm.FindMatchingCredentials(ipAddress, hostname, deviceType)
	if err != nil {
		return nil, err
	}

	if len(credentials) == 0 {
		return nil, fmt.Errorf("no matching credentials found")
	}

	// Try each credential
	for _, cred := range credentials {
		if err := cm.TestCredential(ctx, cred, ipAddress, port); err == nil {
			return &cred, nil
		}
	}

	return nil, fmt.Errorf("no working credentials found")
}

// GetCommonCredentials returns common default credentials to try
func (cm *CredentialMatcher) GetCommonCredentials() []struct {
	Username string
	Password string
	Label    string
} {
	return []struct {
		Username string
		Password string
		Label    string
	}{
		{"root", "root", "root/root"},
		{"admin", "admin", "admin/admin"},
		{"admin", "password", "admin/password"},
		{"pi", "raspberry", "Raspberry Pi default"},
		{"ubuntu", "ubuntu", "Ubuntu default"},
	}
}
