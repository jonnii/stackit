# Command Refactoring Summary

## Overview

This document summarizes the systematic refactoring of stackit commands to follow the patterns established in the `submit` command.

## Commands Refactored (18 total)

### ✅ Batch 1: High-Priority Commands (4 commands)
1. **sync** - Complex multi-phase sync operation
2. **restack** - Branch restacking with conflict handling
3. **modify** - Branch modification with upstack restacking
4. **create** - Branch creation with tracking

### ✅ Batch 2: Medium-Priority Commands (6 commands)
5. **checkout** - Interactive branch selection and switching
6. **absorb** - Intelligent change absorption into commits
7. **squash** - Commit squashing with restacking
8. **log** - Branch tree visualization
9. **info** - Branch information display
10. **split** - Branch splitting (by commit, hunk, file)

### ✅ Batch 3: Complex Operations (2 commands)
11. **continue** - Rebase continuation with state management
12. **merge** - PR merging with interactive wizard

### ✅ Batch 4: Navigation Commands (6 commands)
13. **down** - Navigate to parent branch
14. **bottom** - Navigate to stack bottom
15. **top** - Navigate to stack top
16. **parent** - Display parent branch
17. **children** - Display child branches
18. **trunk** - Display/manage trunk branches

## Commands NOT Refactored (Reasons)

### Stub Commands (Not Yet Implemented)
- **delete** - Placeholder only
- **fold** - Placeholder only
- **pop** - Placeholder only
- **move** - Placeholder only
- **reorder** - Placeholder only
- **abort** - Placeholder only

### Special Cases
- **init** - Cannot use `runtime.GetContext()` (initializes stackit itself)
- **passthrough** - Handler function, not a command

## Improvements Applied

### 1. Standardized Context Management

**Before:**
```go
// 15-20 lines of boilerplate per command
if err := git.InitDefaultRepo(); err != nil { ... }
repoRoot, err := git.GetRepoRoot()
if err != nil { ... }
if !config.IsInitialized(repoRoot) { ... }
eng, err := engine.NewEngine(repoRoot)
if err != nil { ... }
ctx := runtime.NewContext(eng)
```

**After:**
```go
// 3 lines
ctx, err := runtime.GetContext()
if err != nil {
    return err
}
```

**Impact:** Reduced ~270 lines of boilerplate across 18 commands

### 2. Clean Options Structs

**Before:**
```go
type SyncOptions struct {
    Force    bool
    Engine   engine.Engine  // ❌ Dependency
    Splog    *tui.Splog     // ❌ Dependency
    RepoRoot string         // ❌ Dependency
}
```

**After:**
```go
type SyncOptions struct {
    Force bool  // ✅ Only user configuration
}
```

**Impact:** Clearer separation, easier testing, options can be serialized

### 3. Consistent Action Signatures

**Before (Mixed):**
```go
func SyncAction(opts SyncOptions) error { ... }
func RestackAction(opts RestackOptions) error { ... }
func CreateAction(opts CreateOptions, ctx *runtime.Context) error { ... }
func LogAction(opts LogOptions, ctx *runtime.Context) error { ... }
```

**After (Consistent):**
```go
func Action(ctx *runtime.Context, opts Options) error { ... }
```

**Impact:** 
- Predictable API across all commands
- Context always comes first
- Options always comes second

### 4. Simplified CLI Layer

**Before (example from sync):**
```go
// CLI layer: 88 lines
// Contains initialization, config checks, engine creation, context setup
```

**After:**
```go
// CLI layer: 48 lines
// Only flag declarations and mapping to Options
```

**Impact:** ~40% reduction in CLI code, zero business logic

## Code Metrics

### Lines of Code Reduced
- **CLI Layer**: ~400 lines eliminated
- **Action Layer**: ~250 lines eliminated
- **Total**: ~650 lines of boilerplate removed

### Files Modified
- 18 CLI command files (`internal/cli/`)
- 15 Action files (`internal/actions/`)
- 6 Test files (updated to match new signatures)
- **Total**: 39 files

