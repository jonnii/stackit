package actions

import (
	"encoding/json"
	"fmt"
	"time"

	"stackit.dev/stackit/internal/config"
	"stackit.dev/stackit/internal/engine"
	"stackit.dev/stackit/internal/git"
	"stackit.dev/stackit/internal/runtime"
)

// DebugOptions contains options for the debug command
type DebugOptions struct {
	Limit int // Limit number of recent commands to show (0 = all)
}

// DebugInfo represents the complete debugging information
type DebugInfo struct {
	Timestamp         time.Time              `json:"timestamp"`
	RecentCommands    []CommandSnapshot      `json:"recent_commands"`
	StackState        StackStateInfo         `json:"stack_state"`
	ContinuationState *ContinuationStateInfo `json:"continuation_state,omitempty"`
	RepositoryInfo    RepositoryInfo         `json:"repository_info"`
}

// CommandSnapshot represents a single command from the undo history
type CommandSnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	Command       string    `json:"command"`
	Args          []string  `json:"args"`
	CurrentBranch string    `json:"current_branch"`
}

// StackStateInfo represents the complete stack state
type StackStateInfo struct {
	Trunk         string       `json:"trunk"`
	CurrentBranch string       `json:"current_branch"`
	Branches      []BranchInfo `json:"branches"`
}

// BranchInfo represents detailed information about a branch
type BranchInfo struct {
	Name           string   `json:"name"`
	SHA            string   `json:"sha,omitempty"`
	Parent         string   `json:"parent,omitempty"`
	ParentRevision string   `json:"parent_revision,omitempty"`
	Children       []string `json:"children,omitempty"`
	IsTracked      bool     `json:"is_tracked"`
	IsFixed        bool     `json:"is_fixed"`
	IsTrunk        bool     `json:"is_trunk"`
	PRInfo         *PRInfo  `json:"pr_info,omitempty"`
	MetadataRefSHA string   `json:"metadata_ref_sha,omitempty"`
}

// PRInfo represents PR information for a branch
type PRInfo struct {
	Number  *int   `json:"number,omitempty"`
	Base    string `json:"base,omitempty"`
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Body    string `json:"body,omitempty"`
	State   string `json:"state,omitempty"`
	IsDraft bool   `json:"is_draft"`
}

// ContinuationStateInfo represents continuation state
type ContinuationStateInfo struct {
	BranchesToRestack     []string `json:"branches_to_restack,omitempty"`
	BranchesToSync        []string `json:"branches_to_sync,omitempty"`
	CurrentBranchOverride string   `json:"current_branch_override,omitempty"`
	RebasedBranchBase     string   `json:"rebased_branch_base,omitempty"`
}

// RepositoryInfo represents basic repository information
type RepositoryInfo struct {
	RemoteURL string `json:"remote_url,omitempty"`
	RepoRoot  string `json:"repo_root,omitempty"`
}

