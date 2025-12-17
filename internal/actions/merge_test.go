package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestMergeAction(t *testing.T) {
	t.Run("fails when not on a branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Detach HEAD
		err := scene.Repo.RunGitCommand("checkout", "HEAD~0")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not on a branch")
	})

	t.Run("fails when on trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Make sure we're on main
		err := scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see we're on main)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify we're on trunk
		require.True(t, eng.IsTrunk(eng.CurrentBranch()))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})

	t.Run("fails when branch is not tracked", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		// Create engine (will see branch1 but not track it)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// Note: The branch won't be tracked in the new engine since we didn't track it
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is not tracked
		require.False(t, eng.IsBranchTracked("branch1"))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("returns early when no PRs to merge", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see branch1)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// The engine will rebuild and should see the tracked branch from metadata
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is tracked (metadata should persist)
		require.True(t, eng.IsBranchTracked("branch1"))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		// Should fail because no PRs found
		require.Error(t, err)
		require.Contains(t, err.Error(), "no open PRs found")
	})

	t.Run("dry run mode reports PRs without merging", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see branch1)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Add PR info
		prNumber := 123
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/123",
		}
		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// The engine will rebuild and should see the tracked branch and PR info from metadata
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is tracked and has PR info
		require.True(t, eng.IsBranchTracked("branch1"))
		prInfo, err = eng.GetPrInfo("branch1")
		require.NoError(t, err)
		require.NotNil(t, prInfo)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   true,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.NoError(t, err)
	})
}
