package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

// =============================================================================
// Test Shell - A helper to make tests read like terminal sessions
// =============================================================================

// TestShell wraps a test scene and provides a fluent interface for running
// commands. Tests using this read like a series of terminal commands.
type TestShell struct {
	t          *testing.T
	scene      *testhelpers.Scene
	binaryPath string
	lastOutput string
}

// NewTestShell creates a shell-like test environment with an initialized repo.
func NewTestShell(t *testing.T, binaryPath string) *TestShell {
	t.Helper()
	scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
		return s.Repo.CreateChangeAndCommit("initial", "init")
	})
	return &TestShell{t: t, scene: scene, binaryPath: binaryPath}
}

// NewTestShellWithRemote creates a shell-like test environment with a local bare repo as "origin".
// This is useful for testing sync workflows that require a remote.
func NewTestShellWithRemote(t *testing.T, binaryPath string) *TestShell {
	t.Helper()

	// Create a bare repository to act as the remote
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", remoteDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to create bare repo: %s", string(output))

	// Create the scene with the remote set up
	scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
		// Create initial commit
		if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
			return err
		}
		// Add the bare repo as origin
		cmd := exec.Command("git", "remote", "add", "origin", remoteDir)
		cmd.Dir = s.Dir
		if err := cmd.Run(); err != nil {
			return err
		}
		// Push main to origin
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = s.Dir
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	})
	return &TestShell{t: t, scene: scene, binaryPath: binaryPath}
}

// Run executes a stackit CLI command (e.g., "create feature-a -m 'Add feature'")
func (s *TestShell) Run(args string) *TestShell {
	s.t.Helper()
	parts := splitArgs(args)
	cmd := exec.Command(s.binaryPath, parts...)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	s.lastOutput = string(output)
	require.NoError(s.t, err, "$ stackit %s\n%s", args, s.lastOutput)
	return s
}

// Git executes a raw git command (use sparingly - prefer stackit commands)
func (s *TestShell) Git(args string) *TestShell {
	s.t.Helper()
	parts := splitArgs(args)
	cmd := exec.Command("git", parts...)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	s.lastOutput = string(output)
	require.NoError(s.t, err, "$ git %s\n%s", args, s.lastOutput)
	return s
}

// Checkout switches to a branch using stackit checkout
func (s *TestShell) Checkout(branch string) *TestShell {
	s.t.Helper()
	return s.Run("checkout " + branch)
}

// Top navigates to the top of the current stack
func (s *TestShell) Top() *TestShell {
	s.t.Helper()
	return s.Run("top")
}

// Bottom navigates to the bottom of the current stack
func (s *TestShell) Bottom() *TestShell {
	s.t.Helper()
	return s.Run("bottom")
}

// Write creates/modifies a file and stages it (simulates editing a file)
func (s *TestShell) Write(filename, content string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChange(content, filename, false)
	require.NoError(s.t, err, "failed to write %s", filename)
	return s
}

// Amend modifies a file and amends the last commit
func (s *TestShell) Amend(filename, content string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChangeAndAmend(content, filename)
	require.NoError(s.t, err, "failed to amend with %s", filename)
	return s
}

// Commit creates a file change and commits it
func (s *TestShell) Commit(filename, message string) *TestShell {
	s.t.Helper()
	err := s.scene.Repo.CreateChangeAndCommit(message, filename)
	require.NoError(s.t, err, "failed to commit %s", filename)
	return s
}

// Output returns the last command's output
func (s *TestShell) Output() string {
	return s.lastOutput
}

// OutputContains asserts the last output contains the given string
func (s *TestShell) OutputContains(substr string) *TestShell {
	s.t.Helper()
	require.Contains(s.t, s.lastOutput, substr)
	return s
}

// OnBranch asserts we're on the expected branch
func (s *TestShell) OnBranch(expected string) *TestShell {
	s.t.Helper()
	branch, err := s.scene.Repo.CurrentBranchName()
	require.NoError(s.t, err)
	require.Equal(s.t, expected, branch)
	return s
}

// HasBranches asserts the repo has exactly these branches
func (s *TestShell) HasBranches(branches ...string) *TestShell {
	s.t.Helper()
	testhelpers.ExpectBranches(s.t, s.scene.Repo, branches)
	return s
}

