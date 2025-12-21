package actions

import (
	"errors"
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/tui"
)

// ConfigListAction prints all configuration values in a formatted way
func ConfigListAction(repoRoot string) error {
	splog := tui.NewSplog()

	// Get trunk
	trunk, err := config.GetTrunk(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get trunk: %w", err)
	}

	// Get all trunks
	trunks, err := config.GetAllTrunks(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get trunks: %w", err)
	}

	// Get branch name pattern
	branchPattern, err := config.GetBranchNamePattern(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get branch.pattern: %w", err)
	}

	// Get submit.footer
	submitFooter, err := config.GetSubmitFooter(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get submit.footer: %w", err)
	}

	// Format and print
	var lines []string
	lines = append(lines, fmt.Sprintf("%s: %s", tui.ColorCyan("trunk"), trunk))

	if len(trunks) > 1 {
		additionalTrunks := []string{}
		for _, t := range trunks {
			if t != trunk {
				additionalTrunks = append(additionalTrunks, t)
			}
		}
		if len(additionalTrunks) > 0 {
			lines = append(lines, fmt.Sprintf("%s: %s", tui.ColorCyan("trunks"), strings.Join(additionalTrunks, ", ")))
		}
	}

	lines = append(lines, fmt.Sprintf("%s: %s", tui.ColorCyan("branch.pattern"), branchPattern))
	lines = append(lines, fmt.Sprintf("%s: %v", tui.ColorCyan("submit.footer"), submitFooter))

	splog.Page(strings.Join(lines, "\n"))
	splog.Newline()

	return nil
}

// ConfigTUIAction provides an interactive TUI for editing configuration
func ConfigTUIAction(repoRoot string) error {
	splog := tui.NewSplog()

	for {
		// Get current values
		branchPattern, err := config.GetBranchNamePattern(repoRoot)
		if err != nil {
			splog.Debug("Failed to get branch.pattern: %v", err)
			branchPattern = "{username}/{date}/{message}" // fallback
		}

		submitFooter, err := config.GetSubmitFooter(repoRoot)
		if err != nil {
			splog.Debug("Failed to get submit.footer: %v", err)
			submitFooter = true // fallback
		}

		// Build options with current values displayed
		options := []tui.SelectOption{
			{
				Label: fmt.Sprintf("branch.pattern: %s", branchPattern),
				Value: "branch.pattern",
			},
			{
				Label: fmt.Sprintf("submit.footer: %v", submitFooter),
				Value: "submit.footer",
			},
			{
				Label: "Exit",
				Value: "exit",
			},
		}

		// Show selection menu
		selected, err := tui.PromptSelect("Select a configuration option to edit:", options, 0)
		if err != nil {
			return err
		}

		if selected == "exit" {
			break
		}

		// Handle each option
		switch selected {
		case "branch.pattern":
			newPattern, err := tui.PromptTextInput(fmt.Sprintf("Enter branch name pattern (current: %s):", branchPattern), branchPattern)
			if err != nil {
				if errors.Is(err, tui.ErrInteractiveDisabled) || strings.Contains(err.Error(), "canceled") {
					continue
				}
				return err
			}
			if newPattern != "" && newPattern != branchPattern {
				if err := config.SetBranchNamePattern(repoRoot, newPattern); err != nil {
					splog.Info("Failed to set branch.pattern: %v", err)
					continue
				}
				splog.Info("Set branch.pattern to: %s", newPattern)
			}

		case "submit.footer":
			newValue, err := tui.PromptConfirm(fmt.Sprintf("Include PR footer in descriptions? (current: %v):", submitFooter), submitFooter)
			if err != nil {
				if errors.Is(err, tui.ErrInteractiveDisabled) || strings.Contains(err.Error(), "canceled") {
					continue
				}
				return err
			}
			if newValue != submitFooter {
				if err := config.SetSubmitFooter(repoRoot, newValue); err != nil {
					splog.Info("Failed to set submit.footer: %v", err)
					continue
				}
				splog.Info("Set submit.footer to: %v", newValue)
			}
		}
	}

	return nil
}
