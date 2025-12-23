package engine_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestTrackBranch(t *testing.T) {
	t.Run("tracks branch with parent", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create feature branch
		s.CreateBranch("feature").
			Commit("feature change")
		s.Checkout("main")

		// Track branch
		err := s.Engine.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Verify parent relationship
		parent := s.Engine.GetParent("feature")
		require.Equal(t, "main", parent)

		// Verify children relationship
		children := s.Engine.GetChildren("main")
		require.Contains(t, children, "feature")
	})

	t.Run("tracks branch with non-trunk parent", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create both branches first
		s.CreateBranch("branch1").
			Commit("branch1 change").
			CreateBranch("branch2").
			Commit("branch2 change").
			Checkout("main")

		// Track branch1 first
		err := s.Engine.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Track branch2 (branch1 is now tracked and in the branch list)
		err = s.Engine.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Verify relationships
		require.Equal(t, "main", s.Engine.GetParent("branch1"))
		require.Equal(t, "branch1", s.Engine.GetParent("branch2"))
		require.Contains(t, s.Engine.GetChildren("main"), "branch1")
		require.Contains(t, s.Engine.GetChildren("branch1"), "branch2")
	})

	t.Run("fails when branch does not exist", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		err := s.Engine.TrackBranch(context.Background(), "nonexistent", "main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when parent does not exist", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create feature branch
		s.CreateBranch("feature").
			Commit("feature change").
			Checkout("main")

		// Try to track with nonexistent parent - should fail
		err := s.Engine.TrackBranch(context.Background(), "feature", "nonexistent")
		require.Error(t, err)
		require.Contains(t, err.Error(), "parent branch nonexistent does not exist")
	})
}

func TestSetParent(t *testing.T) {
	t.Run("updates parent relationship", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create all branches first
		s.CreateBranch("branch1").
			Commit("branch1 change").
			CreateBranch("branch2").
			Commit("branch2 change").
			Checkout("branch1").
			CreateBranch("branch3").
			Commit("branch3 change").
			Checkout("main")

		// Track branches
		err := s.Engine.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = s.Engine.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = s.Engine.TrackBranch(context.Background(), "branch3", "branch1")
		require.NoError(t, err)

		// Verify initial state
		require.Equal(t, "branch1", s.Engine.GetParent("branch2"))

		// Change parent of branch2 to main
		err = s.Engine.SetParent(context.Background(), "branch2", "main")
		require.NoError(t, err)

		// Verify new parent
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
		require.Contains(t, s.Engine.GetChildren("main"), "branch2")
		require.NotContains(t, s.Engine.GetChildren("branch1"), "branch2")
	})
}

func TestDeleteBranch(t *testing.T) {
	t.Run("deletes branch and updates children", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create branch structure: main -> branch1 -> branch2, branch3
		s.CreateBranch("branch1").
			Commit("branch1 change").
			CreateBranch("branch2").
			Commit("branch2 change").
			Checkout("branch1").
			CreateBranch("branch3").
			Commit("branch3 change").
			Checkout("main")

		// Track all branches
		err := s.Engine.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = s.Engine.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = s.Engine.TrackBranch(context.Background(), "branch3", "branch1")
		require.NoError(t, err)

		// Verify initial state
		require.Contains(t, s.Engine.GetChildren("branch1"), "branch2")
		require.Contains(t, s.Engine.GetChildren("branch1"), "branch3")

		// Delete branch1
		err = s.Engine.DeleteBranch(context.Background(), "branch1")
		require.NoError(t, err)

		// Verify branch1 is removed
		require.False(t, s.Engine.IsBranchTracked("branch1"))
		require.NotContains(t, s.Engine.AllBranchNames(), "branch1")

		// Verify children now point to main
		require.Equal(t, "main", s.Engine.GetParent("branch2"))
		require.Equal(t, "main", s.Engine.GetParent("branch3"))
		require.Contains(t, s.Engine.GetChildren("main"), "branch2")
		require.Contains(t, s.Engine.GetChildren("main"), "branch3")
	})

	t.Run("deletes branch with multiple siblings and children", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"P":   "main",
				"C1":  "P",
				"GC1": "C1",
				"C2":  "P",
				"C3":  "P",
				"GC3": "C3",
			})

		// Verify initial parent of children is P
		require.Equal(t, "P", s.Engine.GetParent("C1"))
		require.Equal(t, "P", s.Engine.GetParent("C2"))
		require.Equal(t, "P", s.Engine.GetParent("C3"))

		// Delete P
		err := s.Engine.DeleteBranch(context.Background(), "P")
		require.NoError(t, err)

		// Verify P is removed
		require.False(t, s.Engine.IsBranchTracked("P"))

		// Verify all direct children of P now point to main
		require.Equal(t, "main", s.Engine.GetParent("C1"))
		require.Equal(t, "main", s.Engine.GetParent("C2"))
		require.Equal(t, "main", s.Engine.GetParent("C3"))

		// Verify grandchildren still point to their parents
		require.Equal(t, "C1", s.Engine.GetParent("GC1"))
		require.Equal(t, "C3", s.Engine.GetParent("GC3"))

		// Verify main's children list contains C1, C2, C3
		mainChildren := s.Engine.GetChildren("main")
		require.Contains(t, mainChildren, "C1")
		require.Contains(t, mainChildren, "C2")
		require.Contains(t, mainChildren, "C3")
	})

	t.Run("fails when trying to delete trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		err := s.Engine.DeleteBranch(context.Background(), "main")
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot delete trunk")
	})
}

