# Command Structure Patterns

This document outlines the patterns extracted from the `submit` command that should be applied to all commands in stackit.

## Summary of Patterns

The submit command demonstrates several architectural patterns that improve consistency, testability, and maintainability:

1. **Standardized Context Management**
2. **Clean Separation of Concerns**
3. **Structured Options Pattern**
4. **UI Abstraction Layer**
5. **Modular Organization for Complex Commands**
6. **Consistent Validation Pattern**
7. **Quiet vs. Verbose Operations**

---

## Pattern 1: Standardized Context Management

### ✅ Good Example (submit)

```go
// internal/cli/submit.go
func newSubmitCmd() *cobra.Command {
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            // Get context (demo or real) - handles all initialization
            ctx, err := runtime.GetContext()
            if err != nil {
                return err
            }
            
            return submit.Action(ctx, submit.Options{...})
        },
    }
    return cmd
}
```

### ❌ Inconsistent Examples (other commands)

```go
// internal/cli/sync.go - Manual initialization
func newSyncCmd() *cobra.Command {
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            if err := git.InitDefaultRepo(); err != nil {
                return fmt.Errorf("not a git repository: %w", err)
            }
            repoRoot, err := git.GetRepoRoot()
            if err != nil {
                return fmt.Errorf("failed to get repo root: %w", err)
            }
            if !config.IsInitialized(repoRoot) {
                return fmt.Errorf("stackit not initialized. Run 'stackit init' first")
            }
            eng, err := engine.NewEngine(repoRoot)
            if err != nil {
                return fmt.Errorf("failed to create engine: %w", err)
            }
            ctx := runtime.NewContext(eng)
            // ... use ctx
        },
    }
}

// internal/cli/create.go - Uses helper
func newCreateCmd() *cobra.Command {
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            repoRoot, err := EnsureInitialized()  // Different helper
            if err != nil {
                return err
            }
            eng, err := engine.NewEngine(repoRoot)
            if err != nil {
                return fmt.Errorf("failed to create engine: %w", err)
            }
            ctx := runtime.NewContext(eng)
            // ... use ctx
        },
    }
}
```

### 💡 Improvement

**All commands should use `runtime.GetContext()`** which:
- Handles demo vs. real engine selection
- Initializes git repository
- Checks if stackit is initialized
- Creates and configures the engine
- Sets up GitHub client if needed
- Returns a fully-configured context

This eliminates boilerplate and ensures consistent behavior.

---

## Pattern 2: Clean Separation of Concerns

### ✅ Good Example (submit)

**CLI Layer** (`internal/cli/submit.go`):
- Defines command metadata (Use, Short, Long)
- Declares flags
- Maps flags to Options struct
- Calls action function
- **NO business logic**

**Action Layer** (`internal/actions/submit/submit.go`):
- Contains all business logic
- Orchestrates the operation
- Delegates to helper functions
- Returns errors up to CLI layer

### ❌ Mixed Concerns (sync, modify)

```go
// internal/actions/sync.go
type SyncOptions struct {
    All     bool
    Force   bool
    Restack bool
    Engine  engine.Engine  // ❌ Dependencies in Options
    Splog   *tui.Splog     // ❌ Dependencies in Options
}

// internal/cli/modify.go
return actions.ModifyAction(actions.ModifyOptions{
    All:      all,
    Engine:   eng,      // ❌ Passing engine separately
    Splog:    ctx.Splog, // ❌ Passing splog separately
    RepoRoot: repoRoot,  // ❌ Passing repo root separately
})
```

### 💡 Improvement

**Options should contain only user-facing configuration:**
```go
// ✅ Options contain only flags/config
type Options struct {
    Branch  string
    Force   bool
    DryRun  bool
    // ... other user-facing options
}

// ✅ Action receives context with all dependencies
func Action(ctx *runtime.Context, opts Options) error {
    eng := ctx.Engine
    splog := ctx.Splog
    // ...
}
```

**Benefits:**
- Clear distinction between user input and system state
- Easier to test (mock context, real options)
- Options can be serialized/logged easily

---

## Pattern 3: Structured Options Pattern

### ✅ Good Example (submit)

```go
// Dedicated Options struct in action package
type Options struct {
    Branch               string
    Stack                bool
    Force                bool
    DryRun               bool
    // ... 20+ options well-organized
}

// CLI maps flags directly to Options
return submit.Action(ctx, submit.Options{
    Branch:  branch,
    Stack:   stack,
    Force:   force,
    DryRun:  dryRun,
    // ...
})
```

