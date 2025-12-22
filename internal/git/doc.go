// Package git provides low-level Git operations.
//
// It wraps git command execution and provides a Go-friendly interface for:
//   - Branch management (create, delete, checkout, rename)
//   - Commit operations (commit, amend, cherry-pick)
//   - Repo state queries (status, diff, log, refs)
//   - Remote operations (push, fetch, pull)
//
// This package should be the only place where direct git commands are executed.
package git
