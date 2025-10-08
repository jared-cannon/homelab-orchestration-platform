package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/99designs/keyring"
)

// DeviceCredentials represents credentials for accessing a device
type DeviceCredentials struct {
	Type         string `json:"type"` // "password" or "ssh_key"
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`      // For password auth
	SSHKey       string `json:"ssh_key,omitempty"`       // For SSH key auth
	SSHKeyPasswd string `json:"ssh_key_passwd,omitempty"` // SSH key passphrase if needed
}

// CredentialService manages secure storage of device credentials using OS keychain
type CredentialService struct {
	ring keyring.Keyring
}

// NewCredentialService creates a new credential service
func NewCredentialService() (*CredentialService, error) {
	// Configure keyring
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "homelab-orchestration-platform",
		// Try OS keychain first, fallback to encrypted file if unavailable
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,  // macOS Keychain
			keyring.SecretServiceBackend, // Linux Secret Service (gnome-keyring, kwallet)
			keyring.WinCredBackend, // Windows Credential Manager
			keyring.FileBackend,    // Encrypted file fallback
		},
		// For file backend (fallback)
		FileDir: "~/.homelab",
		FilePasswordFunc: func(prompt string) (string, error) {
			// In production, this should prompt the user
			// For now, use a default (this is the fallback anyway)
			return "homelab-secret-key", nil
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	return &CredentialService{ring: ring}, nil
}

// StoreCredentials stores device credentials securely in the OS keychain
func (s *CredentialService) StoreCredentials(deviceID string, creds *DeviceCredentials) error {
	// Marshal credentials to JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Store in keyring with device ID as key
	item := keyring.Item{
		Key:  deviceID,
		Data: data,
	}

	if err := s.ring.Set(item); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

// GetCredentials retrieves device credentials from the OS keychain
func (s *CredentialService) GetCredentials(deviceID string) (*DeviceCredentials, error) {
	// Retrieve from keyring
	item, err := s.ring.Get(deviceID)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, fmt.Errorf("credentials not found for device: %s", deviceID)
		}
		return nil, fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	// Unmarshal JSON
	var creds DeviceCredentials
	if err := json.Unmarshal(item.Data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// EncryptData encrypts data using the keyring
func (s *CredentialService) EncryptData(data string) (string, error) {
	// For now, return as-is. In production, use proper encryption
	// TODO: Implement proper AES encryption
	return data, nil
}

// DecryptData decrypts data using the keyring
func (s *CredentialService) DecryptData(encrypted string) (string, error) {
	// For now, return as-is. In production, use proper decryption
	// TODO: Implement proper AES decryption
	return encrypted, nil
}

// DeleteCredentials removes device credentials from the OS keychain
func (s *CredentialService) DeleteCredentials(deviceID string) error {
	if err := s.ring.Remove(deviceID); err != nil {
		if err == keyring.ErrKeyNotFound {
			// Already deleted, not an error
			return nil
		}
		// File backend may return "no such file" error instead of ErrKeyNotFound
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such file") || strings.Contains(errMsg, "not found") {
			return nil
		}
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	return nil
}

// TestCredentials checks if credentials can be retrieved (used for testing connection)
func (s *CredentialService) TestCredentials(deviceID string) (bool, error) {
	_, err := s.GetCredentials(deviceID)
	if err != nil {
		return false, err
	}
	return true, nil
}
