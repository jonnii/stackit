package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"stackit.dev/stackit/internal/git"
)

// NewAgentCmd creates the agent command
func NewAgentCmd() *cobra.Command {
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
	var local bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize agent integration files",
		Long: `Install agent integration files for AI assistants.

By default, files are installed globally in ~/.claude/ and work across all repositories.
Use --local to install files in the current repository instead.

This will create:
  - .cursor/rules/stackit.md (for Cursor)
  - CLAUDE.md (for Claude Code - local only)
  - .claude/skills/stackit/ (Claude Code skill)
  - .claude/commands/ (Claude Code slash commands)

These files contain instructions for AI agents on how to use stackit commands
to manage stacked branches, create commits, submit PRs, and more.`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runAgentInit(local, force)
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install files in current repository instead of globally")
	cmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")

	return cmd
}

func runAgentInit(local, force bool) error {
	var baseDir string
	var err error

	if local {
		// Local installation - install in current repo
		if err := git.InitDefaultRepo(); err != nil {
			return fmt.Errorf("not a git repository: %w", err)
		}

		baseDir, err = git.GetRepoRoot()
		if err != nil {
			return fmt.Errorf("failed to get repo root: %w", err)
		}
	} else {
		// Global installation - install in home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = homeDir
	}

	// Check for existing installation and version
	if !force {
		if err := checkExistingInstallation(baseDir, local); err != nil {
			return err
		}
	}

	// Create .claude/skills/stackit directory
	skillDir := filepath.Join(baseDir, ".claude", "skills", "stackit")
	if err := os.MkdirAll(skillDir, 0750); err != nil {
		return fmt.Errorf("failed to create .claude/skills/stackit directory: %w", err)
	}

	// Write skill files
	skillFiles := map[string]string{
		"SKILL.md":     "templates/skill/SKILL.md",
		"reference.md": "templates/skill/reference.md",
	}

	for filename, templatePath := range skillFiles {
		content, err := skillTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		filePath := filepath.Join(skillDir, filename)
		if err := os.WriteFile(filePath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Create .claude/commands directory
	commandsDir := filepath.Join(baseDir, ".claude", "commands")
	if err := os.MkdirAll(commandsDir, 0750); err != nil {
		return fmt.Errorf("failed to create .claude/commands directory: %w", err)
	}

	// Write slash command files
	commands := []string{
		"stack-status.md",
		"stack-create.md",
		"stack-submit.md",
		"stack-fix.md",
		"stack-sync.md",
		"stack-restack.md",
	}

	for _, filename := range commands {
		templatePath := "templates/commands/" + filename
		content, err := commandTemplates.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", templatePath, err)
		}

		cmdPath := filepath.Join(commandsDir, filename)
		if err := os.WriteFile(cmdPath, content, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Create .cursor/rules directory if it doesn't exist
	cursorRulesDir := filepath.Join(baseDir, ".cursor", "rules")
	if err := os.MkdirAll(cursorRulesDir, 0750); err != nil {
		return fmt.Errorf("failed to create .cursor/rules directory: %w", err)
	}

	// Write Cursor rules file
	cursorContent, err := cursorTemplates.ReadFile("templates/cursor/stackit.md")
	if err != nil {
		return fmt.Errorf("failed to read Cursor template: %w", err)
	}

	cursorRulesPath := filepath.Join(cursorRulesDir, "stackit.md")
	if err := os.WriteFile(cursorRulesPath, cursorContent, 0600); err != nil {
		return fmt.Errorf("failed to write Cursor rules file: %w", err)
	}

	// Write CLAUDE.md file (only for local installation)
	if local {
		claudeContent, err := claudeTemplates.ReadFile("templates/claude/CLAUDE.md")
		if err != nil {
			return fmt.Errorf("failed to read CLAUDE.md template: %w", err)
		}

		claudePath := filepath.Join(baseDir, "CLAUDE.md")
		if err := os.WriteFile(claudePath, claudeContent, 0600); err != nil {
			return fmt.Errorf("failed to write CLAUDE.md file: %w", err)
		}
	}

	// Print success message
	installType := "globally"
	if local {
		installType = "locally"
	}

	fmt.Printf("✓ Installed agent files %s (version %s)\n\n", installType, TemplateVersion)
	fmt.Println("Claude Code integration:")
	fmt.Printf("✓ Created %s/.claude/skills/stackit/SKILL.md\n", getDisplayPath(baseDir, local))
	fmt.Printf("✓ Created %s/.claude/skills/stackit/reference.md\n", getDisplayPath(baseDir, local))
	fmt.Println()
	fmt.Println("Slash commands:")
	fmt.Printf("✓ Created %s/.claude/commands/stack-*.md (6 commands)\n", getDisplayPath(baseDir, local))
	fmt.Println()
	fmt.Println("Cursor integration:")
	fmt.Printf("✓ Created %s/.cursor/rules/stackit.md\n", getDisplayPath(baseDir, local))

	if local {
		fmt.Println()
		fmt.Println("Local integration:")
		fmt.Println("✓ Created CLAUDE.md")
	}

	fmt.Println()
	fmt.Println("Available Claude Code commands:")
	fmt.Println("  /stack-status  - View stack state and health")
	fmt.Println("  /stack-create  - Create branch with auto-naming")
	fmt.Println("  /stack-submit  - Submit PRs with generated descriptions")
	fmt.Println("  /stack-fix     - Diagnose and fix stack issues")
	fmt.Println("  /stack-sync    - Sync with trunk and cleanup")
	fmt.Println("  /stack-restack - Rebase all branches in stack")

	if !local {
		fmt.Println()
		fmt.Println("Tip: Use 'stackit agent init --local' to install files in a specific repository")
	}

	return nil
}

func checkExistingInstallation(baseDir string, local bool) error {
	// Check if SKILL.md exists and has version info
	skillPath := filepath.Join(baseDir, ".claude", "skills", "stackit", "SKILL.md")
	if content, err := os.ReadFile(skillPath); err == nil {
		// File exists, check version
		existingVersion := extractVersion(string(content))
		if existingVersion != "" && existingVersion != TemplateVersion {
			fmt.Printf("Found existing installation (version %s)\n", existingVersion)
			fmt.Printf("New version available: %s\n", TemplateVersion)
			fmt.Println()
			fmt.Println("Run with --force to update")
			return fmt.Errorf("existing installation found")
		}
	}

	return nil
}

func extractVersion(content string) string {
	// Look for version in frontmatter
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}

		if inFrontmatter && strings.HasPrefix(line, "version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return ""
}

func getDisplayPath(baseDir string, local bool) string {
	if local {
		return "."
	}
	return "~"
}
