package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// PushBranch pushes a branch to remote with optional force
// If forceWithLease is true, uses --force-with-lease (safer)
// If force is true, uses --force (overwrites remote)
// If both are false, does a normal push
func PushBranch(branchName string, remote string, force bool, forceWithLease bool) error {
	args := []string{"push", "-u", remote}

	if force {
		args = append(args, "--force")
	} else if forceWithLease {
		args = append(args, "--force-with-lease")
	}

	args = append(args, branchName)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)
		// Check for stale info error (force-with-lease failed)
		if strings.Contains(outputStr, "stale info") || strings.Contains(outputStr, "forced update") {
			return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning: %w", branchName, err)
		}
		return fmt.Errorf("failed to push branch %s: %s: %w", branchName, outputStr, err)
	}

	return nil
}
