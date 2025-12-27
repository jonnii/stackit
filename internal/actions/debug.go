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
	Name           string         `json:"name"`
	SHA            string         `json:"sha,omitempty"`
	Parent         string         `json:"parent,omitempty"`
	ParentRevision string         `json:"parent_revision,omitempty"`
	Children       []string       `json:"children,omitempty"`
	IsTracked      bool           `json:"is_tracked"`
	IsFixed        bool           `json:"is_fixed"`
	IsTrunk        bool           `json:"is_trunk"`
	PRInfo         *engine.PrInfo `json:"pr_info,omitempty"`
	MetadataRefSHA string         `json:"metadata_ref_sha,omitempty"`
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

	snapshotInfos, err := eng.GetSnapshots()
	if err != nil {
		snapshotInfos = []engine.SnapshotInfo{}
	}

	limit := opts.Limit
	if limit > 0 && limit < len(snapshotInfos) {
		snapshotInfos = snapshotInfos[:limit]
	}

	recentCommands := make([]CommandSnapshot, 0, len(snapshotInfos))
	for _, snapshotInfo := range snapshotInfos {
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

	trunk := eng.Trunk()
	currentBranch := eng.CurrentBranch()
	allBranches := eng.AllBranches()

	metadataRefs, err := eng.ListMetadataRefs()
	if err != nil {
		metadataRefs = make(map[string]string)
	}

	branchNames := make([]string, len(allBranches))
	for i, b := range allBranches {
		branchNames[i] = b.GetName()
	}
	allMeta, _ := eng.BatchReadMetadataRefs(branchNames)

	branchInfos := make([]BranchInfo, 0, len(allBranches))
	for _, branch := range allBranches {
		branchName := branch.GetName()
		branchInfo := BranchInfo{
			Name:      branchName,
			IsTrunk:   branch.IsTrunk(),
			IsTracked: branch.IsTracked(),
		}

		branchObj := eng.GetBranch(branchName)
		sha, err := branchObj.GetRevision()
		if err == nil {
			branchInfo.SHA = sha
		}

		parent := eng.GetParent(branchObj)
		if parent != nil {
			branchInfo.Parent = parent.GetName()
		}

		children := branchObj.GetChildren()
		if len(children) > 0 {
			childNames := make([]string, len(children))
			for i, c := range children {
				childNames[i] = c.GetName()
			}
			branchInfo.Children = childNames
		}

		if meta, ok := allMeta[branchName]; ok && meta != nil {
			if meta.ParentBranchRevision != nil {
				branchInfo.ParentRevision = *meta.ParentBranchRevision
			}

			branch := eng.GetBranch(branchName)
			prInfo, err := eng.GetPrInfo(branch)
			if err == nil && prInfo != nil {
				branchInfo.PRInfo = prInfo
			}
		}

		if metadataSHA, ok := metadataRefs[branchName]; ok {
			branchInfo.MetadataRefSHA = metadataSHA
		}

		if !branchInfo.IsTrunk {
			branch := eng.GetBranch(branchName)
			branchInfo.IsFixed = branch.IsBranchUpToDate()
		} else {
			branchInfo.IsFixed = true
		}

		branchInfos = append(branchInfos, branchInfo)
	}

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

	repoInfo := RepositoryInfo{
		RepoRoot: repoRoot,
	}
	remoteURL, err := git.RunGitCommandWithContext(ctx.Context, "config", "--get", "remote.origin.url")
	if err == nil {
		repoInfo.RemoteURL = remoteURL
	}

	debugInfo := DebugInfo{
		Timestamp:      time.Now(),
		RecentCommands: recentCommands,
		StackState: StackStateInfo{
			Trunk: trunk.GetName(),
			CurrentBranch: func() string {
				if currentBranch != nil {
					return currentBranch.GetName()
				}
				return ""
			}(),
			Branches: branchInfos,
		},
		ContinuationState: continuationState,
		RepositoryInfo:    repoInfo,
	}

	jsonData, err := json.MarshalIndent(debugInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal debug info: %w", err)
	}

	ctx.Splog.Page(string(jsonData))
	ctx.Splog.Newline()

	return nil
}
