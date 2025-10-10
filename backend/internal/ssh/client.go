package ssh

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// connectionWrapper wraps an SSH connection with metadata
type connectionWrapper struct {
	client      *ssh.Client
	createdAt   time.Time
	lastUsedAt  time.Time
	mu          sync.RWMutex
}

// Client represents an SSH client with connection pooling
type Client struct {
	connections      sync.Map // map[string]*connectionWrapper
	mu               sync.Mutex
	maxIdleTime      time.Duration
	cleanupInterval  time.Duration
	shutdownChan     chan struct{}
	knownHostsFile   string
	knownHostsCallback ssh.HostKeyCallback
}

// NewClient creates a new SSH client with connection pool management
func NewClient() *Client {
	// Determine known_hosts file path
	knownHostsPath := os.Getenv("SSH_KNOWN_HOSTS")
	if knownHostsPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory
			knownHostsPath = ".homelab_known_hosts"
		} else {
			// Use ~/.homelab/known_hosts
			homelabDir := filepath.Join(homeDir, ".homelab")
			os.MkdirAll(homelabDir, 0700)
			knownHostsPath = filepath.Join(homelabDir, "known_hosts")
		}
	}

	c := &Client{
		maxIdleTime:     15 * time.Minute, // Close connections idle for 15 minutes
		cleanupInterval: 5 * time.Minute,  // Check every 5 minutes
		shutdownChan:    make(chan struct{}),
		knownHostsFile:  knownHostsPath,
	}

	// Initialize host key callback
	c.initHostKeyCallback()

	// Start background cleanup goroutine
	go c.cleanupIdleConnections()

	return c
}

// initHostKeyCallback initializes the host key verification callback with TOFU support
func (c *Client) initHostKeyCallback() {
	// Try to load existing known_hosts file
	var callback ssh.HostKeyCallback
	var err error

	// Check if known_hosts file exists
	if _, statErr := os.Stat(c.knownHostsFile); statErr == nil {
		// File exists, load it
		callback, err = knownhosts.New(c.knownHostsFile)
		if err != nil {
			log.Printf("[SSH] Warning: failed to load known_hosts file: %v", err)
			callback = nil
		}
	}

	// Wrap callback with TOFU logic
	c.knownHostsCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// If we have a known_hosts callback, try it first
		if callback != nil {
			err := callback(hostname, remote, key)
			if err == nil {
				// Key is known and valid
				return nil
			}

			// Check if this is a "key not found" error (unknown host)
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
				// Unknown host - Trust On First Use
				if addErr := c.addHostKey(hostname, remote, key); addErr != nil {
					log.Printf("[SSH] Failed to add host key: %v", addErr)
					return fmt.Errorf("unknown host and failed to store key: %w", addErr)
				}
				log.Printf("[SSH] Trust On First Use: added host key for %s (%s)", hostname, keyFingerprint(key))
				return nil
			}

			// Key mismatch or other error
			return fmt.Errorf("host key verification failed: %w", err)
		}

		// No known_hosts file yet - Trust On First Use for first host
		if addErr := c.addHostKey(hostname, remote, key); addErr != nil {
			log.Printf("[SSH] Failed to create known_hosts: %v", addErr)
			return fmt.Errorf("failed to store host key: %w", addErr)
		}
		log.Printf("[SSH] Trust On First Use: added first host key for %s (%s)", hostname, keyFingerprint(key))

		// Reload the callback for future connections
		if newCallback, err := knownhosts.New(c.knownHostsFile); err == nil {
			callback = newCallback
		}

		return nil
	}
}

// addHostKey adds a host key to the known_hosts file
func (c *Client) addHostKey(hostname string, remote net.Addr, key ssh.PublicKey) error {
	// Ensure parent directory exists
	dir := filepath.Dir(c.knownHostsFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open known_hosts file for appending
	f, err := os.OpenFile(c.knownHostsFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts: %w", err)
	}
	defer f.Close()

	// Write the host key entry
	line := knownhosts.Line([]string{hostname}, key)
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}

	return nil
}

// keyFingerprint returns SHA256 fingerprint of the public key
func keyFingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

