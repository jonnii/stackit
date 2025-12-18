package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestConfigCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("config get returns default pattern when not set", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Get branch-name-pattern (should return default)
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "config get command failed: %s", string(output))

		// Should return default pattern
		require.Equal(t, "{username}/{date}/{message}", strings.TrimSpace(string(output)))
	})

	t.Run("config set and get branch-name-pattern", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Set a custom pattern
		pattern := "{username}/{date}/{message}"
		cmd = exec.Command(binaryPath, "config", "set", "branch-name-pattern", pattern)
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "config set command failed: %s", string(output))
		require.Contains(t, string(output), "Set branch-name-pattern to:")

		// Get the pattern back
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "config get command failed: %s", string(output))
		require.Equal(t, pattern, strings.TrimSpace(string(output)))
	})

	t.Run("config set rejects pattern without message placeholder", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Try to set a pattern without {message}
		cmd = exec.Command(binaryPath, "config", "set", "branch-name-pattern", "{username}/{date}")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config set should fail without {message} placeholder")
		require.Contains(t, string(output), "must contain {message}")

		// Verify pattern was not set (should still be default)
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, "{username}/{date}/{message}", strings.TrimSpace(string(output)))
	})

	t.Run("config set accepts pattern with only message placeholder", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Set pattern with only {message}
		pattern := "{message}"
		cmd = exec.Command(binaryPath, "config", "set", "branch-name-pattern", pattern)
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "config set command failed: %s", string(output))

		// Verify it was set
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, pattern, strings.TrimSpace(string(output)))
	})

	t.Run("config get fails for unknown key", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Try to get unknown key
		cmd = exec.Command(binaryPath, "config", "get", "unknown-key")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config get should fail for unknown key")
		require.Contains(t, string(output), "unknown configuration key")
	})

	t.Run("config set fails for unknown key", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Try to set unknown key
		cmd = exec.Command(binaryPath, "config", "set", "unknown-key", "value")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config set should fail for unknown key")
		require.Contains(t, string(output), "unknown configuration key")
	})

	t.Run("config get fails when not in git repository", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Don't initialize git or stackit - just try to run config get
		cmd := exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config get should fail when not in git repository")
		require.Contains(t, string(output), "not a git repository")
	})

	t.Run("config set fails when not in git repository", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Don't initialize git or stackit - just try to run config set
		cmd := exec.Command(binaryPath, "config", "set", "branch-name-pattern", "{message}")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config set should fail when not in git repository")
		require.Contains(t, string(output), "not a git repository")
	})

	t.Run("config set persists pattern across commands", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Set a custom pattern
		pattern := "{username}/dev/{date}/{message}"
		cmd = exec.Command(binaryPath, "config", "set", "branch-name-pattern", pattern)
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Get it back
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, pattern, strings.TrimSpace(string(output)))

		// Set a different pattern
		pattern2 := "{date}/{message}"
		cmd = exec.Command(binaryPath, "config", "set", "branch-name-pattern", pattern2)
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Verify it changed
		cmd = exec.Command(binaryPath, "config", "get", "branch-name-pattern")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, pattern2, strings.TrimSpace(string(output)))
	})
}
