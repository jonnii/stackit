package actions_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
)

func TestPreparePRMetadata_DraftStatus(t *testing.T) {
	t.Run("new PR with --draft flag creates as draft", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		opts := actions.SubmitMetadataOptions{
			Draft: true,
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be created as draft when --draft flag is set")
	})

	t.Run("new PR with --publish flag creates as non-draft", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		opts := actions.SubmitMetadataOptions{
			Publish: true,
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be created as non-draft when --publish flag is set")
	})

	t.Run("new PR in non-interactive mode defaults to draft", func(t *testing.T) {
		// Redirect stdin to simulate non-interactive mode
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		// Open /dev/null to simulate non-interactive terminal
		nullFile, err := os.Open(os.DevNull)
		require.NoError(t, err)
		defer nullFile.Close()
		os.Stdin = nullFile

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		opts := actions.SubmitMetadataOptions{
			// No draft or publish flag
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should default to draft in non-interactive mode")
	})

	t.Run("existing PR preserves draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		// Create existing PR info with draft status
		err = eng.UpsertPrInfo(branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: true,
		})
		require.NoError(t, err)

		opts := actions.SubmitMetadataOptions{
			// No draft or publish flag
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should preserve existing draft status")

		// Test with non-draft existing PR
		err = eng.UpsertPrInfo(branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: false,
		})
		require.NoError(t, err)

		metadata, err = actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should preserve existing non-draft status")
	})

	t.Run("--draft flag overrides existing PR draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		// Create existing PR info with non-draft status
		err = eng.UpsertPrInfo(branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: false,
		})
		require.NoError(t, err)

		opts := actions.SubmitMetadataOptions{
			Draft: true,
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be marked as draft when --draft flag is set, even if existing PR is not draft")
	})

	t.Run("--publish flag overrides existing PR draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		// Create existing PR info with draft status
		err = eng.UpsertPrInfo(branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: true,
		})
		require.NoError(t, err)

		opts := actions.SubmitMetadataOptions{
			Publish: true,
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be marked as non-draft when --publish flag is set, even if existing PR is draft")
	})
}

func TestPreparePRMetadata_NoEdit(t *testing.T) {
	t.Run("no-edit skips title and body editing", func(t *testing.T) {
		// Redirect stdin to simulate non-interactive mode
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		nullFile, err := os.Open(os.DevNull)
		require.NoError(t, err)
		defer nullFile.Close()
		os.Stdin = nullFile

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := context.NewContext(eng)
		branchName := "feature"

		// Create a commit with a subject
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("change", "feat: test feature")
		require.NoError(t, err)

		opts := actions.SubmitMetadataOptions{
			NoEdit: true,
		}

		metadata, err := actions.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.NotEmpty(t, metadata.Title, "Title should be set from commit subject")
		require.Equal(t, "feat: test feature", metadata.Title)
	})
}
