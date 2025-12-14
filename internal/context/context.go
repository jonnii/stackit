package context

import (
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/output"
)

// Context provides access to engine and output for commands
type Context struct {
	Engine engine.Engine
	Splog  *output.Splog
}

// NewContext creates a new context
func NewContext(eng engine.Engine) *Context {
	return &Context{
		Engine: eng,
		Splog:  output.NewSplog(),
	}
}
