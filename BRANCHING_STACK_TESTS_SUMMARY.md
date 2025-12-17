# Branching Stack Tests for Restack Command

## Summary

Added comprehensive branching stack tests to `internal/cli/restack_test.go` to ensure the restack command properly handles scenarios where branches have multiple children (branching stacks).

## Tests Added

### 1. **restack with branching stack - parent with multiple children**
- **Scenario**: Tests restacking when a parent branch has multiple children
- **Structure**:
  ```
  main → parent → child1
                → child2
  ```
- **Actions**:
  - Makes a change to main
  - Runs `restack --upstack` from parent
  - Verifies both children are still properly related to parent

### 2. **restack branching stacks in topological order**
- **Scenario**: Tests complex branching structure with multiple stacks
- **Structure**:
  ```
  main
  ├── stackA
  │   ├── stackA-child1
  │   └── stackA-child2
  └── stackB
      └── stackB-child1
  ```
- **Actions**:
  - Makes a change to main
  - Runs `restack --upstack` from stackA
  - Verifies stackA and its children are properly restacked
  - Ensures parent-child relationships are preserved

### 3. **restack auto-reparents multiple children when parent is merged**
- **Scenario**: Tests that all children are reparented when their parent is merged into trunk
- **Structure**:
  ```
  main → parent → child1
                → child2
                → child3
  ```
- **Actions**:
  - Merges parent into main
  - Runs `restack --only` on each child
  - Verifies all three children are now parented to main

### 4. **restack with --downstack includes siblings**
- **Scenario**: Tests that `--downstack` flag works correctly with siblings
- **Structure**:
  ```
  main → parent → child1
                → child2 (current)
  ```
- **Actions**:
  - Makes a change to main
  - Runs `restack --downstack` from child2
  - Verifies child2 is still properly related to parent

### 5. **restack --upstack from parent restacks all children**
- **Scenario**: Tests that restacking from a parent affects all its children
- **Structure**:
  ```
  main → parent → child1
                → child2
                → child3
  ```
- **Actions**:
  - Amends the parent branch
  - Runs `restack --upstack` from parent
  - Verifies all three children are restacked and still children of parent

## Coverage Comparison

### Before (Existing Tests)
- ✅ Single linear stacks (main → branch1 → branch2)
- ✅ Auto-reparenting when parent is merged/deleted (linear)
- ✅ Flags: --only, --downstack, --upstack, --branch
- ✅ Error handling
- ✅ Conflict resolution

### After (With New Tests)
- ✅ Single linear stacks
- ✅ **Branching stacks (parent with multiple children)**
- ✅ Auto-reparenting when parent is merged/deleted (linear)
- ✅ **Auto-reparenting multiple children when parent is merged**
- ✅ Flags: --only, --downstack, --upstack, --branch
- ✅ **Flag behavior with branching structures**
- ✅ **Topological ordering with branching stacks**
- ✅ Error handling
- ✅ Conflict resolution

## Test Patterns Used

All tests follow the established patterns in the codebase:
- Use `testhelpers.NewSceneParallel` for parallel test execution
- Use `exec.Command(binaryPath, ...)` to invoke stackit commands
- Use `require` assertions from testify
- Create realistic Git scenarios with commits and branch operations
- Verify behavior using `stackit info` command output

## Alignment with Sync Tests

These restack tests mirror the branching stack tests already present in `internal/actions/sync_test.go`:
- "restacks branches in topological order (parents before children)" - line 85
- "restacks branching stacks in topological order" - line 130

This ensures consistent test coverage across both restack and sync operations.

## Running the Tests

```bash
# Run all restack tests
go test -v ./internal/cli/ -run TestRestackCommand

# Or using just
just test-pkg ./internal/cli
```

## Notes

- All new tests use `t.Parallel()` for faster test execution
- Tests create real Git repositories using the test helpers
- Each test is self-contained and doesn't affect other tests
- Tests verify both the command output and the actual Git state
