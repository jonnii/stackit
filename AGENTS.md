# Stackit

Go-based CLI tool for managing stacked changes in Git repositories.

**Tech stack:** Go 1.22, Cobra, Just (justfile)

## Architecture

- `cmd/stackit`: CLI entry point and command definitions.
- `internal/actions`: High-level business logic for CLI commands (create, submit, sync, etc.).
- `internal/engine`: Core logic for managing stacked branches, metadata state, and branch relationships.
- `internal/git`: Low-level Git operations (executing git commands, reading config, managing refs).
- `internal/utils`: Shared utilities for branch naming, sanitization, and UI helpers.

## Requirements

**All changes must pass tests and lint before committing:**

```bash
just test              # Run all tests
just test-pkg ./pkg    # Run tests for a specific package (e.g. ./internal/git)
just lint              # Run linter
# Or run both:
just check             # Runs fmt, lint, and test
```

## Build

```bash
just build   # Builds ./stackit binary
just deps    # Install dependencies
```

## Commit Messages

Use **Conventional Commits**:

```
<type>[optional scope]: <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`

**Examples:**
- `feat: add branch traversal functionality`
- `fix: resolve merge conflict detection issue`
- `refactor: simplify merge plan logic`

## Implementation Details

### Metadata Handling
Stackit manages branch relationships and PR state using custom Git references and notes. 
- **Branch Metadata**: Stored in `refs/stackit/metadata/` for each branch.
- **PR Information**: Managed through the `Engine` which abstracts the storage of PR titles, bodies, and status.
- **State Management**: The `internal/engine` package is the source of truth for the stack structure. Always use the `Engine` to query or modify branch relationships.


