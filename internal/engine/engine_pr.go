package engine

import (
	"context"

	"stackit.dev/stackit/internal/git"
)

// GetPrInfo returns PR information for a branch
func (e *engineImpl) GetPrInfo(_ context.Context, branchName string) (*PrInfo, error) {
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
func (e *engineImpl) UpsertPrInfo(_ context.Context, branchName string, prInfo *PrInfo) error {
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
