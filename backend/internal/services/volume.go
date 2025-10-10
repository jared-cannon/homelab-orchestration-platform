package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/models"
	"github.com/jaredcannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// VolumeService handles Docker volume management
type VolumeService struct {
	db              *gorm.DB
	sshClient       *ssh.Client
	softwareService *SoftwareService
}

// NewVolumeService creates a new volume service
func NewVolumeService(db *gorm.DB, sshClient *ssh.Client, softwareService *SoftwareService) *VolumeService {
	return &VolumeService{
		db:              db,
		sshClient:       sshClient,
		softwareService: softwareService,
	}
}

// CreateLocalVolume creates a standard local Docker volume
func (s *VolumeService) CreateLocalVolume(deviceID uuid.UUID, name string) (*models.Volume, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Volume] Creating local volume '%s' on %s", name, device.Name)

	// Ensure Docker is installed
	if err := s.ensureDockerInstalled(host, deviceID); err != nil {
		return nil, err
	}

	// Check if volume already exists
	existing, _ := s.checkVolumeExists(host, name)
	if existing {
		log.Printf("[Volume] Volume '%s' already exists", name)

		// Return existing record or create one
		var existingVol models.Volume
		err := s.db.Where("device_id = ? AND name = ?", deviceID, name).First(&existingVol).Error
		if err == nil {
			return &existingVol, nil
		}
	}

	// Create volume
	createCmd := fmt.Sprintf("docker volume create %s", name)
	_, err = s.sshClient.Execute(host, createCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	// Get volume details
	inspectCmd := fmt.Sprintf("docker volume inspect %s --format '{{json .}}'", name)
	inspectOutput, err := s.sshClient.Execute(host, inspectCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}

	// Parse volume info
	var volumeInfo map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(inspectOutput)), &volumeInfo); err != nil {
		log.Printf("[Volume] Warning: failed to parse volume info: %v", err)
	}

	log.Printf("[Volume] Local volume '%s' created successfully", name)

	// Record in database
	volume := &models.Volume{
		DeviceID: deviceID,
		Name:     name,
		Type:     models.VolumeTypeLocal,
		Driver:   "local",
		InUse:    false,
	}

	if err := s.db.Create(volume).Error; err != nil {
		return nil, fmt.Errorf("failed to record volume: %w", err)
	}

	return volume, nil
}

// CreateNFSVolume creates a Docker volume backed by NFS
func (s *VolumeService) CreateNFSVolume(deviceID uuid.UUID, name, nfsServerIP, nfsPath string, options map[string]string) (*models.Volume, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Volume] Creating NFS volume '%s' on %s: %s:%s", name, device.Name, nfsServerIP, nfsPath)

	// Ensure Docker is installed
	if err := s.ensureDockerInstalled(host, deviceID); err != nil {
		return nil, err
	}

	// Check if volume already exists
	existing, _ := s.checkVolumeExists(host, name)
	if existing {
		log.Printf("[Volume] Volume '%s' already exists", name)

		var existingVol models.Volume
		err := s.db.Where("device_id = ? AND name = ?", deviceID, name).First(&existingVol).Error
		if err == nil {
			return &existingVol, nil
		}
	}

	// Build NFS mount options
	nfsOpts := "addr=" + nfsServerIP
	if options != nil {
		for k, v := range options {
			nfsOpts += "," + k
			if v != "" {
				nfsOpts += "=" + v
			}
		}
	} else {
		nfsOpts += ",rw"
	}

	// Create NFS volume using local driver with NFS options
	createCmd := fmt.Sprintf(
		"docker volume create --name %s --driver local --opt type=nfs --opt o=%s --opt device=:%s",
		name, nfsOpts, nfsPath,
	)

	log.Printf("[Volume] Running: %s", createCmd)
	_, err = s.sshClient.Execute(host, createCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to create NFS volume: %w - ensure NFS server is accessible", err)
	}

	log.Printf("[Volume] NFS volume '%s' created successfully", name)

	// Serialize driver options
	driverOpts := map[string]string{
		"type":   "nfs",
		"o":      nfsOpts,
		"device": ":" + nfsPath,
	}
	driverOptsJSON, _ := json.Marshal(driverOpts)

	// Record in database
	volume := &models.Volume{
		DeviceID:    deviceID,
		Name:        name,
		Type:        models.VolumeTypeNFS,
		Driver:      "local",
		DriverOpts:  driverOptsJSON,
		NFSServerIP: nfsServerIP,
		NFSPath:     nfsPath,
		InUse:       false,
	}

	if err := s.db.Create(volume).Error; err != nil {
		return nil, fmt.Errorf("failed to record volume: %w", err)
	}

	return volume, nil
}

