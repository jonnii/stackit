package git

import (
	"encoding/json"
	"fmt"
	"strings"
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
	refName := fmt.Sprintf("refs/branch-metadata/%s", branchName)

	// Get the SHA of the ref
	sha, err := RunGitCommand("rev-parse", "--verify", refName)
	if err != nil {
		// Ref doesn't exist - return empty meta
		return &Meta{}, nil
	}

	// Get the content of the blob
	content, err := RunGitCommand("cat-file", "-p", sha)
	if err != nil {
		return &Meta{}, nil
	}

	var meta Meta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return &Meta{}, nil
	}

	return &meta, nil
}

// GetMetadataRefList returns all metadata refs
func GetMetadataRefList() (map[string]string, error) {
	// Use shell git to list refs (consistent with WriteMetadataRef which uses shell git)
	output, err := RunGitCommand("for-each-ref", "--format=%(refname) %(objectname)", "refs/branch-metadata/")
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata refs: %w", err)
	}

	result := make(map[string]string)
	if output == "" {
		return result, nil
	}

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

		// Extract branch name from refs/branch-metadata/<branch>
		const prefix = "refs/branch-metadata/"
		if strings.HasPrefix(refName, prefix) {
			branchName := refName[len(prefix):]
			result[branchName] = sha
		}
	}

	return result, nil
}

// DeleteMetadataRef deletes a metadata ref for a branch
func DeleteMetadataRef(branchName string) error {
	refName := fmt.Sprintf("refs/branch-metadata/%s", branchName)
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
	refName := fmt.Sprintf("refs/branch-metadata/%s", branchName)
	_, err = RunGitCommand("update-ref", refName, sha)
	if err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	return nil
}