func TestGetRelativeStack(t *testing.T) {
	t.Run("returns downstack (ancestors) excluding trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Get downstack from branch2 - should NOT include trunk (main)
		scope := engine.Scope{RecursiveParents: true}
		stack := s.Engine.GetRelativeStack("branch2", scope)
		require.Equal(t, []string{"branch1"}, stack)
		require.NotContains(t, stack, "main", "trunk should not be included in ancestors")
	})

	t.Run("returns upstack (descendants)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch1",
			})

		// Get upstack from branch1
		scope := engine.Scope{RecursiveChildren: true}
		stack := s.Engine.GetRelativeStack("branch1", scope)
		require.Contains(t, stack, "branch2")
		require.Contains(t, stack, "branch3")
		require.Len(t, stack, 2)
	})

	t.Run("returns only current branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		scope := engine.Scope{IncludeCurrent: true}
		stack := s.Engine.GetRelativeStack("branch1", scope)
		require.Equal(t, []string{"branch1"}, stack)
	})

	t.Run("returns full stack (downstack + current + upstack) excluding trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch2",
			})

		// Get full stack from branch2 - should NOT include trunk (main)
		scope := engine.Scope{
			RecursiveParents:  true,
			IncludeCurrent:    true,
			RecursiveChildren: true,
		}
		stack := s.Engine.GetRelativeStack("branch2", scope)
		require.NotContains(t, stack, "main", "trunk should not be included in ancestors")
		require.Contains(t, stack, "branch1")
		require.Contains(t, stack, "branch2")
		require.Contains(t, stack, "branch3")
		require.Len(t, stack, 3)
	})

	t.Run("returns branching stacks in DFS order (parents before children)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"stackA":       "main",
				"stackA-child": "stackA",
				"stackB":       "main",
				"stackB-child": "stackB",
			})

		// Get all descendants from trunk
		scope := engine.Scope{RecursiveChildren: true}
		stack := s.Engine.GetRelativeStack("main", scope)

		// Should have all 4 branches
		require.Len(t, stack, 4)
		require.Contains(t, stack, "stackA")
		require.Contains(t, stack, "stackA-child")
		require.Contains(t, stack, "stackB")
		require.Contains(t, stack, "stackB-child")

		// Verify topological order: parents must come before their children
		stackAIdx := indexOf(stack, "stackA")
		stackAChildIdx := indexOf(stack, "stackA-child")
		stackBIdx := indexOf(stack, "stackB")
		stackBChildIdx := indexOf(stack, "stackB-child")

		require.Less(t, stackAIdx, stackAChildIdx, "stackA should come before stackA-child")
		require.Less(t, stackBIdx, stackBChildIdx, "stackB should come before stackB-child")
	})
}

