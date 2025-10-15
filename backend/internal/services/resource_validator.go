package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
)

// ResourceValidator validates system resources on devices
type ResourceValidator struct {
	sshClient *ssh.Client
}

// NewResourceValidator creates a new resource validator
func NewResourceValidator(sshClient *ssh.Client) *ResourceValidator {
	return &ResourceValidator{
		sshClient: sshClient,
	}
}

// ResourceStatus contains current resource availability on a device
type ResourceStatus struct {
	TotalRAMMB      int   `json:"total_ram_mb"`
	AvailableRAMMB  int   `json:"available_ram_mb"`
	UsedRAMMB       int   `json:"used_ram_mb"`
	TotalStorageGB  int   `json:"total_storage_gb"`
	AvailableStorageGB int `json:"available_storage_gb"`
	UsedStorageGB   int   `json:"used_storage_gb"`
	CPUCores        int   `json:"cpu_cores"`
	UsedPorts       []int `json:"used_ports"`
}

// CheckResources checks available resources on a device
func (rv *ResourceValidator) CheckResources(device *models.Device) (*ResourceStatus, error) {
	host := fmt.Sprintf("%s:22", device.IPAddress)

	resources := &ResourceStatus{
		UsedPorts: []int{},
	}

	// Check RAM using free command
	ramOutput, err := rv.sshClient.ExecuteWithTimeout(host, "free -m | grep Mem:", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to check RAM: %w", err)
	}

	// Parse RAM output: Mem: total used free shared buff/cache available
	ramFields := strings.Fields(ramOutput)
	if len(ramFields) >= 7 {
		if total, err := strconv.Atoi(ramFields[1]); err == nil {
			resources.TotalRAMMB = total
		}
		if used, err := strconv.Atoi(ramFields[2]); err == nil {
			resources.UsedRAMMB = used
		}
		if available, err := strconv.Atoi(ramFields[6]); err == nil {
			resources.AvailableRAMMB = available
		}
	} else {
		return nil, fmt.Errorf("unexpected RAM output format: %s", ramOutput)
	}

	// Check storage using df command (root filesystem)
	storageOutput, err := rv.sshClient.ExecuteWithTimeout(host, "df -BG / | tail -n 1", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to check storage: %w", err)
	}

	// Parse storage output: Filesystem 1G-blocks Used Available Use% Mounted on
	storageFields := strings.Fields(storageOutput)
	if len(storageFields) >= 4 {
		// Remove 'G' suffix and convert
		if total, err := strconv.Atoi(strings.TrimSuffix(storageFields[1], "G")); err == nil {
			resources.TotalStorageGB = total
		}
		if used, err := strconv.Atoi(strings.TrimSuffix(storageFields[2], "G")); err == nil {
			resources.UsedStorageGB = used
		}
		if available, err := strconv.Atoi(strings.TrimSuffix(storageFields[3], "G")); err == nil {
			resources.AvailableStorageGB = available
		}
	} else {
		return nil, fmt.Errorf("unexpected storage output format: %s", storageOutput)
	}

	// Check CPU cores
	cpuOutput, err := rv.sshClient.ExecuteWithTimeout(host, "nproc", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to check CPU cores: %w", err)
	}
	if cores, err := strconv.Atoi(strings.TrimSpace(cpuOutput)); err == nil {
		resources.CPUCores = cores
	}

	// Check used ports (TCP listening ports)
	portsOutput, err := rv.sshClient.ExecuteWithTimeout(host, "ss -tuln | grep LISTEN | awk '{print $5}' | sed 's/.*://' | sort -u", 10*time.Second)
	if err == nil {
		portLines := strings.Split(strings.TrimSpace(portsOutput), "\n")
		for _, portStr := range portLines {
			portStr = strings.TrimSpace(portStr)
			if portStr == "" {
				continue
			}
			if port, err := strconv.Atoi(portStr); err == nil {
				resources.UsedPorts = append(resources.UsedPorts, port)
			}
		}
	}

	return resources, nil
}

// ValidateResourceRequirements checks if device has sufficient resources
func (rv *ResourceValidator) ValidateResourceRequirements(
	device *models.Device,
	requiredRAMMB int,
	requiredStorageGB int,
	requiredCPUCores int,
	requiredPorts []int,
) (*ResourceValidationResult, error) {

	// Get current device resources
	resources, err := rv.CheckResources(device)
	if err != nil {
		return nil, fmt.Errorf("failed to check device resources: %w", err)
	}

	result := &ResourceValidationResult{
		DeviceResources: resources,
		RAMSufficient:   resources.AvailableRAMMB >= requiredRAMMB,
		StorageSufficient: resources.AvailableStorageGB >= requiredStorageGB,
		CPUSufficient:   resources.CPUCores >= requiredCPUCores,
		PortsAvailable:  true,
		PortConflicts:   []int{},
	}

	// Check for port conflicts
	usedPortsMap := make(map[int]bool)
	for _, port := range resources.UsedPorts {
		usedPortsMap[port] = true
	}

	for _, requiredPort := range requiredPorts {
		if usedPortsMap[requiredPort] {
			result.PortsAvailable = false
			result.PortConflicts = append(result.PortConflicts, requiredPort)
		}
	}

	result.Valid = result.RAMSufficient && result.StorageSufficient && result.CPUSufficient && result.PortsAvailable

	return result, nil
}

// ResourceValidationResult contains the result of resource validation
type ResourceValidationResult struct {
	Valid             bool              `json:"valid"`
	DeviceResources   *ResourceStatus   `json:"device_resources"`
	RAMSufficient     bool              `json:"ram_sufficient"`
	StorageSufficient bool              `json:"storage_sufficient"`
	CPUSufficient     bool              `json:"cpu_sufficient"`
	PortsAvailable    bool              `json:"ports_available"`
	PortConflicts     []int             `json:"port_conflicts"`
}

// ExtractPortsFromConfig extracts port numbers from deployment config
func ExtractPortsFromConfig(config map[string]interface{}) []int {
	ports := []int{}

	// Common port field names
	portFields := []string{"port", "internal_port", "external_port", "http_port", "https_port", "dashboard_port"}

	for _, field := range portFields {
		if portVal, ok := config[field]; ok {
			switch v := portVal.(type) {
			case int:
				ports = append(ports, v)
			case float64:
				ports = append(ports, int(v))
			case string:
				if port, err := strconv.Atoi(v); err == nil {
					ports = append(ports, port)
				}
			}
		}
	}

	return ports
}
