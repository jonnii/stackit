package actions_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/testhelpers"
)

func TestSyncAction(t *testing.T) {
	t.Run("syncs when trunk is up to date", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		splog := output.NewSplog()
		err = actions.SyncAction(actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
			Engine:  eng,
			Splog:   splog,
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

		splog := output.NewSplog()
		err = actions.SyncAction(actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: false,
			Engine:  eng,
			Splog:   splog,
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

		err = eng.TrackBranch("branch1", "main")
		require.NoError(t, err)

		splog := output.NewSplog()
		err = actions.SyncAction(actions.SyncOptions{
			All:     false,
			Force:   false,
			Restack: true,
			Engine:  eng,
			Splog:   splog,
		})
		// Should succeed (even if no restacking needed)
		require.NoError(t, err)
	})
}
