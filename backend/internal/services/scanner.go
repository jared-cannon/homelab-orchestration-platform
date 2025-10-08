package services

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DiscoveredDevice represents a device found during network scanning
type DiscoveredDevice struct {
	IPAddress        string              `json:"ip_address"`
	MACAddress       string              `json:"mac_address,omitempty"`
	Hostname         string              `json:"hostname,omitempty"`
	Type             models.DeviceType   `json:"type"`
	SSHAvailable     bool                `json:"ssh_available"`
	DockerDetected   bool                `json:"docker_detected"`
	ServicesDetected []string            `json:"services_detected,omitempty"` // e.g., ["docker", "portainer", "proxmox"]
	OS               string              `json:"os,omitempty"`                // e.g., "Ubuntu 22.04", "Synology DSM"
	Status           string              `json:"status"`                      // "discovered", "checking_credentials", "ready", "needs_credentials", "already_added"
	CredentialStatus string              `json:"credential_status,omitempty"` // "working", "failed", "untested"
	CredentialID     string              `json:"credential_id,omitempty"`     // ID of working credential
	AlreadyAdded     bool                `json:"already_added"`               // True if device already exists in database
}

// ScanProgress represents the current state of a network scan
type ScanProgress struct {
	ID              string              `json:"id"`
	Status          string              `json:"status"` // "scanning", "completed", "failed"
	TotalHosts      int                 `json:"total_hosts"`
	ScannedHosts    int                 `json:"scanned_hosts"`
	DiscoveredCount int                 `json:"discovered_count"`
	Devices         []DiscoveredDevice  `json:"devices"`
	Error           string              `json:"error,omitempty"`
	StartedAt       time.Time           `json:"started_at"`
	CompletedAt     *time.Time          `json:"completed_at,omitempty"`
}

// ScannerService handles network device discovery
type ScannerService struct {
	sshClient   *ssh.Client
	validator   *ValidatorService
	credMatcher *CredentialMatcher
	db          *gorm.DB
	scans       map[string]*ScanProgress
	mu          sync.RWMutex
}

// NewScannerService creates a new scanner service
func NewScannerService(db *gorm.DB, sshClient *ssh.Client, credMatcher *CredentialMatcher) *ScannerService {
	return &ScannerService{
		db:          db,
		sshClient:   sshClient,
		validator:   NewValidatorService(sshClient),
		credMatcher: credMatcher,
		scans:       make(map[string]*ScanProgress),
	}
}

// StartScan initiates a network scan and returns a scan ID
func (s *ScannerService) StartScan(ctx context.Context, cidr string) (string, error) {
	scanID := uuid.New().String()

	// Parse CIDR to determine total hosts
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}

	totalHosts := s.calculateHostCount(ipNet)

	// Create scan progress
	progress := &ScanProgress{
		ID:         scanID,
		Status:     "scanning",
		TotalHosts: totalHosts,
		Devices:    []DiscoveredDevice{},
		StartedAt:  time.Now(),
	}

	s.mu.Lock()
	s.scans[scanID] = progress
	s.mu.Unlock()

	// Start scanning in background
	go s.performScan(ctx, scanID, cidr)

	return scanID, nil
}

// GetScanProgress retrieves the current progress of a scan
func (s *ScannerService) GetScanProgress(scanID string) (*ScanProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	progress, exists := s.scans[scanID]
	if !exists {
		return nil, fmt.Errorf("scan not found")
	}

	return progress, nil
}

