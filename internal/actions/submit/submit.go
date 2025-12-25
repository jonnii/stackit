// Package submit provides functionality for submitting stacked branches as pull requests.
package submit

import (
	"context"
	"errors"
	"fmt"

	"sync"

	"stackit.dev/stackit/internal/actions"
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
	SubmitFooter         bool // Whether to include PR footer (from config)
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
	defer ui.Complete()

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
	renderer := getStackTreeRenderer(branches, opts, eng)
	ui.ShowStack(renderer, eng.Trunk().Name)

	// Restack if requested
	if opts.Restack {
		ui.ShowRestackStart()
		// Convert []string to []engine.Branch for RestackBranches
		branchObjects := make([]engine.Branch, len(branches))
		for i, branchName := range branches {
			branchObjects[i] = eng.GetBranch(branchName)
		}
		if err := actions.RestackBranches(context, branchObjects, eng, splog); err != nil {
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
	submissionInfos, err := prepareBranchesForSubmit(branches, opts, eng, ctx, currentBranch.Name, ui)
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

	githubClient, err := getGitHubClient(ctx)
	if err != nil {
		return err
	}
	repoOwner, repoName := githubClient.GetOwnerRepo()

	remote := git.GetRemote()
	var wg sync.WaitGroup
	var submitErr error
	var errMu sync.Mutex

	for _, submissionInfo := range submissionInfos {
		wg.Add(1)
		go func(info Info) {
			defer wg.Done()

			ui.UpdateSubmitItem(info.BranchName, "submitting", "", nil)

			if err := pushBranchIfNeeded(context, info, opts, remote, eng); err != nil {
				ui.UpdateSubmitItem(info.BranchName, "error", "", err)
				errMu.Lock()
				if submitErr == nil {
					submitErr = err
				}
				errMu.Unlock()
				return
			}

			var prURL string
			const (
				actionCreate = "create"
				actionUpdate = "update"
			)
			var err error
			if info.Action == actionCreate {
				prURL, err = createPullRequestQuiet(context, info, eng, githubClient, repoOwner, repoName)
			} else {
				prURL, err = updatePullRequestQuiet(context, info, opts, eng, githubClient, repoOwner, repoName)
			}

			if err != nil {
				ui.UpdateSubmitItem(info.BranchName, "error", "", err)
				errMu.Lock()
				if submitErr == nil {
					submitErr = err
				}
				errMu.Unlock()
				return
			}

			ui.UpdateSubmitItem(info.BranchName, "done", prURL, nil)

			// Open in browser if requested
			if opts.View && prURL != "" {
				if err := utils.OpenBrowser(prURL); err != nil {
					splog.Debug("Failed to open browser: %v", err)
				}
			}
		}(submissionInfo)
	}
	wg.Wait()

	if submitErr != nil {
		return submitErr
	}

	// Update PR body footers silently
	if opts.SubmitFooter {
		actions.UpdateStackPRMetadata(context, branches, eng, githubClient, repoOwner, repoName)
	}

	return nil
}

// prepareBranchesForSubmit prepares submission info for each branch, outputting via UI
func prepareBranchesForSubmit(branches []string, opts Options, eng engine.Engine, runtimeCtx *runtime.Context, currentBranch string, ui tui.SubmitUI) ([]Info, error) {
	submissionInfos := make([]Info, 0, len(branches))

	for _, branchName := range branches {
		status, err := eng.GetPRSubmissionStatus(branchName)
		if err != nil {
			return nil, err
		}

		action := status.Action
		prNumber := status.PRNumber
		prInfo := status.PRInfo

		isCurrent := branchName == currentBranch

		// Check if we should skip
		if opts.UpdateOnly && action == "create" {
			ui.ShowBranchPlan(branchName, action, isCurrent, true, "skipped, no existing PR")
			continue
		}

		needsUpdate := status.NeedsUpdate
		if action == "update" {
			// Check if draft status needs to change
			draftStatusNeedsChange := false
			if prInfo != nil {
				if opts.Draft && !prInfo.IsDraft {
					draftStatusNeedsChange = true
				} else if opts.Publish && prInfo.IsDraft {
					draftStatusNeedsChange = true
				}
			}

			needsUpdate = needsUpdate || opts.Edit || opts.Always || draftStatusNeedsChange

			if !needsUpdate && !opts.Draft && !opts.Publish {
				ui.ShowBranchPlan(branchName, action, isCurrent, true, status.Reason)
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

		metadata, err := PreparePRMetadata(branchName, metadataOpts, eng, runtimeCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare metadata for %s: %w", branchName, err)
		}

		// Get SHAs
		branch := eng.GetBranch(branchName)
		headSHA, _ := branch.GetRevision()
		parentBranchName := branch.GetParentPrecondition()
		parentBranch := eng.GetBranch(parentBranchName)
		baseSHA, _ := parentBranch.GetRevision()

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
		currentBranch := eng.CurrentBranch()
		if currentBranch == nil {
			return nil, fmt.Errorf("not on a branch and no branch specified")
		}
		branchName = currentBranch.Name
	}

	var allBranches []string
	if opts.Stack {
		// Include descendants and ancestors
		branch := eng.GetBranch(branchName)
		stackBranches := eng.GetFullStack(branch)
		allBranches = make([]string, len(stackBranches))
		for i, b := range stackBranches {
			allBranches[i] = b.Name
		}
	} else {
		// Just ancestors (including current branch)
		branch := eng.GetBranch(branchName)
		downstackBranches := eng.GetRelativeStackDownstack(branch)
		allBranches = make([]string, len(downstackBranches)+1)
		for i, b := range downstackBranches {
			allBranches[i] = b.Name
		}
		allBranches[len(downstackBranches)] = branchName
	}

	// Remove duplicates and trunk
	branches := []string{}
	branchSet := make(map[string]bool)
	for _, b := range allBranches {
		branchObj := eng.GetBranch(b)
		if !branchObj.IsTrunk() && !branchSet[b] {
			branches = append(branches, b)
			branchSet[b] = true
		}
	}

	return branches, nil
}

// getGitHubClient returns the GitHub client from context
func getGitHubClient(ctx *runtime.Context) (github.Client, error) {
	if ctx.GitHubClient != nil {
		return ctx.GitHubClient, nil
	}
	return nil, fmt.Errorf("no GitHub client available - check your GITHUB_TOKEN")
}

// pushBranchIfNeeded pushes a branch to remote if needed
func pushBranchIfNeeded(ctx context.Context, submissionInfo Info, opts Options, remote string, eng engine.SyncManager) error {
	// Skip if dry run
	if opts.DryRun {
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
func createPullRequestQuiet(ctx context.Context, submissionInfo Info, eng engine.PRManager, githubClient github.Client, repoOwner, repoName string) (string, error) {
	createOpts := github.CreatePROptions{
		Title:         submissionInfo.Metadata.Title,
		Body:          submissionInfo.Metadata.Body,
		Head:          submissionInfo.Head,
		Base:          submissionInfo.Base,
		Draft:         submissionInfo.Metadata.IsDraft,
		Reviewers:     submissionInfo.Metadata.Reviewers,
		TeamReviewers: submissionInfo.Metadata.TeamReviewers,
	}
	pr, err := githubClient.CreatePullRequest(ctx, repoOwner, repoName, createOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Update PR info
	prNumber := pr.Number
	prURL := pr.HTMLURL
	_ = eng.UpsertPrInfo(submissionInfo.BranchName, &engine.PrInfo{
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
func updatePullRequestQuiet(ctx context.Context, submissionInfo Info, opts Options, eng engine.Engine, githubClient github.Client, repoOwner, repoName string) (string, error) {
	// Check if base changed
	prInfo, _ := eng.GetPrInfo(submissionInfo.BranchName)
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

	// Before updating the base, check if there are commits between the new base and head
	// GitHub will reject the update if there are no commits between them
	baseToStore := submissionInfo.Base
	if baseChanged {
		baseUpdated := false
		// Only update base if there are commits between base and head
		if submissionInfo.BaseSHA != submissionInfo.HeadSHA {
			// Check if there are actually commits between base and head
			commits, err := git.GetCommitRangeSHAs(submissionInfo.BaseSHA, submissionInfo.HeadSHA)
			if err == nil && len(commits) > 0 {
				// There are commits, safe to update base
				updateOpts.Base = &submissionInfo.Base
				baseUpdated = true
			}
			// If no commits or error, skip base update to avoid GitHub 422 error
		}
		// If base SHA equals head SHA, skip base update (no commits between them)

		if !baseUpdated && prInfo != nil {
			// If we skipped the update, keep the existing base in our local cache
			// so it reflects what is actually on GitHub.
			baseToStore = prInfo.Base
		}
	}

	if err := githubClient.UpdatePullRequest(ctx, repoOwner, repoName, *submissionInfo.PRNumber, updateOpts); err != nil {
		return "", fmt.Errorf("failed to update PR for %s: %w", submissionInfo.BranchName, err)
	}

	// Get PR URL
	prInfo, _ = eng.GetPrInfo(submissionInfo.BranchName)
	var prURL string
	if prInfo != nil && prInfo.URL != "" {
		prURL = prInfo.URL
	} else {
		// Get from GitHub
		pr, err := githubClient.GetPullRequestByBranch(ctx, repoOwner, repoName, submissionInfo.BranchName)
		if err == nil && pr != nil {
			prURL = pr.HTMLURL
		}
	}

	_ = eng.UpsertPrInfo(submissionInfo.BranchName, &engine.PrInfo{
		Number:  submissionInfo.PRNumber,
		Title:   submissionInfo.Metadata.Title,
		Body:    submissionInfo.Metadata.Body,
		IsDraft: submissionInfo.Metadata.IsDraft,
		State:   "OPEN",
		Base:    baseToStore,
		URL:     prURL,
	})

	return prURL, nil
}

// getStackTreeRenderer returns the stack tree renderer with PR annotations
func getStackTreeRenderer(branches []string, opts Options, eng engine.Engine) *tui.StackTreeRenderer {
	// Create the tree renderer
	currentBranchObj := eng.CurrentBranch()
	trunk := eng.Trunk()
	renderer := tui.NewStackTreeRenderer(
		currentBranchObj.Name,
		trunk.Name,
		func(branchName string) []string {
			branch := eng.GetBranch(branchName)
			children := branch.GetChildren()
			childNames := make([]string, len(children))
			for i, c := range children {
				childNames[i] = c.Name
			}
			return childNames
		},
		func(branchName string) string {
			branch := eng.GetBranch(branchName)
			parent := eng.GetParent(branch)
			if parent == nil {
				return ""
			}
			return parent.Name
		},
		func(branchName string) bool { return eng.GetBranch(branchName).IsTrunk() },
		func(branchName string) bool {
			return eng.GetBranch(branchName).IsBranchUpToDate()
		},
	)

	// Build annotations for each branch
	annotations := make(map[string]tui.BranchAnnotation)
	branchSet := make(map[string]bool)
	for _, b := range branches {
		branchSet[b] = true
	}

	for _, branch := range eng.AllBranches() {
		branchName := branch.Name
		prInfo, _ := eng.GetPrInfo(branchName)
		if prInfo == nil && !branchSet[branchName] {
			continue
		}

		annotation := tui.BranchAnnotation{
			NeedsRestack: !eng.GetBranch(branchName).IsBranchUpToDate(),
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

	return renderer
}
