package actions

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/github"
)

var scopeRegex = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// UpdateStackPRMetadata updates PR titles and body footers for a list of branches
func UpdateStackPRMetadata(ctx context.Context, branches []string, eng engine.Engine, githubClient github.Client, repoOwner, repoName string) {
	var wg sync.WaitGroup
	for _, branchName := range branches {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			branch := eng.GetBranch(name)
			prInfo, err := eng.GetPrInfo(branch)
			if err != nil || prInfo == nil || prInfo.Number() == nil {
				return
			}

			// 1. Update Title with Scope
			scope := eng.GetScopeInternal(name)
			updatedTitle := prInfo.Title()
			if !scope.IsEmpty() {
				// If title already has a scope prefix, replace it
				if scopeRegex.MatchString(updatedTitle) {
					// Only replace if it's NOT already the correct scope
					if !strings.HasPrefix(strings.ToUpper(updatedTitle), "["+strings.ToUpper(scope.String())+"]") {
						updatedTitle = scopeRegex.ReplaceAllString(updatedTitle, "["+scope.String()+"] ")
					}
				} else {
					// No scope prefix, add it
					updatedTitle = fmt.Sprintf("[%s] %s", scope.String(), updatedTitle)
				}
			}

			// 2. Update Body Footer
			footer := CreatePRBodyFooter(name, eng)
			updatedBody := UpdatePRBodyFooter(prInfo.Body(), footer)

			// 3. Apply changes if any
			if updatedTitle != prInfo.Title() || updatedBody != prInfo.Body() {
				updateOpts := github.UpdatePROptions{}
				if updatedTitle != prInfo.Title() {
					updateOpts.Title = &updatedTitle
				}
				if updatedBody != prInfo.Body() {
					updateOpts.Body = &updatedBody
				}

				if err := githubClient.UpdatePullRequest(ctx, repoOwner, repoName, *prInfo.Number(), updateOpts); err != nil {
					return
				}

				// Update engine's cache
				_ = eng.UpsertPrInfo(branch, prInfo.WithTitleAndBody(updatedTitle, updatedBody))
			}
		}(branchName)
	}
	wg.Wait()
}
