package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
)

// FirewallService manages firewall rules on remote devices
type FirewallService struct {
	sshClient *ssh.Client
}

// NewFirewallService creates a new firewall service
func NewFirewallService(sshClient *ssh.Client) *FirewallService {
	return &FirewallService{
		sshClient: sshClient,
	}
}

// FirewallStatus represents the status of the firewall
type FirewallStatus struct {
	Installed bool     `json:"installed"`
	Enabled   bool     `json:"enabled"`
	OpenPorts []int    `json:"open_ports"`
	Type      string   `json:"type"` // "ufw", "firewalld", "iptables", "none"
}

// PortSpec represents a port with its protocol
type PortSpec struct {
	Port     int
	Protocol string // "tcp" or "udp"
}

// CheckFirewall checks the firewall status on a device
func (f *FirewallService) CheckFirewall(device *models.Device) (*FirewallStatus, error) {
	host := device.GetSSHHost()

	status := &FirewallStatus{
		OpenPorts: []int{},
	}

	// Check for UFW (Ubuntu/Debian)
	ufwCheck, err := f.sshClient.ExecuteWithTimeout(host, "which ufw", 5*time.Second)
	if err == nil && strings.TrimSpace(ufwCheck) != "" {
		status.Installed = true
		status.Type = "ufw"

		// Check if UFW is enabled
		ufwStatus, err := f.sshClient.ExecuteWithTimeout(host, "sudo ufw status", 5*time.Second)
		if err == nil {
			status.Enabled = strings.Contains(ufwStatus, "Status: active")

			// Parse open ports from UFW status
			if status.Enabled {
				ports := f.parseUFWPorts(ufwStatus)
				status.OpenPorts = ports
			}
		}

		return status, nil
	}

	// Check for firewalld (RHEL/CentOS/Fedora)
	firewalldCheck, err := f.sshClient.ExecuteWithTimeout(host, "which firewall-cmd", 5*time.Second)
	if err == nil && strings.TrimSpace(firewalldCheck) != "" {
		status.Installed = true
		status.Type = "firewalld"

		// Check if firewalld is running
		firewalldStatus, err := f.sshClient.ExecuteWithTimeout(host, "sudo firewall-cmd --state", 5*time.Second)
		if err == nil {
			status.Enabled = strings.Contains(firewalldStatus, "running")

			// Parse open ports from firewalld status
			if status.Enabled {
				ports := f.parseFirewalldPorts(host)
				status.OpenPorts = ports
			}
		}

		return status, nil
	}

	// No recognized firewall
	status.Type = "none"
	return status, nil
}

// parseUFWPorts extracts port numbers from UFW status output
func (f *FirewallService) parseUFWPorts(ufwStatus string) []int {
	ports := []int{}
	seenPorts := make(map[int]bool)
	portRegex := regexp.MustCompile(`(\d+)(?:/tcp|/udp)?\s+ALLOW`)

	matches := portRegex.FindAllStringSubmatch(ufwStatus, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if port, err := strconv.Atoi(match[1]); err == nil {
				if !seenPorts[port] {
					ports = append(ports, port)
					seenPorts[port] = true
				}
			}
		}
	}

	return ports
}

// parseFirewalldPorts extracts port numbers from firewalld
func (f *FirewallService) parseFirewalldPorts(host string) []int {
	ports := []int{}
	seenPorts := make(map[int]bool)

	// Get list of open ports from firewalld
	output, err := f.sshClient.ExecuteWithTimeout(host, "sudo firewall-cmd --list-ports", 5*time.Second)
	if err != nil {
		return ports
	}

	// Parse output like: "80/tcp 443/tcp 8080/tcp"
	portRegex := regexp.MustCompile(`(\d+)/(tcp|udp)`)
	matches := portRegex.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) > 1 {
			if port, err := strconv.Atoi(match[1]); err == nil {
				if !seenPorts[port] {
					ports = append(ports, port)
					seenPorts[port] = true
				}
			}
		}
	}

	return ports
}

