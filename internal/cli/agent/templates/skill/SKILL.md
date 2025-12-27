---
name: stackit
description: Manage stacked Git branches with Stackit. Use this skill when creating branches, submitting PRs, navigating stacks, or troubleshooting stack issues.
allowed-tools: Bash(stackit:*), Bash(git:*), Read, Grep, Glob
version: 1.0.0
---

# Stackit - Stacked Branch Management

You are an expert at using Stackit to manage stacked Git branches. Stackit helps developers break large features into small, focused PRs that stack on top of each other.

## Before Any Operation

Always run `stackit log` first to understand:
- Current branch position in the stack
- Parent/child relationships
- Which branches need attention

## Core Workflows

### Creating a New Branch
1. Stage changes: `git add <files>` or use `--all` flag
2. Create branch: `stackit create [name] -m "commit message"`
3. If no name provided, generate one from the commit message

### Submitting PRs
1. Check stack state: `stackit log`
2. Submit current + ancestors: `stackit submit`
3. Submit entire stack: `stackit submit --stack`
4. For drafts: `stackit submit --draft`

### Syncing with Main
1. Run `stackit sync` to pull trunk and cleanup merged branches
2. If branches were deleted, run `stackit restack`

### Fixing Issues
1. Run `stackit doctor` to diagnose problems
2. For rebase conflicts: resolve files, then `stackit continue`
3. To abort: `stackit abort`
4. To undo: `stackit undo`

## Auto-Generation Guidelines

### Branch Names
When user doesn't provide a name, generate from:
- Commit message (kebab-case, max 50 chars)
- Example: "Add user authentication" -> "add-user-authentication"

### PR Descriptions
Generate from commit messages:
- Title: First commit message line
- Body: Bullet points from all commits in branch
- Include "## Test Plan" section

## Important Rules

1. **Never use raw git for branch operations** - always use stackit commands
2. **Check state before destructive operations** - run `stackit log` first
3. **Handle conflicts gracefully** - guide user through resolution
4. **Keep PRs small and focused** - suggest splitting if too large
