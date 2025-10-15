package services

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"github.com/jared-cannon/homelab-orchestration-platform/internal/ssh"
	"gorm.io/gorm"
)

// DeviceScorer scores devices based on their resources vs recipe requirements
type DeviceScorer struct {
	db        *gorm.DB
	sshClient *ssh.Client
}

// NewDeviceScorer creates a new device scorer
func NewDeviceScorer(db *gorm.DB, sshClient *ssh.Client) *DeviceScorer {
	return &DeviceScorer{
		db:        db,
		sshClient: sshClient,
	}
}

// DeviceScore represents a device's suitability score for a recipe
type DeviceScore struct {
	DeviceID       uuid.UUID `json:"device_id"`
	DeviceName     string    `json:"device_name"`
	DeviceIP       string    `json:"device_ip"`
	Score          int       `json:"score"`          // 0-100
	Recommendation string    `json:"recommendation"` // "best", "good", "not-recommended"
	Reasons        []string  `json:"reasons"`
	Available      bool      `json:"available"` // Can this device be used at all?
}

// RecipeRequirements represents resource requirements from a recipe
type RecipeRequirements struct {
	MinRAMMB     int
	MinStorageGB int
	CPUCores     int
}

// DeviceResources represents available resources on a device
type DeviceResources struct {
	AvailableRAMMB    int
	AvailableStorageGB int
	TotalCPUCores      int
	DockerInstalled    bool
	DockerRunning      bool
}

// ScoreDevicesForRecipe scores all online devices for a recipe
func (s *DeviceScorer) ScoreDevicesForRecipe(requirements RecipeRequirements) ([]DeviceScore, error) {
	// Get all online devices
	var devices []models.Device
	if err := s.db.Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch devices: %w", err)
	}

	scores := make([]DeviceScore, 0, len(devices))

	for _, device := range devices {
		score := s.scoreDevice(device, requirements)
		scores = append(scores, score)
	}

	// Sort by score (highest first) - simple bubble sort for now
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	return scores, nil
}

// scoreDevice scores a single device against recipe requirements
func (s *DeviceScorer) scoreDevice(device models.Device, requirements RecipeRequirements) DeviceScore {
	score := DeviceScore{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		DeviceIP:   device.IPAddress,
		Score:      0,
		Reasons:    make([]string, 0),
		Available:  true,
	}

	// Get device resources
	resources, err := s.getDeviceResources(device)
	if err != nil {
		score.Available = false
		score.Reasons = append(score.Reasons, fmt.Sprintf("❌ Unable to check resources: %v", err))
		score.Recommendation = "not-recommended"
		return score
	}

	// Check Docker (required) - 20 points
	if !resources.DockerInstalled {
		score.Available = false
		score.Reasons = append(score.Reasons, "❌ Docker not installed")
		score.Recommendation = "not-recommended"
		return score
	}
	if !resources.DockerRunning {
		score.Available = false
		score.Reasons = append(score.Reasons, "❌ Docker not running")
		score.Recommendation = "not-recommended"
		return score
	}
	score.Score += 20
	score.Reasons = append(score.Reasons, "✓ Docker installed and running")

	// Check RAM (40 points max)
	ramScore, ramReason := s.scoreRAM(resources.AvailableRAMMB, requirements.MinRAMMB)
	score.Score += ramScore
	score.Reasons = append(score.Reasons, ramReason)
	if ramScore == 0 {
		score.Available = false
	}

	// Check Storage (30 points max)
	storageScore, storageReason := s.scoreStorage(resources.AvailableStorageGB, requirements.MinStorageGB)
	score.Score += storageScore
	score.Reasons = append(score.Reasons, storageReason)
	if storageScore == 0 {
		score.Available = false
	}

	// Check CPU (10 points max)
	cpuScore, cpuReason := s.scoreCPU(resources.TotalCPUCores, requirements.CPUCores)
	score.Score += cpuScore
	score.Reasons = append(score.Reasons, cpuReason)

	// Set recommendation based on final score
	if !score.Available {
		score.Recommendation = "not-recommended"
	} else if score.Score >= 80 {
		score.Recommendation = "best"
	} else if score.Score >= 60 {
		score.Recommendation = "good"
	} else {
		score.Recommendation = "acceptable"
	}

	return score
}

