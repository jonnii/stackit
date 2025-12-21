package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestSwitchBranchAction(t *testing.T) {
	t.Run("traverses downward to bottom branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Switch to branch2 (top of stack)
		s.Checkout("branch2")

		// Traverse downward should go to branch1 (first branch from trunk)
		err := actions.SwitchBranchAction(actions.DirectionBottom, s.Context)
		require.NoError(t, err)

		// Should be on branch1
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})

	t.Run("traverses upward to top branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Switch to branch1
		s.Checkout("branch1")

		// Traverse upward should go to branch2 (top of stack)
		err := actions.SwitchBranchAction(actions.DirectionTop, s.Context)
		require.NoError(t, err)

		// Should be on branch2
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch2", currentBranch)
	})

	t.Run("returns error when not on a branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Detach HEAD
		s.RunGit("checkout", "HEAD~0").Rebuild()

		err := actions.SwitchBranchAction(actions.DirectionBottom, s.Context)
		require.Error(t, err)
	})

	t.Run("stays on branch when already at bottom", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Switch to branch1 (bottom of stack)
		s.Checkout("branch1")

		// Already on branch1 (bottom of stack)
		err := actions.SwitchBranchAction(actions.DirectionBottom, s.Context)
		require.NoError(t, err)

		// Should still be on branch1
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})

	t.Run("stays on branch when already at top", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Switch to branch1 (top of stack)
		s.Checkout("branch1")

		// Already at top
		err := actions.SwitchBranchAction(actions.DirectionTop, s.Context)
		require.NoError(t, err)

		// Should still be on branch1
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})
}