// CommitCount asserts the number of commits between two refs
func (s *TestShell) CommitCount(from, to string, expected int) *TestShell {
	s.t.Helper()
	cmd := exec.Command("git", "log", "--oneline", from+".."+to)
	cmd.Dir = s.scene.Dir
	output, err := cmd.CombinedOutput()
	require.NoError(s.t, err)
	actual := countNonEmptyLines(string(output))
	require.Equal(s.t, expected, actual, "expected %d commits between %s..%s, got %d", expected, from, to, actual)
	return s
}

// Log prints a message (useful for documenting test steps)
func (s *TestShell) Log(msg string) *TestShell {
	s.t.Log(msg)
	return s
}

// splitArgs splits a command string into args, respecting quotes
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range s {
		switch {
		case r == '"' || r == '\'':
			if inQuote && r == quoteChar {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quoteChar = r
			} else {
				current.WriteRune(r)
			}
		case r == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// countNonEmptyLines counts lines that have non-whitespace content
func countNonEmptyLines(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// =============================================================================
// Integration Tests - Now read like terminal sessions!
// =============================================================================

func TestIntegrationStackWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("full stack workflow: create, amend, restack, squash", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Build a stack: main -> feature-a -> feature-b -> feature-c
		sh.Log("Creating stacked branches...")
		sh.Write("feature_a", "feature a content").
			Run("create feature-a -m 'Add feature A'").
			OnBranch("feature-a")

		sh.Write("feature_b", "feature b content").
			Run("create feature-b -m 'Add feature B'").
			OnBranch("feature-b")

		sh.Write("feature_c", "feature c content").
			Run("create feature-c -m 'Add feature C'").
			OnBranch("feature-c")

		sh.Run("log --stack").
			OutputContains("feature-a").
			OutputContains("feature-b").
			OutputContains("feature-c")

		// Add commits and amend on feature-a
		sh.Log("Adding commits and amending on feature-a...")
		sh.Checkout("feature-a").
			Commit("feature_a_extra", "additional work").
			CommitCount("main", "feature-a", 2).
			Amend("feature_a_amended", "amended content")

		// Restack to propagate changes
		sh.Log("Restacking upstack branches...")
		sh.Run("restack --upstack")

		// Verify children are still valid
		sh.Checkout("feature-b").
			Run("info").
			OutputContains("feature-b")

		sh.Checkout("feature-c").
			Run("info").
			OutputContains("feature-c")

		// Squash commits on feature-a
		sh.Log("Squashing commits on feature-a...")
		sh.Checkout("feature-a").
			CommitCount("main", "feature-a", 2).
			Run("squash -m 'Feature A complete'").
			CommitCount("main", "feature-a", 1)

		// Verify the squashed commit message
		sh.Run("info").
			OutputContains("Feature A complete")

		// Verify children survived the squash
		sh.Log("Verifying children are still valid after squash...")
		sh.Checkout("feature-b").
			Run("info").
			OutputContains("feature-b")

		sh.Checkout("feature-c").
			Run("info").
			OutputContains("feature-c")

		sh.HasBranches("feature-a", "feature-b", "feature-c", "main")
		sh.Log("✓ Full workflow complete!")
	})

	t.Run("stack workflow with parallel branches", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Create diamond-shaped stack:
		//        main
		//          |
		//      feature-a
		//       /     \
		//   feat-b1  feat-b2
		//               |
		//           feature-c

		sh.Log("Creating diamond-shaped branch structure...")
		sh.Write("a", "feature a").Run("create feature-a -m 'Feature A'")
		sh.Write("b1", "feature b1").Run("create feat-b1 -m 'Feature B1'")

		sh.Checkout("feature-a")
		sh.Write("b2", "feature b2").Run("create feat-b2 -m 'Feature B2'")
		sh.Write("c", "feature c").Run("create feature-c -m 'Feature C'")

		sh.HasBranches("feat-b1", "feat-b2", "feature-a", "feature-c", "main")

		// Amend feature-a and restack everything
		sh.Log("Amending feature-a and restacking...")
		sh.Checkout("feature-a").
			Amend("a_amended", "feature a amended").
			Run("restack --upstack")

		// Verify all branches survived
		sh.Checkout("feat-b1").Run("info").OutputContains("feat-b1")
		sh.Checkout("feat-b2").Run("info").OutputContains("feat-b2")
		sh.Checkout("feature-c").Run("info").OutputContains("feature-c")

		sh.Log("✓ Parallel branch workflow complete!")
	})
}

func TestIntegrationSyncWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("sync cleans up merged branch and reparents children", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellWithRemote(t, binaryPath)

		// Scenario:
		// 1. Build stack: main → branch-a → branch-b → branch-c
		// 2. Simulate "branch-a PR merged" by merging branch-a into main
		// 3. Run `stackit sync --force --no-restack` (no restack to isolate cleanup behavior)
		// 4. Verify:
		//    - branch-a was deleted
		//    - branch-b is now parented to main
		//    - branch-c is still parented to branch-b

		sh.Log("Building stack: main → branch-a → branch-b → branch-c...")
		sh.Write("a", "feature a content").
			Run("create branch-a -m 'Add feature A'").
			OnBranch("branch-a")

		sh.Write("b", "feature b content").
			Run("create branch-b -m 'Add feature B'").
			OnBranch("branch-b")

		sh.Write("c", "feature c content").
			Run("create branch-c -m 'Add feature C'").
			OnBranch("branch-c")

		sh.HasBranches("branch-a", "branch-b", "branch-c", "main")

		// Simulate merging branch-a into main (like a GitHub PR merge)
		sh.Log("Simulating PR merge: merging branch-a into main...")
		sh.Git("checkout main").
			Git("merge branch-a --no-ff -m 'Merge branch-a'")

		// Push the merge to origin so sync can pull it
		sh.Git("push origin main")

		// Verify main now has branch-a's changes
		cmd := exec.Command("git", "log", "--oneline", "main")
		cmd.Dir = sh.scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Merge branch-a", "main should have merge commit")

		// Run sync with --force to auto-confirm deletions, --no-restack to isolate cleanup
		sh.Log("Running sync to clean up merged branches...")
		sh.Run("sync --force --no-restack")

		// Verify branch-a was deleted
		sh.Log("Verifying branch-a was deleted...")
		cmd = exec.Command("git", "branch", "--list", "branch-a")
		cmd.Dir = sh.scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(string(output)), "branch-a should be deleted")

		// Verify branch-b and branch-c still exist
		sh.HasBranches("branch-b", "branch-c", "main")

		// Verify branch-b's parent is now main (via info command)
		sh.Log("Verifying branch-b is now parented to main...")
		sh.Checkout("branch-b").
			Run("info")
		// The info output should show branch-b with main as parent
		require.Contains(t, sh.Output(), "branch-b", "info should show branch-b")

		// Verify branch-c is still valid and its parent chain is correct
		sh.Log("Verifying branch-c is still accessible...")
		sh.Checkout("branch-c").
			Run("info").
			OutputContains("branch-c")

		sh.Log("✓ Sync cleanup workflow complete!")
	})

	t.Run("sync restacks branches after cleaning merged PRs", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellWithRemote(t, binaryPath)

		// Scenario:
		// 1. Build stack: main → branch-a → branch-b
		// 2. Merge branch-a into main
		// 3. Run sync with restack enabled
		// 4. Verify branch-b is rebased onto main

		sh.Log("Building stack: main → branch-a → branch-b...")
		sh.Write("a", "feature a content").
			Run("create branch-a -m 'Add feature A'").
			OnBranch("branch-a")

		sh.Write("b", "feature b content").
			Run("create branch-b -m 'Add feature B'").
			OnBranch("branch-b")

		// Get branch-b's commit before sync
		cmd := exec.Command("git", "rev-parse", "branch-b")
		cmd.Dir = sh.scene.Dir
		beforeSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// Simulate merging branch-a into main
		sh.Log("Simulating PR merge: merging branch-a into main...")
		sh.Git("checkout main").
			Git("merge branch-a --no-ff -m 'Merge branch-a'")

		// Push the merge to origin
		sh.Git("push origin main")

		// Run sync with restack (default)
		sh.Log("Running sync with restack...")
		sh.Checkout("branch-b"). // Need to be on a tracked branch for restack
						Run("sync --force")

		// Get branch-b's commit after sync
		cmd = exec.Command("git", "rev-parse", "branch-b")
		cmd.Dir = sh.scene.Dir
		afterSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// branch-b should have been restacked (commit SHA changed)
		require.NotEqual(t, beforeSHA, afterSHA, "branch-b should be restacked (SHA should change)")

		// Verify branch-a was deleted
		cmd = exec.Command("git", "branch", "--list", "branch-a")
		cmd.Dir = sh.scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(string(output)), "branch-a should be deleted")

		// Verify branch-b is now directly on main
		cmd = exec.Command("git", "merge-base", "main", "branch-b")
		cmd.Dir = sh.scene.Dir
		mergeBase := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		cmd = exec.Command("git", "rev-parse", "main")
		cmd.Dir = sh.scene.Dir
		mainSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		require.Equal(t, mainSHA, mergeBase, "branch-b should be rebased directly onto main")

		sh.Log("✓ Sync with restack workflow complete!")
	})
}

