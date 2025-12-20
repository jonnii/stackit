package submit

import (
	"context"
	"errors"
	"fmt"

	"stackit.dev/stackit/internal/actions"
	"stackit.dev/stackit/internal/ai"
	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/github"
	"stackit.dev/stackit/internal/runtime"
	"stackit.dev/stackit/internal/tui"
	"stackit.dev/stackit/internal/utils"
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
	AI                   bool
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
	context := ctx.Context // Use context from runtime context

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
	if err := eng.PopulateRemoteShas(context); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Display the stack tree with PR annotations
	renderer, rootBranch := getStackTreeRenderer(context, branches, opts, eng, currentBranch)
	ui.ShowStack(renderer, rootBranch)

	// Restack if requested
	if opts.Restack {
		ui.ShowRestackStart()
		repoRoot := ctx.RepoRoot
		if repoRoot == "" {
			repoRoot, _ = git.GetRepoRoot()
		}
		if err := actions.RestackBranches(context, branches, eng, splog, repoRoot); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
		ui.ShowRestackComplete()
	}

	// Validate and prepare branches
	ui.ShowPreparing()

	if err := ValidateBranchesToSubmit(context, branches, eng, ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Prepare branches for submit (show planning phase with current indicator)
	submissionInfos, err := prepareBranchesForSubmit(context, branches, opts, eng, ctx, currentBranch, ui)
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

	// TODO: Add interactive confirmation prompt if opts.Confirm is set

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

	githubCtx := context
	githubClient, err := getGitHubClient(ctx)
	if err != nil {
		return err
	}
	repoOwner, repoName := githubClient.GetOwnerRepo()

	remote := "origin" // TODO: Get from config
	for _, submissionInfo := range submissionInfos {
		ui.UpdateSubmitItem(submissionInfo.BranchName, "submitting", "", nil)

		if err := pushBranchIfNeeded(context, submissionInfo, opts, remote, eng); err != nil {
			ui.UpdateSubmitItem(submissionInfo.BranchName, "error", "", err)
			ui.Complete()
			return err
		}

		var prURL string
		const (
			actionCreate = "create"
			actionUpdate = "update"
		)
		if submissionInfo.Action == actionCreate {
			prURL, err = createPullRequestQuiet(context, submissionInfo, eng, githubCtx, githubClient, repoOwner, repoName)
		} else {
			prURL, err = updatePullRequestQuiet(context, submissionInfo, opts, eng, githubCtx, githubClient, repoOwner, repoName)
		}

		if err != nil {
			ui.UpdateSubmitItem(submissionInfo.BranchName, "error", "", err)
			ui.Complete()
			return err
		}

		ui.UpdateSubmitItem(submissionInfo.BranchName, "done", prURL, nil)

		// Open in browser if requested
		if opts.View && prURL != "" {
			if err := utils.OpenBrowser(prURL); err != nil {
				splog.Debug("Failed to open browser: %v", err)
			}
		}
	}

	// Update PR body footers silently
	updatePRFootersQuiet(context, branches, eng, githubCtx, githubClient, repoOwner, repoName)

	ui.Complete()

	return nil
}

// prepareBranchesForSubmit prepares submission info for each branch, outputting via UI
func prepareBranchesForSubmit(ctx context.Context, branches []string, opts Options, eng engine.Engine, runtimeCtx *runtime.Context, currentBranch string, ui tui.SubmitUI) ([]Info, error) {
	var submissionInfos []Info

	// Determine AI preference: flag takes precedence, then config default
	aiEnabled := opts.AI
	if !aiEnabled {
		// Check config default
		repoRoot := runtimeCtx.RepoRoot
		if repoRoot == "" {
			var err error
			repoRoot, err = git.GetRepoRoot()
			if err == nil {
				configAI, err := config.GetCreateAI(repoRoot)
				if err == nil {
					aiEnabled = configAI
				}
			}
		} else {
			configAI, err := config.GetCreateAI(repoRoot)
			if err == nil {
				aiEnabled = configAI
			}
		}
	}

	// Create AI client if enabled
	var aiClient ai.AIClient
	if aiEnabled {
		aiClient = ai.NewMockClient()
		runtimeCtx.Splog.Debug("AI-powered PR description generation enabled")
	}

	for _, branchName := range branches {
		parentBranchName := eng.GetParentPrecondition(branchName)
		prInfo, _ := eng.GetPrInfo(ctx, branchName)

		// Determine action
		const (
			actionCreate = "create"
			actionUpdate = "update"
		)
		action := actionCreate
		prNumber := (*int)(nil)
		if prInfo != nil && prInfo.Number != nil {
			action = actionUpdate
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
			branchChanged, _ := eng.BranchMatchesRemote(ctx, branchName)

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
			AI:                aiEnabled,
			AIClient:          aiClient,
		}

		metadata, err := PreparePRMetadata(branchName, metadataOpts, eng, runtimeCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare metadata for %s: %w", branchName, err)
		}

		// Get SHAs
		headSHA, _ := eng.GetRevision(ctx, branchName)
		baseSHA, _ := eng.GetRevision(ctx, parentBranchName)

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

	var allBranches []string
	if opts.Stack {
		// Include descendants and ancestors
		allBranches = utils.GetFullStack(eng, branchName)
	} else {
		// Just ancestors (including current branch)
		allBranches = utils.GetDownstack(eng, branchName)
		allBranches = append(allBranches, branchName)
	}

	// Remove duplicates and trunk
	branches := []string{}
	branchSet := make(map[string]bool)
	for _, b := range allBranches {
		if !eng.IsTrunk(b) && !branchSet[b] {
			branches = append(branches, b)
			branchSet[b] = true
		}
	}

	return branches, nil
}

// getGitHubClient returns the GitHub client from context
func getGitHubClient(ctx *runtime.Context) (github.GitHubClient, error) {
	if ctx.GitHubClient != nil {
		return ctx.GitHubClient, nil
	}
	return nil, fmt.Errorf("no GitHub client available - check your GITHUB_TOKEN")
}

// pushBranchIfNeeded pushes a branch to remote if needed
func pushBranchIfNeeded(ctx context.Context, submissionInfo Info, opts Options, remote string, eng engine.SyncManager) error {
	// Skip if dry run or skip push is set
	if opts.DryRun || opts.SkipPush {
		return nil
	}

	forceWithLease := !opts.Force
	if err := eng.PushBranch(ctx, submissionInfo.BranchName, remote, opts.Force, forceWithLease); err != nil {
		if errors.Is(err, git.ErrStaleRemoteInfo) {
			return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning", submissionInfo.BranchName)
		}
		return fmt.Errorf("failed to push branch %s: %w", submissionInfo.BranchName, err)
	}
	return nil
}

// createPullRequestQuiet creates a new pull request without logging
func createPullRequestQuiet(ctx context.Context, submissionInfo Info, eng engine.PRManager, githubCtx context.Context, githubClient github.GitHubClient, repoOwner, repoName string) (string, error) {
	createOpts := github.CreatePROptions{
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
	_ = eng.UpsertPrInfo(ctx, submissionInfo.BranchName, &engine.PrInfo{
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

// updatePullRequestQuiet updates an existing pull request without logging
func updatePullRequestQuiet(ctx context.Context, submissionInfo Info, opts Options, eng engine.Engine, githubCtx context.Context, githubClient github.GitHubClient, repoOwner, repoName string) (string, error) {
	// Check if base changed
	prInfo, _ := eng.GetPrInfo(ctx, submissionInfo.BranchName)
	baseChanged := false
	if prInfo != nil && prInfo.Base != submissionInfo.Base {
		baseChanged = true
	}

	updateOpts := github.UpdatePROptions{
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
	prInfo, _ = eng.GetPrInfo(ctx, submissionInfo.BranchName)
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

	_ = eng.UpsertPrInfo(ctx, submissionInfo.BranchName, &engine.PrInfo{
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

// updatePRFootersQuiet updates PR body footers silently (no logging)
func updatePRFootersQuiet(ctx context.Context, branches []string, eng engine.Engine, githubCtx context.Context, githubClient github.GitHubClient, repoOwner, repoName string) {
	for _, branchName := range branches {
		prInfo, err := eng.GetPrInfo(ctx, branchName)
		if err != nil || prInfo == nil || prInfo.Number == nil {
			continue
		}

		footer := actions.CreatePRBodyFooter(ctx, branchName, eng)
		updatedBody := actions.UpdatePRBodyFooter(prInfo.Body, footer)

		if updatedBody != prInfo.Body {
			updateOpts := github.UpdatePROptions{
				Body: &updatedBody,
			}
			if err := githubClient.UpdatePullRequest(githubCtx, repoOwner, repoName, *prInfo.Number, updateOpts); err != nil {
				continue
			}
		}
	}
}

// getStackTreeRenderer returns the stack tree renderer with PR annotations
func getStackTreeRenderer(ctx context.Context, branches []string, opts Options, eng engine.Engine, currentBranch string) (*tui.StackTreeRenderer, string) {
	// Create the tree renderer
	renderer := tui.NewStackTreeRenderer(
		currentBranch,
		eng.Trunk(),
		eng.GetChildren,
		eng.GetParent,
		eng.IsTrunk,
		func(branchName string) bool {
			return eng.IsBranchFixed(ctx, branchName)
		},
	)

	// Build annotations for each branch
	annotations := make(map[string]tui.BranchAnnotation)
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	for _, branchName := range eng.AllBranchNames() {
		prInfo, _ := eng.GetPrInfo(ctx, branchName)
		if prInfo == nil && !branchSet[branchName] {
			continue
		}

		annotation := tui.BranchAnnotation{
			NeedsRestack: !eng.IsBranchFixed(ctx, branchName),
		}

		const actionUpdate = "update"
		const actionCreate = "create"

		if prInfo != nil && prInfo.Number != nil {
			annotation.PRNumber = prInfo.Number
			if branchSet[branchName] {
				annotation.PRAction = actionUpdate
			}
			annotation.IsDraft = prInfo.IsDraft
		} else if branchSet[branchName] {
			annotation.PRAction = actionCreate
			annotation.IsDraft = opts.Draft
		}

		annotations[branchName] = annotation
	}
	renderer.SetAnnotations(annotations)

	return renderer, eng.Trunk()
}
