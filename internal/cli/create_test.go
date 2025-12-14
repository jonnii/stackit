package cli_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

func TestCreateCommand(t *testing.T) {
	// Build the stackit binary first
	binaryPath := buildStackitBinary(t)

	t.Run("create branch with name", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a new branch
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("create branch with staged changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch with commit message
		cmd = exec.Command(binaryPath, "create", "feature", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created and has commit
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify commit was created
		commits, err := scene.Repo.ListCurrentBranchCommitMessages()
		require.NoError(t, err)
		require.Contains(t, commits, "Add feature")
	})

	t.Run("create empty branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a new branch with no changes
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))
		require.Contains(t, string(output), "No staged changes")

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("create branch with --all flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create unstaged changes
		err = scene.Repo.CreateChange("new content", "test", true)
		require.NoError(t, err)

		// Create a new branch with --all flag
		cmd = exec.Command(binaryPath, "create", "feature", "--all", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created and has commit
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify commit was created
		commits, err := scene.Repo.ListCurrentBranchCommitMessages()
		require.NoError(t, err)
		require.Contains(t, commits, "Add feature")
	})

	t.Run("create branch from commit message", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch from commit message (no branch name provided)
		cmd = exec.Command(binaryPath, "create", "-m", "Add new feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created (name should be generated from message)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.NotEqual(t, "main", currentBranch)
		require.Contains(t, currentBranch, "Add-new-feature")
	})

	t.Run("create branch with --update flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit with a file
			if err := s.Repo.CreateChange("initial", "test", false); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Modify tracked file (unstaged)
		err = scene.Repo.CreateChange("modified content", "test", true)
		require.NoError(t, err)

		// Create a new branch with --update flag
		cmd = exec.Command(binaryPath, "create", "feature", "--update", "-m", "Update file")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created and has commit
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("create branch tracks parent relationship", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch
		cmd = exec.Command(binaryPath, "create", "feature", "-m", "Add feature")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Verify branch is tracked (check via log command)
		cmd = exec.Command(binaryPath, "log", "--stack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "log command failed: %s", string(output))
		require.Contains(t, string(output), "feature")
		require.Contains(t, string(output), "main")
	})

	t.Run("create fails when not on a branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
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

		// Try to create a branch (should fail)
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "create should fail when not on a branch")
		require.Contains(t, string(output), "not on a branch")
	})

	t.Run("create fails when branch already exists", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// Create branch manually
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Try to create the same branch (should fail)
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "create should fail when branch exists")
		require.Contains(t, string(output), "already exists")
	})
}
