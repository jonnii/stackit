package actions_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestFoldAction(t *testing.T) {
	t.Run("folds branch into parent", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit on main
			if err := s.Repo.CreateChangeAndCommit("initial", "init"); err != nil {
				return err
			}
			// Create branch1 with a commit
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

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Fold branch2 into branch1
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch2 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch2")

		// Verify we're on branch1
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)

		// Verify branch1 contains both commits by checking log
		logOutput, err := scene.Repo.RunGitCommandAndGetOutput("log", "--oneline", "main..branch1")
		require.NoError(t, err)
		require.Contains(t, logOutput, "branch1 change")
		require.Contains(t, logOutput, "branch2 change")
	})

	t.Run("reparents children when folding branch", func(t *testing.T) {
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
			if err := s.Repo.CreateChangeAndCommit("branch2 change", "file2"); err != nil {
				return err
			}
			// Create branch3 as child of branch2
			if err := s.Repo.CreateAndCheckoutBranch("branch3"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch3 change", "file3")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Fold branch2 into branch1
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch3's parent is now branch1
		parent := eng.GetParent("branch3")
		require.Equal(t, "branch1", parent)

		// Verify branch2 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch2")
	})

	t.Run("folds with --keep flag", func(t *testing.T) {
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

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Fold branch1 into branch2 with --keep
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: true})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")

		// Verify we're on branch2
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch2", currentBranch)

		// Verify branch2's parent is now main
		parent := eng.GetParent("branch2")
		require.Equal(t, "main", parent)

		// Verify branch2 contains both commits by checking log
		logOutput, err := scene.Repo.RunGitCommandAndGetOutput("log", "--oneline", "main..branch2")
		require.NoError(t, err)
		require.Contains(t, logOutput, "branch1 change")
		require.Contains(t, logOutput, "branch2 change")
	})

	t.Run("folds with --keep and reparents siblings", func(t *testing.T) {
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
			if err := s.Repo.CreateChangeAndCommit("branch2 change", "file2"); err != nil {
				return err
			}
			// Create branch3 as another child of branch1
			if err := s.Repo.CheckoutBranch("branch1"); err != nil {
				return err
			}
			if err := s.Repo.CreateAndCheckoutBranch("branch3"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch3 change", "file3")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch1")
		require.NoError(t, err)

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Fold branch1 into branch2 with --keep
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: true})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch1 is deleted
		branches, err := git.GetAllBranchNames()
		require.NoError(t, err)
		require.NotContains(t, branches, "branch1")

		// Verify branch3's parent is now branch2
		parent := eng.GetParent("branch3")
		require.Equal(t, "branch2", parent)

		// Verify branch2's parent is now main
		parent = eng.GetParent("branch2")
		require.Equal(t, "main", parent)
	})

	t.Run("fails when trying to fold trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to fold trunk (main)
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold trunk branch")
	})

	t.Run("fails when trying to fold untracked branch", func(t *testing.T) {
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

		// Try to fold untracked branch
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold untracked branch")
	})

	t.Run("fails when trying to fold into trunk with --keep", func(t *testing.T) {
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

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to fold into trunk with --keep
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: true})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot fold into trunk with --keep")
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

		// Try to fold with dirty tree
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "uncommitted changes")
	})

	t.Run("returns clear error message on merge conflict", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			if err := s.Repo.CreateChangeAndCommit("initial content\n", "conflict.txt"); err != nil {
				return err
			}
			if err := s.Repo.CreateAndCheckoutBranch("branch1"); err != nil {
				return err
			}
			// branch2 branches from branch1
			if err := s.Repo.CreateAndCheckoutBranch("branch2"); err != nil {
				return err
			}

			// parent (branch1) adds a conflicting change
			if err := s.Repo.CheckoutBranch("branch1"); err != nil {
				return err
			}
			if err := s.Repo.CreateChangeAndCommit("branch1 content\n", "conflict.txt"); err != nil {
				return err
			}

			// child (branch2) adds a conflicting change
			if err := s.Repo.CheckoutBranch("branch2"); err != nil {
				return err
			}
			return s.Repo.CreateChangeAndCommit("branch2 content\n", "conflict.txt")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches (main -> branch1 -> branch2)
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Fold branch2 into branch1 - should conflict
		err = actions.FoldAction(ctx, actions.FoldOptions{Keep: false})
		require.Error(t, err)
		require.Contains(t, err.Error(), "due to conflicts. Please resolve the conflicts and run 'git commit'")
	})
}
