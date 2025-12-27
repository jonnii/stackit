// Package git provides a wrapper around git commands and go-git for repository operations.
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	stackiterrors "stackit.dev/stackit/internal/errors"
)

// DefaultCommandTimeout is the default timeout for git commands
const DefaultCommandTimeout = 5 * time.Minute

// ErrStaleRemoteInfo indicates that a push failed because the remote has changed
var ErrStaleRemoteInfo = errors.New("stale info")

// CommandRunner handles execution of git commands
type CommandRunner struct {
	workingDir string
}

// NewCommandRunner creates a new CommandRunner
func NewCommandRunner(workingDir string) *CommandRunner {
	return &CommandRunner{workingDir: workingDir}
}

// defaultRunner is the global runner used by the package-level functions
var defaultRunner = &CommandRunner{}

// SetWorkingDir sets the working directory for the default git runner.
func SetWorkingDir(dir string) {
	defaultRunner.workingDir = dir
}

// GetWorkingDir returns the current working directory setting for the default runner.
func GetWorkingDir() string {
	return defaultRunner.workingDir
}

// RunGitCommand executes a git command using the default runner and returns the output.
// It uses context.Background() with a default timeout.
func RunGitCommand(args ...string) (string, error) {
	return defaultRunner.Run(context.Background(), args...)
}

// RunGitCommandInDir executes a git command in a specific directory and returns the output.
func RunGitCommandInDir(dir string, args ...string) (string, error) {
	runner := &CommandRunner{workingDir: dir}
	return runner.Run(context.Background(), args...)
}

// RunGitCommandWithContext executes a git command with the given context using the default runner.
func RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	return defaultRunner.Run(ctx, args...)
}

// runWithEnv executes a git command with environment variables
func (r *CommandRunner) runWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if r.workingDir != "" {
		cmd.Dir = r.workingDir
	}
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", stackiterrors.NewGitCommandError("git", args, stdout.String(), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Run executes a git command with the given context and returns the output
func (r *CommandRunner) Run(ctx context.Context, args ...string) (string, error) {
	return r.runInternal(ctx, "", true, args...)
}

// runInternal is the internal implementation that handles directory and input
func (r *CommandRunner) runInternal(ctx context.Context, input string, trim bool, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// If no timeout/deadline is set in the context, add the default one
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	if r.workingDir != "" {
		cmd.Dir = r.workingDir
	}
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", stackiterrors.NewGitCommandError("git", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", stackiterrors.NewGitCommandError("git", args, stdout.String(), stderr.String(), err)
	}
	if trim {
		return strings.TrimSpace(stdout.String()), nil
	}
	return stdout.String(), nil
}

// RunGitCommandRaw executes a git command using the default runner and returns the raw output (no trimming)
func RunGitCommandRaw(args ...string) (string, error) {
	return defaultRunner.runInternal(context.Background(), "", false, args...)
}

// RunGitCommandRawWithContext executes a git command using the default runner and returns the raw output (no trimming) with context
func RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error) {
	return defaultRunner.runInternal(ctx, "", false, args...)
}

// RunGitCommandLines executes a git command using the default runner and returns output as lines
func RunGitCommandLines(args ...string) ([]string, error) {
	output, err := RunGitCommand(args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// RunGitCommandLinesWithContext executes a git command with context and returns output as lines
func RunGitCommandLinesWithContext(ctx context.Context, args ...string) ([]string, error) {
	output, err := RunGitCommandWithContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "\n"), nil
}

// RunGitCommandWithInput executes a git command with input using the default runner and returns the output
func RunGitCommandWithInput(input string, args ...string) (string, error) {
	return defaultRunner.runInternal(context.Background(), input, true, args...)
}

// RunGitCommandWithInputAndContext executes a git command with input and context using the default runner
func RunGitCommandWithInputAndContext(ctx context.Context, input string, args ...string) (string, error) {
	return defaultRunner.runInternal(ctx, input, true, args...)
}

// RunGitCommandWithEnv executes a git command with environment variables
func RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	return defaultRunner.runWithEnv(ctx, env, args...)
}

