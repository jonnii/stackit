package submit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

const (
	featureBranch = "feature"
)

func TestPreparePRMetadata_DraftStatus(t *testing.T) {
	t.Run("new PR with --draft flag creates as draft", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			Draft: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be created as draft when --draft flag is set")
	})

	t.Run("new PR with --publish flag creates as non-draft", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			Publish: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be created as non-draft when --publish flag is set")
	})

	t.Run("new PR defaults to published (not draft)", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		opts := submit.MetadataOptions{
			// No draft or publish flag - should default to published
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should default to published (not draft) when no flag is specified")
	})

	t.Run("existing PR preserves draft status", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		// Create existing PR info with draft status
		branch := s.Engine.GetBranch(branchName)
		err := s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfoEmpty().
			WithTitle("Existing PR").
			WithBody("PR body").
			WithIsDraft(true))
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			// No draft or publish flag
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should preserve existing draft status")

		// Test with non-draft existing PR
		branch = s.Engine.GetBranch(branchName)
		err = s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfoEmpty().
			WithTitle("Existing PR").
			WithBody("PR body").
			WithIsDraft(false))
		require.NoError(t, err)

		metadata, err = submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should preserve existing non-draft status")
	})

	t.Run("--draft flag overrides existing PR draft status", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		// Create existing PR info with non-draft status
		branch := s.Engine.GetBranch(branchName)
		err := s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfoEmpty().
			WithTitle("Existing PR").
			WithBody("PR body").
			WithIsDraft(false))
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			Draft: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.True(t, metadata.IsDraft, "PR should be marked as draft when --draft flag is set, even if existing PR is not draft")
	})

	t.Run("--publish flag overrides existing PR draft status", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		// Create existing PR info with draft status
		branch := s.Engine.GetBranch(branchName)
		err := s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfoEmpty().
			WithTitle("Existing PR").
			WithBody("PR body").
			WithIsDraft(true))
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			Publish: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.False(t, metadata.IsDraft, "PR should be marked as non-draft when --publish flag is set, even if existing PR is draft")
	})
}

func TestPreparePRMetadata_NoEdit(t *testing.T) {
	t.Run("no-edit skips title and body editing", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := featureBranch

		// Create a commit with a subject
		s.CreateBranch(branchName).
			CommitChange("change", "feat: test feature")

		// Track the branch so GetCommitSubject knows the range
		err := s.Engine.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		opts := submit.MetadataOptions{
			NoEdit: true,
		}

		metadata, err := submit.PreparePRMetadata(branchName, opts, s.Engine, s.Context)
		require.NoError(t, err)
		require.NotEmpty(t, metadata.Title, "Title should be set from commit subject")
		require.Equal(t, "feat: test feature", metadata.Title)
	})
}

func TestGetPRBody_MultipleCommits(t *testing.T) {
	t.Run("returns a bulleted list of subjects for multiple commits", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)
		branchName := "feature"

		// Create multiple commits
		s.CreateBranch(branchName).
			CommitChange("change1", "feat: commit 1").
			CommitChange("change2", "feat: commit 2\n\nwith body")

		// Track the branch
		err := s.Engine.TrackBranch(context.Background(), branchName, "main")
		require.NoError(t, err)

		// Get PR body
		body, err := submit.GetPRBody(branchName, false, "", s.Engine)
		require.NoError(t, err)

		// Note: GetPRBody formats as a list of subjects
		expectedBody := "- feat: commit 1\n- feat: commit 2"
		require.Equal(t, expectedBody, body)
	})
}
