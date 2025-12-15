package cli_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

func TestBottomCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("bottom from middle of stack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b -> c using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'c', run bottom command
		cmd = exec.Command(binaryPath, "bottom")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "bottom command failed: %s", string(output))

		// Should be on branch 'a' (first branch from trunk)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})

	t.Run("bottom from first branch from trunk", func(t *testing.T) {
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
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Run bottom from 'a' (already at bottom)
		cmd = exec.Command(binaryPath, "bottom")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "bottom command failed: %s", string(output))

		// Should still be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Already at the bottom most")
	})

	t.Run("bottom from trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Run bottom from trunk
		cmd := exec.Command(binaryPath, "bottom")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "bottom command failed: %s", string(output))

		// Should still be on main
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)
	})

	t.Run("bottom with single branch stack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a single branch using create command
		err = scene.Repo.CreateChangeAndCommit("feature", "feature")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Run bottom from feature
		cmd = exec.Command(binaryPath, "bottom")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "bottom command failed: %s", string(output))

		// Should be on feature (it's the first branch from trunk)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})
}

func TestTopCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("top from middle of stack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a stack: main -> a -> b -> c using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Now we're on branch 'c', go back to 'a' and run top command
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "top")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "top command failed: %s", string(output))

		// Should be on branch 'c' (tip of stack)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "c", currentBranch)
	})

	t.Run("top from tip of stack", func(t *testing.T) {
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
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Run top from 'a' (already at top)
		cmd = exec.Command(binaryPath, "top")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "top command failed: %s", string(output))

		// Should still be on branch 'a'
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
		require.Contains(t, string(output), "Already at the top most")
	})

	t.Run("top from trunk", func(t *testing.T) {
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
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Go back to main and run top from trunk
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "top")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "top command failed: %s", string(output))

		// Should be on branch 'a' (tip)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "a", currentBranch)
	})

	t.Run("top with single branch stack", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create a single branch using create command
		err = scene.Repo.CreateChangeAndCommit("feature", "feature")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "feature", "-m", "feature")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Run top from feature
		cmd = exec.Command(binaryPath, "top")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "top command failed: %s", string(output))

		// Should be on feature (it's the tip)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("top with multiple children fails in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create branch 'a' using create command
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Create branch 'b' from 'a'
		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Go back to 'a' and create branch 'c' from 'a'
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		err = cmd.Run()
		require.NoError(t, err)

		// Run top from 'a' in non-interactive mode (no stdin)
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "top")
		cmd.Dir = scene.Dir
		cmd.Env = append(os.Environ(), "STACKIT_DISABLE_TELEMETRY=1")
		// Redirect stdin to /dev/null to simulate non-interactive mode
		nullFile, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
		require.NoError(t, err)
		defer nullFile.Close()
		cmd.Stdin = nullFile

		output, err := cmd.CombinedOutput()
		require.Error(t, err, "top command should fail in non-interactive mode with multiple children")
		// The error message may vary, but should indicate failure and list the branches
		require.Contains(t, string(output), "Multiple branches found")
		require.Contains(t, string(output), "b")
		require.Contains(t, string(output), "c")
		require.Contains(t, string(output), "Error")
	})
}

