package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/testhelpers"
)

func TestInitCommand(t *testing.T) {
	t.Parallel()
	// Build the stackit binary first
	binaryPath := getStackitBinary(t)

	t.Run("can run init", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit (needed for init)
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Remove existing config to test fresh init
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		err = os.Remove(configPath)
		// Ignore error if file doesn't exist

		// Run init command
		cmd := exec.Command(binaryPath, "init", "--trunk", "main", "--no-interactive")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "init command failed: %s", string(output))

		// Verify config was created with correct trunk
		cfg := readRepoConfig(t, scene.Dir)
		require.NotNil(t, cfg.Trunk)
		require.Equal(t, "main", *cfg.Trunk)
	})

	t.Run("errors on invalid trunk when explicitly provided", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Remove existing config
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		os.Remove(configPath)

		// Run init with non-existent branch
		cmd := exec.Command(binaryPath, "init", "--trunk", "random", "--no-interactive")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		// Should fail with error about branch not found
		require.Error(t, err, "init should fail with invalid branch")
		require.Contains(t, string(output), "not found", "error message should mention branch not found")
	})

	t.Run("errors on invalid trunk when cannot infer", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit first
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Rename main branch to main2
			return s.Repo.RunGitCommand("branch", "-m", "main2")
		})

		// Remove existing config
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		os.Remove(configPath)

		// Run init with non-existent branch and no way to infer
		cmd := exec.Command(binaryPath, "init", "--trunk", "random", "--no-interactive")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		// Should fail with error
		require.Error(t, err, "init should fail when trunk cannot be inferred and invalid branch provided")
		require.Contains(t, string(output), "not found", "error message should mention branch not found")
	})

	t.Run("infers trunk when not provided", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Remove existing config
		configPath := filepath.Join(scene.Dir, ".git", ".stackit_config")
		os.Remove(configPath)

		// Run init without specifying trunk (should infer main)
		cmd := exec.Command(binaryPath, "init", "--no-interactive")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "init command failed: %s", string(output))

		// Verify config was created with inferred trunk (main)
		cfg := readRepoConfig(t, scene.Dir)
		require.NotNil(t, cfg.Trunk)
		require.Equal(t, "main", *cfg.Trunk)
	})
}

// readRepoConfig reads and parses the repository config file
func readRepoConfig(t *testing.T, repoDir string) *config.RepoConfig {
	t.Helper()

	configPath := filepath.Join(repoDir, ".git", ".stackit_config")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "failed to read config file")

	var cfg config.RepoConfig
	err = json.Unmarshal(data, &cfg)
	require.NoError(t, err, "failed to parse config JSON")

	return &cfg
}
