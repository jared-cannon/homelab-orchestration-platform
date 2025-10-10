package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// SoftwareService handles software installation and management
type SoftwareService struct {
	db        *gorm.DB
	sshClient *ssh.Client
	registry  *SoftwareRegistry
}

// NewSoftwareService creates a new software service
func NewSoftwareService(db *gorm.DB, sshClient *ssh.Client, registry *SoftwareRegistry) *SoftwareService {
	return &SoftwareService{
		db:        db,
		sshClient: sshClient,
		registry:  registry,
	}
}

// IsInstalled checks if software is installed on a device
func (s *SoftwareService) IsInstalled(host string, softwareName models.SoftwareType) (bool, string, error) {
	var checkCmd string

	switch softwareName {
	case models.SoftwareDocker:
		checkCmd = "docker --version"
	case models.SoftwareDockerCompose:
		checkCmd = "docker compose version"
	case models.SoftwareNFSServer:
		checkCmd = "systemctl is-active nfs-kernel-server"
	case models.SoftwareNFSClient:
		checkCmd = "dpkg -l | grep nfs-common"
	default:
		return false, "", fmt.Errorf("unknown software type: %s", softwareName)
	}

	output, err := s.sshClient.Execute(host, checkCmd)
	if err != nil {
		return false, "", nil // Not installed
	}

	version := strings.TrimSpace(output)
	return true, version, nil
}

// InstallDocker installs Docker Engine on Ubuntu 24.04
func (s *SoftwareService) InstallDocker(deviceID uuid.UUID, addUserToGroup bool) (*models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Check if already installed
	installed, version, _ := s.IsInstalled(host, models.SoftwareDocker)
	if installed {
		log.Printf("[Software] Docker already installed on %s: %s", device.Name, version)

		// Return existing record or create one
		var existing models.InstalledSoftware
		err := s.db.Where("device_id = ? AND name = ?", deviceID, models.SoftwareDocker).First(&existing).Error
		if err == nil {
			return &existing, nil
		}

		// Create record for existing installation
		software := &models.InstalledSoftware{
			DeviceID:    deviceID,
			Name:        models.SoftwareDocker,
			Version:     version,
			InstalledBy: "detected",
		}
		if err := s.db.Create(software).Error; err != nil {
			return nil, fmt.Errorf("failed to record software: %w", err)
		}
		return software, nil
	}

	log.Printf("[Software] Installing Docker on %s", device.Name)

	// Pre-flight checks
	if err := s.checkPrerequisites(host); err != nil {
		return nil, fmt.Errorf("pre-flight check failed: %w", err)
	}

	// Download installation script
	log.Printf("[Software] Downloading Docker installation script...")
	_, err = s.sshClient.Execute(host, "curl -fsSL https://get.docker.com -o /tmp/get-docker.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to download Docker script: %w", err)
	}

	// Run installation script
	log.Printf("[Software] Running Docker installation...")
	_, err = s.sshClient.Execute(host, "sudo sh /tmp/get-docker.sh")
	if err != nil {
		// Cleanup
		s.sshClient.Execute(host, "rm /tmp/get-docker.sh")
		return nil, fmt.Errorf("Docker installation failed: %w", err)
	}

	// Start and enable Docker service
	log.Printf("[Software] Starting Docker service...")
	_, err = s.sshClient.Execute(host, "sudo systemctl start docker && sudo systemctl enable docker")
	if err != nil {
		return nil, fmt.Errorf("failed to start Docker service: %w", err)
	}

	// Add user to docker group if requested
	if addUserToGroup {
		username := s.getSSHUsername(device)
		if username != "" {
			log.Printf("[Software] Adding user %s to docker group...", username)
			s.sshClient.Execute(host, fmt.Sprintf("sudo usermod -aG docker %s", username))
		}
	}

	// Cleanup
	s.sshClient.Execute(host, "rm /tmp/get-docker.sh")

	// Get version
	versionOutput, _ := s.sshClient.Execute(host, "docker --version")
	version = strings.TrimSpace(versionOutput)

	log.Printf("[Software] Docker installed successfully: %s", version)

	// Record installation
	software := &models.InstalledSoftware{
		DeviceID:    deviceID,
		Name:        models.SoftwareDocker,
		Version:     version,
		InstalledBy: "system",
	}

	if err := s.db.Create(software).Error; err != nil {
		return nil, fmt.Errorf("failed to record software: %w", err)
	}

	return software, nil
}

