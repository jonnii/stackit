package actions_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestMergeAction(t *testing.T) {
	t.Run("fails when not on a branch", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Detach HEAD
		s.RunGit("checkout", "HEAD~0").Rebuild()

		err := actions.MergeAction(s.Context, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not on a branch")
	})

	t.Run("fails when on trunk", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup)

		// Make sure we're on main
		s.Checkout("main")

		// Verify we're on trunk
		require.True(t, s.Engine.IsTrunk(s.Engine.CurrentBranch()))

		err := actions.MergeAction(s.Context, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})

	t.Run("fails when branch is not tracked", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			CreateBranch("untracked")

		// Verify branch is not tracked
		require.False(t, s.Engine.IsBranchTracked("untracked"))

		err := actions.MergeAction(s.Context, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("returns early when no PRs to merge", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Switch to branch1
		s.Checkout("branch1")

		// Verify branch is tracked
		require.True(t, s.Engine.IsBranchTracked("branch1"))

		err := actions.MergeAction(s.Context, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		// Should fail because no PRs found
		require.Error(t, err)
		require.Contains(t, err.Error(), "no open PRs found")
	})

	t.Run("dry run mode reports PRs without merging", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch1": "main",
			})

		// Add PR info
		prNumber := 123
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/123",
		}
		err := s.Engine.UpsertPrInfo(context.Background(), "branch1", prInfo)
		require.NoError(t, err)

		// Switch to branch1
		s.Checkout("branch1")

		// Verify branch is tracked and has PR info
		require.True(t, s.Engine.IsBranchTracked("branch1"))
		prInfo, err = s.Engine.GetPrInfo(context.Background(), "branch1")
		require.NoError(t, err)
		require.NotNil(t, prInfo)

		err = actions.MergeAction(s.Context, actions.MergeOptions{
			DryRun:   true,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.NoError(t, err)
	})

	t.Run("preserves stack structure when merging bottom PR", func(t *testing.T) {
		s := scenario.NewScenario(t, testhelpers.BasicSceneSetup).
			WithStack(map[string]string{
				"branch-a": "main",
				"branch-b": "branch-a",
				"branch-c": "branch-b",
			})

		// Set up mock GitHub server with PRs
		config := testhelpers.NewMockGitHubServerConfig()
		config.PRs["branch-a"] = testhelpers.NewSamplePullRequest(testhelpers.SamplePRData{
			Number:  101,
			Title:   "Branch A",
			Head:    "branch-a",
			Base:    "main",
			State:   "open",
			HTMLURL: "https://github.com/owner/repo/pull/101",
		})
		config.PRs["branch-b"] = testhelpers.NewSamplePullRequest(testhelpers.SamplePRData{
			Number:  102,
			Title:   "Branch B",
			Head:    "branch-b",
			Base:    "branch-a",
			State:   "open",
			HTMLURL: "https://github.com/owner/repo/pull/102",
		})
		config.PRs["branch-c"] = testhelpers.NewSamplePullRequest(testhelpers.SamplePRData{
			Number:  103,
			Title:   "Branch C",
			Head:    "branch-c",
			Base:    "branch-b",
			State:   "open",
			HTMLURL: "https://github.com/owner/repo/pull/103",
		})

		rawClient, owner, repo := testhelpers.NewMockGitHubClient(t, config)
		githubClient := testhelpers.NewMockGitHubClientInterface(rawClient, owner, repo, config)

		// Add PR info to engine
		prA := 101
		err := s.Engine.UpsertPrInfo(context.Background(), "branch-a", &engine.PrInfo{
			Number: &prA,
			State:  "OPEN",
			Base:   "main",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		require.NoError(t, err)

		prB := 102
		err = s.Engine.UpsertPrInfo(context.Background(), "branch-b", &engine.PrInfo{
			Number: &prB,
			State:  "OPEN",
			Base:   "branch-a",
			URL:    "https://github.com/owner/repo/pull/102",
		})
		require.NoError(t, err)

		prC := 103
		err = s.Engine.UpsertPrInfo(context.Background(), "branch-c", &engine.PrInfo{
			Number: &prC,
			State:  "OPEN",
			Base:   "branch-b",
			URL:    "https://github.com/owner/repo/pull/103",
		})
		require.NoError(t, err)

		// Switch to branch-a (the bottom PR we'll merge)
		s.Checkout("branch-a")

		// Create context with GitHub client
		s.Context.GitHubClient = githubClient

		// Execute merge plan (merge branch-a)
		plan, validation, err := actions.CreateMergePlan(s.Context, actions.CreateMergePlanOptions{
			Strategy: actions.MergeStrategyBottomUp,
			Force:    true,
		})
		require.NoError(t, err)
		require.NotNil(t, plan)
		require.True(t, validation.Valid)

		// Verify plan includes restacking upstack branches
		require.Contains(t, plan.UpstackBranches, "branch-b")
		require.Contains(t, plan.UpstackBranches, "branch-c")

		// Set up a remote so PullTrunk can work
		remoteDir, err := os.MkdirTemp("", "stackit-test-remote-*")
		require.NoError(t, err)
		defer os.RemoveAll(remoteDir)

		s.RunGit("init", "--bare", remoteDir)

		// Add remote and push main
		s.RunGit("remote", "add", "origin", remoteDir).
			RunGit("push", "-u", "origin", "main").
			RunGit("push", "-u", "origin", "branch-a").
			RunGit("push", "-u", "origin", "branch-b").
			RunGit("push", "-u", "origin", "branch-c")

		// Now merge branch-a into main locally and push to simulate the PR merge
		s.Checkout("main").
			RunGit("merge", "branch-a", "--no-ff", "-m", "Merge branch-a").
			RunGit("push", "origin", "main")

		// Switch back to branch-a for the merge execution
		s.Checkout("branch-a")

		// Execute the merge plan
		err = actions.ExecuteMergePlan(s.Context, actions.ExecuteMergePlanOptions{
			Plan:  plan,
			Force: true,
		})
		require.NoError(t, err)

		// Verify PR base branches were updated correctly
		// branch-b should point to main (since branch-a was merged)
		updatedPRB, exists := config.UpdatedPRs[102]
		require.True(t, exists, "branch-b PR should have been updated")
		require.NotNil(t, updatedPRB.Base)
		require.NotNil(t, updatedPRB.Base.Ref)
		require.Equal(t, "main", *updatedPRB.Base.Ref, "branch-b PR base should be main after branch-a is merged")

		// branch-c should point to branch-b (not main - this is the bug fix!)
		updatedPRC, exists := config.UpdatedPRs[103]
		require.True(t, exists, "branch-c PR should have been updated")
		require.NotNil(t, updatedPRC.Base)
		require.NotNil(t, updatedPRC.Base.Ref)
		require.Equal(t, "branch-b", *updatedPRC.Base.Ref, "branch-c PR base should be branch-b (not main) to preserve stack structure")
	})
}
