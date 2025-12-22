package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

// HardReset performs a hard reset to a specific SHA
func HardReset(ctx context.Context, sha string) error {
	_, err := RunGitCommandWithContext(ctx, "reset", "--hard", sha)
	if err != nil {
		return fmt.Errorf("failed to hard reset to %s: %w", sha, err)
	}
	return nil
}

// SoftReset performs a soft reset to a specific SHA
func SoftReset(ctx context.Context, sha string) error {
	_, err := RunGitCommandWithContext(ctx, "reset", "-q", "--soft", sha)
	if err != nil {
		return fmt.Errorf("failed to soft reset to %s: %w", sha, err)
	}
	return nil
}

// GetRemoteSha returns the SHA of a remote branch
func GetRemoteSha(_ context.Context, remote, branchName string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remote, branchName))
	ref, err := repo.Reference(refName, true)
	if err != nil {
		return "", fmt.Errorf("failed to get remote SHA for %s/%s: %w", remote, branchName, err)
	}

	return ref.Hash().String(), nil
}
