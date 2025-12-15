package engine_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
)

func TestTrackBranch(t *testing.T) {
	t.Run("tracks branch with parent", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial commit
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create feature branch
		err := scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature change", "feat")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branch
		err = eng.TrackBranch("feature", "main")
		require.NoError(t, err)

		// Verify parent relationship
		parent := eng.GetParent("feature")
		require.Equal(t, "main", parent)

		// Verify children relationship
		children := eng.GetChildren("main")
		require.Contains(t, children, "feature")
	})

	t.Run("tracks branch with non-trunk parent", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create both branches first, before creating engine
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

		// Track branch1 first
		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Track branch2 (branch1 is now tracked and in the branch list)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Verify relationships
		require.Equal(t, "main", eng.GetParent("branch1"))
		require.Equal(t, "branch1", eng.GetParent("branch2"))
		require.Contains(t, eng.GetChildren("main"), "branch1")
		require.Contains(t, eng.GetChildren("branch1"), "branch2")
	})

	t.Run("fails when branch does not exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("nonexistent", "main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when parent does not exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create feature branch before creating engine
		err := scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature change", "feat")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see feature branch)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Try to track with nonexistent parent - should fail
		err = eng.TrackBranch("feature", "nonexistent")
		require.Error(t, err)
		// Error should mention the parent branch doesn't exist
		require.Contains(t, err.Error(), "parent branch nonexistent does not exist")
	})
}

func TestSetParent(t *testing.T) {
	t.Run("updates parent relationship", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create all branches first
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch("branch3", "branch1")
		require.NoError(t, err)

		// Verify initial state
		require.Equal(t, "branch1", eng.GetParent("branch2"))

		// Change parent of branch2 to main
		err = eng.SetParent("branch2", "main")
		require.NoError(t, err)

		// Verify new parent
		require.Equal(t, "main", eng.GetParent("branch2"))
		require.Contains(t, eng.GetChildren("main"), "branch2")
		require.NotContains(t, eng.GetChildren("branch1"), "branch2")
	})
}

func TestDeleteBranch(t *testing.T) {
	t.Run("deletes branch and updates children", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch structure: main -> branch1 -> branch2, branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track all branches
		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch("branch3", "branch1")
		require.NoError(t, err)

		// Verify initial state
		require.Contains(t, eng.GetChildren("branch1"), "branch2")
		require.Contains(t, eng.GetChildren("branch1"), "branch3")

		// Delete branch1
		err = eng.DeleteBranch("branch1")
		require.NoError(t, err)

		// Verify branch1 is removed
		require.False(t, eng.IsBranchTracked("branch1"))
		require.NotContains(t, eng.AllBranchNames(), "branch1")

		// Verify children now point to main
		require.Equal(t, "main", eng.GetParent("branch2"))
		require.Equal(t, "main", eng.GetParent("branch3"))
		require.Contains(t, eng.GetChildren("main"), "branch2")
		require.Contains(t, eng.GetChildren("main"), "branch3")
	})

	t.Run("fails when trying to delete trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.DeleteBranch("main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot delete trunk")
	})
}

func TestGetRelativeStack(t *testing.T) {
	t.Run("returns downstack (ancestors)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2
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

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Get downstack from branch2
		scope := engine.Scope{RecursiveParents: true}
		stack := eng.GetRelativeStack("branch2", scope)
		require.Equal(t, []string{"main", "branch1"}, stack)
	})

	t.Run("returns upstack (descendants)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2, branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch("branch3", "branch1")
		require.NoError(t, err)

		// Get upstack from branch1
		scope := engine.Scope{RecursiveChildren: true}
		stack := eng.GetRelativeStack("branch1", scope)
		require.Contains(t, stack, "branch2")
		require.Contains(t, stack, "branch3")
		require.Len(t, stack, 2)
	})

	t.Run("returns only current branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		scope := engine.Scope{IncludeCurrent: true}
		stack := eng.GetRelativeStack("branch1", scope)
		require.Equal(t, []string{"branch1"}, stack)
	})

	t.Run("returns full stack (downstack + current + upstack)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2
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

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		scope := engine.Scope{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := eng.GetRelativeStack("branch1", scope)
		require.Contains(t, stack, "main")
		require.Contains(t, stack, "branch1")
		require.Contains(t, stack, "branch2")
	})
}

func TestRestackBranch(t *testing.T) {
	t.Run("restacks branch when parent has moved", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch1
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		// Create branch2 on top of branch1
		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)

		// Add commit to main (parent moves forward)
		err = scene.Repo.CreateChangeAndCommit("main update", "main")
		require.NoError(t, err)

		// Restack branch1 (should rebase onto new main)
		result, err := eng.RestackBranch("branch1")
		require.NoError(t, err)
		require.Equal(t, engine.RestackDone, result.Result)

		// Verify branch1 is now fixed
		require.True(t, eng.IsBranchFixed("branch1"))
	})

	t.Run("returns unneeded when branch is already fixed", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Branch is already fixed (no changes to main)
		result, err := eng.RestackBranch("branch1")
		require.NoError(t, err)
		require.Equal(t, engine.RestackUnneeded, result.Result)
	})

	t.Run("returns error when branch is not tracked", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Don't track the branch
		result, err := eng.RestackBranch("branch1")
		require.Error(t, err)
		require.Equal(t, engine.RestackUnneeded, result.Result)
		require.Contains(t, err.Error(), "not tracked")
	})
}

