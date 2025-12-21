package git

import (
	"fmt"
	"os"

	gogit "github.com/go-git/go-git/v5"
)

// GetRepoRoot returns the root directory of the Git repository
func GetRepoRoot() (string, error) {
	// Try to open repository from current directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Use go-git to find the repository
	repo, err := gogit.PlainOpenWithOptions(wd, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	// Get the worktree to find the root
	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	return worktree.Filesystem.Root(), nil
}