// DebugAction collects and outputs debugging information
func DebugAction(ctx *runtime.Context, opts DebugOptions) error {
	eng := ctx.Engine
	repoRoot := ctx.RepoRoot

	// Collect recent commands
	snapshotInfos, err := eng.GetSnapshots()
	if err != nil {
		// Log but continue - snapshots might not exist
		snapshotInfos = []engine.SnapshotInfo{}
	}

	// Apply limit if specified
	limit := opts.Limit
	if limit > 0 && limit < len(snapshotInfos) {
		snapshotInfos = snapshotInfos[:limit]
	}

	// Load full snapshots to get current branch for each command
	recentCommands := make([]CommandSnapshot, 0, len(snapshotInfos))
	for _, snapshotInfo := range snapshotInfos {
		// Load full snapshot to get current branch
		fullSnapshot, err := eng.LoadSnapshot(snapshotInfo.ID)
		currentBranch := ""
		if err == nil && fullSnapshot != nil {
			currentBranch = fullSnapshot.CurrentBranch
		}

		recentCommands = append(recentCommands, CommandSnapshot{
			Timestamp:     snapshotInfo.Timestamp,
			Command:       snapshotInfo.Command,
			Args:          snapshotInfo.Args,
			CurrentBranch: currentBranch,
		})
	}

	// Collect stack state
	trunk := eng.Trunk()
	currentBranch := eng.CurrentBranch()
	allBranches := eng.AllBranchNames()

	// Get all metadata refs
	metadataRefs, err := git.GetMetadataRefList()
	if err != nil {
		metadataRefs = make(map[string]string)
	}

	// Build branch info for each branch
	branchInfos := make([]BranchInfo, 0, len(allBranches))
	for _, branchName := range allBranches {
		branch := eng.GetBranch(branchName)
		branchInfo := BranchInfo{
			Name:      branchName,
			IsTrunk:   branch.IsTrunk(),
			IsTracked: branch.IsTracked(),
		}

		// Get SHA
		sha, err := eng.GetRevision(branchName)
		if err == nil {
			branchInfo.SHA = sha
		}

		// Get parent
		parent := eng.GetParent(branchName)
		if parent != "" {
			branchInfo.Parent = parent
		}

		// Get children
		children := eng.GetChildren(branchName)
		if len(children) > 0 {
			branchInfo.Children = children
		}

		// Get metadata
		meta, err := git.ReadMetadataRef(branchName)
		if err == nil && meta != nil {
			if meta.ParentBranchRevision != nil {
				branchInfo.ParentRevision = *meta.ParentBranchRevision
			}

			// Get PR info
			prInfo, err := eng.GetPrInfo(branchName)
			if err == nil && prInfo != nil {
				branchInfo.PRInfo = &PRInfo{
					Number:  prInfo.Number,
					Base:    prInfo.Base,
					URL:     prInfo.URL,
					Title:   prInfo.Title,
					Body:    prInfo.Body,
					State:   prInfo.State,
					IsDraft: prInfo.IsDraft,
				}
			}
		}

		// Get metadata ref SHA
		if metadataSHA, ok := metadataRefs[branchName]; ok {
			branchInfo.MetadataRefSHA = metadataSHA
		}

		// Check if branch is up to date with its parent
		if !branchInfo.IsTrunk {
			branchInfo.IsFixed = eng.IsBranchUpToDate(branchName)
		} else {
			branchInfo.IsFixed = true // Trunk is always up to date
		}

		branchInfos = append(branchInfos, branchInfo)
	}

	// Collect continuation state
	var continuationState *ContinuationStateInfo
	contState, err := config.GetContinuationState(repoRoot)
	if err == nil && contState != nil {
		continuationState = &ContinuationStateInfo{
			BranchesToRestack:     contState.BranchesToRestack,
			BranchesToSync:        contState.BranchesToSync,
			CurrentBranchOverride: contState.CurrentBranchOverride,
			RebasedBranchBase:     contState.RebasedBranchBase,
		}
	}

	// Collect repository info
	repoInfo := RepositoryInfo{
		RepoRoot: repoRoot,
	}
	remoteURL, err := git.RunGitCommandWithContext(ctx.Context, "config", "--get", "remote.origin.url")
	if err == nil {
		repoInfo.RemoteURL = remoteURL
	}

	// Build debug info
	debugInfo := DebugInfo{
		Timestamp:      time.Now(),
		RecentCommands: recentCommands,
		StackState: StackStateInfo{
			Trunk:         trunk,
			CurrentBranch: currentBranch,
			Branches:      branchInfos,
		},
		ContinuationState: continuationState,
		RepositoryInfo:    repoInfo,
	}

	// Output as pretty-printed JSON
	jsonData, err := json.MarshalIndent(debugInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal debug info: %w", err)
	}

	ctx.Splog.Page(string(jsonData))
	ctx.Splog.Newline()

	return nil
}
