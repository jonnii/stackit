package actions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestPopAction(t *testing.T) {
	t.Run("pops branch and retains changes as staged", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit on main
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch1 with a commit
			if err := s.Repo.CreateAndCheckoutBranch("branch1"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch1 change", "file1")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branch1
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Verify we're on branch1
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)

		// Pop the branch
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.NoError(t, err)

		// Verify we're now on main
		currentBranch, err = scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "main", currentBranch)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")

		// Verify changes are staged
		hasStaged, err := git.HasStagedChanges(ctx.Context)
		require.NoError(t, err)
		require.True(t, hasStaged, "Changes should be staged after pop")
	})

	t.Run("reparents children when popping branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit on main
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateAndCheckoutBranch("branch1"); err != nil {
				return err
			}
			if err := s.Repo.CreateChangeAndCommit("branch1 change", "file1"); err != nil {
				return err
			}
			// Create branch2 as child of branch1
			if err := s.Repo.CreateAndCheckoutBranch("branch2"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch2 change", "file2")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Switch to branch1 and rebuild engine to update currentBranch
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Pop branch1
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch2's parent is now main
		parent := eng.GetParent("branch2")
		require.Equal(t, "main", parent)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")
	})

	t.Run("fails when trying to pop trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to pop trunk (main)
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot pop trunk branch")
	})

	t.Run("fails when trying to pop untracked branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create untracked branch
			return s.Repo.CreateAndCheckoutBranch("untracked")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to pop untracked branch
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot pop untracked branch")
	})

	t.Run("fails when rebase is in progress", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch1
			if err := s.Repo.CreateAndCheckoutBranch("branch1"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch1 change", "file1")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branch1
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Manually create a rebase-merge directory to simulate rebase in progress
		// This is a simpler way to test the rebase check without actually running a rebase
		rebasePath := filepath.Join(scene.Dir, ".git", "rebase-merge")
		err = os.MkdirAll(rebasePath, 0755)
		require.NoError(t, err)

		// Verify rebase is detected as in progress
		require.True(t, scene.Repo.RebaseInProgress(), "Rebase should be in progress")

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to pop during rebase
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "rebase is already in progress")
	})

	t.Run("fails when there are uncommitted changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			if err := s.Repo.CreateAndCheckoutBranch("branch1"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch1 change", "file1")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branch1 off main
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Create uncommitted change
		err = scene.Repo.CreateChange("dirty file", "dirty", true)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to pop with dirty tree
		err = actions.PopAction(ctx, actions.PopOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})
}
