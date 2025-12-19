package cli_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestDownCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("down moves to parent branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b
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

		// Now we're on branch 'b', run down command
		cmd = exec.Command(binaryPath, "down")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down command failed: %s", string(output))

		// Should be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("down with steps moves multiple levels", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b -> c
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

		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'c', run down 2
		cmd = exec.Command(binaryPath, "down", "2")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down 2 command failed: %s", string(output))

		// Should be on branch 'a' (moved 2 levels: c -> b -> a)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})

	t.Run("down with --steps flag", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b -> c
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

		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'c', run down -n 2
		cmd = exec.Command(binaryPath, "down", "-n", "2")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down -n 2 command failed: %s", string(output))

		// Should be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})

	t.Run("down from trunk shows already at trunk", func(t *testing.T) {
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

		// Run down command from trunk
		cmd = exec.Command(binaryPath, "down")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down command failed: %s", string(output))

		// Should still be on main
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
		require.Contains(t, string(output), "trunk")
	})

	t.Run("down moves to trunk when on first branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a branch: main -> a
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Run down command from 'a'
		cmd = exec.Command(binaryPath, "down")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down command failed: %s", string(output))

		// Should be on trunk (main)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("down stops early when not enough parents", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b
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

		// Now we're on branch 'b', run down 10 (more than available)
		cmd = exec.Command(binaryPath, "down", "10")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "down 10 command failed: %s", string(output))

		// Should be on trunk (main) - stopped at the bottom
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
	})

	t.Run("down with invalid steps argument", func(t *testing.T) {
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

		// Run down with invalid argument
		cmd = exec.Command(binaryPath, "down", "abc")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "down with invalid argument should fail")
		require.Contains(t, string(output), "invalid")
	})

	t.Run("down with zero steps fails", func(t *testing.T) {
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

		// Run down with 0 steps
		cmd = exec.Command(binaryPath, "down", "0")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "down with 0 steps should fail")
		require.Contains(t, string(output), "at least 1")
	})
}
