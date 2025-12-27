package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// GetPrInfo returns PR information for a branch
func (e *engineImpl) GetPrInfo(branch Branch) (*PrInfo, error) {
	meta, err := e.readMetadataRef(branch.GetName())
	if err != nil {
		return nil, err
	}

	if meta.PrInfo == nil {
		return nil, nil
	}

	prInfo := NewPrInfo(
		meta.PrInfo.Number,
		getStringValue(meta.PrInfo.Title),
		getStringValue(meta.PrInfo.Body),
		getStringValue(meta.PrInfo.State),
		getStringValue(meta.PrInfo.Base),
		getStringValue(meta.PrInfo.URL),
		getBoolValue(meta.PrInfo.IsDraft),
	)

	return prInfo, nil
}

// UpsertPrInfo updates or creates PR information for a branch
func (e *engineImpl) UpsertPrInfo(branch Branch, prInfo *PrInfo) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	meta, err := e.readMetadataRef(branch.GetName())
	if err != nil {
		meta = &Meta{}
	}

	if prInfo == nil {
		meta.PrInfo = nil
		return e.writeMetadataRef(branch.GetName(), meta)
	}

	if meta.PrInfo == nil {
		meta.PrInfo = &PrInfoPersistence{}
	}

	// Update PR info fields
	if prInfo.Number() != nil {
		meta.PrInfo.Number = prInfo.Number()
	}
	if prInfo.Title() != "" {
		title := prInfo.Title()
		meta.PrInfo.Title = &title
	}
	if prInfo.Body() != "" {
		body := prInfo.Body()
		meta.PrInfo.Body = &body
	}
	isDraft := prInfo.IsDraft()
	meta.PrInfo.IsDraft = &isDraft
	if prInfo.State() != "" {
		state := prInfo.State()
		meta.PrInfo.State = &state
	}
	if prInfo.Base() != "" {
		base := prInfo.Base()
		meta.PrInfo.Base = &base
	}
	if prInfo.URL() != "" {
		url := prInfo.URL()
		meta.PrInfo.URL = &url
	}

	return e.writeMetadataRef(branch.GetName(), meta)
}

// GetPRSubmissionStatus returns the submission status of a branch
func (e *engineImpl) GetPRSubmissionStatus(branch Branch) (PRSubmissionStatus, error) {
	prInfo, err := e.GetPrInfo(branch)
	if err != nil {
		return PRSubmissionStatus{}, err
	}

	parentBranch := e.GetParent(branch)
	parentBranchName := ""
	if parentBranch == nil {
		parentBranchName = e.trunk
	} else {
		parentBranchName = parentBranch.GetName()
	}

	if prInfo == nil || prInfo.Number() == nil {
		return PRSubmissionStatus{
			Action:      "create",
			NeedsUpdate: true,
			PRInfo:      prInfo,
		}, nil
	}

	// It's an update
	baseChanged := prInfo.Base() != parentBranchName
	branchChanged, _ := e.BranchMatchesRemote(branch.GetName())

	// Check if PR title needs update due to scope changes
	titleNeedsUpdate := e.prTitleNeedsUpdate(branch, prInfo)

	needsUpdate := baseChanged || !branchChanged || titleNeedsUpdate

	reason := ""
	if !needsUpdate {
		reason = "no changes"
	}

	return PRSubmissionStatus{
		Action:      "update",
		NeedsUpdate: needsUpdate,
		Reason:      reason,
		PRNumber:    prInfo.Number(),
		PRInfo:      prInfo,
	}, nil
}

var scopeRegex = regexp.MustCompile(`^\[[^\]]+\]\s*`)

// prTitleNeedsUpdate checks if the PR title needs to be updated due to scope changes
func (e *engineImpl) prTitleNeedsUpdate(branch Branch, prInfo *PrInfo) bool {
	if prInfo == nil || prInfo.Title() == "" {
		return false
	}

	scope := e.GetScopeInternal(branch.GetName())
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

	return updatedTitle != prInfo.Title()
}
