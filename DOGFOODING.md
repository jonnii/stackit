# Dogfooding Stackit

This guide explains how to use Stackit to manage the Stackit repository itself.

## Quick Start

### 1. Build the Binary

```bash
just build
# or
go build -o stackit ./cmd/stackit
```

This creates a `stackit` binary in the current directory.

### 2. Initialize Stackit

```bash
just init
# or
./stackit init
```

This will:
- Detect your trunk branch (defaults to "main" if it exists)
- Create `.git/.stackit_config` with trunk configuration
- Set up Stackit for the repository

You can specify a different trunk:
```bash
./stackit init --trunk main
```

### 3. Use Stackit Commands

All commands can be run via the justfile:

```bash
# View branch tree
just run log

# View with options
just run "log --reverse"
just run "log --stack"
just run "log --show-untracked"
```

Or directly:
```bash
./stackit log
./stackit log --reverse
./stackit log --stack
```

## Workflow

### Creating Feature Branches

Once initialized, you can use Stackit to manage your branches:

1. **Create a feature branch** (when branch create is implemented):
   ```bash
   ./stackit branch create my-feature
   ```

2. **View your stack**:
   ```bash
   ./stackit log
   ```

3. **Check current branch info**:
   ```bash
   ./stackit branch info
   ```

### Current Available Commands

- `stackit init` - Initialize Stackit in the repository
- `stackit log` - Display branch tree visualization
  - `--reverse` - Reverse the output order
  - `--stack` - Show only current branch's stack
  - `--steps N` - Limit depth to N levels
  - `--show-untracked` - Include untracked branches

## Justfile Recipes

The justfile includes helpful recipes:

- `just build` - Build the stackit binary
- `just install` - Build and install (same as build for now)
- `just run <cmd>` - Build if needed, then run a stackit command
- `just init` - Initialize Stackit in this repo

## Tips

1. **Rebuild after changes**: After modifying the code, rebuild with `just build`
2. **Test in isolation**: Use the testhelpers to test commands without affecting your main repo
3. **Track your work**: As more commands are implemented, use them to manage your feature branches

## Troubleshooting

### "not a git repository" error
Make sure you're in the `stackit/` directory (where `.git` is located).

### "no branches found" error
You need at least one commit in your repository before initializing.

### Config file issues
If the config gets corrupted, you can reinitialize:
```bash
./stackit init --reset
```

## Next Steps

As more commands are ported, you'll be able to:
- Create and manage stacked branches
- Track parent-child relationships
- Submit branches
- Restack branches
- And more!