// indexOf returns the index of item in slice, or -1 if not found
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

func TestRestackBranch(t *testing.T) {
	t.Run("restacks branch when parent has moved", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
			})

		// Add commit to main (parent moves forward)
		s.Checkout("main").
			Commit("main update")

		// Restack branch1 (should rebase onto new main)
		result, err := s.Engine.RestackBranch(context.Background(), "branch1")
		require.NoError(t, err)
		require.Equal(t, engine.RestackDone, result.Result)

		// Verify branch1 is now fixed
		require.True(t, s.Engine.IsBranchFixed("branch1"))
	})

	t.Run("returns unneeded when branch is already fixed", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Branch is already fixed (no changes to main)
		result, err := s.Engine.RestackBranch(context.Background(), "branch1")
		require.NoError(t, err)
		require.Equal(t, engine.RestackUnneeded, result.Result)
	})

	t.Run("auto-tracks branch when branch is not tracked", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change")

		// Don't track the branch explicitly
		// RestackBranch should auto-discover parent (main) and succeed
		// In this case, main is still at the fork point, so FindMostRecentTrackedAncestors finds it.
		result, err := s.Engine.RestackBranch(context.Background(), "branch1")
		require.NoError(t, err)
		require.Equal(t, engine.RestackUnneeded, result.Result)

		// Verify it is now tracked
		require.True(t, s.Engine.IsBranchTracked("branch1"))
		require.Equal(t, "main", s.Engine.GetParent("branch1"))
	})
}

func TestRebuild(t *testing.T) {
	t.Run("rebuilds cache from Git state", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Verify initial state
		require.Contains(t, s.Engine.AllBranchNames(), "branch1")
		require.Equal(t, "main", s.Engine.GetParent("branch1"))

		// Create new branch externally (not tracked)
		s.CreateBranch("branch2").
			Commit("branch2 change").
			Checkout("main")

		// Rebuild should pick up new branch
		err := s.Engine.Rebuild("main")
		require.NoError(t, err)

		// New branch should be in list
		require.Contains(t, s.Engine.AllBranchNames(), "branch2")
		// But not tracked yet
		require.False(t, s.Engine.IsBranchTracked("branch2"))
	})
}

func TestIsBranchTracked(t *testing.T) {
	t.Run("returns true for tracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change").
			Checkout("main")

		require.False(t, s.Engine.IsBranchTracked("branch1"))

		err := s.Engine.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		require.True(t, s.Engine.IsBranchTracked("branch1"))
	})

	t.Run("returns false for untracked branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change").
			Checkout("main")

		require.False(t, s.Engine.IsBranchTracked("branch1"))
	})
}

func TestIsTrunk(t *testing.T) {
	t.Run("returns true for trunk branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		require.True(t, s.Engine.IsTrunk("main"))
		require.False(t, s.Engine.IsTrunk("other"))
	})
}

func TestGetParentPrecondition(t *testing.T) {
	t.Run("returns parent when exists", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		parent := s.Engine.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent)
	})

	t.Run("returns trunk when no parent", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change").
			Checkout("main")

		// Don't track branch1
		parent := s.Engine.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent) // Should return trunk
	})
}

func TestIsMergedIntoTrunk(t *testing.T) {
	t.Run("returns false for unmerged branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change").
			Checkout("main")

		merged, err := s.Engine.IsMergedIntoTrunk(context.Background(), "branch1")
		require.NoError(t, err)
		require.False(t, merged)
	})
}

func TestIsBranchEmpty(t *testing.T) {
	t.Run("returns false for branch with changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			CommitChange("file1", "branch1 change").
			Checkout("main")

		empty, err := s.Engine.IsBranchEmpty(context.Background(), "branch1")
		require.NoError(t, err)
		require.False(t, empty)
	})

	t.Run("returns true for branch with no changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Checkout("main")

		empty, err := s.Engine.IsBranchEmpty(context.Background(), "branch1")
		require.NoError(t, err)
		require.True(t, empty)
	})
}

