package ai_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestCollectPRContext(t *testing.T) {
	t.Run("collects context for a branch with parent", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> feature
		err := scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("feature change", "feature.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "feature")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.Equal(t, "feature", prCtx.BranchName)
		require.Equal(t, "main", prCtx.ParentBranchName)
		require.Equal(t, "main", prCtx.TrunkBranchName)
		require.Len(t, prCtx.CommitMessages, 1)
		require.Contains(t, prCtx.CommitMessages[0], "feature change")
		require.NotEmpty(t, prCtx.CodeDiff)
		require.Contains(t, prCtx.ChangedFiles, "feature.txt_test.txt")
	})

	t.Run("collects context for trunk branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "main")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.Equal(t, "main", prCtx.BranchName)
		require.Equal(t, "main", prCtx.TrunkBranchName)
		require.Empty(t, prCtx.ParentBranchName) // Trunk has no parent
	})

	t.Run("collects parent PR info when available", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> parent -> child
		err := scene.Repo.CreateAndCheckoutBranch("parent")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("parent change", "parent.txt")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("child")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("child change", "child.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "parent", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "child", "parent")
		require.NoError(t, err)

		// Add PR info for parent
		prNum := 101
		err = eng.UpsertPrInfo(context.Background(), "parent", &engine.PrInfo{
			Number: &prNum,
			Title:  "Parent PR",
			URL:    "https://github.com/owner/repo/pull/101",
			State:  "OPEN",
		})
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "child")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.NotNil(t, prCtx.ParentPRInfo)
		require.Equal(t, "Parent PR", prCtx.ParentPRInfo.Title)
		require.Equal(t, 101, *prCtx.ParentPRInfo.Number)
	})

	t.Run("collects child PR info when available", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> parent -> child
		err := scene.Repo.CreateAndCheckoutBranch("parent")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("parent change", "parent.txt")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("child")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("child change", "child.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "parent", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "child", "parent")
		require.NoError(t, err)

		// Add PR info for child
		prNum := 102
		err = eng.UpsertPrInfo(context.Background(), "child", &engine.PrInfo{
			Number: &prNum,
			Title:  "Child PR",
			URL:    "https://github.com/owner/repo/pull/102",
			State:  "OPEN",
		})
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "parent")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.NotNil(t, prCtx.ChildPRInfo)
		require.Equal(t, "Child PR", prCtx.ChildPRInfo.Title)
		require.Equal(t, 102, *prCtx.ChildPRInfo.Number)
	})

	t.Run("collects related PRs in stack", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> branch1 -> branch2 -> branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1.txt")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2.txt")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		// Add PR info for branch1 and branch3
		pr1 := 101
		err = eng.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number: &pr1,
			Title:  "Branch1 PR",
			URL:    "https://github.com/owner/repo/pull/101",
			State:  "OPEN",
		})
		require.NoError(t, err)

		pr3 := 103
		err = eng.UpsertPrInfo(context.Background(), "branch3", &engine.PrInfo{
			Number: &pr3,
			Title:  "Branch3 PR",
			URL:    "https://github.com/owner/repo/pull/103",
			State:  "OPEN",
		})
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "branch2")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		// Should find branch1 and branch3 PRs (but not branch2 itself)
		require.GreaterOrEqual(t, len(prCtx.RelatedPRs), 1)
		foundBranch1 := false
		foundBranch3 := false
		for _, relatedPR := range prCtx.RelatedPRs {
			if relatedPR.BranchName == "branch1" {
				foundBranch1 = true
				require.Equal(t, "Branch1 PR", relatedPR.Title)
			}
			if relatedPR.BranchName == "branch3" {
				foundBranch3 = true
				require.Equal(t, "Branch3 PR", relatedPR.Title)
			}
		}
		require.True(t, foundBranch1, "should find branch1 PR")
		require.True(t, foundBranch3, "should find branch3 PR")
	})

	t.Run("excludes closed PRs from related PRs", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create a stack: main -> branch1 -> branch2
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1.txt")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Add closed PR info for branch1
		pr1 := 101
		err = eng.UpsertPrInfo(context.Background(), "branch1", &engine.PrInfo{
			Number: &pr1,
			Title:  "Branch1 PR",
			State:  "CLOSED",
		})
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "branch2")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		// Should not include closed PRs
		for _, relatedPR := range prCtx.RelatedPRs {
			require.NotEqual(t, "branch1", relatedPR.BranchName, "should not include closed PR")
		}
	})

	t.Run("reads project conventions from CONTRIBUTING.md", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create CONTRIBUTING.md
		contributingPath := filepath.Join(scene.Dir, "CONTRIBUTING.md")
		err := os.WriteFile(contributingPath, []byte("# Contributing\n\nFollow these guidelines."), 0644)
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "main")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.Contains(t, prCtx.ProjectConventions, "CONTRIBUTING.md")
		require.Contains(t, prCtx.ProjectConventions, "Follow these guidelines")
		require.NotContains(t, prCtx.ProjectConventions, "README.md")
	})

	t.Run("handles missing convention files gracefully", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "main")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		// Should not fail, just have empty conventions
		require.Empty(t, prCtx.ProjectConventions)
	})

	t.Run("handles missing branch gracefully", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		_, err = ai.CollectPRContext(ctx, eng, "nonexistent")
		require.Error(t, err)
	})

	t.Run("collects commit messages correctly", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch with multiple commits
		err := scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("first change", "file1.txt")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("second change", "file2.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "feature")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.Len(t, prCtx.CommitMessages, 2)
		// Git log returns commits in reverse chronological order (newest first)
		require.Contains(t, prCtx.CommitMessages[0], "second change")
		require.Contains(t, prCtx.CommitMessages[1], "first change")
	})

	t.Run("collects changed files correctly", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create branch with multiple file changes
		err := scene.Repo.CreateAndCheckoutBranch("feature")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("change 1", "file1.txt")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("change 2", "file2.txt")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "feature", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.Context = context.Background()
		ctx.RepoRoot = scene.Dir

		prCtx, err := ai.CollectPRContext(ctx, eng, "feature")
		require.NoError(t, err)
		require.NotNil(t, prCtx)

		require.Contains(t, prCtx.ChangedFiles, "file1.txt_test.txt")
		require.Contains(t, prCtx.ChangedFiles, "file2.txt_test.txt")
	})
}
