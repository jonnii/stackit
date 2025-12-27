package engine

import (
	"context"
	"time"

	"stackit.dev/stackit/internal/git"
)

// GitRunner defines the interface for git operations used by the engine.
// This allows the engine to be used with both real git and mock implementations.
type GitRunner interface {
	// Repository and Config
	InitDefaultRepo() error
	GetRemote() string
	FetchRemoteShas(remote string) (map[string]string, error)
	GetRemoteSha(remote, branchName string) (string, error)

	// Branch Management
	GetCurrentBranch() (string, error)
	GetAllBranchNames() ([]string, error)
	CheckoutBranch(ctx context.Context, branchName string) error
	DeleteBranch(ctx context.Context, branchName string) error
	RenameBranch(ctx context.Context, oldName, newName string) error
	UpdateBranchRef(branchName, revision string) error
	GetRemoteRevision(branchName string) (string, error)

	// Metadata Management
	GetMetadataRefList() (map[string]string, error)
	ReadMetadataRef(branchName string) (*git.Meta, error)
	WriteMetadataRef(branchName string, meta *git.Meta) error
	DeleteMetadataRef(branchName string) error
	BatchReadMetadataRefs(branchNames []string) (map[string]*git.Meta, map[string]error)
	RenameMetadataRef(oldName, newName string) error

	// Commit and Revision Information
	GetRevision(branchName string) (string, error)
	BatchGetRevisions(branchNames []string) (map[string]string, []error)
	GetMergeBase(rev1, rev2 string) (string, error)
	IsAncestor(ancestor, descendant string) (bool, error)
	GetCommitDate(branchName string) (time.Time, error)
	GetCommitAuthor(branchName string) (string, error)
	GetCommitRange(base, head, format string) ([]string, error)
	GetCommitRangeSHAs(base, head string) ([]string, error)
	GetCommitHistorySHAs(branchName string) ([]string, error)
	GetCommitSHA(branchName string, offset int) (string, error)

	// Git Operations
	PullBranch(ctx context.Context, remote, branchName string) (git.PullResult, error)
	PushBranch(ctx context.Context, branchName, remote string, force, forceWithLease bool) error
	Rebase(ctx context.Context, branchName, upstream, oldUpstream string) (git.RebaseResult, error)
	RebaseContinue(ctx context.Context) (git.RebaseResult, error)
	FinalizeRebase(ctx context.Context, branchName, newRev, oldUpstream, upstream string) error
	HardReset(ctx context.Context, revision string) error
	SoftReset(ctx context.Context, revision string) error
	CommitWithOptions(opts git.CommitOptions) error
	IsMerged(ctx context.Context, branchName, target string) (bool, error)
	IsDiffEmpty(ctx context.Context, branchName, base string) (bool, error)

	// Low-level Commands
	RunGitCommand(args ...string) (string, error)
	RunGitCommandWithContext(ctx context.Context, args ...string) (string, error)
}

// realGitRunner implements GitRunner by calling the actual git package functions
type realGitRunner struct{}

func (r *realGitRunner) InitDefaultRepo() error {
	return git.InitDefaultRepo()
}

func (r *realGitRunner) GetRemote() string {
	return git.GetRemote()
}

func (r *realGitRunner) FetchRemoteShas(remote string) (map[string]string, error) {
	return git.FetchRemoteShas(remote)
}

func (r *realGitRunner) GetRemoteSha(remote, branchName string) (string, error) {
	return git.GetRemoteSha(remote, branchName)
}

func (r *realGitRunner) GetCurrentBranch() (string, error) {
	return git.GetCurrentBranch()
}

func (r *realGitRunner) GetAllBranchNames() ([]string, error) {
	return git.GetAllBranchNames()
}

func (r *realGitRunner) CheckoutBranch(ctx context.Context, branchName string) error {
	return git.CheckoutBranch(ctx, branchName)
}

func (r *realGitRunner) DeleteBranch(ctx context.Context, branchName string) error {
	return git.DeleteBranch(ctx, branchName)
}

func (r *realGitRunner) RenameBranch(ctx context.Context, oldName, newName string) error {
	return git.RenameBranch(ctx, oldName, newName)
}

func (r *realGitRunner) UpdateBranchRef(branchName, revision string) error {
	return git.UpdateBranchRef(branchName, revision)
}

