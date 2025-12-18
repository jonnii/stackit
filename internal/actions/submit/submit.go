package submit

import (
	"context"
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
)

// Options contains options for the submit command
type Options struct {
	Branch               string
	Stack                bool
	Force                bool
	DryRun               bool
	Confirm              bool
	UpdateOnly           bool
	Always               bool
	Restack              bool
	Draft                bool
	Publish              bool
	Edit                 bool
	EditTitle            bool
	EditDescription      bool
	NoEdit               bool
	NoEditTitle          bool
	NoEditDescription    bool
	Reviewers            string
	TeamReviewers        string
	MergeWhenReady       bool
	RerequestReview      bool
	View                 bool
	Web                  bool
	Comment              string
	TargetTrunk          string
	IgnoreOutOfSyncTrunk bool
	// SkipPush skips pushing branches to remote (for testing)
	SkipPush bool
}

// Info contains information about a branch to submit
type Info struct {
	BranchName string
	Head       string
	Base       string
	HeadSHA    string
	BaseSHA    string
	Action     string // "create" or "update"
	PRNumber   *int
	Metadata   *PRMetadata
}

// Action performs the submit operation
func Action(ctx *runtime.Context, opts Options) error {
	eng := ctx.Engine
	splog := ctx.Splog

	// Create UI early - all output goes through this
	ui := tui.NewSubmitUI(splog)

	// Validate flags
	if opts.Draft && opts.Publish {
		return fmt.Errorf("can't use both --publish and --draft flags in one command")
	}

	// Get branches to submit
	branches, err := getBranchesToSubmit(opts, eng)
	if err != nil {
		return err
	}
	if len(branches) == 0 {
		splog.Info("No branches to submit.")
		return nil
	}

	currentBranch := eng.CurrentBranch()

	// Populate remote SHAs early for accurate display
	if err := eng.PopulateRemoteShas(); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Display the stack tree with PR annotations
	renderer, rootBranch := getStackTreeRenderer(branches, opts, eng, currentBranch)
	ui.ShowStack(renderer, rootBranch)

	// Restack if requested
	if opts.Restack {
		ui.ShowRestackStart()
		repoRoot := ctx.RepoRoot
		if repoRoot == "" {
			repoRoot, _ = git.GetRepoRoot()
		}
		if err := actions.RestackBranches(branches, eng, splog, repoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
		ui.ShowRestackComplete()
	}

	// Validate and prepare branches
	ui.ShowPreparing()

	if err := ValidateBranchesToSubmit(branches, eng, ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Prepare branches for submit (show planning phase with current indicator)
	submissionInfos, err := prepareBranchesForSubmit(branches, opts, eng, ctx, currentBranch, ui)
	if err != nil {
		return fmt.Errorf("failed to prepare branches: %w", err)
	}

	// Check if we should abort
	if opts.DryRun {
		ui.ShowDryRunComplete()
		return nil
	}

	if len(submissionInfos) == 0 {
		ui.ShowNoChanges()
		return nil
	}

	if opts.Confirm {
		// TODO: Add interactive confirmation prompt
	}

	// Build progress items
	progressItems := make([]tui.SubmitItem, len(submissionInfos))
	for i, info := range submissionInfos {
		progressItems[i] = tui.SubmitItem{
			BranchName: info.BranchName,
			Action:     info.Action,
			PRNumber:   info.PRNumber,
			Status:     "pending",
		}
	}

	// Start submission phase
	ui.StartSubmitting(progressItems)

	githubCtx := context.Background()
	githubClient, err := getGitHubClient(ctx)
	if err != nil {
		return err
	}
	repoOwner, repoName := githubClient.GetOwnerRepo()

	remote := "origin" // TODO: Get from config
	for _, submissionInfo := range submissionInfos {
		ui.UpdateSubmitItem(submissionInfo.BranchName, "submitting", "", nil)

		if err := pushBranchIfNeeded(submissionInfo, opts, remote, eng); err != nil {
			ui.UpdateSubmitItem(submissionInfo.BranchName, "error", "", err)
			ui.Complete()
			return err
		}

		var prURL string
		if submissionInfo.Action == "create" {
			prURL, err = createPullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
		} else {
			prURL, err = updatePullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
		}

		if err != nil {
			ui.UpdateSubmitItem(submissionInfo.BranchName, "error", "", err)
			ui.Complete()
			return err
		}

		ui.UpdateSubmitItem(submissionInfo.BranchName, "done", prURL, nil)

		// Open in browser if requested
		if opts.View && prURL != "" {
			if err := actions.OpenBrowser(prURL); err != nil {
				splog.Debug("Failed to open browser: %v", err)
			}
		}
	}

	// Update PR body footers silently
	if err := updatePRFootersQuiet(branches, eng, githubCtx, githubClient, repoOwner, repoName); err != nil {
		splog.Debug("Failed to update PR footers: %v", err)
	}

	ui.Complete()

	return nil
}

// prepareBranchesForSubmit prepares submission info for each branch, outputting via UI
func prepareBranchesForSubmit(branches []string, opts Options, eng engine.Engine, ctx *runtime.Context, currentBranch string, ui tui.SubmitUI) ([]Info, error) {
	var submissionInfos []Info

	for _, branchName := range branches {
		parentBranchName := eng.GetParentPrecondition(branchName)
		prInfo, _ := eng.GetPrInfo(branchName)

		// Determine action
		action := "create"
		prNumber := (*int)(nil)
		if prInfo != nil && prInfo.Number != nil {
			action = "update"
			prNumber = prInfo.Number
		}

		isCurrent := branchName == currentBranch

		// Check if we should skip
		if opts.UpdateOnly && action == "create" {
			ui.ShowBranchPlan(branchName, action, isCurrent, true, "skipped, no existing PR")
			continue
		}

		// Check if branch needs update
		if action == "update" {
			baseChanged := prInfo.Base != parentBranchName
			branchChanged, _ := eng.BranchMatchesRemote(branchName)

			// Check if draft status needs to change
			draftStatusNeedsChange := false
			if opts.Draft && !prInfo.IsDraft {
				draftStatusNeedsChange = true
			} else if opts.Publish && prInfo.IsDraft {
				draftStatusNeedsChange = true
			}

			needsUpdate := baseChanged || !branchChanged || opts.Edit || opts.Always || draftStatusNeedsChange

			if !needsUpdate && !opts.Draft && !opts.Publish {
				ui.ShowBranchPlan(branchName, action, isCurrent, true, "no changes")
				continue
			}
		}

		// Prepare metadata
		metadataOpts := MetadataOptions{
			Edit:              opts.Edit && !opts.NoEdit,
			EditTitle:         opts.EditTitle && !opts.NoEditTitle,
			EditDescription:   opts.EditDescription && !opts.NoEditDescription,
			NoEdit:            opts.NoEdit,
			NoEditTitle:       opts.NoEditTitle,
			NoEditDescription: opts.NoEditDescription,
			Draft:             opts.Draft,
			Publish:           opts.Publish,
			Reviewers:         opts.Reviewers,
			ReviewersPrompt:   opts.Reviewers == "" && opts.Edit,
		}

		metadata, err := PreparePRMetadata(branchName, metadataOpts, eng, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare metadata for %s: %w", branchName, err)
		}

		// Get SHAs
		headSHA, _ := eng.GetRevision(branchName)
		baseSHA, _ := eng.GetRevision(parentBranchName)

		submissionInfo := Info{
			BranchName: branchName,
			Head:       branchName,
			Base:       parentBranchName,
			HeadSHA:    headSHA,
			BaseSHA:    baseSHA,
			Action:     action,
			PRNumber:   prNumber,
			Metadata:   metadata,
		}

		ui.ShowBranchPlan(branchName, action, isCurrent, false, "")

		submissionInfos = append(submissionInfos, submissionInfo)
	}

	return submissionInfos, nil
}

// getRecursiveDescendants gets all descendants of a branch recursively
func getRecursiveDescendants(eng engine.Engine, branchName string) []string {
	var descendants []string
	children := eng.GetChildren(branchName)
	for _, child := range children {
		descendants = append(descendants, child)
		// Recursively get descendants of this child
		childDescendants := getRecursiveDescendants(eng, child)
		descendants = append(descendants, childDescendants...)
	}
	return descendants
}

// getBranchesToSubmit returns the list of branches to submit based on options
func getBranchesToSubmit(opts Options, eng engine.Engine) ([]string, error) {
	// Get branch scope
	branchName := opts.Branch
	if branchName == "" {
		branchName = eng.CurrentBranch()
	}
	if branchName == "" {
		return nil, fmt.Errorf("not on a branch and no branch specified")
	}

	// Get stack of branches to submit
	scope := engine.Scope{RecursiveParents: true}
	var branches []string
	if opts.Stack {
		// Include descendants
		allBranches := eng.GetRelativeStack(branchName, scope)
		// Add the current branch itself
		allBranches = append(allBranches, branchName)
		// Also get descendants recursively
		children := eng.GetChildren(branchName)
		for _, child := range children {
			// Add the child itself
			allBranches = append(allBranches, child)
			// Get all descendants of this child recursively
			descendants := getRecursiveDescendants(eng, child)
			allBranches = append(allBranches, descendants...)
		}
		// Remove duplicates and trunk
		branchSet := make(map[string]bool)
		for _, b := range allBranches {
			if !eng.IsTrunk(b) && !branchSet[b] {
				branches = append(branches, b)
				branchSet[b] = true
			}
		}
	} else {
		// Just ancestors (including current branch)
		allBranches := eng.GetRelativeStack(branchName, scope)
		// Add the current branch itself
		allBranches = append(allBranches, branchName)
		// Filter out trunk
		for _, b := range allBranches {
			if !eng.IsTrunk(b) {
				branches = append(branches, b)
			}
		}
	}

	return branches, nil
}

// getGitHubClient returns the GitHub client from context
func getGitHubClient(ctx *runtime.Context) (git.GitHubClient, error) {
	if ctx.GitHubClient != nil {
		return ctx.GitHubClient, nil
	}
	return nil, fmt.Errorf("no GitHub client available - check your GITHUB_TOKEN")
}

// pushBranchIfNeeded pushes a branch to remote if needed
func pushBranchIfNeeded(submissionInfo Info, opts Options, remote string, eng engine.Engine) error {
	// Skip if dry run or skip push is set
	if opts.DryRun || opts.SkipPush {
		return nil
	}

	forceWithLease := !opts.Force
	if err := eng.PushBranch(submissionInfo.BranchName, remote, opts.Force, forceWithLease); err != nil {
		if strings.Contains(err.Error(), "stale info") {
			return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning", submissionInfo.BranchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", submissionInfo.BranchName, err)
	}
	return nil
}

// createPullRequest creates a new pull request
func createPullRequest(submissionInfo Info, opts Options, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *tui.Splog) (string, error) {
	prURL, err := createPullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
	if err != nil {
		return "", err
	}

	splog.Info("%s: %s (%s)",
		tui.ColorBranchName(submissionInfo.BranchName, true),
		prURL,
		tui.ColorDim("created"))

	return prURL, nil
}

// createPullRequestQuiet creates a new pull request without logging
func createPullRequestQuiet(submissionInfo Info, opts Options, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string) (string, error) {
	createOpts := git.CreatePROptions{
		Title:         submissionInfo.Metadata.Title,
		Body:          submissionInfo.Metadata.Body,
		Head:          submissionInfo.Head,
		Base:          submissionInfo.Base,
		Draft:         submissionInfo.Metadata.IsDraft,
		Reviewers:     submissionInfo.Metadata.Reviewers,
		TeamReviewers: submissionInfo.Metadata.TeamReviewers,
	}
	pr, err := githubClient.CreatePullRequest(githubCtx, repoOwner, repoName, createOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Update PR info
	prNumber := pr.Number
	prURL := pr.HTMLURL
	eng.UpsertPrInfo(submissionInfo.BranchName, &engine.PrInfo{
		Number:  &prNumber,
		Title:   submissionInfo.Metadata.Title,
		Body:    submissionInfo.Metadata.Body,
		IsDraft: submissionInfo.Metadata.IsDraft,
		State:   "OPEN",
		Base:    submissionInfo.Base,
		URL:     prURL,
	})

	return prURL, nil
}

// updatePullRequest updates an existing pull request
func updatePullRequest(submissionInfo Info, opts Options, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *tui.Splog) (string, error) {
	prURL, err := updatePullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
	if err != nil {
		return "", err
	}

	splog.Info("%s: %s (%s)",
		tui.ColorBranchName(submissionInfo.BranchName, true),
		prURL,
		tui.ColorDim("updated"))

	return prURL, nil
}

// updatePullRequestQuiet updates an existing pull request without logging
func updatePullRequestQuiet(submissionInfo Info, opts Options, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string) (string, error) {
	// Check if base changed
	prInfo, _ := eng.GetPrInfo(submissionInfo.BranchName)
	baseChanged := false
	if prInfo != nil && prInfo.Base != submissionInfo.Base {
		baseChanged = true
	}

	updateOpts := git.UpdatePROptions{
		Title:           &submissionInfo.Metadata.Title,
		Body:            &submissionInfo.Metadata.Body,
		Reviewers:       submissionInfo.Metadata.Reviewers,
		TeamReviewers:   submissionInfo.Metadata.TeamReviewers,
		MergeWhenReady:  &opts.MergeWhenReady,
		RerequestReview: opts.RerequestReview,
	}

	// Only update draft status if it's explicitly set via flags
	if opts.Draft || opts.Publish {
		updateOpts.Draft = &submissionInfo.Metadata.IsDraft
	}
	if baseChanged {
		updateOpts.Base = &submissionInfo.Base
	}
	if err := githubClient.UpdatePullRequest(githubCtx, repoOwner, repoName, *submissionInfo.PRNumber, updateOpts); err != nil {
		return "", fmt.Errorf("failed to update PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Get PR URL
	prInfo, _ = eng.GetPrInfo(submissionInfo.BranchName)
	var prURL string
	if prInfo != nil && prInfo.URL != "" {
		prURL = prInfo.URL
	} else {
		// Get from GitHub
		pr, err := githubClient.GetPullRequestByBranch(githubCtx, repoOwner, repoName, submissionInfo.BranchName)
		if err == nil && pr != nil {
			prURL = pr.HTMLURL
		}
	}

	eng.UpsertPrInfo(submissionInfo.BranchName, &engine.PrInfo{
		Number:  submissionInfo.PRNumber,
		Title:   submissionInfo.Metadata.Title,
		Body:    submissionInfo.Metadata.Body,
		IsDraft: submissionInfo.Metadata.IsDraft,
		State:   "OPEN",
		Base:    submissionInfo.Base,
		URL:     prURL,
	})

	return prURL, nil
}

// updatePRFooters updates PR body footers with dependency trees
func updatePRFooters(branches []string, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *tui.Splog) error {
	splog.Info("Updating dependency trees in PR bodies...")
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(branchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			continue
		}

		footer := actions.CreatePRBodyFooter(branchName, eng)
		updatedBody := actions.UpdatePRBodyFooter(prInfo.Body, footer)

		if updatedBody != prInfo.Body {
			updateOpts := git.UpdatePROptions{
				Body: &updatedBody,
			}
			if err := githubClient.UpdatePullRequest(githubCtx, repoOwner, repoName, *prInfo.Number, updateOpts); err != nil {
				splog.Debug("Failed to update PR footer for %s: %v", branchName, err)
				continue
			}

			prURL := ""
			if prInfo.URL != "" {
				prURL = prInfo.URL
			}
			splog.Info("%s: %s (%s)",
				tui.ColorBranchName(branchName, true),
				prURL,
				tui.ColorDim("updated"))
		}
	}
	return nil
}

// updatePRFootersQuiet updates PR body footers silently (no logging)
func updatePRFootersQuiet(branches []string, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string) error {
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(branchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			continue
		}

		footer := actions.CreatePRBodyFooter(branchName, eng)
		updatedBody := actions.UpdatePRBodyFooter(prInfo.Body, footer)

		if updatedBody != prInfo.Body {
			updateOpts := git.UpdatePROptions{
				Body: &updatedBody,
			}
			if err := githubClient.UpdatePullRequest(githubCtx, repoOwner, repoName, *prInfo.Number, updateOpts); err != nil {
				continue
			}
		}
	}
	return nil
}

// getStackTreeRenderer returns the stack tree renderer with PR annotations
func getStackTreeRenderer(branches []string, opts Options, eng engine.Engine, currentBranch string) (*tui.StackTreeRenderer, string) {
	// Create the tree renderer
	renderer := tui.NewStackTreeRenderer(
		currentBranch,
		eng.Trunk(),
		eng.GetChildren,
		eng.GetParent,
		eng.IsTrunk,
		eng.IsBranchFixed,
	)

	// Build annotations for each branch
	annotations := make(map[string]tui.BranchAnnotation)
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	for _, branchName := range eng.AllBranchNames() {
		prInfo, _ := eng.GetPrInfo(branchName)
		if prInfo == nil && !branchSet[branchName] {
			continue
		}

		annotation := tui.BranchAnnotation{
			NeedsRestack: !eng.IsBranchFixed(branchName),
		}

		if prInfo != nil && prInfo.Number != nil {
			annotation.PRNumber = prInfo.Number
			if branchSet[branchName] {
				annotation.PRAction = "update"
			}
			annotation.IsDraft = prInfo.IsDraft
		} else if branchSet[branchName] {
			annotation.PRAction = "create"
			annotation.IsDraft = opts.Draft
		}

		annotations[branchName] = annotation
	}
	renderer.SetAnnotations(annotations)

	return renderer, eng.Trunk()
}