func TestUpsertPrInfo(t *testing.T) {
	t.Run("creates PR info for branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

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

		err := s.Engine.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Verify PR info
		retrieved, err := s.Engine.GetPrInfo("branch1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, prNumber, *retrieved.Number)
		require.Equal(t, "Test PR", retrieved.Title)
		require.Equal(t, "Test body", retrieved.Body)
		require.False(t, retrieved.IsDraft)
	})

	t.Run("updates existing PR info", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		prNumber := 123
		prInfo := &engine.PrInfo{
			Number:  &prNumber,
			Title:   "Original Title",
			IsDraft: false,
		}

		err := s.Engine.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Update PR info
		prInfo.Title = "Updated Title"
		prInfo.Body = "Updated body"
		err = s.Engine.UpsertPrInfo("branch1", prInfo)
		require.NoError(t, err)

		// Verify updated PR info
		retrieved, err := s.Engine.GetPrInfo("branch1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, "Updated Title", retrieved.Title)
		require.Equal(t, "Updated body", retrieved.Body)
	})
}

func TestGetRelativeStackUpstack(t *testing.T) {
	t.Run("returns all descendants", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
				"branch2": "branch1",
				"branch3": "branch1",
			})

		upstack := s.Engine.GetRelativeStackUpstack("branch1")
		require.Contains(t, upstack, "branch2")
		require.Contains(t, upstack, "branch3")
		require.Len(t, upstack, 2)
		require.NotContains(t, upstack, "branch1") // Should not include starting branch
	})
}

func TestReset(t *testing.T) {
	t.Run("resets engine with new trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Reset with same trunk
		err := s.Engine.Reset("main")
		require.NoError(t, err)

		// Branch should still exist but not be tracked
		require.Contains(t, s.Engine.AllBranchNames(), "branch1")
		require.False(t, s.Engine.IsBranchTracked("branch1"))
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent reads safely", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Concurrent reads should be safe
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = s.Engine.GetParent("branch1")
				_ = s.Engine.GetChildren("main")
				_ = s.Engine.IsBranchTracked("branch1")
				_ = s.Engine.AllBranchNames()
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestBranchMatchesRemote(t *testing.T) {
	t.Run("returns true when branch matches remote", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main first
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// Create and push a branch
		s.CreateBranch("feature").
			Commit("feature change")
		err = s.Scene.Repo.PushBranch("origin", "feature")
		require.NoError(t, err)
		s.Checkout("main")

		// Populate remote SHAs
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// Branch should match remote
		matches, err := s.Engine.BranchMatchesRemote("feature")
		require.NoError(t, err)
		require.True(t, matches, "branch should match remote after push")
	})

	t.Run("returns false when branch has local changes", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main first
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// Create and push a branch
		s.CreateBranch("feature").
			Commit("feature change")
		err = s.Scene.Repo.PushBranch("origin", "feature")
		require.NoError(t, err)

		// Make local change (not pushed)
		s.Commit("local change").
			Checkout("main")

		// Populate remote SHAs
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// Branch should NOT match remote (local has extra commit)
		matches, err := s.Engine.BranchMatchesRemote("feature")
		require.NoError(t, err)
		require.False(t, matches, "branch should not match remote with local changes")
	})

	t.Run("returns false when branch does not exist on remote", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main (but not feature)
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// Create a branch locally but don't push it
		s.CreateBranch("feature").
			Commit("feature change").
			Checkout("main")

		// Populate remote SHAs
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// Branch should NOT match remote (doesn't exist on remote)
		matches, err := s.Engine.BranchMatchesRemote("feature")
		require.NoError(t, err)
		require.False(t, matches, "branch should not match when it doesn't exist on remote")
	})

	t.Run("returns false after amend (branch diverged)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main first
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// Create and push a branch
		s.CreateBranch("feature").
			Commit("feature change")
		err = s.Scene.Repo.PushBranch("origin", "feature")
		require.NoError(t, err)

		// Amend the commit locally (simulates squash or rebase)
		err = s.Scene.Repo.CreateChangeAndAmend("amended change", "amended")
		require.NoError(t, err)

		s.Checkout("main")

		// Populate remote SHAs
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// Branch should NOT match remote (local was amended)
		matches, err := s.Engine.BranchMatchesRemote("feature")
		require.NoError(t, err)
		require.False(t, matches, "branch should not match remote after amend")
	})
}

