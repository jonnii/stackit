package git

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Repository wraps a go-git repository
type Repository struct {
	*git.Repository
	path string
}

// OpenRepository opens a git repository at the given path
func OpenRepository(path string) (*Repository, error) {
	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Find .git directory (could be .git or a worktree)
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Try parent directory (for bare repos or submodules)
		parentGitDir := filepath.Join(filepath.Dir(absPath), ".git")
		if _, err := os.Stat(parentGitDir); err == nil {
			gitDir = parentGitDir
		}
	}

	// Open repository
	repo, err := git.PlainOpenWithOptions(absPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	return &Repository{
		Repository: repo,
		path:       absPath,
	}, nil
}

// GetRepoRoot returns the root directory of the repository
func (r *Repository) GetRepoRoot() string {
	return r.path
}

// GetReference returns a reference by name
func (r *Repository) GetReference(name string) (*plumbing.Reference, error) {
	return r.Reference(plumbing.ReferenceName(name), true)
}

// GetBranchNames returns all branch names
func (r *Repository) GetBranchNames() ([]string, error) {
	branches, err := r.Branches()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var names []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			names = append(names, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate branches: %w", err)
	}

	return names, nil
}

// GetCurrentBranch returns the current branch name
func (r *Repository) GetCurrentBranch() (string, error) {
	head, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not on a branch")
	}

	return head.Name().Short(), nil
}
