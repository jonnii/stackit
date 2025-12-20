package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestTrackCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("track with --parent flag tracks single branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a tracked branch
		err = scene.Repo.CreateChange("a content", "a", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch from a
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("b content", "b", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("b")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add b")
		require.NoError(t, err)

		// Track branch b with parent a
		cmd = exec.Command(binaryPath, "track", "b", "--parent", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify branch is tracked (check via parent command)
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "a", strings.TrimSpace(string(output)))
	})

	t.Run("track with --parent flag using trunk", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch
		err = scene.Repo.CreateChange("feature content", "feature", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add feature")
		require.NoError(t, err)

		// Track branch with trunk as parent
		cmd = exec.Command(binaryPath, "track", "feature", "--parent", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify branch is tracked (check via parent command)
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "main", strings.TrimSpace(string(output)))
	})

	t.Run("track with --parent fails when parent not tracked", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch
		err = scene.Repo.CreateChange("feature content", "feature", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add feature")
		require.NoError(t, err)

		// Create another untracked branch
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("other content", "other", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("other")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add other")
		require.NoError(t, err)

		// Try to track feature with untracked parent (should fail)
		cmd = exec.Command(binaryPath, "track", "feature", "--parent", "other")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("track should have failed but didn't. Output: %s", string(output))
		}
		t.Logf("Track command output: %s", string(output))
		t.Logf("Track command error: %v", err)
		require.Error(t, err, "track should fail when parent is not tracked")
		require.Contains(t, string(output), "must be tracked", "error output: %s", string(output))
	})

	t.Run("track with --parent fails when branch doesn't exist", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Try to track non-existent branch
		cmd = exec.Command(binaryPath, "track", "nonexistent", "--parent", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "track should fail when branch doesn't exist")
		require.Contains(t, string(output), "reference not found")
	})

	t.Run("track with --force finds most recent tracked ancestor", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a stack: main -> a -> b
		err = scene.Repo.CreateChange("a content", "a", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		err = scene.Repo.CreateChange("b content", "b", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "Add b")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch from b
		err = scene.Repo.CreateChange("c content", "c", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("c")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add c")
		require.NoError(t, err)

		// Track branch c with --force (should find b as most recent ancestor)
		cmd = exec.Command(binaryPath, "track", "c", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify branch is tracked with b as parent
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "b", strings.TrimSpace(string(output)))
	})

	t.Run("track with --force falls back to trunk when no tracked ancestor", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch from main
		err = scene.Repo.CreateChange("feature content", "feature", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add feature")
		require.NoError(t, err)

		// Track branch with --force (should find trunk as parent)
		cmd = exec.Command(binaryPath, "track", "feature", "--force")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify branch is tracked with main as parent
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "main", strings.TrimSpace(string(output)))
	})

	t.Run("track defaults to current branch", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a tracked branch
		err = scene.Repo.CreateChange("a content", "a", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create an untracked branch and checkout
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("b content", "b", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("b")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add b")
		require.NoError(t, err)

		// Track current branch (b) with parent a
		cmd = exec.Command(binaryPath, "track", "--parent", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify branch is tracked
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "a", strings.TrimSpace(string(output)))
	})

	t.Run("track fails when not on branch and no branch specified", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Detach HEAD
		err = scene.Repo.RunGitCommand("checkout", "HEAD~0")
		require.NoError(t, err)

		// Try to track without specifying branch (should fail)
		cmd = exec.Command(binaryPath, "track", "--parent", "main")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "track should fail when not on branch")
		require.Contains(t, string(output), "not on a branch")
	})

	t.Run("track already tracked branch updates parent", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a stack: main -> a -> b
		err = scene.Repo.CreateChange("a content", "a", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		err = scene.Repo.CreateChange("b content", "b", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "b", "-m", "Add b")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create another branch c from b
		err = scene.Repo.CheckoutBranch("b")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("c content", "c", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("c")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add c")
		require.NoError(t, err)

		// Track c with parent a (a is an ancestor of b, and b is an ancestor of c, so a is an ancestor of c)
		cmd = exec.Command(binaryPath, "track", "c", "--parent", "a")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify c has a as parent
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "a", strings.TrimSpace(string(output)))

		// Re-track c with parent b (should update parent)
		cmd = exec.Command(binaryPath, "track", "c", "--parent", "b")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify c now has b as parent
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "b", strings.TrimSpace(string(output)))
	})

	t.Run("track can fix corrupted metadata", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a tracked branch
		err = scene.Repo.CreateChange("a content", "a", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "a", "-m", "Add a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create another branch from a and track it
		err = scene.Repo.CheckoutBranch("a")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("b content", "b", false)
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("b")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("commit", "-m", "Add b")
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "track", "b", "--parent", "a")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Corrupt metadata by deleting the metadata ref
		err = scene.Repo.RunGitCommand("update-ref", "-d", "refs/branch-metadata/b")
		require.NoError(t, err)

		// Verify b is no longer tracked
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "no parent")

		// Re-track b to fix corrupted metadata
		cmd = exec.Command(binaryPath, "track", "b", "--parent", "a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "track command failed: %s", string(output))

		// Verify b is tracked again
		cmd = exec.Command(binaryPath, "parent")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "parent command failed: %s", string(output))
		require.Equal(t, "a", strings.TrimSpace(string(output)))
	})
}
