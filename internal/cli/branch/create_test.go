package branch_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
)

func TestCreateCommand(t *testing.T) {
	t.Parallel()
	// Build the stackit binary first
	binaryPath := getStackitBinary(t)

	t.Run("create branch with name", func(t *testing.T) {
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
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
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

		// Try to create a branch (should fail)
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "create should fail when not on a branch")
		require.Contains(t, string(output), "not on a branch")
	})

	t.Run("create fails when branch already exists", func(t *testing.T) {
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

	t.Run("create fails when no name or message provided", func(t *testing.T) {
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

		// Try to create a branch without name or message
		cmd = exec.Command(binaryPath, "create")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.Error(t, err, "create should fail without name or message")
		require.Contains(t, string(output), "must specify either a branch name or commit message")
	})

	t.Run("create with --insert flag moves single child", func(t *testing.T) {
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

		// Create first child branch
		err = scene.Repo.CreateChange("child1 content", "child1", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "child1", "-m", "Add child1")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "create child1 failed: %s", string(output))

		// Go back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create a new branch with --insert that should be inserted between main and child1
		err = scene.Repo.CreateChange("inserted content", "inserted", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "inserted", "--insert", "-m", "Insert branch")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "create with --insert failed: %s", string(output))

		// Verify: inserted branch should have main as parent
		cmd = exec.Command(binaryPath, "info", "inserted")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info inserted failed: %s", string(output))
		require.Contains(t, string(output), "Parent")
		require.Contains(t, string(output), "main")

		// Verify: child1 should now have inserted as parent
		cmd = exec.Command(binaryPath, "info", "child1")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info child1 failed: %s", string(output))
		require.Contains(t, string(output), "Parent")
		require.Contains(t, string(output), "inserted")
	})

	t.Run("create with --insert flag moves multiple children", func(t *testing.T) {
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

		// Create first child branch from main
		err = scene.Repo.CreateChange("child1 content", "child1", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "child1", "-m", "Add child1")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "create child1 failed: %s", string(output))

		// Go back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create second child branch from main
		err = scene.Repo.CreateChange("child2 content", "child2", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "child2", "-m", "Add child2")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "create child2 failed: %s", string(output))

		// Go back to main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create a new branch with --insert that should be inserted between main and both children
		// Note: In non-interactive mode, all children are moved
		err = scene.Repo.CreateChange("inserted content", "inserted", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "inserted", "--insert", "-m", "Insert branch")
		cmd.Dir = scene.Dir
		cmd.Env = append(cmd.Environ(), "STACKIT_NON_INTERACTIVE=1")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "create with --insert failed: %s", string(output))

		// Verify: inserted branch should have main as parent and both children as children
		cmd = exec.Command(binaryPath, "info", "inserted")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info inserted failed: %s", string(output))
		require.Contains(t, string(output), "Parent")
		require.Contains(t, string(output), "main")
		require.Contains(t, string(output), "Children")
		require.Contains(t, string(output), "child1")
		require.Contains(t, string(output), "child2")

		// Verify: child1 should now have inserted as parent
		cmd = exec.Command(binaryPath, "info", "child1")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info child1 failed: %s", string(output))
		require.Contains(t, string(output), "inserted")

		// Verify: child2 should now have inserted as parent
		cmd = exec.Command(binaryPath, "info", "child2")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "info child2 failed: %s", string(output))
		require.Contains(t, string(output), "inserted")
	})

	t.Run("create auto-initializes when not initialized", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Do NOT initialize stackit - let create auto-initialize

		// Create a change and stage it
		err := scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch (should auto-initialize)
		cmd := exec.Command(binaryPath, "create", "feature", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create with auto-init failed: %s", string(output))
		// Note: The auto-init message may or may not appear in combined output depending
		// on timing and buffering. The key test is that the command succeeds and the
		// branch is created.

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify stackit is now initialized by running a command that requires init
		// The log command would fail if not initialized, so success here proves auto-init worked
		cmd = exec.Command(binaryPath, "log", "--stack")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "log command failed after auto-init: %s", string(output))
		require.Contains(t, string(output), "feature")
	})

	t.Run("create uses branch name pattern from config", func(t *testing.T) {
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

		// Set branch name pattern to just {message} for deterministic testing
		cmd = exec.Command(binaryPath, "config", "set", "branch.pattern", "{message}")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch from commit message
		cmd = exec.Command(binaryPath, "create", "-m", "Add new feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created with expected name (just the message, no prefix)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "Add-new-feature", currentBranch)
	})

	t.Run("create uses default pattern when none configured", func(t *testing.T) {
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

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch from commit message (should use default pattern)
		cmd = exec.Command(binaryPath, "create", "-m", "Add new feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create command failed: %s", string(output))

		// Verify branch was created with default pattern (username/date/message)
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		// Should contain username, date, and message parts
		require.Contains(t, currentBranch, "Add-new-feature")
		// Should have slashes (from pattern)
		require.Contains(t, currentBranch, "/")
		// Should have Test User (from git config in test setup)
		require.Contains(t, currentBranch, "Test-User")
	})

	t.Run("config get returns branch name pattern", func(t *testing.T) {
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

		// Set a custom pattern
		cmd = exec.Command(binaryPath, "config", "set", "branch.pattern", "{username}/{date}/{message}")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Get the pattern back
		cmd = exec.Command(binaryPath, "config", "get", "branch.pattern")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Equal(t, "{username}/{date}/{message}\n", string(output))
	})

	t.Run("config set rejects pattern without message placeholder", func(t *testing.T) {
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

		// Try to set a pattern without {message} (should fail)
		cmd = exec.Command(binaryPath, "config", "set", "branch.pattern", "{username}/{date}")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "config set should fail without {message} placeholder")
		require.Contains(t, string(output), "must contain {message}")
	})

	t.Run("create with branch name works without message", func(t *testing.T) {
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

		// Create a new branch with branch name (no message needed)
		cmd = exec.Command(binaryPath, "create", "feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create with branch name should work")
		require.Contains(t, string(output), "No staged changes")

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)
	})

	t.Run("create with --scope sets explicit scope on new branch", func(t *testing.T) {
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

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch with scope
		cmd = exec.Command(binaryPath, "create", "feature", "--scope", "PROJ-123", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create with scope failed: %s", string(output))

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify scope was set
		cmd = exec.Command(binaryPath, "scope", "--show")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "scope show failed: %s", string(output))
		require.Contains(t, string(output), "PROJ-123")
	})

	t.Run("create with --scope inherits scope when not provided", func(t *testing.T) {
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

		// Create a parent branch with scope
		err = scene.Repo.CreateChange("parent content", "test", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "parent", "--scope", "PROJ-456", "-m", "Add parent")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch without explicit scope (should inherit from parent)
		cmd = exec.Command(binaryPath, "create", "feature", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create without scope failed: %s", string(output))

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify scope was inherited
		cmd = exec.Command(binaryPath, "scope", "--show")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "scope show failed: %s", string(output))
		require.Contains(t, string(output), "PROJ-456")
		require.Contains(t, string(output), "inherits scope")
	})

	t.Run("create with --scope overrides inherited scope", func(t *testing.T) {
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

		// Create a parent branch with scope
		err = scene.Repo.CreateChange("parent content", "test", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "parent", "--scope", "PROJ-789", "-m", "Add parent")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch with different scope (should override inheritance)
		cmd = exec.Command(binaryPath, "create", "feature", "--scope", "PROJ-999", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create with scope override failed: %s", string(output))

		// Verify branch was created
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature", currentBranch)

		// Verify explicit scope was set (not inherited)
		cmd = exec.Command(binaryPath, "scope", "--show")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "scope show failed: %s", string(output))
		require.Contains(t, string(output), "PROJ-999")
		require.Contains(t, string(output), "explicit scope")
	})

	t.Run("create with --scope uses scope in branch name pattern", func(t *testing.T) {
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

		// Set branch pattern to include scope
		cmd = exec.Command(binaryPath, "config", "set", "branch.pattern", "{scope}/{message}")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create a change and stage it
		err = scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		// Create a new branch with scope (name should include scope)
		cmd = exec.Command(binaryPath, "create", "--scope", "PROJ-111", "-m", "Add feature")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()

		require.NoError(t, err, "create with scope in pattern failed: %s", string(output))

		// Verify branch was created with scope in name
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Contains(t, currentBranch, "PROJ-111")
		require.Contains(t, currentBranch, "Add-feature")
	})
}
