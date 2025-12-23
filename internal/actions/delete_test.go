package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestDelete(t *testing.T) {
	t.Run("deletes a single branch", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "branch1",
			Force:      true,
		})
		require.NoError(t, err)

		// branch1 should be gone, branch2 should be reparented to main
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())
		require.True(t, s.Engine.GetBranch("branch2").IsTracked())
		parent2 := s.Engine.GetParent("branch2")
		require.NotNil(t, parent2)
		require.Equal(t, "main", parent2.Name)
	})

	t.Run("deletes upstack", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "branch1",
			Upstack:    true,
			Force:      true,
		})
		require.NoError(t, err)

		// All branches should be gone
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())
		require.False(t, s.Engine.GetBranch("branch2").IsTracked())
		require.False(t, s.Engine.GetBranch("branch3").IsTracked())
	})

	t.Run("deletes downstack", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "branch3",
			Downstack:  true,
			Force:      true,
		})
		require.NoError(t, err)

		// All branches should be gone
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())
		require.False(t, s.Engine.GetBranch("branch2").IsTracked())
		require.False(t, s.Engine.GetBranch("branch3").IsTracked())
	})

	t.Run("fails without force if not merged", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Add a commit to branch1 so it's not merged
		s.Checkout("branch1").Commit("some change")
		s.Engine.Rebuild("main")

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "branch1",
			Force:      false,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "use --force")
	})

	t.Run("deletes current branch and switches to trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
			})

		s.Checkout("branch1")
		currentBranch := s.Engine.CurrentBranch()
		require.NotNil(t, currentBranch)
		require.Equal(t, "branch1", currentBranch.Name)

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "branch1",
			Force:      true,
		})
		require.NoError(t, err)

		currentBranch = s.Engine.CurrentBranch()
		require.NotNil(t, currentBranch)
		require.Equal(t, "main", currentBranch.Name)
	})

	t.Run("deletes a branch in a branching stack", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"parent": "main",
				"child1": "parent",
				"child2": "parent",
			})

		err := actions.Delete(s.Context, actions.DeleteOptions{
			BranchName: "parent",
			Force:      true,
		})
		require.NoError(t, err)

		// parent should be gone
		require.False(t, s.Engine.GetBranch("parent").IsTracked())

		// Both children should be reparented to main and still be tracked
		require.True(t, s.Engine.GetBranch("child1").IsTracked())
		require.True(t, s.Engine.GetBranch("child2").IsTracked())
		parent1 := s.Engine.GetParent("child1")
		require.NotNil(t, parent1)
		require.Equal(t, "main", parent1.Name)
		parent2 := s.Engine.GetParent("child2")
		require.NotNil(t, parent2)
		require.Equal(t, "main", parent2.Name)
	})
}