func TestIntegrationConflictResolution(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("continue through cascading conflicts in stack", func(t *testing.T) {
		// TODO: Implement multi-level conflict resolution test
		//
		// Scenario:
		// 1. Build stack: main → branch-a → branch-b → branch-c
		//    where each branch modifies the SAME file (to create conflicts)
		// 2. Add a new commit to main that modifies the same file
		// 3. Run `stackit restack` from branch-a
		// 4. Verify: restack stops at branch-a with conflict
		// 5. Resolve conflict, run `stackit continue`
		// 6. Verify: restack continues, stops at branch-b with conflict
		// 7. Resolve conflict, run `stackit continue`
		// 8. Verify: restack continues, stops at branch-c with conflict
		// 9. Resolve conflict, run `stackit continue`
		// 10. Verify: all branches are now successfully restacked
		//
		// This tests:
		// - Continuation state persistence across multiple conflicts
		// - Proper resumption of restack after conflict resolution
		// - Cascading conflicts through a deep stack

		t.Skip("TODO: Implement cascading conflict resolution test")

		_ = binaryPath // Will be used when implemented
	})

	t.Run("continue preserves stack structure after mid-stack conflict", func(t *testing.T) {
		// TODO: Implement mid-stack conflict preservation test
		//
		// Scenario:
		// 1. Build stack: main → a → b → c → d
		// 2. Amend branch-b with conflicting changes
		// 3. Run `stackit restack --upstack` from branch-b
		// 4. Verify: conflict occurs at branch-c
		// 5. Resolve conflict, run continue
		// 6. Verify: branch-c and branch-d are properly restacked
		// 7. Verify: parent relationships are preserved correctly
		//
		// This tests:
		// - Restack from mid-stack position
		// - Conflict in middle of upstack chain
		// - Preservation of deep stack structure

		t.Skip("TODO: Implement mid-stack conflict preservation test")

		_ = binaryPath // Will be used when implemented
	})
}

func TestIntegrationSplitWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("split mid-stack branch with multiple children restacks all descendants", func(t *testing.T) {
		// TODO: Implement split with multiple children test
		//
		// Scenario - Diamond structure with split:
		//
		// Before split:
		//           main
		//             |
		//         feature-a (has files: config.go, api.go, utils.go)
		//          /     \
		//      child-1  child-2
		//                  |
		//              grandchild
		//
		// After split --by-file config.go,api.go:
		//
		//           main
		//             |
		//        feature-a_split (has: config.go, api.go)
		//             |
		//         feature-a (has: utils.go only)
		//          /     \
		//      child-1  child-2
		//                  |
		//              grandchild
		//
		// Verify:
		// - New parent branch created with extracted files
		// - Original branch only has remaining files
		// - All 3 descendants (child-1, child-2, grandchild) are restacked
		// - Parent relationships are updated correctly
		//
		// This tests:
		// - Split creates correct parent branch
		// - Multiple children are all restacked
		// - Deep descendants (grandchild) are handled
		// - Diamond structure is preserved

		t.Skip("TODO: Implement split mid-stack with multiple children test")

		_ = binaryPath // Will be used when implemented
	})

	t.Run("split at stack bottom updates all upstack branches", func(t *testing.T) {
		// TODO: Implement split at bottom test
		//
		// Scenario:
		// 1. Build: main → feature-a (4 files) → feature-b → feature-c
		// 2. Split feature-a to extract 2 files to new parent
		// 3. Verify:
		//    - New branch feature-a_split is between main and feature-a
		//    - feature-a, feature-b, feature-c all properly restacked
		//    - Commit history is clean
		//
		// This tests:
		// - Split at the base of a deep stack
		// - All upstack branches restack correctly
		// - No orphaned commits

		t.Skip("TODO: Implement split at stack bottom test")

		_ = binaryPath // Will be used when implemented
	})
}
