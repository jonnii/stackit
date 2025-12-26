package tui

import (
	"github.com/charmbracelet/lipgloss"

	"stackit.dev/stackit/internal/tui/style"
)

// Forward style functions for convenience and backward compatibility

// ColorRed colors text red
func ColorRed(text string) string { return style.ColorRed(text) }

// ColorYellow colors text yellow
func ColorYellow(text string) string { return style.ColorYellow(text) }

// ColorCyan colors text cyan
func ColorCyan(text string) string { return style.ColorCyan(text) }

// ColorBranchName colors a branch name based on whether it's current
func ColorBranchName(branchName string, isCurrent bool) string {
	return style.ColorBranchName(branchName, isCurrent)
}

// ColorNeedsRestack colors the "needs restack" text
func ColorNeedsRestack(text string) string { return style.ColorNeedsRestack(text) }

// ColorDim makes text dim/gray
func ColorDim(text string) string { return style.ColorDim(text) }

// ColorMagenta colors text magenta
func ColorMagenta(text string) string { return style.ColorMagenta(text) }

// ColorPRNumber colors a PR number
func ColorPRNumber(prNumber int) string { return style.ColorPRNumber(prNumber) }

// ColorPRState colors PR state text
func ColorPRState(state string, isDraft bool) string { return style.ColorPRState(state, isDraft) }

// GetScopeColor returns a deterministic color for a scope string
func GetScopeColor(scope string) (lipgloss.Color, bool) { return style.GetScopeColor(scope) }

// ColorScope colors a scope string deterministically
func ColorScope(scope string) string { return style.ColorScope(scope) }

// GetLogShortColor returns a styled string with the color from StackitColors
func GetLogShortColor(text string, index int) string { return style.GetLogShortColor(text, index) }

// FormatShortLine applies color formatting to a short log line
func FormatShortLine(line string, circleIndex, arrowIndex int, isCurrent bool, overallIndent int) string {
	return style.FormatShortLine(line, circleIndex, arrowIndex, isCurrent, overallIndent)
}

// StackitColors defines the color palette for branch visualization
var StackitColors = style.StackitColors
