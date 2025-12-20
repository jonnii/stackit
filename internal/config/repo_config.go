// Package config provides repository configuration management,
// including reading and writing stackit configuration files.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RepoConfig represents the repository configuration
type RepoConfig struct {
	Trunk                      *string  `json:"trunk,omitempty"`
	Trunks                     []string `json:"trunks,omitempty"`
	IsGithubIntegrationEnabled *bool    `json:"isGithubIntegrationEnabled,omitempty"`
	BranchNamePattern          *string  `json:"branchNamePattern,omitempty"`
	CreateAI                   *bool    `json:"create.ai,omitempty"`
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

// GetTrunk returns the primary trunk branch name, or "main" as default
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

// GetAllTrunks returns all configured trunk branches
// Returns at least the primary trunk
func GetAllTrunks(repoRoot string) ([]string, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	// Start with the primary trunk
	var trunks []string
	if config.Trunk != nil && *config.Trunk != "" {
		trunks = append(trunks, *config.Trunk)
	}

	// Add additional trunks (avoiding duplicates)
	for _, t := range config.Trunks {
		if !contains(trunks, t) {
			trunks = append(trunks, t)
		}
	}

	// Default to "main" if no trunks configured
	if len(trunks) == 0 {
		return []string{"main"}, nil
	}

	return trunks, nil
}

// IsTrunk checks if a branch is configured as a trunk
func IsTrunk(repoRoot string, branchName string) (bool, error) {
	trunks, err := GetAllTrunks(repoRoot)
	if err != nil {
		return false, err
	}

	return contains(trunks, branchName), nil
}

// AddTrunk adds an additional trunk branch to the config
func AddTrunk(repoRoot string, trunkName string) error {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	// Check if already a trunk
	if config.Trunk != nil && *config.Trunk == trunkName {
		return fmt.Errorf("'%s' is already the primary trunk", trunkName)
	}
	if contains(config.Trunks, trunkName) {
		return fmt.Errorf("'%s' is already configured as a trunk", trunkName)
	}

	// Add to trunks list
	config.Trunks = append(config.Trunks, trunkName)

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
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

	return os.WriteFile(configPath, configJSON, 0600)
}

// GetBranchNamePattern returns the branch name pattern from config, or default if not set
func GetBranchNamePattern(repoRoot string) (string, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return "", err
	}

	if config.BranchNamePattern != nil {
		return *config.BranchNamePattern, nil
	}

	// Default pattern
	return "{username}/{date}/{message}", nil
}

// SetBranchNamePattern updates the branch name pattern in the config
func SetBranchNamePattern(repoRoot string, pattern string) error {
	// Validate that pattern contains {message} placeholder
	if !strings.Contains(pattern, "{message}") {
		return fmt.Errorf("branch name pattern must contain {message} placeholder")
	}

	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	config.BranchNamePattern = &pattern

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}

// GetCreateAI returns whether AI-powered PR description is enabled, or false by default
func GetCreateAI(repoRoot string) (bool, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return false, err
	}

	if config.CreateAI != nil {
		return *config.CreateAI, nil
	}

	// Default to false
	return false, nil
}

// SetCreateAI updates the create.ai configuration
func SetCreateAI(repoRoot string, enabled bool) error {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	// Validate repo root exists
	if _, err := os.Stat(repoRoot); err != nil {
		return fmt.Errorf("repository root does not exist: %w", err)
	}

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	config.CreateAI = &enabled

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}