// performScan executes the actual network scanning
func (s *ScannerService) performScan(ctx context.Context, scanID, cidr string) {
	s.mu.Lock()
	progress := s.scans[scanID]
	s.mu.Unlock()

	// Get list of IPs to scan
	ips, err := s.generateIPList(cidr)
	if err != nil {
		s.updateScanError(scanID, err.Error())
		return
	}

	// Use ARP scan to find active hosts quickly
	activeHosts := s.scanWithARP(ips)

	// Scan each active host for SSH and Docker
	var wg sync.WaitGroup
	deviceChan := make(chan DiscoveredDevice, len(activeHosts))

	// Limit concurrent scans to avoid overwhelming the network
	semaphore := make(chan struct{}, 10)

	for _, ip := range activeHosts {
		wg.Add(1)
		go func(ipAddr string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()

				device := s.scanHost(ipAddr)
				if device != nil {
					deviceChan <- *device
				}

				s.mu.Lock()
				progress.ScannedHosts++
				s.mu.Unlock()
			}
		}(ip)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(deviceChan)
	}()

	// Collect discovered devices
	for device := range deviceChan {
		s.mu.Lock()
		progress.Devices = append(progress.Devices, device)
		progress.DiscoveredCount = len(progress.Devices)
		s.mu.Unlock()
	}

	// Mark scan as completed
	now := time.Now()
	s.mu.Lock()
	progress.Status = "completed"
	progress.CompletedAt = &now
	s.mu.Unlock()
}

// scanHost checks a single host for SSH availability, services, and credentials
func (s *ScannerService) scanHost(ip string) *DiscoveredDevice {
	// Check if SSH port is open
	if !s.isPortOpen(ip, 22, 2*time.Second) {
		return nil
	}

	device := &DiscoveredDevice{
		IPAddress:        ip,
		SSHAvailable:     true,
		Status:           "discovered",
		ServicesDetected: []string{},
		CredentialStatus: "untested",
	}

	// Try to get MAC address
	if mac, err := s.GetMACAddress(ip); err == nil {
		device.MACAddress = mac
	}

	// Try to get hostname via reverse DNS
	if hostname, err := s.getHostname(ip); err == nil {
		device.Hostname = hostname
	}

	// Detect device type based on hostname
	device.Type = s.detectDeviceType(device.Hostname)

	// Try to detect additional services by port scanning
	s.detectServicesByPorts(device)

	// Generate smart name after we have all detection info
	// This will be updated again after SSH detection if credentials work
	device.Hostname = s.generateSmartName(device)

	// Check if device already exists in database
	if s.isDeviceAlreadyAdded(device.IPAddress, device.MACAddress) {
		device.AlreadyAdded = true
		device.Status = "already_added"
		return device
	}

	// If credential matcher is available, try to find and test credentials
	if s.credMatcher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if workingCred, err := s.credMatcher.TestAllCredentials(ctx, ip, device.Hostname, device.Type, 22); err == nil {
			device.CredentialID = workingCred.ID
			device.CredentialStatus = "working"
			device.Status = "ready"

			// If we have working credentials, detect services via SSH
			s.detectServicesViaSSH(device, workingCred)

			// Regenerate smart name with OS info
			device.Hostname = s.generateSmartName(device)
		} else {
			device.CredentialStatus = "failed"
			device.Status = "needs_credentials"
		}
	}

	return device
}

// detectServicesByPorts detects services by scanning common ports
func (s *ScannerService) detectServicesByPorts(device *DiscoveredDevice) {
	ports := map[int]string{
		80:   "http",
		443:  "https",
		3000: "grafana",
		5000: "portainer",
		8006: "proxmox",
		9000: "portainer-alt",
		8123: "home-assistant",
	}

	for port, service := range ports {
		if s.isPortOpen(device.IPAddress, port, 1*time.Second) {
			device.ServicesDetected = append(device.ServicesDetected, service)
		}
	}
}

