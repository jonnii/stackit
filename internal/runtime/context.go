// Package runtime provides a context type that holds the engine and logger
// for use throughout the application. This avoids passing multiple parameters.
package runtime

import (
	"context"
	"fmt"
	"os"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/tui"
)

// Context provides access to engine and output for commands
type Context struct {
	Engine       engine.Engine
	Splog        *tui.Splog
	RepoRoot     string
	GitHubClient git.GitHubClient
}

// NewContext creates a new context with the given engine
func NewContext(eng engine.Engine) *Context {
	return &Context{
		Engine: eng,
		Splog:  tui.NewSplog(),
	}
}

// NewContextWithRepoRoot creates a new context with the given engine and repo root
func NewContextWithRepoRoot(eng engine.Engine, repoRoot string) *Context {
	return &Context{
		Engine:   eng,
		Splog:    tui.NewSplog(),
		RepoRoot: repoRoot,
	}
}

// IsDemoMode returns true if STACKIT_DEMO environment variable is set
func IsDemoMode() bool {
	return os.Getenv("STACKIT_DEMO") != ""
}

// DemoEngineFactory is a function that creates a demo engine.
// This is set by the demo package to avoid circular imports.
var DemoEngineFactory func() engine.Engine

// DemoGitHubClientFactory is a function that creates a demo GitHub client.
// This is set by the demo package to avoid circular imports.
var DemoGitHubClientFactory func() git.GitHubClient

// NewContextAuto creates a context automatically based on the environment.
// In demo mode, it creates a demo engine. Otherwise, it creates a real engine
// using the provided repoRoot.
func NewContextAuto(repoRoot string) (*Context, error) {
	if IsDemoMode() && DemoEngineFactory != nil {
		eng := DemoEngineFactory()
		ctx := NewContext(eng)
		if DemoGitHubClientFactory != nil {
			ctx.GitHubClient = DemoGitHubClientFactory()
		}
		return ctx, nil
	}

	// Create real engine
	eng, err := engine.NewEngine(repoRoot)
	if err != nil {
		return nil, err
	}

	ctx := NewContextWithRepoRoot(eng, repoRoot)

	// Try to create real GitHub client (may fail if no token)
	ghClient, err := git.NewRealGitHubClient(context.Background())
	if err == nil {
		ctx.GitHubClient = ghClient
	}

	return ctx, nil
}

// GetContext returns the appropriate context (demo or real) based on the environment.
// This handles git initialization and config checks for real mode.
func GetContext() (*Context, error) {
	// Check for demo mode first
	if IsDemoMode() {
		return NewContextAuto("")
	}

	// Initialize git repository
	if err := git.InitDefaultRepo(); err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	// Get repo root
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo root: %w", err)
	}

	// Check if initialized
	if !config.IsInitialized(repoRoot) {
		return nil, fmt.Errorf("stackit not initialized. Run 'stackit init' first")
	}

	return NewContextAuto(repoRoot)
}
