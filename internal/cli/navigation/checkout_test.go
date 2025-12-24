package navigation_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/internal/cli/testhelper"
	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestCheckoutCommand(t *testing.T) {
	t.Parallel()
	binaryPath := testhelper.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelper.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("direct checkout with branch name", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a stack: main -> a -> b
		s.RunCli("create", "a", "-m", "a").
			RunCli("create", "b", "-m", "b")

		// Now we're on branch 'b', checkout 'a'
		output, err := s.RunCliAndGetOutput("checkout", "a")
		require.NoError(t, err, "checkout command failed: %s", output)

		// Should be on branch 'a'
		s.ExpectBranch("a")
		require.Contains(t, output, "Checked out")
	})

	t.Run("checkout trunk flag", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Now we're on branch 'a', checkout trunk
		output, err := s.RunCliAndGetOutput("checkout", "--trunk")
		require.NoError(t, err, "checkout --trunk command failed: %s", output)

		// Should be on main (trunk)
		s.ExpectBranch("main")
		require.Contains(t, output, "Checked out")
	})

	t.Run("checkout trunk flag short form", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Now we're on branch 'a', checkout trunk with short flag
		output, err := s.RunCliAndGetOutput("checkout", "-t")
		require.NoError(t, err, "checkout -t command failed: %s", output)

		// Should be on main (trunk)
		s.ExpectBranch("main")
	})

	t.Run("already on branch", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Try to checkout the branch we're already on
		output, err := s.RunCliAndGetOutput("checkout", "a")
		require.NoError(t, err, "checkout command should succeed even when already on branch")

		// Should still be on branch 'a'
		s.ExpectBranch("a")
		require.Contains(t, output, "Already on")
	})

	t.Run("invalid branch", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Try to checkout a non-existent branch
		output, err := s.RunCliAndGetOutput("checkout", "nonexistent")
		require.Error(t, err, "checkout should fail for non-existent branch")
		require.Contains(t, output, "failed to checkout")
	})

	t.Run("alias co works", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a stack: main -> a -> b
		s.RunCli("create", "a", "-m", "a").
			RunCli("create", "b", "-m", "b")

		// Now we're on branch 'b', use alias 'co' to checkout 'a'
		output, err := s.RunCliAndGetOutput("co", "a")
		require.NoError(t, err, "co alias command failed: %s", output)

		// Should be on branch 'a'
		s.ExpectBranch("a")
		require.Contains(t, output, "Checked out")
	})

	t.Run("checkout with show-untracked flag", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a tracked branch
		s.RunCli("create", "a", "-m", "a")

		// Create an untracked branch (not using stackit create)
		s.Checkout("main").
			CreateBranch("untracked-branch").
			CommitChange("untracked", "untracked commit")

		// Try to checkout with --show-untracked (but no branch specified - would need interactive)
		output, err := s.RunCliAndGetOutput("checkout", "--show-untracked", "a")
		// This should work (flag is accepted, even if not used for direct checkout)
		require.NoError(t, err, "checkout with --show-untracked flag failed: %s", output)

		// Should be on branch 'a'
		s.ExpectBranch("a")
	})

	t.Run("checkout with stack flag in non-interactive mode should fail", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Try to checkout with --stack flag but no branch (would need interactive)
		output, err := s.RunCliAndGetOutput("checkout", "--stack")
		// Should fail in non-interactive mode with a clear error
		require.Error(t, err, "checkout --stack should fail in non-interactive mode")
		require.Contains(t, output, "interactive", "error should mention interactive mode")
	})

	t.Run("checkout from trunk to branch and back", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Verify we're on 'a'
		s.ExpectBranch("a")

		// Checkout trunk
		s.RunCli("checkout", "--trunk")
		s.ExpectBranch("main")

		// Checkout back to 'a'
		s.RunCli("checkout", "a")
		s.ExpectBranch("a")
	})
}
