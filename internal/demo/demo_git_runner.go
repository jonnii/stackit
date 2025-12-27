package demo

import (
	"context"
	"fmt"
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
		trunk:         GetDemoTrunk(),
		currentBranch: GetDemoCurrentBranch(),
		branches:      GetDemoBranches(),
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

func (d *demoGitRunner) GetRemoteSha(_, branchName string) (string, error) {
	return "remote-sha-" + branchName, nil
}

func (d *demoGitRunner) GetCurrentBranch() (string, error) {
	return d.currentBranch, nil
}

func (d *demoGitRunner) GetAllBranchNames() ([]string, error) {
	names := []string{d.trunk}
	for _, b := range d.branches {
		names = append(names, b.Name)
	}
	return names, nil
}

func (d *demoGitRunner) CheckoutBranch(_ context.Context, branchName string) error {
	d.currentBranch = branchName
	return nil
}

func (d *demoGitRunner) DeleteBranch(_ context.Context, _ string) error {
	return nil
}

func (d *demoGitRunner) RenameBranch(_ context.Context, _, _ string) error {
	return nil
}

func (d *demoGitRunner) UpdateBranchRef(_, _ string) error {
	return nil
}

func (d *demoGitRunner) GetRemoteRevision(branchName string) (string, error) {
	return "remote-rev-" + branchName, nil
}

func (d *demoGitRunner) GetMetadataRefList() (map[string]string, error) {
	refs := make(map[string]string)
	for _, b := range d.branches {
		refs[b.Name] = "meta-sha-" + b.Name
	}
	return refs, nil
}

func (d *demoGitRunner) ReadMetadataRef(branchName string) (*git.Meta, error) {
	if branchName == d.trunk {
		return &git.Meta{}, nil
	}

	for _, b := range d.branches {
		if b.Name == branchName {
			parentRev := "sha-" + b.Parent
			scope := b.Scope
			num := b.PRNumber
			title := b.PRTitle
			state := b.PRState
			isDraft := b.IsDraft
			url := fmt.Sprintf("https://github.com/example/repo/pull/%d", num)
			body := "Demo PR body for " + branchName

			return &git.Meta{
				ParentBranchName:     &b.Parent,
				ParentBranchRevision: &parentRev,
				Scope:                &scope,
				PrInfo: &git.PrInfo{
					Number:  &num,
					Title:   &title,
					Body:    &body,
					IsDraft: &isDraft,
					State:   &state,
					Base:    &b.Parent,
					URL:     &url,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("metadata not found for branch %s", branchName)
}

func (d *demoGitRunner) WriteMetadataRef(_ string, _ *git.Meta) error {
	return nil
}

func (d *demoGitRunner) DeleteMetadataRef(_ string) error {
	return nil
}

func (d *demoGitRunner) BatchReadMetadataRefs(branchNames []string) (map[string]*git.Meta, map[string]error) {
	result := make(map[string]*git.Meta)
	errs := make(map[string]error)
	for _, name := range branchNames {
		if meta, err := d.ReadMetadataRef(name); err == nil {
			result[name] = meta
		} else {
			errs[name] = err
		}
	}
	return result, errs
}

func (d *demoGitRunner) RenameMetadataRef(_, _ string) error {
	return nil
}

func (d *demoGitRunner) GetRevision(branchName string) (string, error) {
	return "sha-" + branchName, nil
}

func (d *demoGitRunner) BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	result := make(map[string]string)
	for _, name := range branchNames {
		result[name] = "sha-" + name
	}
	return result, nil
}

func (d *demoGitRunner) GetMergeBase(_, rev2 string) (string, error) {
	return rev2, nil // Assume rev2 is parent
}

func (d *demoGitRunner) IsAncestor(_, _ string) (bool, error) {
	return true, nil
}

func (d *demoGitRunner) GetCommitDate(_ string) (time.Time, error) {
	return time.Now().Add(-24 * time.Hour), nil
}

func (d *demoGitRunner) GetCommitAuthor(_ string) (string, error) {
	return "Demo User <demo@example.com>", nil
}

func (d *demoGitRunner) GetCommitRange(_, head, _ string) ([]string, error) {
	// Look up branch name from head which is "sha-branchName"
	branchName := ""
	if len(head) > 4 && head[:4] == "sha-" {
		branchName = head[4:]
	}

	commits := 0
	for _, b := range d.branches {
		if b.Name == branchName {
			commits = b.Commits
			break
		}
	}

	result := make([]string, commits)
	for i := 0; i < commits; i++ {
		result[i] = fmt.Sprintf("sha-%s-%d commit %d", branchName, i, i)
	}
	return result, nil
}

func (d *demoGitRunner) GetCommitRangeSHAs(base, head string) ([]string, error) {
	return d.GetCommitRange(base, head, "SHA")
}

func (d *demoGitRunner) GetCommitHistorySHAs(branchName string) ([]string, error) {
	return []string{"sha-" + branchName}, nil
}

func (d *demoGitRunner) GetCommitSHA(branchName string, _ int) (string, error) {
	return "sha-" + branchName, nil
}

func (d *demoGitRunner) PullBranch(_ context.Context, _, _ string) (git.PullResult, error) {
	return git.PullUnneeded, nil
}

func (d *demoGitRunner) PushBranch(_ context.Context, _, _ string, _, _ bool) error {
	return nil
}

func (d *demoGitRunner) Rebase(_ context.Context, branchName, _, _ string) (git.RebaseResult, error) {
	if branchName == "conflict" {
		return git.RebaseConflict, nil
	}
	return git.RebaseDone, nil
}

func (d *demoGitRunner) RebaseContinue(_ context.Context) (git.RebaseResult, error) {
	return git.RebaseDone, nil
}

func (d *demoGitRunner) FinalizeRebase(_ context.Context, _, _, _, _ string) error {
	return nil
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

func (d *demoGitRunner) IsMerged(_ context.Context, branchName, _ string) (bool, error) {
	for _, b := range d.branches {
		if b.Name == branchName && b.PRState == "MERGED" {
			return true, nil
		}
	}
	return false, nil
}

func (d *demoGitRunner) IsDiffEmpty(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (d *demoGitRunner) RunGitCommand(args ...string) (string, error) {
	if len(args) > 0 && args[0] == "diff" {
		// Handle diff --numstat
		branchName := ""
		if len(args) > 3 {
			head := args[len(args)-1]
			if len(head) > 4 && head[:4] == "sha-" {
				branchName = head[4:]
			}
		}

		for _, b := range d.branches {
			if b.Name == branchName {
				return fmt.Sprintf("%d\t%d\tfile.txt", b.Added, b.Deleted), nil
			}
		}
	}
	return "", nil
}

func (d *demoGitRunner) RunGitCommandWithContext(_ context.Context, args ...string) (string, error) {
	return d.RunGitCommand(args...)
}
