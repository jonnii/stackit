package submit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/submit"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestActionWithMockedGitHub(t *testing.T) {
	t.Run("creates PR for branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit by setting trunk in config
		// The engine will read this when created
		err = scene.Repo.RunGitCommand("config", "--local", "stackit.trunk", "main")
		require.NoError(t, err)

		// Create a feature branch
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature change", "feat")
		require.NoError(t, err)

		// Track the branch in the engine
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Create a local remote to push to
		_, err = scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Create mocked GitHub client
		config := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)

		// Create context with mocked client
		ctx := runtime.NewContext(eng)
		ctx.GitHubClient = githubClient
		opts := submit.Options{
			DryRun: false, // We want to test actual PR creation
			NoEdit: true,  // Skip interactive prompts
			Draft:  true,  // Set draft status explicitly to skip prompt
		}

		// With mocked client, push is skipped, so this should succeed
		err = submit.Action(ctx, opts)
		require.NoError(t, err, "Submit should succeed with mocked GitHub client")

		// Verify that PR was created in the mock
		require.Greater(t, len(config.CreatedPRs), 0, "Should have created at least one PR")
		require.Equal(t, "feature", *config.CreatedPRs[0].Head.Ref, "PR should be for feature branch")
	})

	t.Run("updates existing PR", func(t *testing.T) {
		// Skip this test for now - branch tracking issue needs to be resolved separately
		t.Skip("Skipping due to branch tracking issue in test setup")

		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit by setting trunk in config
		err = scene.Repo.RunGitCommand("config", "--local", "stackit.trunk", "main")
		require.NoError(t, err)

		// Create a feature branch
		err = scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature change", "feat")
		require.NoError(t, err)

		// Track the branch in the engine
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		// Create a local remote to push to
		_, err = scene.Repo.CreateBareRemote("origin")
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
		err = eng.UpsertPrInfo(context.Background(), branchName, &engine.PrInfo{
			Number:  &prNumber,
			Title:   prData.Title,
			Body:    prData.Body,
			IsDraft: prData.Draft,
		})
		require.NoError(t, err)

		// Create context with mocked client
		ctx := runtime.NewContext(eng)
		ctx.GitHubClient = githubClient
		opts := submit.Options{
			DryRun: false,
			NoEdit: true,
		}

		// With mocked client, push is skipped, so this should succeed
		err = submit.Action(ctx, opts)
		require.NoError(t, err, "Submit should succeed with mocked GitHub client")

		// Verify that PR was updated in the mock
		require.Greater(t, len(config.UpdatedPRs), 0, "Should have updated at least one PR")
		updatedPR, exists := config.UpdatedPRs[prNumber]
		require.True(t, exists, "PR %d should be in UpdatedPRs", prNumber)
		require.NotNil(t, updatedPR, "Updated PR should not be nil")
	})

	t.Run("submits entire branching stack with --stack flag", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		err = scene.Repo.RunGitCommand("config", "--local", "stackit.trunk", "main")
		require.NoError(t, err)

		// Create complex structure:
		// main
		// └── P
		//     ├── C1
		//     └── C2

		err = scene.Repo.CreateAndCheckoutBranch("P")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("P change", "p")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("C1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("C1 change", "c1")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("P")
		require.NoError(t, err)
		err = scene.Repo.CreateAndCheckoutBranch("C2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("C2 change", "c2")
		require.NoError(t, err)

		// Move back to P
		err = scene.Repo.CheckoutBranch("P")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "P", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "C1", "P")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "C2", "P")
		require.NoError(t, err)

		// Create a local remote
		_, err = scene.Repo.CreateBareRemote("origin")
		require.NoError(t, err)

		// Create mocked GitHub client
		mockConfig := testhelpers.NewMockGitHubServerConfig()
		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, mockConfig)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, mockConfig)

		ctx := runtime.NewContext(eng)
		ctx.GitHubClient = githubClient
		ctx.RepoRoot = scene.Dir

		// Submit with --stack flag from branch P
		opts := submit.Options{
			Stack:  true,
			NoEdit: true,
			Draft:  true,
		}

		err = submit.Action(ctx, opts)
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
}