// detectServicesViaSSH detects services and OS via SSH commands
func (s *ScannerService) detectServicesViaSSH(device *DiscoveredDevice, cred *models.Credential) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	address := fmt.Sprintf("%s:22", device.IPAddress)

	// Decrypt credentials
	password, _ := s.credMatcher.credSvc.DecryptData(cred.Password)
	sshKey, _ := s.credMatcher.credSvc.DecryptData(cred.SSHKey)
	sshKeyPassphrase, _ := s.credMatcher.credSvc.DecryptData(cred.SSHKeyPassphrase)

	// Connect with credentials
	var err error
	if cred.Type == models.CredentialTypePassword {
		_, err = s.sshClient.ConnectWithPassword(address, cred.Username, password)
	} else {
		_, err = s.sshClient.ConnectWithKey(address, cred.Username, sshKey, sshKeyPassphrase)
	}

	if err != nil {
		return // Can't detect services without connection
	}

	// Check for Docker
	dockerCmd := "docker --version 2>/dev/null"
	if output, err := s.sshClient.Execute(address, dockerCmd); err == nil && output != "" {
		device.DockerDetected = true
		device.ServicesDetected = append(device.ServicesDetected, "docker")
	}

	// Detect OS
	osCmd := "cat /etc/os-release 2>/dev/null || uname -s"
	if osOutput, err := s.sshClient.Execute(address, osCmd); err == nil && osOutput != "" {
		// Parse OS info
		if strings.Contains(osOutput, "Ubuntu") {
			device.OS = "Ubuntu"
		} else if strings.Contains(osOutput, "Debian") {
			device.OS = "Debian"
		} else if strings.Contains(osOutput, "CentOS") {
			device.OS = "CentOS"
		} else if strings.Contains(osOutput, "Synology") {
			device.OS = "Synology DSM"
		} else if strings.Contains(osOutput, "Darwin") {
			device.OS = "macOS"
		} else {
			device.OS = strings.TrimSpace(strings.Split(osOutput, "\n")[0])
		}
	}

	// Note: ctx is not used but kept for future timeout implementation
	_ = ctx
}

// isPortOpen checks if a TCP port is open on a host
func (s *ScannerService) isPortOpen(host string, port int, timeout time.Duration) bool {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getHostname attempts to resolve the hostname for an IP
func (s *ScannerService) getHostname(ip string) (string, error) {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return "", err
	}
	// Remove trailing dot from FQDN
	return strings.TrimSuffix(names[0], "."), nil
}

// detectDeviceType tries to determine device type from hostname
func (s *ScannerService) detectDeviceType(hostname string) models.DeviceType {
	lower := strings.ToLower(hostname)

	// Common patterns for different device types
	patterns := map[models.DeviceType][]string{
		models.DeviceTypeRouter: {"router", "gateway", "gw", "edge"},
		models.DeviceTypeNAS:    {"nas", "storage", "fileserver", "synology", "qnap"},
		models.DeviceTypeSwitch: {"switch", "sw"},
		models.DeviceTypeServer: {"server", "srv", "node", "host"},
	}

	for deviceType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				return deviceType
			}
		}
	}

	// Default to server if unknown
	return models.DeviceTypeServer
}

// generateSmartName creates a user-friendly device name based on detected attributes
func (s *ScannerService) generateSmartName(device *DiscoveredDevice) string {
	// Priority 1: Specific service-based names
	for _, service := range device.ServicesDetected {
		switch service {
		case "proxmox":
			return fmt.Sprintf("Proxmox Server (%s)", device.IPAddress)
		case "portainer", "portainer-alt":
			if device.DockerDetected {
				return fmt.Sprintf("Portainer Docker Host (%s)", device.IPAddress)
			}
			return fmt.Sprintf("Portainer Server (%s)", device.IPAddress)
		case "home-assistant":
			return fmt.Sprintf("Home Assistant (%s)", device.IPAddress)
		case "grafana":
			return fmt.Sprintf("Grafana Server (%s)", device.IPAddress)
		}
	}

	// Priority 2: OS-based names
	if device.OS != "" {
		switch {
		case strings.Contains(device.OS, "Synology"):
			return fmt.Sprintf("Synology NAS (%s)", device.IPAddress)
		case strings.Contains(device.OS, "Ubuntu"):
			if device.DockerDetected {
				return fmt.Sprintf("Ubuntu Docker Host (%s)", device.IPAddress)
			}
			return fmt.Sprintf("Ubuntu Server (%s)", device.IPAddress)
		case strings.Contains(device.OS, "Debian"):
			if device.DockerDetected {
				return fmt.Sprintf("Debian Docker Host (%s)", device.IPAddress)
			}
			return fmt.Sprintf("Debian Server (%s)", device.IPAddress)
		case strings.Contains(device.OS, "CentOS"):
			return fmt.Sprintf("CentOS Server (%s)", device.IPAddress)
		case strings.Contains(device.OS, "macOS"):
			return fmt.Sprintf("macOS Host (%s)", device.IPAddress)
		}
	}

	// Priority 3: Docker detection
	if device.DockerDetected {
		return fmt.Sprintf("Docker Host (%s)", device.IPAddress)
	}

	// Priority 4: Hostname if meaningful (not an IP)
	if device.Hostname != "" && !strings.Contains(device.Hostname, device.IPAddress) {
		// Clean up hostname (remove domain suffix)
		hostname := strings.Split(device.Hostname, ".")[0]
		if hostname != "" && hostname != "localhost" {
			return fmt.Sprintf("%s (%s)", strings.Title(hostname), device.IPAddress)
		}
	}

	// Priority 5: Device type
	switch device.Type {
	case models.DeviceTypeRouter:
		return fmt.Sprintf("Router (%s)", device.IPAddress)
	case models.DeviceTypeNAS:
		return fmt.Sprintf("NAS (%s)", device.IPAddress)
	case models.DeviceTypeSwitch:
		return fmt.Sprintf("Switch (%s)", device.IPAddress)
	default:
		return fmt.Sprintf("Server (%s)", device.IPAddress)
	}
}