// InstallNFSServer installs NFS server packages
func (s *SoftwareService) InstallNFSServer(deviceID uuid.UUID) (*models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Check if already installed
	installed, _, _ := s.IsInstalled(host, models.SoftwareNFSServer)
	if installed {
		log.Printf("[Software] NFS server already installed on %s", device.Name)
		var existing models.InstalledSoftware
		err := s.db.Where("device_id = ? AND name = ?", deviceID, models.SoftwareNFSServer).First(&existing).Error
		if err == nil {
			return &existing, nil
		}
	}

	log.Printf("[Software] Installing NFS server on %s", device.Name)

	// Update package list
	_, err = s.sshClient.Execute(host, "sudo apt-get update")
	if err != nil {
		return nil, fmt.Errorf("failed to update package list: %w", err)
	}

	// Install nfs-kernel-server
	_, err = s.sshClient.Execute(host, "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nfs-kernel-server")
	if err != nil {
		return nil, fmt.Errorf("failed to install nfs-kernel-server: %w", err)
	}

	// Start and enable service
	_, err = s.sshClient.Execute(host, "sudo systemctl start nfs-kernel-server && sudo systemctl enable nfs-kernel-server")
	if err != nil {
		return nil, fmt.Errorf("failed to start NFS service: %w", err)
	}

	// Get version
	versionOutput, _ := s.sshClient.Execute(host, "dpkg -l | grep nfs-kernel-server | awk '{print $3}'")
	version := strings.TrimSpace(versionOutput)

	log.Printf("[Software] NFS server installed successfully: %s", version)

	// Record installation
	software := &models.InstalledSoftware{
		DeviceID:    deviceID,
		Name:        models.SoftwareNFSServer,
		Version:     version,
		InstalledBy: "system",
	}

	if err := s.db.Create(software).Error; err != nil {
		return nil, fmt.Errorf("failed to record software: %w", err)
	}

	return software, nil
}

// InstallNFSClient installs NFS client packages
func (s *SoftwareService) InstallNFSClient(deviceID uuid.UUID) (*models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Check if already installed
	installed, _, _ := s.IsInstalled(host, models.SoftwareNFSClient)
	if installed {
		log.Printf("[Software] NFS client already installed on %s", device.Name)
		var existing models.InstalledSoftware
		err := s.db.Where("device_id = ? AND name = ?", deviceID, models.SoftwareNFSClient).First(&existing).Error
		if err == nil {
			return &existing, nil
		}
	}

	log.Printf("[Software] Installing NFS client on %s", device.Name)

	// Update package list
	_, err = s.sshClient.Execute(host, "sudo apt-get update")
	if err != nil {
		return nil, fmt.Errorf("failed to update package list: %w", err)
	}

	// Install nfs-common
	_, err = s.sshClient.Execute(host, "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nfs-common")
	if err != nil {
		return nil, fmt.Errorf("failed to install nfs-common: %w", err)
	}

	// Get version
	versionOutput, _ := s.sshClient.Execute(host, "dpkg -l | grep nfs-common | awk '{print $3}'")
	version := strings.TrimSpace(versionOutput)

	log.Printf("[Software] NFS client installed successfully: %s", version)

	// Record installation
	software := &models.InstalledSoftware{
		DeviceID:    deviceID,
		Name:        models.SoftwareNFSClient,
		Version:     version,
		InstalledBy: "system",
	}

	if err := s.db.Create(software).Error; err != nil {
		return nil, fmt.Errorf("failed to record software: %w", err)
	}

	return software, nil
}

