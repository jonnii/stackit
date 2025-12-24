package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

// newAgentCmd creates the agent command
func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agent integration files for Cursor and Claude Code",
		Long: `Manage agent integration files that help AI assistants use stackit effectively.

This command generates configuration files that enable AI agents (like Cursor and Claude Code)
to understand how to use stackit commands for managing stacked branches.`,
		SilenceUsage: true,
	}

	cmd.AddCommand(newAgentInitCmd())

	return cmd
}

// newAgentInitCmd creates the agent init command
func newAgentInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize agent integration files in the current repository",
		Long: `Creates agent integration files that help AI assistants use stackit effectively.

This will create:
  - .cursor/rules/stackit.md (for Cursor)
  - CLAUDE.md (for Claude Code)

These files contain instructions for AI agents on how to use stackit commands
to manage stacked branches, create commits, submit PRs, and more.`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Get repo root
			if err := git.InitDefaultRepo(); err != nil {
				return fmt.Errorf("not a git repository: %w", err)
			}

			repoRoot, err := git.GetRepoRoot()
			if err != nil {
				return fmt.Errorf("failed to get repo root: %w", err)
			}

			// Create .cursor/rules directory if it doesn't exist
			cursorRulesDir := filepath.Join(repoRoot, ".cursor", "rules")
			if err := os.MkdirAll(cursorRulesDir, 0750); err != nil {
				return fmt.Errorf("failed to create .cursor/rules directory: %w", err)
			}

			// Write Cursor rules file
			cursorRulesPath := filepath.Join(cursorRulesDir, "stackit.md")
			if err := os.WriteFile(cursorRulesPath, []byte(cursorRulesContent), 0600); err != nil {
				return fmt.Errorf("failed to write Cursor rules file: %w", err)
			}

			// Write CLAUDE.md file
			claudePath := filepath.Join(repoRoot, "CLAUDE.md")
			if err := os.WriteFile(claudePath, []byte(claudeContent), 0600); err != nil {
				return fmt.Errorf("failed to write CLAUDE.md file: %w", err)
			}

			fmt.Println("✓ Created .cursor/rules/stackit.md")
			fmt.Println("✓ Created CLAUDE.md")
			fmt.Println()
			fmt.Println("Agent integration files have been created. AI assistants can now use stackit effectively!")

			return nil
		},
	}

	return cmd
}

const cursorRulesContent = `# Stackit Agent Rules

This repository uses [stackit](https://github.com/jonnii/stackit) for managing stacked branches.

## Overview

Stackit is a CLI tool for managing stacked changes in Git repositories. When working with this codebase, you should use stackit commands to create, manage, and submit branches.

## Key Commands

### Creating Branches

- ` + "`stackit create [name]`" + ` - Create a new branch stacked on top of the current branch
  - If no name is provided, branch name is generated from commit message
  - Use ` + "`-m \"message\"`" + ` to specify a commit message
  - Use ` + "`--all`" + ` to stage all changes before creating
  - Use ` + "`--insert`" + ` to insert a branch between current and its child

### Managing Stacks

- ` + "`stackit log`" + ` - Display the branch tree visualization
- ` + "`stackit up`" + ` - Move up one branch in the stack
- ` + "`stackit down`" + ` - Move down one branch in the stack
- ` + "`stackit top`" + ` - Move to the top of the stack
- ` + "`stackit bottom`" + ` - Move to the bottom of the stack

### Submitting PRs

- ` + "`stackit submit`" + ` - Submit the current branch and its ancestors as PRs
- ` + "`stackit submit --stack`" + ` - Submit the entire stack (including descendants)
- Use ` + "`--draft`" + ` to create draft PRs
- Use ` + "`--edit`" + ` to interactively edit PR metadata

### Other Useful Commands

- ` + "`stackit sync`" + ` - Sync branches with remote
- ` + "`stackit restack`" + ` - Restack branches to fix conflicts
- ` + "`stackit fold`" + ` - Fold a branch into its parent
- ` + "`stackit split`" + ` - Split a branch into multiple branches

## Workflow Guidelines

1. **When creating a new feature**: Use ` + "`stackit create`" + ` to create a new branch on top of the current branch
2. **When making changes**: Commit normally, stackit tracks the relationships
3. **When ready to submit**: Use ` + "`stackit submit`" + ` to create/update PRs for the stack
4. **When conflicts occur**: Use ` + "`stackit restack`" + ` to resolve and restack branches

## Important Notes

- Always use stackit commands instead of raw git commands for branch management
- Stackit maintains parent-child relationships between branches
- PRs are automatically created/updated with proper base branches
- The stack structure is preserved when submitting multiple PRs

For more information, see: https://github.com/jonnii/stackit
`

const claudeContent = `# Stackit Integration

This repository uses [stackit](https://github.com/jonnii/stackit) for managing stacked branches.

## Overview

Stackit is a CLI tool for managing stacked changes in Git repositories. When working with this codebase, you should use stackit commands to create, manage, and submit branches.

## Key Commands

### Creating Branches

- ` + "`stackit create [name]`" + ` - Create a new branch stacked on top of the current branch
  - If no name is provided, branch name is generated from commit message
  - Use ` + "`-m \"message\"`" + ` to specify a commit message
  - Use ` + "`--all`" + ` to stage all changes before creating
  - Use ` + "`--insert`" + ` to insert a branch between current and its child

### Managing Stacks

- ` + "`stackit log`" + ` - Display the branch tree visualization
- ` + "`stackit up`" + ` - Move up one branch in the stack
- ` + "`stackit down`" + ` - Move down one branch in the stack
- ` + "`stackit top`" + ` - Move to the top of the stack
- ` + "`stackit bottom`" + ` - Move to the bottom of the stack

### Submitting PRs

- ` + "`stackit submit`" + ` - Submit the current branch and its ancestors as PRs
- ` + "`stackit submit --stack`" + ` - Submit the entire stack (including descendants)
- Use ` + "`--draft`" + ` to create draft PRs
- Use ` + "`--edit`" + ` to interactively edit PR metadata

### Other Useful Commands

- ` + "`stackit sync`" + ` - Sync branches with remote
- ` + "`stackit restack`" + ` - Restack branches to fix conflicts
- ` + "`stackit fold`" + ` - Fold a branch into its parent
- ` + "`stackit split`" + ` - Split a branch into multiple branches

## Workflow Guidelines

1. **When creating a new feature**: Use ` + "`stackit create`" + ` to create a new branch on top of the current branch
2. **When making changes**: Commit normally, stackit tracks the relationships
3. **When ready to submit**: Use ` + "`stackit submit`" + ` to create/update PRs for the stack
4. **When conflicts occur**: Use ` + "`stackit restack`" + ` to resolve and restack branches

## Important Notes

- Always use stackit commands instead of raw git commands for branch management
- Stackit maintains parent-child relationships between branches
- PRs are automatically created/updated with proper base branches
- The stack structure is preserved when submitting multiple PRs

For more information, see: https://github.com/jonnii/stackit
`
