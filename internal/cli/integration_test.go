package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

// TestIntegrationStackWorkflow tests a realistic stacked branch workflow:
// 1. Create multiple stacked branches
// 2. Amend commits on a branch
// 3. Restack to propagate changes
// 4. Squash commits
func TestIntegrationStackWorkflow(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("full stack workflow: create, amend, restack, squash", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			// Create initial commit on main
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// =====================================================
		// STEP 1: Create multiple stacked branches
		// Stack structure: main -> feature-a -> feature-b -> feature-c
		// =====================================================

		// Create feature-a with a commit
		err := scene.Repo.CreateChange("feature a content", "feature_a", false)
		require.NoError(t, err)

		cmd := exec.Command(binaryPath, "create", "feature-a", "-m", "Add feature A")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "failed to create feature-a: %s", string(output))

		// Verify we're on feature-a
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "feature-a", currentBranch)

		// Create feature-b stacked on feature-a
		err = scene.Repo.CreateChange("feature b content", "feature_b", false)
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "create", "feature-b", "-m", "Add feature B")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "failed to create feature-b: %s", string(output))

		// Create feature-c stacked on feature-b
		err = scene.Repo.CreateChange("feature c content", "feature_c", false)
		require.NoError(t, err)

		cmd = exec.Command(binaryPath, "create", "feature-c", "-m", "Add feature C")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "failed to create feature-c: %s", string(output))

		// Verify stack structure with log
		cmd = exec.Command(binaryPath, "log", "--stack")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "failed to run log: %s", string(output))
		logOutput := string(output)
		require.Contains(t, logOutput, "feature-a")
		require.Contains(t, logOutput, "feature-b")
		require.Contains(t, logOutput, "feature-c")

		t.Log("Step 1 complete: Created stack main -> feature-a -> feature-b -> feature-c")

		// =====================================================
		// STEP 2: Add more commits and amend on feature-a
		// =====================================================

		// Go back to feature-a
		err = scene.Repo.CheckoutBranch("feature-a")
		require.NoError(t, err)

		// Add another commit to feature-a
		err = scene.Repo.CreateChange("feature a - more content", "feature_a_extra", false)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature a - additional work", "extra")
		require.NoError(t, err)

		// Verify feature-a now has 2 commits above main
		cmd = exec.Command("git", "log", "--oneline", "main..feature-a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		commitCount := countNonEmptyLines(string(output))
		require.Equal(t, 2, commitCount, "feature-a should have 2 commits above main")

		// Now amend the last commit
		err = scene.Repo.CreateChangeAndAmend("feature a - amended content", "feature_a_amended")
		require.NoError(t, err)

		t.Log("Step 2 complete: Added commits and amended on feature-a")

		// =====================================================
		// STEP 3: Restack to propagate changes to child branches
		// =====================================================

		// Restack upstack from feature-a (should update feature-b and feature-c)
		cmd = exec.Command(binaryPath, "restack", "--upstack")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "failed to restack: %s", string(output))

		t.Log("Step 3 complete: Restacked upstack branches")

		// Verify feature-b is still valid after restack
		err = scene.Repo.CheckoutBranch("feature-b")
		require.NoError(t, err)

		cmd = exec.Command("git", "log", "--oneline", "feature-a..feature-b")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Add feature B", "feature-b should still have its commit")

		// Verify feature-c is still valid after restack
		err = scene.Repo.CheckoutBranch("feature-c")
		require.NoError(t, err)

		cmd = exec.Command("git", "log", "--oneline", "feature-b..feature-c")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Add feature C", "feature-c should still have its commit")

		// =====================================================
		// STEP 4: Squash commits on feature-a
		// =====================================================

		// Go back to feature-a
		err = scene.Repo.CheckoutBranch("feature-a")
		require.NoError(t, err)

		// Verify feature-a has 2 commits before squash
		cmd = exec.Command("git", "log", "--oneline", "main..feature-a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		beforeSquash := countNonEmptyLines(string(output))
		require.Equal(t, 2, beforeSquash, "feature-a should have 2 commits before squash")

		// Squash commits on feature-a
		cmd = exec.Command(binaryPath, "squash", "-m", "Feature A complete")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "failed to squash: %s", string(output))

		t.Log("Step 4 complete: Squashed commits on feature-a")

		// Verify feature-a now has only 1 commit
		cmd = exec.Command("git", "log", "--oneline", "main..feature-a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		afterSquash := countNonEmptyLines(string(output))
		require.Equal(t, 1, afterSquash, "feature-a should have 1 commit after squash")

		// Verify the commit message is correct
		cmd = exec.Command("git", "log", "-1", "--format=%s", "feature-a")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Feature A complete")

		// =====================================================
		// STEP 5: Verify child branches are still valid after squash
		// =====================================================

		// Squash should have automatically restacked children
		err = scene.Repo.CheckoutBranch("feature-b")
		require.NoError(t, err)

		// Verify feature-b's commit is still there
		cmd = exec.Command("git", "log", "--oneline", "feature-a..feature-b")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Add feature B", "feature-b should still have its commit after squash restack")

		// Verify feature-c's commit is still there
		err = scene.Repo.CheckoutBranch("feature-c")
		require.NoError(t, err)

		cmd = exec.Command("git", "log", "--oneline", "feature-b..feature-c")
		cmd.Dir = scene.Dir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Add feature C", "feature-c should still have its commit after squash restack")

		t.Log("Step 5 complete: Verified all branches are valid after full workflow")

		// =====================================================
		// Final verification: All branches still exist
		// =====================================================
		testhelpers.ExpectBranches(t, scene.Repo, []string{"feature-a", "feature-b", "feature-c", "main"})
	})

	t.Run("stack workflow with parallel branches", func(t *testing.T) {
		t.Parallel()
		scene := testhelpers.NewSceneParallel(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a diamond-shaped stack:
		//     main
		//       |
		//   feature-a
		//    /     \
		// feat-b1  feat-b2
		//    \     /
		//   feature-c (has both as parents conceptually, but only one direct parent)

		// Create feature-a
		err := scene.Repo.CreateChange("feature a", "a", false)
		require.NoError(t, err)
		cmd := exec.Command(binaryPath, "create", "feature-a", "-m", "Feature A")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create feat-b1 from feature-a
		err = scene.Repo.CreateChange("feature b1", "b1", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "feat-b1", "-m", "Feature B1")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Go back to feature-a and create feat-b2
		err = scene.Repo.CheckoutBranch("feature-a")
		require.NoError(t, err)
		err = scene.Repo.CreateChange("feature b2", "b2", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "feat-b2", "-m", "Feature B2")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Create feature-c on top of feat-b2
		err = scene.Repo.CreateChange("feature c", "c", false)
		require.NoError(t, err)
		cmd = exec.Command(binaryPath, "create", "feature-c", "-m", "Feature C")
		cmd.Dir = scene.Dir
		_, err = cmd.CombinedOutput()
		require.NoError(t, err)

		// Verify structure
		testhelpers.ExpectBranches(t, scene.Repo, []string{"feat-b1", "feat-b2", "feature-a", "feature-c", "main"})

		// Now amend feature-a and restack everything
		err = scene.Repo.CheckoutBranch("feature-a")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndAmend("feature a amended", "a_amended")
		require.NoError(t, err)

		// Restack upstack - should update both b1, b2, and c
		cmd = exec.Command(binaryPath, "restack", "--upstack")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "failed to restack parallel branches: %s", string(output))

		// Verify all branches are still valid
		for _, branch := range []string{"feat-b1", "feat-b2", "feature-c"} {
			err = scene.Repo.CheckoutBranch(branch)
			require.NoError(t, err, "should be able to checkout %s", branch)
		}

		t.Log("Parallel branch workflow complete")
	})
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