// ListInstalled lists all installed software on a device
func (s *SoftwareService) ListInstalled(deviceID uuid.UUID) ([]models.InstalledSoftware, error) {
	var software []models.InstalledSoftware
	err := s.db.Where("device_id = ?", deviceID).Find(&software).Error
	return software, err
}

// DetectInstalled scans a device for installed software and syncs with database
func (s *SoftwareService) DetectInstalled(deviceID uuid.UUID) ([]models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Software] Detecting installed software on %s", device.Name)

	// Check all known software types
	softwareTypes := []models.SoftwareType{
		models.SoftwareDocker,
		models.SoftwareNFSServer,
		models.SoftwareNFSClient,
	}

	var detected []models.InstalledSoftware

	for _, softwareType := range softwareTypes {
		installed, version, _ := s.IsInstalled(host, softwareType)
		if installed {
			log.Printf("[Software] Detected %s on %s: %s", softwareType, device.Name, version)

			// Check if already in database
			var existing models.InstalledSoftware
			err := s.db.Where("device_id = ? AND name = ?", deviceID, softwareType).First(&existing).Error
			if err == nil {
				// Already exists, just add to results
				detected = append(detected, existing)
				continue
			}

			// Create new record
			software := &models.InstalledSoftware{
				DeviceID:    deviceID,
				Name:        softwareType,
				Version:     version,
				InstalledBy: "detected",
			}

			if err := s.db.Create(software).Error; err != nil {
				log.Printf("[Software] Warning: failed to record %s: %v", softwareType, err)
				continue
			}

			detected = append(detected, *software)
		}
	}

	log.Printf("[Software] Detection complete on %s: found %d installed packages", device.Name, len(detected))

	return detected, nil
}

// CheckUpdates checks for available updates for installed software
func (s *SoftwareService) CheckUpdates(deviceID uuid.UUID) ([]models.SoftwareUpdateInfo, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Get installed software
	var installedSoftware []models.InstalledSoftware
	if err := s.db.Where("device_id = ?", deviceID).Find(&installedSoftware).Error; err != nil {
		return nil, fmt.Errorf("failed to get installed software: %w", err)
	}

	log.Printf("[Software] Checking updates for %d packages on %s", len(installedSoftware), device.Name)

	var updateInfo []models.SoftwareUpdateInfo

	for _, software := range installedSoftware {
		// Get software definition
		def, err := s.registry.GetDefinition(string(software.Name))
		if err != nil {
			log.Printf("[Software] Warning: no definition for %s, skipping update check", software.Name)
			continue
		}

		// Check for updates using the definition's check_updates command
		if def.Commands.CheckUpdates == "" {
			log.Printf("[Software] No update check command for %s, skipping", software.Name)
			continue
		}

		output, err := s.sshClient.Execute(host, def.Commands.CheckUpdates)
		updateAvailable := err == nil && strings.TrimSpace(output) != ""

		info := models.SoftwareUpdateInfo{
			SoftwareID:      string(software.Name),
			CurrentVersion:  software.Version,
			UpdateAvailable: updateAvailable,
		}

		if updateAvailable {
			info.Message = strings.TrimSpace(output)
			log.Printf("[Software] Update available for %s on %s", software.Name, device.Name)
		}

		updateInfo = append(updateInfo, info)
	}

	log.Printf("[Software] Update check complete on %s: %d packages checked", device.Name, len(updateInfo))
	return updateInfo, nil
}

