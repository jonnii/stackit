package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestGetSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("returns true when config does not exist", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		enabled, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.True(t, enabled)
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

		enabled, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.True(t, enabled)
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

		result, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.True(t, result)
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

		result, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.False(t, result)
	})
}

func TestSetSubmitFooter(t *testing.T) {
	t.Parallel()

	t.Run("sets submit.footer to true", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		err := SetSubmitFooter(scene.Dir, true)
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.True(t, *config.SubmitFooter)

		// Verify GetSubmitFooter returns true
		enabled, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.True(t, enabled)
	})

	t.Run("sets submit.footer to false", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		err := SetSubmitFooter(scene.Dir, false)
		require.NoError(t, err)

		// Verify config was written
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)

		// Verify GetSubmitFooter returns false
		enabled, err := GetSubmitFooter(scene.Dir)
		require.NoError(t, err)
		require.False(t, enabled)
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
		err = SetSubmitFooter(scene.Dir, false)
		require.NoError(t, err)

		// Verify both fields are present
		config, err := GetRepoConfig(scene.Dir)
		require.NoError(t, err)
		require.NotNil(t, config.Trunk)
		require.Equal(t, "main", *config.Trunk)
		require.NotNil(t, config.SubmitFooter)
		require.False(t, *config.SubmitFooter)
	})

	t.Run("fails when repo root does not exist", func(t *testing.T) {
		t.Parallel()
		nonExistentDir := "/non/existent/directory"

		err := SetSubmitFooter(nonExistentDir, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "repository root does not exist")
	})
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
