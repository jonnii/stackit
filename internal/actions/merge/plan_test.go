package merge_test

import (
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
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
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
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
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
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
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
		err := s.Engine.UpsertPrInfo("P", &engine.PrInfo{Number: &prP, State: "OPEN"})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("C1", &engine.PrInfo{Number: &prC1, State: "OPEN"})
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

		// Verify info for sibling C2
		foundInfo := false
		for _, info := range plan.Infos {
			if strings.Contains(info, "C2") && strings.Contains(info, "moved to") {
				foundInfo = true
				break
			}
		}
		require.True(t, foundInfo, "Should have an info message about sibling C2 being moved")
	})

	t.Run("creates plan for scope-based merge", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"feature-a": "main",
				"feature-b": "feature-a",
				"feature-c": "feature-b",
				"other":     "main", // Different branch not in scope
			})

		// Set scopes on branches
		err := s.Engine.SetScope(s.Engine.GetBranch("feature-a"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("feature-b"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("feature-c"), engine.NewScope("PROJ-123"))
		require.NoError(t, err)
		// other branch has no scope

		// Add PR info for scoped branches
		prA := 101
		prB := 102
		prC := 103
		err = s.Engine.UpsertPrInfo("feature-a", &engine.PrInfo{
			Number: &prA,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("feature-b", &engine.PrInfo{
			Number: &prB,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("feature-c", &engine.PrInfo{
			Number: &prC,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Create plan with scope
		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Scope:    "PROJ-123",
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.True(t, validation.Valid)
		require.Equal(t, merge.StrategyBottomUp, plan.Strategy)
		require.Equal(t, "feature-c", plan.CurrentBranch) // Top-most branch in scope

		// Should include all branches with PROJ-123 scope, in topological order
		require.Len(t, plan.BranchesToMerge, 3)
		require.Equal(t, "feature-a", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "feature-b", plan.BranchesToMerge[1].BranchName)
		require.Equal(t, "feature-c", plan.BranchesToMerge[2].BranchName)
		require.Equal(t, 101, plan.BranchesToMerge[0].PRNumber)
		require.Equal(t, 102, plan.BranchesToMerge[1].PRNumber)
		require.Equal(t, 103, plan.BranchesToMerge[2].PRNumber)
	})

	t.Run("scope-based merge excludes branches without matching scope", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"scoped-a":   "main",
				"scoped-b":   "scoped-a",
				"unscoped-c": "main", // Different branch not in scope
			})

		// Set scopes on some branches
		err := s.Engine.SetScope(s.Engine.GetBranch("scoped-a"), engine.NewScope("PROJ-456"))
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("scoped-b"), engine.NewScope("PROJ-456"))
		require.NoError(t, err)
		// unscoped-c has no scope

		// Add PR info for all branches
		prA := 201
		prB := 202
		prC := 203
		err = s.Engine.UpsertPrInfo("scoped-a", &engine.PrInfo{
			Number: &prA,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("scoped-b", &engine.PrInfo{
			Number: &prB,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("unscoped-c", &engine.PrInfo{
			Number: &prC,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Create plan with scope
		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Scope:    "PROJ-456",
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.True(t, validation.Valid)

		// Should only include branches with PROJ-456 scope
		require.Len(t, plan.BranchesToMerge, 2)
		require.Equal(t, "scoped-a", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "scoped-b", plan.BranchesToMerge[1].BranchName)
	})

	t.Run("scope-based merge fails when no branches found with scope", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
			})

		// Set different scope
		err := s.Engine.SetScope(s.Engine.GetBranch("branch-a"), engine.NewScope("PROJ-789"))
		require.NoError(t, err)

		// Try to create plan with non-existent scope
		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Scope:    "NONEXISTENT",
		})

		require.Error(t, err)
		require.Nil(t, plan)
		require.Nil(t, validation)
		require.Contains(t, err.Error(), "no branches found with scope NONEXISTENT")
	})

	t.Run("scope-based merge handles scope inheritance", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"parent":     "main",
				"child":      "parent",
				"grandchild": "child",
			})

		// Set explicit scope only on parent
		err := s.Engine.SetScope(s.Engine.GetBranch("parent"), engine.NewScope("PROJ-999"))
		require.NoError(t, err)
		// child and grandchild should inherit PROJ-999

		// Add PR info for all branches
		prParent := 301
		prChild := 302
		prGrandchild := 303
		err = s.Engine.UpsertPrInfo("parent", &engine.PrInfo{
			Number: &prParent,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("child", &engine.PrInfo{
			Number: &prChild,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("grandchild", &engine.PrInfo{
			Number: &prGrandchild,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Create plan with scope - should include all branches that inherit the scope
		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Scope:    "PROJ-999",
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.True(t, validation.Valid)

		// Should include all branches that have or inherit PROJ-999 scope
		require.Len(t, plan.BranchesToMerge, 3)
		require.Equal(t, "parent", plan.BranchesToMerge[0].BranchName)
		require.Equal(t, "child", plan.BranchesToMerge[1].BranchName)
		require.Equal(t, "grandchild", plan.BranchesToMerge[2].BranchName)
	})

	t.Run("creates plan for consolidate strategy", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add PR info
		pr1 := 101
		pr2 := 102
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
		})
		require.NoError(t, err)

		// Switch to branch2
		s.Checkout("branch2")

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.NotNil(t, validation)
		require.Equal(t, merge.StrategyConsolidate, plan.Strategy)
		require.Equal(t, "branch2", plan.CurrentBranch)
		require.Len(t, plan.BranchesToMerge, 2)

		// Should have single consolidation step
		require.Len(t, plan.Steps, 3)
		require.Equal(t, merge.StepConsolidate, plan.Steps[0].StepType)
		require.Equal(t, merge.StepDeleteBranch, plan.Steps[1].StepType)
		require.Equal(t, merge.StepDeleteBranch, plan.Steps[2].StepType)
	})
}
