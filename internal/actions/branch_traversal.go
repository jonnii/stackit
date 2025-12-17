package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/errors"
	"stackit.dev/stackit/internal/git"
)

// Direction represents the traversal direction
type Direction string

const (
	DirectionBottom Direction = "BOTTOM"
	DirectionTop    Direction = "TOP"
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
	if err := git.CheckoutBranch(targetBranch); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", targetBranch, err)
	}

	ctx.Splog.Info("Checked out %s.", targetBranch)
	return nil
}

// traverseDownward walks down the parent chain to find the first branch from trunk
func traverseDownward(currentBranch string, ctx *runtime.Context) string {
	if ctx.Engine.IsTrunk(currentBranch) {
		return currentBranch
	}

	parent := ctx.Engine.GetParent(currentBranch)
	if parent == "" {
		// No parent, we're at the bottom
		return currentBranch
	}

	// If parent is trunk, we're at the first branch from trunk
	if ctx.Engine.IsTrunk(parent) {
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
		nextBranch, err = handleMultipleChildren(children, ctx)
		if err != nil {
			return "", err
		}
	}

	ctx.Splog.Info("⮑  %s", nextBranch)
	return traverseUpward(nextBranch, ctx)
}

// handleMultipleChildren prompts the user to select a branch when multiple children exist
func handleMultipleChildren(children []string, ctx *runtime.Context) (string, error) {
	if !isInteractive() {
		return "", fmt.Errorf("cannot get top branch in non-interactive mode; multiple choices available:\n%s", formatBranchList(children))
	}

	ctx.Splog.Info("Multiple branches found at the same level. Select a branch to guide the navigation:")
	for i, child := range children {
		ctx.Splog.Info("%d. %s", i+1, child)
	}
	ctx.Splog.Info("Enter number (1-%d): ", len(children))

	var choice int
	_, err := fmt.Scanln(&choice)
	if err != nil {
		return "", fmt.Errorf("failed to read selection: %w", err)
	}

	if choice < 1 || choice > len(children) {
		return "", fmt.Errorf("invalid selection: %d (must be between 1 and %d)", choice, len(children))
	}

	return children[choice-1], nil
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
