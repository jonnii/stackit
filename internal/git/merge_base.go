package git

import (
	"fmt"
)

// GetMergeBase returns the merge base between two branches
func GetMergeBase(branch1, branch2 string) (string, error) {
	return GetMergeBaseByRef("refs/heads/"+branch1, "refs/heads/"+branch2)
}

// GetMergeBaseByRef returns the merge base between two refs (can be branches or remote refs)
func GetMergeBaseByRef(ref1Name, ref2Name string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	hash1, err := resolveRefHash(repo, ref1Name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref1: %w", err)
	}

	hash2, err := resolveRefHash(repo, ref2Name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref2: %w", err)
	}

	commit1, err := repo.CommitObject(hash1)
	if err != nil {
		return "", fmt.Errorf("failed to get commit1: %w", err)
	}

	commit2, err := repo.CommitObject(hash2)
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

// IsAncestor checks if the first ref is an ancestor of the second ref
func IsAncestor(ancestor, descendant string) (bool, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return false, err
	}

	ancestorHash, err := resolveRefHash(repo, ancestor)
	if err != nil {
		return false, fmt.Errorf("failed to resolve ancestor ref: %w", err)
	}

	descendantHash, err := resolveRefHash(repo, descendant)
	if err != nil {
		return false, fmt.Errorf("failed to resolve descendant ref: %w", err)
	}

	// If they're the same, ancestor is an ancestor
	if ancestorHash == descendantHash {
		return true, nil
	}

	// Get commit objects
	ancestorCommit, err := repo.CommitObject(ancestorHash)
	if err != nil {
		return false, fmt.Errorf("failed to get ancestor commit: %w", err)
	}

	descendantCommit, err := repo.CommitObject(descendantHash)
	if err != nil {
		return false, fmt.Errorf("failed to get descendant commit: %w", err)
	}

	return ancestorCommit.IsAncestor(descendantCommit)
}
