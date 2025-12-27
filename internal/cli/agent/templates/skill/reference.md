# Stackit Command Reference

## Navigation Commands

| Command | Description |
|---------|-------------|
| `stackit log` | Display the branch tree visualization |
| `stackit log full` | Show tree with GitHub PR status and CI checks |
| `stackit checkout` | Interactive branch switcher |
| `stackit up` | Move to the child branch |
| `stackit down` | Move to the parent branch |
| `stackit top` | Move to the top of the stack |
| `stackit bottom` | Move to the bottom of the stack |
| `stackit trunk` | Return to the main/trunk branch |
| `stackit children` | Show children of current branch |
| `stackit parent` | Show parent of current branch |

## Branch Management

| Command | Description |
|---------|-------------|
| `stackit create [name]` | Create a new stacked branch |
| `stackit modify` | Amend current commit (like git commit --amend) |
| `stackit absorb` | Auto-amend changes to correct commits in stack |
| `stackit split` | Split current branch into multiple branches |
| `stackit squash` | Squash all commits on current branch |
| `stackit fold` | Merge current branch into its parent |
| `stackit pop` | Delete branch but keep changes in working tree |
| `stackit delete` | Delete current branch and metadata |
| `stackit rename [name]` | Rename current branch |
| `stackit scope [name]` | Manage logical scope (Jira/Linear ID) |

## Stack Operations

| Command | Description |
|---------|-------------|
| `stackit restack` | Rebase all branches to ensure proper ancestry |
| `stackit foreach` | Run command on each branch in stack |
| `stackit submit` | Push branches and create/update PRs |
| `stackit sync` | Pull trunk, delete merged branches, restack |
| `stackit merge` | Merge approved PRs and cleanup |
| `stackit reorder` | Interactively reorder branches |
| `stackit move` | Rebase branch onto new parent |

## Recovery & Utilities

| Command | Description |
|---------|-------------|
| `stackit undo` | Restore repo to state before a command |
| `stackit continue` | Continue interrupted operation |
| `stackit abort` | Abort interrupted operation |
| `stackit doctor` | Diagnose and fix setup issues |
| `stackit info` | Show detailed branch info |
| `stackit track` | Start tracking a branch |
| `stackit untrack` | Stop tracking a branch |
| `stackit debug` | Dump debugging info |

## Common Flag Patterns

### stackit create
- `-m "message"` - Commit message
- `--all` - Stage all changes first
- `--insert` - Insert between current and child

### stackit submit
- `--stack` - Submit entire stack (alias: `ss`)
- `--draft` - Create as draft PRs
- `--edit` - Edit PR metadata interactively

### stackit sync
- `--restack` - Auto-restack after cleanup

## Workflow Examples

### Start a new feature
```bash
git add .
stackit create feature-name -m "feat: add new feature"
```

### Stack another change
```bash
git add .
stackit create next-part -m "feat: extend feature"
```

### Submit for review
```bash
stackit submit --stack
```

### After code review changes
```bash
git add .
stackit modify
stackit restack
stackit submit
```

### Sync with main
```bash
stackit sync --restack
```

## Troubleshooting

### "Branch needs restack"
Run `stackit restack` to rebase branches onto their parents.

### "Rebase conflict"
1. Resolve conflicts in the files
2. `git add <resolved-files>`
3. `stackit continue`

### "Orphaned branch"
Parent was merged. Run `stackit sync` to reparent.

### "PR base mismatch"
Run `stackit submit` to update PR base branches.
