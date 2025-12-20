package actions_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestMergeAction(t *testing.T) {
	t.Run("fails when not on a branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Detach HEAD by checking out a commit directly
		// Get the current commit SHA first
		commitSHA, err := scene.Repo.RunGitCommandAndGetOutput("rev-parse", "HEAD")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("checkout", "--detach", commitSHA)
		require.NoError(t, err)

		// Reset default repo to ensure it sees the detached HEAD state
		git.ResetDefaultRepo()

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not on a branch")
	})

	t.Run("fails when on trunk", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Make sure we're on main
		err := scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see we're on main)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify we're on trunk
		require.True(t, eng.IsTrunk(eng.CurrentBranch()))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot merge from trunk")
	})

	t.Run("fails when branch is not tracked", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		// Create engine (will see branch1 but not track it)
		_, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// Note: The branch won't be tracked in the new engine since we didn't track it
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is not tracked
		require.False(t, eng.IsBranchTracked("branch1"))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("returns early when no PRs to merge", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see branch1)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// The engine will rebuild and should see the tracked branch from metadata
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is tracked (metadata should persist)
		require.True(t, eng.IsBranchTracked("branch1"))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   false,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		// Should fail because no PRs found
		require.Error(t, err)
		require.Contains(t, err.Error(), "no open PRs found")
	})

	t.Run("dry run mode reports PRs without merging", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch before engine
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		// Create engine (will see branch1)
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		// Add PR info
		prNumber := 123
		prInfo := &engine.PrInfo{
			Number: &prNumber,
			State:  "OPEN",
			URL:    "https://github.com/owner/repo/pull/123",
		}
		err = eng.UpsertPrInfo(context.Background(), "branch1", prInfo)
		require.NoError(t, err)

		// Switch to branch1
		err = scene.Repo.CheckoutBranch("branch1")
		require.NoError(t, err)

		// Create new engine to get updated current branch
		// The engine will rebuild and should see the tracked branch and PR info from metadata
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Verify branch is tracked and has PR info
		require.True(t, eng.IsBranchTracked("branch1"))
		prInfo, err = eng.GetPrInfo(context.Background(), "branch1")
		require.NoError(t, err)
		require.NotNil(t, prInfo)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		err = actions.MergeAction(ctx, actions.MergeOptions{
			DryRun:   true,
			Confirm:  false,
			Strategy: actions.MergeStrategyBottomUp,
		})
		require.NoError(t, err)
	})

	t.Run("preserves stack structure when merging bottom PR", func(t *testing.T) {
		// This test verifies the fix for the bug where merging the bottom PR
		// would unstack all remaining PRs (making them all point to main).
		// After merging branch-a, branch-b should point to main, and branch-c
		// should point to branch-b (not main).

		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create stack: main → branch-a → branch-b → branch-c
		err := scene.Repo.CreateAndCheckoutBranch("branch-a")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch-a change", "a")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch-b")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch-b change", "b")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch-c")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch-c change", "c")
		require.NoError(t, err)

		// Create engine and track branches
		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch-a", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch-b", "branch-a")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch-c", "branch-b")
		require.NoError(t, err)

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
		err = eng.UpsertPrInfo(context.Background(), "branch-a", &engine.PrInfo{
			Number: &prA,
			State:  "OPEN",
			Base:   "main",
			URL:    "https://github.com/owner/repo/pull/101",
		})
		require.NoError(t, err)

		prB := 102
		err = eng.UpsertPrInfo(context.Background(), "branch-b", &engine.PrInfo{
			Number: &prB,
			State:  "OPEN",
			Base:   "branch-a",
			URL:    "https://github.com/owner/repo/pull/102",
		})
		require.NoError(t, err)

		prC := 103
		err = eng.UpsertPrInfo(context.Background(), "branch-c", &engine.PrInfo{
			Number: &prC,
			State:  "OPEN",
			Base:   "branch-b",
			URL:    "https://github.com/owner/repo/pull/103",
		})
		require.NoError(t, err)

		// Switch to branch-a (the bottom PR we'll merge)
		err = scene.Repo.CheckoutBranch("branch-a")
		require.NoError(t, err)

		// Rebuild engine to get updated current branch
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Create context with GitHub client
		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.GitHubClient = githubClient

		// Execute merge plan (merge branch-a)
		plan, validation, err := actions.CreateMergePlan(ctx, actions.CreateMergePlanOptions{
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
		// Create a bare repo to use as remote
		remoteDir, err := os.MkdirTemp("", "stackit-test-remote-*")
		require.NoError(t, err)
		defer os.RemoveAll(remoteDir)

		err = scene.Repo.RunGitCommand("init", "--bare", remoteDir)
		require.NoError(t, err)

		// Add remote and push main
		err = scene.Repo.RunGitCommand("remote", "add", "origin", remoteDir)
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("push", "-u", "origin", "main")
		require.NoError(t, err)

		// Push all branches to remote
		err = scene.Repo.RunGitCommand("push", "-u", "origin", "branch-a")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("push", "-u", "origin", "branch-b")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("push", "-u", "origin", "branch-c")
		require.NoError(t, err)

		// Now merge branch-a into main locally and push to simulate the PR merge
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("merge", "branch-a", "--no-ff", "-m", "Merge branch-a")
		require.NoError(t, err)
		err = scene.Repo.RunGitCommand("push", "origin", "main")
		require.NoError(t, err)

		// Switch back to branch-a for the merge execution
		err = scene.Repo.CheckoutBranch("branch-a")
		require.NoError(t, err)

		// Rebuild engine after the merge
		eng, err = engine.NewEngine(scene.Dir)
		require.NoError(t, err)
		ctx.Engine = eng

		// Execute the merge plan
		err = actions.ExecuteMergePlan(ctx, actions.ExecuteMergePlanOptions{
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
