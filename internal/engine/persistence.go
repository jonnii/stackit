package engine

// Meta represents branch metadata stored in Git refs
type Meta struct {
	ParentBranchName     *string            `json:"parentBranchName,omitempty"`
	ParentBranchRevision *string            `json:"parentBranchRevision,omitempty"`
	PrInfo               *PrInfoPersistence `json:"prInfo,omitempty"`
	Scope                *string            `json:"scope,omitempty"`
}

// PrInfoPersistence represents PR information for persistence
type PrInfoPersistence struct {
	Number  *int    `json:"number,omitempty"`
	Base    *string `json:"base,omitempty"`
	URL     *string `json:"url,omitempty"`
	Title   *string `json:"title,omitempty"`
	Body    *string `json:"body,omitempty"`
	State   *string `json:"state,omitempty"`
	IsDraft *bool   `json:"isDraft,omitempty"`
}
