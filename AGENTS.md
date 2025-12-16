# AGENTS.md - Stackit Project Guide

## Project Overview

Stackit is a Go-based CLI tool for managing stacked changes in Git repositories. The project uses:
-
 **Go 1.24.0** (see `go.mod`)
- **Cobra** for CLI command structure
- **Just** (justfile) for task automation
- Standard Go testing with custom test helpers

## Build Instructions

### Building the Binary

The project uses a `justfile` for build automation. The main entry point is `cmd/stackit/main.go`.

**Using Just (Recommended):**

```bash
just build

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

You should run tests after each set of changes to validate your change.

**Run all tests (default):**
```bash
just test
```


