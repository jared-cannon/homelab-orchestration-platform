package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jared-cannon/homelab-orchestration-platform/internal/models"
	"gopkg.in/yaml.v3"
)

// SoftwareRegistry manages software definitions
type SoftwareRegistry struct {
	definitions map[string]*models.SoftwareDefinition
	definitionsPath string
}

// NewSoftwareRegistry creates a new software registry
func NewSoftwareRegistry(definitionsPath string) *SoftwareRegistry {
	registry := &SoftwareRegistry{
		definitions: make(map[string]*models.SoftwareDefinition),
		definitionsPath: definitionsPath,
	}

	if err := registry.loadDefinitions(); err != nil {
		log.Printf("[SoftwareRegistry] Warning: failed to load definitions: %v", err)
	}

	return registry
}

// loadDefinitions loads all YAML software definitions from the definitions directory
func (r *SoftwareRegistry) loadDefinitions() error {
	// Check if directory exists
	if _, err := os.Stat(r.definitionsPath); os.IsNotExist(err) {
		log.Printf("[SoftwareRegistry] Definitions directory not found: %s", r.definitionsPath)
		return fmt.Errorf("definitions directory not found: %s", r.definitionsPath)
	}

	// Read all YAML files in the directory
	files, err := filepath.Glob(filepath.Join(r.definitionsPath, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob definition files: %w", err)
	}

	yamlFiles, err := filepath.Glob(filepath.Join(r.definitionsPath, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to glob yml files: %w", err)
	}
	files = append(files, yamlFiles...)

	if len(files) == 0 {
		log.Printf("[SoftwareRegistry] No definition files found in %s", r.definitionsPath)
		return nil
	}

	// Load each definition file
	for _, file := range files {
		if err := r.loadDefinitionFile(file); err != nil {
			log.Printf("[SoftwareRegistry] Warning: failed to load %s: %v", file, err)
			continue
		}
	}

	log.Printf("[SoftwareRegistry] Loaded %d software definitions", len(r.definitions))
	return nil
}

// loadDefinitionFile loads a single software definition from a YAML file
func (r *SoftwareRegistry) loadDefinitionFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var def models.SoftwareDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	if def.ID == "" {
		return fmt.Errorf("definition missing required 'id' field")
	}

	r.definitions[def.ID] = &def
	log.Printf("[SoftwareRegistry] Loaded definition: %s (%s)", def.ID, def.Name)
	return nil
}

// GetDefinition returns a software definition by ID
func (r *SoftwareRegistry) GetDefinition(id string) (*models.SoftwareDefinition, error) {
	def, exists := r.definitions[id]
	if !exists {
		return nil, fmt.Errorf("software definition not found: %s", id)
	}
	return def, nil
}

// ListDefinitions returns all available software definitions
func (r *SoftwareRegistry) ListDefinitions() []*models.SoftwareDefinition {
	definitions := make([]*models.SoftwareDefinition, 0, len(r.definitions))
	for _, def := range r.definitions {
		definitions = append(definitions, def)
	}
	return definitions
}

// GetDefinitionsByCategory returns all definitions in a specific category
func (r *SoftwareRegistry) GetDefinitionsByCategory(category string) []*models.SoftwareDefinition {
	definitions := make([]*models.SoftwareDefinition, 0)
	for _, def := range r.definitions {
		if def.Category == category {
			definitions = append(definitions, def)
		}
	}
	return definitions
}

// ReloadDefinitions reloads all software definitions from disk
func (r *SoftwareRegistry) ReloadDefinitions() error {
	r.definitions = make(map[string]*models.SoftwareDefinition)
	return r.loadDefinitions()
}