// UpdateSoftware updates a specific software package to the latest version
func (s *SoftwareService) UpdateSoftware(deviceID uuid.UUID, softwareName models.SoftwareType) (*models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Get software definition
	def, err := s.registry.GetDefinition(string(softwareName))
	if err != nil {
		return nil, fmt.Errorf("software definition not found: %w", err)
	}

	// Check if software is installed
	var existing models.InstalledSoftware
	if err := s.db.Where("device_id = ? AND name = ?", deviceID, softwareName).First(&existing).Error; err != nil {
		return nil, fmt.Errorf("software not installed")
	}

	log.Printf("[Software] Updating %s on %s", softwareName, device.Name)

	// Run update command
	if def.Commands.Update == "" {
		return nil, fmt.Errorf("no update command defined for %s", softwareName)
	}

	_, err = s.sshClient.Execute(host, def.Commands.Update)
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	// Get new version
	newVersion := ""
	if def.Commands.CheckVersion != "" {
		versionOutput, err := s.sshClient.Execute(host, def.Commands.CheckVersion)
		if err == nil {
			newVersion = strings.TrimSpace(versionOutput)
		}
	}

	// Update database record
	if newVersion != "" && newVersion != existing.Version {
		existing.Version = newVersion
		s.db.Save(&existing)
		log.Printf("[Software] Updated %s from %s to %s", softwareName, existing.Version, newVersion)
	}

	log.Printf("[Software] Update complete for %s on %s", softwareName, device.Name)
	return &existing, nil
}

// Install installs software using the plugin system (registry-based)
func (s *SoftwareService) Install(deviceID uuid.UUID, softwareName models.SoftwareType, options map[string]interface{}) (*models.InstalledSoftware, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Get software definition from registry
	def, err := s.registry.GetDefinition(string(softwareName))
	if err != nil {
		return nil, fmt.Errorf("software definition not found: %w", err)
	}

	// Check if already installed
	if def.Commands.CheckInstalled != "" {
		output, err := s.sshClient.Execute(host, def.Commands.CheckInstalled)
		if err == nil && strings.TrimSpace(output) != "" {
			log.Printf("[Software] %s already installed on %s", def.Name, device.Name)

			// Return existing record or create one
			var existing models.InstalledSoftware
			err := s.db.Where("device_id = ? AND name = ?", deviceID, softwareName).First(&existing).Error
			if err == nil {
				return &existing, nil
			}

			// Get version
			version := strings.TrimSpace(output)
			if def.Commands.CheckVersion != "" {
				versionOutput, err := s.sshClient.Execute(host, def.Commands.CheckVersion)
				if err == nil {
					version = strings.TrimSpace(versionOutput)
				}
			}

			// Create record for existing installation
			software := &models.InstalledSoftware{
				DeviceID:    deviceID,
				Name:        softwareName,
				Version:     version,
				InstalledBy: "detected",
			}
			if err := s.db.Create(software).Error; err != nil {
				return nil, fmt.Errorf("failed to record software: %w", err)
			}
			return software, nil
		}
	}

	log.Printf("[Software] Installing %s on %s using plugin system", def.Name, device.Name)

	// Check for passwordless sudo if install command contains sudo
	if strings.Contains(def.Commands.Install, "sudo") {
		log.Printf("[Software] Checking passwordless sudo access...")
		_, testErr := s.sshClient.Execute(host, "sudo -n true")
		if testErr != nil {
			log.Printf("[Software] Passwordless sudo check failed: %v", testErr)
			return nil, models.NewSudoError(device.IPAddress)
		}
		log.Printf("[Software] Passwordless sudo confirmed")
	}

	// Run install command
	if def.Commands.Install == "" {
		return nil, fmt.Errorf("no install command defined for %s", softwareName)
	}

	log.Printf("[Software] Running install command: %s", def.Commands.Install)
	output, err := s.sshClient.Execute(host, def.Commands.Install)
	if err != nil {
		log.Printf("[Software] Install command failed. Output: %s", output)
		log.Printf("[Software] Install command error: %v", err)
		return nil, fmt.Errorf("installation failed: %w (output: %s)", err, output)
	}
	log.Printf("[Software] Install command succeeded. Output: %s", output)

	// Run post-install hook if defined
	if def.Commands.PostInstall != "" {
		log.Printf("[Software] Running post-install hook for %s", def.Name)
		_, err = s.sshClient.Execute(host, def.Commands.PostInstall)
		if err != nil {
			log.Printf("[Software] Warning: post-install hook failed: %v", err)
			// Don't fail the installation if post-install fails
		}
	}

	// Get version
	version := ""
	if def.Commands.CheckVersion != "" {
		versionOutput, err := s.sshClient.Execute(host, def.Commands.CheckVersion)
		if err == nil {
			version = strings.TrimSpace(versionOutput)
		}
	}

	log.Printf("[Software] %s installed successfully: %s", def.Name, version)

	// Record installation
	software := &models.InstalledSoftware{
		DeviceID:    deviceID,
		Name:        softwareName,
		Version:     version,
		InstalledBy: "system",
	}

	if err := s.db.Create(software).Error; err != nil {
		return nil, fmt.Errorf("failed to record software: %w", err)
	}

	return software, nil
}

