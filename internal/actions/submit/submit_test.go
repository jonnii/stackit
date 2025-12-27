package submit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestActionWithMockedGitHub(t *testing.T) {
	t.Run("creates PR for branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"feature": "main",
			})

		// Create a local remote to push to
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Create mocked GitHub client
		config := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)

		// Create context with mocked client
		s.Context.GitHubClient = githubClient
		opts := submit.Options{
			DryRun: false, // We want to test actual PR creation
			NoEdit: true,  // Skip interactive prompts
			Draft:  true,  // Set draft status explicitly to skip prompt
		}

		// With mocked client, push is skipped, so this should succeed
		err = submit.Action(s.Context, opts)
		require.NoError(t, err, "Submit should succeed with mocked GitHub client")

		// Verify that PR was created in the mock
		require.Greater(t, len(config.CreatedPRs), 0, "Should have created at least one PR")
		require.Equal(t, "feature", *config.CreatedPRs[0].Head.Ref, "PR should be for feature branch")
	})

	t.Run("updates existing PR", func(t *testing.T) {
		// Skip this test for now - branch tracking issue needs to be resolved separately
		t.Skip("Skipping due to branch tracking issue in test setup")

		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"feature": "main",
			})

		// Create a local remote to push to
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Create mocked GitHub client with existing PR
		config := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)

		// Pre-create a PR in the mock
		branchName := "feature"
		prNumber := 123
		prData := testhelpers.DefaultPRData()
		prData.Head = branchName
		prData.Number = prNumber
		pr := testhelpers.NewSamplePullRequest(prData)
		config.PRs[branchName] = pr
		config.CreatedPRs = append(config.CreatedPRs, pr)
		// Also add to UpdatedPRs so Get works
		config.UpdatedPRs[prNumber] = pr

		// Store PR info in engine
		branch := s.Engine.GetBranch(branchName)
		err = s.Engine.UpsertPrInfo(branch, testhelpers.NewTestPrInfoWithTitle(prNumber, prData.Title).
			WithBody(prData.Body).
			WithIsDraft(prData.Draft))
		require.NoError(t, err)

		// Create context with mocked client
		s.Context.GitHubClient = githubClient
		opts := submit.Options{
			DryRun: false,
			NoEdit: true,
		}

		// With mocked client, push is skipped, so this should succeed
		err = submit.Action(s.Context, opts)
		require.NoError(t, err, "Submit should succeed with mocked GitHub client")

		// Verify that PR was updated in the mock
		require.Greater(t, len(config.UpdatedPRs), 0, "Should have updated at least one PR")
		updatedPR, exists := config.UpdatedPRs[prNumber]
		require.True(t, exists, "PR %d should be in UpdatedPRs", prNumber)
		require.NotNil(t, updatedPR, "Updated PR should not be nil")
	})

	t.Run("submits entire branching stack with --stack flag", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"P":  "main",
				"C1": "P",
				"C2": "P",
			})

		// Move back to P
		s.Checkout("P")

		// Create a local remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Create mocked GitHub client
		mockConfig := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, mockConfig)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, mockConfig)

		s.Context.GitHubClient = githubClient

		// Submit with --stack flag from branch P
		opts := submit.Options{
			Stack:  true,
			NoEdit: true,
			Draft:  true,
		}

		err = submit.Action(s.Context, opts)
		require.NoError(t, err)

		// Should have created 3 PRs: P, C1, and C2
		require.Equal(t, 3, len(mockConfig.CreatedPRs), "Should have created PRs for P and its children C1, C2")

		// Verify branches are correct
		createdBranches := make(map[string]bool)
		for _, pr := range mockConfig.CreatedPRs {
			createdBranches[*pr.Head.Ref] = true
		}
		require.True(t, createdBranches["P"])
		require.True(t, createdBranches["C1"])
		require.True(t, createdBranches["C2"])
	})

	t.Run("skips base update when no commits between base and head", func(t *testing.T) {
		// This test covers the scenario where after reordering, a branch has no commits
		// between it and its new base, which would cause GitHub to reject the PR update.
		// Our fix should detect this and skip the base update to avoid the error.
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"A": "main",
				"B": "A",
			})

		// Create a local remote
		_, err := s.Scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Get the SHA of branch A
		aSHA, err := s.Engine.GetBranch("A").GetRevision()
		require.NoError(t, err)

		// Make branch B point to the same commit as A
		// This simulates the scenario where there are no commits between B and A
		s.Checkout("A")
		s.RunGit("branch", "-f", "B", aSHA)
		s.Rebuild()

		// Create mocked GitHub client
		config := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)

		// Create a PR for B with A as the base (but B and A point to same commit)
		prNumberB := 101
		prDataB := testhelpers.DefaultPRData()
		prDataB.Head = "B"
		prDataB.Number = prNumberB
		prDataB.Base = "main" // Original base
		prB := testhelpers.NewSamplePullRequest(prDataB)
		config.PRs["B"] = prB
		config.CreatedPRs = append(config.CreatedPRs, prB)
		config.UpdatedPRs[prNumberB] = prB

		// Store PR info in engine with A as the base (simulating after reorder)
		branchB := s.Engine.GetBranch("B")
		err = s.Engine.UpsertPrInfo(branchB, testhelpers.NewTestPrInfoWithTitle(prNumberB, prDataB.Title).
			WithBody(prDataB.Body).
			WithBase("main")) // Will be changed to "A" in prepareBranchesForSubmit
		require.NoError(t, err)

		// Update parent relationship: B's parent is now A
		err = s.Engine.SetParent(context.Background(), s.Engine.GetBranch("B"), s.Engine.GetBranch("A"))
		require.NoError(t, err)

		// Verify that B and A have the same SHA (no commits between them)
		bSHA, err := s.Engine.GetBranch("B").GetRevision()
		require.NoError(t, err)
		require.Equal(t, aSHA, bSHA, "B and A should point to the same commit")

		// Now try to submit B with A as the new base
		// Since B's SHA equals A's SHA, the base update should be skipped
		s.Context.GitHubClient = githubClient
		opts := submit.Options{
			DryRun: false,
			NoEdit: true,
			Draft:  true,
		}

		s.Checkout("B")
		err = submit.Action(s.Context, opts)
		require.NoError(t, err, "Submit should succeed even when base update is skipped due to no commits")

		// Verify that the PR was updated (other fields should be updated)
		updatedPR, exists := config.UpdatedPRs[prNumberB]
		require.True(t, exists, "PR %d should be in UpdatedPRs", prNumberB)
		require.NotNil(t, updatedPR, "Updated PR should not be nil")
	})
}