### ❌ Inconsistent (various commands)

Some commands don't have Options structs, or pass options inconsistently.

### 💡 Improvement

**Every command should:**
1. Define an `Options` struct in its action package
2. Include ALL command-specific configuration in Options
3. Use consistent naming (match flag names when possible)
4. Document complex options

---

## Pattern 4: UI Abstraction Layer

### ✅ Good Example (submit)

```go
// internal/tui/submit_ui.go - Dedicated UI type
type SubmitUI interface {
    ShowStack(lines []string)
    ShowRestackStart()
    ShowRestackComplete()
    ShowPreparing()
    ShowBranchPlan(branch, action string, isCurrent, skipped bool, reason string)
    StartSubmitting(items []SubmitItem)
    UpdateSubmitItem(idx int, status, url string, err error)
    Complete()
    // ...
}

// Usage in action
ui := tui.NewSubmitUI(splog)
ui.ShowStack(stackLines)
ui.ShowPreparing()
// ...
ui.StartSubmitting(progressItems)
ui.UpdateSubmitItem(idx, "done", prURL, nil)
ui.Complete()
```

### ❌ Direct Logging (other commands)

```go
// internal/actions/sync.go
splog.Info("Pulling %s from remote...", tui.ColorBranchName(eng.Trunk(), false))
splog.Info("%s fast-forwarded to %s.", tui.ColorBranchName(eng.Trunk(), true), ...)
```

### 💡 Improvement

**For complex commands with multi-phase output:**
- Create a dedicated UI type (interface + implementation)
- Encapsulate formatting and display logic
- Makes it easier to:
  - Change output formats
  - Add progress indicators
  - Support different output modes (JSON, quiet, etc.)
  - Test business logic without output concerns

**For simple commands:**
- Direct `splog` usage is fine
- Use consistent color helpers (`tui.ColorBranchName`, etc.)

---

## Pattern 5: Modular Organization for Complex Commands

### ✅ Good Example (submit)

```
internal/actions/submit/
├── submit.go              # Main action orchestration
├── submit_validation.go   # Validation logic
├── submit_metadata.go     # Metadata collection
├── submit_test.go         # Tests
└── submit_metadata_test.go # More tests
```

**Benefits:**
- Each file has a single, clear responsibility
- Easier to navigate and understand
- Related code is co-located
- Tests live with implementation

### ❌ Monolithic Files

Some commands have all logic in a single file, even when complex.

### 💡 Improvement

**When to create a package:**
- Command has 300+ lines of logic
- Multiple distinct phases (validation, preparation, execution)
- Reusable sub-operations
- Complex data structures specific to the command

**File organization within package:**
- `{command}.go` - Main Action function and orchestration
- `{command}_validation.go` - Validation logic
- `{command}_types.go` - Types and data structures
- `{command}_helpers.go` - Helper functions
- `{command}_test.go` - Tests

---

## Pattern 6: Consistent Validation Pattern

### ✅ Good Example (submit)

```go
func Action(ctx *runtime.Context, opts Options) error {
    eng := ctx.Engine
    splog := ctx.Splog
    
    // 1. Early flag validation
    if opts.Draft && opts.Publish {
        return fmt.Errorf("can't use both --publish and --draft flags")
    }
    
    // 2. Get data
    branches, err := getBranchesToSubmit(opts, eng)
    if err != nil {
        return err
    }
    
    // 3. Validate data
    if err := ValidateBranchesToSubmit(branches, eng, ctx); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // 4. Proceed with operation
    // ...
}
```

### 💡 Improvement

**Validation phases:**
1. **Flag validation** - Conflicting flags, invalid combinations
2. **State validation** - Repository state, branch existence, etc.
3. **Pre-condition validation** - Can operation proceed?

**Extract validation functions:**
- `ValidateOptions(opts Options) error` - Flag validation
- `Validate{CommandName}State(ctx *runtime.Context, opts Options) error`

---

## Pattern 7: Quiet vs. Verbose Operations

### ✅ Good Example (submit)

```go
// Quiet version - no logging, used internally
func createPullRequestQuiet(...) (string, error) {
    pr, err := githubClient.CreatePullRequest(...)
    if err != nil {
        return "", fmt.Errorf("failed to create PR: %w", err)
    }
    return pr.HTMLURL, nil
}

// Verbose version - includes logging
func createPullRequest(..., splog *tui.Splog) (string, error) {
    prURL, err := createPullRequestQuiet(...)
    if err != nil {
        return "", err
    }
    
    splog.Info("%s: %s (%s)",
        tui.ColorBranchName(branchName, true),
        prURL,
        tui.ColorDim("created"))
    
    return prURL, nil
}
```