// scanWithARP uses ARP to quickly find active hosts on the network
func (s *ScannerService) scanWithARP(ips []string) []string {
	var activeHosts []string

	// Try using arp-scan if available (requires sudo on most systems)
	// Fall back to ping if arp-scan is not available
	for _, ip := range ips {
		if s.isPingReachable(ip) {
			activeHosts = append(activeHosts, ip)
		}
	}

	return activeHosts
}

// isPingReachable checks if a host responds to ping
func (s *ScannerService) isPingReachable(ip string) bool {
	// Use ping with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()
	return err == nil
}

// generateIPList generates a list of IPs from a CIDR range
func (s *ScannerService) generateIPList(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); s.incrementIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

// incrementIP increments an IP address
func (s *ScannerService) incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// calculateHostCount calculates the number of hosts in a CIDR range
func (s *ScannerService) calculateHostCount(ipNet *net.IPNet) int {
	ones, bits := ipNet.Mask.Size()
	return 1 << uint(bits-ones)
}

// updateScanError marks a scan as failed
func (s *ScannerService) updateScanError(scanID, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if progress, exists := s.scans[scanID]; exists {
		progress.Status = "failed"
		progress.Error = errMsg
		now := time.Now()
		progress.CompletedAt = &now
	}
}

// DetectLocalNetwork attempts to detect the local network CIDR
func (s *ScannerService) DetectLocalNetwork() (string, error) {
	// Get local network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Look for the primary non-loopback interface
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6 and loopback
			if ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}

			// Check if it's a private network
			if s.isPrivateIP(ipNet.IP) {
				return ipNet.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no private network interface found")
}

// isPrivateIP checks if an IP is in a private range
func (s *ScannerService) isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, ipNet, _ := net.ParseCIDR(cidr)
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// isDeviceAlreadyAdded checks if a device with the same IP or MAC already exists
func (s *ScannerService) isDeviceAlreadyAdded(ipAddress, macAddress string) bool {
	if s.db == nil {
		return false
	}

	var count int64

	// Check by IP address (primary match)
	s.db.Model(&models.Device{}).Where("ip_address = ?", ipAddress).Count(&count)
	if count > 0 {
		return true
	}

	// Check by MAC address if available (secondary match)
	if macAddress != "" {
		s.db.Model(&models.Device{}).Where("mac_address = ?", macAddress).Count(&count)
		if count > 0 {
			return true
		}
	}

	return false
}

// GetMACAddress attempts to get the MAC address for an IP using ARP
func (s *ScannerService) GetMACAddress(ip string) (string, error) {
	// Try to get MAC from ARP table
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "arp", "-n", ip)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse ARP output to extract MAC address
	// Format varies by OS, but typically contains MAC in format XX:XX:XX:XX:XX:XX
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}[:-]){5}([0-9a-fA-F]{2})`)
	matches := macRegex.FindString(string(output))
	if matches == "" {
		return "", fmt.Errorf("MAC address not found")
	}

	return matches, nil
}
