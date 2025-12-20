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

## Commit Message Guidelines

This project uses **Conventional Commits** for commit messages. This standardizes commit history and enables automated tooling.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Commit Types

- **feat**: A new feature
- **fix**: A bug fix
- **docs**: Documentation only changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring without changing functionality
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks, dependency updates, build changes
- **ci**: Changes to CI configuration files and scripts

### Examples

```
feat: add branch traversal functionality
fix: resolve merge conflict detection issue
docs: update AGENTS.md with commit guidelines
refactor: simplify merge plan logic
test: add integration tests for sync command
chore: update go.mod dependencies
```


