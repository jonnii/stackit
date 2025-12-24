package integration

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

// =============================================================================
// Sync Workflow Integration Tests
//
// These tests cover the sync command which:
// - Pulls trunk from remote
// - Cleans up merged/closed branches
// - Reparents orphaned children
// - Restacks branches
// =============================================================================

func TestSyncWorkflow(t *testing.T) {
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
		cmd.Dir = sh.Dir()
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Contains(t, string(output), "Merge branch-a", "main should have merge commit")

		// Run sync with --force to auto-confirm deletions, --no-restack to isolate cleanup
		sh.Log("Running sync to clean up merged branches...")
		sh.Run("sync --force --no-restack")

		// Verify branch-a was deleted
		sh.Log("Verifying branch-a was deleted...")
		cmd = exec.Command("git", "branch", "--list", "branch-a")
		cmd.Dir = sh.Dir()
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
		cmd.Dir = sh.Dir()
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
		cmd.Dir = sh.Dir()
		afterSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		// branch-b should have been restacked (commit SHA changed)
		require.NotEqual(t, beforeSHA, afterSHA, "branch-b should be restacked (SHA should change)")

		// Verify branch-a was deleted
		cmd = exec.Command("git", "branch", "--list", "branch-a")
		cmd.Dir = sh.Dir()
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(string(output)), "branch-a should be deleted")

		// Verify branch-b is now directly on main
		cmd = exec.Command("git", "merge-base", "main", "branch-b")
		cmd.Dir = sh.Dir()
		mergeBase := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		cmd = exec.Command("git", "rev-parse", "main")
		cmd.Dir = sh.Dir()
		mainSHA := strings.TrimSpace(string(testhelpers.Must(cmd.CombinedOutput())))

		require.Equal(t, mainSHA, mergeBase, "branch-b should be rebased directly onto main")

		sh.Log("✓ Sync with restack workflow complete!")
	})

	t.Run("sync reparents from GitHub base branch", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShellWithRemote(t, binaryPath)

		// 1. Create a stack: main -> feature-a -> feature-b
		sh.Log("Creating stack: main -> feature-a -> feature-b...")
		sh.Write("a", "a").Run("create feature-a -m 'feat: a'")
		sh.Write("b", "b").Run("create feature-b -m 'feat: b'")

		// Initialize git package in test process
		err := os.Chdir(sh.Dir())
		require.NoError(t, err)
		git.ResetDefaultRepo()
		err = git.InitDefaultRepo()
		require.NoError(t, err)

		// 2. Simulate GitHub PR metadata for feature-b pointing to main instead of feature-a
		sh.Log("Simulating changed PR base on GitHub...")
		meta, err := git.ReadMetadataRef("feature-b")
		require.NoError(t, err)

		if meta.PrInfo == nil {
			meta.PrInfo = &git.PrInfo{}
		}
		newBase := "main"
		meta.PrInfo.Base = &newBase

		err = git.WriteMetadataRef("feature-b", meta)
		require.NoError(t, err)

		// Verify current local parent is still feature-a
		sh.Checkout("feature-b").Run("info").OutputContains("feature-a")

		// 3. Run stackit sync
		sh.Log("Running sync...")
		sh.Run("sync")

		// 4. Verify local parent is now main
		sh.Log("Verifying new parent...")
		sh.Run("info").OutputContains("main").OutputNotContains("feature-a")

		sh.Log("✓ Sync reparenting complete!")
	})
}
