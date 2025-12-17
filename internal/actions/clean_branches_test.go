package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestCleanBranches(t *testing.T) {
	t.Run("deletes merged branch and updates children", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine and track branch1 first
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Now create branch2
		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		// Merge branch1 into main
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.MergeBranch("main", "branch1")
		require.NoError(t, err)

		// Rebuild to see branch2
		err = eng.Rebuild("main")
		require.NoError(t, err)

		// Track branch2
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Mark branch1 as merged via PR info
		prNumber := 1
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "MERGED",
			Base:   "main",
		}
		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		result, err := actions.CleanBranches(ctx, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// branch1 should be deleted
		require.False(t, eng.IsBranchTracked("branch1"))

		// branch2 should have new parent (main)
		require.Equal(t, "main", eng.GetParent("branch2"))
		require.Contains(t, result.BranchesWithNewParents, "branch2")
	})

	t.Run("handles multiple children when parent is deleted", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2, branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine and track branch1 first
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Create branch2
		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		// Create branch3
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)

		// Merge branch1
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.MergeBranch("main", "branch1")
		require.NoError(t, err)

		// Rebuild to see branch2 and branch3
		err = eng.Rebuild("main")
		require.NoError(t, err)

		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch("branch3", "branch1")
		require.NoError(t, err)

		// Mark branch1 as merged
		prNumber := 1
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "MERGED",
		}
		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		result, err := actions.CleanBranches(ctx, actions.CleanBranchesOptions{
			Force: true,
		})
		require.NoError(t, err)

		// Both children should have new parent
		require.Equal(t, "main", eng.GetParent("branch2"))
		require.Equal(t, "main", eng.GetParent("branch3"))
		require.Contains(t, result.BranchesWithNewParents, "branch2")
		require.Contains(t, result.BranchesWithNewParents, "branch3")
	})

	t.Run("does not delete branch without PR when not merged", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see branch1)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		result, err := actions.CleanBranches(ctx, actions.CleanBranchesOptions{
			Force: false,
		})
		require.NoError(t, err)

		// Branch should still exist
		require.True(t, eng.IsBranchTracked("branch1"))
		require.Empty(t, result.BranchesWithNewParents)
	})
}
