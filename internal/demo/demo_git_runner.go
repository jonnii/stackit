package demo

import (
	"context"
	"time"

	"stackit.dev/stackit/internal/git"
)

// demoGitRunner implements git.Runner with simulated data for demo mode.
type demoGitRunner struct {
	trunk         string
	currentBranch string
	branches      []Branch
}

// NewDemoGitRunner creates a new demo git runner with simulated data.
func NewDemoGitRunner() git.Runner {
	return &demoGitRunner{
		trunk:         "main",
		currentBranch: "main",
		branches:      demoBranches,
	}
}

func (d *demoGitRunner) InitDefaultRepo() error {
	return nil
}

func (d *demoGitRunner) GetRemote() string {
	return "origin"
}

func (d *demoGitRunner) FetchRemoteShas(_ string) (map[string]string, error) {
	return make(map[string]string), nil
}

func (d *demoGitRunner) GetRemoteSha(_, _ string) (string, error) {
	return "remote-sha", nil
}

func (d *demoGitRunner) GetCurrentBranch() (string, error) {
	return d.currentBranch, nil
}

func (d *demoGitRunner) GetAllBranchNames() ([]string, error) {
	names := make([]string, len(d.branches))
	for i, b := range d.branches {
		names[i] = b.Name
	}
	return names, nil
}

func (d *demoGitRunner) CheckoutBranch(_ context.Context, branchName string) error {
	d.currentBranch = branchName
	return nil
}

func (d *demoGitRunner) CreateAndCheckoutBranch(_ context.Context, branchName string) error {
	d.branches = append(d.branches, Branch{Name: branchName})
	d.currentBranch = branchName
	return nil
}

func (d *demoGitRunner) DeleteBranch(_ context.Context, branchName string) error {
	for i, b := range d.branches {
		if b.Name == branchName {
			d.branches = append(d.branches[:i], d.branches[i+1:]...)
			break
		}
	}
	return nil
}

func (d *demoGitRunner) RenameBranch(_ context.Context, oldName, newName string) error {
	for i, b := range d.branches {
		if b.Name == oldName {
			d.branches[i].Name = newName
			break
		}
	}
	if d.currentBranch == oldName {
		d.currentBranch = newName
	}
	return nil
}

func (d *demoGitRunner) CheckoutDetached(_ context.Context, _ string) error {
	d.currentBranch = ""
	return nil
}

func (d *demoGitRunner) UpdateBranchRef(_, _ string) error {
	return nil
}

func (d *demoGitRunner) GetRemoteRevision(_ string) (string, error) {
	return "remote-rev", nil
}

func (d *demoGitRunner) GetRevision(branchName string) (string, error) {
	for _, b := range d.branches {
		if b.Name == branchName {
			return b.SHA, nil
		}
	}
	return "rev-sha", nil
}

func (d *demoGitRunner) BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	results := make(map[string]string)
	for _, name := range branchNames {
		results[name], _ = d.GetRevision(name)
	}
	return results, nil
}

func (d *demoGitRunner) GetMergeBase(_, _ string) (string, error) {
	return "merge-base-sha", nil
}

func (d *demoGitRunner) GetMergeBaseByRef(_, _ string) (string, error) {
	return "merge-base-sha", nil
}

func (d *demoGitRunner) IsAncestor(_, _ string) (bool, error) {
	return true, nil
}

func (d *demoGitRunner) GetCommitDate(_ string) (time.Time, error) {
	return time.Now(), nil
}

func (d *demoGitRunner) GetCommitAuthor(_ string) (string, error) {
	return "Demo User", nil
}

func (d *demoGitRunner) GetCommitRange(_, _, _ string) ([]string, error) {
	return []string{"commit message"}, nil
}

func (d *demoGitRunner) GetCommitSHA(_ string, _ int) (string, error) {
	return "commit-sha", nil
}

func (d *demoGitRunner) PullBranch(_ context.Context, _, _ string) (git.PullResult, error) {
	return git.PullDone, nil
}

func (d *demoGitRunner) PushBranch(_ context.Context, _, _ string, _, _ bool) error {
	return nil
}

func (d *demoGitRunner) Rebase(_ context.Context, _, _, _ string) (git.RebaseResult, error) {
	return git.RebaseDone, nil
}