### Test Coverage
- ✅ All 18 commands have passing tests
- ✅ No test functionality degraded
- ✅ Test code also simplified (updated to use contexts)

## Pull Requests Created

### Stack Structure (9 PRs)
```
main
├── PR #29: Refactor sync command
├── PR #30: Refactor restack command
├── PR #31: Refactor modify command
├── PR #32: Refactor create and checkout commands
├── PR #33: Refactor absorb command
├── PR #34: Refactor squash and log commands
├── PR #35: Refactor info and split commands
├── PR #36: Refactor continue and merge commands
└── PR #37: Refactor navigation commands
```

## Pattern Benefits

### 1. Developer Experience
- **Consistency**: All commands follow the same structure
- **Discoverability**: Developers know where to find things
- **Less Cognitive Load**: No need to remember different patterns

### 2. Maintainability
- **Less Duplication**: Context management in one place
- **Easier Updates**: Changes to initialization logic only need to be made once
- **Clear Ownership**: Each layer has a single responsibility

### 3. Testability
- **Mock Context**: Easy to create test contexts
- **Pure Options**: Options can be tested independently
- **No Side Effects**: Actions don't create their own dependencies

### 4. Future-Proofing
- **Demo Mode**: `runtime.GetContext()` handles demo vs. real engine
- **Extensibility**: Easy to add new context fields (e.g., config, logger)
- **Configuration**: Context can carry repo-specific configuration

## Key Patterns Established

### Pattern 1: Context Management
✅ All commands use `runtime.GetContext()`
- Handles demo vs. real engine selection
- Initializes git repository
- Checks if stackit is initialized
- Creates and configures the engine
- Sets up GitHub client if needed

### Pattern 2: Options Purity
✅ Options contain ONLY user-facing configuration
- No Engine, Splog, RepoRoot in Options
- All dependencies passed via Context
- Options can be logged, serialized, validated independently

### Pattern 3: Consistent Signatures
✅ All actions follow: `func Action(ctx *runtime.Context, opts Options) error`
- Context first (dependencies)
- Options second (configuration)
- Returns error only

### Pattern 4: CLI Simplicity
✅ CLI layer contains ZERO business logic
- Declare command metadata
- Declare flags
- Map flags → Options
- Call Action function
- Handle errors (let cobra handle them)

## Remaining Work

### Commands to Refactor (When Implemented)
The following stub commands should follow these patterns when implemented:
- `delete` - Branch deletion
- `fold` - Branch folding
- `pop` - Branch popping with state retention
- `move` - Branch moving/rebasing
- `reorder` - Branch reordering
- `abort` - Operation abortion

### Template for Future Commands
See `COMMAND_PATTERNS.md` for the template to use when implementing these commands.

## Migration Notes

### For Future Command Implementations
1. Start with the template in `COMMAND_PATTERNS.md`
2. CLI: Only flags → Options mapping
3. Action: Use `ctx.Engine`, `ctx.Splog`, `ctx.RepoRoot`
4. Options: Only user-facing configuration
5. Tests: Create `runtime.NewContext(eng)` with test engine

### For External Contributors
- New commands MUST follow these patterns
- PRs that don't follow patterns will be asked to refactor
- See `submit` command as the canonical example

## Success Criteria

All refactored commands now:
- ✅ Use `runtime.GetContext()` for initialization
- ✅ Have clean Options structs (no dependencies)
- ✅ Follow consistent `Action(ctx, opts)` signature
- ✅ Have zero business logic in CLI layer
- ✅ Pass all existing tests
- ✅ Are documented in `COMMAND_PATTERNS.md`

## Impact Summary

**Consistency**: 18 commands now follow identical patterns
**Code Quality**: ~650 lines of boilerplate eliminated
**Maintainability**: Single source of truth for initialization
**Testability**: Easier to mock and test
**Documentation**: Clear patterns for future development
