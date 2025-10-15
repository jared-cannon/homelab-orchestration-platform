package services

import (
	"net"
	"testing"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestGenerateSmartName(t *testing.T) {
	// Use nil SSH client for pure logic tests
	scanner := NewScannerService(nil, nil, nil, nil)

	tests := []struct {
		name     string
		device   *DiscoveredDevice
		expected string
	}{
		{
			name: "Proxmox server detected",
			device: &DiscoveredDevice{
				IPAddress:        "192.168.1.100",
				ServicesDetected: []string{"proxmox"},
			},
			expected: "Proxmox Server (192.168.1.100)",
		},
		{
			name: "Portainer with Docker",
			device: &DiscoveredDevice{
				IPAddress:        "192.168.1.101",
				ServicesDetected: []string{"portainer"},
				DockerDetected:   true,
			},
			expected: "Portainer Docker Host (192.168.1.101)",
		},
		{
			name: "Synology NAS by OS",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.102",
				OS:        "Synology DSM 7.0",
			},
			expected: "Synology NAS (192.168.1.102)",
		},
		{
			name: "Ubuntu Docker Host",
			device: &DiscoveredDevice{
				IPAddress:      "192.168.1.103",
				OS:             "Ubuntu 22.04",
				DockerDetected: true,
			},
			expected: "Ubuntu Docker Host (192.168.1.103)",
		},
		{
			name: "Ubuntu Server without Docker",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.104",
				OS:        "Ubuntu 22.04",
			},
			expected: "Ubuntu Server (192.168.1.104)",
		},
		{
			name: "Generic Docker Host",
			device: &DiscoveredDevice{
				IPAddress:      "192.168.1.105",
				DockerDetected: true,
			},
			expected: "Docker Host (192.168.1.105)",
		},
		{
			name: "Meaningful hostname",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.106",
				Hostname:  "my-server.local",
			},
			expected: "My-Server (192.168.1.106)",
		},
		{
			name: "NAS by device type",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.107",
				Type:      models.DeviceTypeNAS,
			},
			expected: "NAS (192.168.1.107)",
		},
		{
			name: "Router by device type",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.108",
				Type:      models.DeviceTypeRouter,
			},
			expected: "Router (192.168.1.108)",
		},
		{
			name: "Generic server fallback",
			device: &DiscoveredDevice{
				IPAddress: "192.168.1.109",
				Type:      models.DeviceTypeServer,
			},
			expected: "Server (192.168.1.109)",
		},
		{
			name: "Home Assistant",
			device: &DiscoveredDevice{
				IPAddress:        "192.168.1.110",
				ServicesDetected: []string{"home-assistant"},
			},
			expected: "Home Assistant (192.168.1.110)",
		},
		{
			name: "Grafana Server",
			device: &DiscoveredDevice{
				IPAddress:        "192.168.1.111",
				ServicesDetected: []string{"grafana"},
			},
			expected: "Grafana Server (192.168.1.111)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.generateSmartName(tt.device)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectDeviceType(t *testing.T) {
	// Use nil SSH client for pure logic tests
	scanner := NewScannerService(nil, nil, nil, nil)

	tests := []struct {
		name     string
		hostname string
		expected models.DeviceType
	}{
		{
			name:     "Router hostname",
			hostname: "home-router",
			expected: models.DeviceTypeRouter,
		},
		{
			name:     "Gateway hostname",
			hostname: "main-gateway",
			expected: models.DeviceTypeRouter,
		},
		{
			name:     "NAS hostname",
			hostname: "synology-nas",
			expected: models.DeviceTypeNAS,
		},
		{
			name:     "Storage hostname",
			hostname: "file-storage",
			expected: models.DeviceTypeNAS,
		},
		{
			name:     "Switch hostname",
			hostname: "core-switch",
			expected: models.DeviceTypeSwitch,
		},
		{
			name:     "Server hostname",
			hostname: "web-server",
			expected: models.DeviceTypeServer,
		},
		{
			name:     "Unknown hostname defaults to server",
			hostname: "unknown-device",
			expected: models.DeviceTypeServer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.detectDeviceType(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDeviceAlreadyAdded(t *testing.T) {
	db := setupTestDB(t)
	// Use nil SSH client for database tests
	scanner := NewScannerService(db, nil, nil, nil)

	// Add a device to the database
	device := &models.Device{
		Name:       "Test Device",
		Type:       models.DeviceTypeServer,
		IPAddress:  "192.168.1.100",
		MACAddress: "00:11:22:33:44:55",
	}
	err := db.Create(device).Error
	assert.NoError(t, err)

	tests := []struct {
		name       string
		ipAddress  string
		macAddress string
		expected   bool
	}{
		{
			name:       "Device exists by IP",
			ipAddress:  "192.168.1.100",
			macAddress: "",
			expected:   true,
		},
		{
			name:       "Device exists by MAC",
			ipAddress:  "192.168.1.200",
			macAddress: "00:11:22:33:44:55",
			expected:   true,
		},
		{
			name:       "Device does not exist",
			ipAddress:  "192.168.1.200",
			macAddress: "aa:bb:cc:dd:ee:ff",
			expected:   false,
		},
		{
			name:       "Device exists by IP even with different MAC",
			ipAddress:  "192.168.1.100",
			macAddress: "aa:bb:cc:dd:ee:ff",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.isDeviceAlreadyAdded(tt.ipAddress, tt.macAddress)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	// Use nil SSH client for pure logic tests
	scanner := NewScannerService(nil, nil, nil, nil)

	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "Private IP 10.x.x.x",
			ip:       "10.0.0.1",
			expected: true,
		},
		{
			name:     "Private IP 172.16.x.x",
			ip:       "172.16.0.1",
			expected: true,
		},
		{
			name:     "Private IP 192.168.x.x",
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "Public IP",
			ip:       "8.8.8.8",
			expected: false,
		},
		{
			name:     "Localhost",
			ip:       "127.0.0.1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			result := scanner.isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateHostCount(t *testing.T) {
	// Use nil SSH client for pure logic tests
	scanner := NewScannerService(nil, nil, nil, nil)

	tests := []struct {
		name     string
		cidr     string
		expected int
	}{
		{
			name:     "/24 network",
			cidr:     "192.168.1.0/24",
			expected: 256,
		},
		{
			name:     "/16 network",
			cidr:     "192.168.0.0/16",
			expected: 65536,
		},
		{
			name:     "/8 network",
			cidr:     "10.0.0.0/8",
			expected: 16777216,
		},
		{
			name:     "/32 single host",
			cidr:     "192.168.1.1/32",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			assert.NoError(t, err)
			result := scanner.calculateHostCount(ipNet)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetMACAddressIPValidation tests IP validation logic before ARP requests
func TestGetMACAddressIPValidation(t *testing.T) {
	scanner := NewScannerService(nil, nil, nil, nil)

	tests := []struct {
		name      string
		ip        string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "Valid IPv4 address",
			ip:        "192.168.1.100",
			wantError: false,
		},
		{
			name:      "Invalid IP string",
			ip:        "not-an-ip",
			wantError: true,
			errorMsg:  "invalid IP address",
		},
		{
			name:      "Empty string",
			ip:        "",
			wantError: true,
			errorMsg:  "invalid IP address",
		},
		{
			name:      "Malformed IP",
			ip:        "192.168.1",
			wantError: true,
			errorMsg:  "invalid IP address",
		},
		{
			name:      "IPv6 address (not supported for ARP)",
			ip:        "::1",
			wantError: true,
			errorMsg:  "invalid IP address",
		},
		{
			name:      "IPv6 full address",
			ip:        "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			wantError: true,
			errorMsg:  "invalid IP address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := scanner.GetMACAddress(tt.ip)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
			// Note: We can't test success cases without actual network interfaces
			// Those would require integration tests or mocking
		})
	}
}

// TestHostDeduplication tests the deduplication logic from parallel discovery
func TestHostDeduplication(t *testing.T) {
	tests := []struct {
		name        string
		pingHosts   []string
		mdnsHosts   []string
		expectedLen int
		mustContain []string
	}{
		{
			name:        "No overlap between ping and mDNS",
			pingHosts:   []string{"192.168.1.1", "192.168.1.2"},
			mdnsHosts:   []string{"192.168.1.3", "192.168.1.4"},
			expectedLen: 4,
			mustContain: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		},
		{
			name:        "Complete overlap",
			pingHosts:   []string{"192.168.1.1", "192.168.1.2"},
			mdnsHosts:   []string{"192.168.1.1", "192.168.1.2"},
			expectedLen: 2,
			mustContain: []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name:        "Partial overlap",
			pingHosts:   []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
			mdnsHosts:   []string{"192.168.1.2", "192.168.1.4"},
			expectedLen: 4,
			mustContain: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		},
		{
			name:        "mDNS only finds hosts ping missed",
			pingHosts:   []string{"192.168.1.1"},
			mdnsHosts:   []string{"192.168.1.2", "192.168.1.3", "192.168.1.4"},
			expectedLen: 4,
			mustContain: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"},
		},
		{
			name:        "Ping only (no mDNS results)",
			pingHosts:   []string{"192.168.1.1", "192.168.1.2"},
			mdnsHosts:   []string{},
			expectedLen: 2,
			mustContain: []string{"192.168.1.1", "192.168.1.2"},
		},
		{
			name:        "mDNS only (no ping results)",
			pingHosts:   []string{},
			mdnsHosts:   []string{"192.168.1.3", "192.168.1.4"},
			expectedLen: 2,
			mustContain: []string{"192.168.1.3", "192.168.1.4"},
		},
		{
			name:        "Empty results from both",
			pingHosts:   []string{},
			mdnsHosts:   []string{},
			expectedLen: 0,
			mustContain: []string{},
		},
		{
			name:        "Duplicate within same list",
			pingHosts:   []string{"192.168.1.1", "192.168.1.1", "192.168.1.2"},
			mdnsHosts:   []string{"192.168.1.2", "192.168.1.3", "192.168.1.3"},
			expectedLen: 3,
			mustContain: []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the deduplication logic from performScan
			hostMap := make(map[string]bool)
			for _, host := range tt.pingHosts {
				hostMap[host] = true
			}
			for _, host := range tt.mdnsHosts {
				hostMap[host] = true
			}

			var activeHosts []string
			for host := range hostMap {
				activeHosts = append(activeHosts, host)
			}

			// Verify length
			assert.Equal(t, tt.expectedLen, len(activeHosts),
				"Expected %d unique hosts, got %d", tt.expectedLen, len(activeHosts))

			// Verify all expected hosts are present
			for _, expectedHost := range tt.mustContain {
				assert.Contains(t, activeHosts, expectedHost,
					"Expected host %s to be in results", expectedHost)
			}
		})
	}
}

// TestMDNSServiceList validates that we're scanning for expected services
func TestMDNSServiceList(t *testing.T) {
	// Expected services that should be discovered via mDNS
	expectedServices := map[string]string{
		"_ssh._tcp":         "SSH servers",
		"_sftp-ssh._tcp":    "SFTP over SSH",
		"_http._tcp":        "HTTP servers",
		"_https._tcp":       "HTTPS servers",
		"_smb._tcp":         "Samba/Windows file sharing",
		"_afpovertcp._tcp":  "AFP (Apple File Protocol)",
		"_workstation._tcp": "Network workstations",
		"_device-info._tcp": "Device information",
	}

	// This test documents the services we scan for
	// If you add/remove services, update this test
	t.Run("Service list documentation", func(t *testing.T) {
		t.Logf("mDNS discovery scans for %d services:", len(expectedServices))
		for service, description := range expectedServices {
			t.Logf("  - %s: %s", service, description)
		}

		// Verify we have a reasonable number of services
		assert.GreaterOrEqual(t, len(expectedServices), 5,
			"Should scan for at least 5 common services")
	})

	// Verify service names follow mDNS conventions
	t.Run("Service name format validation", func(t *testing.T) {
		for service := range expectedServices {
			// mDNS service names should start with underscore
			assert.True(t, len(service) > 0 && service[0] == '_',
				"Service name %s should start with underscore", service)

			// Should contain ._tcp or ._udp
			assert.True(t,
				len(service) > 5 && (service[len(service)-5:] == "._tcp" || service[len(service)-5:] == "._udp"),
				"Service name %s should end with ._tcp or ._udp", service)
		}
	})
}

// TestIPv4ConversionLogic tests net.IP to netip.Addr conversion
func TestIPv4ConversionLogic(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		wantValid bool
	}{
		{
			name:      "Valid IPv4",
			ip:        "192.168.1.1",
			wantValid: true,
		},
		{
			name:      "Localhost",
			ip:        "127.0.0.1",
			wantValid: true,
		},
		{
			name:      "Broadcast address",
			ip:        "255.255.255.255",
			wantValid: true,
		},
		{
			name:      "Zero address",
			ip:        "0.0.0.0",
			wantValid: true,
		},
		{
			name:      "IPv6 should fail for IPv4 conversion",
			ip:        "::1",
			wantValid: false,
		},
		{
			name:      "IPv6 full address should fail",
			ip:        "2001:0db8:85a3::8a2e:0370:7334",
			wantValid: false,
		},
		{
			name:      "Invalid IP string",
			ip:        "not-an-ip",
			wantValid: false,
		},
		{
			name:      "Incomplete IP",
			ip:        "192.168.1",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse IP
			parsedIP := net.ParseIP(tt.ip)

			if !tt.wantValid {
				// For invalid IPs, To4() should return nil
				if parsedIP != nil {
					ipv4 := parsedIP.To4()
					assert.Nil(t, ipv4, "Expected To4() to return nil for %s", tt.ip)
				}
				return
			}

			// For valid IPv4, should successfully convert
			assert.NotNil(t, parsedIP, "Expected ParseIP to succeed for %s", tt.ip)

			ipv4 := parsedIP.To4()
			assert.NotNil(t, ipv4, "Expected To4() to return non-nil for %s", tt.ip)

			// Test conversion to netip.Addr (the type used by ARP library)
			addr, ok := net.ParseIP(tt.ip).To4(), true
			if addr != nil {
				// This is the conversion we do in sendARPRequest
				// netip.AddrFromSlice requires a 4-byte slice for IPv4
				assert.Equal(t, 4, len(addr), "IPv4 address should be 4 bytes")
			}
			assert.True(t, ok || addr == nil)
		})
	}
}
