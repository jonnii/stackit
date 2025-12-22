package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestSyncAction(t *testing.T) {
	t.Run("syncs when trunk is up to date", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		err := actions.SyncAction(s.Context, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
		})
		require.NoError(t, err)
	})

	t.Run("fails when there are uncommitted changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithUncommittedChange("unstaged")

		err := actions.SyncAction(s.Context, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})

	t.Run("syncs with restack flag", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
			})

		err := actions.SyncAction(s.Context, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed (even if no restacking needed)
		require.NoError(t, err)
	})

	t.Run("restacks branches in topological order (parents before children)", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		err := actions.SyncAction(s.Context, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed - branches should be restacked in correct order
		require.NoError(t, err)
	})

	t.Run("restacks branching stacks in topological order", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"stackA":        "main",
				"stackA-child1": "stackA",
				"stackA-child2": "stackA",
				"stackB":        "main",
				"stackB-child1": "stackB",
			})

		err := actions.SyncAction(s.Context, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed - branches should be restacked with parents before children
		require.NoError(t, err)

		// Verify all branches still exist and are properly tracked
		s.ExpectStackStructure(map[string]string{
			"stackA":        "main",
			"stackA-child1": "stackA",
			"stackA-child2": "stackA",
			"stackB":        "main",
			"stackB-child1": "stackB",
		})
	})

	t.Run("restacks multiple deep subtrees correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, nil).
			WithStack(map[string]string{
				"P":   "main",
				"C1":  "P",
				"GC1": "C1",
				"C2":  "P",
				"GC2": "C2",
			})

		// Modify P to trigger restacking of all descendants
		s.Checkout("P").
			Commit("P updated")

		// Refresh engine
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		err = actions.SyncAction(s.Context, actions.SyncOptions{
			All:     true,
			Restack: true,
		})
		require.NoError(t, err)

		// Verify all branches are fixed
		s.ExpectBranchFixed("C1").
			ExpectBranchFixed("GC1").
			ExpectBranchFixed("C2").
			ExpectBranchFixed("GC2")
	})

	t.Run("partial success in branching restack (one child succeeds, one fails)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create: main -> P -> [C1, C2]
		s.CreateBranch("P").
			Commit("P change").
			TrackBranch("P", "main")

		// C1 will restack successfully
		s.Checkout("P").
			CreateBranch("C1").
			Commit("C1 change").
			TrackBranch("C1", "P")

		// C2 will have a conflict
		s.Checkout("P").
			CreateBranch("C2")
		err := s.Scene.Repo.CreateChangeAndCommit("initial content", "conflict")
		require.NoError(t, err)
		s.TrackBranch("C2", "P")

		// Modify P with a change that conflicts with C2 but not C1
		s.Checkout("P")
		err = s.Scene.Repo.CreateChangeAndCommit("conflicting content", "conflict")
		require.NoError(t, err)

		// Refresh engine
		err = s.Engine.Rebuild("main")
		require.NoError(t, err)

		s.Checkout("P")

		err = actions.SyncAction(s.Context, actions.SyncOptions{
			All:     true,
			Restack: true,
		})

		// Should error due to conflict in C2
		require.Error(t, err)
		require.Contains(t, err.Error(), "conflict")

		// C1 should still have been restacked successfully
		s.ExpectBranchFixed("C1")
		// C2 should NOT be fixed
		s.ExpectBranchNotFixed("C2")
	})
}
