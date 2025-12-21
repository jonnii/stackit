package cli_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestUntrackCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("untrack current branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a tracked branch
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Untrack current branch (a)
		cmd = exec.Command(binaryPath, "untrack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "untrack command failed: %s", string(output))
		require.Contains(t, string(output), "Stopped tracking a")

		// Verify branch is no longer tracked
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "has no parent (untracked branch)")
	})

	t.Run("untrack specified branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create tracked branches
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "create", "b", "-m", "Add b")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Untrack branch a while on branch b
		cmd = exec.Command(binaryPath, "untrack", "a", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "untrack command failed: %s", string(output))
		require.Contains(t, string(output), "Stopped tracking a")
		require.Contains(t, string(output), "Stopped tracking b")

		// Verify branch a is no longer tracked
		// Let's use checkout instead

		cmd = exec.Command(binaryPath, "checkout", "a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "has no parent (untracked branch)")

		// Verify branch b is also no longer tracked (it was a child of a)
		cmd = exec.Command(binaryPath, "checkout", "b")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "has no parent (untracked branch)")
	})

	t.Run("untrack fails for untracked branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch
		err = scene.Repo.CreateAndCheckoutBranch("untracked")
		require.NoError(t, err)

		// Try to untrack it
		cmd = exec.Command(binaryPath, "untrack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "branch untracked is not tracked")
	})
}
