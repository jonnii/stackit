package git

import (
	"fmt"
	"strings"
)

// GetCommitRange returns commits in a range in various formats
// base: parent branch revision (or empty string for trunk)
// head: branch revision
// format: "SHA", "READABLE" (oneline), "MESSAGE" (full), "SUBJECT" (first line)
func GetCommitRange(base string, head string, format string) ([]string, error) {
	var formatArg string
	switch format {
	case "SHA":
		formatArg = "%H"
	case "READABLE":
		// Oneline format: short SHA + subject ("%h - %s")
		formatArg = "%h - %s"
	case "MESSAGE":
		formatArg = "%B"
	case "SUBJECT":
		formatArg = "%s"
	default:
		return nil, fmt.Errorf("unknown commit format: %s", format)
	}

	if base != "" {
		// Get commits from base to head
		// First get all SHAs in the range
		shaOutput, err := RunGitCommand("log", "--pretty=format:%H", fmt.Sprintf("%s..%s", base, head))
		if err != nil {
			return nil, fmt.Errorf("failed to get commit SHAs: %w", err)
		}

		if shaOutput == "" {
			return []string{}, nil
		}

		shas := strings.Split(strings.TrimSpace(shaOutput), "\n")
		result := []string{}

		// For each SHA, get the formatted output
		for _, sha := range shas {
			sha = strings.TrimSpace(sha)
			if sha == "" {
				continue
			}

			commitOutput, err := RunGitCommand("log", "-1", "--pretty=format:"+formatArg, sha)
			if err != nil {
				return nil, fmt.Errorf("failed to get commit %s: %w", sha, err)
			}

			commitOutput = strings.TrimSpace(commitOutput)
			if commitOutput != "" {
				result = append(result, commitOutput)
			}
		}

		return result, nil
	}

	// For trunk (no base), get just the one commit
	output, err := RunGitCommand("log", "-1", "--pretty=format:"+formatArg, head)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	if output == "" {
		return []string{}, nil
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}, nil
	}

	return []string{output}, nil
}

// GetCommitSHA returns the SHA at a relative position (0 = HEAD, 1 = HEAD~1)
// This is relative to the current branch or the specified branch
func GetCommitSHA(branchName string, offset int) (string, error) {
	if offset < 0 {
		return "", fmt.Errorf("offset must be non-negative")
	}

	ref := branchName
	if offset > 0 {
		ref = fmt.Sprintf("%s~%d", branchName, offset)
	}

	sha, err := RunGitCommand("rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("failed to get commit SHA: %w", err)
	}

	return sha, nil
}
