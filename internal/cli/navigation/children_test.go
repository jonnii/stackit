package navigation_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/testhelper"
	"stackit.dev/stackit/testhelpers"
)

func TestChildrenCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelper.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelper.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("children shows child branches", func(t *testing.T) {
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

		// Go back to 'a' and run children command
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "children")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "children command failed: %s", string(output))

		// Should output 'b' as the child
		require.Equal(t, "b", strings.TrimSpace(string(output)))
	})

	t.Run("children shows multiple children", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Create branch 'a' from main
		err = scene.Repo.CreateChangeAndCommit("a", "a")
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "a", "-m", "a")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Create branch 'b' from 'a'
		err = scene.Repo.CreateChangeAndCommit("b", "b")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "b")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Go back to 'a' and create branch 'c' from 'a'
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("c", "c")
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "c", "-m", "c")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Go back to 'a' and run children command
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "children")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "children command failed: %s", string(output))

		// Should output both 'b' and 'c' as children
		outputStr := string(output)
		require.Contains(t, outputStr, "b")
		require.Contains(t, outputStr, "c")
	})

	t.Run("children shows no children message at tip", func(t *testing.T) {
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

		// Run children command from 'a' (tip, no children)
		cmd = exec.Command(binaryPath, "children")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "children command failed: %s", string(output))

		// Should indicate no children
		require.Contains(t, string(output), "no children")
	})

	t.Run("children shows trunk's children", func(t *testing.T) {
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

		// Go back to main and run children command
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "children")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "children command failed: %s", string(output))

		// Should output 'a' as the child
		require.Equal(t, "a", strings.TrimSpace(string(output)))
	})
}