// OpenPorts opens specified ports on the firewall (idempotent - skips already-open ports)
func (f *FirewallService) OpenPorts(device *models.Device, portSpecs []PortSpec) error {
	if len(portSpecs) == 0 {
		return nil
	}

	host := device.GetSSHHost()

	// Check firewall status
	status, err := f.CheckFirewall(device)
	if err != nil {
		return fmt.Errorf("failed to check firewall on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}

	// If no firewall or not enabled, no action needed
	if !status.Installed || !status.Enabled {
		return nil
	}

	// Filter out ports that are already open (idempotent operation)
	openPortMap := make(map[int]bool)
	for _, port := range status.OpenPorts {
		openPortMap[port] = true
	}

	portsToOpen := []PortSpec{}
	for _, spec := range portSpecs {
		if !openPortMap[spec.Port] {
			portsToOpen = append(portsToOpen, spec)
		}
	}

	// If all ports already open, nothing to do
	if len(portsToOpen) == 0 {
		return nil
	}

	// Open ports based on firewall type
	switch status.Type {
	case "ufw":
		return f.openPortsUFW(host, device, portsToOpen)
	case "firewalld":
		return f.openPortsFirewalld(host, device, portsToOpen)
	default:
		return fmt.Errorf("unsupported firewall type: %s", status.Type)
	}
}

// openPortsUFW opens ports using UFW
func (f *FirewallService) openPortsUFW(host string, device *models.Device, portSpecs []PortSpec) error {
	for _, spec := range portSpecs {
		// UFW command: sudo ufw allow 8080/tcp
		cmd := fmt.Sprintf("sudo ufw allow %d/%s", spec.Port, spec.Protocol)
		_, err := f.sshClient.ExecuteWithTimeout(host, cmd, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to open port %d/%s on %s (%s): %w",
				spec.Port, spec.Protocol, device.Name, device.GetPrimaryAddress(), err)
		}
	}

	// Reload UFW to apply changes
	_, err := f.sshClient.ExecuteWithTimeout(host, "sudo ufw reload", 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to reload UFW on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}
	return nil
}

// openPortsFirewalld opens ports using firewalld
func (f *FirewallService) openPortsFirewalld(host string, device *models.Device, portSpecs []PortSpec) error {
	for _, spec := range portSpecs {
		// firewalld command: sudo firewall-cmd --permanent --add-port=8080/tcp
		cmd := fmt.Sprintf("sudo firewall-cmd --permanent --add-port=%d/%s", spec.Port, spec.Protocol)
		_, err := f.sshClient.ExecuteWithTimeout(host, cmd, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to open port %d/%s on %s (%s): %w",
				spec.Port, spec.Protocol, device.Name, device.GetPrimaryAddress(), err)
		}
	}

	// Reload firewalld to apply changes
	_, err := f.sshClient.ExecuteWithTimeout(host, "sudo firewall-cmd --reload", 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to reload firewalld on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}
	return nil
}

// ExtractPortsFromCompose extracts all exposed ports from a Docker Compose file with protocols
// Supports formats: "8080:80", 8080:80, "127.0.0.1:8080:80", "443:443/tcp", 8080:80/udp
func ExtractPortsFromCompose(composeContent string) []PortSpec {
	portSpecs := []PortSpec{}
	seenPorts := make(map[string]bool) // key: "port/protocol"

	// Comprehensive regex to match all Docker Compose port formats:
	// - "8080:80" or 8080:80 (quoted or unquoted)
	// - "127.0.0.1:8080:80" (with IP binding)
	// - "8080:80/tcp" or 8080:80/udp (with protocol)
	// Captures: host_port and optional protocol
	portRegex := regexp.MustCompile(`(?:"|')?(?:\d+\.\d+\.\d+\.\d+:)?(\d+):\d+(?:/(tcp|udp))?(?:"|')?`)

	matches := portRegex.FindAllStringSubmatch(composeContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			port, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}

			// Default to TCP if no protocol specified
			protocol := "tcp"
			if len(match) > 2 && match[2] != "" {
				protocol = match[2]
			}

			// Deduplicate by "port/protocol" combination
			key := fmt.Sprintf("%d/%s", port, protocol)
			if !seenPorts[key] {
				portSpecs = append(portSpecs, PortSpec{
					Port:     port,
					Protocol: protocol,
				})
				seenPorts[key] = true
			}
		}
	}

	return portSpecs
}

