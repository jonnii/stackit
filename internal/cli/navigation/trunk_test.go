package navigation_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestTrunkCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("trunk shows primary trunk", func(t *testing.T) {
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

		// Run trunk command
		cmd = exec.Command(binaryPath, "trunk")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "trunk command failed: %s", string(output))

		// Should output 'main' as the trunk
		require.Equal(t, "main", strings.TrimSpace(string(output)))
	})

	t.Run("trunk --all shows all configured trunks", func(t *testing.T) {
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

		// Run trunk --all command
		cmd = exec.Command(binaryPath, "trunk", "--all")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "trunk --all command failed: %s", string(output))

		// Should show main as primary
		require.Contains(t, string(output), "main (primary)")
	})

	t.Run("trunk --add adds additional trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit first (while on main)
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		err = cmd.Run()
		require.NoError(t, err)

		// Create a develop branch
		err = scene.Repo.CreateAndCheckoutBranch("develop")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Add develop as an additional trunk
		cmd = exec.Command(binaryPath, "trunk", "--add", "develop")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "trunk --add command failed: %s", string(output))
		require.Contains(t, string(output), "Added")
		require.Contains(t, string(output), "develop")

		// Verify it shows in --all
		cmd = exec.Command(binaryPath, "trunk", "--all")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "trunk --all command failed: %s", string(output))
		require.Contains(t, string(output), "main (primary)")
		require.Contains(t, string(output), "develop")
	})

	t.Run("trunk --add fails for non-existent branch", func(t *testing.T) {
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

		// Try to add non-existent branch as trunk
		cmd = exec.Command(binaryPath, "trunk", "--add", "nonexistent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "trunk --add should fail for non-existent branch")
		require.Contains(t, string(output), "does not exist")
	})

	t.Run("trunk --add fails for already configured trunk", func(t *testing.T) {
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

		// Try to add main (already primary trunk)
		cmd = exec.Command(binaryPath, "trunk", "--add", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "trunk --add should fail for already configured trunk")
		require.Contains(t, string(output), "already")
	})

	t.Run("trunk shows trunk when on a stacked branch", func(t *testing.T) {
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

		// Run trunk command from 'b'
		cmd = exec.Command(binaryPath, "trunk")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "trunk command failed: %s", string(output))

		// Should output 'main' as the trunk
		require.Equal(t, "main", strings.TrimSpace(string(output)))
	})
}