// ListVolumes lists all Docker volumes on a device
func (s *VolumeService) ListVolumes(deviceID uuid.UUID) ([]models.Volume, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Volume] Listing volumes on %s", device.Name)

	// Check if Docker is installed
	installed, _, _ := s.softwareService.IsInstalled(host, models.SoftwareDocker)
	if !installed {
		log.Printf("[Volume] Docker not installed on %s", device.Name)
		return []models.Volume{}, nil
	}

	// List volumes from Docker
	listCmd := "docker volume ls --format '{{.Name}}'"
	output, err := s.sshClient.Execute(host, listCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	volumeNames := strings.Split(strings.TrimSpace(output), "\n")

	// Sync with database
	var dbVolumes []models.Volume
	s.db.Where("device_id = ?", deviceID).Find(&dbVolumes)

	// Create a map of existing volumes
	existingMap := make(map[string]*models.Volume)
	for i := range dbVolumes {
		existingMap[dbVolumes[i].Name] = &dbVolumes[i]
	}

	// Check each volume from Docker
	for _, name := range volumeNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// If not in DB, add it
		if _, exists := existingMap[name]; !exists {
			log.Printf("[Volume] Discovered volume '%s' not in database, adding...", name)

			volume := &models.Volume{
				DeviceID: deviceID,
				Name:     name,
				Type:     models.VolumeTypeLocal,
				Driver:   "local",
				InUse:    false,
			}

			// Try to inspect to get more details
			inspectCmd := fmt.Sprintf("docker volume inspect %s --format '{{.Driver}}'", name)
			driver, err := s.sshClient.Execute(host, inspectCmd)
			if err == nil {
				volume.Driver = strings.TrimSpace(driver)
			}

			s.db.Create(volume)
			dbVolumes = append(dbVolumes, *volume)
		}
	}

	// Check for volumes in DB that no longer exist in Docker
	dockerVolSet := make(map[string]bool)
	for _, name := range volumeNames {
		dockerVolSet[strings.TrimSpace(name)] = true
	}

	for i := range dbVolumes {
		if !dockerVolSet[dbVolumes[i].Name] {
			log.Printf("[Volume] Volume '%s' in DB but not in Docker, marking as deleted", dbVolumes[i].Name)
			s.db.Delete(&dbVolumes[i])
		}
	}

	// Reload from DB to get accurate state
	s.db.Where("device_id = ?", deviceID).Find(&dbVolumes)

	return dbVolumes, nil
}

// GetVolume gets details about a specific volume
func (s *VolumeService) GetVolume(deviceID uuid.UUID, volumeName string) (*models.Volume, error) {
	var volume models.Volume
	err := s.db.Where("device_id = ? AND name = ?", deviceID, volumeName).First(&volume).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("volume not found")
		}
		return nil, err
	}

	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	// Check if volume is in use
	inUse, err := s.checkVolumeInUse(host, volumeName)
	if err == nil && inUse != volume.InUse {
		// Update in-use status
		s.db.Model(&volume).Update("in_use", inUse)
		volume.InUse = inUse
	}

	return &volume, nil
}

// RemoveVolume removes a Docker volume
func (s *VolumeService) RemoveVolume(deviceID uuid.UUID, volumeName string, force bool) error {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return err
	}

	host := device.IPAddress + ":22"

	log.Printf("[Volume] Removing volume '%s' from %s", volumeName, device.Name)

	// Check if in use
	inUse, _ := s.checkVolumeInUse(host, volumeName)
	if inUse && !force {
		return fmt.Errorf("volume is in use by containers - use force to remove anyway")
	}

	// Remove volume
	removeCmd := fmt.Sprintf("docker volume rm %s", volumeName)
	if force {
		removeCmd = fmt.Sprintf("docker volume rm -f %s", volumeName)
	}

	_, err = s.sshClient.Execute(host, removeCmd)
	if err != nil {
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	// Remove from database
	err = s.db.Where("device_id = ? AND name = ?", deviceID, volumeName).Delete(&models.Volume{}).Error
	if err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}

	log.Printf("[Volume] Volume '%s' removed successfully", volumeName)
	return nil
}

// InspectVolume gets detailed information about a volume from Docker
func (s *VolumeService) InspectVolume(deviceID uuid.UUID, volumeName string) (map[string]interface{}, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	inspectCmd := fmt.Sprintf("docker volume inspect %s", volumeName)
	output, err := s.sshClient.Execute(host, inspectCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}

	// Parse JSON output
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("volume not found")
	}

	return result[0], nil
}

// Helper functions

func (s *VolumeService) ensureDockerInstalled(host string, deviceID uuid.UUID) error {
	installed, _, _ := s.softwareService.IsInstalled(host, models.SoftwareDocker)
	if !installed {
		return fmt.Errorf("Docker is not installed on this device - install Docker first")
	}
	return nil
}

func (s *VolumeService) checkVolumeExists(host, volumeName string) (bool, error) {
	checkCmd := fmt.Sprintf("docker volume inspect %s", volumeName)
	_, err := s.sshClient.Execute(host, checkCmd)
	return err == nil, err
}

func (s *VolumeService) checkVolumeInUse(host, volumeName string) (bool, error) {
	// Check if any containers are using this volume
	checkCmd := fmt.Sprintf("docker ps -a --filter volume=%s --format '{{.ID}}'", volumeName)
	output, err := s.sshClient.Execute(host, checkCmd)
	if err != nil {
		return false, err
	}

	containers := strings.TrimSpace(output)
	return containers != "", nil
}

func (s *VolumeService) getDevice(deviceID uuid.UUID) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, "id = ?", deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("device not found")
		}
		return nil, err
	}
	return &device, nil
}
