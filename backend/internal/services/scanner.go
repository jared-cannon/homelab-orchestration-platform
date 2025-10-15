package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"github.com/mdlayher/arp"
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
	ID              string             `json:"id"`
	Status          string             `json:"status"` // "scanning", "completed", "failed"
	Phase           string             `json:"phase"`  // "ping", "ssh_scan", "credential_test", "completed"
	TotalHosts      int                `json:"total_hosts"`
	ScannedHosts    int                `json:"scanned_hosts"`
	DiscoveredCount int                `json:"discovered_count"`
	CurrentIP       string             `json:"current_ip,omitempty"`
	ScanRate        float64            `json:"scan_rate"` // IPs per second
	Devices         []DiscoveredDevice `json:"devices"`
	Error           string             `json:"error,omitempty"`
	StartedAt       time.Time          `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
}

// WebSocketHub interface for broadcasting messages (avoids circular dependency)
type WebSocketHub interface {
	Broadcast(channel string, event string, data interface{})
}

// ScannerService handles network device discovery
type ScannerService struct {
	sshClient       *ssh.Client
	validator       *ValidatorService
	credMatcher     *CredentialMatcher
	db              *gorm.DB
	wsHub           WebSocketHub
	scans           map[string]*ScanProgress
	cancelFuncs     map[string]context.CancelFunc
	mu              sync.RWMutex
	maxConcurrent   int
	scanExpiry      time.Duration
	cleanupInterval time.Duration
	shutdownChan    chan struct{}
}

// NewScannerService creates a new scanner service
func NewScannerService(db *gorm.DB, sshClient *ssh.Client, credMatcher *CredentialMatcher, wsHub WebSocketHub) *ScannerService {
	s := &ScannerService{
		db:              db,
		sshClient:       sshClient,
		validator:       NewValidatorService(sshClient),
		credMatcher:     credMatcher,
		wsHub:           wsHub,
		scans:           make(map[string]*ScanProgress),
		cancelFuncs:     make(map[string]context.CancelFunc),
		maxConcurrent:   3,                   // Max 3 concurrent scans
		scanExpiry:      30 * time.Minute,    // Scans expire after 30 minutes
		cleanupInterval: 5 * time.Minute,     // Cleanup every 5 minutes
		shutdownChan:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go s.cleanupExpiredScans()

	return s
}

// Shutdown gracefully stops the scanner service
func (s *ScannerService) Shutdown() {
	close(s.shutdownChan)

	// Cancel all active scans
	s.mu.Lock()
	for scanID, cancel := range s.cancelFuncs {
		cancel()
		if progress, exists := s.scans[scanID]; exists {
			progress.Status = "cancelled"
			now := time.Now()
			progress.CompletedAt = &now
		}
	}
	s.mu.Unlock()
}

// cleanupExpiredScans periodically removes expired scans from memory
func (s *ScannerService) cleanupExpiredScans() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			log.Printf("[Scanner] Cleanup goroutine stopping")
			return
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()

			for scanID, progress := range s.scans {
				// Remove completed scans older than expiry time
				if progress.Status == "completed" || progress.Status == "failed" || progress.Status == "cancelled" {
					if progress.CompletedAt != nil && now.Sub(*progress.CompletedAt) > s.scanExpiry {
						log.Printf("[Scanner] Cleaning up expired scan %s", scanID)
						delete(s.scans, scanID)
						delete(s.cancelFuncs, scanID)
					}
				}
			}

			s.mu.Unlock()
		}
	}
}

// StartScan initiates a network scan and returns a scan ID
func (s *ScannerService) StartScan(ctx context.Context, cidr string) (string, error) {
	// Check if we're at max concurrent scans
	s.mu.RLock()
	activeScans := 0
	for _, progress := range s.scans {
		if progress.Status == "scanning" {
			activeScans++
		}
	}
	s.mu.RUnlock()

	if activeScans >= s.maxConcurrent {
		return "", fmt.Errorf("maximum concurrent scans (%d) reached, please wait for existing scans to complete", s.maxConcurrent)
	}

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

	// Create cancelable context for this scan
	scanCtx, cancel := context.WithCancel(ctx)

	s.mu.Lock()
	s.scans[scanID] = progress
	s.cancelFuncs[scanID] = cancel
	s.mu.Unlock()

	// Start scanning in background
	go s.performScan(scanCtx, scanID, cidr)

	return scanID, nil
}

// CancelScan cancels a running scan
func (s *ScannerService) CancelScan(scanID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cancel, exists := s.cancelFuncs[scanID]
	if !exists {
		return fmt.Errorf("scan not found")
	}

	progress, exists := s.scans[scanID]
	if !exists {
		return fmt.Errorf("scan not found")
	}

	if progress.Status != "scanning" {
		return fmt.Errorf("scan is not running")
	}

	// Cancel the context
	cancel()

	// Mark as cancelled
	progress.Status = "cancelled"
	now := time.Now()
	progress.CompletedAt = &now

	log.Printf("[Scanner] Cancelled scan %s", scanID)

	return nil
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
	log.Printf("[Scanner] Starting scan %s for CIDR %s", scanID, cidr)

	s.mu.Lock()
	progress := s.scans[scanID]
	progress.Phase = "ping"
	s.mu.Unlock()

	// Broadcast initial scan start
	s.broadcastProgress(scanID)

	// Get list of IPs to scan
	ips, err := s.generateIPList(cidr)
	if err != nil {
		log.Printf("[Scanner] Error generating IP list: %v", err)
		s.updateScanError(scanID, err.Error())
		s.broadcastProgress(scanID)
		return
	}

	log.Printf("[Scanner] Generated %d IPs to scan", len(ips))

	// Run both ping-based and mDNS discovery in parallel
	var pingHosts, mdnsHosts []string
	var discoveryWg sync.WaitGroup

	// Start ping scan
	discoveryWg.Add(1)
	go func() {
		defer discoveryWg.Done()
		pingHosts = s.scanWithARP(ips)
		log.Printf("[Scanner] Ping scan found %d active hosts", len(pingHosts))
	}()

	// Start mDNS discovery
	discoveryWg.Add(1)
	go func() {
		defer discoveryWg.Done()
		mdnsHosts = s.scanWithMDNS(ctx, 5*time.Second)
		log.Printf("[Scanner] mDNS scan found %d hosts", len(mdnsHosts))
	}()

	// Wait for both discovery methods to complete
	discoveryWg.Wait()

	// Combine and deduplicate results
	hostMap := make(map[string]bool)
	for _, host := range pingHosts {
		hostMap[host] = true
	}
	for _, host := range mdnsHosts {
		hostMap[host] = true
	}

	// Convert back to slice
	var activeHosts []string
	for host := range hostMap {
		activeHosts = append(activeHosts, host)
	}

	log.Printf("[Scanner] Combined discovery found %d unique active hosts (ping: %d, mDNS: %d), now scanning for SSH and services",
		len(activeHosts), len(pingHosts), len(mdnsHosts))

	// Update phase to SSH scanning
	s.mu.Lock()
	progress.Phase = "ssh_scan"
	progress.ScannedHosts = 0 // Reset for SSH scan phase
	progress.TotalHosts = len(activeHosts)
	s.mu.Unlock()
	s.broadcastProgress(scanID)

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

				// Update current IP being scanned
				s.mu.Lock()
				progress.CurrentIP = ipAddr
				s.mu.Unlock()
				s.broadcastProgress(scanID)

				device := s.scanHost(ipAddr)
				if device != nil {
					deviceChan <- *device
				}

				s.mu.Lock()
				progress.ScannedHosts++
				s.mu.Unlock()
				s.broadcastProgress(scanID)
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

		// Broadcast when new device is discovered
		s.broadcastProgress(scanID)
	}

	// Mark scan as completed
	now := time.Now()
	s.mu.Lock()
	progress.Status = "completed"
	progress.Phase = "completed"
	progress.CurrentIP = ""
	progress.CompletedAt = &now
	s.mu.Unlock()

	// Broadcast final completion
	s.broadcastProgress(scanID)
}

// scanHost checks a single host for SSH availability, services, and credentials
func (s *ScannerService) scanHost(ip string) *DiscoveredDevice {
	log.Printf("[Scanner] Scanning host %s for SSH and services", ip)

	// Check if SSH port is open
	if !s.isPortOpen(ip, 22, 2*time.Second) {
		log.Printf("[Scanner] Host %s does not have SSH port open", ip)
		return nil
	}

	log.Printf("[Scanner] Host %s has SSH port open", ip)

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

// scanWithARP uses concurrent ICMP ping to quickly find active hosts on the network
func (s *ScannerService) scanWithARP(ips []string) []string {
	log.Printf("[Scanner] Starting concurrent ping scan for %d IPs", len(ips))

	var activeHosts []string
	var mu sync.Mutex

	// Ping hosts concurrently in batches
	batchSize := 50 // Scan 50 IPs at a time
	for i := 0; i < len(ips); i += batchSize {
		end := i + batchSize
		if end > len(ips) {
			end = len(ips)
		}

		batch := ips[i:end]
		var wg sync.WaitGroup

		for _, ip := range batch {
			wg.Add(1)
			go func(ipAddr string) {
				defer wg.Done()

				if s.isPingReachableGoPing(ipAddr) {
					log.Printf("[Scanner] Host %s is reachable", ipAddr)
					mu.Lock()
					activeHosts = append(activeHosts, ipAddr)
					mu.Unlock()
				}
			}(ip)
		}

		wg.Wait()
	}

	log.Printf("[Scanner] Ping scan complete. Found %d active hosts", len(activeHosts))
	return activeHosts
}

// isPingReachableGoPing checks if a host responds to ICMP ping using go-ping library
func (s *ScannerService) isPingReachableGoPing(ip string) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return false
	}

	// Use unprivileged mode (UDP) to avoid requiring sudo
	pinger.SetPrivileged(false)
	pinger.Count = 1
	pinger.Timeout = 2 * time.Second

	err = pinger.Run()
	if err != nil {
		return false
	}

	stats := pinger.Statistics()
	return stats.PacketsRecv > 0
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

// broadcastProgress sends real-time scan progress via WebSocket
func (s *ScannerService) broadcastProgress(scanID string) {
	if s.wsHub == nil {
		return
	}

	s.mu.RLock()
	progress, exists := s.scans[scanID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// Calculate scan rate (IPs per second)
	elapsed := time.Since(progress.StartedAt).Seconds()
	if elapsed > 0 {
		progress.ScanRate = float64(progress.ScannedHosts) / elapsed
	}

	// Broadcast to scanner channel
	s.wsHub.Broadcast("scanner", "scan:progress", progress)
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

// GetMACAddress attempts to get the MAC address for an IP using ARP protocol
func (s *ScannerService) GetMACAddress(ipStr string) (string, error) {
	targetIP := net.ParseIP(ipStr)
	if targetIP == nil {
		return "", fmt.Errorf("invalid IP address")
	}

	// ARP only works with IPv4
	if targetIP.To4() == nil {
		return "", fmt.Errorf("invalid IP address")
	}

	// Get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	// Try each interface that might be on the same network
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
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			// Check if target IP is in this subnet
			if !ipNet.Contains(targetIP) {
				continue
			}

			// Try ARP request on this interface
			mac, err := s.sendARPRequest(&iface, targetIP)
			if err == nil {
				return mac.String(), nil
			}
		}
	}

	return "", fmt.Errorf("MAC address not found via ARP")
}

// sendARPRequest sends an ARP request and returns the MAC address
func (s *ScannerService) sendARPRequest(iface *net.Interface, targetIP net.IP) (net.HardwareAddr, error) {
	client, err := arp.Dial(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to create ARP client: %w", err)
	}
	defer client.Close()

	// Convert net.IP to netip.Addr (required by mdlayher/arp library)
	targetAddr, ok := netip.AddrFromSlice(targetIP.To4())
	if !ok {
		return nil, fmt.Errorf("invalid IPv4 address")
	}

	// Set read deadline
	client.SetDeadline(time.Now().Add(2 * time.Second))

	// Send ARP request
	if err := client.Request(targetAddr); err != nil {
		return nil, fmt.Errorf("ARP request failed: %w", err)
	}

	// Try to receive response (multiple attempts as responses might be delayed)
	for i := 0; i < 3; i++ {
		packet, _, err := client.Read()
		if err != nil {
			if i == 2 {
				return nil, fmt.Errorf("no ARP response: %w", err)
			}
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Check if this is a reply for our target IP
		if packet.Operation == arp.OperationReply && packet.SenderIP.Compare(targetAddr) == 0 {
			return packet.SenderHardwareAddr, nil
		}
	}

	return nil, fmt.Errorf("no matching ARP response received")
}

// scanWithMDNS discovers devices advertising services via mDNS/Bonjour
func (s *ScannerService) scanWithMDNS(ctx context.Context, timeout time.Duration) []string {
	log.Printf("[Scanner] Starting mDNS discovery scan (timeout: %v)", timeout)

	// Services to discover - common homelab services
	services := []string{
		"_ssh._tcp",         // SSH servers
		"_sftp-ssh._tcp",    // SFTP over SSH
		"_http._tcp",        // HTTP servers
		"_https._tcp",       // HTTPS servers
		"_smb._tcp",         // Samba/Windows file sharing
		"_afpovertcp._tcp",  // AFP (Apple File Protocol)
		"_workstation._tcp", // Network workstations
		"_device-info._tcp", // Device information
	}

	discoveredIPs := make(map[string]bool)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, service := range services {
		wg.Add(1)
		go func(svc string) {
			defer wg.Done()

			// Create resolver
			resolver, err := zeroconf.NewResolver(nil)
			if err != nil {
				log.Printf("[Scanner] Failed to create mDNS resolver for %s: %v", svc, err)
				return
			}

			// Channel to receive service entries
			entries := make(chan *zeroconf.ServiceEntry, 100)

			// Browse for the service
			browseCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			err = resolver.Browse(browseCtx, svc, "local.", entries)
			if err != nil {
				log.Printf("[Scanner] mDNS browse failed for %s: %v", svc, err)
				return
			}

			// Collect discovered IPs
			go func() {
				for entry := range entries {
					if entry == nil {
						continue
					}

					// Log discovery
					log.Printf("[Scanner] mDNS found: %s (%s) on %v",
						entry.Instance, svc, entry.AddrIPv4)

					// Add all IPv4 addresses
					for _, ip := range entry.AddrIPv4 {
						ipStr := ip.String()
						mu.Lock()
						discoveredIPs[ipStr] = true
						mu.Unlock()
					}
				}
			}()

			<-browseCtx.Done()
		}(service)
	}

	wg.Wait()

	// Convert map to slice
	var ips []string
	for ip := range discoveredIPs {
		ips = append(ips, ip)
	}

	log.Printf("[Scanner] mDNS discovery complete. Found %d unique IPs", len(ips))
	return ips
}
