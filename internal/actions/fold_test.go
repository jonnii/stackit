package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestFoldAction(t *testing.T) {
	t.Run("folds branch into parent", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Switch to branch2
		s.Checkout("branch2")

		// Fold branch2 into branch1
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.NoError(t, err)

		// Verify branch2 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch2")

		// Verify we're on branch1
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)

		// Verify branch1 contains both commits by checking log
		logOutput, err := s.Scene.Repo.RunGitCommandAndGetOutput("log", "--oneline", "main..branch1")
		require.NoError(t, err)
		require.Contains(t, logOutput, "change on branch1")
		require.Contains(t, logOutput, "change on branch2")
	})

	t.Run("reparents children when folding branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		// Switch to branch2
		s.Checkout("branch2")

		// Fold branch2 into branch1
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.NoError(t, err)

		// Verify branch3's parent is now branch1
		parent := s.Engine.GetParent("branch3")
		require.Equal(t, "branch1", parent)

		// Verify branch2 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch2")
	})

	t.Run("folds with --keep flag", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Switch to branch2
		s.Checkout("branch2")

		// Fold branch1 into branch2 with --keep
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: true})
		require.NoError(t, err)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")

		// Verify we're on branch2
		currentBranch, err := s.Scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch2", currentBranch)

		// Verify branch2's parent is now main
		parent := s.Engine.GetParent("branch2")
		require.Equal(t, "main", parent)

		// Verify branch2 contains both commits by checking log
		logOutput, err := s.Scene.Repo.RunGitCommandAndGetOutput("log", "--oneline", "main..branch2")
		require.NoError(t, err)
		require.Contains(t, logOutput, "change on branch1")
		require.Contains(t, logOutput, "change on branch2")
	})

	t.Run("folds with --keep and reparents siblings", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch1",
			})

		// Switch to branch2
		s.Checkout("branch2")

		// Fold branch1 into branch2 with --keep
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: true})
		require.NoError(t, err)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")

		// Verify branch3's parent is now branch2
		parent := s.Engine.GetParent("branch3")
		require.Equal(t, "branch2", parent)

		// Verify branch2's parent is now main
		parent = s.Engine.GetParent("branch2")
		require.Equal(t, "main", parent)
	})

	t.Run("fails when trying to fold trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Try to fold trunk (main)
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold trunk branch")
	})

	t.Run("fails when trying to fold untracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("untracked")

		// Try to fold untracked branch
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold untracked branch")
	})

	t.Run("fails when trying to fold into trunk with --keep", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Try to fold into trunk with --keep
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold into trunk with --keep")
	})

	t.Run("fails when there are uncommitted changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			}).
			WithUncommittedChange("dirty")

		// Try to fold with dirty tree
		err := actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})

	t.Run("returns clear error message on merge conflict", func(t *testing.T) {
		s := scenario.NewScenario(t, nil)

		// Create conflicting changes manually
		s.RunGit("commit", "--allow-empty", "-m", "init")
		err := s.Scene.Repo.CreateChangeAndCommit("initial content\n", "conflict.txt")
		require.NoError(t, err)

		s.CreateBranch("branch1")
		s.CreateBranch("branch2")

		// parent (branch1) adds a conflicting change
		s.Checkout("branch1")
		err = s.Scene.Repo.CreateChangeAndCommit("branch1 content\n", "conflict.txt")
		require.NoError(t, err)
		s.TrackBranch("branch1", "main")

		// child (branch2) adds a conflicting change
		s.Checkout("branch2")
		err = s.Scene.Repo.CreateChangeAndCommit("branch2 content\n", "conflict.txt")
		require.NoError(t, err)
		s.TrackBranch("branch2", "branch1")

		// Fold branch2 into branch1 - should conflict
		err = actions.FoldAction(s.Context, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "due to conflicts. Please resolve the conflicts and run 'git commit'")
	})
}
