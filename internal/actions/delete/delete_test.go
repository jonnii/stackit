package delete

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers/scenario"
)

func TestDelete(t *testing.T) {
	t.Run("deletes a single branch", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		err := Action(s.Context, Options{
			BranchName: "branch1",
			Force:      true,
		})
		require.NoError(t, err)

		// branch1 should be gone, branch2 should be reparented to main
		require.False(t, s.Engine.GetBranch("branch1").IsTracked())
		require.True(t, s.Engine.GetBranch("branch2").IsTracked())
		branchparent2 := s.Engine.GetBranch("branch2")
		parent2 := s.Engine.GetParent(branchparent2)
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

		err := Action(s.Context, Options{
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

		err := Action(s.Context, Options{
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

		err := Action(s.Context, Options{
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

		err := Action(s.Context, Options{
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

		err := Action(s.Context, Options{
			BranchName: "parent",
			Force:      true,
		})
		require.NoError(t, err)

		// parent should be gone
		require.False(t, s.Engine.GetBranch("parent").IsTracked())

		// Both children should be reparented to main and still be tracked
		require.True(t, s.Engine.GetBranch("child1").IsTracked())
		require.True(t, s.Engine.GetBranch("child2").IsTracked())
		branchparent1 := s.Engine.GetBranch("child1")
		parent1 := s.Engine.GetParent(branchparent1)
		require.NotNil(t, parent1)
		require.Equal(t, "main", parent1.Name)
		branchparent2 := s.Engine.GetBranch("child2")
		parent2 := s.Engine.GetParent(branchparent2)
		require.NotNil(t, parent2)
		require.Equal(t, "main", parent2.Name)
	})
}
