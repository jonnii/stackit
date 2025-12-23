package actions

import (
	"context"
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
)

// ReorderAction performs the reorder operation
func ReorderAction(ctx *runtime.Context) error {
	eng := ctx.Engine
	splog := ctx.Splog
	gctx := ctx.Context

	// Pre-checks: validate on branch
	currentBranch, err := utils.ValidateOnBranch(ctx)
	if err != nil {
		return err
	}

	// Pre-checks: ensure no rebase in progress
	if err := utils.CheckRebaseInProgress(gctx); err != nil {
		return err
	}

	// Pre-checks: ensure no uncommitted changes
	if utils.HasUncommittedChanges(gctx) {
		return fmt.Errorf("cannot reorder with uncommitted changes. Please commit or stash them first")
	}

	// Prevent reordering trunk
	currentBranchObj := eng.GetBranch(currentBranch)
	if currentBranchObj.IsTrunk() {
		return fmt.Errorf("cannot reorder trunk branch")
	}

	// Collect branches: get ancestors from trunk to current branch
	stack := eng.GetRelativeStack(currentBranch, engine.Scope{
		RecursiveParents:  true,
		IncludeCurrent:    true,
		RecursiveChildren: false,
	})

	// Filter out trunk and get only tracked branches
	branches := []string{}
	for _, branch := range stack {
		branchObj := eng.GetBranch(branch)
		if !branchObj.IsTrunk() && branchObj.IsTracked() {
			branches = append(branches, branch)
		}
	}

	// Minimum requirements: need at least 2 branches to reorder
	if len(branches) < 2 {
		return fmt.Errorf("need at least 2 branches to reorder. Found %d branch(es)", len(branches))
	}

	// Store original order for comparison
	originalOrder := make([]string, len(branches))
	copy(originalOrder, branches)

	// Open TUI or Editor to get new order
	var newOrder []string
	if tui.IsTTY() {
		var err error
		newOrder, err = tui.RunReorderTUI(branches)
		if err != nil {
			if err.Error() == "reorder canceled" {
				splog.Info("Reorder canceled.")
				return nil
			}
			return fmt.Errorf("TUI failed: %w", err)
		}
	} else {
		// Create editor content with instructions
		editorContent := buildEditorContent(branches)

		// Open editor
		editedContent, err := tui.OpenEditor(editorContent, "stackit-reorder-*.txt")
		if err != nil {
			return fmt.Errorf("failed to open editor: %w", err)
		}

		// Parse and validate edited content
		newOrder, err = parseEditorContent(editedContent, originalOrder)
		if err != nil {
			return err
		}
	}

	// Check if order actually changed
	if slicesEqual(originalOrder, newOrder) {
		splog.Info("Branch order unchanged. No action taken.")
		return nil
	}

	// Update parent relationships
	if err := updateParentRelationships(gctx, eng, newOrder); err != nil {
		return fmt.Errorf("failed to update parent relationships: %w", err)
	}

	// Identify affected branches: find the first branch that moved or changed parent
	firstAffectedBranch := findFirstAffectedBranch(eng, originalOrder, newOrder)

	// Get all affected branches (first affected and all its descendants)
	affectedBranches := eng.GetRelativeStack(firstAffectedBranch, engine.Scope{
		RecursiveChildren: true,
		IncludeCurrent:    true,
		RecursiveParents:  false,
	})

	// Restack all affected branches
	if err := RestackBranches(gctx, affectedBranches, eng, splog, ctx.RepoRoot); err != nil {
		return fmt.Errorf("failed to restack branches: %w", err)
	}

	splog.Info("Reordered and restacked branches.")
	return nil
}

// buildEditorContent creates the initial editor content with instructions
func buildEditorContent(branches []string) string {
	var sb strings.Builder
	sb.WriteString("# Reorder branches by moving lines up or down.\n")
	sb.WriteString("# Lines starting with '#' are ignored.\n")
	sb.WriteString("# Do not add or remove branches.\n")
	sb.WriteString("# Save and close to apply changes.\n")
	sb.WriteString("#\n")
	for _, branch := range branches {
		sb.WriteString(branch)
		sb.WriteString("\n")
	}
	return sb.String()
}

// parseEditorContent parses the edited content and validates it
func parseEditorContent(content string, originalBranches []string) ([]string, error) {
	lines := strings.Split(content, "\n")
	branches := []string{}

	// Create a set of original branches for validation
	originalSet := make(map[string]bool)
	for _, b := range originalBranches {
		originalSet[b] = true
	}

	// Parse lines, ignoring comments and empty lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		branches = append(branches, line)
	}

	// Validate: check for duplicates
	seen := make(map[string]bool)
	for _, branch := range branches {
		if seen[branch] {
			return nil, fmt.Errorf("duplicate branch found: %s", branch)
		}
		seen[branch] = true
	}

	// Validate: check that all original branches are present
	for _, original := range originalBranches {
		if !seen[original] {
			return nil, fmt.Errorf("branch %s was removed. Use 'stackit delete' to remove branches", original)
		}
	}

	// Validate: check that no new branches were added
	if len(branches) != len(originalBranches) {
		return nil, fmt.Errorf("number of branches changed. Expected %d, got %d", len(originalBranches), len(branches))
	}

	// Validate: check that all branches are valid (exist in original set)
	for _, branch := range branches {
		if !originalSet[branch] {
			return nil, fmt.Errorf("unknown branch: %s", branch)
		}
	}

	return branches, nil
}

// updateParentRelationships updates the parent of each branch in the new order
func updateParentRelationships(ctx context.Context, eng engine.Engine, newOrder []string) error {
	// Set parent of first branch to trunk
	trunk := eng.Trunk()
	if len(newOrder) > 0 {
		if err := eng.SetParent(ctx, newOrder[0], trunk.Name); err != nil {
			return fmt.Errorf("failed to set parent of %s to %s: %w", newOrder[0], trunk.Name, err)
		}
	}

	// Set parent of each subsequent branch to the branch before it
	for i := 1; i < len(newOrder); i++ {
		if err := eng.SetParent(ctx, newOrder[i], newOrder[i-1]); err != nil {
			return fmt.Errorf("failed to set parent of %s to %s: %w", newOrder[i], newOrder[i-1], err)
		}
	}

	return nil
}

// findFirstAffectedBranch finds the first branch that moved or changed parent
func findFirstAffectedBranch(eng engine.Engine, originalOrder, newOrder []string) string {
	trunk := eng.Trunk().Name
	// Create a map of original positions
	originalPos := make(map[string]int)
	for i, branch := range originalOrder {
		originalPos[branch] = i
	}

	// Find the first branch that moved or has a different parent
	for i, branch := range newOrder {
		// Check if position changed
		if originalPos[branch] != i {
			return branch
		}

		// Check if parent changed
		var expectedParent string
		if i == 0 {
			expectedParent = trunk
		} else {
			expectedParent = newOrder[i-1]
		}

		currentParent := eng.GetParent(branch)
		if currentParent == "" {
			currentParent = trunk
		}

		if currentParent != expectedParent {
			return branch
		}
	}

	// Fallback: return first branch if no changes detected (shouldn't happen)
	if len(newOrder) > 0 {
		return newOrder[0]
	}
	return ""
}

// slicesEqual checks if two string slices are equal
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
