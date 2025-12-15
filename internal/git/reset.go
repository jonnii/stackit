package git

import (
	"fmt"
)

// HardReset performs a hard reset to a specific SHA
func HardReset(sha string) error {
	_, err := RunGitCommand("reset", "--hard", sha)
	if err != nil {
		return fmt.Errorf("failed to hard reset to %s: %w", sha, err)
	}
	return nil
}

// SoftReset performs a soft reset to a specific SHA
func SoftReset(sha string) error {
	_, err := RunGitCommand("reset", "-q", "--soft", sha)
	if err != nil {
		return fmt.Errorf("failed to soft reset to %s: %w", sha, err)
	}
	return nil
}

// GetRemoteSha returns the SHA of a remote branch
func GetRemoteSha(remote, branchName string) (string, error) {
	sha, err := RunGitCommand("rev-parse", fmt.Sprintf("%s/%s", remote, branchName))
	if err != nil {
		return "", fmt.Errorf("failed to get remote SHA for %s/%s: %w", remote, branchName, err)
	}
	return sha, nil
}
