package actions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestCreateMergePlan(t *testing.T) {
	t.Run("creates plan for bottom-up strategy", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> branch1 -> branch2
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Add PR info
		pr1 := 101
		pr2 := 102
		err = eng.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = eng.UpsertPrInfo(context.Background(), "branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)

		// Rebuild engine
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		plan, validation, err := actions.CreateMergePlan(ctx, actions.CreateMergePlanOptions{
			Strategy: actions.MergeStrategyBottomUp,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.Equal(t, actions.MergeStrategyBottomUp, plan.Strategy)
		require.Equal(t, "branch2", plan.CurrentBranch)
		require.Len(t, plan.BranchesToMerge, 2)
		require.Equal(t, "branch1", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "branch2", plan.BranchesToMerge[1].BranchName)
		require.Greater(t, len(plan.Steps), 0)
	})

	t.Run("validates draft PRs", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Add draft PR
		pr1 := 101
		err = eng.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number:  &pr1,
			State:   "OPEN",
			IsDraft: true,
		})
		require.NoError(t, err)

		// Make sure we're on branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Rebuild engine to get current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		plan, validation, err := actions.CreateMergePlan(ctx, actions.CreateMergePlanOptions{
			Strategy: actions.MergeStrategyBottomUp,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.False(t, validation.Valid)
		require.Contains(t, validation.Errors[0], "draft")
	})

	t.Run("allows draft PRs with force", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Add draft PR
		pr1 := 101
		err = eng.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number:  &pr1,
			State:   "OPEN",
			IsDraft: true,
		})
		require.NoError(t, err)

		// Make sure we're on branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Rebuild engine to get current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		plan, validation, err := actions.CreateMergePlan(ctx, actions.CreateMergePlanOptions{
			Strategy: actions.MergeStrategyBottomUp,
			Force:    true,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		// With force, validation should pass (warnings may exist)
		require.True(t, validation.Valid)
	})

	t.Run("identifies upstack branches for restacking in branching stack", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create structure: main -> P -> [C1, C2], C1 -> GC1
		err := scene.Repo.CreateAndCheckoutBranch("P")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("P change", "p")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("C1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("C1 change", "c1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("GC1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("GC1 change", "gc1")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("P")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("C2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("C2 change", "c2")
		require.NoError(t, err)

		// Move back to C1
		err = scene.Repo.CheckoutBranch("C1")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "P", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "C1", "P")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "GC1", "C1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "C2", "P")
		require.NoError(t, err)

		// Add PR info for P and C1
		prP := 101
		prC1 := 102
		err = eng.UpsertPrInfo(context.Background(), "P", &engine.PrInfo{Number: &prP, State: "OPEN"})
		require.NoError(t, err)
		err = eng.UpsertPrInfo(context.Background(), "C1", &engine.PrInfo{Number: &prC1, State: "OPEN"})
		require.NoError(t, err)

		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		plan, _, err := actions.CreateMergePlan(ctx, actions.CreateMergePlanOptions{
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.NoError(t, err)

		// Branches to merge should be P and C1
		require.Len(t, plan.BranchesToMerge, 2)
		require.Equal(t, "P", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "C1", plan.BranchesToMerge[1].BranchName)

		// Upstack branches should include GC1 (child of C1)
		// It should NOT include C2 (sibling of C1) because we are merging up to C1.
		// Wait, if P is merged, C2 SHOULD be restacked onto trunk.
		// Let's see what the current implementation does.
		require.Contains(t, plan.UpstackBranches, "GC1")

		// Check if C2 is in UpstackBranches.
		// Currently, GetRelativeStackUpstack(currentBranch) is used.
		// If currentBranch is C1, it only includes GC1.
		require.NotContains(t, plan.UpstackBranches, "C2", "Sibling C2 should not be in upstack of C1")

		// Verify warning for sibling C2
		foundWarning := false
		for _, warn := range plan.Warnings {
			if strings.Contains(warn, "C2") && strings.Contains(warn, "reparented") {
				foundWarning = true
				break
			}
		}
		require.True(t, foundWarning, "Should have a warning about sibling C2 being reparented")
	})
}
