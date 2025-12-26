package git

import (
	"errors"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// PruneRemote prunes stale remote-tracking branches
func PruneRemote(remote string) error {
	repo, err := GetDefaultRepo()
	if err != nil {
		return err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	r, err := repo.Remote(remote)
	if err != nil {
		goGitMu.Unlock()
		return err
	}

	// Fetch with pruning
	err = r.Fetch(&gogit.FetchOptions{
		Prune: true,
	})
	goGitMu.Unlock()
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		// Prune is not critical, just log and continue
		return nil //nolint:nilerr
	}
	return nil
}

// GetRemote returns the default remote name (usually "origin")
func GetRemote() string {
	repo, err := GetDefaultRepo()
	if err != nil {
		return "origin"
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	defer goGitMu.Unlock()

	// Try to get current branch
	head, err := repo.Head()
	if err == nil && head.Name().IsBranch() {
		branchName := head.Name().Short()
		// Try to get remote for the current branch
		cfg, err := repo.Config()
		if err == nil {
			if b, ok := cfg.Branches[branchName]; ok && b.Remote != "" {
				return b.Remote
			}
		}
	}

	// Fallback to origin
	return "origin"
}

// FetchRemoteShas fetches the SHAs of all branches on the remote.
// Returns a map of branch name -> SHA.
func FetchRemoteShas(remote string) (map[string]string, error) {
	repo, err := GetDefaultRepo()
	if err != nil {
		return nil, err
	}

	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	r, err := repo.Remote(remote)
	goGitMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote %s: %w", remote, err)
	}

	// List remote references
	// Synchronize go-git operations to prevent concurrent packfile access
	goGitMu.Lock()
	refs, err := r.List(&gogit.ListOptions{})
	goGitMu.Unlock()
	if err != nil {
		if errors.Is(err, transport.ErrEmptyRemoteRepository) || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to list remote refs for %s: %w", remote, err)
	}

	remoteShas := make(map[string]string)
	for _, ref := range refs {
		// Only process refs/heads/* (branches)
		if ref.Name().IsBranch() {
			branchName := ref.Name().Short()
			remoteShas[branchName] = ref.Hash().String()
		}
	}

	return remoteShas, nil
}
