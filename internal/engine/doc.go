// Package engine manages the state and relationships of stacked branches.
//
// It is the core of stackit, responsible for:
//   - Tracking parent-child relationships between branches
//   - Storing and retrieving branch metadata (PR info, status, etc.)
//   - Managing the branch stack structure
//   - Coordinating branch operations like splitting, squashing, and restacking
//
// The engine abstracts the underlying storage (git refs, notes) and provides
// a high-level interface for branch management.
package engine
