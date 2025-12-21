package actions_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestMoveAction(t *testing.T) {
	t.Run("moves branch downstack and restacks descendants", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		// Verify initial state
		require.Equal(t, "branch1", s.Engine.GetParent("branch2"))
		require.Equal(t, "branch2", s.Engine.GetParent("branch3"))

		// Move branch2 from branch1 to main (downstack)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch2",
			Onto:   "main",
		})
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
		require.Contains(t, s.Engine.GetChildren("main"), "branch2")
		require.NotContains(t, s.Engine.GetChildren("branch1"), "branch2")

		// Verify branch3 still has branch2 as parent (descendant relationship preserved)
		require.Equal(t, "branch2", s.Engine.GetParent("branch3"))
	})

	t.Run("moves branch upstack and restacks descendants", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branchA":  "main",
				"branchA2": "branchA",
				"branchB":  "main",
			})

		// Move branchA from main to branchB (upstack - moving to a sibling branch)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branchA",
			Onto:   "branchB",
		})
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "branchB", s.Engine.GetParent("branchA"))
		require.Contains(t, s.Engine.GetChildren("branchB"), "branchA")
		require.NotContains(t, s.Engine.GetChildren("main"), "branchA")

		// Verify branchA2 still has branchA as parent (descendant relationship preserved)
		require.Equal(t, "branchA", s.Engine.GetParent("branchA2"))
	})

	t.Run("moves branch across different stack trees", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branchA1": "main",
				"branchA2": "branchA1",
				"branchB1": "main",
				"branchB2": "branchB1",
			})

		// Move branchA2 from branchA1 to branchB1 (across stacks)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branchA2",
			Onto:   "branchB1",
		})
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "branchB1", s.Engine.GetParent("branchA2"))
		require.Contains(t, s.Engine.GetChildren("branchB1"), "branchA2")
		require.NotContains(t, s.Engine.GetChildren("branchA1"), "branchA2")
	})

	t.Run("defaults source to current branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Switch to branch2
		s.Checkout("branch2")

		// Move without specifying source (should use current branch)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "", // Empty means use current branch
			Onto:   "main",
		})
		require.NoError(t, err)

		// Verify branch2 was moved
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
	})

	t.Run("prevents moving trunk branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "main",
			Onto:   "main", // Even if onto is same, should fail earlier
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move trunk branch")
	})

	t.Run("prevents moving onto itself", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "branch1",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move branch onto itself")
	})

	t.Run("prevents moving onto descendant (cycle detection)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		// Try to move branch1 onto branch3 (which is a descendant of branch1)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "branch3",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move")
		require.Contains(t, err.Error(), "onto its own descendant")
	})

	t.Run("fails when source branch is not tracked", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("untracked").
			Checkout("main")

		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "untracked",
			Onto:   "main",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("fails when onto branch does not exist", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "nonexistent",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when not on branch and no source specified", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Detach HEAD
		s.RunGit("checkout", "HEAD~0").Rebuild()

		// Try to move without specifying source - should fail because we're in detached HEAD
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "",
			Onto:   "main",
		})
		require.Error(t, err)
		// The error should be either "not on a branch" or "cannot move trunk branch"
		errMsg := err.Error()
		require.True(t,
			strings.Contains(errMsg, "not on a branch") || strings.Contains(errMsg, "cannot move trunk branch"),
			"error should mention not on a branch or trunk: %s", errMsg)
	})

	t.Run("allows moving onto untracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			}).
			CreateBranch("untracked").
			Checkout("main")

		// Move branch1 onto untracked branch (should work)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "untracked",
		})
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "untracked", s.Engine.GetParent("branch1"))
	})

	t.Run("restacks all descendants after move", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		// Get initial revisions
		branch1RevBefore, _ := s.Scene.Repo.GetRevision("branch1")
		branch2RevBefore, _ := s.Scene.Repo.GetRevision("branch2")
		branch3RevBefore, _ := s.Scene.Repo.GetRevision("branch3")

		// Make a change to main to force restacking
		s.Checkout("main").
			Commit("new main change")

		// Move branch1 to main (which now has new commits)
		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "main",
		})
		require.NoError(t, err)

		// Verify branches were restacked (revisions should have changed)
		branch1RevAfter, _ := s.Scene.Repo.GetRevision("branch1")
		branch2RevAfter, _ := s.Scene.Repo.GetRevision("branch2")
		branch3RevAfter, _ := s.Scene.Repo.GetRevision("branch3")

		// Revisions should be different (branches were rebased)
		require.NotEqual(t, branch1RevBefore, branch1RevAfter)
		require.NotEqual(t, branch2RevBefore, branch2RevAfter)
		require.NotEqual(t, branch3RevBefore, branch3RevAfter)

		// Verify all branches are still fixed (properly restacked)
		s.ExpectBranchFixed("branch1").
			ExpectBranchFixed("branch2").
			ExpectBranchFixed("branch3")
	})

	t.Run("fails when onto is not specified", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		err := actions.MoveAction(s.Context, actions.MoveOptions{
			Source: "branch1",
			Onto:   "", // Empty onto
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "onto branch must be specified")
	})
}
