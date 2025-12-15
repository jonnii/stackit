package testhelpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const textFileName = "test.txt"

// GitRepo represents a Git repository for testing purposes.
// This is the Go equivalent of the TypeScript GitRepo class.
type GitRepo struct {
	Dir            string
	UserConfigPath string
}

// NewGitRepo creates a new Git repository in the specified directory.
// If opts.ExistingRepo is true, it assumes the repo already exists.
// If opts.RepoURL is provided, it clones the repository instead of initializing.
func NewGitRepo(dir string, opts ...GitRepoOption) (*GitRepo, error) {
	repo := &GitRepo{
		Dir:            dir,
		UserConfigPath: filepath.Join(dir, ".git", ".stackit_user_config"),
	}

	options := &gitRepoOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.existingRepo {
		return repo, nil
	}

	if options.repoURL != "" {
		// Clone repository
		cmd := exec.Command("git", "clone", options.repoURL, dir)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to clone repo: %w", err)
		}
	} else {
		// Initialize new repository
		cmd := exec.Command("git", "init", dir, "-b", "main")
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to init repo: %w", err)
		}
	}

	// Configure Git user (required for commits)
	if err := repo.runGitCommand("config", "user.name", "Test User"); err != nil {
		return nil, err
	}
	if err := repo.runGitCommand("config", "user.email", "test@example.com"); err != nil {
		return nil, err
	}

	return repo, nil
}

// gitRepoOptions holds options for creating a GitRepo.
type gitRepoOptions struct {
	existingRepo bool
	repoURL      string
}

// GitRepoOption is a function that configures GitRepo creation.
type GitRepoOption func(*gitRepoOptions)

// WithExistingRepo indicates the repository already exists.
func WithExistingRepo() GitRepoOption {
	return func(opts *gitRepoOptions) {
		opts.existingRepo = true
	}
}

// WithRepoURL specifies a URL to clone from.
func WithRepoURL(url string) GitRepoOption {
	return func(opts *gitRepoOptions) {
		opts.repoURL = url
	}
}

// runGitCommand executes a git command in the repository directory.
func (r *GitRepo) runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	if os.Getenv("DEBUG") == "" {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	return cmd.Run()
}

// RunGitCommand executes a git command and returns an error if it fails.
func (r *GitRepo) RunGitCommand(args ...string) error {
	return r.runGitCommand(args...)
}

// runGitCommandAndGetOutput executes a git command and returns its output.
func (r *GitRepo) runGitCommandAndGetOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w", err)
	}
	// Trim all trailing whitespace including newlines
	result := strings.TrimSpace(string(output))
	return result, nil
}

// RunGitCommandAndGetOutput executes a git command and returns its output.
func (r *GitRepo) RunGitCommandAndGetOutput(args ...string) (string, error) {
	return r.runGitCommandAndGetOutput(args...)
}

// RunCliCommand executes a Stackit CLI command in the repository directory.
// This will need to be updated once the CLI is built.
func (r *GitRepo) RunCliCommand(command []string) error {
	// TODO: Update this path once the CLI binary is built
	// For now, this is a placeholder that will need to be updated
	cliPath := "stackit" // Will be the built binary path
	
	cmd := exec.Command(cliPath, command...)
	cmd.Dir = r.Dir
	
	env := os.Environ()
	env = append(env, "STACKIT_USER_CONFIG_PATH="+r.UserConfigPath)
	env = append(env, "STACKIT_DISABLE_TELEMETRY=1")
	env = append(env, "STACKIT_DISABLE_UPGRADE_PROMPT=1")
	env = append(env, "STACKIT_DISABLE_SURVEY=1")
	env = append(env, "STACKIT_PROFILE=")
	cmd.Env = env
	
	if os.Getenv("DEBUG") == "" {
		cmd.Stdout = nil
		cmd.Stderr = nil
	}
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI command failed: %w", err)
	}
	
	return nil
}

// RunCliCommandAndGetOutput executes a Stackit CLI command and returns its output.
func (r *GitRepo) RunCliCommandAndGetOutput(command []string) (string, error) {
	cliPath := "stackit" // Will be the built binary path
	
	cmd := exec.Command(cliPath, command...)
	cmd.Dir = r.Dir
	
	env := os.Environ()
	env = append(env, "STACKIT_USER_CONFIG_PATH="+r.UserConfigPath)
	env = append(env, "STACKIT_DISABLE_TELEMETRY=1")
	env = append(env, "STACKIT_DISABLE_UPGRADE_PROMPT=1")
	env = append(env, "STACKIT_DISABLE_SURVEY=1")
	cmd.Env = env
	
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("CLI command failed: %w", err)
	}
	
	return string(output), nil
}

