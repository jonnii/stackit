package cli_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestCheckoutCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("direct checkout with branch name", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'b', checkout 'a'
		cmd = exec.Command(binaryPath, "checkout", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "checkout command failed: %s", string(output))

		// Should be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("checkout trunk flag", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'a', checkout trunk
		cmd = exec.Command(binaryPath, "checkout", "--trunk")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "checkout --trunk command failed: %s", string(output))

		// Should be on main (trunk)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("checkout trunk flag short form", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'a', checkout trunk with short flag
		cmd = exec.Command(binaryPath, "checkout", "-t")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "checkout -t command failed: %s", string(output))

		// Should be on main (trunk)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
	})

	t.Run("already on branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Try to checkout the branch we're already on
		cmd = exec.Command(binaryPath, "checkout", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "checkout command should succeed even when already on branch")

		// Should still be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Already on")
	})

	t.Run("invalid branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Try to checkout a non-existent branch
		cmd := exec.Command(binaryPath, "checkout", "nonexistent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "checkout should fail for non-existent branch")
		require.Contains(t, string(output), "failed to checkout")
	})

	t.Run("alias co works", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'b', use alias 'co' to checkout 'a'
		cmd = exec.Command(binaryPath, "co", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "co alias command failed: %s", string(output))

		// Should be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("checkout with show-untracked flag", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a tracked branch
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Create an untracked branch (not using stackit create)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("untracked", "untracked")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("untracked-branch")
		require.NoError(t, err)

		// Try to checkout with --show-untracked (but no branch specified - would need interactive)
		// For non-interactive test, just verify the flag is accepted
		cmd = exec.Command(binaryPath, "checkout", "--show-untracked", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		// This should work (flag is accepted, even if not used for direct checkout)
		require.NoError(t, err, "checkout with --show-untracked flag failed: %s", string(output))

		// Should be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})

	t.Run("checkout with stack flag in non-interactive mode should fail", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Try to checkout with --stack flag but no branch (would need interactive)
		// In non-interactive mode, this should fail with a clear error
		cmd = exec.Command(binaryPath, "checkout", "--stack")
		cmd.Dir = scene.Dir
		// Redirect stdin to /dev/null to simulate non-interactive mode
		nullFile, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
		require.NoError(t, err)
		defer nullFile.Close()
		cmd.Stdin = nullFile

		output, err := cmd.CombinedOutput()
		// Should fail in non-interactive mode with a clear error
		require.Error(t, err, "checkout --stack should fail in non-interactive mode")
		require.Contains(t, string(output), "interactive", "error should mention interactive mode")
	})

	t.Run("checkout from trunk to branch and back", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Verify we're on 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)

		// Checkout trunk
		cmd = exec.Command(binaryPath, "checkout", "--trunk")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "checkout --trunk failed: %s", string(output))

		currentBranch, err = scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)

		// Checkout back to 'a'
		cmd = exec.Command(binaryPath, "checkout", "a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "checkout a failed: %s", string(output))

		currentBranch, err = scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})
}
