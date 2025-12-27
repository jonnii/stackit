package git

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetCommitDate returns the commit date for a branch
func GetCommitDate(branchName string) (time.Time, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return time.Time{}, err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	hash, err := resolveRefHashInternal(repo, branchName)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	commit, err := repo.CommitObject(hash)
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

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	hash, err := resolveRefHashInternal(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	commit, err := repo.CommitObject(hash)
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

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	hash, err := resolveRefHashInternal(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	return hash.String(), nil
}

// GetRemoteRevision returns the SHA of a remote branch (e.g., origin/branchName)
func GetRemoteRevision(branchName string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	// Try refs/remotes/origin/branchName
	hash, err := resolveRefHashInternal(repo, "origin/"+branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get remote branch reference: %w", err)
	}

	return hash.String(), nil
}

// iterateCommitsNoLock iterates commits from head to base without locking
func iterateCommitsNoLock(repo *Repository, headHash, baseHash plumbing.Hash) ([]*object.Commit, error) {
	var commits []*object.Commit
	currentHash := headHash

	for !currentHash.IsZero() && currentHash != baseHash {
		commit, err := repo.CommitObject(currentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %s: %w", currentHash, err)
		}

		commits = append(commits, commit)

		if commit.NumParents() == 0 {
			break
		}
		// Follow the first parent for a linear history walk
		currentHash = commit.ParentHashes[0]
	}

	return commits, nil
}

// resolveRefHash resolves a ref (branch name, SHA, or ref path) to a hash
func resolveRefHash(repo *Repository, ref string) (plumbing.Hash, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	return resolveRefHashInternal(repo, ref)
}

// resolveRefHashInternal resolves a ref without locking
func resolveRefHashInternal(repo *Repository, ref string) (plumbing.Hash, error) {
	// 1. Try as a full reference name
	if r, err := repo.Reference(plumbing.ReferenceName(ref), true); err == nil {
		return r.Hash(), nil
	}

	// 2. Try as a local branch
	if r, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+ref), true); err == nil {
		return r.Hash(), nil
	}

	// 3. Try as a remote branch
	if r, err := repo.Reference(plumbing.ReferenceName("refs/remotes/origin/"+ref), true); err == nil {
		return r.Hash(), nil
	}

	// 4. Try as a tag
	if r, err := repo.Reference(plumbing.ReferenceName("refs/tags/"+ref), true); err == nil {
		return r.Hash(), nil
	}

	// 5. Try ResolveRevision (handles SHAs, short SHAs, and complex expressions like HEAD~1)
	hash, err := repo.ResolveRevision(plumbing.Revision(ref))
	if err == nil {
		return *hash, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("failed to resolve ref %s: reference not found", ref)
}

// BatchGetRevisions returns the SHAs for multiple branches in parallel.
func BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	results := make(map[string]string)
	var errors []error
	resultsMu := sync.Mutex{}
	errorsMu := sync.Mutex{}
	var wg sync.WaitGroup

	for _, branchName := range branchNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			sha, err := GetRevision(name)
			resultsMu.Lock()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("failed to get revision for %s: %w", name, err))
				errorsMu.Unlock()
			} else {
				results[name] = sha
			}
			resultsMu.Unlock()
		}(branchName)
	}

	wg.Wait()
	return results, errors
}