func (r *realGitRunner) GetRemoteRevision(branchName string) (string, error) {
	return git.GetRemoteRevision(branchName)
}

func (r *realGitRunner) GetMetadataRefList() (map[string]string, error) {
	return git.GetMetadataRefList()
}

func (r *realGitRunner) ReadMetadataRef(branchName string) (*git.Meta, error) {
	return git.ReadMetadataRef(branchName)
}

func (r *realGitRunner) WriteMetadataRef(branchName string, meta *git.Meta) error {
	return git.WriteMetadataRef(branchName, meta)
}

func (r *realGitRunner) DeleteMetadataRef(branchName string) error {
	return git.DeleteMetadataRef(branchName)
}

func (r *realGitRunner) BatchReadMetadataRefs(branchNames []string) (map[string]*git.Meta, map[string]error) {
	return git.BatchReadMetadataRefs(branchNames)
}

func (r *realGitRunner) RenameMetadataRef(oldName, newName string) error {
	return git.RenameMetadataRef(oldName, newName)
}

func (r *realGitRunner) GetRevision(branchName string) (string, error) {
	return git.GetRevision(branchName)
}

func (r *realGitRunner) BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	return git.BatchGetRevisions(branchNames)
}

func (r *realGitRunner) GetMergeBase(rev1, rev2 string) (string, error) {
	return git.GetMergeBase(rev1, rev2)
}

func (r *realGitRunner) IsAncestor(ancestor, descendant string) (bool, error) {
	return git.IsAncestor(ancestor, descendant)
}

func (r *realGitRunner) GetCommitDate(branchName string) (time.Time, error) {
	return git.GetCommitDate(branchName)
}

func (r *realGitRunner) GetCommitAuthor(branchName string) (string, error) {
	return git.GetCommitAuthor(branchName)
}

func (r *realGitRunner) GetCommitRange(base, head, format string) ([]string, error) {
	return git.GetCommitRange(base, head, format)
}

func (r *realGitRunner) GetCommitRangeSHAs(base, head string) ([]string, error) {
	return git.GetCommitRangeSHAs(base, head)
}

func (r *realGitRunner) GetCommitHistorySHAs(branchName string) ([]string, error) {
	return git.GetCommitHistorySHAs(branchName)
}

func (r *realGitRunner) GetCommitSHA(branchName string, offset int) (string, error) {
	return git.GetCommitSHA(branchName, offset)
}

func (r *realGitRunner) PullBranch(ctx context.Context, remote, branchName string) (git.PullResult, error) {
	return git.PullBranch(ctx, remote, branchName)
}

func (r *realGitRunner) PushBranch(ctx context.Context, branchName, remote string, force, forceWithLease bool) error {
	return git.PushBranch(ctx, branchName, remote, force, forceWithLease)
}

func (r *realGitRunner) Rebase(ctx context.Context, branchName, upstream, oldUpstream string) (git.RebaseResult, error) {
	return git.Rebase(ctx, branchName, upstream, oldUpstream)
}

func (r *realGitRunner) RebaseContinue(ctx context.Context) (git.RebaseResult, error) {
	return git.RebaseContinue(ctx)
}

func (r *realGitRunner) FinalizeRebase(ctx context.Context, branchName, newRev, oldUpstream, upstream string) error {
	return git.FinalizeRebase(ctx, branchName, newRev, oldUpstream, upstream)
}

func (r *realGitRunner) HardReset(ctx context.Context, revision string) error {
	return git.HardReset(ctx, revision)
}

func (r *realGitRunner) SoftReset(ctx context.Context, revision string) error {
	return git.SoftReset(ctx, revision)
}

func (r *realGitRunner) CommitWithOptions(opts git.CommitOptions) error {
	return git.CommitWithOptions(opts)
}

func (r *realGitRunner) IsMerged(ctx context.Context, branchName, target string) (bool, error) {
	return git.IsMerged(ctx, branchName, target)
}

func (r *realGitRunner) IsDiffEmpty(ctx context.Context, branchName, base string) (bool, error) {
	return git.IsDiffEmpty(ctx, branchName, base)
}

func (r *realGitRunner) RunGitCommand(args ...string) (string, error) {
	return git.RunGitCommand(args...)
}

func (r *realGitRunner) RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	return git.RunGitCommandWithContext(ctx, args...)
}
