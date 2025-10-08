package ssh

import (
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client represents an SSH client with connection pooling
type Client struct {
	connections sync.Map // map[string]*ssh.Client
	mu          sync.Mutex
}

// NewClient creates a new SSH client
func NewClient() *Client {
	return &Client{}
}

// Connect establishes an SSH connection to a host
func (c *Client) Connect(host string, username string, auth ssh.AuthMethod) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification in production
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}

	// Store connection for reuse
	c.connections.Store(host, client)

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

// GetConnection retrieves an existing connection or creates a new one
func (c *Client) GetConnection(host string) (*ssh.Client, error) {
	if conn, ok := c.connections.Load(host); ok {
		client := conn.(*ssh.Client)
		// Test if connection is still alive
		session, err := client.NewSession()
		if err == nil {
			session.Close()
			return client, nil
		}
		// Connection is dead, remove it
		c.connections.Delete(host)
	}
	return nil, fmt.Errorf("no active connection to %s", host)
}

// Execute runs a command on the remote host
func (c *Client) Execute(host string, command string) (string, error) {
	client, err := c.GetConnection(host)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
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
		client := conn.(*ssh.Client)
		c.connections.Delete(host)
		return client.Close()
	}
	return nil
}

// CloseAll closes all connections
func (c *Client) CloseAll() {
	c.connections.Range(func(key, value interface{}) bool {
		if client, ok := value.(*ssh.Client); ok {
			client.Close()
		}
		c.connections.Delete(key)
		return true
	})
}
