package merge_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions/merge"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestExecuteInWorktree(t *testing.T) {
	t.Run("successfully merges in worktree", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
			})

		// Set up mock GitHub server with PR
		mockConfig := testhelpers.NewMockGitHubServerConfig()
		mockConfig.PRs["branch-a"] = testhelpers.NewSamplePullRequest(testhelpers.SamplePRData{
			Number:  101,
			Title:   "Branch A",
			Head:    "branch-a",
			Base:    "main",
			State:   "open",
			HTMLURL: "https://github.com/owner/repo/pull/101",
		})

		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, mockConfig)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, mockConfig)

		// Add PR info to engine
		prA := 101
		err := s.Engine.UpsertPrInfo("branch-a", &engine.PrInfo{
			Number: &prA,
			State:  "OPEN",
			Base:   "main",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		require.NoError(t, err)

		// Create a remote
		remoteDir, err := os.MkdirTemp("", "stackit-test-remote-*")
		require.NoError(t, err)
		defer os.RemoveAll(remoteDir)
		s.RunGit("init", "--bare", remoteDir)
		s.RunGit("remote", "add", "origin", remoteDir).
			RunGit("push", "-u", "origin", "main").
			RunGit("push", "-u", "origin", "branch-a")

		// Create merge plan
		s.Checkout("branch-a")
		s.Context.GitHubClient = githubClient

		plan, validation, err := merge.CreateMergePlan(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, merge.CreatePlanOptions{
			Strategy: merge.StrategyBottomUp,
			Force:    true,
		})
		require.NoError(t, err)
		require.True(t, validation.Valid)

		// Now merge branch-a into main locally and push to simulate the PR merge
		s.Checkout("main").
			RunGit("merge", "branch-a", "--no-ff", "-m", "Merge branch-a").
			RunGit("push", "origin", "main")

		// Switch back to branch-a
		s.Checkout("branch-a")

		// Execute in worktree
		err = merge.ExecuteInWorktree(s.Context.Context, s.Engine, s.Context.Splog, s.Context.GitHubClient, s.Context.RepoRoot, merge.ExecuteOptions{
			Plan:  plan,
			Force: true,
		})
		require.NoError(t, err)

		// Verify we are still on branch-a in the main workspace
		require.Equal(t, "branch-a", s.Engine.CurrentBranch().Name)
	})
}