func TestPopulateRemoteShas(t *testing.T) {
	t.Run("populates SHAs for all remote branches", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Push main
		err = s.Scene.Repo.PushBranch("origin", "main")
		require.NoError(t, err)

		// Create and push multiple branches - checkout main between each
		s.CreateBranch("feature1").
			Commit("feature1 change")
		err = s.Scene.Repo.PushBranch("origin", "feature1")
		require.NoError(t, err)
		s.Checkout("main")

		s.CreateBranch("feature2").
			Commit("feature2 change")
		err = s.Scene.Repo.PushBranch("origin", "feature2")
		require.NoError(t, err)
		s.Checkout("main")

		// Populate remote SHAs
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// All branches should match remote
		for _, branch := range []string{"main", "feature1", "feature2"} {
			matches, err := s.Engine.BranchMatchesRemote(branch)
			require.NoError(t, err)
			require.True(t, matches, "branch %s should match remote", branch)
		}
	})

	t.Run("handles empty remote gracefully", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create a bare remote but don't push anything
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Populate should not fail
		err = s.Engine.PopulateRemoteShas()
		require.NoError(t, err)

		// Branches should not match (nothing on remote)
		matches, err := s.Engine.BranchMatchesRemote("main")
		require.NoError(t, err)
		require.False(t, matches, "main should not match empty remote")
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("handles branch with no parent gracefully", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("branch1").
			Commit("branch1 change").
			Checkout("main")

		// Branch exists but not tracked
		parent := s.Engine.GetParent("branch1")
		require.Empty(t, parent)

		// GetParentPrecondition should return trunk
		parent = s.Engine.GetParentPrecondition("branch1")
		require.Equal(t, "main", parent)
	})

	t.Run("handles multiple children correctly", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create multiple branches from main
		branchNames := []string{"branch1", "branch2", "branch3", "branch4", "branch5"}
		for _, branchName := range branchNames {
			s.CreateBranch(branchName).
				Commit(branchName + " change").
				Checkout("main")
		}

		// Track all branches
		for _, branchName := range branchNames {
			err := s.Engine.TrackBranch(context.Background(), branchName, "main")
			require.NoError(t, err)
		}

		// Verify all are children of main
		children := s.Engine.GetChildren("main")
		require.Len(t, children, 5)
	})
}

