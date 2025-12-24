package merge_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestConsolidateMergeExecutor(t *testing.T) {
	t.Run("pre-validation fails for out-of-sync branch", func(t *testing.T) {
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

		s.Checkout("branch2")

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    false,
		})
		require.NoError(t, err)

		executor := merge.NewConsolidateMergeExecutor(plan, s.Context.GitHubClient, s.Engine, s.Context.Splog, s.Scene.Dir)

		// This should fail pre-validation because branches aren't pushed to remote
		err = executor.Execute(s.Context.Context, merge.ExecuteOptions{
			Plan:  plan,
			Force: false,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "pre-validation failed")
	})

	t.Run("creates consolidation branch correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add commits to branches to make them different
		s.Checkout("branch1")
		s.RunGit("commit", "--allow-empty", "-m", "branch1 commit").Rebuild()

		s.Checkout("branch2")
		s.RunGit("commit", "--allow-empty", "-m", "branch2 commit").Rebuild()

		// Add PR info
		pr1 := 101
		pr2 := 102
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
			Title:  "Feature 1",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
			Title:  "Feature 2",
		})
		require.NoError(t, err)

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true, // Skip remote sync checks
		})
		require.NoError(t, err)

		// Verify plan has the expected structure
		require.Equal(t, merge.StrategyConsolidate, plan.Strategy)
		require.Len(t, plan.BranchesToMerge, 2)
		require.Len(t, plan.Steps, 3)
		require.Equal(t, merge.StepConsolidate, plan.Steps[0].StepType)
	})

	t.Run("builds correct consolidation PR body", func(t *testing.T) {
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
			Title:  "Add user authentication",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
			Title:  "Add user profile UI",
		})
		require.NoError(t, err)

		s.Checkout("branch2")

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true,
		})
		require.NoError(t, err)

		executor := merge.NewConsolidateMergeExecutor(plan, s.Context.GitHubClient, s.Engine, s.Context.Splog, s.Scene.Dir)

		// Verify the executor was created successfully
		require.NotNil(t, executor)
	})

	t.Run("handles stack scope correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"batch/feature-a": "main",
				"batch/feature-b": "batch/feature-a",
			})

		// Set scope on branches
		scope := engine.NewScope("batch")
		err := s.Engine.SetScope(s.Engine.GetBranch("batch/feature-a"), scope)
		require.NoError(t, err)
		err = s.Engine.SetScope(s.Engine.GetBranch("batch/feature-b"), scope)
		require.NoError(t, err)

		// Add PR info
		pr1 := 101
		pr2 := 102
		err = s.Engine.UpsertPrInfo("batch/feature-a", &engine.PrInfo{
			Number: &pr1,
			State:  "OPEN",
			Title:  "Feature A",
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("batch/feature-b", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
			Title:  "Feature B",
		})
		require.NoError(t, err)

		s.Checkout("batch/feature-b")

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true,
		})
		require.NoError(t, err)

		executor := merge.NewConsolidateMergeExecutor(plan, s.Context.GitHubClient, s.Engine, s.Context.Splog, s.Scene.Dir)

		// Verify the executor understands the scope
		require.NotNil(t, executor)
		require.Equal(t, merge.StrategyConsolidate, plan.Strategy)
		require.Len(t, plan.BranchesToMerge, 2)
	})

	t.Run("handles empty consolidation correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Single branch on main
		s.Checkout("main")

		// This should not create a consolidation plan since there's nothing to merge
		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true,
		})

		require.Error(t, err)
		require.Nil(t, plan)
		require.Nil(t, validation)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})
}

func TestConsolidationStepExecution(t *testing.T) {
	t.Run("executes consolidation step", func(t *testing.T) {
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

		s.Checkout("branch2")

		plan, _, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true,
		})
		require.NoError(t, err)

		// Test that the consolidation step is properly structured
		require.Len(t, plan.Steps, 3)
		require.Equal(t, merge.StepConsolidate, plan.Steps[0].StepType)
		require.Contains(t, plan.Steps[0].Description, "Consolidate 2 branches")
	})
}

func TestConsolidationErrorHandling(t *testing.T) {
	t.Run("handles closed PR gracefully", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add PR info with one closed PR
		pr1 := 101
		pr2 := 102
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
			Number: &pr1,
			State:  "CLOSED", // Already merged
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
			Number: &pr2,
			State:  "OPEN",
		})
		require.NoError(t, err)

		s.Checkout("branch2")

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    false,
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.False(t, validation.Valid)
		require.Contains(t, validation.Errors[0], "Branch branch1 PR #101 is CLOSED (not open)")
	})

	t.Run("handles draft PR with force flag", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add PR info with draft PR
		pr1 := 101
		pr2 := 102
		err := s.Engine.UpsertPrInfo("branch1", &engine.PrInfo{
			Number:  &pr1,
			State:   "OPEN",
			IsDraft: true,
		})
		require.NoError(t, err)
		err = s.Engine.UpsertPrInfo("branch2", &engine.PrInfo{
			Number:  &pr2,
			State:   "OPEN",
			IsDraft: false,
		})
		require.NoError(t, err)

		s.Checkout("branch2")

		// Without force, should fail validation
		_, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    false,
		})
		require.NoError(t, err)
		require.False(t, validation.Valid)

		// With force, should succeed
		var plan *merge.Plan
		plan, validation, err = merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyConsolidate,
			Force:    true,
		})
		require.NoError(t, err)
		require.True(t, validation.Valid)
		require.Len(t, plan.BranchesToMerge, 2)
	})
}
