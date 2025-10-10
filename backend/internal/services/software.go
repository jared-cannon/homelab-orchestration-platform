package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// sshExecutor is an interface for SSH command execution (allows mocking in tests)
type sshExecutor interface {
	Execute(host, command string) (string, error)
	ExecuteWithTimeout(host, command string, timeout time.Duration) (string, error)
}

// SoftwareService handles software installation and management
type SoftwareService struct {
	db        *gorm.DB
	sshClient sshExecutor // Use interface instead of concrete type
	registry  *SoftwareRegistry
	wsHub     WSHub
}

// NewSoftwareService creates a new software service
func NewSoftwareService(db *gorm.DB, sshClient *ssh.Client, registry *SoftwareRegistry, wsHub WSHub) *SoftwareService {
	return &SoftwareService{
		db:        db,
		sshClient: sshClient, // *ssh.Client implements sshExecutor
		registry:  registry,
		wsHub:     wsHub,
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
	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	software := make([]models.InstalledSoftware, 0)
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

	// First, get all currently recorded installed software from database
	var currentlyRecorded []models.InstalledSoftware
	if err := s.db.Where("device_id = ?", deviceID).Find(&currentlyRecorded).Error; err != nil {
		log.Printf("[Software] Warning: failed to query existing records: %v", err)
	}

	// Check each recorded software to see if it's still installed
	// Remove records for software that is no longer installed
	for _, recorded := range currentlyRecorded {
		installed, _, _ := s.IsInstalled(host, recorded.Name)
		if !installed {
			log.Printf("[Software] %s is no longer installed on %s, removing database record", recorded.Name, device.Name)
			if err := s.db.Delete(&recorded).Error; err != nil {
				log.Printf("[Software] Warning: failed to remove record for %s: %v", recorded.Name, err)
			}
		}
	}

	// Check all known software types
	softwareTypes := []models.SoftwareType{
		models.SoftwareDocker,
		models.SoftwareNFSServer,
		models.SoftwareNFSClient,
	}

	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	detected := make([]models.InstalledSoftware, 0)

	for _, softwareType := range softwareTypes {
		installed, version, _ := s.IsInstalled(host, softwareType)
		if installed {
			log.Printf("[Software] Detected %s on %s: %s", softwareType, device.Name, version)

			// Check if already in database
			var existing models.InstalledSoftware
			err := s.db.Where("device_id = ? AND name = ?", deviceID, softwareType).First(&existing).Error
			if err == nil {
				// Already exists, update version if changed
				if existing.Version != version {
					existing.Version = version
					s.db.Save(&existing)
					log.Printf("[Software] Updated version for %s: %s", softwareType, version)
				}
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

	// Initialize as empty slice (not nil) to ensure JSON serializes as [] not null
	updateInfo := make([]models.SoftwareUpdateInfo, 0)

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

	// Check for passwordless sudo if update command contains sudo
	if strings.Contains(def.Commands.Update, "sudo") {
		_, testErr := s.sshClient.Execute(host, "sudo -n true")
		if testErr != nil {
			log.Printf("[Software] Passwordless sudo check failed for update on %s", device.Name)
			return nil, models.NewSudoError(device.IPAddress)
		}
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

// Install installs software using the plugin system (registry-based) - async version
func (s *SoftwareService) Install(deviceID uuid.UUID, softwareName models.SoftwareType, options map[string]interface{}) (*models.SoftwareInstallation, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	// Get software definition from registry
	def, err := s.registry.GetDefinition(string(softwareName))
	if err != nil {
		return nil, fmt.Errorf("software definition not found: %w", err)
	}

	// Create installation record
	installation := &models.SoftwareInstallation{
		DeviceID:     deviceID,
		SoftwareName: softwareName,
		Status:       models.InstallationStatusPending,
	}

	if err := s.db.Create(installation).Error; err != nil {
		return nil, fmt.Errorf("failed to create installation record: %w", err)
	}

	// Start installation in background
	go s.executeInstallation(installation, device, def, options)

	return installation, nil
}

// executeInstallation performs the actual installation with logging
func (s *SoftwareService) executeInstallation(installation *models.SoftwareInstallation, device *models.Device, def *models.SoftwareDefinition, options map[string]interface{}) {
	// Panic recovery to prevent crashing the entire application
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Installation panic: %v", r)
			log.Printf("[Software] PANIC during installation: %v", r)
			s.appendInstallLog(installation, fmt.Sprintf("❌ Critical error: %v", r))
			s.updateInstallStatus(installation, models.InstallationStatusFailed, errorMsg)
		}
	}()

	host := device.IPAddress + ":22"
	softwareName := installation.SoftwareName

	s.appendInstallLog(installation, fmt.Sprintf("▶ Starting installation of %s on device %s (%s)", def.Name, device.Name, device.IPAddress))
	s.updateInstallStatus(installation, models.InstallationStatusInstalling, "")

	// Check if already installed
	if def.Commands.CheckInstalled != "" {
		s.appendInstallLog(installation, "▶ Checking if already installed...")
		output, err := s.sshClient.Execute(host, def.Commands.CheckInstalled)
		if err == nil && strings.TrimSpace(output) != "" {
			s.appendInstallLog(installation, fmt.Sprintf("⚠️  %s is already installed", def.Name))

			// Get version
			version := strings.TrimSpace(output)
			if def.Commands.CheckVersion != "" {
				versionOutput, err := s.sshClient.Execute(host, def.Commands.CheckVersion)
				if err == nil {
					version = strings.TrimSpace(versionOutput)
				}
			}

			// Create InstalledSoftware record (check for errors to ensure data consistency)
			software := &models.InstalledSoftware{
				DeviceID:    device.ID,
				Name:        softwareName,
				Version:     version,
				InstalledBy: "detected",
			}

			if err := s.db.Create(software).Error; err != nil {
				s.appendInstallLog(installation, fmt.Sprintf("❌ Failed to record installation in database: %v", err))
				s.updateInstallStatus(installation, models.InstallationStatusFailed, fmt.Sprintf("database error: %v", err))
				return
			}

			s.appendInstallLog(installation, fmt.Sprintf("✓ Installation complete: %s (detected version %s)", def.Name, version))
			s.updateInstallStatus(installation, models.InstallationStatusSuccess, "")
			return
		}
	}

	// Check for passwordless sudo if install command contains sudo
	if strings.Contains(def.Commands.Install, "sudo") {
		s.appendInstallLog(installation, "▶ Checking passwordless sudo access...")
		_, testErr := s.sshClient.Execute(host, "sudo -n true")
		if testErr != nil {
			s.appendInstallLog(installation, "❌ Passwordless sudo check failed")
			s.updateInstallStatus(installation, models.InstallationStatusFailed, "Passwordless sudo not configured. Please configure passwordless sudo for the SSH user.")
			return
		}
		s.appendInstallLog(installation, "✓ Passwordless sudo confirmed")
	}

	// Run install command
	if def.Commands.Install == "" {
		s.appendInstallLog(installation, "❌ No install command defined")
		s.updateInstallStatus(installation, models.InstallationStatusFailed, "No install command defined for this software")
		return
	}

	s.appendInstallLog(installation, "▶ Running installation command (this may take several minutes)...")
	_, err := s.sshClient.ExecuteWithTimeout(host, def.Commands.Install, 15*time.Minute)
	if err != nil {
		s.appendInstallLog(installation, fmt.Sprintf("❌ Installation failed: %v", err))
		s.updateInstallStatus(installation, models.InstallationStatusFailed, fmt.Sprintf("Installation command failed: %v", err))
		return
	}
	s.appendInstallLog(installation, "✓ Installation command completed successfully")

	// Run post-install hook if defined
	if def.Commands.PostInstall != "" {
		s.appendInstallLog(installation, "▶ Running post-installation tasks...")

		// Replace $USER placeholder with actual username from device
		postInstallCmd := def.Commands.PostInstall
		if device.Username != "" {
			postInstallCmd = strings.ReplaceAll(postInstallCmd, "$USER", device.Username)
		} else if strings.Contains(postInstallCmd, "$USER") {
			// Username not available - filter out lines containing $USER
			var filteredLines []string
			for _, line := range strings.Split(postInstallCmd, "\n") {
				if !strings.Contains(line, "$USER") {
					filteredLines = append(filteredLines, line)
				}
			}
			postInstallCmd = strings.Join(filteredLines, "\n")
			s.appendInstallLog(installation, "⚠️  Username not available - skipping user group commands")
		}

		output, err := s.sshClient.Execute(host, postInstallCmd)
		if err != nil {
			s.appendInstallLog(installation, fmt.Sprintf("⚠️  Post-install tasks failed (non-fatal): %v", err))
			if strings.TrimSpace(output) != "" {
				s.appendInstallLog(installation, fmt.Sprintf("Output: %s", strings.TrimSpace(output)))
			}
			// Don't fail the installation if post-install fails
		} else {
			s.appendInstallLog(installation, "✓ Post-installation tasks completed")
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

	// Create InstalledSoftware record (check for errors to ensure data consistency)
	software := &models.InstalledSoftware{
		DeviceID:    device.ID,
		Name:        softwareName,
		Version:     version,
		InstalledBy: "system",
	}

	if err := s.db.Create(software).Error; err != nil {
		s.appendInstallLog(installation, fmt.Sprintf("❌ Failed to record installation in database: %v", err))
		s.updateInstallStatus(installation, models.InstallationStatusFailed, fmt.Sprintf("software installed but database error: %v", err))
		return
	}

	s.appendInstallLog(installation, fmt.Sprintf("✓ %s installed successfully (version: %s)", def.Name, version))
	s.updateInstallStatus(installation, models.InstallationStatusSuccess, "")
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

// appendInstallLog adds a timestamped log entry to the software installation
func (s *SoftwareService) appendInstallLog(installation *models.SoftwareInstallation, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] %s\n", timestamp, message)
	installation.InstallLogs += logEntry

	// Save logs to database
	s.db.Model(installation).Update("install_logs", installation.InstallLogs)

	// Broadcast log update via WebSocket (check nil early for efficiency)
	if s.wsHub == nil {
		return
	}
	s.wsHub.Broadcast("software", "software:log", map[string]interface{}{
		"id":      installation.ID,
		"message": logEntry,
	})
}

// updateInstallStatus updates the installation status and broadcasts to WebSocket
func (s *SoftwareService) updateInstallStatus(installation *models.SoftwareInstallation, status models.SoftwareInstallationStatus, errorDetails string) {
	installation.Status = status
	installation.ErrorDetails = errorDetails

	if status == models.InstallationStatusSuccess || status == models.InstallationStatusFailed {
		now := time.Now()
		installation.CompletedAt = &now
	}

	// Save to database
	s.db.Save(installation)

	// Broadcast status update via WebSocket (check nil early for efficiency)
	if s.wsHub == nil {
		return
	}
	s.wsHub.Broadcast("software", "software:status", map[string]interface{}{
		"id":            installation.ID,
		"status":        installation.Status,
		"error_details": installation.ErrorDetails,
	})
}

// GetInstallation retrieves a software installation by ID
func (s *SoftwareService) GetInstallation(id uuid.UUID) (*models.SoftwareInstallation, error) {
	var installation models.SoftwareInstallation
	if err := s.db.Preload("Device").First(&installation, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &installation, nil
}

// ListInstallations retrieves all software installations for a device
func (s *SoftwareService) ListInstallations(deviceID uuid.UUID) ([]models.SoftwareInstallation, error) {
	installations := make([]models.SoftwareInstallation, 0)
	err := s.db.Where("device_id = ?", deviceID).
		Order("created_at DESC").
		Find(&installations).Error
	return installations, err
}

// GetActiveInstallation retrieves the currently active installation for a device (if any)
func (s *SoftwareService) GetActiveInstallation(deviceID uuid.UUID) (*models.SoftwareInstallation, error) {
	var installations []models.SoftwareInstallation
	err := s.db.Where("device_id = ? AND status IN (?)", deviceID, []string{"pending", "installing"}).
		Order("created_at DESC").
		Limit(1).
		Find(&installations).Error

	if err != nil {
		return nil, err
	}

	if len(installations) == 0 {
		return nil, nil // No active installation
	}

	return &installations[0], nil
}
