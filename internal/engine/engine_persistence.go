package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

const (
	// MetadataRefPrefix is the prefix for Git refs where branch metadata is stored
	MetadataRefPrefix = "refs/stackit/metadata/"
)

// ReadMetadataRef reads metadata for a branch from Git refs
func (e *engineImpl) ReadMetadataRef(branchName string) (*Meta, error) {
	return e.readMetadataRef(branchName)
}

// readMetadataRef reads metadata for a branch from Git refs
func (e *engineImpl) readMetadataRef(branchName string) (*Meta, error) {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)

	sha, err := e.git.GetRef(refName)
	if err != nil {
		// If ref doesn't exist, it's not an error, just means no metadata
		return &Meta{}, nil //nolint:nilerr
	}

	content, err := e.git.ReadBlob(sha)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata blob %s: %w", sha, err)
	}

	if content == "" {
		return &Meta{}, nil
	}

	var meta Meta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata for %s: %w", branchName, err)
	}

	return &meta, nil
}

// WriteMetadataRef writes metadata for a branch to Git refs
func (e *engineImpl) WriteMetadataRef(branch Branch, meta *Meta) error {
	return e.writeMetadataRef(branch.GetName(), meta)
}

// writeMetadataRef writes metadata for a branch to Git refs
func (e *engineImpl) writeMetadataRef(branchName string, meta *Meta) error {
	jsonData, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sha, err := e.git.CreateBlob(string(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metadata blob: %w", err)
	}

	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branchName)
	if err := e.git.UpdateRef(refName, sha); err != nil {
		return fmt.Errorf("failed to write metadata ref: %w", err)
	}

	return nil
}

// DeleteMetadataRef deletes a metadata ref for a branch
func (e *engineImpl) DeleteMetadataRef(branch Branch) error {
	refName := fmt.Sprintf("%s%s", MetadataRefPrefix, branch.GetName())
	return e.git.DeleteRef(refName)
}

// RenameMetadataRef renames a metadata ref from one branch name to another
func (e *engineImpl) RenameMetadataRef(oldBranch, newBranch Branch) error {
	oldRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, oldBranch.GetName())
	newRefName := fmt.Sprintf("%s%s", MetadataRefPrefix, newBranch.GetName())

	sha, err := e.git.GetRef(oldRefName)
	if err != nil {
		return nil //nolint:nilerr // Nothing to rename
	}

	if err := e.git.UpdateRef(newRefName, sha); err != nil {
		return fmt.Errorf("failed to create new metadata ref: %w", err)
	}

	if err := e.git.DeleteRef(oldRefName); err != nil {
		return fmt.Errorf("failed to delete old metadata ref: %w", err)
	}

	return nil
}

// BatchReadMetadataRefs reads metadata for multiple branches in parallel
func (e *engineImpl) BatchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error) {
	return e.batchReadMetadataRefs(branchNames)
}

// batchReadMetadataRefs reads metadata for multiple branches in parallel
func (e *engineImpl) batchReadMetadataRefs(branchNames []string) (map[string]*Meta, map[string]error) {
	results := make(map[string]*Meta)
	errs := make(map[string]error)
	resultsMu := sync.Mutex{}
	errsMu := sync.Mutex{}
	var wg sync.WaitGroup

	for _, branchName := range branchNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			meta, err := e.readMetadataRef(name)
			if err != nil {
				errsMu.Lock()
				errs[name] = err
				errsMu.Unlock()
				return
			}
			resultsMu.Lock()
			results[name] = meta
			resultsMu.Unlock()
		}(branchName)
	}

	wg.Wait()
	return results, errs
}

// ListMetadataRefs returns all metadata refs
func (e *engineImpl) ListMetadataRefs() (map[string]string, error) {
	refs, err := e.git.ListRefs(MetadataRefPrefix)
	if err != nil {
		return nil, err
	}

	// Remove prefix from branch names
	result := make(map[string]string)
	for refName, sha := range refs {
		branchName := strings.TrimPrefix(refName, MetadataRefPrefix)
		result[branchName] = sha
	}
	return result, nil
}
