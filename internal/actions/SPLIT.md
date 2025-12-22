# Split Action

The `split` command allows you to break a single branch into multiple focused branches, which is a key part of the stacked PR workflow.

## Overview

Sometimes a branch grows too large or contains multiple logical changes that should be reviewed separately. The `split` command helps you decompose these changes into a stack of smaller PRs without manually shuffling commits and branches.

## Split Styles

Stackit supports three styles of splitting:

### 1. By Commit (`--by-commit`)
Best for branches with multiple commits where each commit (or group of commits) represents a logical unit.
- **Workflow**: Interactively select which commits should mark the start of a new branch.
- **Result**: A stack of branches, each containing the selected commits.

### 2. By Hunk (`--by-hunk`)
Best for branches with a single large commit or many messy commits that need to be reorganized from scratch.
- **Workflow**: Similar to `git add -p`. You interactively select hunks of changes to include in each new branch.
- **Result**: A stack of new branches, each with a single commit containing the selected hunks.

### 3. By File (`--by-file`)
Best for quickly moving specific files into a separate PR earlier in the stack.
- **Workflow**: Specify filenames or use an interactive selector to pick files to move.
- **Result**: A new parent branch containing the selected files, with the original branch stacked on top (minus those files).

## Usage Examples

```bash
# Split by commit (interactive)
stackit split

# Split by hunk (interactive)
stackit split --hunk

# Split by extracting specific files
stackit split internal/api.go internal/types.go
```

## How it Works

1. **Validation**: Ensures no unstaged changes and that the branch is tracked.
2. **Analysis**: Determines the parent and child relationships.
3. **Execution**: Performs the split based on the selected style.
4. **Restacking**: Automatically restacks any branches that were stacked on top of the original branch.
5. **PR Linking**: If the original branch had a PR, it remains linked to one of the resulting branches.
