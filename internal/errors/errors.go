// Package errors provides sentinel errors and custom error types for the stackit application.
// Use errors.Is() and errors.As() to check for specific error types.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common conditions
var (
	// ErrNotOnBranch indicates that HEAD is not on a branch
	ErrNotOnBranch = errors.New("not on a branch")

	// ErrBranchNotFound indicates that a branch does not exist
	ErrBranchNotFound = errors.New("branch not found")

	// ErrRebaseConflict indicates that a rebase operation encountered a conflict
	ErrRebaseConflict = errors.New("rebase conflict")

	// ErrRebaseNotInProgress indicates that no rebase is currently in progress
	ErrRebaseNotInProgress = errors.New("no rebase in progress")

	// ErrTrunkOperation indicates an invalid operation on the trunk branch
	ErrTrunkOperation = errors.New("invalid operation on trunk branch")
)

// BranchNotFoundError represents an error when a branch is not found
type BranchNotFoundError struct {
	BranchName string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("branch %s does not exist", e.BranchName)
}

// Is returns true if the target error is ErrBranchNotFound
func (e *BranchNotFoundError) Is(target error) bool {
	return target == ErrBranchNotFound
}

// NewBranchNotFoundError creates a new BranchNotFoundError
func NewBranchNotFoundError(branchName string) *BranchNotFoundError {
	return &BranchNotFoundError{BranchName: branchName}
}

// RebaseConflictError represents an error when a rebase encounters a conflict
type RebaseConflictError struct {
	BranchName string
	Message    string
}

func (e *RebaseConflictError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("rebase conflict on branch %s: %s", e.BranchName, e.Message)
	}
	return fmt.Sprintf("rebase conflict on branch %s", e.BranchName)
}

// Is returns true if the target error is ErrRebaseConflict
func (e *RebaseConflictError) Is(target error) bool {
	return target == ErrRebaseConflict
}

// NewRebaseConflictError creates a new RebaseConflictError
func NewRebaseConflictError(branchName string, message string) *RebaseConflictError {
	return &RebaseConflictError{
		BranchName: branchName,
		Message:    message,
	}
}

// GitCommandError represents an error from a git command execution
type GitCommandError struct {
	Command string
	Args    []string
	Stdout  string
	Stderr  string
	Err     error
}

func (e *GitCommandError) Error() string {
	msg := fmt.Sprintf("git command failed: %s", e.Command)
	if len(e.Args) > 0 {
		msg += fmt.Sprintf(" %v", e.Args)
	}
	if e.Stderr != "" {
		msg += fmt.Sprintf("\nstderr: %s", e.Stderr)
	}
	if e.Stdout != "" {
		msg += fmt.Sprintf("\nstdout: %s", e.Stdout)
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\n%v", e.Err)
	}
	return msg
}

func (e *GitCommandError) Unwrap() error {
	return e.Err
}

// NewGitCommandError creates a new GitCommandError
func NewGitCommandError(command string, args []string, stdout, stderr string, err error) *GitCommandError {
	return &GitCommandError{
		Command: command,
		Args:    args,
		Stdout:  stdout,
		Stderr:  stderr,
		Err:     err,
	}
}
