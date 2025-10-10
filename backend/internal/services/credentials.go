package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/99designs/keyring"
)

// DeviceCredentials represents credentials for accessing a device
type DeviceCredentials struct {
	Type         string `json:"type"` // "password", "ssh_key", or "auto" (agent/default keys)
	Username     string `json:"username"`
	Password     string `json:"password,omitempty"`      // For password auth
	SSHKey       string `json:"ssh_key,omitempty"`       // For SSH key auth
	SSHKeyPasswd string `json:"ssh_key_passwd,omitempty"` // SSH key passphrase if needed
	// For "auto" type, only Username is required - tries SSH agent first, then default SSH keys
}

// CredentialService manages secure storage of device credentials using OS keychain
type CredentialService struct {
	ring          keyring.Keyring
	encryptionKey []byte
}

// getEncryptionKey derives a 32-byte AES-256 key from environment or generates one
func getEncryptionKey() []byte {
	// Try to get key from environment variable
	keyStr := os.Getenv("ENCRYPTION_KEY")
	if keyStr == "" {
		// For production, this should fail and require explicit key
		// For development, use a default key (NOT SECURE for production)
		keyStr = "homelab-default-encryption-key-change-in-production"
	}

	// Derive 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(keyStr))
	return hash[:]
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

	return &CredentialService{
		ring:          ring,
		encryptionKey: getEncryptionKey(),
	}, nil
}

// StoreCredentials stores device credentials securely in the OS keychain
// deviceName and deviceIP are optional - used for a more descriptive keychain label
func (s *CredentialService) StoreCredentials(deviceID string, creds *DeviceCredentials, deviceName, deviceIP string) error {
	// Marshal credentials to JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Create a user-friendly label for the keychain
	label := fmt.Sprintf("Homelab Device: %s", deviceID)
	if deviceName != "" && deviceIP != "" {
		label = fmt.Sprintf("Homelab: %s (%s)", deviceName, deviceIP)
	} else if deviceName != "" {
		label = fmt.Sprintf("Homelab: %s", deviceName)
	} else if deviceIP != "" {
		label = fmt.Sprintf("Homelab: %s", deviceIP)
	}

	// Store in keyring with device ID as key and a descriptive label
	item := keyring.Item{
		Key:         deviceID,
		Data:        data,
		Label:       label,
		Description: "SSH credentials for homelab device",
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

// EncryptData encrypts data using AES-256-GCM
func (s *CredentialService) EncryptData(data string) (string, error) {
	if data == "" {
		return "", nil
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode (Galois/Counter Mode - provides authentication)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (number used once)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate the data
	// Format: nonce + ciphertext + authentication tag
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptData decrypts data using AES-256-GCM
func (s *CredentialService) DecryptData(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher block
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from ciphertext
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify authentication tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
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
