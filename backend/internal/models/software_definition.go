package models

// SoftwareDefinition defines how to manage a specific piece of software
type SoftwareDefinition struct {
	ID          string            `json:"id" yaml:"id"`                     // Unique identifier (e.g., "docker")
	Name        string            `json:"name" yaml:"name"`                 // Display name (e.g., "Docker Engine")
	Description string            `json:"description" yaml:"description"`   // Short description
	Category    string            `json:"category" yaml:"category"`         // Category (e.g., "container", "storage", "database")
	Icon        string            `json:"icon,omitempty" yaml:"icon"`       // Icon name or URL
	Commands    SoftwareCommands  `json:"commands" yaml:"commands"`         // Commands for managing the software
	Options     map[string]string `json:"options,omitempty" yaml:"options"` // Additional options/metadata
}

// SoftwareCommands defines the shell commands for managing software
type SoftwareCommands struct {
	// CheckInstalled returns exit code 0 if installed, non-zero otherwise
	// Should output version info to stdout
	CheckInstalled string `json:"check_installed" yaml:"check_installed"`

	// CheckVersion returns the currently installed version
	CheckVersion string `json:"check_version" yaml:"check_version"`

	// CheckUpdates checks if updates are available
	// Should output "updates available" or similar to stdout if updates exist
	CheckUpdates string `json:"check_updates" yaml:"check_updates"`

	// Install installs the software
	Install string `json:"install" yaml:"install"`

	// Update updates the software to the latest version
	Update string `json:"update" yaml:"update"`

	// Uninstall removes the software (optional, can be empty)
	Uninstall string `json:"uninstall,omitempty" yaml:"uninstall"`

	// PostInstall runs after successful installation (optional)
	PostInstall string `json:"post_install,omitempty" yaml:"post_install"`

	// PreUninstall runs before uninstallation (optional)
	PreUninstall string `json:"pre_uninstall,omitempty" yaml:"pre_uninstall"`
}

// SoftwareUpdateInfo represents update availability information
type SoftwareUpdateInfo struct {
	SoftwareID      string `json:"software_id"`
	CurrentVersion  string `json:"current_version"`
	AvailableVersion string `json:"available_version,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	Message         string `json:"message,omitempty"`
}
