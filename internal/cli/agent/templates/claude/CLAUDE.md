# Stackit Integration

This repository uses [stackit](https://github.com/jonnii/stackit) for managing stacked branches.

## Overview

Stackit is a CLI tool for managing stacked changes in Git repositories. When working with this codebase, you should use stackit commands to create, manage, and submit branches.

## Key Commands

### Creating Branches

- `stackit create [name]` - Create a new branch stacked on top of the current branch
  - If no name is provided, branch name is generated from commit message
  - Use `-m "message"` to specify a commit message
  - Use `--all` to stage all changes before creating
  - Use `--insert` to insert a branch between current and its child

### Managing Stacks

- `stackit log` - Display the branch tree visualization
- `stackit up` - Move up one branch in the stack
- `stackit down` - Move down one branch in the stack
- `stackit top` - Move to the top of the stack
- `stackit bottom` - Move to the bottom of the stack

### Submitting PRs

- `stackit submit` - Submit the current branch and its ancestors as PRs
- `stackit submit --stack` - Submit the entire stack (including descendants)
- Use `--draft` to create draft PRs
- Use `--edit` to interactively edit PR metadata

### Other Useful Commands

- `stackit sync` - Sync branches with remote
- `stackit restack` - Restack branches to fix conflicts
- `stackit fold` - Fold a branch into its parent
- `stackit split` - Split a branch into multiple branches

## Workflow Guidelines

1. **When creating a new feature**: Use `stackit create` to create a new branch on top of the current branch
2. **When making changes**: Commit normally, stackit tracks the relationships
3. **When ready to submit**: Use `stackit submit` to create/update PRs for the stack
4. **When conflicts occur**: Use `stackit restack` to resolve and restack branches

## Important Notes

- Always use stackit commands instead of raw git commands for branch management
- Stackit maintains parent-child relationships between branches
- PRs are automatically created/updated with proper base branches
- The stack structure is preserved when submitting multiple PRs

For more information, see: https://github.com/jonnii/stackit
