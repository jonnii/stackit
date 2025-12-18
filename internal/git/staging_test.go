package git_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/testhelpers"
)

func TestStageAll(t *testing.T) {
	t.Run("stages all changes including untracked", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create unstaged change
		err := scene.Repo.CreateChange("new content", "test", true)
		require.NoError(t, err)

		// Create untracked file
		err = scene.Repo.CreateChange("untracked", "newfile", true)
		require.NoError(t, err)

		// Verify no staged changes initially
		hasStaged, err := git.HasStagedChanges()
		require.NoError(t, err)
		require.False(t, hasStaged)

		// Stage all
		err = git.StageAll()
		require.NoError(t, err)

		// Verify changes are staged
		hasStaged, err = git.HasStagedChanges()
		require.NoError(t, err)
		require.True(t, hasStaged)
	})
}

func TestStageTracked(t *testing.T) {
	t.Run("stages only tracked file changes", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			// Create initial tracked file
			return s.Repo.CreateChangeAndCommit("initial", "test")
		})

		// Modify tracked file (unstaged)
		err := scene.Repo.CreateChange("modified", "test", true)
		require.NoError(t, err)

		// Create untracked file
		err = scene.Repo.CreateChange("untracked", "newfile", true)
		require.NoError(t, err)

		// Stage tracked only
		err = git.StageTracked()
		require.NoError(t, err)

		// Verify tracked file is staged
		hasStaged, err := git.HasStagedChanges()
		require.NoError(t, err)
		require.True(t, hasStaged)
	})
}

func TestHasStagedChanges(t *testing.T) {
	t.Run("returns false when no staged changes", func(t *testing.T) {
		_ = testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		hasStaged, err := git.HasStagedChanges()
		require.NoError(t, err)
		require.False(t, hasStaged)
	})

	t.Run("returns true when changes are staged", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create and stage change
		err := scene.Repo.CreateChange("new content", "test", false)
		require.NoError(t, err)

		hasStaged, err := git.HasStagedChanges()
		require.NoError(t, err)
		require.True(t, hasStaged)
	})
}

func TestHasUnstagedChanges(t *testing.T) {
	t.Run("returns false when no unstaged changes", func(t *testing.T) {
		_ = testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		hasUnstaged, err := git.HasUnstagedChanges()
		require.NoError(t, err)
		require.False(t, hasUnstaged)
	})

	t.Run("returns true when unstaged changes exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "test")
		})

		// Create unstaged change
		err := scene.Repo.CreateChange("modified", "test", true)
		require.NoError(t, err)

		hasUnstaged, err := git.HasUnstagedChanges()
		require.NoError(t, err)
		require.True(t, hasUnstaged)
	})
}

func TestHasUntrackedFiles(t *testing.T) {
	t.Run("returns false when no untracked files", func(t *testing.T) {
		_ = testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		hasUntracked, err := git.HasUntrackedFiles()
		require.NoError(t, err)
		require.False(t, hasUntracked)
	})

	t.Run("returns true when untracked files exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create untracked file
		err := scene.Repo.CreateChange("content", "newfile", true)
		require.NoError(t, err)

		hasUntracked, err := git.HasUntrackedFiles()
		require.NoError(t, err)
		require.True(t, hasUntracked)
	})
}

func TestAddAll(t *testing.T) {
	t.Run("is alias for StageAll", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create unstaged change
		err := scene.Repo.CreateChange("new content", "test", true)
		require.NoError(t, err)

		// Use AddAll
		err = git.AddAll()
		require.NoError(t, err)

		// Verify changes are staged
		hasStaged, err := git.HasStagedChanges()
		require.NoError(t, err)
		require.True(t, hasStaged)
	})
}
