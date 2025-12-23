package actions

import (
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// TrackOptions contains options for the track command
type TrackOptions struct {
	BranchName string
	Force      bool
	Parent     string
}

// TrackAction performs the track operation
func TrackAction(ctx *runtime.Context, opts TrackOptions) error {
	eng := ctx.Engine
	branchName := opts.BranchName

	// Handle --parent flag (single branch tracking)
	if opts.Parent != "" {
		parent := opts.Parent
		// Validate parent exists (refresh branch list if needed)
		allBranches := eng.AllBranchNames()
		parentExists := false
		for _, name := range allBranches {
			if name == parent {
				parentExists = true
				break
			}
		}
		if !parentExists && parent != eng.Trunk() {
			// Refresh branches list and check again
			branches, err := git.GetAllBranchNames()
			if err == nil {
				parentExists = false
				for _, name := range branches {
					if name == parent {
						parentExists = true
						break
					}
				}
			}
			if !parentExists {
				return fmt.Errorf("parent branch %s does not exist", parent)
			}
		}

		// Validate parent is tracked (or is trunk)
		if !eng.IsTrunk(parent) && !eng.IsBranchTracked(parent) {
			return fmt.Errorf("parent branch %s must be tracked (or be trunk)", parent)
		}

		// Validate parent is an ancestor (unless force is used)
		if !opts.Force {
			parentRev, err := git.GetRevision(parent)
			if err != nil {
				return fmt.Errorf("failed to get parent revision: %w", err)
			}
			branchRev, err := git.GetRevision(branchName)
			if err != nil {
				return fmt.Errorf("failed to get branch revision: %w", err)
			}
			isAnc, err := git.IsAncestor(parentRev, branchRev)
			if err != nil {
				return fmt.Errorf("failed to check ancestry: %w", err)
			}
			if !isAnc {
				return fmt.Errorf("parent branch %s is not an ancestor of %s (use --force to override)", parent, branchName)
			}
		}

		// Track the branch
		if err := eng.TrackBranch(ctx.Context, branchName, parent); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}

		ctx.Splog.Info("Tracked %s with parent %s.", tui.ColorBranchName(branchName, false), tui.ColorBranchName(parent, false))
		return nil
	}

	// Handle --force flag (auto-detection without prompt)
	if opts.Force {
		ancestors, err := eng.FindMostRecentTrackedAncestors(ctx.Context, branchName)
		if err != nil {
			return fmt.Errorf("failed to find tracked ancestor: %w", err)
		}
		parentBranch := ancestors[0]

		if err := eng.TrackBranch(ctx.Context, branchName, parentBranch); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}

		ctx.Splog.Info("Tracked %s with parent %s.", tui.ColorBranchName(branchName, false), tui.ColorBranchName(parentBranch, false))
		return nil
	}

	// Interactive mode: recursively track a stack
	return trackBranchRecursively(ctx, branchName)
}

// trackBranchRecursively interactively tracks a branch and its descendants
func trackBranchRecursively(ctx *runtime.Context, branchName string) error {
	eng := ctx.Engine

	// Check if branch is already tracked
	if eng.IsBranchTracked(branchName) {
		ctx.Splog.Info("%s is already tracked.", tui.ColorBranchName(branchName, false))
		// Still ask if user wants to track descendants
	} else {
		// Try auto-detection (single unambiguous non-trunk tracked ancestor)
		var parentBranch string
		ancestors, err := eng.FindMostRecentTrackedAncestors(ctx.Context, branchName)
		if err == nil && len(ancestors) == 1 && ancestors[0] != eng.Trunk() {
			parentBranch = ancestors[0]
			ctx.Splog.Info("Auto-detected parent %s for %s.", tui.ColorBranchName(parentBranch, false), tui.ColorBranchName(branchName, false))
		} else {
			// Select parent interactively
			parentBranch, err = selectParentBranch(ctx, branchName)
			if err != nil {
				return err
			}
		}

		// Track the branch
		if err := eng.TrackBranch(ctx.Context, branchName, parentBranch); err != nil {
			return fmt.Errorf("failed to track branch: %w", err)
		}

		ctx.Splog.Info("Tracked %s with parent %s.", tui.ColorBranchName(branchName, false), tui.ColorBranchName(parentBranch, false))
	}

	// Find untracked children and ask to track them
	allBranches := eng.AllBranchNames()
	untrackedChildren := []string{}

	for _, candidate := range allBranches {
		if candidate == branchName {
			continue
		}

		// Check if candidate is a child (has this branch as merge base)
		mergeBase, err := git.GetMergeBase(candidate, branchName)
		if err != nil {
			continue
		}

		branchRev, err := git.GetRevision(branchName)
		if err != nil {
			continue
		}

		// If merge base is the branch we just tracked, candidate is a child
		if mergeBase == branchRev && !eng.IsBranchTracked(candidate) {
			untrackedChildren = append(untrackedChildren, candidate)
		}
	}

	// Recursively track children
	for _, child := range untrackedChildren {
		// Ask if user wants to track this child
		shouldTrack, err := promptTrackChild(child, branchName)
		if err != nil {
			return err
		}

		if shouldTrack {
			if err := trackBranchRecursively(ctx, child); err != nil {
				return err
			}
		}
	}

	return nil
}

