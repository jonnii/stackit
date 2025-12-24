// Package doctor provides diagnostic functionality for checking stackit environment and repository health.
package doctor

import (
	"fmt"

	"stackit.dev/stackit/internal/runtime"
)

// Options contains options for the doctor command
type Options struct {
	Fix   bool
	Trunk string // Trunk branch name from config
}

// Action runs diagnostic checks on the stackit environment and repository
func Action(ctx *runtime.Context, opts Options) error {
	splog := ctx.Splog
	eng := ctx.Engine

	if opts.Fix {
		splog.Info("Running stackit doctor with --fix...")
	} else {
		splog.Info("Running stackit doctor...")
	}
	splog.Newline()

	var warnings []string
	var errors []string

	// Environment checks
	splog.Info("Environment:")
	warnings, errors = checkEnvironment(splog, warnings, errors)

	splog.Newline()

	// Repository checks
	splog.Info("Repository:")
	warnings, errors = checkRepository(ctx, splog, warnings, errors, opts.Trunk)

	splog.Newline()

	// Stack state checks
	splog.Info("Stack State:")
	warnings, errors = checkStackState(eng, splog, warnings, errors, opts.Fix)

	// Summary
	splog.Newline()
	switch {
	case len(errors) > 0:
		splog.Warn("Doctor found %d error(s) and %d warning(s).", len(errors), len(warnings))
		for _, err := range errors {
			splog.Error("  %s", err)
		}
		for _, warn := range warnings {
			splog.Warn("  %s", warn)
		}
		return fmt.Errorf("doctor found %d error(s)", len(errors))
	case len(warnings) > 0:
		if opts.Fix {
			splog.Info("Doctor found %d warning(s), some of which may have been fixed.", len(warnings))
		} else {
			splog.Info("Doctor found %d warning(s). Your stackit setup is mostly healthy.", len(warnings))
		}
		for _, warn := range warnings {
			splog.Warn("  %s", warn)
		}
	default:
		splog.Info("✅ All checks passed. Your stackit setup is healthy.")
	}

	return nil
}
