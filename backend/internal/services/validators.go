package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
)

// ValidatorService handles pre-flight validation checks
type ValidatorService struct {
	sshClient *ssh.Client
}

// NewValidatorService creates a new validator service
func NewValidatorService(sshClient *ssh.Client) *ValidatorService {
	return &ValidatorService{
		sshClient: sshClient,
	}
}

// DockerInstalled checks if Docker is installed on the device
func (v *ValidatorService) DockerInstalled(host string) (bool, string, error) {
	output, err := v.sshClient.Execute(host, "docker --version")
	if err != nil {
		return false, "", fmt.Errorf("Docker not found. Install with: curl -fsSL https://get.docker.com | sh")
	}

	// Parse version
	version := strings.TrimSpace(output)
	return true, version, nil
}

// DockerRunning checks if Docker daemon is running
func (v *ValidatorService) DockerRunning(host string) (bool, error) {
	_, err := v.sshClient.Execute(host, "docker ps")
	if err != nil {
		return false, fmt.Errorf("Docker daemon not running. Start with: sudo systemctl start docker")
	}
	return true, nil
}

// PortAvailable checks if a port is available on the device
func (v *ValidatorService) PortAvailable(host string, port int) (bool, error) {
	// Try to check if port is in use using netstat or ss
	output, err := v.sshClient.Execute(host, fmt.Sprintf("ss -tuln | grep ':%d ' || netstat -tuln | grep ':%d '", port, port))

	// If command returns no output or error, port is likely available
	if err != nil || output == "" {
		return true, nil
	}

	// Port is in use
	return false, fmt.Errorf("port %d is already in use", port)
}

// CheckResources checks available memory and disk space
func (v *ValidatorService) CheckResources(host string, requiredRAM int64, requiredDisk int64) error {
	// Check available memory (in MB)
	memOutput, err := v.sshClient.Execute(host, "free -m | awk 'NR==2 {print $7}'")
	if err != nil {
		return fmt.Errorf("failed to check memory: %w", err)
	}

	availableRAM, err := strconv.ParseInt(strings.TrimSpace(memOutput), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse memory output: %w", err)
	}

	requiredRAMMB := requiredRAM / (1024 * 1024) // Convert bytes to MB
	if availableRAM < requiredRAMMB {
		return fmt.Errorf("insufficient memory: need %dMB, have %dMB available", requiredRAMMB, availableRAM)
	}

	// Check available disk space (in GB)
	diskOutput, err := v.sshClient.Execute(host, "df -BG / | awk 'NR==2 {print $4}' | sed 's/G//'")
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	availableDisk, err := strconv.ParseInt(strings.TrimSpace(diskOutput), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse disk output: %w", err)
	}

	requiredDiskGB := requiredDisk / (1024 * 1024 * 1024) // Convert bytes to GB
	if availableDisk < requiredDiskGB {
		return fmt.Errorf("insufficient disk space: need %dGB, have %dGB available", requiredDiskGB, availableDisk)
	}

	return nil
}

// Ping checks if a host is reachable
func (v *ValidatorService) Ping(host string) (bool, error) {
	// Extract just the hostname/IP (remove port if present)
	hostOnly := host
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		hostOnly = parts[0]
	}

	_, err := v.sshClient.Execute(host, fmt.Sprintf("ping -c 1 -W 2 %s", hostOnly))
	if err != nil {
		return false, fmt.Errorf("host unreachable")
	}
	return true, nil
}

// ValidateDockerCompose checks if docker-compose is available
func (v *ValidatorService) ValidateDockerCompose(host string) (bool, string, error) {
	// Try docker compose (new)
	output, err := v.sshClient.Execute(host, "docker compose version")
	if err == nil {
		return true, strings.TrimSpace(output), nil
	}

	// Try docker-compose (old)
	output, err = v.sshClient.Execute(host, "docker-compose --version")
	if err == nil {
		return true, strings.TrimSpace(output), nil
	}

	return false, "", fmt.Errorf("docker compose not found")
}

// GetSystemInfo retrieves system information from the device
func (v *ValidatorService) GetSystemInfo(host string) (map[string]string, error) {
	info := make(map[string]string)

	// OS Info
	if output, err := v.sshClient.Execute(host, "cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d '\"'"); err == nil {
		info["os"] = strings.TrimSpace(output)
	}

	// Kernel
	if output, err := v.sshClient.Execute(host, "uname -r"); err == nil {
		info["kernel"] = strings.TrimSpace(output)
	}

	// CPU
	if output, err := v.sshClient.Execute(host, "nproc"); err == nil {
		info["cpu_cores"] = strings.TrimSpace(output)
	}

	// Total Memory
	if output, err := v.sshClient.Execute(host, "free -h | awk 'NR==2 {print $2}'"); err == nil {
		info["total_memory"] = strings.TrimSpace(output)
	}

	// Total Disk
	if output, err := v.sshClient.Execute(host, "df -h / | awk 'NR==2 {print $2}'"); err == nil {
		info["total_disk"] = strings.TrimSpace(output)
	}

	// Uptime
	if output, err := v.sshClient.Execute(host, "uptime -p"); err == nil {
		info["uptime"] = strings.TrimSpace(output)
	}

	return info, nil
}

// ValidateIPAddress checks if an IP address is valid (IPv4 or IPv6)
func ValidateIPAddress(ip string) bool {
	if ip == "" {
		return false
	}

	// IPv4 validation with proper range checking
	ipv4Regex := regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	if ipv4Regex.MatchString(ip) {
		// Check each octet is 0-255
		parts := strings.Split(ip, ".")
		for _, part := range parts {
			num, err := strconv.Atoi(part)
			if err != nil || num < 0 || num > 255 {
				return false
			}
		}
		return true
	}

	// IPv6 validation (simplified - covers most common cases)
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$|^::1$|^::$`)
	return ipv6Regex.MatchString(ip)
}

// ValidateMACAddress checks if a MAC address is valid
func ValidateMACAddress(mac string) bool {
	macRegex := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)
	return macRegex.MatchString(mac)
}

// ValidateHostname checks if a hostname is valid (DNS name or IP address)
// Allows:
// - IPv4 addresses (192.168.1.1)
// - IPv6 addresses
// - Hostnames (my-server, server.local)
// - FQDNs (my-server.example.com, machine.wolf-bear.ts.net)
func ValidateHostname(hostname string) bool {
	if hostname == "" {
		return false
	}

	// First check if it's a valid IP address
	if ValidateIPAddress(hostname) {
		return true
	}

	// Validate as hostname/FQDN
	// Allow: alphanumeric, hyphens, dots, and underscores
	// Must not start/end with hyphen or dot
	// Each label must be 1-63 chars, total max 253 chars
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?)*$`)
	return len(hostname) <= 253 && hostnameRegex.MatchString(hostname)
}
