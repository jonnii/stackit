package demo

import (
	"time"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/utils"
)

// Delay constants for simulating real operations
const (
	delayShort  = 150 * time.Millisecond
	delayMedium = 300 * time.Millisecond
	delayLong   = 500 * time.Millisecond
)

// simulateDelay adds a random delay around the base duration
func simulateDelay(base time.Duration) {
	// Use a fixed jitter for demo to avoid weak random number generator warnings
	// and because true randomness isn't critical for demo simulation
	jitter := time.Duration(base.Nanoseconds()%100) * time.Millisecond
	time.Sleep(base + jitter)
}

func init() {
	// Register the demo engine factory with runtime package
	runtime.DemoEngineFactory = func() engine.Engine {
		eng, _ := NewDemoEngine()
		return eng
	}
}

// NewDemoContext creates a demo engine and context if in demo mode.
// Returns (context, true) if in demo mode, (nil, false) otherwise.
// Deprecated: Use runtime.NewContextAuto instead.
func NewDemoContext() (*runtime.Context, bool) {
	if !utils.IsDemoMode() {
		return nil, false
	}
	eng, _ := NewDemoEngine()
	return runtime.NewContext(eng), true
}

// NewDemoEngine creates a new demo engine using the standard engine implementation
// but with a simulated Git runner.
func NewDemoEngine() (engine.Engine, error) {
	opts := engine.Options{
		RepoRoot: "/demo",
		Trunk:    GetDemoTrunk(),
		Git:      NewDemoGitRunner(),
	}

	return engine.NewEngine(opts)
}
