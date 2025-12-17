// Package demo provides a simulated engine for testing TUI interactions
// without requiring a real git repository.
package demo

import "stackit.dev/stackit/internal/engine"

// DemoBranch represents a simulated branch with PR info
type DemoBranch struct {
	Name     string
	Parent   string
	PRNumber int
	PRState  string // OPEN, MERGED, CLOSED
	PRTitle  string
	IsDraft  bool
	Checks   string // PASSING, FAILING, PENDING, NONE
}

// Demo stack structure - branching stack with no upstack from current branch
//
// main
// ├── feature/auth-base (#101)
// │   ├── feature/auth-validation (#102)
// │   │   └── feature/auth-login (#103) ← current
// │   └── feature/auth-oauth (#105)
// │       └── feature/auth-oauth-google (#106)
// └── feature/api-refactor (#107)
//
//	└── feature/api-v2 (#108)
var demoBranches = []DemoBranch{
	{
		Name:     "feature/auth-base",
		Parent:   "main",
		PRNumber: 101,
		PRState:  "OPEN",
		PRTitle:  "Add authentication base module",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/auth-validation",
		Parent:   "feature/auth-base",
		PRNumber: 102,
		PRState:  "OPEN",
		PRTitle:  "Add input validation for auth",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/auth-login",
		Parent:   "feature/auth-validation",
		PRNumber: 103,
		PRState:  "OPEN",
		PRTitle:  "Implement login flow",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/auth-oauth",
		Parent:   "feature/auth-base",
		PRNumber: 105,
		PRState:  "OPEN",
		PRTitle:  "Add OAuth support",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/auth-oauth-google",
		Parent:   "feature/auth-oauth",
		PRNumber: 106,
		PRState:  "OPEN",
		PRTitle:  "Add Google OAuth provider",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/api-refactor",
		Parent:   "main",
		PRNumber: 107,
		PRState:  "OPEN",
		PRTitle:  "Refactor API layer",
		IsDraft:  false,
		Checks:   "PASSING",
	},
	{
		Name:     "feature/api-v2",
		Parent:   "feature/api-refactor",
		PRNumber: 108,
		PRState:  "OPEN",
		PRTitle:  "Implement API v2 endpoints",
		IsDraft:  false,
		Checks:   "PASSING",
	},
}

// Current branch is a leaf node (no children) to avoid warnings
var demoCurrentBranch = "feature/auth-login"
var demoTrunk = "main"

// GetDemoBranches returns the demo branch data
func GetDemoBranches() []DemoBranch {
	return demoBranches
}

// GetDemoCurrentBranch returns the simulated current branch
func GetDemoCurrentBranch() string {
	return demoCurrentBranch
}

// GetDemoTrunk returns the simulated trunk branch
func GetDemoTrunk() string {
	return demoTrunk
}

// GetDemoPrInfo returns simulated PR info for a branch
func GetDemoPrInfo(branchName string) *engine.PrInfo {
	for _, b := range demoBranches {
		if b.Name == branchName {
			num := b.PRNumber
			return &engine.PrInfo{
				Number:  &num,
				Title:   b.PRTitle,
				Body:    "Demo PR body for " + branchName,
				IsDraft: b.IsDraft,
				State:   b.PRState,
				Base:    b.Parent,
				URL:     "https://github.com/example/repo/pull/" + string(rune('0'+num%10)),
			}
		}
	}
	return nil
}

// GetDemoChecksStatus returns the simulated checks status for a branch
func GetDemoChecksStatus(branchName string) string {
	for _, b := range demoBranches {
		if b.Name == branchName {
			return b.Checks
		}
	}
	return "NONE"
}