// ListAvailableSoftware returns all software definitions from the registry
func (s *SoftwareService) ListAvailableSoftware() []*models.SoftwareDefinition {
	return s.registry.ListDefinitions()
}

// Uninstall removes software from a device
func (s *SoftwareService) Uninstall(deviceID uuid.UUID, softwareName models.SoftwareType) error {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Software] Uninstalling %s from %s", softwareName, device.Name)

	var uninstallCmd string
	switch softwareName {
	case models.SoftwareDocker:
		// Stop service
		s.sshClient.Execute(host, "sudo systemctl stop docker")
		// Remove packages
		uninstallCmd = "sudo apt-get remove -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"
	case models.SoftwareNFSServer:
		s.sshClient.Execute(host, "sudo systemctl stop nfs-kernel-server")
		uninstallCmd = "sudo apt-get remove -y nfs-kernel-server"
	case models.SoftwareNFSClient:
		uninstallCmd = "sudo apt-get remove -y nfs-common"
	default:
		return fmt.Errorf("unknown software type: %s", softwareName)
	}

	_, err = s.sshClient.Execute(host, uninstallCmd)
	if err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}

	// Remove from database
	err = s.db.Where("device_id = ? AND name = ?", deviceID, softwareName).Delete(&models.InstalledSoftware{}).Error
	if err != nil {
		return fmt.Errorf("failed to remove database record: %w", err)
	}

	log.Printf("[Software] Successfully uninstalled %s from %s", softwareName, device.Name)
	return nil
}

// checkPrerequisites verifies system requirements before installation
func (s *SoftwareService) checkPrerequisites(host string) error {
	// Check sudo access
	_, err := s.sshClient.Execute(host, "sudo -n true")
	if err != nil {
		return fmt.Errorf("sudo access required - ensure SSH user has passwordless sudo")
	}

	// Check internet connectivity
	_, err = s.sshClient.Execute(host, "curl -I https://get.docker.com --connect-timeout 5")
	if err != nil {
		return fmt.Errorf("no internet connectivity - cannot reach Docker servers")
	}

	// Check disk space (need at least 10GB)
	output, err := s.sshClient.Execute(host, "df / | awk 'NR==2 {print $4}'")
	if err == nil {
		// output is in KB
		availableKB := strings.TrimSpace(output)
		// Just log warning, don't fail
		log.Printf("[Software] Available disk space: %s KB", availableKB)
	}

	return nil
}

// getDevice retrieves device by ID
func (s *SoftwareService) getDevice(deviceID uuid.UUID) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, "id = ?", deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("device not found")
		}
		return nil, err
	}
	return &device, nil
}

// getSSHUsername extracts username from device credentials (simplified)
func (s *SoftwareService) getSSHUsername(device *models.Device) string {
	// This is a simplified version - in production, retrieve from credentials service
	// For now, return empty to skip user group addition
	return ""
}
