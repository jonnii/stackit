package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/testhelpers"
)

func TestSwitchBranchAction(t *testing.T) {
	t.Run("traverses downward to bottom branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2
		// Create all branches first
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see both branches)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Create context
		splog := tui.NewSplog()
		ctx := &runtime.Context{
			Engine: eng,
			Splog:  splog,
		}

		// Switch to branch2 (top of stack)
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Update context with new engine
		ctx.Engine = eng

		// Verify parent relationships are correct
		require.Equal(t, "main", eng.GetParent("branch1"), "branch1 should have main as parent")
		require.Equal(t, "branch1", eng.GetParent("branch2"), "branch2 should have branch1 as parent")

		// Traverse downward should go to branch1 (first branch from trunk)
		err = actions.SwitchBranchAction(actions.DirectionBottom, ctx)
		require.NoError(t, err)

		// Should be on branch1
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})

	t.Run("traverses upward to top branch", func(t *testing.T) {
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
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Rebuild to see branch2
		err = eng.Rebuild("main")
		require.NoError(t, err)

		// Track branch2
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Create context
		splog := tui.NewSplog()
		ctx := &runtime.Context{
			Engine: eng,
			Splog:  splog,
		}

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Update context with new engine
		ctx.Engine = eng

		// Traverse upward should go to branch2 (top of stack)
		err = actions.SwitchBranchAction(actions.DirectionTop, ctx)
		require.NoError(t, err)

		// Should be on branch2
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch2", currentBranch)
	})

	t.Run("returns error when not on a branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Detach HEAD
		err := scene.Repo.RunGitCommand("checkout", "HEAD~0")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		splog := tui.NewSplog()
		ctx := &runtime.Context{
			Engine: eng,
			Splog:  splog,
		}

		err = actions.SwitchBranchAction(actions.DirectionBottom, ctx)
		require.Error(t, err)
	})

	t.Run("stays on branch when already at bottom", func(t *testing.T) {
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

		splog := tui.NewSplog()
		ctx := &runtime.Context{
			Engine: eng,
			Splog:  splog,
		}

		// Switch to branch1 (bottom of stack)
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Update context with new engine
		ctx.Engine = eng

		// Already on branch1 (bottom of stack)
		err = actions.SwitchBranchAction(actions.DirectionBottom, ctx)
		require.NoError(t, err)

		// Should still be on branch1
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})

	t.Run("stays on branch when already at top", func(t *testing.T) {
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

		// Switch to branch1 (top of stack)
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		splog := tui.NewSplog()
		ctx := &runtime.Context{
			Engine: eng,
			Splog:  splog,
		}

		// Already at top
		err = actions.SwitchBranchAction(actions.DirectionTop, ctx)
		require.NoError(t, err)

		// Should still be on branch1
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch1", currentBranch)
	})
}
