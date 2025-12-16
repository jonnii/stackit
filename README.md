# Stackit

**Stackit** is a command-line tool that makes working with stacked changes fast and intuitive.

Stacked changes (or stacked diffs) are a workflow where you break large features into small, reviewable branches that build on top of each other. Stackit manages the complexity of keeping these branches in sync, rebased, and submitted as pull requests.

## Why Stacked Changes?

- **Faster reviews** — Small, focused PRs are easier to review
- **Ship incrementally** — Merge and deploy pieces of a feature as they're approved
- **Cleaner history** — Each PR tells a coherent story
- **Parallel work** — Keep working while waiting for review

## Features

- 🌳 **Visual branch tree** — See your entire stack at a glance with `stackit log`
- 🔄 **Automatic restacking** — Keep all branches up to date when you rebase
- 📤 **Submit entire stacks** — Push all branches and create/update PRs in one command
- 🔀 **Smart merging** — Merge stacks bottom-up or squash top-down
- 🔧 **Absorb changes** — Automatically amend changes to the right commit in your stack
- 🧭 **Easy navigation** — Move up, down, top, or bottom of your stack
- 🧹 **Auto cleanup** — Detect and delete merged branches during sync

## Installation

### From Source

Requires Go 1.25+:

```bash
git clone https://github.com/jonnii/stackit
cd stackit/stackit
go build -o stackit ./cmd/stackit

# Move to your PATH
mv stackit /usr/local/bin/
```

### Using Just

If you have [just](https://github.com/casey/just) installed:

```bash
just build
just install
```

## Quick Start

### 1. Initialize in your repository

```bash
cd your-repo
stackit init
```

This detects your trunk branch (usually `main`) and sets up Stackit.

### 2. Create your first stacked branch

```bash
# Stage some changes, then:
stackit create my-feature -m "Add new feature"
```

### 3. Create another branch on top

```bash
# Make more changes
stackit create my-feature-part-2 -m "Continue feature work"
```

### 4. View your stack

```bash
stackit log
```

```
main
│
├─◯ my-feature
│ │
│ └─● my-feature-part-2 ← you are here
```

### 5. Submit your stack

```bash
stackit submit
```

This pushes all branches and creates/updates PRs on GitHub.

## Commands

### Stack Navigation

| Command | Description |
|---------|-------------|
| `stackit log` | Display branch tree with PR info and sync status |
| `stackit up` | Move to the child branch |
| `stackit down` | Move to the parent branch |
| `stackit top` | Move to the top of the current stack |
| `stackit bottom` | Move to the bottom of the current stack |
| `stackit trunk` | Move to the trunk branch |
| `stackit checkout` | Interactive branch switcher |

### Branch Management

| Command | Description |
|---------|-------------|
| `stackit create [name]` | Create a new branch on top of current |
| `stackit delete` | Delete the current branch |
| `stackit fold` | Merge the current branch into its parent |
| `stackit split` | Split the current branch's commits |
| `stackit squash` | Squash all commits on the current branch |
| `stackit modify` | Amend the current commit |
| `stackit absorb` | Intelligently amend changes to the right commits |

### Stack Operations

| Command | Description |
|---------|-------------|
| `stackit restack` | Rebase branches to ensure proper ancestry |
| `stackit submit` | Push branches and create/update PRs |
| `stackit sync` | Pull trunk, clean merged branches, restack |
| `stackit merge` | Merge PRs in the stack via GitHub |

### Utilities

| Command | Description |
|---------|-------------|
| `stackit info` | Show info about current branch |
| `stackit parent` | Print the parent branch name |
| `stackit children` | Print child branch names |
| `stackit continue` | Continue after resolving conflicts |
| `stackit abort` | Abort the current operation |

## Common Workflows

### Starting a new feature

```bash
stackit trunk                    # Start from main
stackit create setup -m "Setup infrastructure"
# ... make more changes ...
stackit create feature -m "Implement feature"
# ... make more changes ...
stackit create tests -m "Add tests"
```

### Updating after code review

```bash
# On any branch in your stack, make changes then:
stackit modify           # Amend the current commit
stackit restack          # Update all child branches
stackit submit           # Push updates
```

### Smart change absorption

```bash
# Stage changes that should go to different commits
git add -p

# Stackit figures out which commits to amend
stackit absorb
```

### Syncing with main

```bash
stackit sync
```

This will:
1. Pull the latest trunk
2. Prompt to delete branches for merged PRs
3. Restack any branches that need updating

### Merging your stack

```bash
stackit merge
```

Interactive wizard helps you choose:
- **Bottom-up**: Merge each PR individually (preserves history)
- **Top-down**: Squash everything into one PR

## Command Options

### `stackit create`

```
-m, --message    Commit message
-a, --all        Stage all changes (including untracked)
-u, --update     Stage only tracked file changes  
-p, --patch      Interactively select hunks to stage
-i, --insert     Insert between current branch and its children
```

### `stackit submit`

```
-s, --stack      Include descendant branches
-d, --draft      Create PRs as drafts
-c, --confirm    Preview and confirm before submitting
--dry-run        Show what would happen without doing it
--restack        Restack before submitting
```

### `stackit log`

```
-s, --stack      Only show current branch's stack
-r, --reverse    Display bottom-to-top
-n, --steps N    Limit to N levels up/down
-u, --show-untracked  Include untracked branches
```

### `stackit restack`

```
--only           Restack only current branch
--upstack        Restack current and descendants
--downstack      Restack current and ancestors
```

## Conflict Resolution

When a rebase has conflicts:

```bash
# Resolve conflicts in your editor
git add <resolved-files>

# Continue the operation
stackit continue

# Or abort and return to previous state
stackit abort
```

## Requirements

- Git 2.0+
- GitHub CLI (`gh`) for PR operations
- Go 1.25+ (for building from source)

## Development

```bash
# Run tests
just test

# Run tests with coverage
just test-coverage

# Format code
just fmt

# Run linter
just lint

# Run all checks
just check
```

## Philosophy

Stackit is designed around these principles:

1. **Non-destructive** — Operations are safe by default, with confirmations for dangerous actions
2. **Fast** — Common operations should be instant
3. **Intuitive** — Commands do what you expect
4. **Git-native** — Uses standard Git under the hood, no magic

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.
