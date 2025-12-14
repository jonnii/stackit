package git

import (
	"fmt"
)

var defaultRepo *Repository

// InitDefaultRepo initializes the default repository from the current directory
func InitDefaultRepo() error {
	if defaultRepo != nil {
		return nil // Already initialized
	}

	repoRoot, err := GetRepoRoot()
	if err != nil {
		return err
	}

	repo, err := OpenRepository(repoRoot)
	if err != nil {
		return err
	}

	defaultRepo = repo
	return nil
}

// GetDefaultRepo returns the default repository (must call InitDefaultRepo first)
func GetDefaultRepo() (*Repository, error) {
	if defaultRepo == nil {
		return nil, fmt.Errorf("repository not initialized, call InitDefaultRepo first")
	}
	return defaultRepo, nil
}

// GetAllBranchNames returns all branch names in the repository
func GetAllBranchNames() ([]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}
	return repo.GetBranchNames()
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}
	return repo.GetCurrentBranch()
}
