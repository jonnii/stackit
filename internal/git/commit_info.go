package git

import (
	"fmt"
	"strings"
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

	hash, err := resolveRefHash(repo, branchName)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

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

	hash, err := resolveRefHash(repo, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to resolve branch reference: %w", err)
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

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

	hash, err := resolveRefHash(repo, branchName)
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

	// Try refs/remotes/origin/branchName
	hash, err := resolveRefHash(repo, "origin/"+branchName)
	if err != nil {
		return "", fmt.Errorf("failed to get remote branch reference: %w", err)
	}

	return hash.String(), nil
}

// iterateCommits iterates commits from head to base (exclusive of base)
// Returns commits in order from head to base (newest first)
func iterateCommits(repo *Repository, headHash, baseHash plumbing.Hash) ([]*object.Commit, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	var commits []*object.Commit
	visited := make(map[plumbing.Hash]bool)

	// If base is zero, we want all reachable commits
	// If base is non-zero, we want commits reachable from head but NOT from base (base..head)

	// Use BFS to collect all commits
	queue := []plumbing.Hash{headHash}
	for len(queue) > 0 {
		hash := queue[0]
		queue = queue[1:]

		if visited[hash] || (!baseHash.IsZero() && hash == baseHash) {
			continue
		}
		visited[hash] = true

		commit, err := repo.CommitObject(hash)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit %s: %w", hash, err)
		}

		commits = append(commits, commit)

		for _, parentHash := range commit.ParentHashes {
			if !visited[parentHash] && (baseHash.IsZero() || parentHash != baseHash) {
				queue = append(queue, parentHash)
			}
		}
	}

	return commits, nil
}

// resolveRefHash resolves a ref (branch name, SHA, or ref path) to a hash
func resolveRefHash(repo *Repository, ref string) (plumbing.Hash, error) {
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

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

// GetCommitMessages returns all commit messages for a branch (excluding parent)
func GetCommitMessages(branchName string) ([]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	// Get parent branch to determine range
	meta, err := ReadMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	// Resolve branch head
	branchRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		goGitMu.Unlock()
		return nil, fmt.Errorf("failed to get branch reference: %w", err)
	}
	headHash := branchRef.Hash()
	goGitMu.Unlock()

	// Resolve base (parent revision or zero)
	var baseHash plumbing.Hash
	if meta.ParentBranchRevision != nil {
		baseHash, err = resolveRefHash(repo, *meta.ParentBranchRevision)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve parent revision: %w", err)
		}
	}

	commits, err := iterateCommits(repo, headHash, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	var messages []string
	for _, commit := range commits {
		message := strings.TrimSpace(commit.Message)
		if message != "" {
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// GetCommitSubject returns the subject (first line) of the oldest commit on a branch
func GetCommitSubject(branchName string) (string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "", err
	}

	// Get parent branch to determine range
	meta, err := ReadMetadataRef(branchName)
	if err != nil {
		return "", err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	// Resolve branch head
	branchRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	if err != nil {
		goGitMu.Unlock()
		return "", fmt.Errorf("failed to get branch reference: %w", err)
	}
	headHash := branchRef.Hash()
	goGitMu.Unlock()

	// Resolve base (parent revision or zero)
	var baseHash plumbing.Hash
	if meta.ParentBranchRevision != nil {
		baseHash, err = resolveRefHash(repo, *meta.ParentBranchRevision)
		if err != nil {
			return "", fmt.Errorf("failed to resolve parent revision: %w", err)
		}
	}

	commits, err := iterateCommits(repo, headHash, baseHash)
	if err != nil {
		return "", fmt.Errorf("failed to iterate commits: %w", err)
	}

	if len(commits) == 0 {
		return "", nil
	}

	// Get the oldest commit (last in the list, or first if we walked backwards)
	// Since we walk from head to base, the oldest is the last one
	oldestCommit := commits[len(commits)-1]
	message := strings.TrimSpace(oldestCommit.Message)
	if message == "" {
		return "", nil
	}

	// Get first line (subject)
	lines := strings.Split(message, "\n")
	return strings.TrimSpace(lines[0]), nil
}

// GetCommitRangeSHAs returns the commit SHAs between two revisions (base..head)
func GetCommitRangeSHAs(base, head string) ([]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	headHash, err := resolveRefHash(repo, head)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve head: %w", err)
	}

	baseHash, err := resolveRefHash(repo, base)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base: %w", err)
	}

	commits, err := iterateCommits(repo, headHash, baseHash)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	shas := make([]string, 0, len(commits))
	for _, commit := range commits {
		shas = append(shas, commit.Hash.String())
	}

	return shas, nil
}

// GetCommitHistorySHAs returns the commit SHAs for a branch
func GetCommitHistorySHAs(branchName string) ([]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	branchRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+branchName), true)
	goGitMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get branch reference: %w", err)
	}

	// Get all commits (base is zero hash)
	commits, err := iterateCommits(repo, branchRef.Hash(), plumbing.ZeroHash)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	shas := make([]string, 0, len(commits))
	for _, commit := range commits {
		shas = append(shas, commit.Hash.String())
	}

	return shas, nil
}

// BatchGetRevisions gets SHA revisions for multiple branches in parallel
// Returns a map of branchName -> SHA, with errors collected separately
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