// GetFirewallTroubleshootingSteps returns troubleshooting steps for firewall issues
func (f *FirewallService) GetFirewallTroubleshootingSteps(device *models.Device, portSpecs []PortSpec) (string, error) {
	status, err := f.CheckFirewall(device)
	if err != nil {
		return "", err
	}

	var steps strings.Builder
	steps.WriteString("## Firewall Troubleshooting\n\n")

	if !status.Installed {
		steps.WriteString("ℹ️  No firewall detected on this device.\n")
		steps.WriteString("Ports should be accessible without additional configuration.\n\n")
		return steps.String(), nil
	}

	if !status.Enabled {
		steps.WriteString(fmt.Sprintf("ℹ️  %s is installed but not enabled.\n", status.Type))
		steps.WriteString("Ports should be accessible without additional configuration.\n\n")
		return steps.String(), nil
	}

	// Build map of open ports for O(1) lookup
	openPortMap := make(map[int]bool)
	for _, port := range status.OpenPorts {
		openPortMap[port] = true
	}

	steps.WriteString(fmt.Sprintf("🔥 Firewall: %s (active)\n\n", status.Type))
	steps.WriteString("**Required Ports:**\n")

	for _, spec := range portSpecs {
		if openPortMap[spec.Port] {
			steps.WriteString(fmt.Sprintf("- ✅ Port %d/%s: OPEN\n", spec.Port, spec.Protocol))
		} else {
			steps.WriteString(fmt.Sprintf("- ❌ Port %d/%s: BLOCKED\n", spec.Port, spec.Protocol))
		}
	}

	steps.WriteString("\n**To manually open ports:**\n")
	if status.Type == "ufw" {
		steps.WriteString("```bash\n")
		for _, spec := range portSpecs {
			steps.WriteString(fmt.Sprintf("sudo ufw allow %d/%s\n", spec.Port, spec.Protocol))
		}
		steps.WriteString("sudo ufw reload\n")
		steps.WriteString("```\n")
	} else if status.Type == "firewalld" {
		steps.WriteString("```bash\n")
		for _, spec := range portSpecs {
			steps.WriteString(fmt.Sprintf("sudo firewall-cmd --permanent --add-port=%d/%s\n", spec.Port, spec.Protocol))
		}
		steps.WriteString("sudo firewall-cmd --reload\n")
		steps.WriteString("```\n")
	}

	return steps.String(), nil
}

// ClosePorts closes specified ports on the firewall
func (f *FirewallService) ClosePorts(device *models.Device, portSpecs []PortSpec) error {
	if len(portSpecs) == 0 {
		return nil
	}

	host := device.GetSSHHost()

	// Check firewall status
	status, err := f.CheckFirewall(device)
	if err != nil {
		return fmt.Errorf("failed to check firewall on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}

	// If no firewall or not enabled, no action needed
	if !status.Installed || !status.Enabled {
		return nil
	}

	// Close ports based on firewall type
	switch status.Type {
	case "ufw":
		return f.closePortsUFW(host, device, portSpecs)
	case "firewalld":
		return f.closePortsFirewalld(host, device, portSpecs)
	default:
		return fmt.Errorf("unsupported firewall type: %s", status.Type)
	}
}

// closePortsUFW closes ports using UFW
func (f *FirewallService) closePortsUFW(host string, device *models.Device, portSpecs []PortSpec) error {
	for _, spec := range portSpecs {
		// UFW command: sudo ufw delete allow 8080/tcp
		cmd := fmt.Sprintf("sudo ufw delete allow %d/%s", spec.Port, spec.Protocol)
		_, err := f.sshClient.ExecuteWithTimeout(host, cmd, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to close port %d/%s on %s (%s): %w",
				spec.Port, spec.Protocol, device.Name, device.GetPrimaryAddress(), err)
		}
	}

	// Reload UFW to apply changes
	_, err := f.sshClient.ExecuteWithTimeout(host, "sudo ufw reload", 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to reload UFW on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}
	return nil
}

// closePortsFirewalld closes ports using firewalld
func (f *FirewallService) closePortsFirewalld(host string, device *models.Device, portSpecs []PortSpec) error {
	for _, spec := range portSpecs {
		// firewalld command: sudo firewall-cmd --permanent --remove-port=8080/tcp
		cmd := fmt.Sprintf("sudo firewall-cmd --permanent --remove-port=%d/%s", spec.Port, spec.Protocol)
		_, err := f.sshClient.ExecuteWithTimeout(host, cmd, 10*time.Second)
		if err != nil {
			return fmt.Errorf("failed to close port %d/%s on %s (%s): %w",
				spec.Port, spec.Protocol, device.Name, device.GetPrimaryAddress(), err)
		}
	}

	// Reload firewalld to apply changes
	_, err := f.sshClient.ExecuteWithTimeout(host, "sudo firewall-cmd --reload", 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to reload firewalld on %s (%s): %w", device.Name, device.GetPrimaryAddress(), err)
	}
	return nil
}
