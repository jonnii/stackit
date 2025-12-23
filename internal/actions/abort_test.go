package actions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func init() {
	// Disable interactive prompts in tests
	os.Setenv("STACKIT_TEST_NO_INTERACTIVE", "1")
}

func TestAbortAction(t *testing.T) {
	t.Run("reports when no operation is in progress", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit()

		err := actions.AbortAction(s.Context, actions.AbortOptions{})
		require.NoError(t, err)
	})

	t.Run("aborts when continuation state exists", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		s.WithInitialCommit().
			CreateBranch("feature").
			Commit("feature change").
			Checkout("main").
			TrackBranch("feature", "main")

		// Get initial state
		initialSHA, err := s.Engine.GetRevision("feature")
		require.NoError(t, err)

		// Take snapshot
		err = s.Engine.TakeSnapshot("restack", []string{"feature"})
		require.NoError(t, err)

		// Manually create continuation state
		continuationPath := filepath.Join(s.Scene.Dir, ".git", ".stackit_continue")
		err = os.WriteFile(continuationPath, []byte("{}"), 0644)
		require.NoError(t, err)

		// Run abort
		err = actions.AbortAction(s.Context, actions.AbortOptions{Force: true})
		require.NoError(t, err)

		// Verify continuation state is gone
		_, err = os.Stat(continuationPath)
		require.True(t, os.IsNotExist(err), "continuation state should be gone")

		// Verify state restored
		s.Engine.Rebuild(s.Engine.Trunk())
		restoredSHA, err := s.Engine.GetRevision("feature")
		require.NoError(t, err)
		require.Equal(t, initialSHA, restoredSHA)
	})
}
