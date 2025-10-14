package services

import (
	"fmt"
	"testing"
)

// TestExtractPortsFromCompose tests port extraction from various Docker Compose formats
func TestExtractPortsFromCompose(t *testing.T) {
	tests := []struct {
		name     string
		compose  string
		expected []PortSpec
	}{
		{
			name: "Quoted TCP ports",
			compose: `
services:
  web:
    ports:
      - "80:80"
      - "443:443"
`,
			expected: []PortSpec{
				{Port: 80, Protocol: "tcp"},
				{Port: 443, Protocol: "tcp"},
			},
		},
		{
			name: "Unquoted TCP ports",
			compose: `
services:
  web:
    ports:
      - 8080:80
      - 3000:3000
`,
			expected: []PortSpec{
				{Port: 8080, Protocol: "tcp"},
				{Port: 3000, Protocol: "tcp"},
			},
		},
		{
			name: "Ports with explicit protocol",
			compose: `
services:
  app:
    ports:
      - "53:53/udp"
      - "80:80/tcp"
`,
			expected: []PortSpec{
				{Port: 53, Protocol: "udp"},
				{Port: 80, Protocol: "tcp"},
			},
		},
		{
			name: "Ports with IP binding",
			compose: `
services:
  redis:
    ports:
      - "127.0.0.1:6379:6379"
      - "192.168.1.100:3306:3306"
`,
			expected: []PortSpec{
				{Port: 6379, Protocol: "tcp"},
				{Port: 3306, Protocol: "tcp"},
			},
		},
		{
			name: "Mixed formats",
			compose: `
services:
  traefik:
    ports:
      - "80:80"
      - 443:443/tcp
      - "127.0.0.1:8080:8080"
      - 51820:51820/udp
`,
			expected: []PortSpec{
				{Port: 80, Protocol: "tcp"},
				{Port: 443, Protocol: "tcp"},
				{Port: 8080, Protocol: "tcp"},
				{Port: 51820, Protocol: "udp"},
			},
		},
		{
			name: "Duplicate ports - should deduplicate",
			compose: `
services:
  app1:
    ports:
      - "80:8080"
  app2:
    ports:
      - "80:9000"
`,
			expected: []PortSpec{
				{Port: 80, Protocol: "tcp"},
			},
		},
		{
			name: "Same port different protocols",
			compose: `
services:
  dns:
    ports:
      - "53:53/tcp"
      - "53:53/udp"
`,
			expected: []PortSpec{
				{Port: 53, Protocol: "tcp"},
				{Port: 53, Protocol: "udp"},
			},
		},
		{
			name:     "Empty compose",
			compose:  ``,
			expected: []PortSpec{},
		},
		{
			name: "No ports section",
			compose: `
services:
  app:
    image: nginx
`,
			expected: []PortSpec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPortsFromCompose(tt.compose)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d ports, got %d", len(tt.expected), len(result))
				t.Logf("Expected: %+v", tt.expected)
				t.Logf("Got: %+v", result)
				return
			}

			// Create maps for easier comparison (order doesn't matter)
			expectedMap := make(map[string]bool)
			for _, spec := range tt.expected {
				key := formatPortSpec(spec)
				expectedMap[key] = true
			}

			resultMap := make(map[string]bool)
			for _, spec := range result {
				key := formatPortSpec(spec)
				resultMap[key] = true
			}

			for key := range expectedMap {
				if !resultMap[key] {
					t.Errorf("Expected port %s not found in result", key)
				}
			}

			for key := range resultMap {
				if !expectedMap[key] {
					t.Errorf("Unexpected port %s found in result", key)
				}
			}
		})
	}
}

// TestParseUFWPorts tests UFW port parsing
func TestParseUFWPorts(t *testing.T) {
	service := &FirewallService{}

	tests := []struct {
		name     string
		output   string
		expected []int
	}{
		{
			name: "Standard UFW output",
			output: `
Status: active

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       Anywhere
8080/tcp                   ALLOW       Anywhere
`,
			expected: []int{22, 80, 443, 8080},
		},
		{
			name: "UFW with UDP ports",
			output: `
Status: active

To                         Action      From
--                         ------      ----
53/udp                     ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere
`,
			expected: []int{53, 80},
		},
		{
			name: "UFW with duplicates",
			output: `
Status: active

To                         Action      From
--                         ------      ----
80/tcp                     ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere (v6)
`,
			expected: []int{80},
		},
		{
			name:     "Empty UFW output",
			output:   `Status: active`,
			expected: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseUFWPorts(tt.output)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d ports, got %d", len(tt.expected), len(result))
				return
			}

			resultMap := make(map[int]bool)
			for _, port := range result {
				resultMap[port] = true
			}

			for _, expected := range tt.expected {
				if !resultMap[expected] {
					t.Errorf("Expected port %d not found", expected)
				}
			}
		})
	}
}

// TestFormatPortSpecs tests the port spec formatting function
func TestFormatPortSpecs(t *testing.T) {
	tests := []struct {
		name     string
		specs    []PortSpec
		expected string
	}{
		{
			name:     "Empty list",
			specs:    []PortSpec{},
			expected: "none",
		},
		{
			name: "Single port",
			specs: []PortSpec{
				{Port: 80, Protocol: "tcp"},
			},
			expected: "80/tcp",
		},
		{
			name: "Multiple ports",
			specs: []PortSpec{
				{Port: 80, Protocol: "tcp"},
				{Port: 443, Protocol: "tcp"},
				{Port: 53, Protocol: "udp"},
			},
			expected: "80/tcp, 443/tcp, 53/udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPortSpecs(tt.specs)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Helper function for tests
func formatPortSpec(spec PortSpec) string {
	return fmt.Sprintf("%d/%s", spec.Port, spec.Protocol)
}
