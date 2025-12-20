package git

import (
	"context"
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetMergeBase returns the merge base between two branches
func GetMergeBase(ctx context.Context, branch1, branch2 string) (string, error) {
	return GetMergeBaseByRef(ctx, "refs/heads/"+branch1, "refs/heads/"+branch2)
}

// GetMergeBaseByRef returns the merge base between two refs (can be branches or remote refs)
func GetMergeBaseByRef(ctx context.Context, ref1Name, ref2Name string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	ref1, err := repo.Reference(plumbing.ReferenceName(ref1Name), true)
	if err != nil {
		return "", fmt.Errorf("failed to get ref1 reference: %w", err)
	}

	ref2, err := repo.Reference(plumbing.ReferenceName(ref2Name), true)
	if err != nil {
		return "", fmt.Errorf("failed to get ref2 reference: %w", err)
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

// IsAncestor checks if the first ref is an ancestor of the second ref
func IsAncestor(ctx context.Context, ancestor, descendant string) (bool, error) {
	// We can use the shell command here as it's very efficient and handles various ref types
	_, err := RunGitCommandWithContext(ctx, "merge-base", "--is-ancestor", ancestor, descendant)
	if err == nil {
		return true, nil
	}
	// Exit code 1 means it's not an ancestor
	return false, nil
}