func TestDetachAndResetBranchChanges(t *testing.T) {
	t.Run("detaches and soft resets to parent merge base", func(t *testing.T) {
		s := scenario.NewScenario(t, nil)
		err := s.Scene.Repo.CreateChangeAndCommit("initial content", "shared")
		require.NoError(t, err)

		// Create feature branch that modifies the existing file
		s.CreateBranch("feature")
		err = s.Scene.Repo.CreateChangeAndCommit("feature content", "shared")
		require.NoError(t, err)

		// Get the main branch commit (merge base)
		mainCommit, _ := s.Scene.Repo.GetRevision("main")

		// Track branch
		err = s.Engine.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Call DetachAndResetBranchChanges
		err = s.Engine.DetachAndResetBranchChanges(context.Background(), "feature")
		require.NoError(t, err)

		// Verify HEAD is detached
		currentBranch := s.Engine.CurrentBranch()
		require.Empty(t, currentBranch, "should be in detached HEAD state")

		// Verify we're at the merge base commit
		headCommit, _ := s.Scene.Repo.GetRevision("HEAD")
		require.Equal(t, mainCommit, headCommit, "HEAD should be at parent merge base")

		// Verify the feature changes are now unstaged (modified tracked file)
		hasUnstaged, _ := s.Scene.Repo.HasUnstagedChanges()
		require.True(t, hasUnstaged, "feature changes should appear as unstaged")
	})

	t.Run("works with multi-commit branch modifying same file", func(t *testing.T) {
		s := scenario.NewScenario(t, nil)
		err := s.Scene.Repo.CreateChangeAndCommit("initial", "shared")
		require.NoError(t, err)

		// Create feature branch with multiple commits modifying the same file
		s.CreateBranch("feature").
			CommitChange("shared", "commit 1").
			CommitChange("shared", "commit 2").
			CommitChange("shared", "commit 3")

		// Get main commit
		mainCommit, _ := s.Scene.Repo.GetRevision("main")

		// Track branch
		err = s.Engine.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Call DetachAndResetBranchChanges
		err = s.Engine.DetachAndResetBranchChanges(context.Background(), "feature")
		require.NoError(t, err)

		// Verify we're at main
		headCommit, _ := s.Scene.Repo.GetRevision("HEAD")
		require.Equal(t, mainCommit, headCommit)

		// Verify changes are unstaged
		hasUnstaged, _ := s.Scene.Repo.HasUnstagedChanges()
		require.True(t, hasUnstaged)
	})

	t.Run("works with stacked branches", func(t *testing.T) {
		s := scenario.NewScenario(t, nil)
		err := s.Scene.Repo.CreateChangeAndCommit("initial", "shared")
		require.NoError(t, err)

		// Create branch1 on main
		s.CreateBranch("branch1").
			CommitChange("shared", "branch1 change")

		// Get branch1 commit (this will be the merge base for branch2)
		branch1Commit, _ := s.Scene.Repo.GetRevision("branch1")

		// Create branch2 on branch1
		s.CreateBranch("branch2").
			CommitChange("shared", "branch2 change")

		// Track branches
		err = s.Engine.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = s.Engine.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Call DetachAndResetBranchChanges on branch2
		err = s.Engine.DetachAndResetBranchChanges(context.Background(), "branch2")
		require.NoError(t, err)

		// Verify we're at branch1's commit (the parent)
		headCommit, _ := s.Scene.Repo.GetRevision("HEAD")
		require.Equal(t, branch1Commit, headCommit, "HEAD should be at branch1 (parent of branch2)")

		// Verify branch2 changes are unstaged
		hasUnstaged, _ := s.Scene.Repo.HasUnstagedChanges()
		require.True(t, hasUnstaged)
	})

	t.Run("handles untracked branch using trunk as parent", func(t *testing.T) {
		s := scenario.NewScenario(t, nil)
		err := s.Scene.Repo.CreateChangeAndCommit("initial", "shared")
		require.NoError(t, err)

		// Create feature branch (not tracked)
		s.CreateBranch("feature").
			CommitChange("shared", "feature change")

		mainCommit, _ := s.Scene.Repo.GetRevision("main")

		// Call DetachAndResetBranchChanges without tracking
		err = s.Engine.DetachAndResetBranchChanges(context.Background(), "feature")
		require.NoError(t, err)

		// Should use trunk (main) as the parent
		headCommit, _ := s.Scene.Repo.GetRevision("HEAD")
		require.Equal(t, mainCommit, headCommit, "should use trunk as parent for untracked branch")

		// Verify changes are unstaged
		hasUnstaged, _ := s.Scene.Repo.HasUnstagedChanges()
		require.True(t, hasUnstaged)
	})

	t.Run("handles new files as untracked", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Create feature branch with a NEW file (doesn't exist on main)
		s.CreateBranch("feature")
		err := s.Scene.Repo.CreateChangeAndCommit("new file content", "newfile")
		require.NoError(t, err)

		mainCommit, _ := s.Scene.Repo.GetRevision("main")

		// Track branch
		err = s.Engine.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Call DetachAndResetBranchChanges
		err = s.Engine.DetachAndResetBranchChanges(context.Background(), "feature")
		require.NoError(t, err)

		// Verify we're at main
		headCommit, _ := s.Scene.Repo.GetRevision("HEAD")
		require.Equal(t, mainCommit, headCommit)

		// New files should appear as untracked (not unstaged)
		hasUntracked, _ := s.Scene.Repo.HasUntrackedFiles()
		require.True(t, hasUntracked, "new files should appear as untracked")
	})
}

