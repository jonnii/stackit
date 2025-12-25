// Package config provides repository configuration management,
// including reading and writing stackit configuration files.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents a repository configuration with getters and setters
type Config struct {
	repoRoot string
	data     *RepoConfig
}

// LoadConfig creates a new Config instance from a repository root
func LoadConfig(repoRoot string) (*Config, error) {
	data, err := GetRepoConfig(repoRoot)
	if err != nil {
		return nil, err
	}

	return &Config{
		repoRoot: repoRoot,
		data:     data,
	}, nil
}

// Save persists the configuration to disk
func (c *Config) Save() error {
	configPath := filepath.Join(c.repoRoot, ".git", ".stackit_config")

	configJSON, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}

// Trunk returns the primary trunk branch name, or "main" as default
func (c *Config) Trunk() string {
	if c.data.Trunk != nil && *c.data.Trunk != "" {
		return *c.data.Trunk
	}
	return "main"
}

// SetTrunk sets the primary trunk branch name
func (c *Config) SetTrunk(trunkName string) {
	c.data.Trunk = &trunkName
	if c.data.IsGithubIntegrationEnabled == nil {
		enabled := false
		c.data.IsGithubIntegrationEnabled = &enabled
	}
}

// AllTrunks returns all configured trunk branches
func (c *Config) AllTrunks() []string {
	var trunks []string
	if c.data.Trunk != nil && *c.data.Trunk != "" {
		trunks = append(trunks, *c.data.Trunk)
	}

	// Add additional trunks (avoiding duplicates)
	for _, t := range c.data.Trunks {
		if !contains(trunks, t) {
			trunks = append(trunks, t)
		}
	}

	// Default to "main" if no trunks configured
	if len(trunks) == 0 {
		return []string{"main"}
	}

	return trunks
}

// IsTrunk checks if a branch is configured as a trunk
func (c *Config) IsTrunk(branchName string) bool {
	trunks := c.AllTrunks()
	return contains(trunks, branchName)
}

// AddTrunk adds an additional trunk branch to the config
func (c *Config) AddTrunk(trunkName string) error {
	// Check if already a trunk
	if c.data.Trunk != nil && *c.data.Trunk == trunkName {
		return fmt.Errorf("'%s' is already the primary trunk", trunkName)
	}
	if contains(c.data.Trunks, trunkName) {
		return fmt.Errorf("'%s' is already configured as a trunk", trunkName)
	}

	// Add to trunks list
	c.data.Trunks = append(c.data.Trunks, trunkName)
	return nil
}

// IsInitialized checks if Stackit has been initialized
func (c *Config) IsInitialized() bool {
	return c.data.Trunk != nil && *c.data.Trunk != ""
}

// BranchNamePattern returns the branch name pattern from config, or default if not set
func (c *Config) BranchNamePattern() string {
	return c.data.GetBranchPattern().String()
}

// SetBranchNamePattern sets the branch name pattern in the config
func (c *Config) SetBranchNamePattern(pattern string) error {
	// Validate the pattern
	branchPattern, err := NewBranchPattern(pattern)
	if err != nil {
		return err
	}

	patternStr := branchPattern.String()
	c.data.BranchNamePattern = &patternStr
	return nil
}

// SubmitFooter returns whether PR footer is enabled, or true by default
func (c *Config) SubmitFooter() bool {
	if c.data.SubmitFooter != nil {
		return *c.data.SubmitFooter
	}
	return true
}

// SetSubmitFooter sets whether PR footer is enabled
func (c *Config) SetSubmitFooter(enabled bool) {
	c.data.SubmitFooter = &enabled
}

// UndoStackDepth returns the maximum number of undo snapshots to keep, or 10 by default
func (c *Config) UndoStackDepth() int {
	if c.data.UndoStackDepth != nil {
		return *c.data.UndoStackDepth
	}
	return 10
}

// SetUndoStackDepth sets the maximum number of undo snapshots to keep
func (c *Config) SetUndoStackDepth(depth int) {
	c.data.UndoStackDepth = &depth
}

// RepoConfig represents the repository configuration
type RepoConfig struct {
	Trunk                      *string  `json:"trunk,omitempty"`
	Trunks                     []string `json:"trunks,omitempty"`
	IsGithubIntegrationEnabled *bool    `json:"isGithubIntegrationEnabled,omitempty"`
	BranchNamePattern          *string  `json:"branchNamePattern,omitempty"`
	SubmitFooter               *bool    `json:"submit.footer,omitempty"`
	UndoStackDepth             *int     `json:"undo.stackDepth,omitempty"`
}

// GetBranchPattern returns the branch name pattern as a BranchPattern type
// Always returns a valid pattern (default if not set or invalid)
func (c *RepoConfig) GetBranchPattern() BranchPattern {
	if c.BranchNamePattern != nil && *c.BranchNamePattern != "" {
		pattern, err := NewBranchPattern(*c.BranchNamePattern)
		if err != nil {
			// If invalid, return default
			return DefaultBranchPattern
		}
		return pattern.WithDefault()
	}
	return DefaultBranchPattern
}

// GetRepoConfig reads the repository configuration
func GetRepoConfig(repoRoot string) (*RepoConfig, error) {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist - return default
		return &RepoConfig{}, nil //nolint:nilerr
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

	return config.GetBranchPattern().String(), nil
}

// GetBranchPattern returns the branch name pattern as a BranchPattern type
// Always returns a valid pattern (default if not set or invalid)
func GetBranchPattern(repoRoot string) (BranchPattern, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return DefaultBranchPattern, err
	}

	return config.GetBranchPattern().WithDefault(), nil
}

// SetBranchNamePattern updates the branch name pattern in the config
func SetBranchNamePattern(repoRoot string, pattern string) error {
	// Validate the pattern
	branchPattern, err := NewBranchPattern(pattern)
	if err != nil {
		return err
	}

	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	patternStr := branchPattern.String()
	config.BranchNamePattern = &patternStr

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}

// GetSubmitFooter returns whether PR footer is enabled, or true by default
func GetSubmitFooter(repoRoot string) (bool, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return true, err
	}

	if config.SubmitFooter != nil {
		return *config.SubmitFooter, nil
	}

	// Default to true
	return true, nil
}

// SetSubmitFooter updates the submit.footer configuration
func SetSubmitFooter(repoRoot string, enabled bool) error {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	// Validate repo root exists
	if _, err := os.Stat(repoRoot); err != nil {
		return fmt.Errorf("repository root does not exist: %w", err)
	}

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	config.SubmitFooter = &enabled

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}

// GetUndoStackDepth returns the maximum number of undo snapshots to keep, or 10 by default
func GetUndoStackDepth(repoRoot string) (int, error) {
	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		return 10, err
	}

	if config.UndoStackDepth != nil {
		return *config.UndoStackDepth, nil
	}

	// Default to 10
	return 10, nil
}

// SetUndoStackDepth updates the undo.stackDepth configuration
func SetUndoStackDepth(repoRoot string, depth int) error {
	configPath := filepath.Join(repoRoot, ".git", ".stackit_config")

	// Validate repo root exists
	if _, err := os.Stat(repoRoot); err != nil {
		return fmt.Errorf("repository root does not exist: %w", err)
	}

	config, err := GetRepoConfig(repoRoot)
	if err != nil {
		config = &RepoConfig{}
	}

	config.UndoStackDepth = &depth

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(configPath, configJSON, 0600)
}
