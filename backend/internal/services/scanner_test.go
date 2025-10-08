package services

import (
	"testing"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&models.Device{}, &models.Credential{})
	assert.NoError(t, err)

	return db
}

func TestGenerateSmartName(t *testing.T) {
	sshClient := ssh.NewClient()
	scanner := NewScannerService(nil, sshClient, nil)

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
			expected: "My-server (192.168.1.106)",
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
	sshClient := ssh.NewClient()
	scanner := NewScannerService(nil, sshClient, nil)

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
	sshClient := ssh.NewClient()
	scanner := NewScannerService(db, sshClient, nil)

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
	sshClient := ssh.NewClient()
	scanner := NewScannerService(nil, sshClient, nil)

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
			import "net"
			ip := net.ParseIP(tt.ip)
			result := scanner.isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateHostCount(t *testing.T) {
	sshClient := ssh.NewClient()
	scanner := NewScannerService(nil, sshClient, nil)

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
			import "net"
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			assert.NoError(t, err)
			result := scanner.calculateHostCount(ipNet)
			assert.Equal(t, tt.expected, result)
		})
	}
}
