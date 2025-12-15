package git

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/branch-metadata/%s", branchName))
	ref, err := repo.Reference(refName, false)
	if err != nil {
		// Metadata ref doesn't exist - return empty meta
		return &Meta{}, nil
	}

	// Get the object content
	obj, err := repo.Object(plumbing.AnyObject, ref.Hash())
	if err != nil {
		return &Meta{}, nil
	}

	// Read the content
	var content []byte
	switch obj := obj.(type) {
	case *object.Blob:
		reader, err := obj.Reader()
		if err != nil {
			return &Meta{}, nil
		}
		defer reader.Close()

		content, err = io.ReadAll(reader)
		if err != nil {
			return &Meta{}, nil
		}
	default:
		return &Meta{}, nil
	}

	var meta Meta
	if err := json.Unmarshal(content, &meta); err != nil {
		return &Meta{}, nil
	}

	return &meta, nil
}

// GetMetadataRefList returns all metadata refs
func GetMetadataRefList() (map[string]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	result := make(map[string]string)
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsTag() {
			return nil
		}

		name := ref.Name().String()
		if len(name) > 20 && name[:20] == "refs/branch-metadata/" {
			branchName := name[20:]
			result[branchName] = ref.Hash().String()
		}
		return nil
	})

	return result, err
}

// DeleteMetadataRef deletes a metadata ref for a branch
func DeleteMetadataRef(branchName string) error {
	repo, err := GetDefaultRepo()
	if err != nil {
		return err
	}

	refName := plumbing.ReferenceName(fmt.Sprintf("refs/branch-metadata/%s", branchName))
	return repo.Storer.RemoveReference(refName)
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