// RunGHCommandWithContext executes a gh command with the given context.
func RunGHCommandWithContext(ctx context.Context, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// If no timeout/deadline is set in the context, add the default one
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultCommandTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	if defaultRunner.workingDir != "" {
		cmd.Dir = defaultRunner.workingDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", stackiterrors.NewGitCommandError("gh", args, stdout.String(), stderr.String(), ctx.Err())
		}
		return "", stackiterrors.NewGitCommandError("gh", args, stdout.String(), stderr.String(), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RunGitCommandInteractive executes a git command interactively with stdin/stdout/stderr
// connected to the terminal.
func RunGitCommandInteractive(args ...string) error {
	cmd := exec.Command("git", args...)
	if defaultRunner.workingDir != "" {
		cmd.Dir = defaultRunner.workingDir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Runner defines the interface for git operations used by the engine.
// This allows the engine to be used with both real git and mock implementations.
type Runner interface {
	// Repository and Config
	InitDefaultRepo() error
	GetRemote() string
	FetchRemoteShas(remote string) (map[string]string, error)
	GetRemoteSha(remote, branchName string) (string, error)

	// Branch Management
	GetCurrentBranch() (string, error)
	GetAllBranchNames() ([]string, error)
	CheckoutBranch(ctx context.Context, branchName string) error
	CreateAndCheckoutBranch(ctx context.Context, branchName string) error
	DeleteBranch(ctx context.Context, branchName string) error
	RenameBranch(ctx context.Context, oldName, newName string) error
	CheckoutDetached(ctx context.Context, revision string) error
	UpdateBranchRef(branchName, revision string) error
	GetRemoteRevision(branchName string) (string, error)

	// Commit and Revision Information
	GetRevision(branchName string) (string, error)
	BatchGetRevisions(branchNames []string) (map[string]string, []error)
	GetMergeBase(rev1, rev2 string) (string, error)
	GetMergeBaseByRef(ref1, ref2 string) (string, error)
	IsAncestor(ancestor, descendant string) (bool, error)
	GetCommitDate(branchName string) (time.Time, error)
	GetCommitAuthor(branchName string) (string, error)
	GetCommitRange(base, head, format string) ([]string, error)
	GetCommitRangeSHAs(base, head string) ([]string, error)
	GetCommitHistorySHAs(branchName string) ([]string, error)
	GetCommitSHA(branchName string, offset int) (string, error)

	// Git Operations
	PullBranch(ctx context.Context, remote, branchName string) (PullResult, error)
	PushBranch(ctx context.Context, branchName, remote string, force, forceWithLease bool) error
	Rebase(ctx context.Context, branchName, upstream, oldUpstream string) (RebaseResult, error)
	RebaseContinue(ctx context.Context) (RebaseResult, error)
	CherryPick(ctx context.Context, commitSHA, onto string) (string, error)
	StashPush(ctx context.Context, message string) (string, error)
	StashPop(ctx context.Context) error
	HardReset(ctx context.Context, revision string) error
	SoftReset(ctx context.Context, revision string) error
	CommitWithOptions(opts CommitOptions) error
	Commit(message string, verbose int) error
	StageAll(ctx context.Context) error
	HasStagedChanges(ctx context.Context) (bool, error)
	HasUnstagedChanges(ctx context.Context) (bool, error)
	IsMerged(ctx context.Context, branchName, target string) (bool, error)
	IsDiffEmpty(ctx context.Context, branchName, base string) (bool, error)
	GetChangedFiles(ctx context.Context, base, head string) ([]string, error)
	GetRebaseHead() (string, error)
	ParseStagedHunks(ctx context.Context) ([]Hunk, error)
	ShowDiff(ctx context.Context, left, right string, stat bool) (string, error)
	ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error)
	GetUnmergedFiles(ctx context.Context) ([]string, error)

	// Worktree operations
	AddWorktree(ctx context.Context, path string, branch string, detach bool) error
	RemoveWorktree(ctx context.Context, path string) error
	ListWorktrees(ctx context.Context) ([]string, error)

	// Runner state
	SetWorkingDir(dir string)
	GetWorkingDir() string

	// Low-level Commands
	RunGitCommand(args ...string) (string, error)
	RunGitCommandWithContext(ctx context.Context, args ...string) (string, error)
	RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error)
	RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error)

	// Low-level Ref and Object Management
	GetRef(name string) (string, error)
	UpdateRef(name, sha string) error
	DeleteRef(name string) error
	CreateBlob(content string) (string, error)
	ReadBlob(sha string) (string, error)
	ListRefs(prefix string) (map[string]string, error)

	// Absorb and Hunks
	GetParentCommitSHA(commitSHA string) (string, error)
	CheckCommutation(hunk Hunk, commitSHA, parentSHA string) (bool, error)
}

// NewRealRunner returns a standard implementation of Runner that calls
// the package-level git functions.
func NewRealRunner() Runner {
	return &realRunner{}
}

// NewRealRunnerWithDir returns a standard implementation of Runner that calls
// the package-level git functions in a specific directory.
func NewRealRunnerWithDir(dir string) Runner {
	return &realRunner{workingDir: dir}
}

// realRunner implements Runner by calling the actual git package functions
type realRunner struct {
	workingDir string
}

func (r *realRunner) SetWorkingDir(dir string) {
	r.workingDir = dir
}

func (r *realRunner) GetWorkingDir() string {
	return r.workingDir
}

func (r *realRunner) RunGitCommandWithContext(ctx context.Context, args ...string) (string, error) {
	if r.workingDir != "" {
		runner := &CommandRunner{workingDir: r.workingDir}
		return runner.Run(ctx, args...)
	}
	return RunGitCommandWithContext(ctx, args...)
}

func (r *realRunner) RunGitCommandRawWithContext(ctx context.Context, args ...string) (string, error) {
	if r.workingDir != "" {
		runner := &CommandRunner{workingDir: r.workingDir}
		return runner.runInternal(ctx, "", false, args...)
	}
	return RunGitCommandRawWithContext(ctx, args...)
}

func (r *realRunner) InitDefaultRepo() error {
	if r.workingDir != "" {
		_, err := RunGitCommandInDir(r.workingDir, "rev-parse", "--is-inside-work-tree")
		return err
	}
	return InitDefaultRepo()
}

func (r *realRunner) GetRemote() string {
	return GetRemote()
}

func (r *realRunner) FetchRemoteShas(remote string) (map[string]string, error) {
	return FetchRemoteShas(remote)
}

func (r *realRunner) GetRemoteSha(remote, branchName string) (string, error) {
	return GetRemoteSha(remote, branchName)
}

func (r *realRunner) GetCurrentBranch() (string, error) {
	return GetCurrentBranch()
}

func (r *realRunner) GetAllBranchNames() ([]string, error) {
	return GetAllBranchNames()
}

func (r *realRunner) CheckoutBranch(ctx context.Context, branchName string) error {
	return CheckoutBranch(ctx, branchName)
}

func (r *realRunner) CreateAndCheckoutBranch(ctx context.Context, branchName string) error {
	return CreateAndCheckoutBranch(ctx, branchName)
}

func (r *realRunner) DeleteBranch(ctx context.Context, branchName string) error {
	return DeleteBranch(ctx, branchName)
}

func (r *realRunner) RenameBranch(ctx context.Context, oldName, newName string) error {
	return RenameBranch(ctx, oldName, newName)
}

func (r *realRunner) CheckoutDetached(ctx context.Context, revision string) error {
	return CheckoutDetached(ctx, revision)
}

func (r *realRunner) UpdateBranchRef(branchName, revision string) error {
	return UpdateBranchRef(branchName, revision)
}

func (r *realRunner) GetRemoteRevision(branchName string) (string, error) {
	return GetRemoteRevision(branchName)
}

func (r *realRunner) GetRevision(branchName string) (string, error) {
	return GetRevision(branchName)
}

func (r *realRunner) BatchGetRevisions(branchNames []string) (map[string]string, []error) {
	return BatchGetRevisions(branchNames)
}

func (r *realRunner) GetMergeBase(rev1, rev2 string) (string, error) {
	return GetMergeBase(rev1, rev2)
}

func (r *realRunner) GetMergeBaseByRef(ref1, ref2 string) (string, error) {
	return GetMergeBaseByRef(ref1, ref2)
}

func (r *realRunner) IsAncestor(ancestor, descendant string) (bool, error) {
	return IsAncestor(ancestor, descendant)
}

func (r *realRunner) GetCommitDate(branchName string) (time.Time, error) {
	return GetCommitDate(branchName)
}

func (r *realRunner) GetCommitAuthor(branchName string) (string, error) {
	return GetCommitAuthor(branchName)
}

func (r *realRunner) GetCommitRange(base, head, format string) ([]string, error) {
	return GetCommitRange(base, head, format)
}

func (r *realRunner) GetCommitRangeSHAs(base, head string) ([]string, error) {
	return GetCommitRangeSHAs(base, head)
}

func (r *realRunner) GetCommitHistorySHAs(branchName string) ([]string, error) {
	return GetCommitHistorySHAs(branchName)
}

func (r *realRunner) GetCommitSHA(branchName string, offset int) (string, error) {
	return GetCommitSHA(branchName, offset)
}

func (r *realRunner) PullBranch(ctx context.Context, remote, branchName string) (PullResult, error) {
	return PullBranch(ctx, remote, branchName)
}

func (r *realRunner) PushBranch(ctx context.Context, branchName, remote string, force, forceWithLease bool) error {
	return PushBranch(ctx, branchName, remote, force, forceWithLease)
}

func (r *realRunner) Rebase(ctx context.Context, branchName, upstream, oldUpstream string) (RebaseResult, error) {
	return Rebase(ctx, branchName, upstream, oldUpstream)
}

func (r *realRunner) RebaseContinue(ctx context.Context) (RebaseResult, error) {
	return RebaseContinue(ctx)
}

func (r *realRunner) CherryPick(ctx context.Context, commitSHA, onto string) (string, error) {
	return CherryPick(ctx, commitSHA, onto)
}

func (r *realRunner) StashPush(ctx context.Context, message string) (string, error) {
	return StashPush(ctx, message)
}

func (r *realRunner) StashPop(ctx context.Context) error {
	return StashPop(ctx)
}

func (r *realRunner) HardReset(ctx context.Context, revision string) error {
	return HardReset(ctx, revision)
}

func (r *realRunner) SoftReset(ctx context.Context, revision string) error {
	return SoftReset(ctx, revision)
}

func (r *realRunner) CommitWithOptions(opts CommitOptions) error {
	return CommitWithOptions(opts)
}

func (r *realRunner) Commit(message string, verbose int) error {
	return Commit(message, verbose)
}

func (r *realRunner) StageAll(ctx context.Context) error {
	return StageAll(ctx)
}

func (r *realRunner) HasStagedChanges(ctx context.Context) (bool, error) {
	return HasStagedChanges(ctx)
}

func (r *realRunner) HasUnstagedChanges(ctx context.Context) (bool, error) {
	return HasUnstagedChanges(ctx)
}

func (r *realRunner) IsMerged(ctx context.Context, branchName, target string) (bool, error) {
	return IsMerged(ctx, branchName, target)
}

func (r *realRunner) IsDiffEmpty(ctx context.Context, branchName, base string) (bool, error) {
	return IsDiffEmpty(ctx, branchName, base)
}

func (r *realRunner) GetChangedFiles(ctx context.Context, base, head string) ([]string, error) {
	return GetChangedFiles(ctx, base, head)
}

func (r *realRunner) ParseStagedHunks(ctx context.Context) ([]Hunk, error) {
	return ParseStagedHunks(ctx)
}

func (r *realRunner) ShowDiff(ctx context.Context, left, right string, stat bool) (string, error) {
	return ShowDiff(ctx, left, right, stat)
}

func (r *realRunner) ShowCommits(ctx context.Context, base, head string, patch, stat bool) (string, error) {
	return ShowCommits(ctx, base, head, patch, stat)
}

func (r *realRunner) GetUnmergedFiles(ctx context.Context) ([]string, error) {
	return GetUnmergedFiles(ctx)
}

func (r *realRunner) AddWorktree(ctx context.Context, path string, branch string, detach bool) error {
	return AddWorktree(ctx, path, branch, detach)
}

func (r *realRunner) RemoveWorktree(ctx context.Context, path string) error {
	return RemoveWorktree(ctx, path)
}

func (r *realRunner) ListWorktrees(ctx context.Context) ([]string, error) {
	return ListWorktrees(ctx)
}

func (r *realRunner) RunGitCommand(args ...string) (string, error) {
	return RunGitCommand(args...)
}

func (r *realRunner) RunGitCommandWithEnv(ctx context.Context, env []string, args ...string) (string, error) {
	return RunGitCommandWithEnv(ctx, env, args...)
}

func (r *realRunner) GetRef(name string) (string, error) {
	return GetRef(name)
}

func (r *realRunner) UpdateRef(name, sha string) error {
	return UpdateRef(name, sha)
}

func (r *realRunner) DeleteRef(name string) error {
	return DeleteRef(name)
}

func (r *realRunner) CreateBlob(content string) (string, error) {
	return CreateBlob(content)
}

func (r *realRunner) ReadBlob(sha string) (string, error) {
	return ReadBlob(sha)
}

func (r *realRunner) ListRefs(prefix string) (map[string]string, error) {
	return ListRefs(prefix)
}

func (r *realRunner) GetParentCommitSHA(commitSHA string) (string, error) {
	return GetParentCommitSHA(commitSHA)
}

func (r *realRunner) CheckCommutation(hunk Hunk, commitSHA, parentSHA string) (bool, error) {
	return CheckCommutation(hunk, commitSHA, parentSHA)
}

func (r *realRunner) GetRebaseHead() (string, error) {
	return GetRebaseHead()
}
