package actions

import (
	"context"
	"fmt"
	"strings"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
	"stackit.dev/stackit/internal/runtime"
)

// isDemoMode checks if we're running in demo mode
func isDemoMode() bool {
	return runtime.IsDemoMode()
}

// SubmitOptions contains options for the submit command
type SubmitOptions struct {
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
	// For testing: optional GitHub client
	// If nil, will use context's GitHubClient
	GitHubClient git.GitHubClient
	// SkipPush skips pushing branches to remote (for testing)
	SkipPush bool
}

// SubmissionInfo contains information about a branch to submit
type SubmissionInfo struct {
	BranchName string
	Head       string
	Base       string
	HeadSHA    string
	BaseSHA    string
	Action     string // "create" or "update"
	PRNumber   *int
	Metadata   *PRMetadata
}

// SubmitResult tracks the results of a submit operation
type SubmitResult struct {
	BranchName string
	Action     string // "created" or "updated"
	URL        string
	IsCurrent  bool
}

// SubmitAction performs the submit operation
func SubmitAction(ctx *runtime.Context, opts SubmitOptions) error {
	eng := ctx.Engine
	splog := ctx.Splog

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
	displaySubmitStackTree(branches, opts, eng, splog, currentBranch)

	// Restack if requested (skip in demo mode)
	if opts.Restack && !isDemoMode() {
		splog.Info("Restacking branches before submitting...")
		repoRoot := ctx.RepoRoot
		if repoRoot == "" {
			repoRoot, _ = git.GetRepoRoot()
		}
		if err := RestackBranches(branches, eng, splog, repoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	} else if opts.Restack && isDemoMode() {
		splog.Info("[DEMO] Would restack branches before submitting...")
	}

	// Validate and prepare branches (combined message)
	splog.Info("Preparing...")

	// Skip validation in demo mode
	if !isDemoMode() {
		if err := ValidateBranchesToSubmit(branches, eng, ctx); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Prepare branches for submit (show planning phase with current indicator)
	submissionInfos, err := prepareBranchesForSubmitDemo(branches, opts, eng, ctx, currentBranch)
	if err != nil {
		return fmt.Errorf("failed to prepare branches: %w", err)
	}

	// Check if we should abort
	shouldAbort, err := shouldAbortSubmit(opts, len(submissionInfos) > 0, ctx)
	if err != nil {
		return err
	}
	if shouldAbort {
		return nil
	}

	// In demo mode, simulate the submission
	if isDemoMode() {
		return submitActionDemo(submissionInfos, currentBranch, eng, splog)
	}

	// Push branches and create/update PRs
	splog.Newline()
	splog.Info("Submitting...")
	githubCtx := context.Background()

	githubClient, err := getGitHubClient(opts, ctx)
	if err != nil {
		return err
	}
	repoOwner, repoName := githubClient.GetOwnerRepo()

	// Track results for summary
	var results []SubmitResult

	remote := "origin" // TODO: Get from config
	for _, submissionInfo := range submissionInfos {
		if err := pushBranchIfNeeded(submissionInfo, opts, remote); err != nil {
			return err
		}

		var prURL string
		var action string
		if submissionInfo.Action == "create" {
			prURL, err = createPullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
			if err != nil {
				return err
			}
			action = "created"
		} else {
			prURL, err = updatePullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
			if err != nil {
				return err
			}
			action = "updated"
		}

		results = append(results, SubmitResult{
			BranchName: submissionInfo.BranchName,
			Action:     action,
			URL:        prURL,
			IsCurrent:  submissionInfo.BranchName == currentBranch,
		})

		// Open in browser if requested
		if opts.View && prURL != "" {
			if err := openBrowser(prURL); err != nil {
				// Log error but don't fail the operation
				splog.Debug("Failed to open browser: %v", err)
			}
		}
	}

	// Update PR body footers silently
	if err := updatePRFootersQuiet(branches, eng, githubCtx, githubClient, repoOwner, repoName); err != nil {
		splog.Debug("Failed to update PR footers: %v", err)
	}

	// Print results
	for _, result := range results {
		splog.Info("  ✓ %s → %s", result.BranchName, result.URL)
	}

	// Print summary
	splog.Newline()
	createdCount := 0
	updatedCount := 0
	for _, result := range results {
		if result.Action == "created" {
			createdCount++
		} else {
			updatedCount++
		}
	}
	splog.Info("Done! %s", formatSubmitSummary(createdCount, updatedCount))

	return nil
}

// submitActionDemo simulates the submit operation in demo mode
func submitActionDemo(submissionInfos []SubmissionInfo, currentBranch string, eng engine.Engine, splog *output.Splog) error {
	splog.Newline()
	splog.Info("Submitting PRs...")

	// Build TUI items
	items := make([]output.SubmitItem, len(submissionInfos))
	for i, info := range submissionInfos {
		items[i] = output.SubmitItem{
			BranchName: info.BranchName,
			Action:     info.Action,
			PRNumber:   info.PRNumber,
			Status:     "pending",
		}
	}

	// Submit function for each item
	submitFunc := func(idx int) (string, error) {
		info := submissionInfos[idx]
		prNum := 100 + idx + 1
		if info.PRNumber != nil {
			prNum = *info.PRNumber
		}
		prURL := fmt.Sprintf("https://github.com/example/repo/pull/%d", prNum)

		err := eng.UpsertPrInfo(info.BranchName, &engine.PrInfo{
			Number:  &prNum,
			Title:   info.Metadata.Title,
			Body:    info.Metadata.Body,
			IsDraft: info.Metadata.IsDraft,
			State:   "OPEN",
			Base:    info.Base,
			URL:     prURL,
		})
		return prURL, err
	}

	return output.RunSubmitTUISimple(items, submitFunc, splog)
}

// prepareBranchesForSubmitDemo prepares submission info for demo mode (without interactive prompts)
func prepareBranchesForSubmitDemo(branches []string, opts SubmitOptions, eng engine.Engine, ctx *runtime.Context, currentBranch string) ([]SubmissionInfo, error) {
	// In demo mode, skip interactive prompts
	if isDemoMode() {
		var submissionInfos []SubmissionInfo
		for _, branchName := range branches {
			parentBranchName := eng.GetParent(branchName)
			if parentBranchName == "" {
				parentBranchName = eng.Trunk()
			}
			prInfo, _ := eng.GetPrInfo(branchName)

			action := "create"
			prNumber := (*int)(nil)
			if prInfo != nil && prInfo.Number != nil {
				action = "update"
				prNumber = prInfo.Number
			}

			isCurrent := branchName == currentBranch

			// Skip if update-only and no existing PR
			if opts.UpdateOnly && action == "create" {
				displayName := branchName
				if isCurrent {
					displayName = branchName + " (current)"
				}
				ctx.Splog.Info("  ▸ %s %s", output.ColorDim(displayName), output.ColorDim("— skipped, no existing PR"))
				continue
			}

			// Get SHAs (fake for demo)
			headSHA, _ := eng.GetRevision(branchName)
			baseSHA, _ := eng.GetRevision(parentBranchName)

			// Use PR title from existing PR info or generate from branch name
			title := branchName
			if prInfo != nil && prInfo.Title != "" {
				title = prInfo.Title
			}

			submissionInfo := SubmissionInfo{
				BranchName: branchName,
				Head:       branchName,
				Base:       parentBranchName,
				HeadSHA:    headSHA,
				BaseSHA:    baseSHA,
				Action:     action,
				PRNumber:   prNumber,
				Metadata: &PRMetadata{
					Title:   title,
					Body:    "Demo PR body",
					IsDraft: opts.Draft,
				},
			}

			actionLabel := "create"
			if action == "update" {
				actionLabel = "update"
			}
			ctx.Splog.Info("  ▸ %s → %s",
				output.ColorBranchName(branchName, isCurrent),
				output.ColorDim(actionLabel))

			submissionInfos = append(submissionInfos, submissionInfo)
		}
		return submissionInfos, nil
	}

	// Normal mode - use original function
	return prepareBranchesForSubmit(branches, opts, eng, ctx, currentBranch)
}

// formatSubmitSummary formats the summary message
func formatSubmitSummary(created, updated int) string {
	var parts []string
	if created > 0 {
		if created == 1 {
			parts = append(parts, "1 PR created")
		} else {
			parts = append(parts, fmt.Sprintf("%d PRs created", created))
		}
	}
	if updated > 0 {
		if updated == 1 {
			parts = append(parts, "1 PR updated")
		} else {
			parts = append(parts, fmt.Sprintf("%d PRs updated", updated))
		}
	}
	if len(parts) == 0 {
		return "No changes"
	}
	return strings.Join(parts, ", ")
}

// prepareBranchesForSubmit prepares submission info for each branch
func prepareBranchesForSubmit(branches []string, opts SubmitOptions, eng engine.Engine, ctx *runtime.Context, currentBranch string) ([]SubmissionInfo, error) {
	var submissionInfos []SubmissionInfo

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
			// For skipped branches, show dimmed with (current) indicator if applicable
			displayName := branchName
			if isCurrent {
				displayName = branchName + " (current)"
			}
			ctx.Splog.Info("  ▸ %s %s", output.ColorDim(displayName), output.ColorDim("— skipped, no existing PR"))
			continue
		}

		// Check if branch needs update
		if action == "update" {
			baseChanged := prInfo.Base != parentBranchName
			branchChanged, _ := eng.BranchMatchesRemote(branchName)

			// Check if draft status needs to change
			draftStatusNeedsChange := false
			if opts.Draft && !prInfo.IsDraft {
				// Want to mark as draft, but it's not draft
				draftStatusNeedsChange = true
			} else if opts.Publish && prInfo.IsDraft {
				// Want to publish, but it's currently draft
				draftStatusNeedsChange = true
			}

			needsUpdate := baseChanged || !branchChanged || opts.Edit || opts.Always || draftStatusNeedsChange

			if !needsUpdate && !opts.Draft && !opts.Publish {
				displayName := branchName
				if isCurrent {
					displayName = branchName + " (current)"
				}
				ctx.Splog.Info("  ▸ %s %s", output.ColorDim(displayName), output.ColorDim("— no changes"))
				continue
			}
		}

		// Prepare metadata
		metadataOpts := SubmitMetadataOptions{
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

		submissionInfo := SubmissionInfo{
			BranchName: branchName,
			Head:       branchName,
			Base:       parentBranchName,
			HeadSHA:    headSHA,
			BaseSHA:    baseSHA,
			Action:     action,
			PRNumber:   prNumber,
			Metadata:   metadata,
		}

		// Log action in planning phase - ColorBranchName adds (current) automatically
		actionLabel := "create"
		if action == "update" {
			actionLabel = "update"
		}
		ctx.Splog.Info("  ▸ %s → %s",
			output.ColorBranchName(branchName, isCurrent),
			output.ColorDim(actionLabel))

		submissionInfos = append(submissionInfos, submissionInfo)
	}

	return submissionInfos, nil
}

// shouldAbortSubmit checks if we should abort the submit operation
func shouldAbortSubmit(opts SubmitOptions, hasAnyPRs bool, ctx *runtime.Context) (bool, error) {
	if opts.DryRun {
		ctx.Splog.Info("Dry run complete.")
		return true, nil
	}

	if !hasAnyPRs {
		ctx.Splog.Info("All PRs up to date.")
		return true, nil
	}

	if opts.Confirm {
		// TODO: Add interactive confirmation prompt
		// For now, we'll just continue
	}

	return false, nil
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
func getBranchesToSubmit(opts SubmitOptions, eng engine.Engine) ([]string, error) {
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

// getGitHubClient returns the GitHub client from options or context
func getGitHubClient(opts SubmitOptions, ctx *runtime.Context) (git.GitHubClient, error) {
	if opts.GitHubClient != nil {
		return opts.GitHubClient, nil
	}

	if ctx.GitHubClient != nil {
		return ctx.GitHubClient, nil
	}

	// Try to create a new client
	ghClient, err := git.NewRealGitHubClient(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub client: %w", err)
	}
	return ghClient, nil
}

// pushBranchIfNeeded pushes a branch to remote if needed
func pushBranchIfNeeded(submissionInfo SubmissionInfo, opts SubmitOptions, remote string) error {
	// Skip if dry run or skip push is set
	if opts.DryRun || opts.SkipPush {
		return nil
	}

	forceWithLease := !opts.Force
	if err := git.PushBranch(submissionInfo.BranchName, remote, opts.Force, forceWithLease); err != nil {
		if strings.Contains(err.Error(), "stale info") {
			return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning", submissionInfo.BranchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", submissionInfo.BranchName, err)
	}
	return nil
}

// createPullRequest creates a new pull request
func createPullRequest(submissionInfo SubmissionInfo, opts SubmitOptions, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *output.Splog) (string, error) {
	prURL, err := createPullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
	if err != nil {
		return "", err
	}

	splog.Info("%s: %s (%s)",
		output.ColorBranchName(submissionInfo.BranchName, true),
		prURL,
		output.ColorDim("created"))

	return prURL, nil
}

// createPullRequestQuiet creates a new pull request without logging
func createPullRequestQuiet(submissionInfo SubmissionInfo, opts SubmitOptions, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string) (string, error) {
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
func updatePullRequest(submissionInfo SubmissionInfo, opts SubmitOptions, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *output.Splog) (string, error) {
	prURL, err := updatePullRequestQuiet(submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
	if err != nil {
		return "", err
	}

	splog.Info("%s: %s (%s)",
		output.ColorBranchName(submissionInfo.BranchName, true),
		prURL,
		output.ColorDim("updated"))

	return prURL, nil
}

// updatePullRequestQuiet updates an existing pull request without logging
func updatePullRequestQuiet(submissionInfo SubmissionInfo, opts SubmitOptions, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string) (string, error) {
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
func updatePRFooters(branches []string, eng engine.Engine, githubCtx context.Context, githubClient git.GitHubClient, repoOwner, repoName string, splog *output.Splog) error {
	splog.Info("Updating dependency trees in PR bodies...")
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(branchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			continue
		}

		footer := CreatePRBodyFooter(branchName, eng)
		updatedBody := UpdatePRBodyFooter(prInfo.Body, footer)

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
				output.ColorBranchName(branchName, true),
				prURL,
				output.ColorDim("updated"))
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

		footer := CreatePRBodyFooter(branchName, eng)
		updatedBody := UpdatePRBodyFooter(prInfo.Body, footer)

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

// displaySubmitStackTree displays the stack tree with PR annotations before submitting
func displaySubmitStackTree(branches []string, opts SubmitOptions, eng engine.Engine, splog *output.Splog, currentBranch string) {
	// Create the tree renderer
	renderer := output.NewStackTreeRenderer(
		currentBranch,
		eng.Trunk(),
		eng.GetChildren,
		eng.GetParent,
		eng.IsTrunk,
		eng.IsBranchFixed,
	)

	// Build annotations for each branch
	annotations := make(map[string]output.BranchAnnotation)
	for _, branchName := range branches {
		prInfo, _ := eng.GetPrInfo(branchName)

		annotation := output.BranchAnnotation{
			NeedsRestack: !eng.IsBranchFixed(branchName),
		}

		if prInfo != nil && prInfo.Number != nil {
			annotation.PRNumber = prInfo.Number
			annotation.PRAction = "update"
			annotation.IsDraft = prInfo.IsDraft
		} else {
			annotation.PRAction = "create"
			annotation.IsDraft = opts.Draft
		}

		annotations[branchName] = annotation
	}
	renderer.SetAnnotations(annotations)

	// Render a simple list of branches to submit (in order from bottom to top)
	splog.Info("Stack to submit:")
	stackLines := renderer.RenderBranchList(branches)
	for _, line := range stackLines {
		splog.Info("%s", line)
	}
	splog.Newline()
}
