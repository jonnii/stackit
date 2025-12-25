package engine

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestApplySplitToCommits(t *testing.T) {
	t.Run("creates branches at specified commit points", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		repo := scene.Repo

		// Create a stack: main -> feature
		err := repo.CreateChangeAndCommit("file1 content", "file1")
		require.NoError(t, err)

		eng, err := NewEngine(Options{
			RepoRoot: scene.Dir,
			Trunk:    "main",
		})
		require.NoError(t, err)

		// Create feature branch with 3 commits on top of main
		err = repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = repo.CreateBranch("feature")
		require.NoError(t, err)
		err = repo.CheckoutBranch("feature")
		require.NoError(t, err)

		err = repo.CreateChangeAndCommit("c1", "file1")
		require.NoError(t, err)
		err = repo.CreateChangeAndCommit("c2", "file2")
		require.NoError(t, err)
		err = repo.CreateChangeAndCommit("c3", "file3")
		require.NoError(t, err)

		// Track feature branch
		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Detach before split to avoid "used by worktree" error
		featureSHA, err := repo.GetBranchSHA("feature")
		require.NoError(t, err)
		err = repo.CheckoutDetached(featureSHA)
		require.NoError(t, err)

		// Split feature into branch1 (c1), branch2 (c2), feature (c3)
		opts := ApplySplitOptions{
			BranchToSplit: "feature",
			BranchNames:   []string{"branch1", "branch2", "feature"},
			BranchPoints:  []int{0, 1, 2}, // Will be reversed to [2, 1, 0]
		}

		err = eng.ApplySplitToCommits(context.Background(), opts)
		require.NoError(t, err)

		// Cast to engineImpl to check internal state
		e := eng.(*engineImpl)

		// Verify branches
		testhelpers.ExpectBranches(t, repo, []string{"main", "branch1", "branch2", "feature"})

		// Verify parent relationships
		require.Equal(t, "main", e.parentMap["branch1"])
		require.Equal(t, "branch1", e.parentMap["branch2"])
		require.Equal(t, "branch2", e.parentMap["feature"])

		// Verify commit counts
		// branch1 should have 1 commit from main (c1)
		count, err := repo.GetCommitCount("main", "branch1")
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// branch2 should have 1 commit from branch1 (c2)
		count, err = repo.GetCommitCount("branch1", "branch2")
		require.NoError(t, err)
		require.Equal(t, 1, count)

		// feature should have 1 commit from branch2 (c3)
		count, err = repo.GetCommitCount("branch2", "feature")
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("successfully applies split when branch ref is updated after new commits", func(t *testing.T) {
		// This test simulates what split --by-hunk does:
		// 1. Start with feature branch
		// 2. Create new commits in detached HEAD (on top of main)
		// 3. Update feature branch ref to point to the new tip
		// 4. Apply split

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		repo := scene.Repo

		// Trunk commit
		err := repo.CreateChangeAndCommit("initial", "file")
		require.NoError(t, err)

		eng, err := NewEngine(Options{
			RepoRoot: scene.Dir,
			Trunk:    "main",
		})
		require.NoError(t, err)

		// Original feature branch
		err = repo.CreateBranch("feature")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Detach and create new commits (simulating splitByHunk)
		mainSHA, err := repo.GetBranchSHA("main")
		require.NoError(t, err)
		err = repo.CheckoutDetached(mainSHA)
		require.NoError(t, err)

		err = repo.CreateChangeAndCommit("new c1", "file1")
		require.NoError(t, err)
		c1SHA, _ := repo.GetCurrentSHA()

		err = repo.CreateChangeAndCommit("new c2", "file2")
		require.NoError(t, err)
		c2SHA, _ := repo.GetCurrentSHA()

		// Update feature ref to point to the new tip (the fix!)
		err = git.UpdateBranchRef("feature", c2SHA)
		require.NoError(t, err)

		// Apply split
		opts := ApplySplitOptions{
			BranchToSplit: "feature",
			BranchNames:   []string{"branch1", "feature"},
			BranchPoints:  []int{0, 1}, // Will be reversed to [1, 0]
		}

		err = eng.ApplySplitToCommits(context.Background(), opts)
		require.NoError(t, err)

		// Verify branch1 points to c1 and feature points to c2
		b1SHA, err := repo.GetBranchSHA("branch1")
		require.NoError(t, err)
		require.Equal(t, c1SHA, b1SHA)

		fSHA, err := repo.GetBranchSHA("feature")
		require.NoError(t, err)
		require.Equal(t, c2SHA, fSHA)
	})

	t.Run("validates branch names and points match", func(t *testing.T) {
		// Create a mock engine for this validation test
		eng := &engineImpl{}

		opts := ApplySplitOptions{
			BranchToSplit: "feature",
			BranchNames:   []string{"branch1"}, // 1 name
			BranchPoints:  []int{0, 1},         // 2 points
		}

		err := eng.ApplySplitToCommits(context.Background(), opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid number of branch names")
	})

	t.Run("fails when branch has no parent", func(t *testing.T) {
		t.Skip("Requires engine with metadata setup")
	})

	t.Run("successfully applies split with valid inputs", func(t *testing.T) {
		t.Skip("Requires full git repository and engine setup")
	})
}

// Test helper functions

func TestContains(t *testing.T) {
	// Test slices.Contains used in ApplySplitToCommits
	require.True(t, slices.Contains([]string{"a", "b", "c"}, "b"))
	require.False(t, slices.Contains([]string{"a", "b", "c"}, "d"))
	require.False(t, slices.Contains([]string{}, "a"))
}
