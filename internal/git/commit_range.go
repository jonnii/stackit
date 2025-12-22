package git

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetCommitRange returns commits in a range in various formats
// base: parent branch revision (or empty string for trunk)
// head: branch revision
// format: "SHA", "READABLE" (oneline), "MESSAGE" (full), "SUBJECT" (first line)
func GetCommitRange(base string, head string, format string) ([]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	headHash, err := resolveRefHash(repo, head)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head: %w", err)
	}

	var baseHash plumbing.Hash
	if base != "" {
		baseHash, err = resolveRefHash(repo, base)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve base: %w", err)
		}
	}

	var commits []*object.Commit
	commits, err = iterateCommits(repo, headHash, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	result := make([]string, 0, len(commits))
	for _, commit := range commits {
		var formatted string
		switch format {
		case "SHA":
			formatted = commit.Hash.String()
		case "READABLE":
			// Oneline format: short SHA + subject
			shortHash := commit.Hash.String()[:7]
			subject := strings.Split(strings.TrimSpace(commit.Message), "\n")[0]
			formatted = fmt.Sprintf("%s - %s", shortHash, subject)
		case "MESSAGE":
			formatted = strings.TrimSpace(commit.Message)
		case "SUBJECT":
			subject := strings.Split(strings.TrimSpace(commit.Message), "\n")[0]
			formatted = strings.TrimSpace(subject)
		default:
			return nil, fmt.Errorf("unknown commit format: %s", format)
		}

		if formatted != "" {
			result = append(result, formatted)
		}
	}

	return result, nil
}

// GetCommitSHA returns the SHA at a relative position (0 = HEAD, 1 = HEAD~1)
// This is relative to the current branch or the specified branch
func GetCommitSHA(branchName string, offset int) (string, error) {
	if offset < 0 {
		return "", fmt.Errorf("offset must be non-negative")
	}

	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	// Resolve branch reference
	branchRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		return "", fmt.Errorf("failed to get branch reference: %w", err)
	}

	commit, err := repo.CommitObject(branchRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	// Walk back offset number of commits
	for i := 0; i < offset; i++ {
		if commit.NumParents() == 0 {
			return "", fmt.Errorf("commit has no parent at offset %d", i)
		}
		// Get first parent
		commit, err = commit.Parent(0)
		if err != nil {
			return "", fmt.Errorf("failed to get parent commit: %w", err)
		}
	}

	return commit.Hash.String(), nil
}