// Shutdown gracefully closes all connections and stops cleanup goroutine
func (c *Client) Shutdown() {
	close(c.shutdownChan)
	c.CloseAll()
}

// cleanupIdleConnections periodically removes idle or dead connections
func (c *Client) cleanupIdleConnections() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.shutdownChan:
			log.Printf("[SSH] Connection cleanup goroutine stopping")
			return
		case <-ticker.C:
			now := time.Now()
			var toRemove []string

			c.connections.Range(func(key, value interface{}) bool {
				host := key.(string)
				wrapper := value.(*connectionWrapper)

				wrapper.mu.RLock()
				lastUsed := wrapper.lastUsedAt
				wrapper.mu.RUnlock()

				// Mark connections idle for too long
				if now.Sub(lastUsed) > c.maxIdleTime {
					toRemove = append(toRemove, host)
				} else {
					// Test if connection is still alive
					session, err := wrapper.client.NewSession()
					if err != nil {
						// Connection is dead
						toRemove = append(toRemove, host)
					} else {
						session.Close()
					}
				}

				return true
			})

			// Remove idle/dead connections
			for _, host := range toRemove {
				log.Printf("[SSH] Cleaning up idle/dead connection to %s", host)
				c.Close(host)
			}
		}
	}
}

// Connect establishes an SSH connection to a host
func (c *Client) Connect(host string, username string, auth ssh.AuthMethod) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: c.knownHostsCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	// Wrap connection with metadata
	now := time.Now()
	wrapper := &connectionWrapper{
		client:     client,
		createdAt:  now,
		lastUsedAt: now,
	}

	// Store connection for reuse
	c.connections.Store(host, wrapper)

	return client, nil
}

// ConnectWithPassword connects using password authentication
func (c *Client) ConnectWithPassword(host string, username string, password string) (*ssh.Client, error) {
	return c.Connect(host, username, ssh.Password(password))
}

// ConnectWithKey connects using SSH key authentication
func (c *Client) ConnectWithKey(host string, username string, privateKey string, passphrase string) (*ssh.Client, error) {
	var signer ssh.Signer
	var err error

	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey([]byte(privateKey))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return c.Connect(host, username, ssh.PublicKeys(signer))
}

// getSSHAgent attempts to connect to the SSH agent
func (c *Client) getSSHAgent() (agent.ExtendedAgent, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not set - no SSH agent available")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	return agent.NewClient(conn), nil
}

// ConnectWithAgent connects using SSH agent authentication
func (c *Client) ConnectWithAgent(host string, username string) (*ssh.Client, error) {
	agentClient, err := c.getSSHAgent()
	if err != nil {
		return nil, err
	}

	return c.Connect(host, username, ssh.PublicKeysCallback(agentClient.Signers))
}

// getDefaultSSHKeys returns paths to common SSH key files
func (c *Client) getDefaultSSHKeys() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Try keys in order of preference (modern to legacy)
	keyFiles := []string{
		filepath.Join(sshDir, "id_ed25519"),
		filepath.Join(sshDir, "id_ecdsa"),
		filepath.Join(sshDir, "id_rsa"),
		filepath.Join(sshDir, "id_dsa"),
	}

	var existingKeys []string
	for _, keyFile := range keyFiles {
		if _, err := os.Stat(keyFile); err == nil {
			existingKeys = append(existingKeys, keyFile)
		}
	}

	return existingKeys
}

// ConnectWithDefaultKeys tries to connect using default SSH keys from ~/.ssh
func (c *Client) ConnectWithDefaultKeys(host string, username string) (*ssh.Client, error) {
	keyFiles := c.getDefaultSSHKeys()
	if len(keyFiles) == 0 {
		return nil, fmt.Errorf("no default SSH keys found in ~/.ssh")
	}

	var lastErr error
	for _, keyFile := range keyFiles {
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			log.Printf("[SSH] Failed to read key file %s: %v", keyFile, err)
			lastErr = err
			continue
		}

		// Try without passphrase first
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			log.Printf("[SSH] Key %s requires passphrase or is invalid, skipping", keyFile)
			lastErr = err
			continue
		}

		// Try to connect with this key
		client, err := c.Connect(host, username, ssh.PublicKeys(signer))
		if err == nil {
			log.Printf("[SSH] Successfully connected using default key: %s", keyFile)
			return client, nil
		}

		log.Printf("[SSH] Failed to connect with key %s: %v", keyFile, err)
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to connect with any default SSH key: %w", lastErr)
	}
	return nil, fmt.Errorf("no usable default SSH keys found")
}

