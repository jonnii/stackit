---
description: Sync with trunk, cleanup merged branches, and restack
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Sync

Synchronize stack with remote: pull trunk, cleanup merged branches, restack.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log 2>/dev/null`

## Instructions

1. Show current stack state with `stackit log`

2. Run `stackit sync` to:
   - Pull latest trunk changes
   - Delete branches whose PRs were merged
   - Delete branches whose PRs were closed

3. Report what was cleaned up:
   - Which branches were deleted
   - Which branches were reparented

4. If branches remain, run `stackit restack` to rebase onto updated trunk

5. Show final stack state with `stackit log`

## Error Handling
- If rebase conflicts occur: help resolve, then `stackit continue`
- If network error: retry or inform user
- If branches can't be deleted: show why (local changes, etc.)
