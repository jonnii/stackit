package git

// PruneRemote prunes stale remote-tracking branches
func PruneRemote(remote string) error {
	_, err := RunGitCommand("remote", "prune", remote)
	if err != nil {
		// Prune is not critical, just log and continue
		return nil
	}
	return nil
}

// GetRemote returns the default remote name (usually "origin")
func GetRemote() string {
	// Try to get remote from config
	remote, err := RunGitCommand("config", "--get", "branch.$(git symbolic-ref --short HEAD).remote")
	if err == nil && remote != "" {
		return remote
	}

	// Fallback to origin
	return "origin"
}
