package actions

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	stackitcontext "stackit.dev/stackit/internal/context"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/output"
)

// SubmitOptions contains options for the submit command
type SubmitOptions struct {
	Branch              string
	Stack               bool
	Force               bool
	DryRun              bool
	Confirm             bool
	UpdateOnly          bool
	Always              bool
	Restack             bool
	Draft               bool
	Publish             bool
	Edit                bool
	EditTitle           bool
	EditDescription     bool
	NoEdit              bool
	NoEditTitle         bool
	NoEditDescription   bool
	Reviewers           string
	TeamReviewers       string
	MergeWhenReady      bool
	RerequestReview     bool
	View                bool
	Web                 bool
	Comment             string
	TargetTrunk         string
	IgnoreOutOfSyncTrunk bool
	Engine              engine.Engine
	Splog               *output.Splog
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

// SubmitAction performs the submit operation
func SubmitAction(opts SubmitOptions) error {
	eng := opts.Engine
	splog := opts.Splog

	// Validate flags
	if opts.Draft && opts.Publish {
		return fmt.Errorf("can't use both --publish and --draft flags in one command")
	}

	// Get branch scope
	branchName := opts.Branch
	if branchName == "" {
		branchName = eng.CurrentBranch()
	}
	if branchName == "" {
		return fmt.Errorf("not on a branch and no branch specified")
	}

	// Get stack of branches to submit
	scope := engine.Scope{RecursiveParents: true}
	var branches []string
	if opts.Stack {
		// Include descendants
		allBranches := eng.GetRelativeStack(branchName, scope)
		// Add the current branch itself
		allBranches = append(allBranches, branchName)
		// Also get descendants
		children := eng.GetChildren(branchName)
		for _, child := range children {
			descendants := eng.GetRelativeStack(child, engine.Scope{RecursiveParents: false})
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

	if len(branches) == 0 {
		splog.Info("No branches to submit.")
		return nil
	}

	// Restack if requested
	if opts.Restack {
		splog.Info("Restacking branches before submitting...")
		if err := RestackBranches(branches, eng, splog); err != nil {
			return fmt.Errorf("failed to restack branches: %w", err)
		}
	}

	// Validate branches
	splog.Info("Validating that this stack is ready to submit...")
	ctx := stackitcontext.NewContext(eng)
	ctx.Splog = splog
	if err := ValidateBranchesToSubmit(branches, eng, ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Populate remote SHAs
	if err := eng.PopulateRemoteShas(); err != nil {
		splog.Debug("Failed to populate remote SHAs: %v", err)
	}

	// Prepare branches for submit
	splog.Info("Preparing to submit PRs for the following branches...")
	submissionInfos, err := prepareBranchesForSubmit(branches, opts, eng, ctx)
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

	// Push branches and create/update PRs
	splog.Info("Pushing to remote and creating/updating PRs...")
	githubCtx := context.Background()
	githubClient, repoOwner, repoName, err := git.GetGitHubClient(githubCtx)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	remote := "origin" // TODO: Get from config
	for _, submissionInfo := range submissionInfos {
		// Push branch
		forceWithLease := !opts.Force
		if err := git.PushBranch(submissionInfo.BranchName, remote, opts.Force, forceWithLease); err != nil {
			if strings.Contains(err.Error(), "stale info") {
				return fmt.Errorf("force-with-lease push of %s failed due to external changes to the remote branch. If you are collaborating on this stack, try 'stackit sync' to pull in changes. Alternatively, use the --force option to bypass the stale info warning", submissionInfo.BranchName)
			}
			return fmt.Errorf("failed to push branch %s: %w", submissionInfo.BranchName, err)
		}

		// Create or update PR
		if submissionInfo.Action == "create" {
			createOpts := git.CreatePROptions{
				Title:         submissionInfo.Metadata.Title,
				Body:          submissionInfo.Metadata.Body,
				Head:          submissionInfo.Head,
				Base:          submissionInfo.Base,
				Draft:         submissionInfo.Metadata.IsDraft,
				Reviewers:     submissionInfo.Metadata.Reviewers,
				TeamReviewers: submissionInfo.Metadata.TeamReviewers,
			}
			pr, err := git.CreatePullRequest(githubCtx, githubClient, repoOwner, repoName, createOpts)
			if err != nil {
				return fmt.Errorf("failed to create PR for %s: %w", submissionInfo.BranchName, err)
			}

			// Update PR info
			prNumber := pr.Number
			prURL := ""
			if pr.HTMLURL != nil {
				prURL = *pr.HTMLURL
			}
			eng.UpsertPrInfo(submissionInfo.BranchName, &engine.PrInfo{
				Number:  prNumber,
				Title:   submissionInfo.Metadata.Title,
				Body:    submissionInfo.Metadata.Body,
				IsDraft: submissionInfo.Metadata.IsDraft,
				State:   "OPEN",
				Base:    submissionInfo.Base,
				URL:     prURL,
			})

			splog.Info("%s: %s (%s)", 
				output.ColorBranchName(submissionInfo.BranchName, true),
				prURL,
				output.ColorDim("created"))

			// Open in browser if requested
			if opts.View {
				openBrowser(prURL)
			}
		} else {
			// Update PR - check if base changed
			prInfo, _ := eng.GetPrInfo(submissionInfo.BranchName)
			baseChanged := false
			if prInfo != nil && prInfo.Base != submissionInfo.Base {
				baseChanged = true
			}

			updateOpts := git.UpdatePROptions{
				Title:         &submissionInfo.Metadata.Title,
				Body:          &submissionInfo.Metadata.Body,
				Draft:         &submissionInfo.Metadata.IsDraft,
				Reviewers:     submissionInfo.Metadata.Reviewers,
				TeamReviewers: submissionInfo.Metadata.TeamReviewers,
				MergeWhenReady: &opts.MergeWhenReady,
				RerequestReview: opts.RerequestReview,
			}
			if baseChanged {
				updateOpts.Base = &submissionInfo.Base
			}
			if err := git.UpdatePullRequest(githubCtx, githubClient, repoOwner, repoName, *submissionInfo.PRNumber, updateOpts); err != nil {
				return fmt.Errorf("failed to update PR for %s: %w", submissionInfo.BranchName, err)
			}

			// Update PR info - prInfo already declared above
			prInfo, _ = eng.GetPrInfo(submissionInfo.BranchName)
			var prURL string
			if prInfo != nil {
				prURL = prInfo.URL
			}
			if prURL == "" {
				// Get from GitHub
				pr, err := git.GetPullRequestByBranch(githubCtx, githubClient, repoOwner, repoName, submissionInfo.BranchName)
				if err == nil && pr != nil && pr.HTMLURL != nil {
					prURL = *pr.HTMLURL
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

			// Get PR URL
			prInfo, _ = eng.GetPrInfo(submissionInfo.BranchName)
			if prInfo != nil && prInfo.URL != "" {
				prURL = prInfo.URL
			} else {
				// Get from GitHub
				pr, err := git.GetPullRequestByBranch(githubCtx, githubClient, repoOwner, repoName, submissionInfo.BranchName)
				if err == nil && pr != nil && pr.HTMLURL != nil {
					prURL = *pr.HTMLURL
				}
			}

			splog.Info("%s: %s (%s)", 
				output.ColorBranchName(submissionInfo.BranchName, true),
				prURL,
				output.ColorDim("updated"))

			// Open in browser if requested
			if opts.View && prURL != "" {
				openBrowser(prURL)
			}
		}
	}

	// Update PR body footers
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
			if err := git.UpdatePullRequest(githubCtx, githubClient, repoOwner, repoName, *prInfo.Number, updateOpts); err != nil {
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

// prepareBranchesForSubmit prepares submission info for each branch
func prepareBranchesForSubmit(branches []string, opts SubmitOptions, eng engine.Engine, ctx *stackitcontext.Context) ([]SubmissionInfo, error) {
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

		// Check if we should skip
		if opts.UpdateOnly && action == "create" {
			ctx.Splog.Info("▸ %s (Skipped - no existing PR)", output.ColorDim(branchName))
			continue
		}

		// Check if branch needs update
		if action == "update" {
			baseChanged := prInfo.Base != parentBranchName
			branchChanged, _ := eng.BranchMatchesRemote(branchName)
			needsUpdate := baseChanged || !branchChanged || opts.Edit || opts.Always

			if !needsUpdate && !opts.Draft && !opts.Publish {
				ctx.Splog.Info("▸ %s (No-op)", output.ColorDim(branchName))
				continue
			}
		}

		// Prepare metadata
		metadataOpts := SubmitMetadataOptions{
			Edit:            opts.Edit && !opts.NoEdit,
			EditTitle:       opts.EditTitle && !opts.NoEditTitle,
			EditDescription: opts.EditDescription && !opts.NoEditDescription,
			NoEdit:          opts.NoEdit,
			NoEditTitle:     opts.NoEditTitle,
			NoEditDescription: opts.NoEditDescription,
			Draft:           opts.Draft,
			Publish:         opts.Publish,
			Reviewers:       opts.Reviewers,
			ReviewersPrompt: opts.Reviewers == "" && (opts.Edit || opts.Reviewers == ""),
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

		// Log action
		status := action
		if action == "update" {
			status = "Update"
		} else {
			status = "Create"
		}
		ctx.Splog.Info("▸ %s (%s)", 
			output.ColorBranchName(branchName, true),
			output.ColorDim(status))

		submissionInfos = append(submissionInfos, submissionInfo)
	}

	return submissionInfos, nil
}

// shouldAbortSubmit checks if we should abort the submit operation
func shouldAbortSubmit(opts SubmitOptions, hasAnyPRs bool, ctx *stackitcontext.Context) (bool, error) {
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

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	cmd := exec.Command("open", url) // macOS
	if err := cmd.Run(); err != nil {
		// Try other commands for different OS
		exec.Command("xdg-open", url).Run() // Linux
		exec.Command("start", url).Run()    // Windows
	}
}