// scoreRAM scores RAM availability (0-40 points)
func (s *DeviceScorer) scoreRAM(availableMB, requiredMB int) (int, string) {
	if requiredMB == 0 {
		return 40, "✓ No specific RAM requirement"
	}

	ratio := float64(availableMB) / float64(requiredMB)

	if ratio < 1.0 {
		return 0, fmt.Sprintf("❌ Insufficient RAM (%d MB available, need %d MB)", availableMB, requiredMB)
	} else if ratio >= 4.0 {
		return 40, fmt.Sprintf("✓ Plenty of RAM (%d MB available, app needs %d MB)", availableMB, requiredMB)
	} else if ratio >= 2.0 {
		return 30, fmt.Sprintf("✓ Good RAM availability (%d MB available, app needs %d MB)", availableMB, requiredMB)
	} else if ratio >= 1.5 {
		return 20, fmt.Sprintf("✓ Sufficient RAM (%d MB available, app needs %d MB)", availableMB, requiredMB)
	} else {
		return 10, fmt.Sprintf("⚠️ Tight RAM fit (%d MB available, app needs %d MB)", availableMB, requiredMB)
	}
}

// scoreStorage scores storage availability (0-30 points)
func (s *DeviceScorer) scoreStorage(availableGB, requiredGB int) (int, string) {
	if requiredGB == 0 {
		return 30, "✓ No specific storage requirement"
	}

	ratio := float64(availableGB) / float64(requiredGB)

	if ratio < 1.0 {
		return 0, fmt.Sprintf("❌ Insufficient storage (%d GB available, need %d GB)", availableGB, requiredGB)
	} else if ratio >= 10.0 {
		return 30, fmt.Sprintf("✓ Plenty of storage (%d GB available, app needs %d GB)", availableGB, requiredGB)
	} else if ratio >= 5.0 {
		return 25, fmt.Sprintf("✓ Good storage availability (%d GB available, app needs %d GB)", availableGB, requiredGB)
	} else if ratio >= 2.0 {
		return 20, fmt.Sprintf("✓ Sufficient storage (%d GB available, app needs %d GB)", availableGB, requiredGB)
	} else {
		return 10, fmt.Sprintf("⚠️ Limited storage (%d GB available, app needs %d GB)", availableGB, requiredGB)
	}
}

// scoreCPU scores CPU cores (0-10 points)
func (s *DeviceScorer) scoreCPU(totalCores, requiredCores int) (int, string) {
	if requiredCores == 0 || totalCores == 0 {
		return 10, "✓ CPU cores available"
	}

	if totalCores >= requiredCores*2 {
		return 10, fmt.Sprintf("✓ %d CPU cores (%d required)", totalCores, requiredCores)
	} else if totalCores >= requiredCores {
		return 7, fmt.Sprintf("✓ %d CPU cores (%d required)", totalCores, requiredCores)
	} else {
		return 5, fmt.Sprintf("⚠️ %d CPU cores (app recommends %d)", totalCores, requiredCores)
	}
}

// getDeviceResources retrieves current resource availability from a device
func (s *DeviceScorer) getDeviceResources(device models.Device) (*DeviceResources, error) {
	host := device.IPAddress + ":22"
	resources := &DeviceResources{}

	// Get available RAM in MB
	output, err := s.sshClient.Execute(host, "free -m | awk 'NR==2 {print $7}'")
	if err == nil {
		if ram, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			resources.AvailableRAMMB = ram
		}
	}

	// Get available storage in GB (for root filesystem)
	output, err = s.sshClient.Execute(host, "df -BG / | awk 'NR==2 {print $4}' | sed 's/G//'")
	if err == nil {
		if storage, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			resources.AvailableStorageGB = storage
		}
	}

	// Get CPU cores
	output, err = s.sshClient.Execute(host, "nproc")
	if err == nil {
		if cores, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			resources.TotalCPUCores = cores
		}
	}

	// Check Docker installation
	_, err = s.sshClient.Execute(host, "which docker")
	resources.DockerInstalled = (err == nil)

	// Check if Docker is running
	if resources.DockerInstalled {
		_, err = s.sshClient.Execute(host, "docker info >/dev/null 2>&1")
		resources.DockerRunning = (err == nil)
	}

	return resources, nil
}
