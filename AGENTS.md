# AGENTS.md - Stackit Project Guide

## Project Overview

Stackit is a Go-based CLI tool for managing stacked changes in Git repositories. The project uses:
- **Go 1.22** (see `go.mod`)
- **Cobra** for CLI command structure
- **Just** (justfile) for task automation
- Standard Go testing with custom test helpers

## Build Instructions

### Building the Binary

The project uses a `justfile` for build automation. The main entry point is `cmd/stackit/main.go`.

**Using Just (Recommended):**

```bash
just build
```

**Install dependencies first:**

```bash
just deps
# or
go mod download
go mod tidy
```

### Build Output

- Binary location: `./stackit` (in project root)
- The binary is a standalone executable for the current platform

## Testing Instructions

⚠️ **IMPORTANT: You MUST run tests after each set of changes to validate your work.**

### Quick Test Command

**If `just` is available:**

```bash
just test
```

**If `just` is NOT available (use this as fallback):**

```bash
STACKIT_TEST_NO_INTERACTIVE=1 go test ./...
```

### Test Environment Variables

- `STACKIT_TEST_NO_INTERACTIVE=1` - Required for all tests to prevent interactive prompts during test execution

### Additional Test Commands

**Run tests without cache (for debugging flaky tests):**

```bash
STACKIT_TEST_NO_INTERACTIVE=1 go test ./... -count=1
```

**Run tests with verbose output:**

```bash
STACKIT_TEST_NO_INTERACTIVE=1 go test -v ./...
```

**Run tests for a specific package:**

```bash
STACKIT_TEST_NO_INTERACTIVE=1 go test -v ./internal/cli
```

**Run tests with race detection:**

```bash
STACKIT_TEST_NO_INTERACTIVE=1 go test -race ./...
```

### Test Success Criteria

- All tests must pass (no failures)
- Exit code must be 0
- No race conditions detected (when running with `-race`)

## Agent Test Running Rules

1. **ALWAYS** run `go mod tidy` before running tests to ensure dependencies are up to date
2. **ALWAYS** set `STACKIT_TEST_NO_INTERACTIVE=1` environment variable when running tests
3. **ALWAYS** use the fallback Go command if `just` is not available
4. **VERIFY** that tests pass after making any code changes
5. If tests fail, analyze the output and fix the issues before considering the task complete




