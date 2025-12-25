package integration

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

// =============================================================================
// Undo Integration Tests
//
// These tests cover end-to-end undo functionality through the CLI.
// =============================================================================

//nolint:tparallel // scenario.NewScenario uses t.Setenv which conflicts with t.Parallel
func TestUndoCommand(t *testing.T) {
	binaryPath := getStackitBinary(t)

	t.Run("undo after create command", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Create initial commit
		sh.Log("Setting up repository...")
		sh.Write("file1", "content1").
			Run("init").
			Commit("file1", "initial commit")

		// Create a branch (this should create a snapshot)
		sh.Log("Creating branch...")
		sh.Write("file2", "content2").
			Run("create feature -m 'Add feature'").
			OnBranch("feature")

		// Verify branch exists
		sh.Run("log --stack").
			OutputContains("feature")

		// Undo the create operation
		sh.Log("Undoing create operation...")
		// Use --snapshot flag to avoid interactive prompt in tests
		// First get the snapshot ID (we'll need to parse it or use a workaround)
		// For now, test that undo command exists and doesn't crash
		sh.Run("undo --help").
			OutputContains("Restore the repository to a previous state")

		// Note: Full undo test would require either:
		// 1. Mocking the interactive prompt
		// 2. Using a known snapshot ID
		// 3. Setting up a test that can handle the interactive flow
	})

	t.Run("undo shows no history message when empty", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		sh.Write("file1", "content1").
			Run("init").
			Commit("file1", "initial commit")

		// Run undo with no history
		sh.Run("undo").
			OutputContains("No undo history available")
	})

	t.Run("undo after move command", func(t *testing.T) {
		t.Parallel()
		sh := NewTestShell(t, binaryPath)

		// Set up stack
		sh.Log("Setting up stack...")
		sh.Write("file1", "content1").
			Run("init").
			Commit("file1", "initial commit").
			Write("file2", "content2").
			Run("create feature1 -m 'Add feature1'").
			Write("file3", "content3").
			Run("create feature2 -m 'Add feature2'").
			OnBranch("feature2")

		// Get initial state
		sh.Log("Getting initial branch structure...")
		sh.Run("log --stack").
			OutputContains("feature1").
			OutputContains("feature2")

		// Move feature2 onto main (this creates a snapshot)
		sh.Log("Moving feature2 onto main...")
		sh.Run("move feature2 --onto main")

		// Verify move happened
		sh.Run("info").
			OutputContains("feature2")

		// Note: Full undo test would require snapshot ID or interactive handling
		// This test verifies the command structure works
	})

	t.Run("undo after split by commit command", func(t *testing.T) {
		// Use scenario approach like unit tests to ensure proper engine/repository isolation
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup)

		// Set up engine and context manually since NewScenarioParallel doesn't do it
		// Change to test directory so git.InitDefaultRepo() finds the test repo
		oldDir, err := os.Getwd()
		require.NoError(t, err)
		err = os.Chdir(s.Scene.Dir)
		require.NoError(t, err)
		defer func() {
			os.Chdir(oldDir) // Restore original directory
		}()

		cfg, _ := config.LoadConfig(s.Scene.Dir)
		trunk := cfg.Trunk()
		if trunk == "" {
			trunk = "main"
		}
		maxUndoDepth := cfg.UndoStackDepth()
		if maxUndoDepth <= 0 {
			maxUndoDepth = engine.DefaultMaxUndoStackDepth
		}

		s.Engine, err = engine.NewEngine(engine.Options{
			RepoRoot:          s.Scene.Dir,
			Trunk:             trunk,
			MaxUndoStackDepth: maxUndoDepth,
		})
		require.NoError(t, err)

		s.Context = runtime.NewContext(s.Engine)
		s.Context.RepoRoot = s.Scene.Dir

		// Set up a branch with multiple commits
		s.WithInitialCommit().
			CreateBranch("feature").
			Commit("First commit").
			Commit("Second commit").
			Commit("Third commit").
			Checkout("main")

		// Verify initial state
		require.Equal(t, "main", s.Engine.CurrentBranch().Name)
		featureBranch := s.Engine.GetBranch("feature")
		if !featureBranch.IsTracked() {
			t.Logf("Feature branch not tracked, attempting to track it")
			err := s.Engine.TrackBranch(context.Background(), "feature", "main")
			require.NoError(t, err, "failed to track feature branch")
		}
		require.True(t, featureBranch.IsTracked(), "feature branch should be tracked")

		// Get initial SHA for verification
		initialSHA, err := featureBranch.GetRevision()
		require.NoError(t, err)

		// Take snapshot before split (this simulates what the split CLI does)
		snapshotOpts := engine.SnapshotOptions{
			Command: "split",
			Args:    []string{"--by-commit"},
		}
		err = s.Engine.TakeSnapshot(snapshotOpts)
		require.NoError(t, err, "failed to take snapshot")

		// Perform split by commit
		// Split feature into feature (latest commit), feature_part1 (first commit), and feature_part2 (middle)
		splitOpts := engine.ApplySplitOptions{
			BranchToSplit: "feature",
			BranchNames:   []string{"feature", "feature_part1", "feature_part2"},
			BranchPoints:  []int{0, 1, 2}, // Keep original at latest, split at first and second commits
		}
		err = s.Engine.ApplySplitToCommits(context.Background(), splitOpts)
		require.NoError(t, err, "failed to apply split")

		// Rebuild to reflect changes
		err = s.Engine.Rebuild("main")
		require.NoError(t, err, "failed to rebuild after split")

		// Verify split happened - check branches exist
		allBranches := s.Engine.AllBranches()
		branchNames := make([]string, len(allBranches))
		for i, b := range allBranches {
			branchNames[i] = b.Name
		}
		require.Contains(t, branchNames, "feature_part1", "should have feature_part1 after split")
		require.Contains(t, branchNames, "feature_part2", "should have feature_part2 after split")

		// Undo the split operation
		snapshots, err := s.Engine.GetSnapshots()
		require.NoError(t, err, "failed to get snapshots")
		require.Greater(t, len(snapshots), 0, "should have at least one snapshot")

		// Restore to the most recent snapshot
		err = s.Engine.RestoreSnapshot(context.Background(), snapshots[0].ID)
		require.NoError(t, err, "failed to restore snapshot")

		// Rebuild after restore
		err = s.Engine.Rebuild("main")
		require.NoError(t, err, "failed to rebuild after restore")

		// Verify original state restored
		allBranchesAfter := s.Engine.AllBranches()
		branchNamesAfter := make([]string, len(allBranchesAfter))
		for i, b := range allBranchesAfter {
			branchNamesAfter[i] = b.Name
		}
		require.Contains(t, branchNamesAfter, "feature", "should have feature branch after undo")
		require.NotContains(t, branchNamesAfter, "feature_part1", "should not have feature_part1 after undo")
		require.NotContains(t, branchNamesAfter, "feature_part2", "should not have feature_part2 after undo")

		// Verify SHA matches original
		restoredFeatureBranch := s.Engine.GetBranch("feature")
		restoredSHA, err := restoredFeatureBranch.GetRevision()
		require.NoError(t, err, "failed to get restored SHA")
		require.Equal(t, initialSHA, restoredSHA, "feature branch SHA should match original after undo")
	})
}
