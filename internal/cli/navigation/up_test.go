package navigation_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/cli/testhelper"
	"stackit.dev/stackit/testhelpers"
)

func TestUpCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelper.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelper.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("up moves to child branch", func(t *testing.T) {
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

		// Move back to 'a'
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)

		// Now we're on branch 'a', run up command
		cmd = exec.Command(binaryPath, "up")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "up command failed: %s", string(output))

		// Should be on branch 'b'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "b", currentBranch)
		require.Contains(t, string(output), "Checked out")
	})

	t.Run("up with steps moves multiple levels", func(t *testing.T) {
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

		// Move back to 'main'
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Now we're on 'main', run up 3
		cmd = exec.Command(binaryPath, "up", "3")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "up 3 command failed: %s", string(output))

		// Should be on branch 'c'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "c", currentBranch)
	})

	t.Run("up with --to flag disambiguates", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create branches:
		// main -> a -> b
		//      -> c -> d

		// Create a
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Create b
		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create c
		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Create d
		err = scene.Repo.CreateChangeAndCommit("d", "d")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "d", "-m", "d")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Run up --to d (main has a and c as children, d is descendant of c)
		cmd = exec.Command(binaryPath, "up", "--to", "d")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "up --to d failed: %s", string(output))

		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "c", currentBranch)
		require.Contains(t, string(output), "⮑  c")

		// Back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Run up --to b (should go main -> a)
		cmd = exec.Command(binaryPath, "up", "--to", "b")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "up --to b failed: %s", string(output))

		currentBranch, err = scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "⮑  a")

		// Back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Run up --to nonexistent (ambiguous from main since it has children a and c)
		cmd = exec.Command(binaryPath, "up", "--to", "nonexistent")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_TEST_NO_INTERACTIVE=1")
		output, err = cmd.CombinedOutput()
		require.Error(t, err, "up --to nonexistent should fail in non-interactive mode")
		require.Contains(t, string(output), "is not a descendant")
	})

	t.Run("up fails in non-interactive mode when ambiguous", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// main -> a
		//      -> b
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Run up in non-interactive mode
		cmd = exec.Command(binaryPath, "up")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_TEST_NO_INTERACTIVE=1")
		output, err := cmd.CombinedOutput()
		require.Error(t, err)
		require.Contains(t, string(output), "multiple children found")
	})
}
