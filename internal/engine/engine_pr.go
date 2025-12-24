package engine

import (
	"fmt"
	"regexp"
	"strings"

	"stackit.dev/stackit/internal/git"
)

// GetPrInfo returns PR information for a branch
func (e *engineImpl) GetPrInfo(branchName string) (*PrInfo, error) {
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		return nil, err
	}

	if meta.PrInfo == nil {
		return nil, nil
	}

	prInfo := &PrInfo{
		Number:  meta.PrInfo.Number,
		Title:   getStringValue(meta.PrInfo.Title),
		Body:    getStringValue(meta.PrInfo.Body),
		IsDraft: getBoolValue(meta.PrInfo.IsDraft),
		State:   getStringValue(meta.PrInfo.State),
		Base:    getStringValue(meta.PrInfo.Base),
		URL:     getStringValue(meta.PrInfo.URL),
	}

	return prInfo, nil
}

// UpsertPrInfo updates or creates PR information for a branch
func (e *engineImpl) UpsertPrInfo(branchName string, prInfo *PrInfo) error {
	meta, err := git.ReadMetadataRef(branchName)
	if err != nil {
		meta = &git.Meta{}
	}

	if meta.PrInfo == nil {
		meta.PrInfo = &git.PrInfo{}
	}

	// Update PR info fields
	if prInfo.Number != nil {
		meta.PrInfo.Number = prInfo.Number
	}
	if prInfo.Title != "" {
		meta.PrInfo.Title = &prInfo.Title
	}
	if prInfo.Body != "" {
		meta.PrInfo.Body = &prInfo.Body
	}
	meta.PrInfo.IsDraft = &prInfo.IsDraft
	if prInfo.State != "" {
		meta.PrInfo.State = &prInfo.State
	}
	if prInfo.Base != "" {
		meta.PrInfo.Base = &prInfo.Base
	}
	if prInfo.URL != "" {
		meta.PrInfo.URL = &prInfo.URL
	}

	return git.WriteMetadataRef(branchName, meta)
}

// GetPRSubmissionStatus returns the submission status of a branch
func (e *engineImpl) GetPRSubmissionStatus(branchName string) (PRSubmissionStatus, error) {
	prInfo, err := e.GetPrInfo(branchName)
	if err != nil {
		return PRSubmissionStatus{}, err
	}

	branch := e.GetBranch(branchName)
	parentBranch := e.GetParent(branch)
	parentBranchName := ""
	if parentBranch == nil {
		parentBranchName = e.trunk
	} else {
		parentBranchName = parentBranch.Name
	}

	if prInfo == nil || prInfo.Number == nil {
		return PRSubmissionStatus{
			Action:      "create",
			NeedsUpdate: true,
			PRInfo:      prInfo,
		}, nil
	}

	// It's an update
	baseChanged := prInfo.Base != parentBranchName
	branchChanged, _ := e.BranchMatchesRemote(branchName)

	// Check if PR title needs update due to scope changes
	titleNeedsUpdate := e.prTitleNeedsUpdate(branchName, prInfo)

	needsUpdate := baseChanged || !branchChanged || titleNeedsUpdate

	reason := ""
	if !needsUpdate {
		reason = "no changes"
	}

	return PRSubmissionStatus{
		Action:      "update",
		NeedsUpdate: needsUpdate,
		Reason:      reason,
		PRNumber:    prInfo.Number,
		PRInfo:      prInfo,
	}, nil
}

var scopeRegex = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// prTitleNeedsUpdate checks if the PR title needs to be updated due to scope changes
func (e *engineImpl) prTitleNeedsUpdate(branchName string, prInfo *PrInfo) bool {
	if prInfo == nil || prInfo.Title == "" {
		return false
	}

	scope := e.GetScopeInternal(branchName)
	updatedTitle := prInfo.Title

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

	return updatedTitle != prInfo.Title
}
