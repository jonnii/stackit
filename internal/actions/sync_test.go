package actions_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestSyncAction(t *testing.T) {
	t.Run("syncs when trunk is up to date", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.SyncAction(ctx, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
		})
		require.NoError(t, err)
	})

	t.Run("fails when there are uncommitted changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create uncommitted change
		err := scene.Repo.CreateChange("unstaged", "test", true)
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.SyncAction(ctx, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})

	t.Run("syncs with restack flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.SyncAction(ctx, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed (even if no restacking needed)
		require.NoError(t, err)
	})

	t.Run("restacks branches in topological order (parents before children)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create stack: main -> branch1 -> branch2 -> branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.SyncAction(ctx, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed - branches should be restacked in correct order
		require.NoError(t, err)
	})

	t.Run("restacks branching stacks in topological order", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branching stack structure:
		// main
		// ├── stackA
		// │   ├── stackA-child1
		// │   └── stackA-child2
		// └── stackB
		//     └── stackB-child1

		// First stack: main -> stackA -> stackA-child1
		err := scene.Repo.CreateAndCheckoutBranch("stackA")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("stackA change", "sA")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("stackA-child1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("stackA-child1 change", "sAc1")
		require.NoError(t, err)

		// Branch off stackA for stackA-child2
		err = scene.Repo.CheckoutBranch("stackA")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("stackA-child2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("stackA-child2 change", "sAc2")
		require.NoError(t, err)

		// Second stack: main -> stackB -> stackB-child1
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("stackB")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("stackB change", "sB")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("stackB-child1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("stackB-child1 change", "sBc1")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track all branches
		err = eng.TrackBranch(context.Background(), "stackA", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "stackA-child1", "stackA")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "stackA-child2", "stackA")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "stackB", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "stackB-child1", "stackB")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.SyncAction(ctx, actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
		})
		// Should succeed - branches should be restacked with parents before children
		// Order should be something like: stackA, stackA-child1, stackA-child2, stackB, stackB-child1
		// (exact sibling order may vary, but parents must come before their children)
		require.NoError(t, err)

		// Verify all branches still exist and are properly tracked
		require.True(t, eng.IsBranchTracked("stackA"))
		require.True(t, eng.IsBranchTracked("stackA-child1"))
		require.True(t, eng.IsBranchTracked("stackA-child2"))
		require.True(t, eng.IsBranchTracked("stackB"))
		require.True(t, eng.IsBranchTracked("stackB-child1"))

		// Verify parent relationships are preserved
		require.Equal(t, "main", eng.GetParent("stackA"))
		require.Equal(t, "stackA", eng.GetParent("stackA-child1"))
		require.Equal(t, "stackA", eng.GetParent("stackA-child2"))
		require.Equal(t, "main", eng.GetParent("stackB"))
		require.Equal(t, "stackB", eng.GetParent("stackB-child1"))
	})
}
