package navigation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stackit.dev/stackit/testhelpers"
	"stackit.dev/stackit/testhelpers/scenario"
)

func TestLogCommand(t *testing.T) {
	t.Parallel()
	// Build the stackit binary first
	binaryPath := testhelpers.GetSharedBinaryPath()
	if binaryPath == "" {
		if err := testhelpers.GetBinaryError(); err != nil {
			t.Fatalf("failed to build stackit binary: %v", err)
		}
		t.Fatal("stackit binary not built")
	}

	t.Run("log in empty repo", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Run log command
		output, err := s.RunCliAndGetOutput("log")

		// Should succeed and show trunk branch
		require.NoError(t, err, "log command failed: %s", output)
		require.Contains(t, output, "main")
	})

	t.Run("log with branches", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create a branch
		s.CreateBranch("feature").
			CommitChange("feature", "feature commit")

		// Checkout main
		s.Checkout("main")

		// Run log command with --show-untracked to see untracked branches
		output, err := s.RunCliAndGetOutput("log", "--show-untracked")

		require.NoError(t, err, "log command failed: %s", output)
		require.Contains(t, output, "main")
		require.Contains(t, output, "feature")
	})

	t.Run("log with --reverse flag", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Run log command with reverse
		output, err := s.RunCliAndGetOutput("log", "--reverse")

		require.NoError(t, err, "log command failed: %s", output)
		require.Contains(t, output, "main")
	})

	t.Run("log with --stack flag", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		// Create and checkout a branch
		s.CreateBranch("feature")

		// Run log command with stack
		output, err := s.RunCliAndGetOutput("log", "--stack")

		require.NoError(t, err, "log command failed: %s", output)
		require.Contains(t, output, "feature")
	})

	t.Run("log subcommands and top-level aliases", func(t *testing.T) {
		t.Parallel()
		s := scenario.NewScenarioParallel(t, testhelpers.BasicSceneSetup).WithBinaryPath(binaryPath)

		commands := []string{"log short", "log long", "ls", "ll", "log ls", "log ll"}
		for _, cmdStr := range commands {
			t.Run(cmdStr, func(t *testing.T) {
				args := strings.Fields(cmdStr)
				output, err := s.RunCliAndGetOutput(args...)
				require.NoError(t, err, "%s command failed: %s", cmdStr, output)
				require.Contains(t, output, "main")
			})
		}
	})
}
