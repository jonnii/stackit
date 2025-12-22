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
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/tui"
)

// Context provides access to engine and output for commands
type Context struct {
	context.Context
	Engine       engine.Engine
	Splog        *tui.Splog
	RepoRoot     string
	GitHubClient github.Client
}

// NewContext creates a new context with the given engine
func NewContext(eng engine.Engine) *Context {
	return &Context{
		Context: context.Background(),
		Engine:  eng,
		Splog:   tui.NewSplog(),
	}
}

// NewContextWithRepoRoot creates a new context with the given engine and repo root
func NewContextWithRepoRoot(eng engine.Engine, repoRoot string) *Context {
	return &Context{
		Context:  context.Background(),
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
var DemoGitHubClientFactory func() github.Client

// NewContextAuto creates a context automatically based on the environment.
// In demo mode, it creates a demo engine. Otherwise, it creates a real engine
// using the provided repoRoot.
func NewContextAuto(ctx context.Context, repoRoot string) (*Context, error) {
	if IsDemoMode() && DemoEngineFactory != nil {
		eng := DemoEngineFactory()
		runtimeCtx := NewContext(eng)
		runtimeCtx.Context = ctx
		if DemoGitHubClientFactory != nil {
			runtimeCtx.GitHubClient = DemoGitHubClientFactory()
		}
		return runtimeCtx, nil
	}

	// Read config and create engine options
	trunk, err := config.GetTrunk(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get trunk: %w", err)
	}
	maxUndoDepth, err := config.GetUndoStackDepth(repoRoot)
	if err != nil {
		maxUndoDepth = engine.DefaultMaxUndoStackDepth
	}

	// Create real engine
	eng, err := engine.NewEngine(engine.Options{
		RepoRoot:          repoRoot,
		Trunk:             trunk,
		MaxUndoStackDepth: maxUndoDepth,
	})
	if err != nil {
		return nil, err
	}

	runtimeCtx := NewContextWithRepoRoot(eng, repoRoot)
	runtimeCtx.Context = ctx

	// Try to create real GitHub client (may fail if no token)
	ghClient, err := github.NewRealGitHubClient(ctx)
	if err == nil {
		runtimeCtx.GitHubClient = ghClient
	}

	return runtimeCtx, nil
}

// GetContext returns the appropriate context (demo or real) based on the environment.
// This handles git initialization and config checks for real mode.
func GetContext(ctx context.Context) (*Context, error) {
	// Check for demo mode first
	if IsDemoMode() {
		return NewContextAuto(ctx, "")
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

	return NewContextAuto(ctx, repoRoot)
}