// TryAutoAuth attempts to authenticate using SSH agent first, then default keys
func (c *Client) TryAutoAuth(host string, username string) (*ssh.Client, error) {
	// Try SSH agent first (best option - keys stay encrypted and secure)
	client, err := c.ConnectWithAgent(host, username)
	if err == nil {
		log.Printf("[SSH] Successfully connected to %s using SSH agent", host)
		return client, nil
	}
	log.Printf("[SSH] SSH agent auth failed, trying default keys: %v", err)

	// Fall back to default SSH keys
	client, err = c.ConnectWithDefaultKeys(host, username)
	if err == nil {
		return client, nil
	}

	return nil, fmt.Errorf("auto authentication failed - no SSH agent or default keys worked: %w", err)
}

// GetConnection retrieves an existing connection or creates a new one
func (c *Client) GetConnection(host string) (*ssh.Client, error) {
	if conn, ok := c.connections.Load(host); ok {
		wrapper := conn.(*connectionWrapper)

		// Update last used time
		wrapper.mu.Lock()
		wrapper.lastUsedAt = time.Now()
		wrapper.mu.Unlock()

		// Test if connection is still alive
		session, err := wrapper.client.NewSession()
		if err == nil {
			session.Close()
			return wrapper.client, nil
		}
		// Connection is dead, remove it
		c.connections.Delete(host)
	}
	return nil, fmt.Errorf("no active connection to %s", host)
}

// Execute runs a command on the remote host with a default 5-minute timeout
func (c *Client) Execute(host string, command string) (string, error) {
	return c.ExecuteWithTimeout(host, command, 5*time.Minute)
}

// ExecuteWithTimeout runs a command on the remote host with a specified timeout
func (c *Client) ExecuteWithTimeout(host string, command string, timeout time.Duration) (string, error) {
	client, err := c.GetConnection(host)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Create a channel to receive the result
	type result struct {
		output string
		err    error
	}
	resultChan := make(chan result, 1)

	// Run command in goroutine
	go func() {
		output, err := session.CombinedOutput(command)
		if err != nil {
			resultChan <- result{output: string(output), err: fmt.Errorf("command failed: %w", err)}
		} else {
			resultChan <- result{output: string(output), err: nil}
		}
	}()

	// Wait for result or timeout
	select {
	case res := <-resultChan:
		return res.output, res.err
	case <-time.After(timeout):
		// Timeout occurred - close session to kill the command
		session.Close()
		return "", fmt.Errorf("command timed out after %v", timeout)
	}
}

// CopyFile copies a file to the remote host
func (c *Client) CopyFile(host string, remotePath string, content string) error {
	client, err := c.GetConnection(host)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use SCP-like approach
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		fmt.Fprintf(w, "C0644 %d %s\n", len(content), remotePath)
		io.WriteString(w, content)
		fmt.Fprint(w, "\x00")
	}()

	if err := session.Run("/usr/bin/scp -t " + remotePath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// TestConnection tests if a connection is alive
func (c *Client) TestConnection(host string) error {
	_, err := c.GetConnection(host)
	return err
}

// Close closes a specific connection
func (c *Client) Close(host string) error {
	if conn, ok := c.connections.Load(host); ok {
		wrapper := conn.(*connectionWrapper)
		c.connections.Delete(host)
		return wrapper.client.Close()
	}
	return nil
}

// CloseAll closes all connections
func (c *Client) CloseAll() {
	c.connections.Range(func(key, value interface{}) bool {
		if wrapper, ok := value.(*connectionWrapper); ok {
			wrapper.client.Close()
		}
		c.connections.Delete(key)
		return true
	})
}
