---
description: Diagnose and fix common stack issues
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Fix

Diagnose and automatically fix common stack problems.

## Context
- Current branch: !`git branch --show-current`
- Git status: !`git status --short 2>/dev/null`

## Instructions

1. Run `stackit doctor` to check overall health

2. Run `stackit log` to identify structural issues

3. Check for and fix these common issues:

   **Rebase in progress:**
   - Check `git status` for "rebase in progress"
   - If conflicts: help user resolve, then `stackit continue`
   - If user wants to abort: `stackit abort`

   **Branches need restack:**
   - Run `stackit restack` to rebase branches
   - Handle conflicts if they occur

   **Orphaned branches (parent was merged):**
   - Run `stackit sync` to reparent

   **PR base mismatch:**
   - Run `stackit submit` to update PR bases

   **Uncommitted changes blocking operation:**
   - Suggest stashing or committing

4. After fixes, show updated stack state with `stackit log`

## Error Handling
- For unrecoverable issues: suggest `stackit undo`
- If unsure: ask user before destructive operations