// CreateChange creates a file change in the repository.
func (r *GitRepo) CreateChange(textValue string, prefix string, unstaged bool) error {
	fileName := textFileName
	if prefix != "" {
		fileName = prefix + "_" + fileName
	}
	filePath := filepath.Join(r.Dir, fileName)
	
	if err := os.WriteFile(filePath, []byte(textValue), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	if !unstaged {
		return r.runGitCommand("add", filePath)
	}
	
	return nil
}

// CreateChangeAndCommit creates a file change and commits it.
func (r *GitRepo) CreateChangeAndCommit(textValue string, prefix string) error {
	if err := r.CreateChange(textValue, prefix, false); err != nil {
		return err
	}
	if err := r.runGitCommand("add", "."); err != nil {
		return err
	}
	return r.runGitCommand("commit", "-m", textValue)
}

// CreateChangeAndAmend creates a file change and amends the last commit.
func (r *GitRepo) CreateChangeAndAmend(textValue string, prefix string) error {
	if err := r.CreateChange(textValue, prefix, false); err != nil {
		return err
	}
	if err := r.runGitCommand("add", "."); err != nil {
		return err
	}
	return r.runGitCommand("commit", "--amend", "--no-edit")
}

// DeleteBranch deletes a branch.
func (r *GitRepo) DeleteBranch(name string) error {
	return r.runGitCommand("branch", "-D", name)
}

// CreatePrecommitHook creates a pre-commit hook.
func (r *GitRepo) CreatePrecommitHook(contents string) error {
	hookDir := filepath.Join(r.Dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}
	
	hookPath := filepath.Join(hookDir, "pre-commit")
	if err := os.WriteFile(hookPath, []byte(contents), 0755); err != nil {
		return fmt.Errorf("failed to write hook: %w", err)
	}
	
	return nil
}

// CreateAndCheckoutBranch creates and checks out a new branch.
func (r *GitRepo) CreateAndCheckoutBranch(name string) error {
	return r.runGitCommand("checkout", "-b", name)
}

// CheckoutBranch checks out a branch.
func (r *GitRepo) CheckoutBranch(name string) error {
	return r.runGitCommand("checkout", name)
}

// RebaseInProgress checks if a rebase is in progress.
func (r *GitRepo) RebaseInProgress() bool {
	rebasePath := filepath.Join(r.Dir, ".git", "rebase-merge")
	_, err := os.Stat(rebasePath)
	return err == nil
}

// ResolveMergeConflicts resolves merge conflicts by accepting theirs.
func (r *GitRepo) ResolveMergeConflicts() error {
	return r.runGitCommand("checkout", "--theirs", ".")
}

// MarkMergeConflictsAsResolved marks merge conflicts as resolved.
func (r *GitRepo) MarkMergeConflictsAsResolved() error {
	return r.runGitCommand("add", ".")
}

// CurrentBranchName returns the name of the current branch.
func (r *GitRepo) CurrentBranchName() (string, error) {
	output, err := r.runGitCommandAndGetOutput("branch", "--show-current")
	if err != nil {
		return "", err
	}
	// The output from runGitCommandAndGetOutput is already trimmed, but ensure it's clean
	return strings.TrimSpace(output), nil
}

// GetRef returns the SHA of a ref.
func (r *GitRepo) GetRef(refName string) (string, error) {
	return r.runGitCommandAndGetOutput("show-ref", "-s", refName)
}

// ListCurrentBranchCommitMessages returns the commit messages on the current branch.
func (r *GitRepo) ListCurrentBranchCommitMessages() ([]string, error) {
	output, err := r.runGitCommandAndGetOutput("log", "--oneline", "--format=%B")
	if err != nil {
		return nil, err
	}
	
	lines := []string{}
	for _, line := range splitLines(output) {
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	
	return lines, nil
}

// MergeBranch merges a branch into another.
func (r *GitRepo) MergeBranch(branch, mergeIn string) error {
	if err := r.CheckoutBranch(branch); err != nil {
		return err
	}
	return r.runGitCommand("merge", mergeIn)
}

// TrackBranch tracks a branch using the CLI.
func (r *GitRepo) TrackBranch(branch string, parentBranch string) error {
	args := []string{"branch", "track"}
	if parentBranch != "" {
		args = append(args, "--parent", parentBranch)
	}
	args = append(args, branch)
	return r.RunCliCommand(args)
}

// splitLines splits a string by newlines and returns non-empty lines.
func splitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
