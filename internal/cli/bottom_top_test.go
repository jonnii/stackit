package cli_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestBottomCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("bottom from middle of stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create stack: main -> a -> b -> c
		s.RunCli("create", "a", "-m", "a").
			RunCli("create", "b", "-m", "b").
			RunCli("create", "c", "-m", "c")

		// Now we're on branch 'c', run bottom command
		s.RunCli("bottom")

		// Should be on branch 'a' (first branch from trunk)
		s.ExpectBranch("a")
	})

	t.Run("bottom from first branch from trunk", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Run bottom from 'a' (already at bottom)
		output, err := s.RunCliAndGetOutput("bottom")
		require.NoError(t, err)

		// Should still be on branch 'a'
		s.ExpectBranch("a")
		require.Contains(t, output, "Already at the bottom most")
	})

	t.Run("bottom from trunk", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Run bottom from trunk
		s.RunCli("bottom")

		// Should still be on main
		s.ExpectBranch("main")
	})

	t.Run("bottom with single branch stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a single branch
		s.RunCli("create", "feature", "-m", "feature")

		// Run bottom from feature
		s.RunCli("bottom")

		// Should be on feature (it's the first branch from trunk)
		s.ExpectBranch("feature")
	})
}

func TestTopCommand(t *testing.T) {
	t.Parallel()
	binaryPath := getStackitBinary(t)

	t.Run("top from middle of stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create stack: main -> a -> b -> c
		s.RunCli("create", "a", "-m", "a").
			RunCli("create", "b", "-m", "b").
			RunCli("create", "c", "-m", "c")

		// Now we're on branch 'c', go back to 'a' and run top command
		s.Checkout("a")
		s.RunCli("top")

		// Should be on branch 'c' (tip of stack)
		s.ExpectBranch("c")
	})

	t.Run("top from tip of stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Run top from 'a' (already at top)
		output, err := s.RunCliAndGetOutput("top")
		require.NoError(t, err)

		// Should still be on branch 'a'
		s.ExpectBranch("a")
		require.Contains(t, output, "Already at the top most")
	})

	t.Run("top from trunk", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.RunCli("create", "a", "-m", "a")

		// Go back to main and run top from trunk
		s.Checkout("main")
		s.RunCli("top")

		// Should be on branch 'a' (tip)
		s.ExpectBranch("a")
	})

	t.Run("top with single branch stack", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a single branch
		s.RunCli("create", "feature", "-m", "feature")

		// Run top from feature
		s.RunCli("top")

		// Should be on feature (it's the tip)
		s.ExpectBranch("feature")
	})

	t.Run("top with multiple children fails in non-interactive mode", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create branch 'a'
		s.RunCli("create", "a", "-m", "a")

		// Create branch 'b' from 'a'
		s.RunCli("create", "b", "-m", "b")

		// Go back to 'a' and create branch 'c' from 'a'
		s.Checkout("a")
		s.RunCli("create", "c", "-m", "c")

		// Run top from 'a' in non-interactive mode
		s.Checkout("a")

		// Redirect stdin to /dev/null to simulate non-interactive mode
		// NewScenario already sets STACKIT_NON_INTERACTIVE=true, but we also want to test
		// what happens when multiple branches are found.

		cmd := "top"
		output, err := s.RunCliAndGetOutput(cmd)
		require.Error(t, err)
		require.Contains(t, output, "multiple branches found")
		require.Contains(t, output, "b")
		require.Contains(t, output, "c")
	})
}
