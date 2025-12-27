package git

import (
	"fmt"
	"os"
	"strings"

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

// GetRef returns the SHA of a ref
func GetRef(name string) (string, error) {
	return RunGitCommand("rev-parse", "--verify", name)
}

// UpdateRef updates a ref to point to a SHA
func UpdateRef(name, sha string) error {
	_, err := RunGitCommand("update-ref", name, sha)
	return err
}

// DeleteRef deletes a ref
func DeleteRef(name string) error {
	_, err := RunGitCommand("update-ref", "-d", name)
	return err
}

// CreateBlob creates a blob and returns its SHA
func CreateBlob(content string) (string, error) {
	return RunGitCommandWithInput(content, "hash-object", "-w", "--stdin")
}

// ReadBlob returns the content of a blob
func ReadBlob(sha string) (string, error) {
	return RunGitCommand("cat-file", "-p", sha)
}

// ListRefs returns all refs matching a prefix
func ListRefs(prefix string) (map[string]string, error) {
	result := make(map[string]string)
	output, err := RunGitCommand("for-each-ref", "--format=%(refname) %(objectname)", prefix)
	if err != nil || output == "" {
		return result, err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}
