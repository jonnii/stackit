// Package actions provides high-level business logic for CLI commands.
//
// Each action corresponds to a stackit command (create, submit, sync, etc.)
// and orchestrates operations across the engine, git, and github packages.
//
// Key patterns:
//   - Actions accept runtime.Context which provides Engine, Splog, and other dependencies
//   - Actions are stateless - all state is managed through the Engine interface
//   - Actions handle user interaction through the tui package
//
// Dependencies:
//   - engine: Core branch state management
//   - git: Low-level git operations
//   - tui: User interface and prompts
package actions
