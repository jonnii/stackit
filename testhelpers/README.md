# Test Helpers

This package provides testing utilities for the Stackit CLI Go port, ported from the TypeScript test infrastructure.

## Overview

The testhelpers package provides:
- **Scene System**: Reusable test scenarios with temporary Git repositories
- **Git Repository Helpers**: Utilities for manipulating Git repos in tests
- **Assertions**: Custom assertion helpers for branches and commits

## Scene System

The scene system creates isolated test environments with temporary Git repositories.

### Basic Usage

```go
import (
    "testing"
    "stackit.dev/stackit/testhelpers"
)

func TestMyFeature(t *testing.T) {
    // Create a basic scene with a single commit
    scene := testhelpers.NewScene(t, testhelpers.BasicSceneSetup)
    
    // Use scene.Repo to interact with the Git repository
    err := scene.Repo.RunCliCommand([]string{"branch", "create", "feature"})
    require.NoError(t, err)
    
    // Assertions
    testhelpers.ExpectBranchesString(t, scene.Repo, "feature, main")
}
```

### Custom Setup

```go
func TestCustomScenario(t *testing.T) {
    customSetup := func(scene *testhelpers.Scene) error {
        // Create multiple commits
        scene.Repo.CreateChangeAndCommit("commit 1", "1")
        scene.Repo.CreateChangeAndCommit("commit 2", "2")
        
        // Create branches
        scene.Repo.CreateAndCheckoutBranch("parent")
        scene.Repo.CheckoutBranch("main")
        
        return nil
    }
    
    scene := testhelpers.NewScene(t, customSetup)
    // Test implementation...
}
```

## Git Repository Helpers

The `GitRepo` type provides methods for Git operations:

### Basic Operations

```go
repo := scene.Repo

// Create commits
repo.CreateChangeAndCommit("content", "prefix")

// Branch operations
repo.CreateAndCheckoutBranch("feature")
repo.CheckoutBranch("main")
repo.DeleteBranch("old-branch")

// Get information
branch, _ := repo.CurrentBranchName()
ref, _ := repo.GetRef("HEAD")
messages, _ := repo.ListCurrentBranchCommitMessages()
```

### Running CLI Commands

```go
// Run a CLI command
err := repo.RunCliCommand([]string{"branch", "create", "feature"})

// Run a CLI command and get output
output, err := repo.RunCliCommandAndGetOutput([]string{"branch", "list"})
```

## Assertions

### Branch Assertions

```go
// Assert branches as a slice
testhelpers.ExpectBranches(t, repo, []string{"main", "feature"})

// Assert branches as a comma-separated string (matches TypeScript API)
testhelpers.ExpectBranchesString(t, repo, "feature, main")
```

### Commit Assertions

```go
// Assert commits on a specific branch
testhelpers.ExpectCommits(t, repo, "main", []string{"Initial commit", "Second commit"})

// Assert commits as a comma-separated string (matches TypeScript API)
testhelpers.ExpectCommitsString(t, repo, "Initial commit, Second commit")
```

## Scene Types

The following scene setups are available:

- **`BasicSceneSetup`**: Creates a scene with a single commit on main branch

Additional scene types can be added as needed, following the pattern:

```go
func MySceneSetup(scene *testhelpers.Scene) error {
    // Setup logic here
    return nil
}
```

## Cleanup

Scenes automatically clean up temporary directories using `t.Cleanup()`. If the `DEBUG` environment variable is set, temporary directories are preserved for inspection.

## Migration Notes

This package is a direct port of the TypeScript test infrastructure:
- `AbstractScene` → `Scene`
- `GitRepo` class → `GitRepo` struct
- `expectBranches` → `ExpectBranches` / `ExpectBranchesString`
- `expectCommits` → `ExpectCommits` / `ExpectCommitsString`

The API is designed to be as similar as possible to the TypeScript version while following Go idioms.
