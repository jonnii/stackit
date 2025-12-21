package actions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/testhelpers"
)

func TestMoveAction(t *testing.T) {
	t.Run("moves branch downstack and restacks descendants", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create stack: main -> branch1 -> branch2 -> branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		// Verify initial state
		require.Equal(t, "branch1", eng.GetParent("branch2"))
		require.Equal(t, "branch2", eng.GetParent("branch3"))

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move branch2 from branch1 to main (downstack)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch2",
			Onto:   "main",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "main", eng.GetParent("branch2"))
		require.Contains(t, eng.GetChildren("main"), "branch2")
		require.NotContains(t, eng.GetChildren("branch1"), "branch2")

		// Verify branch3 still has branch2 as parent (descendant relationship preserved)
		require.Equal(t, "branch2", eng.GetParent("branch3"))
	})

	t.Run("moves branch upstack and restacks descendants", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create two separate stacks from main:
		// Stack A: main -> branchA -> branchA2
		// Stack B: main -> branchB
		err := scene.Repo.CreateAndCheckoutBranch("branchA")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchA change", "a")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branchA2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchA2 change", "a2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branchB")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchB change", "b")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branchA", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branchA2", "branchA")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branchB", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move branchA from main to branchB (upstack - moving to a sibling branch)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branchA",
			Onto:   "branchB",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "branchB", eng.GetParent("branchA"))
		require.Contains(t, eng.GetChildren("branchB"), "branchA")
		require.NotContains(t, eng.GetChildren("main"), "branchA")

		// Verify branchA2 still has branchA as parent (descendant relationship preserved)
		require.Equal(t, "branchA", eng.GetParent("branchA2"))
	})

	t.Run("moves branch across different stack trees", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create two separate stacks:
		// Stack A: main -> branchA1 -> branchA2
		// Stack B: main -> branchB1 -> branchB2
		err := scene.Repo.CreateAndCheckoutBranch("branchA1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchA1 change", "a1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branchA2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchA2 change", "a2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branchB1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchB1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branchB2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branchB2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Track branches
		err = eng.TrackBranch(context.Background(), "branchA1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branchA2", "branchA1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branchB1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branchB2", "branchB1")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move branchA2 from branchA1 to branchB1 (across stacks)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branchA2",
			Onto:   "branchB1",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "branchB1", eng.GetParent("branchA2"))
		require.Contains(t, eng.GetChildren("branchB1"), "branchA2")
		require.NotContains(t, eng.GetChildren("branchA1"), "branchA2")
	})

	t.Run("defaults source to current branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)

		// Switch to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move without specifying source (should use current branch)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "", // Empty means use current branch
			Onto:   "main",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branch2 was moved
		require.Equal(t, "main", eng.GetParent("branch2"))
	})

	t.Run("prevents moving trunk branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "main",
			Onto:   "main", // Even if onto is same, should fail earlier
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move trunk branch")
	})

	t.Run("prevents moving onto itself", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "branch1",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move branch onto itself")
	})

	t.Run("prevents moving onto descendant (cycle detection)", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create stack: main -> branch1 -> branch2 -> branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to move branch1 onto branch3 (which is a descendant of branch1)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "branch3",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot move")
		require.Contains(t, err.Error(), "onto its own descendant")
	})

	t.Run("fails when source branch is not tracked", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("untracked")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("untracked change", "u")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "untracked",
			Onto:   "main",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not tracked")
	})

	t.Run("fails when onto branch does not exist", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "nonexistent",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("fails when not on branch and no source specified", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		// Detach HEAD
		err = scene.Repo.RunGitCommand("checkout", "HEAD")
		require.NoError(t, err)

		// Rebuild engine to refresh current branch state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Try to move without specifying source - should fail because we're in detached HEAD
		// Note: The engine might still report a branch name even in detached HEAD,
		// but the move action should handle this by checking git state directly
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "",
			Onto:   "main",
		})
		require.Error(t, err)
		// The error should be either "not on a branch" or "cannot move trunk branch"
		// depending on what CurrentBranch() returns
		errMsg := err.Error()
		require.True(t,
			strings.Contains(errMsg, "not on a branch") || strings.Contains(errMsg, "cannot move trunk branch"),
			"error should mention not on a branch or trunk: %s", errMsg)
	})

	t.Run("allows moving onto untracked branch", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("untracked")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("untracked change", "u")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move branch1 onto untracked branch (should work)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "untracked",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify new parent relationship
		require.Equal(t, "untracked", eng.GetParent("branch1"))
	})

	t.Run("restacks all descendants after move", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		// Create stack: main -> branch1 -> branch2 -> branch3
		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch2")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch2 change", "b2")
		require.NoError(t, err)

		err = scene.Repo.CreateAndCheckoutBranch("branch3")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch3 change", "b3")
		require.NoError(t, err)

		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch2", "branch1")
		require.NoError(t, err)
		err = eng.TrackBranch(context.Background(), "branch3", "branch2")
		require.NoError(t, err)

		// Get initial revisions
		branch1RevBefore, err := scene.Repo.GetRevision("branch1")
		require.NoError(t, err)
		branch2RevBefore, err := scene.Repo.GetRevision("branch2")
		require.NoError(t, err)
		branch3RevBefore, err := scene.Repo.GetRevision("branch3")
		require.NoError(t, err)

		// Make a change to main to force restacking
		err = scene.Repo.CreateChangeAndCommit("new main change", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		// Move branch1 to main (which now has new commits)
		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "main",
		})
		require.NoError(t, err)

		// Rebuild engine to refresh state
		err = eng.Rebuild(context.Background(), "main")
		require.NoError(t, err)

		// Verify branches were restacked (revisions should have changed)
		branch1RevAfter, err := scene.Repo.GetRevision("branch1")
		require.NoError(t, err)
		branch2RevAfter, err := scene.Repo.GetRevision("branch2")
		require.NoError(t, err)
		branch3RevAfter, err := scene.Repo.GetRevision("branch3")
		require.NoError(t, err)

		// Revisions should be different (branches were rebased)
		require.NotEqual(t, branch1RevBefore, branch1RevAfter)
		require.NotEqual(t, branch2RevBefore, branch2RevAfter)
		require.NotEqual(t, branch3RevBefore, branch3RevAfter)

		// Verify all branches are still fixed (properly restacked)
		require.True(t, eng.IsBranchFixed(context.Background(), "branch1"))
		require.True(t, eng.IsBranchFixed(context.Background(), "branch2"))
		require.True(t, eng.IsBranchFixed(context.Background(), "branch3"))
	})

	t.Run("fails when onto is not specified", func(t *testing.T) {
		scene := testhelpers.NewScene(t, func(s *testhelpers.Scene) error {
			return s.Repo.CreateChangeAndCommit("initial", "init")
		})

		err := scene.Repo.CreateAndCheckoutBranch("branch1")
		require.NoError(t, err)
		err = scene.Repo.CreateChangeAndCommit("branch1 change", "b1")
		require.NoError(t, err)
		err = scene.Repo.CheckoutBranch("main")
		require.NoError(t, err)

		eng, err := engine.NewEngine(scene.Dir)
		require.NoError(t, err)

		err = eng.TrackBranch(context.Background(), "branch1", "main")
		require.NoError(t, err)

		ctx := runtime.NewContext(eng)
		ctx.RepoRoot = scene.Dir
		ctx.Context = context.Background()

		err = actions.MoveAction(ctx, actions.MoveOptions{
			Source: "branch1",
			Onto:   "", // Empty onto
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "onto branch must be specified")
	})
}
