package submit_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

const (
	featureBranch = "feature"
)

func TestPreparePRMetadata_DraftStatus(t *testing.T) {
	t.Run("new PR with --draft flag creates as draft", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			Draft: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be created as draft when --draft flag is set")
	})

	t.Run("new PR with --publish flag creates as non-draft", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			Publish: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be created as non-draft when --publish flag is set")
	})

	t.Run("new PR defaults to published (not draft)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			// No draft or publish flag - should default to published
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should default to published (not draft) when no flag is specified")
	})

	t.Run("existing PR preserves draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create existing PR info with draft status
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: true,
		})
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			// No draft or publish flag
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should preserve existing draft status")

		// Test with non-draft existing PR
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: false,
		})
		require.NoError(t, err)

		metadata, err = submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should preserve existing non-draft status")
	})

	t.Run("--draft flag overrides existing PR draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create existing PR info with non-draft status
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: false,
		})
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			Draft: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be marked as draft when --draft flag is set, even if existing PR is not draft")
	})

	t.Run("--publish flag overrides existing PR draft status", func(t *testing.T) {
		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create existing PR info with draft status
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Title:   "Existing PR",
			Body:    "PR body",
			IsDraft: true,
		})
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			Publish: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be marked as non-draft when --publish flag is set, even if existing PR is draft")
	})
}

func TestPreparePRMetadata_NoEdit(t *testing.T) {
	t.Run("no-edit skips title and body editing", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create a commit with a subject
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		// CreateChangeAndCommit(textValue, prefix) - textValue is used as commit message
		err = scene.Repo.CreateChangeAndCommit("feat: test feature", "change")
		require.NoError(t, err)

		// Track the branch so GetCommitSubject knows the range
		err = eng.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			NoEdit: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.NotEmpty(t, metadata.Title, "Title should be set from commit subject")
		require.Equal(t, "feat: test feature", metadata.Title)
	})
}

func TestPreparePRMetadata_AI(t *testing.T) {
	t.Run("AI generation when enabled and no existing body", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		branchName := featureBranch

		// Create a commit
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feat: add feature", "change")
		require.NoError(t, err)

		// Track the branch
		err = eng.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		// Create mock AI client with response
		mockClient := ai.NewMockClient()
		mockClient.SetMockResponse("AI Generated Title", "AI Generated Body\n\nThis is a test PR description.")

		opts := submit.MetadataOptions{
			AI:       true,
			AIClient: mockClient,
			NoEdit:   true, // Skip editor to test AI generation
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.Equal(t, "AI Generated Title", metadata.Title)
		require.Contains(t, metadata.Body, "AI Generated Body")
	})

	t.Run("AI fallback when generation fails", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		branchName := featureBranch

		// Create a commit
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feat: add feature", "change")
		require.NoError(t, err)

		// Track the branch
		err = eng.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		// Create mock AI client with error
		mockClient := ai.NewMockClient()
		mockClient.SetMockError(fmt.Errorf("AI service unavailable"))

		opts := submit.MetadataOptions{
			AI:       true,
			AIClient: mockClient,
			NoEdit:   true, // Skip editor
		}

		// Should fall back to default behavior (commit message)
		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.Equal(t, "feat: add feature", metadata.Title)
		// Body should be empty or from commit messages, not AI-generated
	})

	t.Run("AI disabled uses default behavior", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create a commit
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feat: add feature", "change")
		require.NoError(t, err)

		// Track the branch
		err = eng.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			AI:       false, // AI disabled
			AIClient: nil,
			NoEdit:   true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.Equal(t, "feat: add feature", metadata.Title)
		// Should not call AI client
	})

	t.Run("AI skipped when existing body present", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := featureBranch

		// Create existing PR info with body
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Title: "Existing Title",
			Body:  "Existing Body",
		})
		require.NoError(t, err)

		// Create mock AI client
		mockClient := ai.NewMockClient()
		mockClient.SetMockResponse("AI Title", "AI Body")

		opts := submit.MetadataOptions{
			AI:       true,
			AIClient: mockClient,
			NoEdit:   true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, eng, ctx)
		require.NoError(t, err)
		require.Equal(t, "Existing Title", metadata.Title)
		require.Equal(t, "Existing Body", metadata.Body)
		// AI should not be called since body already exists
		require.Equal(t, 0, mockClient.CallCount(), "AI client should not be called when body exists")
	})
}

func TestGetPRBody_MultipleCommits(t *testing.T) {
	t.Run("returns a bulleted list of subjects for multiple commits", func(t *testing.T) {
		// Set environment variable to force non-interactive mode
		os.Setenv("STACKIT_NON_INTERACTIVE", "1")
		defer os.Unsetenv("STACKIT_NON_INTERACTIVE")

		scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		branchName := "feature"

		// Create multiple commits
		err = scene.Repo.CreateAndCheckoutBranch(branchName)
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feat: commit 1", "change1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feat: commit 2\n\nwith body", "change2")
		require.NoError(t, err)

		// Track the branch
		err = eng.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		// Get PR body (should not prompt due to STACKIT_NON_INTERACTIVE)
		body, err := submit.GetPRBody(branchName, false, "", ctx)
		require.NoError(t, err)

		// Note: GetPRBody formats as a list of subjects
		expectedBody := "feat: commit 1\nfeat: commit 2"
		require.Equal(t, expectedBody, body)
	})
}
