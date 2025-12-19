package git

import (
	"context"
	"fmt"
	"strings"
)

// PushBranch pushes a branch to remote with optional force
// If forceWithLease is true, uses --force-with-lease (safer)
// If force is true, uses --force (overwrites remote)
// If both are false, does a normal push
func PushBranch(ctx context.Context, branchName string, remote string, force bool, forceWithLease bool) error {
	args := []string{"push", "-u", remote}

	if force {
		args = append(args, "--force")
	} else if forceWithLease {
		args = append(args, "--force-with-lease")
	}

	args = append(args, branchName)

	_, err := RunGitCommandWithContext(ctx, args...)
	if err != nil {
		if strings.Contains(err.Error(), "stale info") || strings.Contains(err.Error(), "forced update") {
			return fmt.Errorf("%w: force-with-lease push of %s failed due to external changes to the remote branch", ErrStaleRemoteInfo, branchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", branchName, err)
	}

	return nil
}
