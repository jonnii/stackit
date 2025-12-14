package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RepoConfig represents the repository configuration
type RepoConfig struct {
	Trunk                      *string `json:"trunk,omitempty"`
	IsGithubIntegrationEnabled *bool   `json:"isGithubIntegrationEnabled,omitempty"`
}

// GetRepoConfig reads the repository configuration
func GetRepoConfig(repoRoot string) (*RepoConfig, error) {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist - return default
		return &RepoConfig{}, nil
	}

	var config RepoConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse repo config: %w", err)
	}

	return &config, nil
}

// GetTrunk returns the trunk branch name, or "main" as default
func GetTrunk(repoRoot string) (string, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return "", err
	}

	if config.Trunk != nil && *config.Trunk != "" {
		return *config.Trunk, nil
	}

	// Default to "main"
	return "main", nil
}
