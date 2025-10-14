package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DockerRegistryClient fetches metadata about Docker images
type DockerRegistryClient struct {
	cacheDir      string
	cacheDuration time.Duration
	mu            sync.RWMutex
}

// NewDockerRegistryClient creates a new Docker registry client
func NewDockerRegistryClient(cacheDir string) *DockerRegistryClient {
	return &DockerRegistryClient{
		cacheDir:      cacheDir,
		cacheDuration: 24 * time.Hour, // Cache for 24 hours
	}
}

// ImageMetadata contains metadata about a Docker image
type ImageMetadata struct {
	Repository      string    `json:"repository"`
	LatestTag       string    `json:"latest_tag"`
	Tags            []string  `json:"tags"`
	Description     string    `json:"description"`
	PullCount       int       `json:"pull_count"`
	StarCount       int       `json:"star_count"`
	LastUpdated     time.Time `json:"last_updated"`
	IsOfficial      bool      `json:"is_official"`
	IsAutomated     bool      `json:"is_automated"`
}

// GetImageMetadata fetches metadata for a Docker image from Docker Hub
func (c *DockerRegistryClient) GetImageMetadata(imageName string) (*ImageMetadata, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Parse image name (handle library/ prefix for official images)
	repo := c.parseImageName(imageName)

	// Check cache first
	cacheFile := filepath.Join(c.cacheDir, fmt.Sprintf("%s.json", strings.ReplaceAll(repo, "/", "_")))
	if c.isCacheValid(cacheFile) {
		metadata, err := c.loadFromCache(cacheFile)
		if err == nil {
			return metadata, nil
		}
	}

	// Fetch from Docker Hub API
	metadata, err := c.fetchFromDockerHub(repo)
	if err != nil {
		return nil, err
	}

	// Save to cache
	os.MkdirAll(c.cacheDir, 0755)
	c.saveToCache(cacheFile, metadata)

	return metadata, nil
}

// parseImageName extracts repository name from image reference
func (c *DockerRegistryClient) parseImageName(imageName string) string {
	// Remove tag if present (e.g., "nginx:latest" -> "nginx")
	parts := strings.Split(imageName, ":")
	repo := parts[0]

	// Official images don't have a namespace
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	return repo
}

// fetchFromDockerHub fetches image metadata from Docker Hub API v2
func (c *DockerRegistryClient) fetchFromDockerHub(repo string) (*ImageMetadata, error) {
	// Docker Hub API v2 endpoint
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s", repo)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Docker Hub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Docker Hub returned HTTP %d for %s", resp.StatusCode, repo)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var hubResponse DockerHubResponse
	if err := json.Unmarshal(body, &hubResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Docker Hub response: %w", err)
	}

	// Fetch tags
	tags, err := c.fetchTags(repo)
	if err != nil {
		// Don't fail if we can't get tags, just log it
		tags = []string{}
	}

	// Determine latest tag
	latestTag := "latest"
	if len(tags) > 0 {
		// Look for "latest" tag, or use first tag
		hasLatest := false
		for _, tag := range tags {
			if tag == "latest" {
				hasLatest = true
				break
			}
		}
		if !hasLatest && len(tags) > 0 {
			latestTag = tags[0]
		}
	}

	metadata := &ImageMetadata{
		Repository:  repo,
		LatestTag:   latestTag,
		Tags:        tags,
		Description: hubResponse.Description,
		PullCount:   hubResponse.PullCount,
		StarCount:   hubResponse.StarCount,
		LastUpdated: hubResponse.LastUpdated,
		IsOfficial:  strings.HasPrefix(repo, "library/"),
		IsAutomated: hubResponse.IsAutomated,
	}

	return metadata, nil
}

// fetchTags fetches available tags for an image
func (c *DockerRegistryClient) fetchTags(repo string) ([]string, error) {
	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/tags?page_size=10", repo)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tags: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tagsResponse DockerHubTagsResponse
	if err := json.Unmarshal(body, &tagsResponse); err != nil {
		return nil, err
	}

	tags := make([]string, 0, len(tagsResponse.Results))
	for _, result := range tagsResponse.Results {
		tags = append(tags, result.Name)
	}

	return tags, nil
}

// DockerHubResponse represents the Docker Hub API response
type DockerHubResponse struct {
	Name        string    `json:"name"`
	Namespace   string    `json:"namespace"`
	Description string    `json:"description"`
	PullCount   int       `json:"pull_count"`
	StarCount   int       `json:"star_count"`
	LastUpdated time.Time `json:"last_updated"`
	IsAutomated bool      `json:"is_automated"`
}

// DockerHubTagsResponse represents the tags API response
type DockerHubTagsResponse struct {
	Count   int `json:"count"`
	Results []struct {
		Name        string    `json:"name"`
		LastUpdated time.Time `json:"last_updated"`
	} `json:"results"`
}

// isCacheValid checks if the cache file is still valid
func (c *DockerRegistryClient) isCacheValid(cacheFile string) bool {
	info, err := os.Stat(cacheFile)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	return age < c.cacheDuration
}

// loadFromCache loads metadata from cache file
func (c *DockerRegistryClient) loadFromCache(cacheFile string) (*ImageMetadata, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var metadata ImageMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// saveToCache saves metadata to cache file
func (c *DockerRegistryClient) saveToCache(cacheFile string, metadata *ImageMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// ExtractImageFromCompose extracts the primary Docker image from a compose template
func ExtractImageFromCompose(composeTemplate string) string {
	// Simple extraction: look for "image: " line
	lines := strings.Split(composeTemplate, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "image:") {
			// Extract image name (remove "image: " and any quotes)
			imageName := strings.TrimPrefix(trimmed, "image:")
			imageName = strings.TrimSpace(imageName)
			imageName = strings.Trim(imageName, "\"'")

			// Remove template variables (e.g., {{.Version}})
			// Find the base image before any ':'
			parts := strings.Split(imageName, ":")
			if len(parts) > 0 {
				baseImage := parts[0]
				// Remove any template syntax
				baseImage = strings.Split(baseImage, "{{")[0]
				return strings.TrimSpace(baseImage)
			}

			return imageName
		}
	}

	return ""
}
