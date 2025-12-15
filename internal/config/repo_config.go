// Package config provides repository configuration management,
// including reading and writing stackit configuration files.
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

// IsInitialized checks if Stackit has been initialized
func IsInitialized(repoRoot string) bool {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return false
	}
	return config.Trunk != nil && *config.Trunk != ""
}

// SetTrunk updates the trunk branch in the config
func SetTrunk(repoRoot string, trunkName string) error {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	config.Trunk = &trunkName
	if config.IsGithubIntegrationEnabled == nil {
		enabled := false
		config.IsGithubIntegrationEnabled = &enabled
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0644)
}
