package git

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// MetadataRefPrefix is the prefix for Git refs where branch metadata is stored
	MetadataRefPrefix = "refs/stackit/metadata/"

	// DeprecatedMetadataRefPrefix is the old prefix for Git refs where branch metadata was stored.
	// Deprecated: Use MetadataRefPrefix instead. This will be removed in a future release.
	DeprecatedMetadataRefPrefix = "refs/branch-metadata/"
)

// Meta represents branch metadata stored in Git refs
type Meta struct {
	ParentBranchName     *string `json:"parentBranchName,omitempty"`
	ParentBranchRevision *string `json:"parentBranchRevision,omitempty"`
	PrInfo               *PrInfo `json:"prInfo,omitempty"`
}

// PrInfo represents PR information
type PrInfo struct {
	Number  *int    `json:"number,omitempty"`
	Base    *string `json:"base,omitempty"`
	URL     *string `json:"url,omitempty"`
	Title   *string `json:"title,omitempty"`
	Body    *string `json:"body,omitempty"`
	State   *string `json:"state,omitempty"`
	IsDraft *bool   `json:"isDraft,omitempty"`
}

// ReadMetadataRef reads metadata for a branch from Git refs
func ReadMetadataRef(branchName string) (*Meta, error) {
	// Use shell git commands for consistency with WriteMetadataRef and DeleteMetadataRef
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)

	// Get the SHA of the ref
	sha, err := RunGitCommand("rev-parse", "--verify", refName)
	if err != nil {
		// Try deprecated ref
		deprecatedRefName := fmt.Sprintf("%s%s", DeprecatedMetadataRefPrefix, branchName)
		sha, err = RunGitCommand("rev-parse", "--verify", deprecatedRefName)
		if err != nil {
			// Ref doesn't exist - return empty meta
			return &Meta{}, nil //nolint:nilerr
		}

		// Found deprecated ref, migrate it lazily
		// We'll read the content first to ensure it's valid
		content, err := RunGitCommand("cat-file", "-p", sha)
		if err == nil {
			var meta Meta
			if err := json.Unmarshal([]byte(content), &meta); err == nil {
				// Migrate to new ref
				_ = WriteMetadataRef(branchName, &meta)
				// Delete old ref
				_, _ = RunGitCommand("update-ref", "-d", deprecatedRefName)
				return &meta, nil
			}
		}
	}

	// Get the content of the blob
	content, err := RunGitCommand("cat-file", "-p", sha)
	if err != nil {
		return &Meta{}, nil //nolint:nilerr
	}

	var meta Meta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return &Meta{}, nil //nolint:nilerr
	}

	return &meta, nil
}

// GetMetadataRefList returns all metadata refs
func GetMetadataRefList() (map[string]string, error) {
	result := make(map[string]string)

	// Get current metadata refs
	output, err := RunGitCommand("for-each-ref", "--format=%(refname) %(objectname)", MetadataRefPrefix)
	if err == nil && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
			refName := parts[0]
			sha := parts[1]

			if strings.HasPrefix(refName, MetadataRefPrefix) {
				branchName := refName[len(MetadataRefPrefix):]
				result[branchName] = sha
			}
		}
	}

	// Also get deprecated metadata refs
	// Deprecated: This section supports migration from the old prefix.
	output, err = RunGitCommand("for-each-ref", "--format=%(refname) %(objectname)", DeprecatedMetadataRefPrefix)
	if err == nil && output != "" {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, " ", 2)
			if len(parts) != 2 {
				continue
			}
			refName := parts[0]
			sha := parts[1]

			if strings.HasPrefix(refName, DeprecatedMetadataRefPrefix) {
				branchName := refName[len(DeprecatedMetadataRefPrefix):]
				// Only add if not already present in the new prefix (new prefix takes precedence)
				if _, exists := result[branchName]; !exists {
					result[branchName] = sha
				}
			}
		}
	}

	return result, nil
}

// DeleteMetadataRef deletes a metadata ref for a branch
func DeleteMetadataRef(branchName string) error {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	_, err := RunGitCommand("update-ref", "-d", refName)

	// Also delete deprecated ref if it exists
	deprecatedRefName := fmt.Sprintf("%s%s", DeprecatedMetadataRefPrefix, branchName)
	_, _ = RunGitCommand("update-ref", "-d", deprecatedRefName)

	return err
}

// WriteMetadataRef writes metadata for a branch to Git refs
func WriteMetadataRef(branchName string, meta *Meta) error {
	// Marshal metadata to JSON
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create blob using git hash-object
	sha, err := RunGitCommandWithInput(string(jsonData), "hash-object", "-w", "--stdin")
	if err != nil {
		return fmt.Errorf("failed to create metadata blob: %w", err)
	}

	// Create or update the ref using git update-ref
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	_, err = RunGitCommand("update-ref", refName, sha)
	if err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	return nil
}
