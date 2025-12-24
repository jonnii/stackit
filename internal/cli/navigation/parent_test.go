package navigation_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/cli/testhelper"
	"stackit.dev/stackit/testhelpers"
)

func TestParentCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelper.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelper.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("parent shows parent branch", func(t *testing.T) {
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

		// Run parent command from 'b'
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))

		// Should output 'a' as the parent
		require.Equal(t, "a", strings.TrimSpace(string(output)))
	})

	t.Run("parent shows trunk as parent when on first branch", func(t *testing.T) {
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

		// Run parent command from 'a'
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))

		// Should output 'main' as the parent
		require.Equal(t, "main", strings.TrimSpace(string(output)))
	})

	t.Run("parent on trunk shows no parent message", func(t *testing.T) {
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

		// Run parent command from 'main'
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))

		// Should indicate trunk has no parent
		require.Contains(t, string(output), "trunk")
		require.Contains(t, string(output), "no parent")
	})
}