func TestSetParentScenarios(t *testing.T) {
	t.Run("preserves divergence point when parent is rebased and merged into trunk", func(t *testing.T) {
		// Scenario: main -> branch1 -> branch2
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// 1. Create branch1
		s.CreateBranch("branch1").
			CommitChange("file1.txt", "feat: branch1").
			TrackBranch("branch1", "main")

		branch1OriginalSHA, _ := s.Engine.GetRevision("branch1")

		// 2. Create branch2 on top of branch1
		s.CreateBranch("branch2").
			CommitChange("file2.txt", "feat: branch2").
			TrackBranch("branch2", "branch1")

		// branch2 diverged from branch1 at branch1OriginalSHA

		// 3. Move main forward
		s.Checkout("main").
			CommitChange("main.txt", "feat: main")

		// 4. Rebase branch1 onto main (changing its SHA)
		s.Checkout("branch1")
		s.Engine.RestackBranch(context.Background(), "branch1")
		branch1NewSHA, _ := s.Engine.GetRevision("branch1")
		require.NotEqual(t, branch1OriginalSHA, branch1NewSHA)

		// 5. Merge branch1 into main
		s.Checkout("main")
		s.RunGit("merge", "branch1", "--no-ff", "-m", "Merge branch1")

		// 6. Reparent branch2 to main (what happens during 'stackit merge' or 'stackit sync')
		err := s.Engine.SetParent(context.Background(), "branch2", "main")
		require.NoError(t, err)

		// VERIFY: ParentBranchRevision should still be branch1OriginalSHA
		// because it's a valid ancestor and the old parent (branch1) was merged into main.
		meta, _ := git.ReadMetadataRef("branch2")
		require.Equal(t, branch1OriginalSHA, *meta.ParentBranchRevision, "Divergence point should be preserved to avoid conflicts during restack")
	})

	t.Run("updates divergence point when parent is folded into child (upward merge)", func(t *testing.T) {
		// Scenario: main -> branch1 -> branch2
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// 1. Setup stack
		s.CreateBranch("branch1").
			CommitChange("file1.txt", "feat: branch1").
			TrackBranch("branch1", "main")

		s.CreateBranch("branch2").
			CommitChange("file2.txt", "feat: branch2").
			TrackBranch("branch2", "branch1")

		// 2. Fold branch1 into branch2 (upward merge)
		s.Checkout("branch2")
		s.RunGit("merge", "branch1", "--no-ff", "-m", "Merge branch1 into branch2")

		// 3. Reparent branch2 to main (branch1 will be deleted in a real fold)
		err := s.Engine.SetParent(context.Background(), "branch2", "main")
		require.NoError(t, err)

		// VERIFY: ParentBranchRevision should be updated to main's tip
		// because branch1 was NOT merged into main; it was merged into branch2.
		// If we kept the old divergence point (before branch1), a restack would
		// try to re-apply branch1's changes which are already in branch2.
		mainSHA, _ := s.Engine.GetRevision("main")
		meta, _ := git.ReadMetadataRef("branch2")
		require.Equal(t, mainSHA, *meta.ParentBranchRevision, "Divergence point should be updated to new parent when folding upward")
	})

	t.Run("updates divergence point after manual rebase onto same parent", func(t *testing.T) {
		// Scenario: main -> branch1
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		s.CreateBranch("branch1").
			CommitChange("file1.txt", "feat: branch1").
			TrackBranch("branch1", "main")

		originalMeta, _ := git.ReadMetadataRef("branch1")

		// 1. Move main forward
		s.Checkout("main").
			CommitChange("main.txt", "feat: main")
		mainNewSHA, _ := s.Engine.GetRevision("main")

		// 2. Manually rebase branch1 onto main
		s.Checkout("branch1")
		s.RunGit("rebase", "main")

		// 3. Call SetParent with the same parent (main)
		err := s.Engine.SetParent(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// VERIFY: ParentBranchRevision should be updated to mainNewSHA
		// because the branch has moved forward relative to its parent.
		meta, _ := git.ReadMetadataRef("branch1")
		require.Equal(t, mainNewSHA, *meta.ParentBranchRevision)
		require.NotEqual(t, *originalMeta.ParentBranchRevision, *meta.ParentBranchRevision)
	})
}
