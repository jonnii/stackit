// Package runtime provides a context type that holds the engine and logger
// for use throughout the application. This avoids passing multiple parameters.
package runtime

import (
	"os"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
)

// Context provides access to engine and output for commands
type Context struct {
	Engine engine.Engine
	Splog  *output.Splog
}

// NewContext creates a new context with the given engine
func NewContext(eng engine.Engine) *Context {
	return &Context{
		Engine: eng,
		Splog:  output.NewSplog(),
	}
}

// IsDemoMode returns true if STACKIT_DEMO environment variable is set
func IsDemoMode() bool {
	return os.Getenv("STACKIT_DEMO") != ""
}

// DemoEngineFactory is a function that creates a demo engine.
// This is set by the demo package to avoid circular imports.
var DemoEngineFactory func() engine.Engine

// NewContextAuto creates a context automatically based on the environment.
// In demo mode, it creates a demo engine. Otherwise, it creates a real engine
// using the provided repoRoot.
// Returns (context, repoRoot, error). In demo mode, repoRoot will be empty.
func NewContextAuto(repoRoot string) (*Context, string, error) {
	if IsDemoMode() && DemoEngineFactory != nil {
		eng := DemoEngineFactory()
		return NewContext(eng), "", nil
	}

	// Create real engine
	eng, err := engine.NewEngine(repoRoot)
	if err != nil {
		return nil, "", err
	}

	return NewContext(eng), repoRoot, nil
}
