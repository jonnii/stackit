package git

import (
	"fmt"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
)

// GetCommitDate returns the commit date for a branch
func GetCommitDate(branchName string) (time.Time, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return time.Time{}, err
	}

	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get branch reference: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.When, nil
}

// GetCommitAuthor returns the commit author for a branch
func GetCommitAuthor(branchName string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		return "", fmt.Errorf("failed to get branch reference: %w", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return commit.Author.Name, nil
}

// GetRevision returns the SHA of a branch
func GetRevision(branchName string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		return "", fmt.Errorf("failed to get branch reference: %w", err)
	}

	return ref.Hash().String(), nil
}
