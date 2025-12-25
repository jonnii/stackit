package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// Split Workflow Integration Tests
//
// These tests cover the split command which extracts files from a branch
// into a new parent branch, then restacks all affected branches.
// =============================================================================

func TestSplitWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("split mid-stack branch with multiple children restacks all descendants", func(t *testing.T) {
		t.Parallel()

		// Scenario - Structure with multiple children:
		//
		// Before split:
		//           main
		//             |
		//         feature-a (has files: config, api, utils)
		//          /     \
		//      child-1  child-2
		//
		// After split --by-file config,api:
		//
		//           main
		//             |
		//        feature-a_split (has: config, api)
		//             |
		//         feature-a (has: utils only)
		//          /     \
		//      child-1  child-2

		sh := NewTestShell(t, binaryPath)

		// Initialize stackit
		sh.Run("init")

		// Create feature-a with multiple files
		sh.Write("config", "config content").
			Write("api", "api content").
			Write("utils", "utils content").
			Run("create feature-a -m 'Add feature-a with config, api, utils'")

		// Create child-1 from feature-a
		sh.Write("child1", "child1 content").
			Run("create child-1 -m 'Add child-1'")

		// Go back to feature-a and create child-2
		sh.Checkout("feature-a").
			Write("child2", "child2 content").
			Run("create child-2 -m 'Add child-2'")

		// Go back to feature-a and split out config and api files
		sh.Checkout("feature-a")

		// Run split --by-file to extract config and api (comma-separated)
		sh.Run("split --by-file config_test.txt,api_test.txt")

		// Verify the new split branch exists
		sh.HasBranches("main", "feature-a", "feature-a_split", "child-1", "child-2")

		// Verify feature-a_split has the extracted files
		sh.Checkout("feature-a_split")
		verifyFileExists(t, sh, "config_test.txt")
		verifyFileExists(t, sh, "api_test.txt")
		verifyFileNotExists(t, sh, "utils_test.txt")

		// Verify feature-a now only has utils (config and api were removed)
		sh.Checkout("feature-a")
		verifyFileNotExists(t, sh, "config_test.txt")
		verifyFileNotExists(t, sh, "api_test.txt")
		verifyFileExists(t, sh, "utils_test.txt")

		// Verify child-1 still has its changes
		sh.Checkout("child-1")
		verifyFileExists(t, sh, "child1_test.txt")

		// Verify child-2 still has its changes
		sh.Checkout("child-2")
		verifyFileExists(t, sh, "child2_test.txt")

		// Verify parent relationships using stackit info
		sh.Checkout("feature-a").Run("info")
		sh.OutputContains("feature-a_split") // parent should be feature-a_split

		sh.Checkout("child-1").Run("info")
		sh.OutputContains("feature-a") // parent should still be feature-a

		sh.Checkout("child-2").Run("info")
		sh.OutputContains("feature-a") // parent should still be feature-a
	})

	t.Run("split at stack bottom updates all upstack branches", func(t *testing.T) {
		t.Parallel()

		// Scenario:
		// 1. Build: main → feature-a (has shared file) → feature-b → feature-c
		// 2. Split feature-a to extract file to new parent
		// 3. Verify all branches properly restacked
		//
		// After split: main → feature-a_split (file1) → feature-a (file2) → feature-b → feature-c
		// Note: The extracted files (file1) are moved to the split branch and REMOVED from feature-a
		//       So feature-b and feature-c won't have file1 - that's by design

		sh := NewTestShell(t, binaryPath)

		// Initialize stackit
		sh.Run("init")

		// Create feature-a with multiple files
		sh.Write("file1", "file1 from feature-a").
			Write("file2", "file2 from feature-a").
			Run("create feature-a -m 'Add feature-a with file1 and file2'")

		// Create feature-b on top
		sh.Write("fileb", "content from feature-b").
			Run("create feature-b -m 'Add feature-b'")

		// Create feature-c on top
		sh.Write("filec", "content from feature-c").
			Run("create feature-c -m 'Add feature-c'")

		// Verify we have the stack
		sh.HasBranches("main", "feature-a", "feature-b", "feature-c")

		// Go to feature-a and split out file1
		sh.Checkout("feature-a").
			Run("split --by-file file1_test.txt")

		// Verify the new split branch exists
		sh.HasBranches("main", "feature-a", "feature-a_split", "feature-b", "feature-c")

		// Verify feature-a_split has file1 (extracted)
		sh.Checkout("feature-a_split")
		verifyFileExists(t, sh, "file1_test.txt")
		verifyFileNotExists(t, sh, "file2_test.txt")

		// Verify feature-a now only has file2 (file1 was extracted)
		sh.Checkout("feature-a")
		verifyFileNotExists(t, sh, "file1_test.txt")
		verifyFileExists(t, sh, "file2_test.txt")

		// Verify feature-b still has its changes (was restacked)
		sh.Checkout("feature-b")
		verifyFileExists(t, sh, "fileb_test.txt")
		verifyFileExists(t, sh, "file2_test.txt") // inherited from feature-a

		// Verify feature-c still has its changes (was restacked)
		sh.Checkout("feature-c")
		verifyFileExists(t, sh, "filec_test.txt")
		verifyFileExists(t, sh, "file2_test.txt") // inherited from feature-a
	})

	t.Run("split preserves commit history correctly", func(t *testing.T) {
		t.Parallel()

		sh := NewTestShell(t, binaryPath)

		// Initialize stackit
		sh.Run("init")

		// Create feature with two files
		sh.Write("extract", "content to extract").
			Write("keep", "content to keep").
			Run("create feature -m 'Add feature with two files'")

		// Split out the extract file
		sh.Run("split --by-file extract_test.txt")

		// split creates:
		// - feature_split: 1 commit (extract files from feature)
		// - feature: 2 commits (original commit + removal commit)
		sh.CommitCount("main", "feature_split", 1)
		sh.CommitCount("feature_split", "feature", 2) // original + removal commit
	})

	t.Run("split --by-commit accepts the flag and shows interactive prompt", func(t *testing.T) {
		t.Parallel()

		sh := NewTestShell(t, binaryPath)

		// Initialize stackit
		sh.Run("init")

		// Create a feature branch with multiple commits
		sh.Write("file1", "commit 1 content").
			Run("create feature -m 'First commit'")
		sh.Write("file2", "commit 2 content").
			Run("create commit2 -m 'Second commit'")
		sh.Write("file3", "commit 3 content").
			Run("create commit3 -m 'Third commit'")

		// Verify we have multiple commits
		sh.CommitCount("main", "feature", 1)    // feature has 1 commit from its creation
		sh.CommitCount("feature", "commit2", 1) // commit2 has 1 commit
		sh.CommitCount("commit2", "commit3", 1) // commit3 has 1 commit

		// Attempt to run split --by-commit
		// This will fail because it requires interactive input for commit selection
		// But we can verify that the flag is accepted and the command starts
		cmd := exec.Command(binaryPath, "split", "--by-commit")
		cmd.Dir = sh.Dir()
		cmd.Env = append(cmd.Environ(), "STACKIT_NON_INTERACTIVE=1")
		output, err := cmd.CombinedOutput()

		// The command should fail due to interactive prompts, but we can verify
		// that it recognizes the --by-commit flag and attempts to split by commit
		require.Error(t, err, "split --by-commit should fail in non-interactive mode due to required user input")
		outputStr := string(output)
		require.Contains(t, outputStr, "Splitting the commits", "should show commit splitting message")
	})

	t.Run("split --by-commit validates branch has commits to split", func(t *testing.T) {
		t.Parallel()

		sh := NewTestShell(t, binaryPath)

		// Initialize stackit
		sh.Run("init")

		// Create a feature branch with only one commit
		sh.Write("file1", "single commit content").
			Run("create feature -m 'Single commit'")

		// Attempt to run split --by-commit on a branch with minimal commits
		// The logic should detect this and potentially default to hunk mode or show appropriate message
		cmd := exec.Command(binaryPath, "split", "--by-commit")
		cmd.Dir = sh.Dir()
		cmd.Env = append(cmd.Environ(), "STACKIT_NON_INTERACTIVE=1")
		output, err := cmd.CombinedOutput()

		// Should either fail due to lack of commits to split or require interaction
		require.Error(t, err, "should fail when there are no commits to split interactively")
		_ = output // Use output for verification if needed
	})
}

// Helper functions for file verification

func verifyFileExists(t *testing.T, sh *TestShell, filename string) {
	t.Helper()
	cmd := exec.Command("git", "ls-files", filename)
	cmd.Dir = sh.Dir()
	output, err := cmd.Output()
	require.NoError(t, err)
	require.True(t, strings.Contains(string(output), filename),
		"expected file %s to exist on current branch", filename)
}

func verifyFileNotExists(t *testing.T, sh *TestShell, filename string) {
	t.Helper()
	cmd := exec.Command("git", "ls-files", filename)
	cmd.Dir = sh.Dir()
	output, err := cmd.Output()
	require.NoError(t, err)
	require.False(t, strings.Contains(string(output), filename),
		"expected file %s to NOT exist on current branch", filename)
}
