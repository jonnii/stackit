---
description: Submit branches as PRs with auto-generated descriptions
allowed-tools: Bash(stackit:*), Bash(git:*), Read
argument-hint: [--stack | --draft]
---

# Stack Submit

Submit branches to GitHub and create/update PRs.

## Context
- Current branch: !`git branch --show-current`
- Stack state: !`stackit log 2>/dev/null`

## Arguments
$ARGUMENTS

## Instructions

1. Run `stackit log` to see branches to submit

2. For each branch that will be submitted, gather commit info:
   `git log --oneline <parent>..<branch>`

3. If PRs don't exist yet, prepare PR metadata:
   - **Title**: First commit message line (without conventional commit prefix for cleaner titles)
   - **Body**: Generate from commits:
     ```
     ## Summary
     - <bullet point for each commit>

     ## Test Plan
     - [ ] <suggested test steps based on changes>
     ```

4. Run submit command:
   - Current branch only: `stackit submit`
   - Entire stack: `stackit submit --stack`
   - As drafts: `stackit submit --draft`

5. Report created/updated PR URLs

## Error Handling
- If branch needs restack: run `stackit restack` first
- If push fails: check for upstream changes
- If PR creation fails: show GitHub error
