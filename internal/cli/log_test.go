package cli_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

func TestLogCommand(t *testing.T) {
	// Build the stackit binary first
	binaryPath := getStackitBinary(t)

	t.Run("log in empty repo", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Run log command
		cmd := exec.Command(binaryPath, "log")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		// Should succeed and show trunk branch
		require.NoError(t, err, "log command failed: %s", string(output))
		require.Contains(t, string(output), "main")
	})

	t.Run("log with branches", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature commit", "feature")
		require.NoError(t, err)

		// Checkout main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Run log command with --show-untracked to see untracked branches
		cmd := exec.Command(binaryPath, "log", "--show-untracked")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "log command failed: %s", string(output))
		require.Contains(t, string(output), "main")
		require.Contains(t, string(output), "feature")
	})

	t.Run("log with --reverse flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Run log command with reverse
		cmd := exec.Command(binaryPath, "log", "--reverse")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "log command failed: %s", string(output))
		require.Contains(t, string(output), "main")
	})

	t.Run("log with --stack flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create and checkout a branch
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)

		// Run log command with stack
		cmd := exec.Command(binaryPath, "log", "--stack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "log command failed: %s", string(output))
		require.Contains(t, string(output), "feature")
	})
}
