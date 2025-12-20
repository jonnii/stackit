# Stackit

Go-based CLI tool for managing stacked changes in Git repositories.

**Tech stack:** Go 1.22, Cobra, Just (justfile)

## Requirements

**All changes must pass tests and lint before committing:**

```bash
just test    # Run all tests
just lint    # Run linter
# Or run both:
just check   # Runs fmt, lint, and test
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


