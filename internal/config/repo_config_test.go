package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestConfigSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("returns true when config does not exist", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns true when config exists but submit.footer is not set", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file without submit.footer
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		config := &RepoConfig{
			Trunk: stringPtr("main"),
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns true when config has submit.footer set to true", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file with submit.footer = true
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		enabled := true
		config := &RepoConfig{
			Trunk:        stringPtr("main"),
			SubmitFooter: &enabled,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg.SubmitFooter())
	})

	t.Run("returns false when config has submit.footer set to false", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create config file with submit.footer = false
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		enabled := false
		config := &RepoConfig{
			Trunk:        stringPtr("main"),
			SubmitFooter: &enabled,
		}
		configJSON, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg.SubmitFooter())
	})
}

func TestConfigSetSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("sets submit.footer to true", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(true)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.True(t, *config.SubmitFooter)

		// Verify Config.SubmitFooter returns true
		cfg2, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.True(t, cfg2.SubmitFooter())
	})

	t.Run("sets submit.footer to false", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(false)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)

		// Verify Config.SubmitFooter returns false
		cfg2, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		require.False(t, cfg2.SubmitFooter())
	})

	t.Run("updates existing config without overwriting other fields", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial config with trunk
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		initialConfig := &RepoConfig{
			Trunk: stringPtr("main"),
		}
		configJSON, err := json.MarshalIndent(initialConfig, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(configPath, configJSON, 0600)
		require.NoError(t, err)

		// Set submit.footer
		cfg, err := LoadConfig(scene.Dir)
		require.NoError(t, err)
		cfg.SetSubmitFooter(false)
		err = cfg.Save()
		require.NoError(t, err)

		// Verify both fields are present
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.Trunk)
		require.Equal(t, "main", *config.Trunk)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