func (d *demoGitRunner) RebaseContinue(_ context.Context) (git.RebaseResult, error) {
	return git.RebaseDone, nil
}

func (d *demoGitRunner) CherryPick(_ context.Context, commitSHA, _ string) (string, error) {
	return commitSHA, nil
}

func (d *demoGitRunner) StashPush(_ context.Context, _ string) (string, error) {
	return "stashed", nil
}

func (d *demoGitRunner) StashPop(_ context.Context) error {
	return nil
}

func (d *demoGitRunner) GetCommitRangeSHAs(_, _ string) ([]string, error) {
	return []string{"sha1", "sha2"}, nil
}

func (d *demoGitRunner) GetCommitHistorySHAs(_ string) ([]string, error) {
	return []string{"sha1", "sha2"}, nil
}

func (d *demoGitRunner) GetRebaseHead() (string, error) {
	return "rebase-head-sha", nil
}

func (d *demoGitRunner) ParseStagedHunks(_ context.Context) ([]git.Hunk, error) {
	return []git.Hunk{}, nil
}

func (d *demoGitRunner) HardReset(_ context.Context, _ string) error {
	return nil
}

func (d *demoGitRunner) SoftReset(_ context.Context, _ string) error {
	return nil
}

func (d *demoGitRunner) CommitWithOptions(_ git.CommitOptions) error {
	return nil
}

func (d *demoGitRunner) Commit(_ string, _ int) error {
	return nil
}

func (d *demoGitRunner) StageAll(_ context.Context) error {
	return nil
}

func (d *demoGitRunner) HasStagedChanges(_ context.Context) (bool, error) {
	return false, nil
}

func (d *demoGitRunner) HasUnstagedChanges(_ context.Context) (bool, error) {
	return false, nil
}

func (d *demoGitRunner) IsMerged(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (d *demoGitRunner) IsDiffEmpty(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (d *demoGitRunner) GetChangedFiles(_ context.Context, _, _ string) ([]string, error) {
	return []string{}, nil
}

func (d *demoGitRunner) ShowDiff(_ context.Context, _, _ string, _ bool) (string, error) {
	return "diff", nil
}

func (d *demoGitRunner) ShowCommits(_ context.Context, _, _ string, _, _ bool) (string, error) {
	return "commits", nil
}

func (d *demoGitRunner) GetUnmergedFiles(_ context.Context) ([]string, error) {
	return []string{}, nil
}

func (d *demoGitRunner) AddWorktree(_ context.Context, _, _ string, _ bool) error {
	return nil
}

func (d *demoGitRunner) RemoveWorktree(_ context.Context, _ string) error {
	return nil
}

func (d *demoGitRunner) ListWorktrees(_ context.Context) ([]string, error) {
	return []string{}, nil
}

func (d *demoGitRunner) SetWorkingDir(_ string) {}

func (d *demoGitRunner) GetWorkingDir() string {
	return ""
}

func (d *demoGitRunner) RunGitCommand(_ ...string) (string, error) {
	return "", nil
}

func (d *demoGitRunner) RunGitCommandWithContext(_ context.Context, _ ...string) (string, error) {
	return "", nil
}

func (d *demoGitRunner) RunGitCommandWithEnv(_ context.Context, _ []string, _ ...string) (string, error) {
	return "", nil
}

func (d *demoGitRunner) RunGitCommandRawWithContext(_ context.Context, _ ...string) (string, error) {
	return "", nil
}

func (d *demoGitRunner) GetRef(_ string) (string, error) {
	return "ref-sha", nil
}

func (d *demoGitRunner) UpdateRef(_, _ string) error {
	return nil
}

func (d *demoGitRunner) DeleteRef(_ string) error {
	return nil
}

func (d *demoGitRunner) CreateBlob(_ string) (string, error) {
	return "blob-sha", nil
}

func (d *demoGitRunner) ReadBlob(_ string) (string, error) {
	return "{}", nil
}

func (d *demoGitRunner) ListRefs(_ string) (map[string]string, error) {
	return make(map[string]string), nil
}

func (d *demoGitRunner) GetParentCommitSHA(_ string) (string, error) {
	return "parent-sha", nil
}

func (d *demoGitRunner) CheckCommutation(_ git.Hunk, _, _ string) (bool, error) {
	return true, nil
}
