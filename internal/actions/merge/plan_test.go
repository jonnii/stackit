package merge_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCreateMergePlan(t *testing.T) {
	t.Run("creates plan for bottom-up strategy", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add PR info
		pr1 := 101
		pr2 := 102
		err := s.Engine.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo(context.Background(), "branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Switch to branch2
		s.Checkout("branch2")

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.Equal(t, merge.StrategyBottomUp, plan.Strategy)
		require.Equal(t, "branch2", plan.CurrentBranch)
		require.Len(t, plan.BranchesToMerge, 2)
		require.Equal(t, "branch1", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "branch2", plan.BranchesToMerge[1].BranchName)
		require.Greater(t, len(plan.Steps), 0)
	})

	t.Run("validates draft PRs", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Add draft PR
		pr1 := 101
		err := s.Engine.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number:  &pr1,
			State:   "OPEN",
			IsDraft: true,
		})
		require.NoError(t, err)

		// Make sure we're on branch1
		s.Checkout("branch1")

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.False(t, validation.Valid)
		require.Contains(t, validation.Errors[0], "draft")
	})

	t.Run("allows draft PRs with force", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Add draft PR
		pr1 := 101
		err := s.Engine.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number:  &pr1,
			State:   "OPEN",
			IsDraft: true,
		})
		require.NoError(t, err)

		// Make sure we're on branch1
		s.Checkout("branch1")

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Force:    true,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		// With force, validation should pass (warnings may exist)
		require.True(t, validation.Valid)
	})

	t.Run("identifies upstack branches for restacking in branching stack", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"P":   "main",
				"C1":  "P",
				"GC1": "C1",
				"C2":  "P",
			})

		// Move back to C1
		s.Checkout("C1")

		// Add PR info for P and C1
		prP := 101
		prC1 := 102
		err := s.Engine.UpsertPrInfo(context.Background(), "P", &engine.PrInfo{Number: &prP, State: "OPEN"})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo(context.Background(), "C1", &engine.PrInfo{Number: &prC1, State: "OPEN"})
		require.NoError(t, err)

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
		})
		require.NoError(t, err)

		// Branches to merge should be P and C1
		require.Len(t, plan.BranchesToMerge, 2)
		require.Equal(t, "P", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "C1", plan.BranchesToMerge[1].BranchName)

		// Upstack branches should include GC1 (child of C1)
		require.Contains(t, plan.UpstackBranches, "GC1")

		// Check if C2 is in UpstackBranches.
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
