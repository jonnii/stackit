package cli_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"stackit.dev/stackit/testhelpers"
)

func TestSubmitCommand(t *testing.T) {
	// Build the stackit binary first
	binaryPath := buildStackitBinary(t)

	t.Run("submit includes current branch in list", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		initOutput, err := cmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(initOutput))

		// Create a stack: main -> branch1 -> branch2 (current)
		// Use stackit create which automatically tracks the parent relationship
		cmd = exec.Command(binaryPath, "create", "branch1")
		cmd.Dir = scene.Dir
		createOutput, err := cmd.CombinedOutput()
		require.NoError(t, err, "create branch1 failed: %s", string(createOutput))

		// Create branch2 from branch1 (which is now current)
		cmd = exec.Command(binaryPath, "create", "branch2")
		cmd.Dir = scene.Dir
		createOutput2, err := cmd.CombinedOutput()
		require.NoError(t, err, "create branch2 failed: %s", string(createOutput2))

		// Verify we're on branch2
		currentBranch, err := scene.Repo.CurrentBranchName()
		require.NoError(t, err)
		require.Equal(t, "branch2", currentBranch)

		// Run submit command with --dry-run and --no-edit to avoid interactive prompts
		cmd = exec.Command(binaryPath, "submit", "--dry-run", "--no-edit")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		
		// Should succeed
		require.NoError(t, err, "submit command failed: %s", string(output))
		
		outputStr := string(output)
		
		// Verify all branches are in the output
		require.Contains(t, outputStr, "branch1", "should include parent branch")
		require.Contains(t, outputStr, "branch2", "should include current branch")
		
		// Verify the current branch (branch2) appears in the list
		// The output should show branch2 being prepared for submit
		lines := strings.Split(outputStr, "\n")
		foundBranch2 := false
		for _, line := range lines {
			if strings.Contains(line, "branch2") && (strings.Contains(line, "Create") || strings.Contains(line, "Update") || strings.Contains(line, "No-op")) {
				foundBranch2 = true
				break
			}
		}
		require.True(t, foundBranch2, "current branch (branch2) should appear in submit list. Output: %s", outputStr)
	})

	t.Run("submit with --stack includes descendants", func(t *testing.T) {
		scene := testhelpers.NewScene(t, nil)

		// Create initial commit
		err := scene.Repo.CreateChangeAndCommit("initial", "init")
		require.NoError(t, err)

		// Initialize stackit
		cmd := exec.Command(binaryPath, "init")
		cmd.Dir = scene.Dir
		initOutput, err := cmd.CombinedOutput()
		require.NoError(t, err, "init failed: %s", string(initOutput))

		// Create a stack: main -> branch1 -> branch2 (current) -> branch3
		// Use stackit create which automatically tracks the parent relationship
		cmd = exec.Command(binaryPath, "create", "branch1")
		cmd.Dir = scene.Dir
		createOutput1, err := cmd.CombinedOutput()
		require.NoError(t, err, "create branch1 failed: %s", string(createOutput1))

		// Create branch2
		cmd = exec.Command(binaryPath, "create", "branch2")
		cmd.Dir = scene.Dir
		createOutput2, err := cmd.CombinedOutput()
		require.NoError(t, err, "create branch2 failed: %s", string(createOutput2))

		// Create branch3 (descendant of branch2)
		cmd = exec.Command(binaryPath, "create", "branch3")
		cmd.Dir = scene.Dir
		createOutput3, err := cmd.CombinedOutput()
		require.NoError(t, err, "create branch3 failed: %s", string(createOutput3))

		// Go back to branch2
		err = scene.Repo.CheckoutBranch("branch2")
		require.NoError(t, err)

		// Run submit command with --stack and --dry-run and --no-edit to avoid interactive prompts
		cmd = exec.Command(binaryPath, "submit", "--stack", "--dry-run", "--no-edit")
		cmd.Dir = scene.Dir
		output, err := cmd.CombinedOutput()
		
		// Should succeed
		require.NoError(t, err, "submit command failed: %s", string(output))
		
		outputStr := string(output)
		
		// Verify all branches are in the output (ancestors, current, and descendants)
		require.Contains(t, outputStr, "branch1", "should include parent branch")
		require.Contains(t, outputStr, "branch2", "should include current branch")
		require.Contains(t, outputStr, "branch3", "should include descendant branch with --stack")
	})
}
