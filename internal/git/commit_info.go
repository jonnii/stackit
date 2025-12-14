package git

import (
	"fmt"
	"strings"
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

// GetCommitMessages returns all commit messages for a branch (excluding parent)
func GetCommitMessages(branchName string) ([]string, error) {
	// Get parent branch to determine range
	meta, err := ReadMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	var args []string
	if meta.ParentBranchRevision != nil {
		// Get commits from parent to branch
		args = []string{"log", "--format=%B", fmt.Sprintf("%s..%s", *meta.ParentBranchRevision, branchName)}
	} else {
		// No parent, get all commits from branch
		args = []string{"log", "--format=%B", branchName}
	}

	output, err := RunGitCommand(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit messages: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	// Split by double newline (commit separator) and filter empty
	commits := strings.Split(output, "\n\n")
	var messages []string
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit != "" {
			messages = append(messages, commit)
		}
	}

	return messages, nil
}

// GetCommitSubject returns the subject (first line) of the oldest commit on a branch
func GetCommitSubject(branchName string) (string, error) {
	// Get parent branch to determine range
	meta, err := ReadMetadataRef(branchName)
	if err != nil {
		return "", err
	}

	var args []string
	if meta.ParentBranchRevision != nil {
		// Get oldest commit subject from parent to branch
		args = []string{"log", "--format=%s", "-1", fmt.Sprintf("%s..%s", *meta.ParentBranchRevision, branchName)}
	} else {
		// No parent, get oldest commit from branch
		args = []string{"log", "--format=%s", "-1", branchName}
	}

	subject, err := RunGitCommand(args...)
	if err != nil {
		return "", fmt.Errorf("failed to get commit subject: %w", err)
	}

	return strings.TrimSpace(subject), nil
}
