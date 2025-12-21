package actions_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCleanBranches(t *testing.T) {
	t.Run("deletes merged branch and updates children", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Merge branch1 into main
		s.Checkout("main").
			RunGit("merge", "branch1")

		// Rebuild to see changes
		err := s.Engine.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Mark branch1 as merged via PR info
		prNumber := 1
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "MERGED",
			Base:   "main",
		}
		err = s.Engine.UpsertPrInfo(context.Background(), "branch1", prInfo)
		require.NoError(t, err)

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// branch1 should be deleted
		require.False(t, s.Engine.IsBranchTracked("branch1"))

		// branch2 should have new parent (main)
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
		require.Contains(t, result.BranchesWithNewParents, "branch2")
	})

	t.Run("handles multiple children when parent is deleted", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch1",
			})

		// Merge branch1
		s.Checkout("main").
			RunGit("merge", "branch1")

		// Rebuild to see changes
		err := s.Engine.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Mark branch1 as merged
		prNumber := 1
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "MERGED",
		}
		err = s.Engine.UpsertPrInfo(context.Background(), "branch1", prInfo)
		require.NoError(t, err)

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// Both children should have new parent
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
		require.Equal(t, "main", s.Engine.GetParent("branch3"))
		require.Contains(t, result.BranchesWithNewParents, "branch2")
		require.Contains(t, result.BranchesWithNewParents, "branch3")
	})

	t.Run("does not delete branch without PR when not merged", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		result, err := actions.CleanBranches(s.Context, actions.CleanBranchesOptions{
			Force: false,
		})
		require.NoError(t, err)

		// Branch should still exist
		require.True(t, s.Engine.IsBranchTracked("branch1"))
		require.Empty(t, result.BranchesWithNewParents)
	})
}
