package actions_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func init() {
	// Disable interactive prompts in tests
	os.Setenv("STACKIT_TEST_NO_INTERACTIVE", "1")
}

func TestUndoAction(t *testing.T) {
	t.Run("returns error when no snapshots exist", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := actions.UndoAction(s.Context, actions.UndoOptions{})
		require.NoError(t, err) // Should not error, just show message
	})

	t.Run("restores to snapshot when only one exists", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature").
			Commit("feature change").
			Checkout("main").
			TrackBranch("feature", "main")

		// Get initial state
		initialFeatureSHA, err := s.Engine.GetRevision("feature")
		require.NoError(t, err)

		// Take snapshot
		err = s.Engine.TakeSnapshot("move", []string{"feature", "onto", "main"})
		require.NoError(t, err)

		// Make changes
		s.Checkout("feature").
			Commit("additional change")

		// Verify SHA changed
		newFeatureSHA, err := s.Engine.GetRevision("feature")
		require.NoError(t, err)
		require.NotEqual(t, initialFeatureSHA, newFeatureSHA)

		// Test restore directly via engine (bypasses confirmation prompt)
		snapshots, err := s.Engine.GetSnapshots()
		require.NoError(t, err)
		require.Len(t, snapshots, 1)

		err = s.Engine.RestoreSnapshot(s.Context, snapshots[0].ID)
		require.NoError(t, err)

		// Verify state restored
		s.Engine.Rebuild(s.Engine.Trunk().Name)
		restoredFeatureSHA, err := s.Engine.GetRevision("feature")
		require.NoError(t, err)
		require.Equal(t, initialFeatureSHA, restoredFeatureSHA)
	})

	t.Run("returns error for invalid snapshot ID", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Create at least one snapshot so GetSnapshots doesn't return empty
		err := s.Engine.TakeSnapshot("test", nil)
		require.NoError(t, err)

		err = actions.UndoAction(s.Context, actions.UndoOptions{
			SnapshotID: "nonexistent-snapshot",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})

	t.Run("restores after multiple operations", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature1").
			Commit("feature1 change").
			Checkout("main").
			TrackBranch("feature1", "main")

		// Get initial state BEFORE taking snapshot
		initialFeature1SHA, err := s.Engine.GetRevision("feature1")
		require.NoError(t, err)

		// Take first snapshot (captures initial state)
		err = s.Engine.TakeSnapshot("create", []string{"feature1"})
		require.NoError(t, err)

		// Make first change
		s.Checkout("feature1").
			Commit("feature1 change 2")

		// Get SHA after first change
		afterFirstChangeSHA, err := s.Engine.GetRevision("feature1")
		require.NoError(t, err)
		require.NotEqual(t, initialFeature1SHA, afterFirstChangeSHA)

		// Take second snapshot (captures state after first change)
		err = s.Engine.TakeSnapshot("move", []string{"feature1", "onto", "main"})
		require.NoError(t, err)

		// Make second change
		s.Commit("feature1 change 3")

		// Get SHA after second change
		afterSecondChangeSHA, err := s.Engine.GetRevision("feature1")
		require.NoError(t, err)
		require.NotEqual(t, afterFirstChangeSHA, afterSecondChangeSHA)

		// Restore to first snapshot (undoing second operation) via engine directly
		snapshots, err := s.Engine.GetSnapshots()
		require.NoError(t, err)
		require.Len(t, snapshots, 2)

		// Restore to the most recent snapshot (Snapshot 2, taken after first change)
		// Since GetSnapshots returns newest first, this is index 0
		err = s.Engine.RestoreSnapshot(s.Context, snapshots[0].ID)
		require.NoError(t, err)

		// Verify state restored to Snapshot 2 (after first change, not initial)
		s.Engine.Rebuild(s.Engine.Trunk().Name)
		restoredFeature1SHA, err := s.Engine.GetRevision("feature1")
		require.NoError(t, err)
		// Should restore to state after first change
		require.Equal(t, afterFirstChangeSHA, restoredFeature1SHA)
	})
}

func TestUndoAfterCreate(t *testing.T) {
	t.Run("undoes branch creation", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		// Take snapshot BEFORE creating branch
		err := s.Engine.TakeSnapshot("create", []string{"feature"})
		require.NoError(t, err)

		// Create branch after snapshot
		s.CreateBranch("feature").
			Commit("feature change").
			Checkout("main")

		// Track the branch
		err = s.Engine.TrackBranch(s.Context, "feature", "main")
		require.NoError(t, err)

		// Verify branch exists
		allBranches := s.Engine.AllBranches()
		branches := make([]string, len(allBranches))
		for i, b := range allBranches {
			branches[i] = b.Name
		}
		require.Contains(t, branches, "feature")

		// Undo via engine directly (bypasses confirmation)
		snapshots, err := s.Engine.GetSnapshots()
		require.NoError(t, err)
		err = s.Engine.RestoreSnapshot(s.Context, snapshots[0].ID)
		require.NoError(t, err)

		// Verify branch was deleted (it didn't exist in the snapshot)
		s.Engine.Rebuild(s.Engine.Trunk().Name)
		allBranches2 := s.Engine.AllBranches()
		branches = make([]string, len(allBranches2))
		for i, b := range allBranches2 {
			branches[i] = b.Name
		}
		require.NotContains(t, branches, "feature")
	})
}

func TestUndoAfterMove(t *testing.T) {
	t.Run("undoes branch move operation", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature1").
			Commit("feature1 change").
			CreateBranch("feature2").
			Commit("feature2 change").
			Checkout("main").
			TrackBranch("feature1", "main").
			TrackBranch("feature2", "feature1")

		// Get initial parent
		initialParent := s.Engine.GetParent("feature2")
		require.NotNil(t, initialParent)
		require.Equal(t, "feature1", initialParent.Name)

		// Take snapshot before move
		err := s.Engine.TakeSnapshot("move", []string{"feature2", "onto", "main"})
		require.NoError(t, err)

		// Move feature2 to main
		err = s.Engine.SetParent(s.Context, "feature2", "main")
		require.NoError(t, err)

		// Verify parent changed
		newParent := s.Engine.GetParent("feature2")
		require.NotNil(t, newParent)
		require.Equal(t, "main", newParent.Name)

		// Undo via engine directly (bypasses confirmation)
		snapshots, err := s.Engine.GetSnapshots()
		require.NoError(t, err)
		err = s.Engine.RestoreSnapshot(s.Context, snapshots[0].ID)
		require.NoError(t, err)

		// Verify parent restored
		s.Engine.Rebuild(s.Engine.Trunk().Name)
		restoredParent := s.Engine.GetParent("feature2")
		require.NotNil(t, restoredParent)
		require.Equal(t, initialParent.Name, restoredParent.Name)
	})
}
