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

// NFSService handles NFS server and client configuration
type NFSService struct {
	db              *gorm.DB
	sshClient       *ssh.Client
	softwareService *SoftwareService
}

// NewNFSService creates a new NFS service
func NewNFSService(db *gorm.DB, sshClient *ssh.Client, softwareService *SoftwareService) *NFSService {
	return &NFSService{
		db:              db,
		sshClient:       sshClient,
		softwareService: softwareService,
	}
}

// SetupServer configures a device as an NFS server
func (s *NFSService) SetupServer(deviceID uuid.UUID, exportPath, clientCIDR, options string) (*models.NFSExport, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[NFS] Setting up NFS server on %s", device.Name)

	// Ensure NFS server software is installed
	installed, _, _ := s.softwareService.IsInstalled(host, models.SoftwareNFSServer)
	if !installed {
		log.Printf("[NFS] Installing NFS server software...")
		_, err = s.softwareService.InstallNFSServer(deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to install NFS server: %w", err)
		}
	}

	// Create export directory
	log.Printf("[NFS] Creating export directory: %s", exportPath)
	_, err = s.sshClient.Execute(host, fmt.Sprintf("sudo mkdir -p %s", exportPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	// Set permissions
	_, err = s.sshClient.Execute(host, fmt.Sprintf("sudo chown nobody:nogroup %s", exportPath))
	if err != nil {
		log.Printf("[NFS] Warning: failed to set ownership: %v", err)
	}

	_, err = s.sshClient.Execute(host, fmt.Sprintf("sudo chmod 755 %s", exportPath))
	if err != nil {
		log.Printf("[NFS] Warning: failed to set permissions: %v", err)
	}

	// Check if export already exists
	existingExports, _ := s.sshClient.Execute(host, "cat /etc/exports")
	if strings.Contains(existingExports, exportPath) {
		log.Printf("[NFS] Export %s already exists in /etc/exports", exportPath)
	} else {
		// Add to /etc/exports
		exportLine := fmt.Sprintf("%s %s(%s)", exportPath, clientCIDR, options)
		log.Printf("[NFS] Adding export: %s", exportLine)

		// Append to /etc/exports
		cmd := fmt.Sprintf("echo '%s' | sudo tee -a /etc/exports", exportLine)
		_, err = s.sshClient.Execute(host, cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to add export to /etc/exports: %w", err)
		}
	}

	// Apply exports
	log.Printf("[NFS] Applying exports...")
	_, err = s.sshClient.Execute(host, "sudo exportfs -ra")
	if err != nil {
		return nil, fmt.Errorf("failed to apply exports: %w", err)
	}

	// Restart NFS server
	log.Printf("[NFS] Restarting NFS server...")
	_, err = s.sshClient.Execute(host, "sudo systemctl restart nfs-kernel-server")
	if err != nil {
		return nil, fmt.Errorf("failed to restart NFS server: %w", err)
	}

	// Verify export
	showmountOutput, err := s.sshClient.Execute(host, "showmount -e localhost")
	if err != nil {
		log.Printf("[NFS] Warning: showmount failed: %v", err)
	} else {
		log.Printf("[NFS] Current exports:\n%s", showmountOutput)
	}

	log.Printf("[NFS] NFS server setup complete on %s", device.Name)

	// Record export in database
	export := &models.NFSExport{
		DeviceID:   deviceID,
		Path:       exportPath,
		ClientCIDR: clientCIDR,
		Options:    options,
		Active:     true,
	}

	if err := s.db.Create(export).Error; err != nil {
		return nil, fmt.Errorf("failed to record export: %w", err)
	}

	return export, nil
}

// CreateExport adds a new export to an existing NFS server
func (s *NFSService) CreateExport(deviceID uuid.UUID, exportPath, clientCIDR, options string) (*models.NFSExport, error) {
	// Check if export already exists
	var existing models.NFSExport
	err := s.db.Where("device_id = ? AND path = ?", deviceID, exportPath).First(&existing).Error
	if err == nil {
		return &existing, nil // Already exists
	}

	return s.SetupServer(deviceID, exportPath, clientCIDR, options)
}

// ListExports lists all NFS exports for a device
func (s *NFSService) ListExports(deviceID uuid.UUID) ([]models.NFSExport, error) {
	var exports []models.NFSExport
	err := s.db.Where("device_id = ?", deviceID).Find(&exports).Error
	return exports, err
}

// RemoveExport removes an NFS export
func (s *NFSService) RemoveExport(deviceID uuid.UUID, exportID uuid.UUID) error {
	var export models.NFSExport
	if err := s.db.First(&export, "id = ? AND device_id = ?", exportID, deviceID).Error; err != nil {
		return fmt.Errorf("export not found")
	}

	device, err := s.getDevice(deviceID)
	if err != nil {
		return err
	}

	host := device.IPAddress + ":22"

	log.Printf("[NFS] Removing export %s from %s", export.Path, device.Name)

	// Remove from /etc/exports
	cmd := fmt.Sprintf("sudo sed -i '\\|%s|d' /etc/exports", export.Path)
	_, err = s.sshClient.Execute(host, cmd)
	if err != nil {
		return fmt.Errorf("failed to remove from /etc/exports: %w", err)
	}

	// Re-export
	_, err = s.sshClient.Execute(host, "sudo exportfs -ra")
	if err != nil {
		return fmt.Errorf("failed to re-export: %w", err)
	}

	// Remove from database
	if err := s.db.Delete(&export).Error; err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}

	log.Printf("[NFS] Export removed successfully")
	return nil
}

// MountShare mounts an NFS share on a client device
func (s *NFSService) MountShare(deviceID uuid.UUID, serverIP, remotePath, localPath, options string, permanent bool) (*models.NFSMount, error) {
	device, err := s.getDevice(deviceID)
	if err != nil {
		return nil, err
	}

	host := device.IPAddress + ":22"

	log.Printf("[NFS] Mounting NFS share on %s: %s:%s -> %s", device.Name, serverIP, remotePath, localPath)

	// Ensure NFS client is installed
	installed, _, _ := s.softwareService.IsInstalled(host, models.SoftwareNFSClient)
	if !installed {
		log.Printf("[NFS] Installing NFS client...")
		_, err = s.softwareService.InstallNFSClient(deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to install NFS client: %w", err)
		}
	}

	// Check if already mounted
	mountCheckCmd := fmt.Sprintf("mount | grep %s", localPath)
	output, _ := s.sshClient.Execute(host, mountCheckCmd)
	if strings.Contains(output, localPath) {
		log.Printf("[NFS] Share already mounted at %s", localPath)

		// Return existing mount record
		var existing models.NFSMount
		err := s.db.Where("device_id = ? AND local_path = ?", deviceID, localPath).First(&existing).Error
		if err == nil {
			return &existing, nil
		}
	}

	// Create mount point
	_, err = s.sshClient.Execute(host, fmt.Sprintf("sudo mkdir -p %s", localPath))
	if err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	// Test connectivity to NFS server
	log.Printf("[NFS] Testing connectivity to NFS server...")
	pingCmd := fmt.Sprintf("ping -c 1 -W 2 %s", serverIP)
	_, err = s.sshClient.Execute(host, pingCmd)
	if err != nil {
		return nil, fmt.Errorf("cannot reach NFS server at %s", serverIP)
	}

	// Check if export is available
	showmountCmd := fmt.Sprintf("showmount -e %s", serverIP)
	exports, err := s.sshClient.Execute(host, showmountCmd)
	if err != nil {
		log.Printf("[NFS] Warning: showmount failed: %v", err)
	} else {
		log.Printf("[NFS] Available exports:\n%s", exports)
	}

	// Mount the share
	if options == "" {
		options = "defaults"
	}
	mountCmd := fmt.Sprintf("sudo mount -t nfs -o %s %s:%s %s", options, serverIP, remotePath, localPath)
	log.Printf("[NFS] Mounting: %s", mountCmd)

	_, err = s.sshClient.Execute(host, mountCmd)
	if err != nil {
		return nil, fmt.Errorf("mount failed: %w - check that NFS server allows this client", err)
	}

	// Verify mount
	verifyCmd := fmt.Sprintf("df -h | grep %s", localPath)
	mountInfo, err := s.sshClient.Execute(host, verifyCmd)
	if err == nil {
		log.Printf("[NFS] Mount verified:\n%s", mountInfo)
	}

	// Add to fstab if permanent
	if permanent {
		log.Printf("[NFS] Adding to /etc/fstab for permanent mount...")
		fstabLine := fmt.Sprintf("%s:%s %s nfs %s 0 0", serverIP, remotePath, localPath, options)

		// Check if already in fstab
		fstabContent, _ := s.sshClient.Execute(host, "cat /etc/fstab")
		if !strings.Contains(fstabContent, localPath) {
			fstabCmd := fmt.Sprintf("echo '%s' | sudo tee -a /etc/fstab", fstabLine)
			_, err = s.sshClient.Execute(host, fstabCmd)
			if err != nil {
				log.Printf("[NFS] Warning: failed to add to fstab: %v", err)
			}
		}
	}

	log.Printf("[NFS] NFS mount successful on %s", device.Name)

	// Record mount in database
	mount := &models.NFSMount{
		DeviceID:   deviceID,
		ServerIP:   serverIP,
		RemotePath: remotePath,
		LocalPath:  localPath,
		Options:    options,
		Permanent:  permanent,
		Active:     true,
	}

	if err := s.db.Create(mount).Error; err != nil {
		return nil, fmt.Errorf("failed to record mount: %w", err)
	}

	return mount, nil
}

// ListMounts lists all NFS mounts for a device
func (s *NFSService) ListMounts(deviceID uuid.UUID) ([]models.NFSMount, error) {
	var mounts []models.NFSMount
	err := s.db.Where("device_id = ?", deviceID).Find(&mounts).Error
	return mounts, err
}

// UnmountShare unmounts an NFS share
func (s *NFSService) UnmountShare(deviceID uuid.UUID, mountID uuid.UUID, removeFromFstab bool) error {
	var mount models.NFSMount
	if err := s.db.First(&mount, "id = ? AND device_id = ?", mountID, deviceID).Error; err != nil {
		return fmt.Errorf("mount not found")
	}

	device, err := s.getDevice(deviceID)
	if err != nil {
		return err
	}

	host := device.IPAddress + ":22"

	log.Printf("[NFS] Unmounting %s from %s", mount.LocalPath, device.Name)

	// Unmount
	unmountCmd := fmt.Sprintf("sudo umount %s", mount.LocalPath)
	_, err = s.sshClient.Execute(host, unmountCmd)
	if err != nil {
		return fmt.Errorf("unmount failed: %w", err)
	}

	// Remove from fstab if requested
	if removeFromFstab {
		log.Printf("[NFS] Removing from /etc/fstab...")
		cmd := fmt.Sprintf("sudo sed -i '\\|%s|d' /etc/fstab", mount.LocalPath)
		_, err = s.sshClient.Execute(host, cmd)
		if err != nil {
			log.Printf("[NFS] Warning: failed to remove from fstab: %v", err)
		}
	}

	// Remove from database
	if err := s.db.Delete(&mount).Error; err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}

	log.Printf("[NFS] Unmount successful")
	return nil
}

// getDevice retrieves device by ID
func (s *NFSService) getDevice(deviceID uuid.UUID) (*models.Device, error) {
	var device models.Device
	if err := s.db.First(&device, "id = ?", deviceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("device not found")
		}
		return nil, err
	}
	return &device, nil
}