### 💡 Improvement

**For operations that might be called multiple times:**
- Create "quiet" version without logging
- Create wrapper with logging for single operations
- Use quiet version when orchestrating multiple calls
- Prevents log spam during batch operations

---

## Actionable Improvements by Command

### High Priority (Complex Commands)

**`sync`**
- [ ] Use `runtime.GetContext()` instead of manual initialization
- [ ] Move Engine/Splog out of Options, pass only via context
- [ ] Consider creating `internal/actions/sync/` package
- [ ] Extract validation functions
- [ ] Create SyncUI for multi-phase output

**`restack`**
- [ ] Use `runtime.GetContext()` instead of manual initialization
- [ ] Move Engine/Splog out of Options
- [ ] Already fairly clean, just needs consistency fixes

**`modify`**
- [ ] Use `runtime.GetContext()` instead of manual initialization
- [ ] Move Engine/Splog/RepoRoot out of Options
- [ ] Already has good separation

### Medium Priority

**`create`**
- [ ] Use `runtime.GetContext()` instead of `EnsureInitialized()`
- [ ] Already fairly clean otherwise

**`checkout`**
- [ ] Use `runtime.GetContext()` instead of `EnsureInitialized()`
- [ ] Good separation already exists

### Other Commands

Apply the same patterns to:
- `absorb`
- `split`
- `merge`
- `squash`
- `log`
- `info`
- etc.

---

## Template for New Commands

```go
// internal/cli/mycommand.go
package cli

import (
    "github.com/spf13/cobra"
    "stackit.dev/stackit/internal/actions/mycommand"
    "stackit.dev/stackit/internal/runtime"
)

func newMyCommandCmd() *cobra.Command {
    var (
        flag1 string
        flag2 bool
    )

    cmd := &cobra.Command{
        Use:   "mycommand",
        Short: "Short description",
        Long:  `Long description`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Get context (handles all initialization)
            ctx, err := runtime.GetContext()
            if err != nil {
                return err
            }

            // Run action
            return mycommand.Action(ctx, mycommand.Options{
                Flag1: flag1,
                Flag2: flag2,
            })
        },
    }

    cmd.Flags().StringVar(&flag1, "flag1", "", "Description")
    cmd.Flags().BoolVar(&flag2, "flag2", false, "Description")

    return cmd
}
```

```go
// internal/actions/mycommand/mycommand.go
package mycommand

import (
    "fmt"
    "stackit.dev/stackit/internal/runtime"
)

// Options contains options for the mycommand command
type Options struct {
    Flag1 string
    Flag2 bool
    // Only user-facing configuration
}

// Action performs the mycommand operation
func Action(ctx *runtime.Context, opts Options) error {
    eng := ctx.Engine
    splog := ctx.Splog

    // 1. Validate flags
    if err := validateOptions(opts); err != nil {
        return err
    }

    // 2. Validate state
    if err := validateState(ctx, opts); err != nil {
        return err
    }

    // 3. Perform operation
    // ...

    return nil
}

func validateOptions(opts Options) error {
    // Flag validation
    return nil
}

func validateState(ctx *runtime.Context, opts Options) error {
    // State validation
    return nil
}
```

---

## Key Takeaways

1. **Consistency is key** - Users and developers benefit from predictable patterns
2. **Context management** - `runtime.GetContext()` should be the standard
3. **Separation of concerns** - CLI declares, Actions execute
4. **Options purity** - Keep dependencies out of Options
5. **UI abstraction** - For complex output, create dedicated UI types
6. **Modular organization** - Split complex commands into multiple files
7. **Validation** - Explicit, early, and separate from execution
8. **Testability** - Patterns improve testability (mock context, real options)

---

## Migration Strategy

1. **Phase 1**: Update high-priority commands (sync, restack, modify)
2. **Phase 2**: Update medium-priority commands (create, checkout)
3. **Phase 3**: Update remaining commands
4. **Phase 4**: Document patterns in contributing guide

For each command:
1. Update context management first (quick win)
2. Refactor Options struct
3. Extract validation if needed
4. Create UI abstraction if complex
5. Add tests
6. Update documentation
