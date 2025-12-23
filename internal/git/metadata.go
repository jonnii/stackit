package git

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

const (
	// MetadataRefPrefix is the prefix for Git refs where branch metadata is stored
	MetadataRefPrefix = "refs/stackit/metadata/"
)

// Meta represents branch metadata stored in Git refs
type Meta struct {
	ParentBranchName     *string `json:"parentBranchName,omitempty"`
	ParentBranchRevision *string `json:"parentBranchRevision,omitempty"`
	PrInfo               *PrInfo `json:"prInfo,omitempty"`
	Scope                *string `json:"scope,omitempty"`
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
	repo, err := GetDefaultRepo()
	if err != nil {
		return &Meta{}, nil //nolint:nilerr
	}

	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)

	// Get the SHA of the ref
	ref, err := repo.Reference(plumbing.ReferenceName(refName), true)
	if err != nil {
		// Ref doesn't exist - return empty meta
		return &Meta{}, nil //nolint:nilerr
	}

	// Get the content of the blob
	blob, err := repo.BlobObject(ref.Hash())
	if err != nil {
		return &Meta{}, nil //nolint:nilerr
	}

	reader, err := blob.Reader()
	if err != nil {
		return &Meta{}, nil //nolint:nilerr
	}
	defer func() {
		_ = reader.Close()
	}()

	content, err := io.ReadAll(reader)
	if err != nil {
		return &Meta{}, nil //nolint:nilerr
	}

	var meta Meta
	if err := json.Unmarshal(content, &meta); err != nil {
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

	return result, nil
}

// DeleteMetadataRef deletes a metadata ref for a branch
func DeleteMetadataRef(branchName string) error {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	_, err := RunGitCommand("update-ref", "-d", refName)

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

// RenameMetadataRef renames a metadata ref from one branch name to another
func RenameMetadataRef(oldBranchName, newBranchName string) error {
	oldRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, oldBranchName)
	newRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, newBranchName)

	// Get the SHA of the old ref
	sha, err := RunGitCommand("rev-parse", "--verify", oldRefName)
	if err != nil {
		return nil //nolint:nilerr // Nothing to rename
	}

	// Create the new ref
	_, err = RunGitCommand("update-ref", newRefName, sha)
	if err != nil {
		return fmt.Errorf("failed to create new metadata ref: %w", err)
	}

	// Delete the old ref
	_, err = RunGitCommand("update-ref", "-d", oldRefName)
	if err != nil {
		return fmt.Errorf("failed to delete old metadata ref: %w", err)
	}

	return nil
}