func TestRebuild(t *testing.T) {
	t.Run("rebuilds cache from Git state", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create and track branches
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Verify initial state
		require.Contains(t, eng.AllBranchNames(), "branch1")
		require.Equal(t, "main", eng.GetParent("branch1"))

		// Create new branch externally (not tracked)
		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Rebuild should pick up new branch
		err = eng.Rebuild("main")
		require.NoError(t, err)

		// New branch should be in list
		require.Contains(t, eng.AllBranchNames(), "branch2")
		// But not tracked yet
		require.False(t, eng.IsBranchTracked("branch2"))
	})
}

func TestIsBranchTracked(t *testing.T) {
	t.Run("returns true for tracked branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		require.False(t, eng.IsBranchTracked("branch1"))

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		require.True(t, eng.IsBranchTracked("branch1"))
	})

	t.Run("returns false for untracked branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		require.False(t, eng.IsBranchTracked("branch1"))
	})
}

func TestIsTrunk(t *testing.T) {
	t.Run("returns true for trunk branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		require.True(t, eng.IsTrunk("main"))
		require.False(t, eng.IsTrunk("other"))
	})
}

func TestGetParentPrecondition(t *testing.T) {
	t.Run("returns parent when exists", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		parent := eng.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent)
	})

	t.Run("returns trunk when no parent", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Don't track branch1
		parent := eng.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent) // Should return trunk
	})
}

func TestIsMergedIntoTrunk(t *testing.T) {
	t.Run("returns false for unmerged branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		merged, err := eng.IsMergedIntoTrunk("branch1")
		require.NoError(t, err)
		require.False(t, merged)
	})
}

func TestIsBranchEmpty(t *testing.T) {
	t.Run("returns false for branch with changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		empty, err := eng.IsBranchEmpty("branch1")
		require.NoError(t, err)
		require.False(t, empty)
	})

	t.Run("returns true for branch with no changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch but don't make any changes
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		empty, err := eng.IsBranchEmpty("branch1")
		require.NoError(t, err)
		require.True(t, empty)
	})
}

func TestUpsertPrInfo(t *testing.T) {
	t.Run("creates PR info for branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		prNumber := 123
		prInfo := &engine.PrInfo{
			Number:  &prNumber,
			Title:   "Test PR",
			Body:    "Test body",
			IsDraft: false,
			State:   "OPEN",
			Base:    "main",
			URL:     "https://github.com/owner/repo/pull/123",
		}

		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Verify PR info
		retrieved, err := eng.GetPrInfo("branch1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, prNumber, *retrieved.Number)
		require.Equal(t, "Test PR", retrieved.Title)
		require.Equal(t, "Test body", retrieved.Body)
		require.False(t, retrieved.IsDraft)
	})

	t.Run("updates existing PR info", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		prNumber := 123
		prInfo := &engine.PrInfo{
			Number:  &prNumber,
			Title:   "Original Title",
			IsDraft: false,
		}

		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Update PR info
		prInfo.Title = "Updated Title"
		prInfo.Body = "Updated body"
		err = eng.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Verify updated PR info
		retrieved, err := eng.GetPrInfo("branch1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, "Updated Title", retrieved.Title)
		require.Equal(t, "Updated body", retrieved.Body)
	})
}

func TestGetRelativeStackUpstack(t *testing.T) {
	t.Run("returns all descendants", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create: main -> branch1 -> branch2, branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch("branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch("branch3", "branch1")
		require.NoError(t, err)

		upstack := eng.GetRelativeStackUpstack("branch1")
		require.Contains(t, upstack, "branch2")
		require.Contains(t, upstack, "branch3")
		require.Len(t, upstack, 2)
		require.NotContains(t, upstack, "branch1") // Should not include starting branch
	})
}

func TestReset(t *testing.T) {
	t.Run("resets engine with new trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Reset with same trunk
		err = eng.Reset("main")
		require.NoError(t, err)

		// Branch should still exist but not be tracked
		require.Contains(t, eng.AllBranchNames(), "branch1")
		require.False(t, eng.IsBranchTracked("branch1"))
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent reads safely", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		// Concurrent reads should be safe
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = eng.GetParent("branch1")
				_ = eng.GetChildren("main")
				_ = eng.IsBranchTracked("branch1")
				_ = eng.AllBranchNames()
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("handles branch with no parent gracefully", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Branch exists but not tracked
		parent := eng.GetParent("branch1")
		require.Empty(t, parent)

		// GetParentPrecondition should return trunk
		parent = eng.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent)
	})

	t.Run("handles multiple children correctly", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create multiple branches from main
		branchNames := []string{"branch1", "branch2", "branch3", "branch4", "branch5"}
		for _, branchName := range branchNames {
			err := scene.Repo.CreateAndCheckoutBranch(branchName)
			require.NoError(t, err)
			err = scene.Repo.CreateChangeAndCommit(branchName+" change", branchName)
			require.NoError(t, err)
			err = scene.Repo.CheckoutBranch("main")
			require.NoError(t, err)
		}

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track all branches
		for _, branchName := range branchNames {
			err = eng.TrackBranch(branchName, "main")
			require.NoError(t, err)
		}

		// Verify all are children of main
		children := eng.GetChildren("main")
		require.Len(t, children, 5)
	})
}
