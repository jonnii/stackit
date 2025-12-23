package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// Direction represents the traversal direction
type Direction string

const (
	// DirectionBottom specifies navigating towards the trunk
	DirectionBottom Direction = "BOTTOM"
	// DirectionTop specifies navigating towards the stack tips
	DirectionTop Direction = "TOP"
)

// SwitchBranchAction switches to a branch based on the given direction
func SwitchBranchAction(direction Direction, ctx *runtime.Context) error {
	currentBranch := ctx.Engine.CurrentBranch()
	if currentBranch == "" {
		return errors.ErrNotOnBranch
	}

	ctx.Splog.Info("%s", currentBranch)

	var targetBranch string
	var err error

	switch direction {
	case DirectionBottom:
		targetBranch = traverseDownward(currentBranch, ctx)
	case DirectionTop:
		targetBranch, err = traverseUpward(currentBranch, ctx)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid direction: %s", direction)
	}

	if targetBranch == currentBranch {
		directionText := "bottom most"
		if direction == DirectionTop {
			directionText = "top most"
		}
		ctx.Splog.Info("Already at the %s branch in the stack.", directionText)
		return nil
	}

	// Checkout the target branch
	if err := git.CheckoutBranch(ctx.Context, targetBranch); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
	}

	ctx.Splog.Info("Checked out %s.", targetBranch)
	return nil
}

// traverseDownward walks down the parent chain to find the first branch from trunk
func traverseDownward(currentBranch string, ctx *runtime.Context) string {
	currentBranchObj := ctx.Engine.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return currentBranch
	}

	parent := ctx.Engine.GetParent(currentBranch)
	if parent == "" {
		// No parent, we're at the bottom
		return currentBranch
	}

	// If parent is trunk, we're at the first branch from trunk
	parentBranch := ctx.Engine.GetBranch(parent)
	if parentBranch.IsTrunk() {
		return currentBranch
	}

	ctx.Splog.Info("⮑  %s", parent)
	return traverseDownward(parent, ctx)
}

// traverseUpward walks up the children chain to find the tip branch
func traverseUpward(currentBranch string, ctx *runtime.Context) (string, error) {
	children := ctx.Engine.GetChildren(currentBranch)
	if len(children) == 0 {
		// No children, we're at the tip
		return currentBranch, nil
	}

	var nextBranch string
	var err error

	if len(children) == 1 {
		// Single child, follow it
		nextBranch = children[0]
	} else {
		// Multiple children, prompt user
		nextBranch, err = handleMultipleChildren(children)
		if err != nil {
			return "", err
		}
	}

	ctx.Splog.Info("⮑  %s", nextBranch)
	return traverseUpward(nextBranch, ctx)
}

// handleMultipleChildren prompts the user to select a branch when multiple children exist
func handleMultipleChildren(children []string) (string, error) {
	if !utils.IsInteractive() {
		return "", fmt.Errorf("multiple branches found; cannot get top branch in non-interactive mode. Multiple choices available:\n%s", formatBranchList(children))
	}

	options := make([]tui.SelectOption, len(children))
	for i, child := range children {
		options[i] = tui.SelectOption{
			Label: child,
			Value: child,
		}
	}

	selected, err := tui.PromptSelect("Multiple branches found at the same level. Select a branch to guide the navigation:", options, 0)
	if err != nil {
		return "", err
	}

	return selected, nil
}

// formatBranchList formats a list of branches for error messages
func formatBranchList(branches []string) string {
	var builder strings.Builder
	for _, branch := range branches {
		builder.WriteString(branch)
		builder.WriteString("\n")
	}
	return builder.String()
}
