package git

import (
	"context"
	"strings"
)

// PruneRemote prunes stale remote-tracking branches
func PruneRemote(ctx context.Context, remote string) error {
	_, err := RunGitCommandWithContext(ctx, "remote", "prune", remote)
	if err != nil {
		// Prune is not critical, just log and continue
		return nil
	}
	return nil
}

// GetRemote returns the default remote name (usually "origin")
func GetRemote(ctx context.Context) string {
	// Try to get remote from config
	remote, err := RunGitCommandWithContext(ctx, "config", "--get", "branch.$(git symbolic-ref --short HEAD).remote")
	if err == nil && remote != "" {
		return remote
	}

	// Fallback to origin
	return "origin"
}

// FetchRemoteShas fetches the SHAs of all branches on the remote.
// Returns a map of branch name -> SHA.
// Sample git ls-remote output:
// 7edb7094e4c66892d783c1effdd106df277a860e        refs/heads/main
func FetchRemoteShas(ctx context.Context, remote string) (map[string]string, error) {
	output, err := RunGitCommandWithContext(ctx, "ls-remote", "--heads", remote)
	if err != nil {
		return nil, err
	}

	remoteShas := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Split on whitespace
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		sha := parts[0]
		ref := parts[1]

		// Only process refs/heads/* (branches)
		const prefix = "refs/heads/"
		if !strings.HasPrefix(ref, prefix) {
			continue
		}

		branchName := ref[len(prefix):]
		remoteShas[branchName] = sha
	}

	return remoteShas, nil
}
