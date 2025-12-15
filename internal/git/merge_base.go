package git

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetMergeBase returns the merge base between two branches
func GetMergeBase(branch1, branch2 string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	ref1, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branch1), true)
	if err != nil {
		return "", fmt.Errorf("failed to get branch1 reference: %w", err)
	}

	ref2, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branch2), true)
	if err != nil {
		return "", fmt.Errorf("failed to get branch2 reference: %w", err)
	}

	var commit1 *object.Commit
	commit1, err = repo.CommitObject(ref1.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit1: %w", err)
	}

	var commit2 *object.Commit
	commit2, err = repo.CommitObject(ref2.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit2: %w", err)
	}

	// Find merge base
	mergeBases, err := commit1.MergeBase(commit2)
	if err != nil {
		return "", fmt.Errorf("failed to find merge base: %w", err)
	}

	if len(mergeBases) == 0 {
		return "", fmt.Errorf("no merge base found")
	}

	return mergeBases[0].Hash.String(), nil
}
