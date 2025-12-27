---
description: View current stack state, branch position, and health status
allowed-tools: Bash(stackit:*), Bash(git:*)
---

# Stack Status

Show the current stack state and identify any issues.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log 2>/dev/null || echo 'Stackit not initialized'`

## Instructions

1. Run `stackit log` to display the branch tree
2. Run `stackit info` to show current branch details
3. Run `stackit doctor` to check for issues
4. Based on the output, suggest next actions:
   - If branches need restack: suggest `stackit restack`
   - If branches have no PR: suggest `stackit submit`
   - If stack is healthy: confirm status is good

## Error Handling
- If not in a git repo: inform user
- If stackit not initialized: suggest `stackit init`