// selectParentBranch interactively selects a parent branch for tracking
func selectParentBranch(ctx *runtime.Context, branchName string) (string, error) {
	eng := ctx.Engine
	trunk := eng.Trunk()

	// Render the tree to get visual context for each branch
	renderer := tui.NewStackTreeRenderer(
		branchName,
		trunk,
		eng.GetChildren,
		eng.GetParent,
		eng.IsTrunk,
		func(b string) bool { return eng.IsBranchUpToDate(b) },
	)

	// Render the full tree from trunk
	treeLines := renderer.RenderStack(trunk, tui.TreeRenderOptions{
		Short:             true,
		NoStyleBranchName: true, // We'll add our own coloring
	})

	// Map branch name to its tree display string
	branchToDisplay := make(map[string]string)
	for _, line := range treeLines {
		// Tree line format: "│ ◯▸branchName" (short format from StackTreeRenderer)
		arrowIdx := strings.Index(line, "▸")
		if arrowIdx != -1 {
			name := strings.Fields(line[arrowIdx+1:])[0]
			branchToDisplay[name] = line
		}
	}

	// Build list of candidate parents (trunk + all tracked branches)
	var choices []tui.BranchChoice
	initialIndex := -1

	// Add trunk first
	display := branchToDisplay[trunk]
	if display == "" {
		display = tui.ColorBranchName(trunk, false) + " (trunk)"
	} else {
		display = strings.Replace(display, trunk, tui.ColorBranchName(trunk, false)+" (trunk)", 1)
	}
	choices = append(choices, tui.BranchChoice{
		Display: display,
		Value:   trunk,
	})

	// Add all tracked branches
	allBranches := eng.AllBranchNames()
	for _, candidate := range allBranches {
		if candidate == branchName || candidate == trunk {
			continue
		}

		if eng.IsBranchTracked(candidate) {
			display := branchToDisplay[candidate]
			if display == "" {
				display = tui.ColorBranchName(candidate, false)
			} else {
				display = strings.Replace(display, candidate, tui.ColorBranchName(candidate, false), 1)
			}
			choices = append(choices, tui.BranchChoice{
				Display: display,
				Value:   candidate,
			})
		}
	}

	if len(choices) == 0 {
		return "", fmt.Errorf("no tracked branches available (trunk should always be available)")
	}

	// Try to find a good default (most recent tracked ancestor)
	ancestors, err := eng.FindMostRecentTrackedAncestors(ctx.Context, branchName)
	if err == nil && len(ancestors) > 0 {
		defaultParent := ancestors[0]
		for i, choice := range choices {
			if choice.Value == defaultParent {
				initialIndex = i
				break
			}
		}
	}

	// If no default found, use trunk
	if initialIndex < 0 {
		initialIndex = 0
	}

	// Prompt user
	selected, err := tui.PromptBranchSelection(
		fmt.Sprintf("Select parent for %s:", tui.ColorBranchName(branchName, false)),
		choices,
		initialIndex,
	)
	if err != nil {
		return "", err
	}

	return selected, nil
}

// promptTrackChild asks if user wants to track a child branch
func promptTrackChild(childName, parentName string) (bool, error) {
	message := fmt.Sprintf("Found untracked child branch %s of %s. Track it?", tui.ColorBranchName(childName, false), tui.ColorBranchName(parentName, false))
	options := []tui.SelectOption{
		{Label: "Yes", Value: yesResponse},
		{Label: "No", Value: noResponse},
	}

	selected, err := tui.PromptSelect(message, options, 0)
	if err != nil {
		return false, err
	}

	return selected == yesResponse, nil
}
